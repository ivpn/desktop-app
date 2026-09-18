//
//  STProxyProvider+Private.h
//
//  Shared private surface for STProxyProvider.m and its relay
//  implementation files (STProxyProvider+TCPRelay.m / +UDPRelay.m) - the
//  provider's bookkeeping state lives here so all three files can see it,
//  while STProxyProvider.h keeps exposing only the NETransparentProxyProvider
//  overrides to the rest of the world.
//
#import "STProxyProvider.h"
#import <Network/Network.h>
#import <os/lock.h>

NS_ASSUME_NONNULL_BEGIN

@class STPhysicalInterfaceSelector;

// Per-flow UDP relay state: one physical-interface nw_connection per remote
// endpoint the app has sent datagrams to (UDP is connectionless, so a single
// app-level flow can talk to many different peers over its lifetime).
// `connectionForKey:`/etc. are called both from the app-facing flow's own
// read callback (an OS-managed queue) and from each peer connection's own
// callback (this instance's `queue`) - two different execution contexts, so
// the internal dictionary is lock-protected rather than assumed confined to
// one queue.
@interface STUDPFlowState : NSObject
// All nw_connection callbacks for every peer under this flow run here - one
// queue per flow, so different flows relay concurrently but one flow's own
// peers stay ordered/thread-safe relative to each other.
@property (nonatomic, strong, readonly) dispatch_queue_t queue;
- (nw_connection_t _Nullable)connectionForKey:(NSString *)key;
- (void)setConnection:(nw_connection_t)connection forKey:(NSString *)key;
// Marks a key as having just carried traffic (send or receive) - resets its
// idle clock so the watchdog/LRU eviction below leave it alone.
- (void)touchKey:(NSString *)key;
- (void)removeConnectionForKey:(NSString *)key;
// Removes and returns the least-recently-touched connection, or nil if
// empty - used to make room when a flow's peer cap is reached.
- (nw_connection_t _Nullable)evictLeastRecentlyUsedConnection;
// Removes and returns every connection idle (no send/receive) for at least
// `maxIdle` seconds - used by the periodic watchdog in STProxyProvider.m.
- (NSArray<id> *)evictConnectionsIdleLongerThan:(NSTimeInterval)maxIdle;
- (NSArray<id> *)allConnections;
- (NSUInteger)connectionCount;
@end

@interface STProxyProvider () {
    // Guards activeFlows/tcpConnectionsByFlow/udpFlowStates below - touched
    // both synchronously from handleNewFlow:'s calling context and
    // asynchronously from per-flow nw_connection callbacks, so plain
    // NSMutableSet/NSMapTable mutation is not safe without it. Use the
    // st_* accessor methods declared below instead of the ivars directly.
    os_unfair_lock _flowsLock;
    // Periodic timer that evicts idle UDP peer connections across every
    // active flow - started in -startProxyWithOptions:, cancelled in
    // -stopProxyWithReason:. The system never calls those two methods
    // concurrently with each other, so this needs no lock of its own.
    dispatch_source_t _Nullable _udpIdleWatchdog;
}
// Owns interface/type resolution and its live-monitoring self-healing (see
// STPhysicalInterfaceSelector.h) - the only interface-selection state
// STProxyProvider (and the relay categories, via the two methods below)
// ever touches.
@property (nonatomic, strong) STPhysicalInterfaceSelector *interfaceSelector;
// We MUST keep a strong reference to every flow we take ownership of.
// Nothing else in the system retains it once handleNewFlow: returns, so
// without this, ARC would deallocate it mid-transfer.
@property (nonatomic, strong) NSMutableSet<NEAppProxyFlow *> *activeFlows;
// Per-UDP-flow relay state (see STUDPFlowState above). Keyed by identity,
// not -isEqual:/-hash, since NEAppProxyFlow doesn't implement NSCopying.
@property (nonatomic, strong) NSMapTable<NEAppProxyUDPFlow *, STUDPFlowState *> *udpFlowStates;
// The physical-interface nw_connection for each currently relayed TCP flow -
// needed so -closeAllActiveFlows can force-cancel it directly.
@property (nonatomic, strong) NSMapTable<NEAppProxyTCPFlow *, id> *tcpConnectionsByFlow;
// The apps currently excluded from the VPN. Replaced wholesale (never
// mutated in place) so reads from any relay queue don't need extra locking.
@property (atomic, copy) NSArray<NSString *> *excludedPaths;
// CFBundleIdentifiers derived from excludedPaths, kept in sync with it -
// the secondary match key, see STBundleIdentifiersForPaths().
@property (atomic, copy) NSSet<NSString *> *excludedBundleIdentifiers;

// Lock-protected access to activeFlows/tcpConnectionsByFlow/udpFlowStates -
// every touch of those three collections must go through these instead of
// the properties above directly.
- (void)st_registerTCPFlow:(NEAppProxyTCPFlow *)flow connection:(nw_connection_t)connection;
// Returns NO if the flow was already unregistered - the caller is then a
// duplicate teardown and must do nothing more (see -teardownTCPFlow:connection:).
- (BOOL)st_unregisterTCPFlow:(NEAppProxyTCPFlow *)flow;
- (nw_connection_t _Nullable)st_connectionForTCPFlow:(NEAppProxyTCPFlow *)flow;
- (void)st_registerUDPFlow:(NEAppProxyUDPFlow *)flow state:(STUDPFlowState *)state;
- (void)st_unregisterUDPFlow:(NEAppProxyUDPFlow *)flow;
- (STUDPFlowState * _Nullable)st_stateForUDPFlow:(NEAppProxyUDPFlow *)flow;
- (NSArray<NEAppProxyFlow *> *)st_activeFlowsSnapshot;

@end

// Declared as a category (not folded into the class extension above) so the
// compiler doesn't expect these to be implemented alongside the main
// @implementation STProxyProvider block - they're implemented in
// STProxyProvider+TCPRelay.m / +UDPRelay.m instead.
@interface STProxyProvider (Relaying)
- (void)relayTCPFlow:(NEAppProxyTCPFlow *)flow;
- (void)relayUDPFlow:(NEAppProxyUDPFlow *)flow;
- (void)teardownTCPFlow:(NEAppProxyTCPFlow *)flow connection:(nw_connection_t)connection;
- (void)teardownUDPFlow:(NEAppProxyUDPFlow *)flow state:(STUDPFlowState *)state;
@end

NS_ASSUME_NONNULL_END
