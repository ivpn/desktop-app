//go:build nowifi
// +build nowifi

package wifiNotifier

import "github.com/ivpn/desktop-app/daemon/logger"

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func implGetAvailableSSIDs() []string {
	return nil
}

// GetCurrentWifiInfo returns current WiFi info
func implGetCurrentWifiInfo() (WifiInfo, error) {
	return WifiInfo{}, nil
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func implSetWifiNotifier(cb func()) error {
	logger.Debug("WiFi functionality disabled in this build")
	return nil
}
