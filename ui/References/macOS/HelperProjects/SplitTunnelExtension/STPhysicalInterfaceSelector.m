//
//  STPhysicalInterfaceSelector.m
//
#import "STPhysicalInterfaceSelector.h"
#import "STLog.h"
#import <os/lock.h>
#import <stddef.h>
#import <arpa/inet.h>
#import <ifaddrs.h>
#import <net/if.h>
#import <net/if_dl.h>
#import <net/if_types.h>
#import <net/route.h>
#import <netdb.h>
#import <netinet/in.h>
#import <sys/socket.h>
#import <sys/sysctl.h>

// Each sockaddr in a routing-socket message is padded to a 4-byte boundary,
// and a zero-length one still consumes a full slot.
#define ST_SA_ROUNDUP(a) ((a) > 0 ? (1 + (((a) - 1) | (sizeof(uint32_t) - 1))) : sizeof(uint32_t))

// Splits a routing message's packed sockaddr blob into the RTAX_* slots its
// rtm_addrs bitmask says are present.
static void STSplitRouteAddresses(int bitmask, struct sockaddr *sa, struct sockaddr *out[RTAX_MAX]) {
    for (int i = 0; i < RTAX_MAX; i++) {
        if (bitmask & (1 << i)) {
            out[i] = sa;
            sa = (struct sockaddr *)((char *)sa + ST_SA_ROUNDUP(sa->sa_len));
        } else {
            out[i] = NULL;
        }
    }
}

// A /0 netmask arrives either as a zero-length sockaddr or as an all-zero one.
// Anything else (notably the VPN's 128.0.0.0 mask for its "0/1" route) is not /0.
static BOOL STIsZeroNetmask(struct sockaddr *mask) {
    if (mask->sa_len == 0) { return YES; }
    if (mask->sa_len >= sizeof(struct sockaddr_in)) {
        return ((struct sockaddr_in *)mask)->sin_addr.s_addr == 0;
    }
    const unsigned char *bytes = (const unsigned char *)mask;
    for (size_t i = offsetof(struct sockaddr_in, sin_addr); i < mask->sa_len; i++) {
        if (bytes[i] != 0) { return NO; }
    }
    return YES;
}

// Whether `name` is an Ethernet-class link - Wi-Fi, Ethernet, USB tethering and
// Thunderbolt/bridge interfaces all report IFT_ETHER, while utun/ipsec tunnels
// report IFT_OTHER. This is what tells a physical egress apart from a tunnel.
static BOOL STIsPhysicalInterface(const char *name) {
    struct ifaddrs *list = NULL;
    if (getifaddrs(&list) != 0) { return NO; }

    BOOL isPhysical = NO;
    for (struct ifaddrs *ifa = list; ifa != NULL; ifa = ifa->ifa_next) {
        if (!ifa->ifa_addr || ifa->ifa_addr->sa_family != AF_LINK) { continue; }
        if (strcmp(ifa->ifa_name, name) != 0) { continue; }
        isPhysical = (((struct sockaddr_dl *)ifa->ifa_addr)->sdl_type == IFT_ETHER);
        break;
    }
    freeifaddrs(list);
    return isPhysical;
}

