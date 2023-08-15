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

//go:build windows
// +build windows

package winlib

import (
	"net"
	"syscall"
)

// filter Weights
const (
	// IMPORTANT! Use only for Local IP/IPv6 of VPN connection
	weightAllowLocalIP            = 10
	weightAllowRemoteLocalhostDNS = 10 // allow DNS requests to 127.0.0.1:53
	weightAllowApplication        = 10 // must have higher priority than weightBlockDNS (to allow port UDP:53 for VPN connections)

	// IMPORTANT! Blocking DNS must have highest priority
	// (only VPN connection have higher priority: weightAllowLocalIP;weightAllowLocalIPV6) //5
	weightBlockDNS = 9

	weightAllowLocalPort = 3
	weightAllowRemoteIP  = 3

	weightBlockAll = 2
	// NOTE: If split-tunnelling not enabled (driver not registered callouts) - this filter will BLOCK everything
	// But it is ok since ST-filters weight = weightBlockAll + 1
	// weightAllowSplittedApps = 3

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

/*
// NewFilterAllowRemotePort creates a filter to allow remote port
func NewFilterAllowRemotePort(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string,
	port uint16,
	isPersistent bool) Filter {

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightAllowRemotePort
	f.Action = FwpActionPermit

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	f.AddCondition(&ConditionIPRemotePort{Match: FwpMatchEqual, Port: port})
	return f
}
*/

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

// AllowRemoteLocalhostDNS allow DNS requests to 127.0.0.1:53
func AllowRemoteLocalhostDNS(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string,
	isPersistent bool) Filter {

	ip := net.ParseIP("127.0.0.1")
	mask := net.ParseIP("255.255.255.255")

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightAllowRemoteLocalhostDNS
	f.Action = FwpActionPermit

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	f.AddCondition(&ConditionIPRemoteAddressV4{Match: FwpMatchEqual, IP: ip, Mask: mask})
	f.AddCondition(&ConditionIPRemotePort{Match: FwpMatchEqual, Port: 53})
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
	f.Weight = weightAllowRemoteIP
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
// (IMPORTANT! Use only for Local IP of VPN connection)
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

	// Do not set FwpmFilterFlagClearActionRight (f.Flags = FwpmFilterFlagClearActionRight)
	// Otherwise, we will overlap blocking rules from Windows Firewall (if they are)
	// For example: if the Windows firewall have rule to block a specific application
	//		-> using FwpmFilterFlagClearActionRight will allow to communicate from 'ip' for this application

	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	f.AddCondition(&ConditionIPLocalAddressV4{Match: FwpMatchEqual, IP: ip, Mask: mask})
	return f
}

// NewFilterAllowLocalIPV6 creates a filter to allow local IP v6
// (IMPORTANT! Use only for Local IPv6 of VPN connection)
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
	f.Weight = weightAllowLocalIP
	f.Action = FwpActionPermit

	// Do not set FwpmFilterFlagClearActionRight (f.Flags = FwpmFilterFlagClearActionRight)
	// Otherwise, we will overlap blocking rules from Windows Firewall (if they are)
	// For example: if the Windows firewall have rule to block a specific application
	//		-> using FwpmFilterFlagClearActionRight will allow to communicate from 'ip' for this application

	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	var ipBytes [16]byte
	copy(ipBytes[:], ip)

	f.AddCondition(&ConditionIPLocalAddressV6{Match: FwpMatchEqual, IP: ipBytes, PrefixLen: prefixLen})
	return f
}

// NewFilterBlockAll creates a filter to block all
// Note: Arguments 'isPersistent' and 'isBootTime' cannot be set together!
func NewFilterBlockAll(
	keyProvider syscall.GUID,
	keyLayer syscall.GUID,
	keySublayer syscall.GUID,
	dispName string,
	dispDescription string,
	isIPv6 bool,
	isPersistent bool, isBootTime bool) Filter {

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightBlockAll
	f.Action = FwpActionBlock

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	} else if isBootTime {
		f.Flags = f.Flags | FwpmFilterFlagBoottime
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
	exceptionIP net.IP,
	isPersistent bool) Filter {

	f := NewFilter(keyProvider, keyLayer, keySublayer, dispName, dispDescription)
	f.Weight = weightBlockDNS
	f.Action = FwpActionBlock

	f.Flags = FwpmFilterFlagClearActionRight
	if isPersistent {
		f.Flags = f.Flags | FwpmFilterFlagPersistent
	}

	f.AddCondition(&ConditionIPRemotePort{Match: FwpMatchEqual, Port: 53})

	if exceptionIP != nil && len(exceptionIP) > 0 && exceptionIP.To4() != nil {
		f.AddCondition(&ConditionIPRemoteAddressV4{Match: FwpMatchNotEqual, IP: exceptionIP, Mask: net.IPv4(255, 255, 255, 255)})
	}
	return f
}
