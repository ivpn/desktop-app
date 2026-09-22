//
//  STPathMatchingTests.m
//
//  Standalone tests for the decision logic in STPathMatching.m, run by
//  Tests/run_tests.sh. Deliberately no XCTest: the extension has no Xcode
//  project, and these need nothing but Foundation and real child processes.
//
#import <Foundation/Foundation.h>
#import <libproc.h>
#import <signal.h>
#import "../STPathMatching.h"

static int gFailures = 0;

#define EXPECT(cond, ...) do { \
    if (!(cond)) { gFailures++; NSLog(@"FAIL line %d: %@", __LINE__, [NSString stringWithFormat:__VA_ARGS__]); } \
    else { NSLog(@"ok   %s", #cond); } \
} while (0)

// Path of this test binary, resolved the same way the extension resolves
// flows - so the tests exercise the real pid -> path lookup too.
static NSString *SelfPath(void) {
    return STExecutablePathForPid(getpid());
}

// Runs `script` under /bin/bash and returns the bash pid. The caller kills it.
static NSTask *StartBash(NSString *script, NSPipe *stdoutPipe) {
    NSTask *task = [NSTask new];
    task.executableURL = [NSURL fileURLWithPath:@"/bin/bash"];
    task.arguments = @[@"-c", script];
    if (stdoutPipe) { task.standardOutput = stdoutPipe; }
    NSError *error = nil;
    if (![task launchAndReturnError:&error]) {
        NSLog(@"cannot launch bash: %@", error);
        exit(2);
    }
    return task;
}

// First live child of `parent`, found by scanning the process table
// (exactly like the ppid walk under test, in the other direction), polling
// briefly until bash has forked it.
static pid_t FirstChildOf(pid_t parent) {
    for (int attempt = 0; attempt < 200; attempt++) {
        pid_t pids[4096] = {0};
        int count = proc_listpids(PROC_ALL_PIDS, 0, pids, sizeof(pids)) / (int)sizeof(pid_t);
        for (int i = 0; i < count; i++) {
            struct proc_bsdinfo info;
            if (pids[i] > 0 &&
                proc_pidinfo(pids[i], PROC_PIDTBSDINFO, 0, &info, sizeof(info)) == (int)sizeof(info) &&
                (pid_t)info.pbi_ppid == parent) {
                return pids[i];
            }
        }
        usleep(5000);
    }
    return 0;
}

// kill(0, ...) would signal our own process group - never do that on a
// failed spawn.
static void KillIfSpawned(pid_t pid) {
    if (pid > 0) { kill(pid, SIGKILL); }
}

static void TestPathMatching(void) {
    NSArray *excluded = @[@"/Applications/Firefox.app", @"/usr/bin/curl", @"/", @"/usr/local"];
    EXPECT(STPathMatchesAny(@"/Applications/Firefox.app/Contents/MacOS/firefox", excluded), @"bundle prefix");
    EXPECT(STPathMatchesAny(@"/usr/bin/curl", excluded), @"bare executable, exact");
    EXPECT(!STPathMatchesAny(@"/usr/bin/curl-config", excluded), @"bare executable is not a prefix");
    EXPECT(!STPathMatchesAny(@"/Applications/Firefox.app.bak/x", excluded), @"bundle prefix needs a separator");
    EXPECT(!STPathMatchesAny(@"/usr/local/bin/tool", excluded), @"plain directory is not a prefix");
    EXPECT(!STPathMatchesAny(@"/bin/ls", excluded), @"root directory is not a prefix");
    EXPECT(!STPathMatchesAny(nil, excluded), @"nil path");
    EXPECT(!STPathMatchesAny(@"/x", @[]), @"empty list");
}

static void TestResolvedSymlinks(void) {
    // <tmp>/Real.app/Contents/MacOS, <tmp>/Link.app -> Real.app
    NSString *dir = [NSTemporaryDirectory() stringByAppendingPathComponent:[NSUUID UUID].UUIDString];
    NSString *real = [dir stringByAppendingPathComponent:@"Real.app"];
    NSString *link = [dir stringByAppendingPathComponent:@"Link.app"];
    NSFileManager *fm = NSFileManager.defaultManager;
    [fm createDirectoryAtPath:[real stringByAppendingPathComponent:@"Contents/MacOS"] withIntermediateDirectories:YES attributes:nil error:nil];
    [fm createSymbolicLinkAtPath:link withDestinationPath:@"Real.app" error:nil];
    // NSTemporaryDirectory() itself may be a symlink (/var -> /private/var).
    char buf[PATH_MAX];
    NSString *realResolved = realpath(real.fileSystemRepresentation, buf) ? [NSString stringWithUTF8String:buf] : real;

    NSArray *excluded = STPathsWithResolvedSymlinks(@[link, @"/usr/bin/curl", @"/nonexistent/x"]);
    EXPECT([excluded containsObject:link], @"literal entry kept");
    EXPECT([excluded containsObject:realResolved], @"symlink resolved: %@", excluded);
    EXPECT([excluded containsObject:@"/nonexistent/x"], @"nonexistent entry kept");
    EXPECT(STPathMatchesAny([realResolved stringByAppendingPathComponent:@"Contents/MacOS/x"], excluded), @"process under the real bundle matches the symlinked entry");
    EXPECT(STPathsWithResolvedSymlinks(@[@"/usr/bin/curl"]).count == 1, @"regular file is not duplicated");
    [fm removeItemAtPath:dir error:nil];
}

static void TestDirectChild(void) {
    // bash -> sleep, with bash kept alive by the trailing command so it
    // forks rather than execs sleep.
    NSTask *bash = StartBash(@"sleep 30; true", nil);
    pid_t sleepPid = FirstChildOf(bash.processIdentifier);
    EXPECT(sleepPid > 0, @"grandchild spawned");

    NSString *self = SelfPath();
    EXPECT([STAncestorPathMatchingAny(sleepPid, @[self]) isEqualToString:self], @"grandchild matches test binary");
    EXPECT([STAncestorPathMatchingAny(sleepPid, @[@"/bin/bash"]) isEqualToString:@"/bin/bash"], @"grandchild matches intermediate bare executable");
    EXPECT([STAncestorPathMatchingAny(bash.processIdentifier, @[self]) isEqualToString:self], @"direct child matches test binary");
    EXPECT(STAncestorPathMatchingAny(sleepPid, @[@"/nonexistent/App.app"]) == nil, @"no match for unrelated path");
    EXPECT(STAncestorPathMatchingAny(sleepPid, @[]) == nil, @"empty list never matches");
    EXPECT(STAncestorPathMatchingAny(0, @[self]) == nil, @"pid 0 never matches");
    EXPECT(STAncestorPathMatchingAny(1, @[self]) == nil, @"launchd never matches");
    EXPECT(STAncestorPathMatchingAny(getpid(), @[self]) == nil, @"a process is not its own ancestor");

    KillIfSpawned(sleepPid);
    [bash terminate];
    [bash waitUntilExit];
}

static void TestDetachedChild(void) {
    // bash starts sleep in the background, prints its pid and exits: sleep
    // is reparented to launchd, so the ppid chain no longer leads anywhere.
    NSPipe *pipe = [NSPipe pipe];
    // sleep must not inherit the pipe, or its EOF would only come when sleep ends.
    NSTask *bash = StartBash(@"sleep 30 >/dev/null 2>&1 & echo $!", pipe);
    NSData *out = [pipe.fileHandleForReading readDataToEndOfFile];
    [bash waitUntilExit];
    pid_t sleepPid = (pid_t)[[[NSString alloc] initWithData:out encoding:NSUTF8StringEncoding] intValue];
    EXPECT(sleepPid > 0, @"detached child spawned");

    struct proc_bsdinfo info = {0};
    proc_pidinfo(sleepPid, PROC_PIDTBSDINFO, 0, &info, sizeof(info));
    EXPECT(info.pbi_ppid == 1, @"detached child was reparented to launchd (ppid=%u)", info.pbi_ppid);

    // Only the responsible-process tier can still attribute it. The test
    // binary is itself attributed to whatever launched it (Terminal, an IDE),
    // so that is the ancestor the tier must find.
    pid_t responsible = STResponsibleProcessForPid(getpid());
    if (responsible <= 1 || responsible == getpid()) {
        NSLog(@"skip: responsible-process API unavailable or test not launched from an app (responsible=%d)", responsible);
    } else {
        NSString *responsiblePath = STExecutablePathForPid(responsible);
        EXPECT([STAncestorPathMatchingAny(sleepPid, @[responsiblePath]) isEqualToString:responsiblePath],
               @"detached child still attributed to %@", responsiblePath);
        EXPECT(STAncestorPathMatchingAny(sleepPid, @[SelfPath()]) == nil,
               @"detached child is not attributed to the exited chain");
    }
    KillIfSpawned(sleepPid);
}

int main(void) {
    @autoreleasepool {
        TestPathMatching();
        TestResolvedSymlinks();
        TestDirectChild();
        TestDetachedChild();
        NSLog(@"%@", gFailures == 0 ? @"ALL TESTS PASSED" : [NSString stringWithFormat:@"%d FAILURE(S)", gFailures]);
    }
    return gFailures == 0 ? 0 : 1;
}