// BSD name (e.g. "en0") of the interface owning the IPv4 'default' route, or nil.
//
// Read straight from the kernel routing table - the same source `netstat -rn`
// uses. Several 'default' routes normally coexist, so picking the right one is
// the whole job here:
//
//   - Only Ethernet-class interfaces are considered (STIsPhysicalInterface). A
//     VPN CAN own the unscoped 'default' - verified on a live machine with a
//     third-party tunnel present - so "the first unscoped default" is not a safe
//     rule on its own.
//   - RTF_IFSCOPE entries are accepted, but only as a fallback. macOS re-scopes
//     the physical interface's 'default' to that interface once something else
//     installs an unscoped one, so the correct answer is frequently a scoped
//     route. Tunnels also carry scoped defaults, which the type test rejects.
//   - The /0-netmask test rejects a VPN's '0/1' route (mask 128.0.0.0), which
//     OpenVPN ('redirect-gateway def1') and WireGuard use to capture traffic.
static NSString * _Nullable STDefaultRouteInterfaceName(void) {
    int mib[6] = { CTL_NET, PF_ROUTE, 0, AF_INET, NET_RT_DUMP, 0 };
    size_t needed = 0;
    if (sysctl(mib, 6, NULL, &needed, NULL, 0) < 0 || needed == 0) {
        STLogError(@"Unable to size the routing table: %s", strerror(errno));
        return nil;
    }
    char *buf = malloc(needed);
    if (!buf) { return nil; }
    if (sysctl(mib, 6, buf, &needed, NULL, 0) < 0) {
        STLogError(@"Unable to read the routing table: %s", strerror(errno));
        free(buf);
        return nil;
    }

    NSString *unscoped = nil;
    NSString *scoped = nil;
    char *limit = buf + needed;
    for (char *next = buf; next + sizeof(struct rt_msghdr) <= limit && unscoped == nil; ) {
        struct rt_msghdr *rtm = (struct rt_msghdr *)next;
        if (rtm->rtm_msglen == 0) { break; }
        next += rtm->rtm_msglen;
        if ((rtm->rtm_flags & RTF_UP) == 0) { continue; }

        struct sockaddr *addrs[RTAX_MAX];
        STSplitRouteAddresses(rtm->rtm_addrs, (struct sockaddr *)(rtm + 1), addrs);
        struct sockaddr *dst = addrs[RTAX_DST];
        struct sockaddr *mask = addrs[RTAX_NETMASK];
        if (!dst || !mask || dst->sa_family != AF_INET) { continue; }
        if (((struct sockaddr_in *)dst)->sin_addr.s_addr != 0 || !STIsZeroNetmask(mask)) { continue; }

        char name[IF_NAMESIZE];
        if (if_indextoname(rtm->rtm_index, name) == NULL) { continue; }
        if (!STIsPhysicalInterface(name)) { continue; }

        if (rtm->rtm_flags & RTF_IFSCOPE) {
            if (!scoped) { scoped = [NSString stringWithUTF8String:name]; }
        } else {
            unscoped = [NSString stringWithUTF8String:name];
        }
    }
    free(buf);
    return unscoped ?: scoped;
}

// If `addressOrName` parses as an IPv4/IPv6 literal, looks it up via
// getifaddrs() and returns the BSD name (e.g. "en0") of the interface that
// currently owns it. Otherwise returns `addressOrName` unchanged, already
// assumed to be a name. Returns nil if `addressOrName` is empty, or if it
// looks like an IP but no local interface currently owns it.
// Only used by the `physicalInterface` tier - see -resolveFromOptions:.
static NSString * _Nullable STInterfaceNameForLocalAddress(NSString *addressOrName) {
    if (addressOrName.length == 0) { return nil; }

    struct in_addr v4;
    struct in6_addr v6;
    const char *cstr = addressOrName.UTF8String;
    BOOL looksLikeIP = (inet_pton(AF_INET, cstr, &v4) == 1) || (inet_pton(AF_INET6, cstr, &v6) == 1);
    if (!looksLikeIP) {
        return addressOrName; // already a BSD interface name
    }

    struct ifaddrs *addrs = NULL;
    if (getifaddrs(&addrs) != 0) {
        STLogError(@"getifaddrs failed while resolving %@: %s", addressOrName, strerror(errno));
        return nil;
    }

    NSString *foundName = nil;
    for (struct ifaddrs *ifa = addrs; ifa != NULL; ifa = ifa->ifa_next) {
        if (ifa->ifa_addr == NULL) { continue; }
        int family = ifa->ifa_addr->sa_family;
        if (family != AF_INET && family != AF_INET6) { continue; }

        char host[NI_MAXHOST];
        socklen_t salen = (family == AF_INET) ? sizeof(struct sockaddr_in) : sizeof(struct sockaddr_in6);
        if (getnameinfo(ifa->ifa_addr, salen, host, sizeof(host), NULL, 0, NI_NUMERICHOST) != 0) {
            continue;
        }

        NSString *candidate = [NSString stringWithUTF8String:host];
        // IPv6 link-local addresses come back as "fe80::1%en0" - strip the
        // zone suffix before comparing against the caller's plain address.
        NSRange percent = [candidate rangeOfString:@"%"];
        if (percent.location != NSNotFound) {
            candidate = [candidate substringToIndex:percent.location];
        }
        if ([candidate caseInsensitiveCompare:addressOrName] == NSOrderedSame) {
            foundName = [NSString stringWithUTF8String:ifa->ifa_name];
            break;
        }
    }
    freeifaddrs(addrs);

    if (!foundName) {
        STLogError(@"No local interface currently owns address %@", addressOrName);
    }
    return foundName;
}

