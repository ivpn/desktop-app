// +build windows

package wifiNotifier

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

}
