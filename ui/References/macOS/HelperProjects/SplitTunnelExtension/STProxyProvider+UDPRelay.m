//
//  STProxyProvider+UDPRelay.m
//
//  UDP has no "connection" the way TCP does at the app-facing side: a single
//  flow's readDatagramsWithCompletionHandler: can hand back datagrams
//  destined for many different remote peers, all interleaved. So instead of
//  one physical-interface connection per flow, we keep one PER (flow,
//  remote endpoint) pair, all still pinned to the physical interface.
//
#import "STProxyProvider+Private.h"
#import "STPhysicalInterfaceSelector.h"
#import "STLog.h"

// One peer connection plus when it last carried traffic - the unit the idle
// watchdog and LRU eviction both operate on.
@interface STUDPPeerEntry : NSObject
@property (nonatomic, strong) id connection; // nw_connection_t
@property (nonatomic, assign) NSTimeInterval lastActivity;
@end

@implementation STUDPPeerEntry
@end

// See the class-level comment in STProxyProvider+Private.h for why this
// needs its own lock rather than relying solely on `queue`.
@interface STUDPFlowState () {
    os_unfair_lock _lock;
}
@property (nonatomic, strong) NSMutableDictionary<NSString *, STUDPPeerEntry *> *connectionsByEndpointKey;
@end

@implementation STUDPFlowState

- (instancetype)init {
    self = [super init];
    if (self) {
        _lock = OS_UNFAIR_LOCK_INIT;
        _queue = dispatch_queue_create("relay.udp", DISPATCH_QUEUE_SERIAL);
        _connectionsByEndpointKey = [NSMutableDictionary dictionary];
    }
    return self;
}

- (nw_connection_t)connectionForKey:(NSString *)key {
    os_unfair_lock_lock(&_lock);
    nw_connection_t connection = self.connectionsByEndpointKey[key].connection;
    os_unfair_lock_unlock(&_lock);
    return connection;
}

- (void)setConnection:(nw_connection_t)connection forKey:(NSString *)key {
    STUDPPeerEntry *entry = [[STUDPPeerEntry alloc] init];
    entry.connection = connection;
    entry.lastActivity = [NSDate timeIntervalSinceReferenceDate];
    os_unfair_lock_lock(&_lock);
    self.connectionsByEndpointKey[key] = entry;
    os_unfair_lock_unlock(&_lock);
}

- (void)touchKey:(NSString *)key {
    os_unfair_lock_lock(&_lock);
    self.connectionsByEndpointKey[key].lastActivity = [NSDate timeIntervalSinceReferenceDate];
    os_unfair_lock_unlock(&_lock);
}

- (void)removeConnectionForKey:(NSString *)key {
    os_unfair_lock_lock(&_lock);
    [self.connectionsByEndpointKey removeObjectForKey:key];
    os_unfair_lock_unlock(&_lock);
}

- (nw_connection_t)evictLeastRecentlyUsedConnection {
    os_unfair_lock_lock(&_lock);
    __block NSString *oldestKey = nil;
    __block STUDPPeerEntry *oldestEntry = nil;
    [self.connectionsByEndpointKey enumerateKeysAndObjectsUsingBlock:^(NSString *key, STUDPPeerEntry *entry, BOOL *stop) {
        if (!oldestEntry || entry.lastActivity < oldestEntry.lastActivity) {
            oldestKey = key;
            oldestEntry = entry;
        }
    }];
    if (oldestKey) {
        [self.connectionsByEndpointKey removeObjectForKey:oldestKey];
    }
    os_unfair_lock_unlock(&_lock);
    return oldestEntry.connection;
}

- (NSArray<id> *)evictConnectionsIdleLongerThan:(NSTimeInterval)maxIdle {
    NSTimeInterval now = [NSDate timeIntervalSinceReferenceDate];
    NSMutableArray<id> *evicted = [NSMutableArray array];
    NSMutableArray<NSString *> *staleKeys = [NSMutableArray array];
    os_unfair_lock_lock(&_lock);
    [self.connectionsByEndpointKey enumerateKeysAndObjectsUsingBlock:^(NSString *key, STUDPPeerEntry *entry, BOOL *stop) {
        if (now - entry.lastActivity >= maxIdle) {
            [staleKeys addObject:key];
            [evicted addObject:entry.connection];
        }
    }];
    for (NSString *key in staleKeys) { [self.connectionsByEndpointKey removeObjectForKey:key]; }
    os_unfair_lock_unlock(&_lock);
    return evicted;
}

