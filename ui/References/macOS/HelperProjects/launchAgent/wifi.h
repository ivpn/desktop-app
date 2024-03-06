typedef void (*OnWifiChangedFunc)();

int GetCurrentNetworkSecurity();
NSString * GetCurrentSSID(void);
NSString*  GetAvailableSSIDs(void);
// Detection WiFi status change in infinite loop. Normally it should never return.
int RunWifiNotifier(OnWifiChangedFunc handler);
