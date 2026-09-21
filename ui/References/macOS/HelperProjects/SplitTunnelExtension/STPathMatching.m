//
//  STPathMatching.m
//
#import "STPathMatching.h"
#import <bsm/libbsm.h>      // audit_token_to_pid()
#import <dlfcn.h>           // dlsym()
#import <libproc.h>         // proc_pidpath(), proc_pidinfo()
#import <sys/proc_info.h>   // struct proc_bsdinfo

pid_t STPidForFlow(NEAppProxyFlow *flow) {
    NSData *tokenData = flow.metaData.sourceAppAuditToken;
    if (tokenData.length != sizeof(audit_token_t)) {
        return 0;
    }
    audit_token_t token;
    [tokenData getBytes:&token length:sizeof(token)];
    pid_t pid = audit_token_to_pid(token);
    return pid > 0 ? pid : 0;
}

NSString * STExecutablePathForPid(pid_t pid) {
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

#pragma mark - Process ancestry

pid_t STResponsibleProcessForPid(pid_t pid) {
    // Private libsystem symbol (no SDK header), so it is looked up once and
    // treated as optional: if a future macOS drops it, we lose only this
    // tier and fall back to the ppid chain below.
    static pid_t (*responsibleForPid)(pid_t) = NULL;
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        responsibleForPid = (pid_t (*)(pid_t))dlsym(RTLD_DEFAULT, "responsibility_get_pid_responsible_for_pid");
    });
    if (responsibleForPid == NULL || pid <= 0) {
        return 0;
    }
    pid_t responsible = responsibleForPid(pid);
    return responsible > 0 ? responsible : 0;
}

static BOOL STReadProcessInfo(pid_t pid, struct proc_bsdinfo *info) {
    return proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, info, sizeof(*info)) == (int)sizeof(*info);
}

// A process cannot be older than its parent or its responsible process. If
// it is, the pid we read was recycled and now names an unrelated process.
static BOOL STStartedAfter(const struct proc_bsdinfo *a, const struct proc_bsdinfo *b) {
    if (a->pbi_start_tvsec != b->pbi_start_tvsec) {
        return a->pbi_start_tvsec > b->pbi_start_tvsec;
    }
    return a->pbi_start_tvusec > b->pbi_start_tvusec;
}

// Bounds the ppid walk. Real chains are 1-5 deep (app -> helper -> shell ->
// tool); this is a safety net, not an expected depth.
static const int kMaxAncestryDepth = 16;

NSString * STAncestorPathMatchingAny(pid_t pid, NSArray<NSString *> *excludedPaths) {
    if (pid <= 1 || excludedPaths.count == 0) {
        return nil;
    }

    struct proc_bsdinfo child;
    if (!STReadProcessInfo(pid, &child)) {
        return nil; // the process is already gone
    }

    // 1. Responsible process. One syscall, and the most likely match: for a
    //    whole tree of helpers it is the top-level app itself. The kernel
    //    keeps this pid after the process exits, hence the start-time check.
    pid_t responsible = STResponsibleProcessForPid(pid);
    if (responsible > 1 && responsible != pid) {
        struct proc_bsdinfo info;
        if (STReadProcessInfo(responsible, &info) && !STStartedAfter(&info, &child)) {
            NSString *path = STExecutablePathForPid(responsible);
            if (STPathMatchesAny(path, excludedPaths)) {
                return path;
            }
        }
    }

    // 2. Parent chain, for excluded bare executables in the middle of a tree
    //    and as the fallback when the responsible-process API is missing.
    for (int depth = 0; depth < kMaxAncestryDepth; depth++) {
        pid_t parentPid = child.pbi_ppid;
        if (parentPid <= 1) {
            return nil; // reached launchd
        }
        struct proc_bsdinfo parent;
        if (!STReadProcessInfo(parentPid, &parent) || STStartedAfter(&parent, &child)) {
            return nil; // parent exited, or its pid was recycled
        }
        NSString *path = STExecutablePathForPid(parentPid);
        if (STPathMatchesAny(path, excludedPaths)) {
            return path;
        }
        child = parent;
    }
    return nil;
}

#pragma mark - Bundle identifiers

NSSet<NSString *> * STBundleIdentifiersForPaths(NSArray<NSString *> *paths) {
    NSMutableSet<NSString *> *identifiers = [NSMutableSet set];
    for (NSString *path in paths) {
        NSString *trimmed = [path hasSuffix:@"/"] ? [path substringToIndex:path.length - 1] : path;
        if (![trimmed hasSuffix:@".app"]) { continue; }
        NSString *identifier = [NSBundle bundleWithPath:trimmed].bundleIdentifier;
        if (identifier.length > 0) {
            [identifiers addObject:identifier];
        }
    }
    return identifiers;
}
