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

package dns

import (
	"net"

	"github.com/ivpn/desktop-app-daemon/logger"
)

var log *logger.Logger
var lastManualDNS net.IP

func init() {
	log = logger.NewLogger("dns")
}

// Initialize is doing initialization stuff
// Must be called on application start
func Initialize() error {
	return implInitialize()
}

// Pause pauses DNS (restore original DNS)
func Pause() error {
	return implPause()
}

// Resume resuming DNS (set DNS back which was before Pause)
func Resume(defaultDNS net.IP) error {
	return implResume(defaultDNS)
}

// SetManual - set manual DNS.
// 'addr' parameter - DNS IP value
// 'localInterfaceIP' (obligatory only for Windows implementation) - local IP of VPN interface
func SetManual(addr net.IP, localInterfaceIP net.IP) error {
	ret := implSetManual(addr, localInterfaceIP)
	if ret == nil {
		lastManualDNS = addr
	}
	return ret
}

// DeleteManual - reset manual DNS configuration to default (DHCP)
// 'localInterfaceIP' (obligatory only for Windows implementation) - local IP of VPN interface
func DeleteManual(localInterfaceIP net.IP) error {
	ret := implDeleteManual(localInterfaceIP)
	if ret == nil {
		lastManualDNS = nil
	}
	return ret
}

// GetLastManualDNS - returns information about current manual DNS
func GetLastManualDNS() string {
	// TODO: get real DNS configuration of the OS
	dns := lastManualDNS
	if dns == nil {
		return ""
	}
	return dns.String()
}