// The nw_interface_t named `name` within `path`, or NULL when that interface
// isn't currently part of the path.
static nw_interface_t _Nullable STInterfaceNamedInPath(NSString * _Nullable name, nw_path_t path) {
    if (name.length == 0) { return NULL; }
    __block nw_interface_t found = NULL;
    nw_path_enumerate_interfaces(path, ^bool(nw_interface_t interface) {
        const char *ifName = nw_interface_get_name(interface);
        if (ifName != NULL && strcmp(ifName, name.UTF8String) == 0) {
            found = interface; // ARC retains this strong assignment
            return false;
        }
        return true;
    });
    return found;
}

// ---------------------------------------------------------------------------
// Last-resort default when no `physicalInterfaceType` option is given, the
// default route can't be read, AND neither wired nor Wi-Fi can be confirmed
// active.
// ---------------------------------------------------------------------------
static const nw_interface_type_t kPhysicalInterfaceType = nw_interface_type_wifi;

// Used only when the caller explicitly requested a `physicalInterfaceType`
// value that isn't recognized - the only interface-selection failure that
// still aborts the whole proxy start, since it can never self-heal by
// waiting (see -resolveFromOptions: below).
static NSString * const kSTInterfaceSelectionErrorDomain = @"STProxyProvider.InterfaceSelection";

// Which precedence tier -resolveFromOptions: locked in for this session -
// drives which nw_path_monitor_t(s) -start keeps alive for the rest of it.
typedef NS_ENUM(NSInteger, STInterfaceSelectionMode) {
    STInterfaceSelectionModePinnedName,   // `physicalInterface` option
    STInterfaceSelectionModeExplicitType, // `physicalInterfaceType` option
    STInterfaceSelectionModeAutoDetect,   // default: default-route interface, falling back to wired/Wi-Fi
};

// Satisfied + "has this type reported at least once" for one interface type -
// only meaningful in AutoDetect mode, where wired and Wi-Fi are tracked
// simultaneously to implement the wired-preferred tie-break. Kept together
// since they're always read/written as a pair.
typedef struct {
    BOOL satisfied;
    BOOL known; // guards against treating "not yet reported" as "down" - see -startAutoDetectMonitorForType:
} STTypeLiveness;

@implementation STPhysicalInterfaceSelector {
    // Guards every field below - written from whichever monitor queue's
    // callback fires, read from any relay queue via the public accessors.
    os_unfair_lock _lock;

    STInterfaceSelectionMode _selectionMode;

    // Raw `physicalInterface` option value (name or IP) when _selectionMode is
    // PinnedName - re-resolved on every monitor callback, since the IP-to-name
    // mapping is only meaningful while that address is actually assigned to
    // something. Fixed for the session, no lock needed.
    NSString * _Nullable _pinnedInterfaceIdentifier;

    // Live handle for whichever interface the resolved tier currently names -
    // the `physicalInterface` pin, or the owner of the 'default' route -
    // NULL whenever that interface isn't in the current path.
    nw_interface_t _Nullable _resolvedInterfaceHandle;
    // Name behind the handle above, kept only so the log line below fires on
    // an actual change rather than on every path callback.
    NSString * _Nullable _lastLoggedInterfaceName;
    // Fallback interface type when _resolvedInterfaceHandle is unused -
    // either the explicit `physicalInterfaceType` option, or the
    // wired-preferred auto-detected type kept fresh by -start.
    nw_interface_type_t _requiredInterfaceType;
    // Whether the CURRENTLY active requirement is satisfiable right now -
    // false means relay code must refuse new connections instead of
    // dialing against a requirement that can't succeed.
    BOOL _requirementSatisfied;

    STTypeLiveness _wiredLiveness;
    STTypeLiveness _wifiLiveness;

    // Every monitor -start creates for the resolved tier - only ever
    // cancelled/nilled out together, in -stop, so one array is enough (no
    // need to track which ivar backs which tier).
    NSMutableArray *_activeMonitors;
}

- (instancetype)init {
    self = [super init];
    if (self) {
        _lock = OS_UNFAIR_LOCK_INIT;
        _activeMonitors = [NSMutableArray array];
    }
    return self;
}

#pragma mark - Resolving the tier

