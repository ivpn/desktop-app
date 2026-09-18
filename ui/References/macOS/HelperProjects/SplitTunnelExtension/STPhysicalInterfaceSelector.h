//
//  STPhysicalInterfaceSelector.h
//
//  Owns "which physical interface (or type) should relay connections use"
//  end to end: resolving the `physicalInterface`/`physicalInterfaceType`
//  start options into a precedence tier, and then keeping that tier's
//  live availability up to date for the rest of the session via persistent
//  nw_path_monitor_t(s) - see STPhysicalInterfaceSelector.m for the tier
//  precedence and self-healing rationale. STProxyProvider owns one instance
//  of this and never touches interface-selection state directly; the
//  TCP/UDP relay code only ever calls -isCurrentlySatisfied and
//  -applyRequirementToParameters:.
//
#import <Foundation/Foundation.h>
#import <Network/Network.h>

NS_ASSUME_NONNULL_BEGIN

@interface STPhysicalInterfaceSelector : NSObject

// Validates `options` and picks a precedence tier (explicit interface,
// explicit type, or auto-detect) - does not block or check live
// availability (that's -start's job). Returns an NSError only for a
// genuinely malformed `physicalInterfaceType` value, which can never
// self-heal by waiting; every other case (including "requested
// interface/type not currently available") returns nil and self-heals once
// -start's monitors confirm it's live.
- (NSError * _Nullable)resolveFromOptions:(NSDictionary<NSString *, id> *)options;

// Starts the persistent nw_path_monitor_t(s) the resolved tier needs to
// keep availability live for the rest of the session. Call after
// -resolveFromOptions: returns nil.
- (void)start;

// Cancels any active monitors and resets availability to unsatisfied. Safe
// to call even if -start was never called.
- (void)stop;

// Whether the currently resolved requirement is satisfiable right now -
// false means relay code should refuse the flow immediately instead of
// dialing against a requirement that can't succeed (see
// STProxyProvider+TCPRelay.m / +UDPRelay.m).
- (BOOL)isCurrentlySatisfied;

// Applies whichever interface requirement is currently resolved (pinned
// interface, explicit/auto-detected type, or the compile-time default) to a
// freshly-created set of connection parameters.
- (void)applyRequirementToParameters:(nw_parameters_t)parameters;

@end

NS_ASSUME_NONNULL_END
