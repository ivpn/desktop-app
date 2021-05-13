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

// +build windows

package winlib

import (
	"net"
	"syscall"
)

// filter Weights
const (
	weightAllowLocalPort   = 10
	weightAllowApplication = 5
	weightAllowRemoteIP    = 3
	weightAllowRemoteIPV6  = 5
	weightAllowLocalIP     = 10
	weightAllowLocalIPV6   = 10
	weightBlockAll         = 2
	weightBlockDNS         = 4
)

// NewFilterAllowLocalPort creates a filter to allow local port
func NewFilterAllowLocalPort(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string,
	port uint16,
	isPersistent bool) Filter {

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightAllowLocalPort
	f.Action = FwpActionPermit

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	f.AddCondition(&ConditionIPLocalPort{Match: FwpMatchEqual, Port: port})
	return f
}

// NewFilterAllowApplication creates a filter to allow application
func NewFilterAllowApplication(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string,
	binaryPath string,
	isPersistent bool) Filter {

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightAllowApplication
	f.Action = FwpActionPermit

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	f.AddCondition(&ConditionAleAppID{Match: FwpMatchEqual, FullPathTobinary: binaryPath})
	return f
}

// NewFilterAllowRemoteIP creates a filter to allow remote IP
func NewFilterAllowRemoteIP(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string,
	ip net.IP,
	mask net.IP,
	isPersistent bool) Filter {

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightAllowRemoteIP
	f.Action = FwpActionPermit

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	f.AddCondition(&ConditionIPRemoteAddressV4{Match: FwpMatchEqual, IP: ip, Mask: mask})
	return f
}

// NewFilterAllowRemoteIPV6 creates a filter to allow remote IP v6
func NewFilterAllowRemoteIPV6(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string,
	ip net.IP,
	prefixLen byte,
	isPersistent bool) Filter {

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightAllowRemoteIPV6
	f.Action = FwpActionPermit

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	var ipBytes [16]byte
	copy(ipBytes[:], ip)

	f.AddCondition(&ConditionIPRemoteAddressV6{Match: FwpMatchEqual, IP: ipBytes, PrefixLen: prefixLen})
	return f
}

// NewFilterAllowLocalIP creates a filter to allow local IP
func NewFilterAllowLocalIP(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string,
	ip net.IP,
	mask net.IP,
	isPersistent bool) Filter {

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightAllowLocalIP
	f.Action = FwpActionPermit

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	f.AddCondition(&ConditionIPLocalAddressV4{Match: FwpMatchEqual, IP: ip, Mask: mask})
	return f
}

// NewFilterAllowLocalIPV6 creates a filter to allow local IP v6
func NewFilterAllowLocalIPV6(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string,
	ip net.IP,
	prefixLen byte,
	isPersistent bool) Filter {

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightAllowLocalIPV6
	f.Action = FwpActionPermit

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	var ipBytes [16]byte
	copy(ipBytes[:], ip)

	f.AddCondition(&ConditionIPLocalAddressV6{Match: FwpMatchEqual, IP: ipBytes, PrefixLen: prefixLen})
	return f
}

// NewFilterBlockAll creates a filter to block all
func NewFilterBlockAll(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string,
	isIPv6 bool,
	isPersistent bool) Filter {

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightBlockAll
	f.Action = FwpActionBlock

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	if !isIPv6 {
		f.AddCondition(&ConditionIPRemoteAddressV4{Match: FwpMatchEqual, IP: net.IPv4(0, 0, 0, 0), Mask: net.IPv4(0, 0, 0, 0)})
	} else {
		var ipBytes [16]byte
		copy(ipBytes[:], net.IPv6zero)
		f.AddCondition(&ConditionIPRemoteAddressV6{Match: FwpMatchEqual, IP: ipBytes, PrefixLen: 0})
	}

	return f
}

// NewFilterBlockDNS creates a filter to block DNS port
func NewFilterBlockDNS(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string,
	exceptioIP net.IP,
	isPersistent bool) Filter {

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightBlockDNS
	f.Action = FwpActionBlock

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	f.AddCondition(&ConditionIPRemotePort{Match: FwpMatchEqual, Port: 53})
	if exceptioIP != nil && len(exceptioIP) > 0 {
		f.AddCondition(&ConditionIPRemoteAddressV4{Match: FwpMatchNotEqual, IP: exceptioIP, Mask: net.IPv4(255, 255, 255, 255)})
	}
	return f
}