// Resolves which physical interface (or interface type) relay connections
// should be pinned to, in this precedence order:
//   1. `physicalInterface` option - a specific interface, given as either a
//      BSD name ("en0") or a local IP address currently assigned to one.
//      A hard requirement: an explicitly requested interface that is down
//      blocks excluded-app traffic rather than silently using another one.
//      NOT CURRENTLY SUPPLIED BY THE HOST - kept working (and proven in the
//      PoC) for a future "always relay over this interface" setting.
//   2. `physicalInterfaceType` option - "wired" or "wifi". A manual override;
//      normally absent.
//   3. Auto-detect (today's only live path): the interface owning the
//      'default' route, re-read from the kernel routing table on every
//      network change - see STDefaultRouteInterfaceName above. Falls back to
//      whichever of wired/Wi-Fi is up (wired preferred) if that lookup fails.
//
// Tier 3 deliberately does NOT take the interface from the host: the daemon
// can only re-resolve it when the VPN state changes, so a plain network
// change (docking from Wi-Fi to Ethernet while WireGuard stays connected)
// would leave a host-supplied value permanently stale. Resolving it here
// means the path monitor that notices the change is also what refreshes the
// answer. Tier 1 is exempt from that reasoning: it is an explicit standing
// choice, not a snapshot of current network state.
//
// This only picks the TIER/parameters - it never blocks or checks live
// availability (that's -start's job, kept alive for the whole session so a
// currently-unavailable interface/type self-heals without a restart).
- (NSError * _Nullable)resolveFromOptions:(NSDictionary<NSString *, id> *)options {
    _resolvedInterfaceHandle = NULL; // ARC releases whatever was previously held here
    _pinnedInterfaceIdentifier = nil;
    _requiredInterfaceType = kPhysicalInterfaceType;
    _requirementSatisfied = NO; // set for real once the first monitor callback lands
    _wiredLiveness = (STTypeLiveness){0};
    _wifiLiveness = (STTypeLiveness){0};

    NSString *identifier = options[@"physicalInterface"];
    if ([identifier isKindOfClass:[NSString class]] && identifier.length > 0) {
        _selectionMode = STInterfaceSelectionModePinnedName;
        _pinnedInterfaceIdentifier = identifier;
        STLogInfo(@"Will pin relay connections to physicalInterface=%@ whenever it's live", identifier);
        return nil;
    }

    NSString *typeOption = options[@"physicalInterfaceType"];
    if ([typeOption isKindOfClass:[NSString class]] && typeOption.length > 0) {
        if ([typeOption caseInsensitiveCompare:@"wired"] == NSOrderedSame) {
            _selectionMode = STInterfaceSelectionModeExplicitType;
            _requiredInterfaceType = nw_interface_type_wired;
            STLogInfo(@"Requiring interface type wired (physicalInterfaceType option)");
            return nil;
        }
        if ([typeOption caseInsensitiveCompare:@"wifi"] == NSOrderedSame) {
            _selectionMode = STInterfaceSelectionModeExplicitType;
            _requiredInterfaceType = nw_interface_type_wifi;
            STLogInfo(@"Requiring interface type Wi-Fi (physicalInterfaceType option)");
            return nil;
        }
        NSString *message = [NSString stringWithFormat:@"Unrecognized physicalInterfaceType=%@ (expected \"wired\" or \"wifi\")", typeOption];
        return [NSError errorWithDomain:kSTInterfaceSelectionErrorDomain code:2 userInfo:@{NSLocalizedDescriptionKey: message}];
    }

    _selectionMode = STInterfaceSelectionModeAutoDetect;
    STLogInfo(@"Auto-detecting the physical interface from the 'default' route (falling back to wired-preferred type), re-evaluated live");
    return nil;
}

#pragma mark - Live monitoring

