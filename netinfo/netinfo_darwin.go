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

// DefaultRoutingInterface - Get active routing interface
func DefaultRoutingInterface() (*net.Interface, error) {
	_, interfaceName, err := doGetDefaultGateway()
	if err != nil {
		return nil, err
	}
	return net.InterfaceByName(interfaceName)
}

// doDefaultGatewayIP - returns: default gateway
func doDefaultGatewayIP() (defGatewayIP net.IP, err error) {
	defGatewayIP, _, err = doGetDefaultGateway()
	return defGatewayIP, err
}

func doGetDefaultGateway() (defGatewayIP net.IP, interfaceName string, err error) {
	defGatewayIP = nil
	// Expected output of "netstat -nr" command:
	//	Routing tables
	//	Internet:
	//	Destination        Gateway            Flags        Netif Expire
	//	default            192.168.1.1        UGSc           en0
	//	127                127.0.0.1          UCS            lo0
	// ...

	log.Info("Checking default getaway info ...")
	cmd := exec.Command("/usr/sbin/netstat", "-nr")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return defGatewayIP, "", fmt.Errorf("unable to obtain default gateway IP: %w", err)
	}
	if len(out) == 0 {
		return defGatewayIP, "", fmt.Errorf("unable to obtain default gateway IP (netstat returns no data)")
	}

	//default            192.168.1.1        UGSc           en0
	outRegexp := regexp.MustCompile("default[\t ]+([0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3})[\t ]*[A-Za-z]*[\t ]+([A-Za-z0-9]*)")
	columns := outRegexp.FindStringSubmatch(string(out))
	if len(columns) < 3 {
		return nil, "", fmt.Errorf("unable to obtain default gateway IP")
	}

	defGatewayIP = net.ParseIP(strings.Trim(columns[1], " \n\r\t"))
	interfaceName = strings.Trim(columns[2], " \n\r\t")

	if defGatewayIP == nil {
		return nil, "", fmt.Errorf("unable to obtain default gateway IP")
	}
	if len(interfaceName) == 0 {
		return nil, "", fmt.Errorf("unable to obtain default interface name")
	}

	return defGatewayIP, interfaceName, nil
}
