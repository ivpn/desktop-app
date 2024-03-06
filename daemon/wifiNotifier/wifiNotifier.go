package wifiNotifier

import (
	"fmt"

	"github.com/ivpn/desktop-app/daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("wifi")
}

type WifiInfo struct {
	SSID       string
	IsInsecure bool
}

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func GetAvailableSSIDs() ([]string, error) {
	ret, err := implGetAvailableSSIDs()
	if err != nil {
		err = log.ErrorE(fmt.Errorf("can not obtain available SSIDs: %w", err), 0)
	}
	return ret, err
}

// GetCurrentWifiInfo returns current WiFi info
func GetCurrentWifiInfo() (WifiInfo, error) {
	w, err := implGetCurrentWifiInfo()
	if err != nil {
		err = log.ErrorE(fmt.Errorf("can not obtain current WiFi info: %w", err), 0)
	}
	return w, err
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func SetWifiNotifier(cb func()) error {
	return implSetWifiNotifier(cb)
}
