//
//  STLog.h
//
//  Single logging seam for this extension: every log call funnels through
//  STLogMessage() (normally via the STLogError/STLogInfo/STLogDebug macros
//  below). With no handler installed, logging has zero effect - the format
//  string isn't even evaluated - and behavior is controlled entirely by
//  whatever handler STLogSetHandler() installs.
//
#import <Foundation/Foundation.h>

NS_ASSUME_NONNULL_BEGIN

typedef NS_ENUM(NSInteger, STLogLevel) {
    STLogLevelDebug,
    STLogLevelInfo,
    STLogLevelError,
};

typedef void (^STLogHandler)(STLogLevel level, NSString *message);

// Pass nil to disable logging again. Safe to call from any thread.
void STLogSetHandler(STLogHandler _Nullable handler);

// Messages below this level are skipped before any locking or formatting -
// default is STLogLevelInfo, so STLogDebug is a no-op until raised. Safe to
// call from any thread.
void STLogSetMinLevel(STLogLevel level);

// Formats and delivers one message, only if a handler is installed and
// `level` is at or above the configured minimum. Prefer the macros below
// over calling this directly.
void STLogMessage(STLogLevel level, NSString *format, ...) NS_FORMAT_FUNCTION(2, 3);

#define STLogError(fmt, ...) STLogMessage(STLogLevelError, (fmt), ##__VA_ARGS__)
#define STLogInfo(fmt, ...)  STLogMessage(STLogLevelInfo,  (fmt), ##__VA_ARGS__)
#define STLogDebug(fmt, ...) STLogMessage(STLogLevelDebug, (fmt), ##__VA_ARGS__)

NS_ASSUME_NONNULL_END
