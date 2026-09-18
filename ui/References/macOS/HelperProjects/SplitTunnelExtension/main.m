//
//  main.m - entry point for the SplitTunnelExtension system extension.
//
//  Unlike a normal app or command-line tool, a NetworkExtension system
//  extension does not "do work" from main() itself. Instead:
//    1. [NEProvider startSystemExtensionMode] registers this process with
//       the OS's NetworkExtension runtime.
//    2. The OS then reads this bundle's Info.plist (NEProviderClasses key)
//       to find out which class to instantiate - STProxyProvider, in our
//       case - and creates it FOR us.
//    3. From then on, the OS calls methods on that instance
//       (startProxyWithOptions:, handleNewFlow:, etc.) whenever appropriate.
//
//  We must never call `[[STProxyProvider alloc] init]` ourselves.
//
#import <Foundation/Foundation.h>
#import <NetworkExtension/NetworkExtension.h>
#import <os/log.h>
#import "STLog.h"

int main(int argc, char *argv[]) {
    @autoreleasepool {
        // Default handler: mirror every log message to the unified logging
        // system, under a dedicated subsystem so it's actually selectable -
        // OS_LOG_DEFAULT logs under an empty subsystem, which a predicate
        // would never match. Subsystem is derived from the bundle's own
        // CFBundleIdentifier rather than hardcoded, so it can't drift out of
        // sync with whatever this bundle is actually signed/installed as.
        // `log stream --predicate 'subsystem == "<this bundle's id>"' --level debug`
        // shows this regardless of whether the host app's window is open.
        NSString *subsystem = [NSBundle mainBundle].bundleIdentifier ?: @"SplitTunnelExtension";
        os_log_t extensionLog = os_log_create(subsystem.UTF8String, "proxy");

        STLogSetHandler(^(STLogLevel level, NSString *message) {
            os_log_type_t type = OS_LOG_TYPE_DEFAULT;
            if (level == STLogLevelError) { type = OS_LOG_TYPE_ERROR; }
            else if (level == STLogLevelDebug) { type = OS_LOG_TYPE_DEBUG; }
            os_log_with_type(extensionLog, type, "%{public}@", message);
        });

        STLogSetMinLevel(STLogLevelDebug);

        // Info: No setgid() here: pf can't except NAT rules by group, so firewall.sh
        //       skips NAT/route-to entirely while Split Tunnel is enabled instead.

        [NEProvider startSystemExtensionMode];
    }
    // Keep this process alive forever; all real work happens on background
    // queues driven by the NetworkExtension runtime from here on.
    [[NSRunLoop mainRunLoop] run];
    return 0;
}
