package wifiNotifier

import "github.com/ivpn/desktop-app/daemon/logger"

var log *logger.Logger

func init() {
	log = logger.NewLogger("wifi")
}

type WifiInfo struct {
	SSID       string
	IsInsecure bool
}

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func GetAvailableSSIDs() []string {
	return implGetAvailableSSIDs()
}

// GetCurrentWifiInfo returns current WiFi info
func GetCurrentWifiInfo() (WifiInfo, error) {
	return implGetCurrentWifiInfo()
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func SetWifiNotifier(cb func()) error {
	return implSetWifiNotifier(cb)
}
