//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2026 IVPN Limited.
//
//  This file is part of the UI for IVPN Client Desktop.
//
//  The UI for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The UI for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the UI for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

// Drives the macOS Split Tunnel system extension's lifecycle
// (OSSystemExtensionManager) and its NETransparentProxyManager /
// NETunnelProviderSession proxy session from the Electron main process - the
// daemon (root LaunchDaemon) cannot call either of these APIs itself, since
// both require a user-session process living inside an app bundle.
//
// Node addon examples:
//    https://github.com/nodejs/node-addon-examples

#include <node_api.h>
#include <string.h>

void runJSStateChangedCallback(const char *jsonUTF8);

//=========================================================================
// OBJECTIVE-C CODE
//=========================================================================

#import <Foundation/Foundation.h>
#import <NetworkExtension/NetworkExtension.h>
#import <SystemExtensions/SystemExtensions.h>

// The extension's bundle id is always the host app's own bundle id with this
// suffix appended (a hard OS requirement - the system extension's bundle id
// must be prefixed by its containing app's). Derived at runtime from
// [NSBundle mainBundle] rather than hardcoded, so it can never drift out of
// sync with whatever identity this build is actually signed as.
static NSString * const kSplitTunnelExtensionSuffix = @".SplitTunnel";

typedef NS_ENUM(NSInteger, STExtState) {
    STExtStateNotInstalled,
    STExtStateInstalling,
    STExtStateNeedsUserApproval,
    STExtStateNeedsReboot,
    STExtStateInstalled,
    STExtStateDisabled,
    STExtStateError,
};

static NSString *StringForExtState(STExtState s) {
    switch (s) {
        case STExtStateNotInstalled:      return @"notInstalled";
        case STExtStateInstalling:        return @"installing";
        case STExtStateNeedsUserApproval: return @"needsUserApproval";
        case STExtStateNeedsReboot:       return @"needsReboot";
        case STExtStateInstalled:         return @"installed";
        case STExtStateDisabled:          return @"disabled";
        case STExtStateError:             return @"error";
    }
    return @"error";
}

static NSString *StringForVPNStatus(NEVPNStatus status) {
    switch (status) {
        case NEVPNStatusDisconnected:  return @"disconnected";
        case NEVPNStatusConnecting:    return @"connecting";
        case NEVPNStatusConnected:     return @"connected";
        case NEVPNStatusReasserting:   return @"reasserting";
        case NEVPNStatusDisconnecting: return @"disconnecting";
        case NEVPNStatusInvalid:
        default:                       return @"invalid";
    }
}

static NSDictionary *ParseJSONDictionary(NSString *json) {
    NSData *data = [json dataUsingEncoding:NSUTF8StringEncoding];
    if (!data) { return @{}; }
    id obj = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
    return [obj isKindOfClass:[NSDictionary class]] ? obj : @{};
}

@interface STSplitTunnelController : NSObject <OSSystemExtensionRequestDelegate>
+ (instancetype)sharedInstance;
- (NSString *)extensionStateString;
- (NSString *)sessionStatusString;
- (NSString *)lastErrorMessage;
- (void)activateExtension;
- (void)deactivateExtension;
- (void)refreshExtensionState;
- (void)registerConfiguration;
- (void)applyConfigJSON:(NSString *)json;
- (void)stopSession;
@end

@implementation STSplitTunnelController {
    STExtState _extState;
    NSString * _Nullable _lastError;
    // Set by -request:actionForReplacingExtension:withExtension: when the OS
    // is about to swap in a newer bundled extension. Consumed once the
    // replacement actually finishes, to restart any active session so the
    // new binary takes over immediately - the OS does not do this on its own.
    BOOL _pendingReplacementRestart;
    // The activation request from submit until it finishes or fails - which,
    // while the OS waits for the user's approval, can be a long time. No
    // properties probe may be submitted meanwhile: the OS would cancel the
    // activation as superseded, and its completion is what reports approval.
    OSSystemExtensionRequest * _Nullable _activationRequest;
    BOOL _observingVPNStatus;
    NETransparentProxyManager * _Nullable _lastManager;
    NSDictionary * _Nullable _lastOptions;
    // Start options waiting for the session to finish tearing down; consumed by
    // -vpnStatusDidChange: (see -reconcileSessionWithOptions:).
    NSDictionary * _Nullable _pendingStartOptions;
    // Properties requests (see -refreshExtensionState) in flight - they share
    // the delegate callbacks below with activation requests, so they are told
    // apart by request identity. A set, not a single pointer: a probe whose
    // completion never arrives must not block every later probe.
    NSMutableSet<OSSystemExtensionRequest *> *_propertiesRequests;
    // Bumped synchronously on entry to -applyConfigJSON: and -stopSession. Their
    // asynchronous completions act only if still current, so of several
    // commands issued in quick succession only the last one takes effect.
    NSUInteger _commandGeneration;
}

