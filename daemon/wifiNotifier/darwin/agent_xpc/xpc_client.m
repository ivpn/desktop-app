// gcc xpc_client.c -o xpc_client

#ifndef _IVPN_XPC_CLIENT_H_
#define _IVPN_XPC_CLIENT_H_

#include <Foundation/Foundation.h>
#include <pthread.h>
#include <xpc/xpc.h>
#include <stdatomic.h>

#include "xpc.h"
#include "xpc_client.h"

static xpc_connection_t     _connection             = NULL;
static atomic_bool          _should_reinitialize    = false;
static GetWifiInfo_func_ptr _getWifiInfo            = NULL;
static GetWifiList_func_ptr _getWifiList            = NULL;

static xpc_connection_t getConnection() {
    return _connection;
}

// Function to send a message
static void send_message(xpc_object_t message) {
    if (message == NULL) {
        NSLog(@"ERROR: Unable to send message: Message is NULL");
        return;
    }

    xpc_connection_t connection = getConnection();
    if (connection == NULL) {
        NSLog(@"ERROR: Unable to send message: Connection is NULL");
        return;
    }
 
    xpc_connection_send_message(connection, message);
}

static void send_wifi_changed() {
    xpc_object_t message = xpc_dictionary_create(NULL, NULL, 0);
    xpc_dictionary_set_int64(message, MSG_FIELD_TYPE, MSG_TYPE_WIFI_CHANGED);
    send_message(message);
    xpc_release(message);
}

static void send_wifi_info(const char* SSID, const int security) {
    xpc_object_t message = xpc_dictionary_create(NULL, NULL, 0);
    xpc_dictionary_set_int64(message, MSG_FIELD_TYPE, MSG_TYPE_WIFI_INFO);
    xpc_dictionary_set_int64(message, MSG_FIELD_SECURITY, security);
    xpc_dictionary_set_string(message, MSG_FIELD_SSID, (SSID==NULL)? "" : SSID );    
    send_message(message);
    xpc_release(message);
}

static void send_wifi_scan_result(const char* allSSID) {
    xpc_object_t message = xpc_dictionary_create(NULL, NULL, 0);
    xpc_dictionary_set_int64(message, MSG_FIELD_TYPE, MSG_TYPE_WIFI_SCAN_RESULT);    
    xpc_dictionary_set_string(message, MSG_FIELD_SSID_LIST, (allSSID==NULL)? "" : allSSID );    
    send_message(message);
    xpc_release(message);
}

static void handle_event(xpc_connection_t connection, xpc_object_t event) {
    if (xpc_get_type(event) == XPC_TYPE_DICTIONARY) {        
        int64_t msgType = xpc_dictionary_get_int64(event, MSG_FIELD_TYPE);
        if (msgType == MSG_TYPE_REQUEST_WIFI_INFO) {
            char ssid[64] = {0};
            int security = 0;
            if (_getWifiInfo(ssid, &security) == 0)                 
                send_wifi_info(ssid, security);
            else
                NSLog(@"Failed to get wifi info");
        }
        else if (msgType == MSG_TYPE_REQUEST_WIFI_SCAN) {
            NSString* list = _getWifiList();
            send_wifi_scan_result((list==NULL)? "" : [list UTF8String]);
        }
        else
            NSLog(@"Unknown message type: %lld", msgType);
    }
    else  if (xpc_get_type(event) == XPC_TYPE_ERROR) {                        
        if (event == XPC_ERROR_CONNECTION_INTERRUPTED || event == XPC_ERROR_CONNECTION_INVALID) {
            NSLog(@"Connection was invalidated");
            _should_reinitialize = true;
        } else
            NSLog(@"Unknown error");
    }
}

static bool connection_init() {   
    _connection = xpc_connection_create_mach_service(SERVICE_NAME, NULL, XPC_CONNECTION_MACH_SERVICE_PRIVILEGED);

    xpc_connection_set_event_handler(_connection, ^(xpc_object_t event) {
        handle_event(_connection, event);
    });

    xpc_connection_resume(_connection);

    // send init message
    send_wifi_changed();

    if (_connection!=NULL)
        NSLog(@"Connection initialized");

    return _connection!=NULL;
}

static void* init(void* param) {
    connection_init();
    // main loop: reinitialize connection if needed
    for (;;) {
        if (_should_reinitialize) {
            _should_reinitialize = false;
            NSLog(@"Reinitializing connection...");
            connection_init();
        }        
        sleep(3);
    }
}

void ivpn_xpc_client_send_wifi_changed() {
    send_wifi_changed();
}

int ivpn_xpc_client_init(GetWifiInfo_func_ptr getFunc, GetWifiList_func_ptr getFuncList) {
    if (getFunc == NULL || getFuncList == NULL) {
        NSLog(@"ERROR: Invalid arguments");
        return -1;        
    }
    if (_getWifiInfo != NULL || _getWifiList != NULL) {
        NSLog(@"ERROR: Already initialized");
        return -1;
    }
    
    _getWifiInfo = getFunc;
    _getWifiList = getFuncList;

    pthread_t thread_id;
    if (pthread_create(&thread_id, NULL, init, NULL) != 0) {
        NSLog(@"ERROR: Failed to create thread");
        return -2;
    }

    return 0;
}
#endif