//
//  STPathMatching.m
//
#import "STPathMatching.h"
#import <bsm/libbsm.h>   // audit_token_to_pid()
#import <libproc.h>      // proc_pidpath()

NSString * STExecutablePathForFlow(NEAppProxyFlow *flow) {
    NSData *tokenData = flow.metaData.sourceAppAuditToken;
    if (tokenData.length != sizeof(audit_token_t)) {
        return nil;
    }
    audit_token_t token;
    [tokenData getBytes:&token length:sizeof(token)];
    pid_t pid = audit_token_to_pid(token);
    if (pid <= 0) {
        return nil;
    }
    char pathBuffer[PROC_PIDPATHINFO_MAXSIZE] = {0};
    if (proc_pidpath(pid, pathBuffer, sizeof(pathBuffer)) <= 0) {
        return nil;
    }
    return [NSString stringWithUTF8String:pathBuffer];
}

BOOL STPathMatchesAny(NSString *path, NSArray<NSString *> *excludedPaths) {
    if (path.length == 0) { return NO; }
    for (NSString *candidate in excludedPaths) {
        if ([path isEqualToString:candidate]) { return YES; }
        NSString *prefix = [candidate hasSuffix:@"/"] ? candidate : [candidate stringByAppendingString:@"/"];
        if ([path hasPrefix:prefix]) { return YES; }
    }
    return NO;
}
