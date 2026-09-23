//
//  STProxyProvider.m
//
//  NETransparentProxyProvider implementation: session lifecycle and the
//  per-flow exclude/pass-through decision. The actual relaying is split out
//  into STProxyProvider+TCPRelay.m / STProxyProvider+UDPRelay.m.
//
//  Settings (excluded-app list, interface) are applied only via
//  startProxyWithOptions: - NETransparentProxyProvider does not reliably
//  deliver app messages sent while a session is already running, so the
//  host applies any change by stopping and restarting the session with new
//  options rather than pushing a live update.
//
//  Which physical interface to pin relay connections to and keeping that
//  live for the rest of the session is entirely owned by
//  STPhysicalInterfaceSelector - see that class for the tier precedence and
//  self-healing rationale. By default it resolves the interface itself (from
//  the 'default' route) rather than taking it from the host, so a network
//  change refreshes it immediately; a `physicalInterface` start option can
//  still override that, though nothing supplies one today. Relay connections
//  are refused (see STProxyProvider+TCPRelay.m / +UDPRelay.m) only while no
//  usable interface is live - blocking an excluded app is the fail-closed
//  choice there, since silently routing it into the tunnel would violate the
//  exclusion the user asked for. Only a genuinely malformed
//  `physicalInterfaceType` value fails the start, since that can never
//  self-heal.
//
#import "STProxyProvider+Private.h"
#import "STPhysicalInterfaceSelector.h"
#import "STLog.h"
#import "STPathMatching.h"

// STUDPFlowState is implemented in STProxyProvider+UDPRelay.m, next to its
// only consumer.

// Every IVPN-shipped binary (daemon, UI, CLI, VPN backends) lives under this
// one bundle - relaying IVPN's own traffic would create an immediate routing
// loop, so this is checked before anything else in -handleNewFlow:,
// regardless of what the host-supplied excludedPaths list contains. (The
// activated extension itself runs from /Library/SystemExtensions and is never
// offered its own flows by NetworkExtension.)
static NSString * const kInternalBypassPathPrefix = @"/Applications/IVPN.app";

BOOL STIsFlowClosedByApp(NSError *error) {
    return [error.domain isEqualToString:NEAppProxyErrorDomain] &&
           (error.code == NEAppProxyFlowErrorNotConnected || error.code == NEAppProxyFlowErrorPeerReset);
}

@implementation STProxyProvider

- (instancetype)init {
    self = [super init];
    if (self) {
        _flowsLock = OS_UNFAIR_LOCK_INIT;
        _activeFlows = [NSMutableSet set];
        _udpFlowStates = [NSMapTable strongToStrongObjectsMapTable];
        _tcpConnectionsByFlow = [NSMapTable strongToStrongObjectsMapTable];
        _excludedPaths = @[]; // exclude nothing until the host configures this via `excludedPaths`
        _excludedBundleIdentifiers = [NSSet set];
        _interfaceSelector = [[STPhysicalInterfaceSelector alloc] init];
    }
    return self;
}

#pragma mark - Lifecycle

