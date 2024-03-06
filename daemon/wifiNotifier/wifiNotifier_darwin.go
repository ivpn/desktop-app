//go:build darwin && !nowifi
// +build darwin,!nowifi

package wifiNotifier

import (
	"fmt"
	"sync"

	"github.com/ivpn/desktop-app/daemon/oshelpers/macos/darwinhelpers"
	"github.com/ivpn/desktop-app/daemon/wifiNotifier/darwin"
	agentxpc "github.com/ivpn/desktop-app/daemon/wifiNotifier/darwin/agent_xpc"
	"github.com/ivpn/desktop-app/daemon/wifiNotifier/darwin/obsolete"
)

type DarwinWifiInfoSource interface {
	Init(darwin.Logger) error
	GetCurrentWifiInfo() (ssid string, security int, err error)
	GetAvailableSSIDs() ([]string, error)
	SetWifiNotifier(cb func()) error
}

var (
	locker sync.Mutex
	source DarwinWifiInfoSource
	once   sync.Once
)

func getSource() DarwinWifiInfoSource {
	once.Do(func() {
		locker.Lock()
		defer locker.Unlock()

		source = agentxpc.GetWifiSourceInstance()

		// Check if we can use old-style native API
		// Since macOS 14 Sonoma (Darwin v23.x.x) it is not possible anymore to obtain WiFi SSID for background daemons.
		// We need to use XPC service for this (to ask our LaunchAgent for this info).
		if majorVer, err := darwinhelpers.GetOsMajorVersion(); err != nil {
			log.Error(fmt.Errorf("Can not obtain macOS version: %w", err))
		} else {
			if majorVer < 23 {
				// obsolete API works, no need to use XPC
				source = obsolete.GetWifiSourceInstance()
			}
		}

		source.Init(log)
	})

	return source
}

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func implGetAvailableSSIDs() ([]string, error) {
	locker.Lock()
	defer locker.Unlock()
	return getSource().GetAvailableSSIDs()
}

// GetCurrentWifiInfo returns current WiFi info
func implGetCurrentWifiInfo() (WifiInfo, error) {
	locker.Lock()
	defer locker.Unlock()

	ssid, security, err := getSource().GetCurrentWifiInfo()
	wInfo := WifiInfo{
		SSID: ssid,
	}
	if len(ssid) > 0 {
		wInfo.IsInsecure = IsInsecure(security)
	}

	return wInfo, err
}

// GetCurrentNetworkIsInsecure returns current security mode
func IsInsecure(security int) bool {
	const (
		CWSecurityNone               = 0
		CWSecurityWEP                = 1
		CWSecurityWPAPersonal        = 2
		CWSecurityWPAPersonalMixed   = 3
		CWSecurityWPA2Personal       = 4
		CWSecurityPersonal           = 5
		CWSecurityDynamicWEP         = 6
		CWSecurityWPAEnterprise      = 7
		CWSecurityWPAEnterpriseMixed = 8
		CWSecurityWPA2Enterprise     = 9
		CWSecurityEnterprise         = 10
		CWSecurityWPA3Personal       = 11
		CWSecurityWPA3Enterprise     = 12
		CWSecurityWPA3Transition     = 13
	)

	switch security {
	case CWSecurityNone,
		CWSecurityWEP,
		CWSecurityDynamicWEP:
		return true
	}
	return false
}

// SetWifiNotifier initializes a callback (event handler for 'OnWifiChanged')
func implSetWifiNotifier(cb func()) error {
	return getSource().SetWifiNotifier(cb)
}
