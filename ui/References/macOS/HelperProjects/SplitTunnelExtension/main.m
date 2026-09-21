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
#import <errno.h>
#import <grp.h>
#import <string.h>
#import <unistd.h>
#import "STLog.h"

// The group is created by the IVPN daemon. The IVPN firewall lets the traffic of this
// group leave over the physical interface while every other way out of the tunnel stays
// blocked, so the switch must happen before any socket is created.
static void SwitchToSplitTunnelGroup(void) {
    static const char *groupName = "ivpn-st";

    struct group *grp = getgrnam(groupName);
    if (grp == NULL) {
        STLogError(@"Group '%s' not found - relayed traffic will be blocked while the IVPN firewall is enabled", groupName);
        return;
    }
    if (setgid(grp->gr_gid) != 0) {
        STLogError(@"setgid(%d) failed (%s) - relayed traffic will be blocked while the IVPN firewall is enabled",
                   (int)grp->gr_gid, strerror(errno));
        return;
    }
    STLogInfo(@"Running under group '%s' (gid %d)", groupName, (int)grp->gr_gid);
}

int main(int argc, char *argv[]) {
    @autoreleasepool {
        // Default handler: mirror every log message to the unified logging
        // system, under a dedicated subsystem so it's actually selectable -
        // OS_LOG_DEFAULT logs under an empty subsystem, which a predicate
        // would never match. Subsystem is derived from the bundle's own
        // CFBundleIdentifier rather than hardcoded, so it can't drift out of
        // sync with whatever this bundle is actually signed/installed as.
        // `log stream --predicate 'subsystem == "<this bundle's id>"' --level debug`
        // (`log stream --predicate 'subsystem == "com.electron.ivpn-ui.SplitTunnel"' --level debug`)
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

        SwitchToSplitTunnelGroup();

        [NEProvider startSystemExtensionMode];
    }
    // Keep this process alive forever; all real work happens on background
    // queues driven by the NetworkExtension runtime from here on.
    [[NSRunLoop mainRunLoop] run];
    return 0;
}
