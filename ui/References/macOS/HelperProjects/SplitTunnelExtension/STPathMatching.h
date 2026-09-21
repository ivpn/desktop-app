//
//  STPathMatching.h
//
//  Turning an intercepted flow into "which app created this?" and testing
//  that against the excluded-app list - the actual decision logic behind
//  split tunneling, independent of the NetworkExtension plumbing around it.
//
#import <Foundation/Foundation.h>
#import <NetworkExtension/NetworkExtension.h>

// NEAppProxyFlow only exposes a low-level, kernel-issued "audit token" for
// the process that owns it (flow.metaData.sourceAppAuditToken). This is the
// Apple-documented way to turn that into a PID (0 if the token is missing or
// malformed)...
pid_t STPidForFlow(NEAppProxyFlow * _Nonnull flow);

// ...and then into an absolute executable path we can actually compare
// against the excluded-app list. nil if the process is gone or inaccessible.
NSString * _Nullable STExecutablePathForPid(pid_t pid);

// Matches a bare executable path exactly, and a ".app" bundle path as a
// PREFIX - so e.g. excluding "/Applications/Firefox.app" also catches
// helper/XPC binaries inside "Contents/MacOS/...".
BOOL STPathMatchesAny(NSString * _Nullable path, NSArray<NSString *> * _Nonnull excludedPaths);

// Exclusion is inherited: a process started by an excluded app runs on its
// behalf (Terminal -> curl, Steam -> a game outside the bundle, an IDE -> its
// build tools) and must take the same route.
//
// Returns the executable path of the first process `pid` runs on behalf of
// that matches `excludedPaths`, or nil. Two sources are consulted, in order:
//  1. the kernel's "responsible process" - the same attribution TCC uses for
//     its "Terminal wants to access..." prompts. It survives the parent
//     exiting (a detached child is reparented to launchd, which breaks the
//     ppid chain, but keeps its responsible process). Private API, resolved
//     with dlsym; silently skipped when unavailable.
//  2. the parent-pid chain up to launchd (public API), guarded against pid
//     reuse: a "parent" that started after its child is a recycled pid.
// Cost is a handful of syscalls per flow, no I/O, no allocation on the
// common path except the ancestor paths themselves.
//
// Not covered, by design of the OS: processes launched through
// LaunchServices (`open`, double-click, XPC services, login items) have
// launchd as parent and are their own responsible process.
NSString * _Nullable STAncestorPathMatchingAny(pid_t pid, NSArray<NSString *> * _Nonnull excludedPaths);

// The kernel's responsible process for `pid`, or 0 when unavailable.
// Exposed for the tests.
pid_t STResponsibleProcessForPid(pid_t pid);

// Reads the CFBundleIdentifier of every ".app" in `paths` (bare executables
// and unreadable bundles are skipped). Call this once, when the excluded-app
// list changes - reading Info.plist per flow would be far too expensive.
//
// The result is the secondary match key for
// `flow.metaData.sourceAppSigningIdentifier`, which the kernel delivers with
// the flow itself and so does not depend on the pid -> path lookup above.
// That lookup is pid-reuse sensitive and simply fails for short-lived
// processes, so an excluded app's flow can arrive with no usable path at
// all; the signing identifier also keeps matching an app that was relocated
// or updated in place.
NSSet<NSString *> * _Nonnull STBundleIdentifiersForPaths(NSArray<NSString *> * _Nullable paths);
