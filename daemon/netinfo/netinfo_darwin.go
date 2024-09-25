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
	"fmt"
	"net"
	"syscall"

	"golang.org/x/net/route"
)

// IsDefaultRoutingInterface - Check if interface is default IPv4 routing interface ('default' or '0/1'+'128/1' route)
func IsDefaultRoutingInterface(interfaceName string) (bool, error) {
	// get interface info by name (to know the interface index)
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return false, fmt.Errorf("unable to get interface by name: %w", err)
	}

	// Check "0/1" and "128/1" routes (they are more specific than "default" route)
	_, dst0, err := net.ParseCIDR("0.0.0.0/1") // "0/1" route
	if err != nil {
		return false, err
	}
	_, dst1, err := net.ParseCIDR("128.0.0.0/1") // "128.0/1" route
	if err != nil {
		return false, err
	}
	gwIp0, ifIdx0, _ := findRouteByDestination(*dst0)
	if gwIp0 != nil {
		gwIp1, ifIdx1, _ := findRouteByDestination(*dst1)
		if gwIp1 != nil {
			// Check if the interface index is the same as the one we are looking for and the gateways are equal
			if iface.Index == ifIdx0 && ifIdx0 == ifIdx1 && gwIp0.Equal(gwIp1) {
				return true, nil
			}
		}
	}

	// Check "default" route (destination = 0.0.0.0 and netmask = 0.0.0.0/0)
	defaultDST := &net.IPNet{
		IP:   net.IPv4(0, 0, 0, 0),
		Mask: net.CIDRMask(0, 32),
	}
	_, ifIdx, _ := findRouteByDestination(*defaultDST)
	if ifIdx >= 0 {
		// Check if the interface index is the same as the one we are looking for
		if iface.Index == ifIdx {
			return true, nil
		}
	}

	return false, nil
}

// doDefaultGatewayIP - returns: 'default' IPv4 gateway IP address
func doDefaultGatewayIP() (defGatewayIP net.IP, err error) {
	// 'default' route: destination = 0.0.0.0 and netmask = 0.0.0.0/0
	defaultDST := &net.IPNet{
		IP:   net.IPv4(0, 0, 0, 0),
		Mask: net.CIDRMask(0, 32),
	}
	gwIp, _, err := findRouteByDestination(*defaultDST)
	if err != nil {
		return nil, fmt.Errorf("unable to find default gateway IP: %w", err)
	}

	if gwIp != nil {
		return gwIp, nil
	}

	return nil, fmt.Errorf("default route not found")
}

// const values for parsing route message addresses
// https://man.openbsd.org/rtrequest.9
const (
	RTAX_DST     = 0 // destination sockaddr presents
	RTAX_GATEWAY = 1 // gateway sockaddr present
	RTAX_NETMASK = 2 // netmask sockaddr present
)

// findRouteByDestination finds the route that matches the given destination network.
// It returns the gateway IP and the interface index for the matching route,
// or (nil, -1, nil) if no route matches.
func findRouteByDestination(dst net.IPNet) (gatewayIP net.IP, interfaceIndex int, err error) {
	// Fetch the IPv4 routing table
	rib, err := route.FetchRIB(syscall.AF_INET, route.RIBTypeRoute, 0)
	if err != nil {
		return nil, -1, fmt.Errorf("unable to fetch routing table: %w", err)
	}

	// Parse the routing table
	msgs, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		return nil, -1, fmt.Errorf("unable to parse routing table: %w", err)
	}

	// Iterate through the routing table entries
	for _, msg := range msgs {
		if m, ok := msg.(*route.RouteMessage); ok {
			if isRouteMatch(m, dst) {
				if r_gw, ok := m.Addrs[RTAX_GATEWAY].(*route.Inet4Addr); ok {
					return net.IP(r_gw.IP[:]), m.Index, nil
				}
			}
		}
	}

	return nil, -1, nil
}

// isRouteMatch checks if the route message matches the given destination network.
func isRouteMatch(m *route.RouteMessage, dst net.IPNet) bool {
	if m.Flags&syscall.RTF_IFSCOPE != 0 {
		return false
	}
	if len(m.Addrs) <= RTAX_NETMASK || m.Addrs[RTAX_NETMASK] == nil || m.Addrs[RTAX_DST] == nil {
		return false
	}

	r_dst, ok := m.Addrs[RTAX_DST].(*route.Inet4Addr)
	if !ok {
		return false
	}
	r_mask, ok := m.Addrs[RTAX_NETMASK].(*route.Inet4Addr)
	if !ok {
		return false
	}

	routeDst := net.IPNet{
		IP:   net.IP(r_dst.IP[:]),
		Mask: r_mask.IP[:],
	}
	return routeDst.String() == dst.String()
}
