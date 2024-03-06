// https://github.com/carlospolop/hacktricks/blob/master/macos-hardening/macos-security-and-privilege-escalation/macos-proces-abuse/macos-ipc-inter-process-communication/macos-xpc/README.md

#include <Foundation/Foundation.h>

#include "xpc.h"
#include "xpc_server.h"

// Hash table to store connections
static NSHashTable*                 _connections        = NULL;
static NSLock*                      _connectionsLock    = NULL;
static OnWifiInfo_func_ptr          _onWifiInfo         = NULL;
static OnWifiScanResult_func_ptr    _onWifiScanResult   = NULL;
static OnWifiChanged_func_ptr       _onWifiChanged      = NULL;
static OnXpcConnection_func_ptr     _onXpcConnection    = NULL;

static int sendMessage(xpc_object_t message) {
    int resultError = -1;
    [_connectionsLock lock];    
    for (id conn in _connections) {
        xpc_object_t connection = (__bridge xpc_object_t)conn;
        //NSLog(@"Sending message to connection: %p\n", connection);
        xpc_connection_send_message(connection, message);
        resultError = 0;
    }
    [_connectionsLock unlock];
    return resultError;
}

static void handle_event(xpc_object_t event) {
    xpc_type_t type = xpc_get_type(event);
    if (type != XPC_TYPE_DICTIONARY) {
        NSLog(@"[XPC] Invalid event type: %lu", (unsigned long)type);
        return; 
    }
    
    int64_t msgType = xpc_dictionary_get_int64(event, MSG_FIELD_TYPE);
    if (msgType == MSG_TYPE_WIFI_CHANGED) {
        if (_onWifiChanged == NULL) {
            NSLog(@"[XPC] _onWifiChanged is NULL");
            return;
        }
        _onWifiChanged();
    }
    else if (msgType == MSG_TYPE_WIFI_INFO) {
        if (_onWifiInfo == NULL) {
            NSLog(@"[XPC] _onWifiInfo is NULL");
            return;
        }
        int64_t security = xpc_dictionary_get_int64(event, MSG_FIELD_SECURITY);
        const char* ssid = xpc_dictionary_get_string(event, MSG_FIELD_SSID);
        _onWifiInfo((ssid==NULL)? "" : ssid, security);
    } else if (msgType == MSG_TYPE_WIFI_SCAN_RESULT) {
        if (_onWifiScanResult == NULL) {
            NSLog(@"[XPC] _onWifiScanResult is NULL");
            return;
        }
        const char* allSSID = xpc_dictionary_get_string(event, MSG_FIELD_SSID_LIST);
        _onWifiScanResult((allSSID==NULL)? "" : allSSID);
    } else 
        NSLog(@"[XPC] Unknown message type: %lld\n", msgType);    
}

// Remember the connection (to use it later to send messages to the client)
// This code is supposed to run as macOS LaunchAgent, so multiple clients can connect to it (one per each active user session)
// So, we need to store all connections to be able to communicate with all clients (LaunchAgents)
static void addConnection(xpc_connection_t connection) {
    unsigned long count = 0;

    [_connectionsLock lock];
    [_connections addObject:(__bridge id)connection];
    count = (unsigned long)_connections.count;
    [_connectionsLock unlock];

    NSLog(@"[XPC] New connection added. Connections count: %lu", count);
    if (_onXpcConnection != NULL)
        _onXpcConnection(count, true);
}

static void removeConnection(xpc_connection_t connection) {
    unsigned long count = 0;
    [_connectionsLock lock];
    [_connections removeObject:(__bridge id)connection];
    count = (unsigned long)_connections.count;    
    [_connectionsLock unlock];

    NSLog(@"[XPC] Connection removed. Connections count: %lu", (unsigned long)_connections.count);
    if (_onXpcConnection != NULL)
        _onXpcConnection(count, false);
}

static void handle_connection(xpc_connection_t connection) {    
    addConnection(connection);

    xpc_connection_set_event_handler(connection, ^(xpc_object_t event) {
        if (xpc_get_type(event) == XPC_TYPE_ERROR) {
            if (event == XPC_ERROR_CONNECTION_INTERRUPTED || event == XPC_ERROR_CONNECTION_INVALID) {
                removeConnection(connection);
            } else
                NSLog(@"[XPC] Unknown error");
        }
        else
            handle_event(event);
    });

    xpc_connection_resume(connection);
}

int ivpn_xpc_server_get_connections_count() {
    [_connectionsLock lock];
    unsigned long count = (unsigned long)_connections.count;
    [_connectionsLock unlock];
    return count;
}

int ivpn_xpc_server_request_wifi_info() {
    xpc_object_t message = xpc_dictionary_create(NULL, NULL, 0);
    xpc_dictionary_set_int64(message, MSG_FIELD_TYPE, MSG_TYPE_REQUEST_WIFI_INFO);
    int ret = sendMessage(message);
    xpc_release(message);
    return ret;
}

int ivpn_xpc_server_request_wifi_scan() {
    xpc_object_t message = xpc_dictionary_create(NULL, NULL, 0);
    xpc_dictionary_set_int64(message, MSG_FIELD_TYPE, MSG_TYPE_REQUEST_WIFI_SCAN);
    int ret = sendMessage(message);
    xpc_release(message);
    return ret;
}

int ivpn_xpc_server_init(
        OnWifiChanged_func_ptr  onWifiChanged,  
        OnWifiInfo_func_ptr onWifiInfo, 
        OnWifiScanResult_func_ptr onWifiScanResult, 
        OnXpcConnection_func_ptr onNewXpcConnection) 
    {
    if (onWifiChanged == NULL || onWifiInfo == NULL || onWifiScanResult == NULL || onNewXpcConnection == NULL) {
        NSLog(@"[XPC] Invalid arguments");
        return -1;
    }
    if (_onWifiChanged != NULL || _onWifiInfo != NULL || _onWifiScanResult != NULL || _onXpcConnection != NULL) {
        NSLog(@"[XPC] Already initialized");
        return -1;
    }

    _onWifiChanged      = onWifiChanged;
    _onWifiInfo         = onWifiInfo;
    _onWifiScanResult   = onWifiScanResult;
    _onXpcConnection    = onNewXpcConnection;

    _connections        = [NSHashTable weakObjectsHashTable];
    _connectionsLock    = [[NSLock alloc] init];

    xpc_connection_t service = xpc_connection_create_mach_service(SERVICE_NAME,
                                                                   NULL,
                                                                   XPC_CONNECTION_MACH_SERVICE_LISTENER);
    if (!service) {
        NSLog(@"[XPC] Failed to create service.");
        return -1;
    }

    xpc_connection_set_event_handler(service, ^(xpc_object_t event) { 
        if (xpc_get_type(event) == XPC_TYPE_CONNECTION) {
            handle_connection(event);
        }
    });

    xpc_connection_resume(service);
    return 0;
}