+ (instancetype)sharedInstance {
    static STSplitTunnelController *instance = nil;
    static dispatch_once_t once;
    dispatch_once(&once, ^{ instance = [[self alloc] initPrivate]; });
    return instance;
}

- (instancetype)initPrivate {
    self = [super init];
    if (self) {
        _extState = STExtStateNotInstalled;
        _propertiesRequests = [NSMutableSet set];
    }
    return self;
}

- (NSString *)extensionBundleID {
    NSString *hostID = [NSBundle mainBundle].bundleIdentifier;
    if (hostID.length == 0) { hostID = @"com.electron.ivpn-ui"; } // should not happen in a packaged app
    return [hostID stringByAppendingString:kSplitTunnelExtensionSuffix];
}

- (NSString *)extensionStateString { return StringForExtState(_extState); }
- (NSString *)lastErrorMessage { return _lastError ?: @""; }

#pragma mark - Extension activation

// Safe to call unconditionally, on every app launch: if the bundled
// extension's CFBundleVersion matches what's already activated, the OS
// finishes immediately with no visible effect; a mismatch is what drives
// -request:actionForReplacingExtension:withExtension: below. This is the
// mechanism that keeps an installed extension from silently staying on an
// old version after an app update.
- (void)activateExtension {
    // A second activation while the first is still validating makes sysextd
    // treat it as a same-version replacement conflict; both then fail.
    if (_activationRequest) { return; }
    _extState = STExtStateInstalling;
    [self notifyStateChanged];

    OSSystemExtensionRequest *request = [OSSystemExtensionRequest activationRequestForExtension:[self extensionBundleID]
                                                                                          queue:dispatch_get_main_queue()];
    request.delegate = self;
    _activationRequest = request;
    [[OSSystemExtensionManager sharedManager] submitRequest:request];
}

- (void)deactivateExtension {
    NSString *bundleID = [self extensionBundleID];
    void (^submitDeactivationRequest)(void) = ^{
        OSSystemExtensionRequest *request = [OSSystemExtensionRequest deactivationRequestForExtension:bundleID
                                                                                                queue:dispatch_get_main_queue()];
        request.delegate = self;
        [[OSSystemExtensionManager sharedManager] submitRequest:request];
    };

    // Also drop the VPN-preferences registration first - otherwise a stale
    // NETransparentProxyManager entry survives even after the extension itself
    // is deactivated below. Best-effort: proceeds to deactivation either way.
    [self findManagerWithCompletion:^(NETransparentProxyManager * _Nullable found, NSError * _Nullable error) {
        if (!found) { submitDeactivationRequest(); return; }
        [found removeFromPreferencesWithCompletionHandler:^(NSError * _Nullable removeError) {
            submitDeactivationRequest();
        }];
    }];
}

// Asks the OS for the extension's actual state. Without this, _extState is a
// process-local cache that starts at "notInstalled" on every launch, so an
// already-installed extension stays misreported until some request completes.
// Unlike an activation request, this never prompts the user.
- (void)refreshExtensionState {
    if (_activationRequest) { return; } // the activation request owns the state until it completes (see its declaration)

    OSSystemExtensionRequest *request = [OSSystemExtensionRequest propertiesRequestForExtension:[self extensionBundleID]
                                                                                          queue:dispatch_get_main_queue()];
    request.delegate = self;
    [_propertiesRequests addObject:request];
    [[OSSystemExtensionManager sharedManager] submitRequest:request];
}

#pragma mark - OSSystemExtensionRequestDelegate

- (OSSystemExtensionReplacementAction)request:(OSSystemExtensionRequest *)request
                   actionForReplacingExtension:(OSSystemExtensionProperties *)existing
                                 withExtension:(OSSystemExtensionProperties *)ext {
    _pendingReplacementRestart = YES;
    return OSSystemExtensionReplacementActionReplace;
}

- (void)requestNeedsUserApproval:(OSSystemExtensionRequest *)request {
    _extState = STExtStateNeedsUserApproval;
    [self notifyStateChanged];
}

