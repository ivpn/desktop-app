
#include <stdlib.h>
#import <Foundation/Foundation.h>
#import <CoreWLAN/CoreWLAN.h>
#import <SystemConfiguration/SystemConfiguration.h>

static inline  char* nsstring2cstring(NSString *s){
    if (s == NULL) { return NULL; }

	char *cstr = strdup([s UTF8String]);
    return cstr;
}

#define NOT_CONNECTED @""

CWInterface * getCWInterface() {
	CWWiFiClient *swc = [CWWiFiClient sharedWiFiClient];
	if (swc == nil) return nil;
	return [swc interface];
}

void wifi_network_changed(SCDynamicStoreRef store, CFArrayRef changedKeys, void *ctx)
{
	extern void __onWifiChangedCallbackC();
	__onWifiChangedCallbackC();
}

char * getCurrentSSID(void) {
	CWInterface * WiFiInterface = getCWInterface();
	if (WiFiInterface == nil) return nsstring2cstring(NOT_CONNECTED);

	NSString *ssid = [WiFiInterface ssid] ? [WiFiInterface ssid] : NOT_CONNECTED;
	return nsstring2cstring(ssid);
}

int getCurrentNetworkSecurity() {
	CWInterface * WiFiInterface = getCWInterface();
	if (WiFiInterface == nil) return 0xFFFFFFFF;

	return [WiFiInterface security];
}

char* getAvailableSSIDs(void) {
	CWInterface * WiFiInterface = getCWInterface();
	if (WiFiInterface == nil) return nil;

	NSError *err = nil;
	NSSet *scanset = [WiFiInterface scanForNetworksWithSSID:Nil error:&err];
    if (err!=nil || scanset == nil || scanset.count == 0) return nil;

	NSString *retString = nil;
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

	return nsstring2cstring(retString);
}

void setWifiNotifier(void) {
    CWInterface * WiFiInterface = getCWInterface();
	if (WiFiInterface == nil) return;

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
}