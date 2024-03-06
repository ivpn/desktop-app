//go:build darwin && !nowifi
// +build darwin,!nowifi

package obsolete

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework SystemConfiguration -framework CoreWLAN -framework Foundation

#include <stdlib.h>

char* getAvailableSSIDs(void);
char * getCurrentSSID(void);
int getCurrentNetworkSecurity();
void setWifiNotifier(void);
*/
import "C"
import (
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ivpn/desktop-app/daemon/wifiNotifier/darwin"
)

var (
	instance *WifiSource
	once     sync.Once
)

type WifiSource struct {
	log                   darwin.Logger
	onWifiChangedCallback func()
}

func GetWifiSourceInstance() *WifiSource {
	once.Do(func() {
		instance = &WifiSource{}
	})
	return instance
}

//export __onWifiChangedCallbackC
func __onWifiChangedCallbackC() {
	cb := GetWifiSourceInstance().onWifiChangedCallback
	if cb != nil {
		cb()
	}
}

func (o *WifiSource) Init(l darwin.Logger) error {
	o.log = l
	o.log.Info("Original implementation in use")
	return nil
}

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func (o *WifiSource) GetAvailableSSIDs() ([]string, error) {
	ssidList := C.getAvailableSSIDs()
	goSsidList := C.GoString(ssidList)
	C.free(unsafe.Pointer(ssidList))
	return strings.Split(goSsidList, "\n"), nil
}

// GetCurrentWifiInfo returns current WiFi info
func (o *WifiSource) GetCurrentWifiInfo() (ssid string, security int, err error) {
	SSID := getCurrentSSID()
	Security := int(C.getCurrentNetworkSecurity())
	return SSID, Security, nil
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func (o *WifiSource) SetWifiNotifier(cb func()) error {
	o.onWifiChangedCallback = cb

	go func() {
		//log.Info("WiFi notifier enter")
		//defer log.Error("WiFi notifier exit")

		for {
			// Detection WiFi status change in infinite loop.
			// C.setWifiNotifier() should never return.
			//
			// BUT! It can return in some corner cases (e.g. we call it on system boot when WiFi interface still not initialized)
			// In this case - we waiting some delay and trying to call this function again
			C.setWifiNotifier()
			//log.Info("Unexpected WiFi notifier exit")
			time.Sleep(time.Second)
			//log.Info("WiFi notifier enter. Retry...")
		}
	}()
	return nil
}

func getCurrentSSID() string {
	ssid := C.getCurrentSSID()
	goSsid := C.GoString(ssid)
	C.free(unsafe.Pointer(ssid))
	return goSsid
}
