//go:build nowifi
// +build nowifi

package wifiNotifier

import "github.com/ivpn/desktop-app/daemon/logger"

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func implGetAvailableSSIDs() []string {
	return nil
}

// GetCurrentSSID returns current WiFi SSID
func implGetCurrentSSID() string {
	return ""
}

// GetCurrentNetworkIsInsecure returns current security mode
func implGetCurrentNetworkIsInsecure() bool {
	return false
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func implSetWifiNotifier(cb func(string)) error {
	logger.Debug("WiFi functionality disabled in this build")
	return nil
}
