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
	"fmt"
	"net"

	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/shell"
)

type platformSpecificProperties struct {
	// The interface-scoped default route added by implOnConnected() (empty if none was added).
	scopedDefaultGateway   net.IP
	scopedDefaultInterface string
}

func (o *OpenVPN) implInit() error             { return nil }
func (o *OpenVPN) implIsCanUseParamsV24() bool { return true }

// Split Tunnel relays excluded apps via a socket bound to the physical interface
// (IP_BOUND_IF), and macOS only routes such sockets using routes scoped to that
// interface - the plain default route is ignored. OpenVPN's 'redirect-gateway def1'
// replaces the default route with 0/1 + 128.0.0.0/1, leaving the physical interface
// without a scoped default, so relayed traffic can't egress. Restore it here, mirroring
// what setRoutes() does for WireGuard.
func (o *OpenVPN) implOnConnected() error {
	gatewayIP, _, interfaceName, err := netinfo.GetDefaultRouteInfo()
	if err != nil || gatewayIP == nil || len(interfaceName) == 0 {
		// Not fatal: only Split Tunnel needs this route, so a failure here must not
		// bring down an otherwise healthy VPN connection.
		log.Warning(fmt.Sprintf("unable to determine the default route (%v): Split Tunnel will not work for this connection", err))
		return nil
	}

	// sudo route -n add -inet default 192.168.1.1 -ifscope en0
	if err := shell.Exec(log, "/sbin/route", "-n", "add", "-inet", "default", gatewayIP.String(), "-ifscope", interfaceName); err != nil {
		log.Warning(fmt.Sprintf("failed to add the interface-scoped default route for '%s' (%v): Split Tunnel will not work for this connection", interfaceName, err))
		return nil
	}

	o.psProps.scopedDefaultGateway = gatewayIP
	o.psProps.scopedDefaultInterface = interfaceName
	return nil
}

func (o *OpenVPN) implOnDisconnected() error {
	if o.psProps.scopedDefaultGateway == nil || len(o.psProps.scopedDefaultInterface) == 0 {
		return nil
	}

	if err := shell.Exec(log, "/sbin/route", "-n", "delete", "-inet", "default",
		o.psProps.scopedDefaultGateway.String(), "-ifscope", o.psProps.scopedDefaultInterface); err != nil {
		log.Warning(fmt.Sprintf("failed to delete the interface-scoped default route for '%s': %v", o.psProps.scopedDefaultInterface, err))
	}

	o.psProps.scopedDefaultGateway = nil
	o.psProps.scopedDefaultInterface = ""
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
