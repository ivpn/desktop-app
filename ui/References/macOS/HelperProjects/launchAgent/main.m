#import <Foundation/Foundation.h>
#import <CoreWLAN/CoreWLAN.h>

#include <stdio.h>
#include <stdlib.h>
#include <signal.h>

#include "wifi.h"
#include "xpc_client.h"

// global variables, just for testing and logging
int randomId;

void sig_handler(int signo)
{
    NSLog(@"[%d] Received quit signal (%d). Exiting...\n", randomId, signo);
    exit(0);
}

// This function is called by wifiNotifier to notify about WiFi change
void onWifiChanged_handler() {
    ivpn_xpc_client_send_wifi_changed();
}

// This function is called by XPC client to get WiFi info
int onGetWifiInfo_handler(char ssid[64], int* security) {
    if (ssid == NULL || security == NULL) 
        return -1;

    int w_security = GetCurrentNetworkSecurity();
    NSString* w_ssid = GetCurrentSSID();
    if (ssid == nil)      
        return -2;

    *security = w_security;
    strncpy(ssid, [w_ssid UTF8String] , 64);
    
    return 0;
}

// This function is called by XPC client to get available WiFi list
NSString* onGetWifiList_handler() {
    NSString* ssids = GetAvailableSSIDs();
    return ssids;
} 

int main(int argc, char *argv[]) {
    randomId = arc4random_uniform(100);

    NSLog(@"[%d] LaunchAgent started", randomId);

    signal(SIGINT, sig_handler);
    signal(SIGTERM, sig_handler);
    signal(SIGQUIT, sig_handler); 

    // Init XPC client to communicate with the Daemon
    int xpcError = ivpn_xpc_client_init(onGetWifiInfo_handler, onGetWifiList_handler);
    if (xpcError != 0) {
        NSLog(@"[%d] Error initializing XPC client!", randomId);
    }

     while (1) {
        // Detection WiFi status change. 
        // Normally 'RunWifiNotifier()' should never return.
        // BUT! It can return in some corner cases (e.g. we call it on system boot when WiFi interface still not initialized)
		// In this case - we waiting some delay and trying to call this function again
        NSLog(@"Starting WiFi change notifier...");
        RunWifiNotifier(onWifiChanged_handler);
        NSLog(@"Unexpected return from RunWifiNotifier. Waiting 1 sec and trying again...");

        sleep(1);
     }

    NSLog(@"[%d] LaunchAgent stopped", randomId);
    
    return 0;
}