- (NSArray<id> *)allConnections {
    os_unfair_lock_lock(&_lock);
    NSArray<STUDPPeerEntry *> *entries = [self.connectionsByEndpointKey allValues];
    os_unfair_lock_unlock(&_lock);
    NSMutableArray<id> *connections = [NSMutableArray arrayWithCapacity:entries.count];
    for (STUDPPeerEntry *entry in entries) { [connections addObject:entry.connection]; }
    return connections;
}

- (NSUInteger)connectionCount {
    os_unfair_lock_lock(&_lock);
    NSUInteger count = self.connectionsByEndpointKey.count;
    os_unfair_lock_unlock(&_lock);
    return count;
}

@end

@implementation STProxyProvider (UDPRelay)

- (void)relayUDPFlow:(NEAppProxyUDPFlow *)flow {
    STUDPFlowState *state = [[STUDPFlowState alloc] init];
    [self st_registerUDPFlow:flow state:state];

    [flow openWithLocalEndpoint:nil completionHandler:^(NSError * _Nullable openError) {
        if (openError) {
            STLogError(@"Failed to open UDP flow: %@", openError);
            [self st_unregisterUDPFlow:flow];
            return;
        }
        STLogDebug(@"UDP flow opened - relaying datagrams");
        [self pumpUDPFlow:flow state:state];
    }];
}

// Reads datagrams (and their destination endpoints) from the app, and for
// each one finds-or-creates the physical-interface connection to that
// specific peer, then keeps reading more datagrams. There is no "EOF" for
// UDP the way there is for TCP; this loop only stops on error/close.
- (void)pumpUDPFlow:(NEAppProxyUDPFlow *)flow state:(STUDPFlowState *)state {
    __weak typeof(self) weakSelf = self;
    [flow readDatagramsWithCompletionHandler:^(NSArray<NSData *> * _Nullable datagrams,
                                                NSArray<NWEndpoint *> * _Nullable remoteEndpoints,
                                                NSError * _Nullable error) {
        @autoreleasepool {
        __strong typeof(self) strongSelf = weakSelf;
        if (!strongSelf) { return; }

        if (error || datagrams.count == 0) {
            STLogDebug(@"UDP flow read ended (error=%@), tearing down %lu peer connection(s)",
                  error, (unsigned long)state.connectionCount);
            [strongSelf teardownUDPFlow:flow state:state];
            return;
        }

        for (NSUInteger i = 0; i < datagrams.count; i++) {
            NWHostEndpoint *remote = (NWHostEndpoint *)remoteEndpoints[i];
            if (![remote isKindOfClass:[NWHostEndpoint class]]) { continue; }
            [strongSelf sendUDPDatagram:datagrams[i] toEndpoint:remote flow:flow state:state];
        }
        [strongSelf pumpUDPFlow:flow state:state]; // keep reading more datagrams
        }
    }];
}

// Finds (or lazily creates) the physical-interface connection for one
// specific remote peer, then sends this datagram over it.
- (void)sendUDPDatagram:(NSData *)data
              toEndpoint:(NWHostEndpoint *)remote
                    flow:(NEAppProxyUDPFlow *)flow
                   state:(STUDPFlowState *)state {
    // Bounded, but generous - the idle watchdog (see STProxyProvider.m) is
    // what actually keeps long-lived flows from accumulating stale peers;
    // this cap is just a backstop against a single flow going unbounded.
    static const NSUInteger kMaxUDPPeersPerFlow = 256;

    if (remote.hostname.length == 0 || remote.port.length == 0) {
        STLogError(@"Dropping UDP datagram with incomplete remote endpoint (%@:%@)", remote.hostname, remote.port);
        return;
    }

    if (![self.interfaceSelector isCurrentlySatisfied]) {
        // Same rationale as the TCP side (see STProxyProvider+TCPRelay.m) -
        // the selected physical interface/type is currently unavailable, so
        // drop the datagram instead of creating a peer connection that can't
        // currently succeed. Self-heals once the interface/type returns.
        STLogDebug(@"Physical interface/type currently unavailable, dropping UDP datagram to %@:%@", remote.hostname, remote.port);
        return;
    }

    NSString *key = [[remote.hostname stringByAppendingString:@":"] stringByAppendingString:remote.port];
    nw_connection_t existing = [state connectionForKey:key];
    if (existing) {
        [state touchKey:key];
        [self sendData:data overConnection:existing label:key];
        return;
    }

    if (state.connectionCount >= kMaxUDPPeersPerFlow) {
        nw_connection_t evicted = [state evictLeastRecentlyUsedConnection];
        if (evicted) {
            STLogDebug(@"UDP peer cap reached for this flow, evicting the least recently used peer");
            nw_connection_cancel(evicted);
        } else {
            STLogError(@"UDP peer cap reached for this flow, dropping datagram to %@", key);
            return;
        }
    }

    nw_endpoint_t endpoint = nw_endpoint_create_host(remote.hostname.UTF8String, remote.port.UTF8String);
    nw_parameters_t parameters = nw_parameters_create_secure_udp(
        NW_PARAMETERS_DISABLE_PROTOCOL,      // no DTLS at this layer - relay raw datagrams
        NW_PARAMETERS_DEFAULT_CONFIGURATION);
    [self.interfaceSelector applyRequirementToParameters:parameters]; // same trick as TCP

    nw_connection_t connection = nw_connection_create(endpoint, parameters);
    // Shared by every peer connection of this one flow (see STUDPFlowState) -
    // different flows still get their own queue and relay concurrently.
    nw_connection_set_queue(connection, state.queue);
    [state setConnection:connection forKey:key];

    __weak typeof(self) weakSelf = self;
    nw_connection_set_state_changed_handler(connection, ^(nw_connection_state_t connState, nw_error_t connError) {
        __strong typeof(self) strongSelf = weakSelf;
        if (!strongSelf) { return; }

        if (connState == nw_connection_state_ready) {
            [strongSelf sendData:data overConnection:connection label:key];
            [strongSelf pumpUDPConnection:connection endpoint:remote flow:flow state:state key:key];
        } else if (connState == nw_connection_state_waiting) {
            STLogDebug(@"UDP peer connection %@ is waiting for connectivity on the physical interface: %@", key, connError);
        } else if (connState == nw_connection_state_failed || connState == nw_connection_state_cancelled) {
            STLogDebug(@"UDP peer connection %@ ended (state=%ld, error=%@)", key, (long)connState, connError);
            [state removeConnectionForKey:key];
        }
    });
    nw_connection_start(connection);
}

