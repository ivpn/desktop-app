//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package firewall

import (
	"fmt"
	"net"
	"sync"

	"github.com/ivpn/desktop-app-daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("frwl")
}

var (
	connectedClientInterfaceIP net.IP
	mutex                      sync.Mutex
	isClientPaused             bool
)

// Initialize is doing initialization stuff
// Must be called on application start
func Initialize() error {
	return implInitialize()
}

// SetEnabled - change firewall state
func SetEnabled(enable bool) error {
	mutex.Lock()
	defer mutex.Unlock()

	if enable {
		log.Info("Enabling...")
	} else {
		log.Info("Disabling...")
	}

	err := implSetEnabled(enable)
	if err != nil {
		log.Error(err)
		return fmt.Errorf("failed to change firewall state : %w", err)
	}

	if enable {
		// To fulfill such flow (example): FWEnable -> Connected -> FWDisable -> FWEnable
		// Here we should notify that client is still connected
		// We must not do it in Paused state!
		clientAddr := connectedClientInterfaceIP
		if clientAddr != nil && isClientPaused == false {
			return implClientConnected(clientAddr)
		}
	}
	return nil
}

// SetPersistant - set persistant firewall state and enable it if necessary
func SetPersistant(persistant bool) error {
	mutex.Lock()
	defer mutex.Unlock()

	log.Info(fmt.Sprintf("Persistent:%t", persistant))

	err := implSetPersistant(persistant)
	if err != nil {
		log.Error(err)
	}
	return err
}

// GetEnabled - get firewall state
func GetEnabled() (bool, error) {
	mutex.Lock()
	defer mutex.Unlock()
	log.Info("Getting status...")

	ret, err := implGetEnabled()
	if err != nil {
		log.Error(err)
	} else {
		log.Info("\t", ret)
	}

	return ret, err
}

// ClientPaused saves info about paused state of vpn
func ClientPaused() {
	isClientPaused = true
}

// ClientResumed saves info about resumed state of vpn
func ClientResumed() {
	isClientPaused = false
}

// ClientConnected - allow communication for local vpn/client IP address
func ClientConnected(clientLocalIPAddress net.IP) error {
	mutex.Lock()
	defer mutex.Unlock()
	ClientResumed()

	log.Info("Client connected: ", clientLocalIPAddress)

	connectedClientInterfaceIP = clientLocalIPAddress
	err := implClientConnected(clientLocalIPAddress)
	if err != nil {
		log.Error(err)
	}
	return err
}

// ClientDisconnected - Remove all hosts exceptions
func ClientDisconnected() error {
	mutex.Lock()
	defer mutex.Unlock()
	ClientResumed()

	// Remove client interface from exceptions
	if connectedClientInterfaceIP != nil {
		connectedClientInterfaceIP = nil
		log.Info("Client disconnected")
		err := implClientDisconnected()
		if err != nil {
			log.Error(err)
		}
		return err
	}
	return nil
}

// AddHostsToExceptions - allow comminication with this hosts
// Note!: all added hosts will be removed from exceptions after client disconnection (after call 'ClientDisconnected()')
func AddHostsToExceptions(IPs []net.IP) error {
	mutex.Lock()
	defer mutex.Unlock()

	err := implAddHostsToExceptions(IPs)
	if err != nil {
		log.Error("Failed to add hosts to exceptions:", err)
	}

	return err
}

// AllowLAN - allow/forbid LAN communication
func AllowLAN(allowLan bool, allowLanMulticast bool) error {
	mutex.Lock()
	defer mutex.Unlock()

	log.Info(fmt.Sprintf("allowLan:%t allowMulticast:%t", allowLan, allowLanMulticast))

	err := implAllowLAN(allowLan, allowLanMulticast)
	if err != nil {
		log.Error(err)
	}
	return err
}

// SetManualDNS - configure firewall to allow DNS which is out of VPN tunnel
// Applicable to Windows implementation (to allow custom DNS from local network)
func SetManualDNS(addr net.IP) error {
	mutex.Lock()
	defer mutex.Unlock()

	err := implSetManualDNS(addr)
	if err != nil {
		log.Error(err)
	}
	return err
}
