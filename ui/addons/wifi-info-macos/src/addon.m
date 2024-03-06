
// Node addon examples:
//    https://github.com/nodejs/node-addon-examples

#include <node_api.h>

void runJSCallbackLsAuthorizationChange();

//=========================================================================
// OBJECTIVE-C CODE
//=========================================================================

#import <CoreLocation/CoreLocation.h>

//=========================================================================
// Manage location info permissions
//=========================================================================

@interface LocationManager : NSObject <CLLocationManagerDelegate>
@property (nonatomic, strong) CLLocationManager *locationManager;
- (CLAuthorizationStatus)getAuthorizationStatus;
- (BOOL)areLocationServicesEnabled;
- (void)requestAlwaysAuthorization;
@end

@implementation LocationManager

+ (instancetype)sharedInstance {
    static LocationManager *sharedInstance = nil;
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        sharedInstance = [[self alloc] initPrivate];
    });
    return sharedInstance;
}

- (instancetype)initPrivate {
    self = [super init];
    if (self) {
        self.locationManager = [[CLLocationManager alloc] init];
        self.locationManager.delegate = self; 
    }
    return self;
}

- (instancetype)init {
    return [[self class] sharedInstance];
}

- (CLAuthorizationStatus)getAuthorizationStatus {
        return self.locationManager.authorizationStatus;
}

- (BOOL)areLocationServicesEnabled {
    return [CLLocationManager locationServicesEnabled];
}

- (void)requestAlwaysAuthorization {
  if (@available(macOS 10.15, *)) {
    [self.locationManager requestAlwaysAuthorization];
  } 
}

// This delegate method is called when the authorization status changes
- (void)locationManagerDidChangeAuthorization:(CLLocationManager *)manager {
  CLAuthorizationStatus status= self.locationManager.authorizationStatus;
  runJSCallbackLsAuthorizationChange();
  NSLog(@"IVPN: Location Services  Authorization status changed: %d", status);
}
@end

//=========================================================================
// NAPI CODE: binding functions to JS
//=========================================================================

static napi_value emptyJSString(napi_env env) {
  napi_value emptyString;
  napi_create_string_utf8(env, "", 0, &emptyString);
  return emptyString;
}

static napi_value LocationServicesAuthorizationStatus(napi_env env, napi_callback_info info) {
  napi_value retVal;  
  LocationManager *locationMgr = [LocationManager sharedInstance];
  napi_create_int32(env, [locationMgr getAuthorizationStatus], &retVal);
  return retVal;
}

static napi_value LocationServicesEnabled(napi_env env, napi_callback_info info) {
  napi_value retVal;  
  LocationManager *locationMgr = [LocationManager sharedInstance];
  napi_create_int32(env, [locationMgr areLocationServicesEnabled] ? 1 : 0, &retVal);
  return retVal;
}

static napi_value LocationServicesRequestPermission(napi_env env, napi_callback_info info) {
  LocationManager *locationMgr = [LocationManager sharedInstance];
  [locationMgr requestAlwaysAuthorization];  
  return NULL;
}

// callback
static napi_threadsafe_function lsAuthorisationChangeCallback  = NULL;
static napi_value LocationServicesSetAuthorizationChangeCallback(napi_env env, napi_callback_info info) {
  napi_status status;
  
  size_t argc = 1;
  napi_value args[1];
  status = napi_get_cb_info(env, info, &argc, args, NULL, NULL);
  if (status != napi_ok) {
    napi_throw_error(env, NULL, "Failed to parse arguments");
    return NULL;
  }

  napi_value resourceName;
  status = napi_create_string_utf8(env, "evt", NAPI_AUTO_LENGTH, &resourceName);
  if (status != napi_ok) {
    napi_throw_error(env, NULL, "Failed to create resource name");
    return NULL;
  }

  napi_value callback = args[0]; // Get the callback function
  status = napi_create_threadsafe_function(env, callback, NULL, resourceName, 0, 1, NULL, NULL, NULL, NULL, &lsAuthorisationChangeCallback );
  if (status != napi_ok) {  
    char errorMsg[128];
    snprintf(errorMsg, sizeof(errorMsg), "Failed to create threadsafe function. Status: %d", status);
    napi_throw_error(env, NULL, errorMsg);
    return NULL;
  }
  
  return NULL;
}

void runJSCallbackLsAuthorizationChange() {
  if (lsAuthorisationChangeCallback == NULL) return;  

  napi_status status = napi_call_threadsafe_function(lsAuthorisationChangeCallback, NULL, napi_tsfn_blocking);
  if (status != napi_ok) {
    // Handle error...
  }
}

//=========================================================================
// LaunchAgent
//=========================================================================
#import <ServiceManagement/SMAppService.h>
#define LAUNCH_AGENT_PLIST @"net.ivpn.LaunchAgent_launchd.plist"

