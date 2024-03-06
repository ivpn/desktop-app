#include <Foundation/Foundation.h>

#include <stdio.h>
#include <string.h>
#include <signal.h>

#include "../xpc_client.h"

void sig_handler(int signo)
{
    NSLog(@"Received quit signal (%d). Exiting...\n", signo);
    exit(0);
}

NSString* testGetWifiList() {
    printf("testGetWifiList: New request from server\n");
    return @"testGetWifiList: XPC client test\n";
}

int testGetWifiInfo(char ssid[64], int* security) {
    if (ssid == NULL || security == NULL) 
        return -1;

    printf("testGetWifiInfo: New request from server\n");

    *security = 1;
    strncpy(ssid, "testGetWifiInfo: XPC client test", 64);
    
    return 0;
}

int main(int argc, const char *argv[]) {
    printf("START\n");

    signal(SIGINT, sig_handler);
    signal(SIGTERM, sig_handler);
    signal(SIGQUIT, sig_handler); 

    printf("ivpn_xpc_client_init:%d\n", ivpn_xpc_client_init(testGetWifiInfo, testGetWifiList));

    // Periodically send a message to the server (e.g. wifi chnaged)
    for (;;) {
        sleep(5);
        ivpn_xpc_client_send_wifi_changed();
    }

    printf("FINISH\n");
    return 0;
}
