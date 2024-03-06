#include <stdio.h>
#include <Foundation/Foundation.h>

#include "../xpc_server.h"

void OnXpcConnection(int connectionsCount, bool isAdded) {
    NSLog(@"OnXpcConnection: %d, %d\n", connectionsCount, isAdded);
}

void OnWifiChanged()
{
    NSLog(@"OnWifiChanged\n");
}

void OnWifiScanResult(const char* ssidList)
{
    NSLog(@"OnWifiScanResult: %s\n", ssidList);
}

void OnWifiInfo(const char* ssid , const int security)
{
    NSLog(@"OnWifiInfo: %s, %d\n", ssid, security);
} 

int main(int argc, const char *argv[]) {
    NSLog(@"Starting server...\n");

    if (ivpn_xpc_server_init(OnWifiChanged, OnWifiInfo, OnWifiScanResult, OnXpcConnection)!=0) {
        NSLog(@"Failed to xpc_init.\n");
        exit(EXIT_FAILURE);
    }

    for (;;) 
    {
        sleep(2);
        ivpn_xpc_server_request_wifi_info();
        sleep(2);
        ivpn_xpc_server_request_wifi_scan();
    }
    
    //dispatch_main();

    NSLog(@"DONE\n");

    return 0;
}