// Only sent for a properties request (-refreshExtensionState).
- (void)request:(OSSystemExtensionRequest *)request foundProperties:(NSArray<OSSystemExtensionProperties *> *)properties {
    // A probe submitted just before an activation answers after it: its
    // "not installed" is stale and would trigger another activation.
    if (_activationRequest) { return; }
    STExtState state = STExtStateNotInstalled;
    for (OSSystemExtensionProperties *p in properties) {
        if (p.isUninstalling) { continue; }
        if (p.isEnabled) { state = STExtStateInstalled; break; }
        // Present but not enabled: still waiting for the user's approval, or
        // switched off by the user in System Settings after it was enabled.
        state = p.isAwaitingUserApproval ? STExtStateNeedsUserApproval : STExtStateDisabled;
    }
    if (state == _extState) { return; }
    NSLog(@"[split-tunnel-macos] extension state %@ -> %@ (%lu entries)", StringForExtState(_extState), StringForExtState(state), (unsigned long)properties.count);
    _extState = state;
    [self notifyStateChanged];
}

- (void)request:(OSSystemExtensionRequest *)request didFailWithError:(NSError *)error {
    NSLog(@"[split-tunnel-macos] extension request failed: %@", error);
    if ([_propertiesRequests containsObject:request]) { [_propertiesRequests removeObject:request]; return; } // probe failed - keep the known state
    if (request == _activationRequest) { _activationRequest = nil; }
    // Superseded by a newer request for the same extension: that request now
    // owns the state, nothing went wrong.
    if ([error.domain isEqualToString:OSSystemExtensionErrorDomain] && error.code == OSSystemExtensionErrorRequestSuperseded) { return; }
    _extState = STExtStateError;
    _lastError = error.localizedDescription;
    [self notifyStateChanged];
}

- (void)request:(OSSystemExtensionRequest *)request didFinishWithResult:(OSSystemExtensionRequestResult)result {
    if ([_propertiesRequests containsObject:request]) { [_propertiesRequests removeObject:request]; return; } // state already applied in -request:foundProperties:
    if (request == _activationRequest) { _activationRequest = nil; }
    _lastError = nil; // the request succeeded - don't keep reporting an error from an earlier attempt
    if (result == OSSystemExtensionRequestWillCompleteAfterReboot) {
        _extState = STExtStateNeedsReboot;
        [self notifyStateChanged];
        return;
    }
    _extState = STExtStateInstalled;
    [self notifyStateChanged];

    if (_pendingReplacementRestart) {
        _pendingReplacementRestart = NO;
        [self restartSessionIfActive];
    }
}

#pragma mark - Session lifecycle (NETransparentProxyManager / NETunnelProviderSession)

// Finds the proxy configuration already registered with the OS, if any.
// Never creates one: saving a new configuration raises the system's
// "IVPN would like to add proxy configurations" authorization prompt.
- (void)findManagerWithCompletion:(void (^)(NETransparentProxyManager * _Nullable, NSError * _Nullable))completion {
    NSString *bundleID = [self extensionBundleID];
    [NETransparentProxyManager loadAllFromPreferencesWithCompletionHandler:^(NSArray<NETransparentProxyManager *> * _Nullable managers, NSError * _Nullable error) {
        if (error) { completion(nil, error); return; }
        for (NETransparentProxyManager *m in managers) {
            NETunnelProviderProtocol *proto = (NETunnelProviderProtocol *)m.protocolConfiguration;
            if ([proto.providerBundleIdentifier isEqualToString:bundleID]) { completion(m, nil); return; }
        }
        completion(nil, nil);
    }];
}

- (void)loadOrCreateManagerWithCompletion:(void (^)(NETransparentProxyManager * _Nullable, NSError * _Nullable))completion {
    NSString *bundleID = [self extensionBundleID];
    [self findManagerWithCompletion:^(NETransparentProxyManager * _Nullable found, NSError * _Nullable error) {
        if (error) { completion(nil, error); return; }

        // Always the config actually registered with the OS, never an
        // in-memory pointer kept across calls - the session is a
        // system-level configuration that outlives this process.
        NETransparentProxyManager *manager = found ?: [[NETransparentProxyManager alloc] init];
        if (!found) {
            NETunnelProviderProtocol *protocolConfig = [[NETunnelProviderProtocol alloc] init];
            protocolConfig.providerBundleIdentifier = bundleID;
            protocolConfig.serverAddress = @"127.0.0.1"; // required by the API, unused by a transparent proxy
            manager.protocolConfiguration = protocolConfig;
            manager.localizedDescription = @"IVPN Split Tunnel";
        }
        manager.enabled = YES;

        // This only registers/updates the proxy config in the OS's VPN
        // preferences database - it does NOT launch the extension process
        // or call -[STProxyProvider startProxyWithOptions:] on it. That only
        // happens later, from -reconcileSessionWithOptions: below, via
        // `session startTunnelWithOptions:`.
        [manager saveToPreferencesWithCompletionHandler:^(NSError * _Nullable saveError) {
            if (saveError) { completion(nil, saveError); return; }
            // Reload after save - `manager.connection` can be stale otherwise.
            [manager loadFromPreferencesWithCompletionHandler:^(NSError * _Nullable loadError) {
                completion(loadError ? nil : manager, loadError);
            }];
        }];
    }];
}

