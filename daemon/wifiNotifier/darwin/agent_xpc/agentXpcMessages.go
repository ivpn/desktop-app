//go:build darwin && !nowifi
// +build darwin,!nowifi

package agentxpc

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation

#include <stdlib.h>
#import <Foundation/Foundation.h>

#include "xpc_server.h"

extern void onXpcConnection(int connectionsCount, bool isAdded);
extern void onWifiChanged();
extern void onWifiInfo(char* ssid, int security);
extern void onWifiScanResult(char* ssidList);

static void xpc_OnXpcConnection(int connectionsCount, bool isAdded) {
	onXpcConnection(connectionsCount, isAdded);
}

static void xpc_OnWifiChanged() {
	onWifiChanged();
}

static void xpc_OnWifiInfo(const char* ssid, const int security) {
	onWifiInfo((char*)ssid, (int)security);
}

static void xpc_OnWifiScanResult(const char* list)	{
	onWifiScanResult((char*)list);
}

static int XPC_init() {
	return ivpn_xpc_server_init(xpc_OnWifiChanged, xpc_OnWifiInfo, xpc_OnWifiScanResult, xpc_OnXpcConnection);
}
*/
import "C"

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/wifiNotifier/darwin"
	"golang.org/x/sync/singleflight"
)

var (
	log                darwin.Logger
	instance           *WifiSource
	once               sync.Once
	sfGroup            singleflight.Group
	wifiInfoChan       chan wifiInfo
	wifiScanResultChan chan []string
)

func init() {
	wifiInfoChan = make(chan wifiInfo, 1)
	wifiScanResultChan = make(chan []string, 1)
}

type wifiInfo struct {
	SSID     string
	Security int
}

type WifiSource struct {
	onWifiChangedCallback func()
	lastWifiInfo          wifiInfo
	lastWifiInfoLocker    sync.Mutex
}

func GetWifiSourceInstance() *WifiSource {
	once.Do(func() {
		instance = &WifiSource{}
	})
	return instance
}

//export onXpcConnection
func onXpcConnection(connectionsCount C.int, isAdded C.bool) {
	if isAdded {
		log.Info("New LaunchAgent connection. Total connections: ", connectionsCount)
	} else {
		log.Info("LaunchAgent connection closed. Total connections: ", connectionsCount)
	}
	if connectionsCount == 0 {
		onWifiChanged() // notify: no wifi info available
	}
}

//export onWifiChanged
func onWifiChanged() {
	GetWifiSourceInstance().notifyChange()
}

//export onWifiInfo
func onWifiInfo(ssid *C.char, security C.int) {
	theSSID := ""
	if ssid != nil {
		theSSID = C.GoString(ssid)
	}

	select {
	case wifiInfoChan <- wifiInfo{SSID: theSSID, Security: int(security)}:
	default:
	}
}

//export onWifiScanResult
func onWifiScanResult(ssidList *C.char) {
	var ssids []string
	if ssidList != nil {
		rawList := C.GoString(ssidList)
		ssids = strings.Split(rawList, "\n")
	}

	select {
	case wifiScanResultChan <- ssids:
	default:
	}
}

func requestAndWaitWifiInfo() (wifiInfo, error) {
	retChan := sfGroup.DoChan("wifi-info", func() (retVal interface{}, retErr error) {
		// drain the channel
		for len(wifiInfoChan) > 0 {
			<-wifiInfoChan
		}

		ret := C.ivpn_xpc_server_request_wifi_info()
		if ret != 0 {
			if C.ivpn_xpc_server_get_connections_count() == 0 {
				return wifiInfo{}, fmt.Errorf("no connection to LaunchAgent")
			}
			return wifiInfo{}, fmt.Errorf("failed to request data from LaunchAgent: %d", ret)
		}

		select {
		case retVal = <-wifiInfoChan:
			return retVal, nil
		case <-time.After(10 * time.Second):
			return wifiInfo{}, fmt.Errorf("timeout waiting for wifi info from LaunchAgent")
		}
	})

	result := <-retChan
	return result.Val.(wifiInfo), result.Err
}

func requestAndWaitWifiScanResult() ([]string, error) {
	retChan := sfGroup.DoChan("wifi-scan-result", func() (retVal interface{}, retErr error) {
		// drain the channel
		for len(wifiScanResultChan) > 0 {
			<-wifiScanResultChan
		}

		ret := C.ivpn_xpc_server_request_wifi_scan()
		if ret != 0 {
			if C.ivpn_xpc_server_get_connections_count() == 0 {
				return wifiInfo{}, fmt.Errorf("no connection to LaunchAgent")
			}
			return []string{}, fmt.Errorf("failed to request data from LaunchAgent: %d", ret)
		}

		select {
		case retVal = <-wifiScanResultChan:
			return retVal, nil
		case <-time.After(20 * time.Second):
			return []string{}, fmt.Errorf("timeout waiting for wifi scan result from LaunchAgent")
		}
	})

	result := <-retChan
	return result.Val.([]string), result.Err
}

func (o *WifiSource) Init(l darwin.Logger) error {
	log = l
	if ret := C.XPC_init(); ret != 0 {
		return fmt.Errorf("failed to initialise XPC server (%d)", ret)
	}
	log.Info("XPC server initialised")

	return nil
}

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func (o *WifiSource) GetAvailableSSIDs() ([]string, error) {
	return requestAndWaitWifiScanResult()
}

// GetCurrentWifiInfo returns current WiFi info
func (o *WifiSource) GetCurrentWifiInfo() (ssid string, security int, err error) {
	ret, err := requestAndWaitWifiInfo()

	// If the info was requested and it's different from the last one reported, trigger the "wifi changed" event.
	//
	// Example: This is useful in situations when the LocationServices permission was granted (changed from OFF to ON).
	// The "wifi changed" event is not triggered because the daemon is unable to detect changes in LocationServices permissions, but the UI can.
	// In this case, the UI requests the WiFi info itself and the daemon should trigger the "wifi changed" event.
	// As a result, the daemon will process WiFi change actions (like auto-connect) if necessary.
	o.lastWifiInfoLocker.Lock()
	defer o.lastWifiInfoLocker.Unlock()
	if ret.SSID != o.lastWifiInfo.SSID {
		o.lastWifiInfo = ret
		go o.notifyChange() // call in a goroutine to avoid deadlocks
	}

	return ret.SSID, ret.Security, err
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func (o *WifiSource) SetWifiNotifier(cb func()) error {
	if o.onWifiChangedCallback != nil {
		return fmt.Errorf("Wifi notifier already set")
	}
	o.onWifiChangedCallback = cb
	return nil
}

func (o *WifiSource) notifyChange() error {
	cb := o.onWifiChangedCallback
	if cb != nil {
		cb()
	}
	return nil
}
