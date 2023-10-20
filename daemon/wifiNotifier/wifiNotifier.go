package wifiNotifier

import "github.com/ivpn/desktop-app/daemon/logger"

var log *logger.Logger

func init() {
	log = logger.NewLogger("wifi")
}

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func GetAvailableSSIDs() []string {
	return implGetAvailableSSIDs()
}

// GetCurrentSSID returns current WiFi SSID
func GetCurrentSSID() string {
	return implGetCurrentSSID()
}

// GetCurrentNetworkIsInsecure returns current security mode
func GetCurrentNetworkIsInsecure() bool {
	return implGetCurrentNetworkIsInsecure()
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func SetWifiNotifier(cb func(string)) error {
	return implSetWifiNotifier(cb)
}
