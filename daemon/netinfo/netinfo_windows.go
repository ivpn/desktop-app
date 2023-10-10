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

package netinfo

import (
	"bytes"
	"fmt"
	"net"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

// doDefaultGatewayIP - returns: default gateway IP
func doDefaultGatewayIP() (defGatewayIP net.IP, err error) {
	routes, err := getWindowsIPv4Routes()
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	for _, route := range routes {
		// Eample:
		// Network 		Destination  	Netmask   		Gateway    		Interface  Metric
		// 0.0.0.0   	0.0.0.0      	192.168.1.1 	192.168.1.248	35
		// 0.0.0.0 		128.0.0.0      	10.59.44.1   	10.59.44.2  	15 <- route to virtual VPN interface !!!
		zeroBytes := []byte{0, 0, 0, 0}
		if bytes.Equal(route.DwForwardDest[:], zeroBytes) && bytes.Equal(route.DwForwardMask[:], zeroBytes) { // Network == 0.0.0.0 && Netmask == 0.0.0.0
			return net.IPv4(route.DwForwardNextHop[0],
					route.DwForwardNextHop[1],
					route.DwForwardNextHop[2],
					route.DwForwardNextHop[3]),
				nil
		}
	}

	return nil, fmt.Errorf("failed to determine default route")
}

// DefaultGatewayEx returns the interface that has the default route for the given address family.
func DefaultGatewayEx(isIpv6 bool) (defGatewayIP net.IP, inf *net.Interface, err error) {
	family := winipcfg.AddressFamily(windows.AF_INET)
	if isIpv6 {
		family = winipcfg.AddressFamily(windows.AF_INET6)
	}

	routes, err := winipcfg.GetIPForwardTable2(family)
	if err != nil {
		return nil, nil, err
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	const NOMETRIC = ^uint32(0)
	var (
		bestIf      net.Interface
		bestMetric  uint32 = NOMETRIC
		bestNextHop net.IP
	)
	for _, route := range routes {
		if route.DestinationPrefix.PrefixLength != 0 {
			continue // skip non-default routes
		}
		for _, ifs := range ifaces {
			if uint32(ifs.Index) != route.InterfaceIndex || ifs.Flags&net.FlagUp == 0 || ifs.Flags&net.FlagLoopback == 1 {
				continue // skip down and loopback interfaces
			}
			if route.Metric < bestMetric {
				bestIf = ifs
				bestMetric = route.Metric
				bestNextHop = route.NextHop.Addr().AsSlice()
			}
		}
	}

	if bestMetric == NOMETRIC {
		return nil, nil, fmt.Errorf(fmt.Sprintf("unable to determine default %v route", family))
	}
	return bestNextHop, &bestIf, nil
}
