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

// GetCurrentNetworkIsInsecure returns current security mode
func GetCurrentNetworkIsInsecure() bool {
	return false
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func SetWifiNotifier(cb func(string)) error {
	return nil
}
