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
	"regexp"
	"strconv"

	"github.com/ivpn/desktop-app/daemon/shell"
)

// doDefaultGatewayIP - returns: default gateway
func doDefaultGatewayIP() (defGatewayIP net.IP, err error) {
	defGatewayIP = nil
	// Expected output of "/sbin/ip route" command:
	// (if more then one default gateways - use  one with smaller metric value)
	//
	// default via 192.168.1.1 dev enp0s3 proto dhcp metric 100
	// 192.168.1.0/24 dev enp0s3 proto kernel scope link src 192.168.1.57 metric 100
	// 192.168.122.0/24 dev virbr0 proto kernel scope link src 192.168.122.1 linkdown

	metric := -1
	outRegexp := regexp.MustCompile("default[ a-z]*([0-9.]*).*metric *([0-9]*)")

	outParse := func(text string, isError bool) {
		if !isError {
			columns := outRegexp.FindStringSubmatch(text)
			if len(columns) <= 2 {
				return
			}

			gw := net.ParseIP(columns[1])
			if gw == nil {
				return
			}
			m, err := strconv.Atoi(columns[2])
			if err != nil {
				return
			}
			if metric == -1 || metric > m {
				defGatewayIP = gw
				metric = m
			}
		}
	}

	retErr := shell.ExecAndProcessOutput(log, outParse, "", "/sbin/ip", "route")

	if retErr == nil {
		if defGatewayIP == nil {
			retErr = fmt.Errorf("Unable to obtain default gateway IP")
		}
	} else {
		log.Error("Failed to obtain local gateway: ", retErr.Error())
	}

	return defGatewayIP, retErr
}
