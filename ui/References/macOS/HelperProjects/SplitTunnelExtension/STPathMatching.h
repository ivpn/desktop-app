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
