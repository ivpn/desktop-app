#import <Foundation/Foundation.h>
#import <CoreWLAN/CoreWLAN.h>
#import <SystemConfiguration/SystemConfiguration.h>

#define EMPTY_STRING @""
#define UNKNOWN_WIFI_SECURITY 0xFFFFFFFF

#include "wifi.h"

OnWifiChangedFunc onWifiChanged = NULL;

static inline CWInterface * getCWInterface() {
    CWWiFiClient *swc = [CWWiFiClient sharedWiFiClient];

    if (swc == nil || [swc interface] ==nil ) NSLog(@"WIFI interface is NIL");

    if (swc == nil) return nil;
    return [swc interface];
}

int GetCurrentNetworkSecurity() {
    CWInterface * WiFiInterface = getCWInterface();
    if (WiFiInterface == nil) return UNKNOWN_WIFI_SECURITY;
    return [WiFiInterface security];
}

NSString* GetCurrentSSID(void) {
    CWInterface * WiFiInterface = getCWInterface();
    return (WiFiInterface != nil && [WiFiInterface ssid]) ? [WiFiInterface ssid] : EMPTY_STRING;
}

NSString*  GetAvailableSSIDs(void) {
    CWInterface * WiFiInterface = getCWInterface();
    if (WiFiInterface == nil) return EMPTY_STRING;

    NSError *err = nil;
    NSSet *scanset = [WiFiInterface scanForNetworksWithSSID:Nil error:&err];
    if (err!=nil || scanset == nil || scanset.count == 0) return EMPTY_STRING;

    NSString *retString = EMPTY_STRING;
    int i=0;
    for (CWNetwork * nw in scanset)
    {
        if (nw == nil || [nw ssid] == nil) continue;
        NSString * ssid = [[[nw ssid] componentsSeparatedByCharactersInSet:[NSCharacterSet newlineCharacterSet]] componentsJoinedByString:@" "];
        if (i++ == 0)
            retString = ssid;
        else
            retString = [NSString stringWithFormat:@"%@\n%@", retString , ssid];
    }

    return retString;
}

static inline void wifi_network_changed(SCDynamicStoreRef store, CFArrayRef changedKeys, void *ctx)
{
	if (onWifiChanged != NULL)
	    onWifiChanged();
}

// Detection WiFi status change in infinite loop. Normally it should never return.
int RunWifiNotifier(OnWifiChangedFunc handler) {
    if (handler == NULL) return -1;
    onWifiChanged = handler;

    CWInterface * WiFiInterface = getCWInterface();
	if (WiFiInterface == nil) return -2;

	NSArray* arr = [CWWiFiClient interfaceNames];
	NSSet *wifiInterfaces = [NSSet setWithArray:arr];

    NSMutableArray *scKeys = [[NSMutableArray alloc] init];
    [wifiInterfaces enumerateObjectsUsingBlock:^(NSString *ifName, BOOL *stop)
     {
         [scKeys addObject: [NSString stringWithFormat:@"State:/Network/Interface/%@/AirPort", ifName]];
     }];

    SCDynamicStoreContext ctx = { 0, NULL, NULL, NULL, NULL };
    SCDynamicStoreRef store = SCDynamicStoreCreate(kCFAllocatorDefault, CFSTR("IVPN"), wifi_network_changed, &ctx);

    SCDynamicStoreSetNotificationKeys(store, (__bridge CFArrayRef)scKeys, NULL);

    CFRunLoopSourceRef src = SCDynamicStoreCreateRunLoopSource(kCFAllocatorDefault, store, 0);
	CFRunLoopAddSource([[NSRunLoop currentRunLoop] getCFRunLoop], src, kCFRunLoopCommonModes);

	CFRunLoopRun();
    return 0;
}