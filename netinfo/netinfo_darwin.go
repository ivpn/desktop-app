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
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/ivpn/desktop-app-daemon/shell"
)

// DefaultRoutingInterface - Get active routing interface
func DefaultRoutingInterface() (*net.Interface, error) {
	return doDefaultRoutingInterface()
}

// doDefaultRoutingInterface - Get active routing interface
func doDefaultRoutingInterface() (*net.Interface, error) {
	// 8.8.8.8 - is well known Google DNS IP
	_, interfaceName, err := getRoute("8.8.8.8")
	if err != nil {
		return nil, fmt.Errorf("failed to get route : %w", err)
	}

	return interfaceByName(interfaceName)
}

// doDefaultGatewayIP - returns: default gateway
func doDefaultGatewayIP() (defGatewayIP net.IP, err error) {
	gatewayIP, _, err := getRoute("default")
	if err != nil {
		return nil, fmt.Errorf("failed to get default route: %w", err)
	}

	return gatewayIP, nil
}

// defaultGatewayIP - returns: default gateway IP and default interface name
func getRoute(routeTo string) (gatewayIP net.IP, interfaceName string, err error) {
	gatewayIP = nil
	interfaceName = ""

	outParse := func(text string, isError bool) {
		if !isError {
			if strings.Contains(text, "gateway:") {
				cols := strings.Split(text, ":")
				if len(cols) == 2 {
					gatewayIP = net.ParseIP(strings.Trim(cols[1], " \n\r\t"))
				}
			} else if strings.Contains(text, "interface:") {
				cols := strings.Split(text, ":")
				if len(cols) == 2 {
					interfaceName = strings.Trim(cols[1], " \n\r\t")
				}
			}
		}
	}

	retErr := shell.ExecAndProcessOutput(log, outParse, "", "/sbin/route", "-n", "get", routeTo) // routeTo = "default" ir IP (e.g. 8.8.8.8)

	if retErr == nil {
		var errorText string

		if gatewayIP == nil {
			log.Error("Unable to obtain default gateway IP")
			errorText += "Unable to obtain default gateway IP "
		}
		if interfaceName == "" {
			log.Error("Unable to obtain default interface name")
			errorText += "Unable to obtain default gateway IP "
		}
		if len(errorText) > 0 {
			retErr = errors.New(errorText)
		}
	} else {
		log.Error("Failed to obtain local gateway: ", retErr.Error())
	}

	return gatewayIP, interfaceName, retErr
}

func interfaceByName(interfaceName string) (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain network interfaces: %w", err)
	}

	for _, ifs := range ifaces {

		addrs, _ := ifs.Addrs()
		if addrs == nil {
			continue
		}

		if ifs.Name == interfaceName {
			return &ifs, nil
		}
	}
	return nil, errors.New("not found network interface : '" + interfaceName + "'")
}
