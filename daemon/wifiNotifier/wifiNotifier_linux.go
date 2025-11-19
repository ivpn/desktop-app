//go:build linux && !nowifi
// +build linux,!nowifi

package wifiNotifier

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/ivpn/desktop-app/daemon/oshelpers/linux/netlink"
	"github.com/mdlayher/wifi"
)

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func implGetAvailableSSIDs() ([]string, error) {
	// Get shared WiFi client
	client, err := wifi.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create WiFi client: %w", err)
	}
	defer client.Close()

	interfaces, err := client.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get WiFi interfaces: %w", err)
	}

	if len(interfaces) == 0 {
		return []string{}, nil
	}

	var allSSIDs []string
	seen := make(map[string]bool)

	for _, ifi := range interfaces {
		if ifi.Name == "" {
			continue
		}

		// Retry logic for EBUSY errors
		var scanSuccess bool
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				time.Sleep(2 * time.Second)
			}

			// Create context for this specific interface scan
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)

			// Scan for available networks
			err = client.Scan(ctx, ifi)
			cancel()

			if err != nil {
				// Check if the error is EBUSY (device or resource busy)
				var errno syscall.Errno
				if errors.As(err, &errno) && errno == syscall.EBUSY {
					continue // Retry this interface
				}
				break
			}

			// Scan succeeded
			scanSuccess = true
			cancel()
			break
		}

		if !scanSuccess {
			continue // Skip to next interface
		}

		accessPoints, err := client.AccessPoints(ifi)
		if err != nil {
			continue
		}

		// Process access points
		for _, ap := range accessPoints {
			if ap.SSID != "" && !seen[ap.SSID] {
				allSSIDs = append(allSSIDs, ap.SSID)
				seen[ap.SSID] = true
			}
		}
	}

	return allSSIDs, nil
}

// GetCurrentWifiInfo returns current WiFi info
func implGetCurrentWifiInfo() (WifiInfo, error) {
	client, err := wifi.New()
	if err != nil {
		return WifiInfo{}, err
	}
	defer client.Close()

	interfaces, err := client.Interfaces()
	if err != nil {
		return WifiInfo{}, fmt.Errorf("failed to get WiFi interfaces: %w", err)
	}

	if len(interfaces) == 0 {
		return WifiInfo{}, nil
	}

	for _, ifi := range interfaces {
		// Get current BSS (connected network)
		bss, err := client.BSS(ifi)
		if err == nil && bss != nil && bss.SSID != "" {
			isInsecure := checkIsInsecure(bss.RSN)

			ret := WifiInfo{
				SSID:       bss.SSID,
				IsInsecure: isInsecure,
			}
			return ret, nil
		}
	}

	return WifiInfo{}, nil
}

// checkIsInsecure determines if a network is insecure based on RSN information
func checkIsInsecure(rsn wifi.RSNInfo) bool {
	// If RSN is not initialized, it's either an open network or uses legacy security
	if !rsn.IsInitialized() {
		return true // Open network or WEP/WPA1 (no RSN IE)
	}

	isInsecureCipher := func(cipher wifi.RSNCipher) bool {
		switch cipher {
		case wifi.RSNCipherWEP40, // WEP-40
			wifi.RSNCipherWEP104, // WEP-104
			wifi.RSNCipherTKIP:   // TKIP
			return true
		}
		return false
	}

	// Check for insecure ciphers in group cipher
	if isInsecureCipher(rsn.GroupCipher) {
		return true
	}

	// Check for insecure ciphers in pairwise ciphers
	for _, cipher := range rsn.PairwiseCiphers {
		if isInsecureCipher(cipher) {
			return true
		}
	}

	return false // Secure (WPA2+ with strong ciphers)
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func implSetWifiNotifier(cb func()) error {
	if cb == nil {
		return fmt.Errorf("callback function not defined")
	}

	onNetChange, err := netlink.RegisterLanChangeListener()
	if err != nil {
		return err
	}

	go func() {
		for {
			_, ok := <-onNetChange
			if !ok {
				log.Warning("Network change monitor stopped for WiFi notifier")
				break
			}
			cb()
		}
	}()

	return nil
}