- (void)ensureObservingVPNStatus {
    if (_observingVPNStatus) { return; }
    _observingVPNStatus = YES;
    [[NSNotificationCenter defaultCenter] addObserver:self
                                              selector:@selector(vpnStatusDidChange:)
                                                  name:NEVPNStatusDidChangeNotification
                                                object:nil];
}

- (void)vpnStatusDidChange:(NSNotification *)note {
    if (_pendingStartOptions) {
        NETunnelProviderSession *session = (NETunnelProviderSession *)_lastManager.connection;
        if (session.status == NEVPNStatusDisconnected || session.status == NEVPNStatusInvalid) {
            NSDictionary *options = _pendingStartOptions;
            _pendingStartOptions = nil;
            [self startSessionWithOptions:options];
        }
    }
    [self notifyStateChanged];
}

- (NSString *)sessionStatusString {
    NETunnelProviderSession *session = (NETunnelProviderSession *)_lastManager.connection;
    return StringForVPNStatus(session.status);
}

// Registers the proxy configuration with the OS without starting a session.
// The first save raises the system's "would like to add proxy configurations"
// prompt; calling this as soon as the extension is installed shows that prompt
// while the user is still enabling Split Tunnel, not on the first VPN connect.
- (void)registerConfiguration {
    __weak typeof(self) weakSelf = self;
    [self loadOrCreateManagerWithCompletion:^(NETransparentProxyManager * _Nullable manager, NSError * _Nullable error) {
        __strong typeof(self) strongSelf = weakSelf;
        if (!strongSelf) { return; }
        if (manager) {
            strongSelf->_lastManager = manager;
            strongSelf->_lastError = nil;
        } else {
            strongSelf->_lastError = error.localizedDescription ?: @"Unable to load the Split Tunnel proxy configuration";
        }
        [strongSelf notifyStateChanged];
    }];
}

// `json` is the resolved Split Tunnel config the daemon computed, forwarded
// verbatim as start options.
- (void)applyConfigJSON:(NSString *)json {
    NSDictionary *cfg = ParseJSONDictionary(json);
    _lastOptions = cfg;
    _pendingStartOptions = nil; // superseded: only the options below may start a session
    NSUInteger generation = ++_commandGeneration;

    __weak typeof(self) weakSelf = self;
    [self loadOrCreateManagerWithCompletion:^(NETransparentProxyManager * _Nullable manager, NSError * _Nullable error) {
        __strong typeof(self) strongSelf = weakSelf;
        if (!strongSelf || generation != strongSelf->_commandGeneration) { return; }
        if (!manager) {
            NSLog(@"[split-tunnel-macos] proxy configuration load/save failed: %@", error);
            strongSelf->_lastError = error.localizedDescription ?: @"Unable to load the Split Tunnel proxy configuration";
            [strongSelf notifyStateChanged];
            return;
        }
        strongSelf->_lastManager = manager;
        [strongSelf ensureObservingVPNStatus];
        [strongSelf reconcileSessionWithOptions:cfg];
    }];
}

- (void)stopSession {
    _pendingStartOptions = nil; // an explicit stop cancels a pending restart
    NSUInteger generation = ++_commandGeneration;

    __weak typeof(self) weakSelf = self;
    // Load-only: no configuration registered means there is nothing to stop,
    // and creating one here would prompt every user on every launch.
    [self findManagerWithCompletion:^(NETransparentProxyManager * _Nullable manager, NSError * _Nullable error) {
        __strong typeof(self) strongSelf = weakSelf;
        if (!strongSelf || !manager || generation != strongSelf->_commandGeneration) { return; }
        strongSelf->_lastManager = manager;
        [strongSelf ensureObservingVPNStatus];
        // Triggers -[STProxyProvider stopProxyWithReason:completionHandler:] on the extension side.
        NETunnelProviderSession *session = (NETunnelProviderSession *)manager.connection;
        [session stopTunnel];
    }];
}

