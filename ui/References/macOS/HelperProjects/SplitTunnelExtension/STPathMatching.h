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
// Apple-documented way to turn that into a PID, and then into an absolute
// executable path we can actually compare against the excluded-app list.
NSString * _Nullable STExecutablePathForFlow(NEAppProxyFlow * _Nonnull flow);

// Matches a bare executable path exactly, and a ".app" bundle path as a
// PREFIX - so e.g. excluding "/Applications/Firefox.app" also catches
// helper/XPC binaries inside "Contents/MacOS/...".
BOOL STPathMatchesAny(NSString * _Nullable path, NSArray<NSString *> * _Nonnull excludedPaths);

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
