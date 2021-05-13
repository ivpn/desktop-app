//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
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

package netinfo

import (
	"bytes"
	"fmt"
	"net"
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