// Called after a confirmed extension replacement, so the newly activated
// binary actually takes over an already-running session instead of being
// left idle until the next unrelated settings change.
- (void)restartSessionIfActive {
    if (!_lastManager) { return; }
    NETunnelProviderSession *session = (NETunnelProviderSession *)_lastManager.connection;
    if (session.status == NEVPNStatusInvalid || session.status == NEVPNStatusDisconnected) { return; }
    [self reconcileSessionWithOptions:_lastOptions ?: @{}];
}

// Always a full stop-then-restart: NETransparentProxyProvider does not
// support live in-place option updates (see the class comment in
// STProxyProvider.m for why sendProviderMessage/handleAppMessage: is not
// used here), so every settings change - app list, mode, or interface -
// goes through this same path.
- (void)reconcileSessionWithOptions:(NSDictionary *)options {
    NETunnelProviderSession *session = (NETunnelProviderSession *)_lastManager.connection;
    if (session.status == NEVPNStatusDisconnected || session.status == NEVPNStatusInvalid) {
        [self startSessionWithOptions:options];
        return;
    }

    // Restarting can only be done once the provider has actually torn down, and how
    // long that takes is up to the provider - so wait for the status change rather
    // than guessing a delay (-vpnStatusDidChange: consumes this).
    _pendingStartOptions = [options copy];
    // Triggers -[STProxyProvider stopProxyWithReason:completionHandler:] on the extension side.
    [session stopTunnel];

    // Safety net: a provider that never reports Disconnected must not leave Split
    // Tunnel silently off.
    __weak typeof(self) weakSelf = self;
    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(5 * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
        __strong typeof(self) strongSelf = weakSelf;
        if (!strongSelf || !strongSelf->_pendingStartOptions) { return; }
        NSDictionary *pending = strongSelf->_pendingStartOptions;
        strongSelf->_pendingStartOptions = nil;
        [strongSelf startSessionWithOptions:pending];
    });
}

// This is the call that actually makes the OS launch (or reuse) the extension
// process and invoke -[STProxyProvider startProxyWithOptions:completionHandler:]
// on it, with `options` passed through unchanged as its `options` parameter.
- (void)startSessionWithOptions:(NSDictionary *)options {
    NETunnelProviderSession *session = (NETunnelProviderSession *)_lastManager.connection;
    NSError *startError = nil;
    [session startTunnelWithOptions:options andReturnError:&startError];
    if (startError) { NSLog(@"[split-tunnel-macos] session start failed: %@", startError); }
    _lastError = startError.localizedDescription; // nil on success, so a stale error never sticks
    [self notifyStateChanged];
}

- (void)notifyStateChanged {
    NSDictionary *payload = @{
        @"extensionState": [self extensionStateString],
        @"sessionStatus": [self sessionStatusString],
        @"lastError": [self lastErrorMessage],
    };
    NSData *data = [NSJSONSerialization dataWithJSONObject:payload options:0 error:nil];
    NSString *json = data ? [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding] : @"{}";
    runJSStateChangedCallback(json.UTF8String);
}

@end

//=========================================================================
// NAPI CODE: binding functions to JS
//=========================================================================

static napi_value CreateJSString(napi_env env, NSString *s) {
    napi_value v;
    napi_create_string_utf8(env, s.UTF8String ?: "", NAPI_AUTO_LENGTH, &v);
    return v;
}

// Copies argument 0 (expected to be a JS string) into a freshly allocated
// NSString. Caller owns nothing extra to free - ARC manages the NSString.
static NSString *CopyJSStringArg0(napi_env env, napi_callback_info info) {
    size_t argc = 1;
    napi_value args[1];
    napi_get_cb_info(env, info, &argc, args, NULL, NULL);
    if (argc < 1) { return @""; }

    size_t len = 0;
    napi_get_value_string_utf8(env, args[0], NULL, 0, &len);
    char *buf = (char *)malloc(len + 1);
    if (!buf) { return @""; }
    napi_get_value_string_utf8(env, args[0], buf, len + 1, &len);
    NSString *result = [NSString stringWithUTF8String:buf];
    free(buf);
    return result ?: @"";
}

static napi_value ExtensionActivate(napi_env env, napi_callback_info info) {
    [[STSplitTunnelController sharedInstance] activateExtension];
    return NULL;
}

