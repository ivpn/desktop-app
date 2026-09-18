//
//  STPhysicalInterfaceSelector.m
//
#import "STPhysicalInterfaceSelector.h"
#import "STLog.h"
#import <os/lock.h>
#import <ifaddrs.h>
#import <arpa/inet.h>
#import <netdb.h>

// If `addressOrName` parses as an IPv4/IPv6 literal, looks it up via
// getifaddrs() and returns the BSD name (e.g. "en0") of the interface that
// currently owns it. Otherwise returns `addressOrName` unchanged, already
// assumed to be a name. Returns nil if `addressOrName` is empty, or if it
// looks like an IP but no local interface currently owns it.
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

// ---------------------------------------------------------------------------
// Last-resort default when no `physicalInterface`/`physicalInterfaceType`
// option is given AND neither wired nor Wi-Fi can be confirmed active.
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
    STInterfaceSelectionModeAutoDetect,   // neither option given
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
    // Raw `physicalInterface` option value (name or IP) when _selectionMode
    // is Pinned - re-resolved to a live interface on every monitor callback,
    // since the IP-to-name mapping is only meaningful while that address is
    // actually assigned to something. Fixed for the session, no lock needed.
    NSString * _Nullable _pinnedInterfaceIdentifier;

    // Live handle for the interface named by _pinnedInterfaceIdentifier,
    // kept fresh by the persistent monitor started in -start - NULL
    // whenever that interface is currently absent.
    nw_interface_t _Nullable _pinnedInterfaceHandle;
    // Fallback interface type when _pinnedInterfaceHandle is unused - either
    // the explicit `physicalInterfaceType` option, or the wired-preferred
    // auto-detected type kept fresh by -start.
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
//   2. `physicalInterfaceType` option - "wired" or "wifi".
//   3. Auto-detected: only reached if NEITHER of the above was given -
//      whichever of wired/Wi-Fi is actually up right now (wired preferred if
//      both are), continuously re-evaluated for the rest of the session.
//
// This only picks the TIER/parameters - it never blocks or checks live
// availability (that's -start's job, kept alive for the whole session so a
// currently-unavailable interface/type self-heals without a restart).
- (NSError * _Nullable)resolveFromOptions:(NSDictionary<NSString *, id> *)options {
    _pinnedInterfaceHandle = NULL; // ARC releases whatever was previously held here
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
    STLogInfo(@"Auto-detecting physical interface type (wired preferred over Wi-Fi), re-evaluated live");
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
            [_activeMonitors addObject:[self startAutoDetectMonitorForType:nw_interface_type_wired onQueue:queue]];
            [_activeMonitors addObject:[self startAutoDetectMonitorForType:nw_interface_type_wifi onQueue:queue]];
            break;
        }
    }
}

- (void)updatePinnedInterfaceFromPath:(nw_path_t)path identifier:(NSString *)identifier {
    // Re-resolved from scratch every callback (not cached) since the
    // identifier may be an IP that's only meaningful while currently
    // assigned to something - see STInterfaceNameForLocalAddress above.
    NSString *name = STInterfaceNameForLocalAddress(identifier);
    __block nw_interface_t found = NULL;
    if (name) {
        nw_path_enumerate_interfaces(path, ^bool(nw_interface_t interface) {
            const char *ifName = nw_interface_get_name(interface);
            if (ifName != NULL && strcmp(ifName, name.UTF8String) == 0) {
                found = interface; // ARC retains this strong assignment
                return false;
            }
            return true;
        });
    }
    BOOL wasSatisfied;
    os_unfair_lock_lock(&_lock);
    wasSatisfied = _requirementSatisfied;
    _pinnedInterfaceHandle = found;
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
    _pinnedInterfaceHandle = NULL;
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
    nw_interface_t pinned = _pinnedInterfaceHandle;
    nw_interface_type_t requiredType = _requiredInterfaceType;
    os_unfair_lock_unlock(&_lock);
    if (pinned) {
        nw_parameters_require_interface(parameters, pinned);
    } else {
        nw_parameters_set_required_interface_type(parameters, requiredType);
    }
}

@end
