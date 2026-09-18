//
//  STLog.m
//
#import "STLog.h"
#import <os/lock.h>
#import <stdatomic.h>

static STLogHandler _Nullable gHandler;
static os_unfair_lock gHandlerLock = OS_UNFAIR_LOCK_INIT;
static _Atomic STLogLevel gMinLevel = STLogLevelInfo;

void STLogSetHandler(STLogHandler _Nullable handler) {
    STLogHandler copied = [handler copy];
    os_unfair_lock_lock(&gHandlerLock);
    gHandler = copied;
    os_unfair_lock_unlock(&gHandlerLock);
}

void STLogSetMinLevel(STLogLevel level) {
    atomic_store_explicit(&gMinLevel, level, memory_order_relaxed);
}

void STLogMessage(STLogLevel level, NSString *format, ...) {
    // Cheapest possible rejection first: no lock, no capture of the handler,
    // no formatting - just an atomic read, so a raised/lowered threshold is
    // effectively free for every call site below it (e.g. STLogDebug in a
    // per-flow/per-packet hot path).
    if (level < atomic_load_explicit(&gMinLevel, memory_order_relaxed)) {
        return;
    }

    os_unfair_lock_lock(&gHandlerLock);
    STLogHandler handler = gHandler;
    os_unfair_lock_unlock(&gHandlerLock);
    if (!handler) {
        return; // no handler installed - skip formatting entirely
    }

    va_list args;
    va_start(args, format);
    NSString *message = [[NSString alloc] initWithFormat:format arguments:args];
    va_end(args);
    handler(level, message);
}