// Called once when the host starts the proxy session. This is where we
// tell the OS which traffic to even offer us via -handleNewFlow: below -
// nothing is intercepted until this succeeds.
// Invoked by the OS in response to the host app calling `session
// startTunnelWithOptions:` - never merely as a result of saving/reloading
// the proxy configuration, which starts nothing on its own.
- (void)startProxyWithOptions:(NSDictionary<NSString *, id> *)options
             completionHandler:(void (^)(NSError * _Nullable))completionHandler {
    // Let the host seed the excluded-app list at start time - including
    // clearing it to empty. Only missing/malformed `options` falls back to
    // whatever default/previous value we already have; an empty array is a
    // legitimate "exclude nothing" request and must not be ignored.
    
    // Debug logging is opt-in per session: at debug level every new flow on
    // the machine formats a line with its executable path into the log.
    STLogSetMinLevel([options[@"debugLogging"] boolValue] ? STLogLevelDebug : STLogLevelInfo);

    NSArray<NSString *> *startExcluded = options[@"excludedPaths"];
    if ([startExcluded isKindOfClass:[NSArray class]]) {
        self.excludedPaths = STPathsWithResolvedSymlinks(startExcluded);
        self.excludedBundleIdentifiers = STBundleIdentifiersForPaths(startExcluded);
    }
    NSError *interfaceError = [self.interfaceSelector resolveFromOptions:options];
    if (interfaceError) {
        // Only a malformed `physicalInterfaceType` value reaches here now -
        // "requested interface/type not currently available" no longer does
        // (see STPhysicalInterfaceSelector), so this is always a genuine
        // misconfiguration that can't self-heal by waiting.
        STLogError(@"Aborting proxy start: %@", interfaceError.localizedDescription);
        completionHandler(interfaceError);
        return;
    }
    // Starts before -setTunnelNetworkSettings: so the live interface/type
    // state has a head start on the first flows that arrive once
    // interception turns on, below.
    [self.interfaceSelector start];
    STLogInfo(@"Starting proxy, %lu excluded path(s)", (unsigned long)self.excludedPaths.count);
    STLogDebug(@"excludedPaths=%@", self.excludedPaths);

    NETransparentProxyNetworkSettings *settings =
        [[NETransparentProxyNetworkSettings alloc] initWithTunnelRemoteAddress:@"127.0.0.1"];

    // "Catch everything" rule, so every flow on the system is offered to
    // -handleNewFlow: (we decide per-flow, below, whether to actually take
    // it). A rule that combines a wildcard hostname with a wildcard port is
    // rejected by the OS; the safe "match everything" form is to pass nil
    // endpoints with prefix 0, as done here.
    NENetworkRule *matchEverything =
        [[NENetworkRule alloc] initWithRemoteNetwork:nil
                                         remotePrefix:0
                                         localNetwork:nil
                                          localPrefix:0
                                             protocol:NENetworkRuleProtocolAny
                                            direction:NETrafficDirectionOutbound];

    // Never intercept loopback or private/link-local traffic - loopback is a
    // common, easy-to-hit way to break local dev tools or our own IPC, and
    // private/link-local traffic never leaves the LAN regardless of VPN
    // state, so relaying either through here only adds overhead for zero
    // benefit (RFC 1918 + RFC 3927/4291 link-local, IPv6 unique-local).
    NSArray<NSArray *> *excludedNetworkRanges = @[
        @[@"127.0.0.0", @8],   // IPv4 loopback (whole block, not just 127.0.0.1)
        @[@"::1", @128],       // IPv6 loopback
        @[@"10.0.0.0", @8],
        @[@"172.16.0.0", @12],
        @[@"192.168.0.0", @16],
        @[@"169.254.0.0", @16],
        @[@"fe80::", @10],
        @[@"fc00::", @7],
    ];
    NSMutableArray<NENetworkRule *> *excludedRules = [NSMutableArray array];
    for (NSArray *range in excludedNetworkRanges) {
        NSString *network = range[0];
        NSNumber *prefix = range[1];
        [excludedRules addObject:[[NENetworkRule alloc] initWithRemoteNetwork:[NWHostEndpoint endpointWithHostname:network port:@"0"]
                                                                   remotePrefix:prefix.integerValue
                                                                   localNetwork:nil
                                                                    localPrefix:0
                                                                       protocol:NENetworkRuleProtocolAny
                                                                      direction:NETrafficDirectionOutbound]];
    }

    settings.includedNetworkRules = @[matchEverything];
    settings.excludedNetworkRules = excludedRules;

    // This call is what actually turns interception on. If it fails,
    // -handleNewFlow: below will simply never be called - always check this
    // error while testing.
    __weak typeof(self) weakSelf = self;
    [self setTunnelNetworkSettings:settings completionHandler:^(NSError * _Nullable error) {
        if (error) {
            STLogError(@"setTunnelNetworkSettings failed: %@", error);
        } else {
            STLogInfo(@"Proxy started - now intercepting flows");
            [weakSelf st_startUDPIdleWatchdog];
        }
        completionHandler(error);
    }];
}

// Invoked by the OS in response to the host app calling `session stopTunnel`.
- (void)stopProxyWithReason:(NEProviderStopReason)reason
           completionHandler:(void (^)(void))completionHandler {
    STLogInfo(@"Stopping proxy (reason=%ld), closing %lu active flow(s)", (long)reason, (unsigned long)[self st_activeFlowsSnapshot].count);
    [self st_stopUDPIdleWatchdog];
    [self.interfaceSelector stop];
    // Without this, flows already relayed before the stop would just keep
    // running (nothing tells NetworkExtension or the app they should stop) -
    // "Stop" needs to mean everything currently bypassing stops immediately.
    [self closeAllActiveFlows];
    completionHandler();
}

