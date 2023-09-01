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

package openvpn

import (
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

type platformSpecificProperties struct {
	manualDNS dns.DnsSettings
}

func (o *OpenVPN) implInit() error             { return nil }
func (o *OpenVPN) implIsCanUseParamsV24() bool { return true }

func (o *OpenVPN) implOnConnected() error {
	// on Windows it is not possible to change network interface properties until it not enabled
	// apply DNS value when VPN connected (TAP interface enabled)
	if !o.psProps.manualDNS.IsEmpty() {
		return dns.SetManual(o.psProps.manualDNS, o.clientIP)
	}

	// There could be manual-dns value saved from last connection in adapter properties. We must ensure that it erased.
	return dns.DeleteManual(o.DefaultDNS(), o.clientIP)
}

func (o *OpenVPN) implOnDisconnected() error {
	return o.implOnResetManualDNS()
}

func (o *OpenVPN) implOnPause() error {
	// not in use in Windows implementation
	return nil
}

func (o *OpenVPN) implOnResume() error {
	// not in use in Windows implementation
	return nil
}

func (o *OpenVPN) implOnSetManualDNS(dnsCfg dns.DnsSettings) error {
	o.psProps.manualDNS = dnsCfg

	if o.state != vpn.CONNECTED {
		// on Windows it is not possible to change network interface properties until it not enabled
		// apply DNS value when VPN connected (TAP interface enabled)
	} else {
		return dns.SetManual(o.psProps.manualDNS, o.clientIP)
	}
	return nil
}

func (o *OpenVPN) implOnResetManualDNS() error {
	if !o.psProps.manualDNS.IsEmpty() {
		o.psProps.manualDNS = dns.DnsSettings{}
		return dns.DeleteManual(o.DefaultDNS(), o.clientIP)
	}
	return nil
}

func (o *OpenVPN) implGetUpDownScriptArgs() string {
	return ""
}
