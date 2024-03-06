#include <Foundation/Foundation.h>

typedef int (*GetWifiInfo_func_ptr)(char[64], int*);    // SSID, Security
typedef NSString* (*GetWifiList_func_ptr)();            // SSID_List, separated by '\n'. 

int ivpn_xpc_client_init(GetWifiInfo_func_ptr getFuncm, GetWifiList_func_ptr getListFunc);
void ivpn_xpc_client_send_wifi_changed();