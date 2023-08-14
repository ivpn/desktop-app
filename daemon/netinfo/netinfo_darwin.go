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
		if strings.Compare(r.InterfaceName, interfaceName) == 0 {
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

	return routes[0].GatewayIP, nil
}

type Route struct {
	Destination   string
	GatewayIP     net.IP
	Flags         string
	InterfaceName string
}

func (r Route) IsSpecified() bool {
	return r.GatewayIP != nil && !r.GatewayIP.IsUnspecified()
}

func GetDefaultRoutes() (routes []Route, err error) {
	return doGetDefaultRoutes(true)
}

// doGetDefaultRoutes returns all main routes
//
//	 getAllDefRoutes == false:
//			returns "default" route
//	 getAllDefRoutes == true:
//			returns all "default" and "0/1" routes
func doGetDefaultRoutes(getAllDefRoutes bool) (routes []Route, err error) {
	// Expected output of "netstat -nr" command:
	//	Routing tables
	//	Internet:
	//	Destination        Gateway            Flags        Netif Expire
	//	0/1                10.56.40.1         UGSc      	 utun
	//	default            192.168.1.1        UGSc           en0
	//	127                127.0.0.1          UCS            lo0
	// ...

	routes = make([]Route, 0, 3)

	log.Info("Checking default getaway info ...")
	cmd := exec.Command("/usr/sbin/netstat", "-nr", "-f", "inet")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("unable to obtain default gateway IP: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("unable to obtain default gateway IP (netstat returns no data)")
	}

	//default            192.168.1.1        UGSc           en0
	// (?m) enables multiline mode, which makes ^ and $ match the start and end of each line (not just the start and end of the entire string).
	regExpString := `(?m)^\s*((default)|(default))\s+([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})\s*([A-Za-z]*)\s+([A-Za-z0-9]*)`
	if getAllDefRoutes {
		regExpString = `(?m)^\s*((0/1)|(default))\s+([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})\s*([A-Za-z]*)\s+([A-Za-z0-9]*)`
	}

	outRegexp := regexp.MustCompile(regExpString)

	maches := outRegexp.FindAllStringSubmatch(string(out), -1)
	for _, m := range maches {
		if len(m) < 7 {
			continue
		}

		destination := strings.Trim(m[1], " \n\r\t")
		gatewayIP := net.ParseIP(strings.Trim(m[4], " \n\r\t"))
		flags := strings.Trim(m[5], " \n\r\t")
		interfaceName := strings.Trim(m[6], " \n\r\t")

		if gatewayIP == nil {
			continue
		}
		if len(interfaceName) == 0 {
			continue
		}

		routes = append(routes, Route{Destination: destination, GatewayIP: gatewayIP, InterfaceName: interfaceName, Flags: flags})
	}

	if len(routes) <= 0 {
		return nil, fmt.Errorf("unable to obtain default gateway IP")
	}

	return routes, nil
}