// Force-closes every currently relayed flow (TCP and UDP) - used when the
// whole proxy session stops.
- (void)closeAllActiveFlows {
    NSArray<NEAppProxyFlow *> *snapshot = [self st_activeFlowsSnapshot];
    for (NEAppProxyFlow *flow in snapshot) {
        if ([flow isKindOfClass:[NEAppProxyTCPFlow class]]) {
            nw_connection_t connection = [self st_connectionForTCPFlow:(NEAppProxyTCPFlow *)flow];
            [self teardownTCPFlow:(NEAppProxyTCPFlow *)flow connection:connection];
        } else if ([flow isKindOfClass:[NEAppProxyUDPFlow class]]) {
            STUDPFlowState *state = [self st_stateForUDPFlow:(NEAppProxyUDPFlow *)flow];
            if (state) {
                [self teardownUDPFlow:(NEAppProxyUDPFlow *)flow state:state];
            }
        }
    }
}

#pragma mark - Lock-protected flow bookkeeping

// See the _flowsLock comment in STProxyProvider+Private.h - every touch of
// activeFlows/tcpConnectionsByFlow/udpFlowStates goes through one of these.

- (void)st_registerTCPFlow:(NEAppProxyTCPFlow *)flow connection:(nw_connection_t)connection {
    os_unfair_lock_lock(&_flowsLock);
    [self.activeFlows addObject:flow];
    [self.tcpConnectionsByFlow setObject:connection forKey:flow];
    os_unfair_lock_unlock(&_flowsLock);
}

- (BOOL)st_unregisterTCPFlow:(NEAppProxyTCPFlow *)flow {
    os_unfair_lock_lock(&_flowsLock);
    BOOL wasRegistered = [self.activeFlows containsObject:flow];
    [self.tcpConnectionsByFlow removeObjectForKey:flow];
    [self.activeFlows removeObject:flow];
    os_unfair_lock_unlock(&_flowsLock);
    return wasRegistered;
}

- (nw_connection_t)st_connectionForTCPFlow:(NEAppProxyTCPFlow *)flow {
    os_unfair_lock_lock(&_flowsLock);
    nw_connection_t connection = [self.tcpConnectionsByFlow objectForKey:flow];
    os_unfair_lock_unlock(&_flowsLock);
    return connection;
}

- (void)st_registerUDPFlow:(NEAppProxyUDPFlow *)flow state:(STUDPFlowState *)state {
    os_unfair_lock_lock(&_flowsLock);
    [self.activeFlows addObject:flow];
    [self.udpFlowStates setObject:state forKey:flow];
    os_unfair_lock_unlock(&_flowsLock);
}

- (void)st_unregisterUDPFlow:(NEAppProxyUDPFlow *)flow {
    os_unfair_lock_lock(&_flowsLock);
    [self.udpFlowStates removeObjectForKey:flow];
    [self.activeFlows removeObject:flow];
    os_unfair_lock_unlock(&_flowsLock);
}

- (STUDPFlowState *)st_stateForUDPFlow:(NEAppProxyUDPFlow *)flow {
    os_unfair_lock_lock(&_flowsLock);
    STUDPFlowState *state = [self.udpFlowStates objectForKey:flow];
    os_unfair_lock_unlock(&_flowsLock);
    return state;
}

- (NSArray<NEAppProxyFlow *> *)st_activeFlowsSnapshot {
    os_unfair_lock_lock(&_flowsLock);
    NSArray<NEAppProxyFlow *> *snapshot = [self.activeFlows allObjects];
    os_unfair_lock_unlock(&_flowsLock);
    return snapshot;
}

#pragma mark - UDP idle-connection watchdog

// A UDP peer connection with no send/receive for this long is considered
// abandoned. 5 minutes matches RFC 4787's recommended NAT UDP mapping
// timeout - evicting sooner than a typical NAT would gains nothing, since
// the mapping wouldn't survive the path anyway.
static const NSTimeInterval kUDPPeerIdleTimeout = 5 * 60.0;
// How often to sweep for idle peers - independent of the timeout above,
// just a cheap periodic check.
static const NSTimeInterval kUDPIdleSweepInterval = 60.0;

