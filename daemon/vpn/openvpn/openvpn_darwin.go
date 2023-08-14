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
)

type platformSpecificProperties struct {
	// no specific properties for macOS implementation
}

func (o *OpenVPN) implInit() error             { return nil }
func (o *OpenVPN) implIsCanUseParamsV24() bool { return true }

func (o *OpenVPN) implOnConnected() error {
	// not in use in macOS implementation
	return nil
}

func (o *OpenVPN) implOnDisconnected() error {
	// not in use in macOS implementation
	return nil
}

func (o *OpenVPN) implOnPause() error {
	return dns.Pause(o.clientIP)
}

func (o *OpenVPN) implOnResume() error {
	return dns.Resume(dns.DnsSettings{}, o.clientIP)
}

func (o *OpenVPN) implOnSetManualDNS(dnsCfg dns.DnsSettings) error {
	return dns.SetManual(dnsCfg, nil)
}

func (o *OpenVPN) implOnResetManualDNS() error {
	return dns.DeleteManual(o.DefaultDNS(), nil)
}

func (o *OpenVPN) implGetUpDownScriptArgs() string {
	return ""
}
