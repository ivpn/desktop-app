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

package netinfo

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
)

// IsDefaultRoutingInterface - Get active routing interface
func IsDefaultRoutingInterface(interfaceName string) (bool, error) {
	routes, e := doGetDefaultRoutes(true)
	if e != nil {
		return false, e
	}

	for _, r := range routes {
		if strings.Compare(r.interfaceName, interfaceName) == 0 {
			return true, nil
		}
	}

	return false, nil
}

// doDefaultGatewayIP - returns: default gateway
func doDefaultGatewayIP() (defGatewayIP net.IP, err error) {
	routes, e := doGetDefaultRoutes(false)
	if e != nil {
		return nil, e
	}

	return routes[0].gatewayIP, nil
}

type route struct {
	gatewayIP     net.IP
	interfaceName string
}

func doGetDefaultRoutes(getAllDefRoutes bool) (routes []route, err error) {
	// Expected output of "netstat -nr" command:
	//	Routing tables
	//	Internet:
	//	Destination        Gateway            Flags        Netif Expire
	//  0/1                10.56.40.1         UGSc         	 utun
	//	default            192.168.1.1        UGSc           en0
	//	127                127.0.0.1          UCS            lo0
	// ...

	routes = make([]route, 0, 3)

	log.Info("Checking default getaway info ...")
	cmd := exec.Command("/usr/sbin/netstat", "-nr")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("unable to obtain default gateway IP: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("unable to obtain default gateway IP (netstat returns no data)")
	}

	//default            192.168.1.1        UGSc           en0
	regExpString := "default[\t ]+([0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3})[\t ]*[A-Za-z]*[\t ]+([A-Za-z0-9]*)"
	columnsOffsetIdx := 0
	if getAllDefRoutes {
		regExpString = "((0/1)|(default))[\t ]+([0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3})[\t ]*[A-Za-z]*[\t ]+([A-Za-z0-9]*)"
		columnsOffsetIdx = 3
	}

	outRegexp := regexp.MustCompile(regExpString)

	maches := outRegexp.FindAllStringSubmatch(string(out), -1)
	for _, m := range maches {
		if len(m) < 3+columnsOffsetIdx {
			continue
		}

		gatewayIP := net.ParseIP(strings.Trim(m[1+columnsOffsetIdx], " \n\r\t"))
		interfaceName := strings.Trim(m[2+columnsOffsetIdx], " \n\r\t")

		if gatewayIP == nil {
			continue
		}
		if len(interfaceName) == 0 {
			continue
		}

		routes = append(routes, route{gatewayIP: gatewayIP, interfaceName: interfaceName})
	}

	if len(routes) <= 0 {
		return nil, fmt.Errorf("unable to obtain default gateway IP")
	}

	return routes, nil
}
