//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
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

package netchange

import (
	"fmt"
	"net"
	"sync"

	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/oshelpers/linux/netlink"
)

// structure contains properties required for Linux implementation
type osSpecificProperties struct {
	mu       sync.Mutex
	receiver <-chan netlink.NetChangeEvt
}

// isRoutingChanged checks if the default routing interface has changed
// by comparing the current default route interface with the protected interface
func (d *Detector) isRoutingChanged() (bool, error) {
	infToProtect := d.interfaceToProtect
	if infToProtect == nil {
		log.Error("failed to check route change. Initial interface not defined")
		return false, fmt.Errorf("interface to protect is not configured")
	}

	// Check current default routing by testing connectivity to known external IPs
	// Similar approach as used in Windows implementation
	testIPs := []net.IP{net.IPv4(1, 1, 1, 1), net.IPv4(8, 8, 8, 8)}
	var lastErr error

	for _, testIP := range testIPs {
		currentInterface, err := getCurrentRoutingInterface(testIP)
		if err != nil {
			lastErr = err
			continue
		}

		if currentInterface != nil && (currentInterface.Index != infToProtect.Index || currentInterface.Name != infToProtect.Name) {
			log.Info(fmt.Sprintf("Routing change detected. Expected route over '%s' (index %d); current route over '%s' (index %d)",
				infToProtect.Name, infToProtect.Index, currentInterface.Name, currentInterface.Index))
			return true, nil
		}
	}

	if lastErr != nil {
		log.Warning("Failed to determine current routing interface")
	}
	return false, lastErr
}

// getCurrentRoutingInterface determines which interface would be used to route to a specific IP
func getCurrentRoutingInterface(destIP net.IP) (*net.Interface, error) {
	// Get the outbound IP that would be used to connect to destIP
	outboundIP, err := netinfo.GetOutboundIPEx(destIP)
	if err != nil {
		return nil, err
	}
	// Find the interface that has this outbound IP
	return netinfo.InterfaceByIPAddr(outboundIP)
}

// doStart begins monitoring routing changes
func (d *Detector) doStart() {
	d.props.mu.Lock()

	if d.props.receiver != nil {
		log.Warning("Route change detector already started")
		d.props.mu.Unlock()
		return
	}

	receiver, err := netlink.RegisterLanChangeListener()
	if err != nil {
		log.Error("Failed to register LAN change listener:", err)
		d.props.mu.Unlock()
		return
	}

	d.props.receiver = receiver
	d.props.mu.Unlock()

	log.Info("Route change detector started")
	defer log.Info("Route change detector stopped")

	for range receiver {
		d.notifyRoutingChangeWithDelay()
	}
}

// doStop stops monitoring routing changes
func (d *Detector) doStop() {
	d.props.mu.Lock()
	receiver := d.props.receiver
	d.props.receiver = nil
	d.props.mu.Unlock()

	if receiver != nil {
		netlink.UnregisterLanChangeListener(receiver)
	}
}