// Starts whichever persistent nw_path_monitor_t(s) _selectionMode needs to
// keep _pinnedInterfaceHandle/_requiredInterfaceType/_requirementSatisfied
// live for the rest of the session. Callbacks run on their own queue and
// only ever touch state through _lock.
- (void)start {
    dispatch_queue_t queue = dispatch_queue_create("st.interface-monitor", DISPATCH_QUEUE_SERIAL);
    __weak typeof(self) weakSelf = self;

    switch (_selectionMode) {
        case STInterfaceSelectionModePinnedName: {
            NSString *identifier = _pinnedInterfaceIdentifier;
            nw_path_monitor_t monitor = nw_path_monitor_create();
            nw_path_monitor_set_queue(monitor, queue);
            nw_path_monitor_set_update_handler(monitor, ^(nw_path_t path) {
                __strong typeof(self) strongSelf = weakSelf;
                if (!strongSelf) { return; }
                [strongSelf updatePinnedInterfaceFromPath:path identifier:identifier];
            });
            nw_path_monitor_start(monitor);
            [_activeMonitors addObject:monitor];
            break;
        }
        case STInterfaceSelectionModeExplicitType: {
            nw_interface_type_t type = _requiredInterfaceType;
            nw_path_monitor_t monitor = nw_path_monitor_create_with_type(type);
            nw_path_monitor_set_queue(monitor, queue);
            nw_path_monitor_set_update_handler(monitor, ^(nw_path_t path) {
                __strong typeof(self) strongSelf = weakSelf;
                if (!strongSelf) { return; }
                [strongSelf updateExplicitTypeSatisfiedFromPath:path];
            });
            nw_path_monitor_start(monitor);
            [_activeMonitors addObject:monitor];
            break;
        }
        case STInterfaceSelectionModeAutoDetect: {
            nw_path_monitor_t monitor = nw_path_monitor_create();
            nw_path_monitor_set_queue(monitor, queue);
            nw_path_monitor_set_update_handler(monitor, ^(nw_path_t path) {
                __strong typeof(self) strongSelf = weakSelf;
                if (!strongSelf) { return; }
                [strongSelf updateDefaultRouteInterfaceFromPath:path];
            });
            nw_path_monitor_start(monitor);
            [_activeMonitors addObject:monitor];
            // Type monitors run alongside the route lookup, so a failed lookup
            // degrades to wired-preferred detection instead of blocking.
            [_activeMonitors addObject:[self startAutoDetectMonitorForType:nw_interface_type_wired onQueue:queue]];
            [_activeMonitors addObject:[self startAutoDetectMonitorForType:nw_interface_type_wifi onQueue:queue]];
            break;
        }
    }
}

- (void)updateDefaultRouteInterfaceFromPath:(nw_path_t)path {
    // Re-read every callback rather than cached: this is the whole point of
    // resolving it here instead of accepting a host-supplied value.
    NSString *name = STDefaultRouteInterfaceName();
    nw_interface_t found = STInterfaceNamedInPath(name, path);
    NSString *previous;
    os_unfair_lock_lock(&_lock);
    previous = _lastLoggedInterfaceName;
    _resolvedInterfaceHandle = found;
    // Only the positive case is decided here; when the lookup fails the
    // auto-detect type monitors own _requirementSatisfied.
    if (found) { _requirementSatisfied = YES; }
    _lastLoggedInterfaceName = found ? name : nil;
    os_unfair_lock_unlock(&_lock);

    NSString *current = found ? name : nil;
    if ((current || previous) && ![current isEqualToString:previous]) {
        STLogInfo(@"Default route interface is now %@", current ?: @"unresolved - falling back to interface-type detection");
    }
}

// Tier 1 (see -resolveFromOptions:). Unlike the default-route tier, an
// explicitly pinned interface is a hard requirement: when it's down there is
// no fallback, because silently relaying over a different interface would
// ignore the very instruction that selected this tier.
- (void)updatePinnedInterfaceFromPath:(nw_path_t)path identifier:(NSString *)identifier {
    // Re-resolved from scratch every callback (not cached) since the
    // identifier may be an IP that's only meaningful while currently
    // assigned to something - see STInterfaceNameForLocalAddress above.
    nw_interface_t found = STInterfaceNamedInPath(STInterfaceNameForLocalAddress(identifier), path);
    BOOL wasSatisfied;
    os_unfair_lock_lock(&_lock);
    wasSatisfied = _requirementSatisfied;
    _resolvedInterfaceHandle = found;
    _requirementSatisfied = (found != NULL);
    os_unfair_lock_unlock(&_lock);
    if ((found != NULL) != wasSatisfied) {
        STLogInfo(@"physicalInterface=%@ is now %@", identifier, found ? @"live - resuming relay" : @"unavailable - blocking excluded-app traffic until it returns");
    }
}