- (void)st_startUDPIdleWatchdog {
    if (_udpIdleWatchdog) { return; } // already running
    dispatch_queue_t queue = dispatch_queue_create("net.ivpn.splittunnel.udp-idle-watchdog", DISPATCH_QUEUE_SERIAL);
    dispatch_source_t timer = dispatch_source_create(DISPATCH_SOURCE_TYPE_TIMER, 0, 0, queue);
    dispatch_source_set_timer(timer,
                               dispatch_time(DISPATCH_TIME_NOW, (int64_t)(kUDPIdleSweepInterval * NSEC_PER_SEC)),
                               (uint64_t)(kUDPIdleSweepInterval * NSEC_PER_SEC),
                               (uint64_t)(5 * NSEC_PER_SEC));
    __weak typeof(self) weakSelf = self;
    dispatch_source_set_event_handler(timer, ^{
        [weakSelf st_reapIdleUDPConnections];
    });
    dispatch_resume(timer);
    _udpIdleWatchdog = timer;
}

- (void)st_stopUDPIdleWatchdog {
    if (_udpIdleWatchdog) {
        dispatch_source_cancel(_udpIdleWatchdog);
        _udpIdleWatchdog = nil;
    }
}

- (void)st_reapIdleUDPConnections {
    os_unfair_lock_lock(&_flowsLock);
    NSArray<STUDPFlowState *> *states = [self.udpFlowStates.objectEnumerator allObjects];
    os_unfair_lock_unlock(&_flowsLock);

    for (STUDPFlowState *state in states) {
        NSArray<id> *evicted = [state evictConnectionsIdleLongerThan:kUDPPeerIdleTimeout];
        for (id connection in evicted) {
            nw_connection_cancel((nw_connection_t)connection);
        }
        if (evicted.count > 0) {
            STLogDebug(@"Idle watchdog evicted %lu UDP peer connection(s)", (unsigned long)evicted.count);
        }
    }
}

#pragma mark - Per-flow decision

// Called by macOS for EVERY new outbound flow on the whole machine, because
// of the catch-all rule registered above.
//   - return NO  -> "not interested", the OS routes it exactly as if this
//                   extension didn't exist (i.e. through an active VPN
//                   tunnel, if one is connected). Zero overhead.
//   - return YES -> "I'm taking this one" - we must then open the flow
//                   ourselves and move its bytes to/from the real network.
- (BOOL)handleNewFlow:(NEAppProxyFlow *)flow {
    pid_t pid = STPidForFlow(flow);
    NSString *path = STExecutablePathForPid(pid);

    if (STPathMatchesAny(path, @[kInternalBypassPathPrefix])) {
        // Never proxy IVPN's own traffic, regardless of excludedPaths -
        // doing so would create an immediate routing loop (see
        // kInternalBypassPathPrefix above). Decided on the flow's own path,
        // before and independently of any ancestry match below.
        return NO;
    }

    // Read the (atomic) list once so the whole decision sees one snapshot.
    NSArray<NSString *> *excludedPaths = self.excludedPaths;
    NSString *signingIdentifier = flow.metaData.sourceAppSigningIdentifier;
    BOOL isExcluded = STPathMatchesAny(path, excludedPaths) ||
                      (signingIdentifier.length > 0 &&
                       [self.excludedBundleIdentifiers containsObject:signingIdentifier]);

    // Called for every new flow on the whole machine - keep this at debug
    // level, an info-level line here would drown out everything else.
    if (!isExcluded) {
        // Not an excluded app itself - but it may be running on behalf of
        // one (Terminal -> curl, Steam -> game). See STAncestorPathMatchingAny.
        NSString *ancestorPath = STAncestorPathMatchingAny(pid, excludedPaths);
        if (ancestorPath == nil) {
            STLogDebug(@"Flow from %@ not excluded, passing through", path ?: signingIdentifier ?: @"(unknown)");
            return NO;
        }
        STLogDebug(@"Flow from %@ excluded through its parent %@", path ?: @"(unknown)", ancestorPath);
    }
    if (path.length == 0) {
        path = signingIdentifier; // pid -> path lookup failed; log what we matched on
    }

    if ([flow isKindOfClass:[NEAppProxyTCPFlow class]]) {
        STLogDebug(@"Relaying TCP flow from %@ over the configured interface", path);
        [self relayTCPFlow:(NEAppProxyTCPFlow *)flow];
        return YES;
    }
    if ([flow isKindOfClass:[NEAppProxyUDPFlow class]]) {
        STLogDebug(@"Relaying UDP flow from %@ over the configured interface", path);
        [self relayUDPFlow:(NEAppProxyUDPFlow *)flow];
        return YES;
    }
    STLogDebug(@"Flow from %@ matched but is neither TCP nor UDP, passing through", path);
    return NO;
}

@end
