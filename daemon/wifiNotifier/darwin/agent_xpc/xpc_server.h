
typedef void (*OnWifiChanged_func_ptr)();
typedef void (*OnWifiInfo_func_ptr)(const char* ssid, const int security);
typedef void (*OnWifiScanResult_func_ptr)(const char* ssdList);             // SSID List, separated by '\n'
typedef void (*OnXpcConnection_func_ptr)(int connectionsCount, bool isAdded); 

int ivpn_xpc_server_init(
    OnWifiChanged_func_ptr      onWifiChanged, 
    OnWifiInfo_func_ptr         onWifiInfo, 
    OnWifiScanResult_func_ptr   onWifiScanResult,
    OnXpcConnection_func_ptr    onNewXpcConnection);

int ivpn_xpc_server_request_wifi_info();
int ivpn_xpc_server_request_wifi_scan();
int ivpn_xpc_server_get_connections_count();