- (void)updateExplicitTypeSatisfiedFromPath:(nw_path_t)path {
    BOOL satisfied = (nw_path_get_status(path) == nw_path_status_satisfied);
    BOOL wasSatisfied;
    os_unfair_lock_lock(&_lock);
    wasSatisfied = _requirementSatisfied;
    _requirementSatisfied = satisfied;
    os_unfair_lock_unlock(&_lock);
    if (satisfied != wasSatisfied) {
        STLogInfo(@"Required interface type is now %@", satisfied ? @"live - resuming relay" : @"unavailable - blocking excluded-app traffic until it returns");
    }
}

// Shared by both monitors in the AutoDetect case - recomputes the
// wired-preferred required type and overall satisfaction every time either
// type's liveness changes. The two monitors start independently and each
// delivers its own first callback asynchronously, so _requirementSatisfied
// is held at NO until BOTH have reported at least once - otherwise
// whichever fires first would compute against the other's still-default
// (unreported) state, producing a momentary false "both down" reading.
- (nw_path_monitor_t)startAutoDetectMonitorForType:(nw_interface_type_t)type onQueue:(dispatch_queue_t)queue {
    __weak typeof(self) weakSelf = self;
    nw_path_monitor_t monitor = nw_path_monitor_create_with_type(type);
    nw_path_monitor_set_queue(monitor, queue);
    nw_path_monitor_set_update_handler(monitor, ^(nw_path_t path) {
        __strong typeof(self) strongSelf = weakSelf;
        if (!strongSelf) { return; }
        [strongSelf updateAutoDetectLivenessForType:type satisfied:(nw_path_get_status(path) == nw_path_status_satisfied)];
    });
    nw_path_monitor_start(monitor);
    return monitor;
}

- (void)updateAutoDetectLivenessForType:(nw_interface_type_t)type satisfied:(BOOL)satisfied {
    BOOL bothKnown;
    BOOL wiredUp, wifiUp;
    os_unfair_lock_lock(&_lock);
    STTypeLiveness *liveness = (type == nw_interface_type_wired) ? &_wiredLiveness : &_wifiLiveness;
    liveness->satisfied = satisfied;
    liveness->known = YES;
    wiredUp = _wiredLiveness.satisfied;
    wifiUp = _wifiLiveness.satisfied;
    bothKnown = _wiredLiveness.known && _wifiLiveness.known;
    if (bothKnown) {
        _requiredInterfaceType = wiredUp ? nw_interface_type_wired : (wifiUp ? nw_interface_type_wifi : kPhysicalInterfaceType);
        _requirementSatisfied = wiredUp || wifiUp;
    } else {
        _requirementSatisfied = NO; // still waiting on the other type's first report
    }
    // A live default-route interface outranks type liveness - it may sit on an
    // interface reporting as neither wired nor Wi-Fi (USB tether, Thunderbolt bridge).
    if (_resolvedInterfaceHandle) { _requirementSatisfied = YES; }
    os_unfair_lock_unlock(&_lock);

    if (!bothKnown) {
        return;
    }
    STLogInfo(@"Auto-detect: wired=%@ wifi=%@ - using %@", wiredUp ? @"up" : @"down", wifiUp ? @"up" : @"down", wiredUp ? @"wired" : (wifiUp ? @"Wi-Fi" : @"compile-time default (neither confirmed active)"));
}

- (void)stop {
    for (id monitor in _activeMonitors) {
        nw_path_monitor_cancel((nw_path_monitor_t)monitor);
    }
    [_activeMonitors removeAllObjects];
    os_unfair_lock_lock(&_lock);
    _resolvedInterfaceHandle = NULL;
    _lastLoggedInterfaceName = nil;
    _requirementSatisfied = NO;
    os_unfair_lock_unlock(&_lock);
}

#pragma mark - Reading the current requirement

- (BOOL)isCurrentlySatisfied {
    os_unfair_lock_lock(&_lock);
    BOOL satisfied = _requirementSatisfied;
    os_unfair_lock_unlock(&_lock);
    return satisfied;
}

- (void)applyRequirementToParameters:(nw_parameters_t)parameters {
    os_unfair_lock_lock(&_lock);
    nw_interface_t pinned = _resolvedInterfaceHandle;
    nw_interface_type_t requiredType = _requiredInterfaceType;
    os_unfair_lock_unlock(&_lock);
    if (pinned) {
        nw_parameters_require_interface(parameters, pinned);
    } else {
        nw_parameters_set_required_interface_type(parameters, requiredType);
    }
}

@end