void logAgentStatus(long status) {
  switch (status) {
    case SMAppServiceStatusNotRegistered:
      NSLog(@"IVPN LaunchAgent status: NotRegistered (%ld)", status);
      break;
    case SMAppServiceStatusEnabled:
      NSLog(@"IVPN LaunchAgent status: Enabled (%ld)", status);
      break;
    case SMAppServiceStatusRequiresApproval:
      NSLog(@"IVPN LaunchAgent status: RequiresApproval (%ld)", status);
      break;
    case SMAppServiceStatusNotFound:
      NSLog(@"IVPN LaunchAgent status: NotFound (%ld)", status);
      break;
    default:
      NSLog(@"IVPN LaunchAgent status: Unknown (%ld)", status);
      break;
  }
}

static napi_value AgentGetStatus(napi_env env, napi_callback_info info) {
  SMAppService *agentService = [SMAppService agentServiceWithPlistName:LAUNCH_AGENT_PLIST];
  napi_value retVal;
  napi_create_int32(env, [agentService status], &retVal);
  return retVal;
}

static napi_value AgentUninstall(napi_env env, napi_callback_info info) {
  napi_value retVal;

  SMAppService *agentService = [SMAppService agentServiceWithPlistName:LAUNCH_AGENT_PLIST];
  if ([agentService status] == SMAppServiceStatusNotRegistered || [agentService status] == SMAppServiceStatusNotFound) {
    napi_create_int32(env, 0, &retVal);
    return retVal; // already uinstalled
  }

  NSLog(@"Uninstalling '%@'...", LAUNCH_AGENT_PLIST);
  NSError* error = nil;
  bool isOk = [agentService unregisterAndReturnError:&error];
  if (error != nil) 
    NSLog(@"Uninstalling '%@' FAILED: %@", LAUNCH_AGENT_PLIST, error);
  else if (!isOk)
    NSLog(@"Uninstalling '%@' FAILED", LAUNCH_AGENT_PLIST);
  else
    NSLog(@"Uninstalled '%@'", LAUNCH_AGENT_PLIST);

  napi_create_int32(env, (error==nil && isOk)? 0: 1, &retVal);
  return retVal; // returns 0 if success
}

static napi_value AgentInstall(napi_env env, napi_callback_info info) {
  napi_value retVal;  

  SMAppService *agentService = [SMAppService agentServiceWithPlistName:LAUNCH_AGENT_PLIST];
  if ([agentService status] == SMAppServiceStatusEnabled) {
    napi_create_int32(env, 0, &retVal);
    NSLog(@"Installing '%@' SKIPPED: already installed", LAUNCH_AGENT_PLIST);
    return retVal; // already installed
  }

  logAgentStatus([agentService status]);
    
  NSLog(@"Installing '%@'...", LAUNCH_AGENT_PLIST);
  NSError* error = nil;
  bool isOk = [agentService registerAndReturnError:&error];
  if (error != nil) 
    NSLog(@"Installing '%@' FAILED: %@", LAUNCH_AGENT_PLIST, error);
  else if (!isOk)
    NSLog(@"Installing '%@' FAILED", LAUNCH_AGENT_PLIST);
  else
    NSLog(@"Installed '%@'", LAUNCH_AGENT_PLIST);

 
  napi_create_int32(env, (error==nil && isOk)? 0: 1, &retVal);
  return retVal; // returns 0 if success
}

//=========================================================================
// INITIALIZATION
//=========================================================================

#define DECLARE_NAPI_METHOD(name, func)                                        \
  { name, 0, func, 0, 0, 0, napi_default, 0 }

napi_value Init(napi_env env, napi_value exports) {

  napi_property_descriptor properties[] = {
    DECLARE_NAPI_METHOD( "LocationServicesAuthorizationStatus", LocationServicesAuthorizationStatus ),
    DECLARE_NAPI_METHOD( "LocationServicesEnabled", LocationServicesEnabled ), 
    DECLARE_NAPI_METHOD( "LocationServicesRequestPermission", LocationServicesRequestPermission ), 
    DECLARE_NAPI_METHOD( "LocationServicesSetAuthorizationChangeCallback", LocationServicesSetAuthorizationChangeCallback ),
    DECLARE_NAPI_METHOD( "AgentInstall", AgentInstall ),
    DECLARE_NAPI_METHOD( "AgentUninstall", AgentUninstall ),   
    DECLARE_NAPI_METHOD( "AgentGetStatus", AgentGetStatus )
  };

  // Define properties on the exports object
  napi_define_properties(
    env, 
    exports, 
    sizeof(properties) / sizeof(properties[0]), 
    properties);

  return exports;
}


NAPI_MODULE( wifi_info_macos, Init )