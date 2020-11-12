// +build darwin,nowifi

package wifiNotifier

import "github.com/ivpn/desktop-app-daemon/logger"

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func GetAvailableSSIDs() []string {
	return nil
}

// GetCurrentSSID returns current WiFi SSID
func GetCurrentSSID() string {
	return ""
}

// GetCurrentNetworkSecurity returns current security mode
func GetCurrentNetworkSecurity() WiFiSecurity {
	return WiFiSecurityUnknown
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func SetWifiNotifier(cb func(string)) {
	logger.Debug("WiFi functionality disabled in this build")
}