- (void)sendData:(NSData *)data overConnection:(nw_connection_t)connection label:(NSString *)label {
    // Custom destructor just keeps `data` alive until the send completes,
    // instead of DISPATCH_DATA_DESTRUCTOR_DEFAULT's unconditional memcpy of
    // every datagram relayed.
    dispatch_data_t payload = dispatch_data_create(data.bytes, data.length,
                                                    dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT, 0),
                                                    ^{ (void)data; });
    nw_connection_send(connection, payload, NW_CONNECTION_DEFAULT_MESSAGE_CONTEXT, true, ^(nw_error_t sendError) {
        if (sendError) {
            STLogError(@"UDP send to %@ over physical interface failed: %@", label, sendError);
        }
    });
}

// Reads replies from one specific peer over the physical interface and
// writes them back to the app, tagged with that same peer's endpoint so the
// app sees the reply as coming from the right place.
- (void)pumpUDPConnection:(nw_connection_t)connection
                  endpoint:(NWHostEndpoint *)remote
                      flow:(NEAppProxyUDPFlow *)flow
                     state:(STUDPFlowState *)state
                       key:(NSString *)key {
    // 65535 is the largest possible single UDP datagram - datagrams are
    // already message-bounded by the OS, so this is a documented ceiling
    // rather than a behavior change from a naive UINT32_MAX request.
    static const uint32_t kMaxUDPDatagramLength = 65535;
    __weak typeof(self) weakSelf = self;
    nw_connection_receive(connection, 1, kMaxUDPDatagramLength,
        ^(dispatch_data_t _Nullable content, nw_content_context_t _Nullable context, bool isComplete, nw_error_t _Nullable receiveError) {
        @autoreleasepool {
        __strong typeof(self) strongSelf = weakSelf;
        if (!strongSelf) { return; }

        if (content) {
            NSData *data = (NSData *)content;
            [state touchKey:key];
            [flow writeDatagrams:@[data] sentByEndpoints:@[remote] completionHandler:^(NSError * _Nullable writeError) {
                if (writeError) {
                    STLogError(@"Failed to write UDP data back to the app: %@", writeError);
                }
            }];
        }

        if (isComplete || receiveError) {
            // The dictionary entry alone isn't enough - without an explicit
            // cancel, this connection's own state-changed handler block
            // keeps it retained forever (a real socket/memory leak, not
            // just a theoretical one - the "failed"/"cancelled" branch above
            // never fires on a *graceful* completion like this).
            nw_connection_cancel(connection);
            [state removeConnectionForKey:key];
            return;
        }
        [strongSelf pumpUDPConnection:connection endpoint:remote flow:flow state:state key:key]; // keep receiving
        }
    });
}

- (void)teardownUDPFlow:(NEAppProxyUDPFlow *)flow state:(STUDPFlowState *)state {
    for (id connection in [state allConnections]) {
        nw_connection_cancel((nw_connection_t)connection);
    }
    [flow closeReadWithError:nil];
    [flow closeWriteWithError:nil];
    [self st_unregisterUDPFlow:flow];
}

@end