static napi_value ExtensionDeactivate(napi_env env, napi_callback_info info) {
    [[STSplitTunnelController sharedInstance] deactivateExtension];
    return NULL;
}

static napi_value ExtensionGetState(napi_env env, napi_callback_info info) {
    return CreateJSString(env, [[STSplitTunnelController sharedInstance] extensionStateString]);
}

static napi_value ExtensionRefreshState(napi_env env, napi_callback_info info) {
    [[STSplitTunnelController sharedInstance] refreshExtensionState];
    return NULL;
}

static napi_value SessionGetStatus(napi_env env, napi_callback_info info) {
    return CreateJSString(env, [[STSplitTunnelController sharedInstance] sessionStatusString]);
}

static napi_value SessionRegisterConfig(napi_env env, napi_callback_info info) {
    [[STSplitTunnelController sharedInstance] registerConfiguration];
    return NULL;
}

static napi_value SessionApplyConfig(napi_env env, napi_callback_info info) {
    NSString *json = CopyJSStringArg0(env, info);
    [[STSplitTunnelController sharedInstance] applyConfigJSON:json];
    return NULL;
}

static napi_value SessionStop(napi_env env, napi_callback_info info) {
    [[STSplitTunnelController sharedInstance] stopSession];
    return NULL;
}

// State-changed callback: fired whenever the extension state and/or the
// tunnel session status changes. Registered once by the JS side at startup.
static napi_threadsafe_function stateChangedCallback = NULL;

static void ThreadsafeCallStateChanged(napi_env env, napi_value js_callback, void *context, void *data) {
    char *jsonUTF8 = (char *)data;
    if (env != NULL && js_callback != NULL) {
        napi_value global, jsString, result;
        napi_get_global(env, &global);
        napi_create_string_utf8(env, jsonUTF8, NAPI_AUTO_LENGTH, &jsString);
        napi_call_function(env, global, js_callback, 1, &jsString, &result);
    }
    free(jsonUTF8);
}

static napi_value SetStateChangedCallback(napi_env env, napi_callback_info info) {
    size_t argc = 1;
    napi_value args[1];
    napi_status status = napi_get_cb_info(env, info, &argc, args, NULL, NULL);
    if (status != napi_ok || argc < 1) {
        napi_throw_error(env, NULL, "SetStateChangedCallback requires a callback argument");
        return NULL;
    }

    if (stateChangedCallback != NULL) {
        napi_release_threadsafe_function(stateChangedCallback, napi_tsfn_release);
        stateChangedCallback = NULL;
    }

    napi_value resourceName;
    napi_create_string_utf8(env, "st_state_changed", NAPI_AUTO_LENGTH, &resourceName);
    napi_create_threadsafe_function(env, args[0], NULL, resourceName, 0, 1, NULL, NULL, NULL,
                                     ThreadsafeCallStateChanged, &stateChangedCallback);
    return NULL;
}

void runJSStateChangedCallback(const char *jsonUTF8) {
    if (stateChangedCallback == NULL) { return; }
    char *copy = strdup(jsonUTF8);
    if (!copy) { return; }
    napi_status status = napi_call_threadsafe_function(stateChangedCallback, copy, napi_tsfn_nonblocking);
    if (status != napi_ok) { free(copy); }
}

//=========================================================================
// INITIALIZATION
//=========================================================================

#define DECLARE_NAPI_METHOD(name, func) \
  { name, 0, func, 0, 0, 0, napi_default, 0 }

napi_value Init(napi_env env, napi_value exports) {
    napi_property_descriptor properties[] = {
        DECLARE_NAPI_METHOD("ExtensionActivate", ExtensionActivate),
        DECLARE_NAPI_METHOD("ExtensionDeactivate", ExtensionDeactivate),
        DECLARE_NAPI_METHOD("ExtensionGetState", ExtensionGetState),
        DECLARE_NAPI_METHOD("ExtensionRefreshState", ExtensionRefreshState),
        DECLARE_NAPI_METHOD("SessionGetStatus", SessionGetStatus),
        DECLARE_NAPI_METHOD("SessionRegisterConfig", SessionRegisterConfig),
        DECLARE_NAPI_METHOD("SessionApplyConfig", SessionApplyConfig),
        DECLARE_NAPI_METHOD("SessionStop", SessionStop),
        DECLARE_NAPI_METHOD("SetStateChangedCallback", SetStateChangedCallback),
    };

    napi_define_properties(env, exports, sizeof(properties) / sizeof(properties[0]), properties);
    return exports;
}

NAPI_MODULE(split_tunnel_macos, Init)
