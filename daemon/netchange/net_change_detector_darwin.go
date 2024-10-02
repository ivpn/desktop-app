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

package netchange

import (
	"net"
	"os"
	"syscall"

	"github.com/ivpn/desktop-app/daemon/netinfo"

	"golang.org/x/net/route"
)

// structure contains properties required for for macOS implementation
type osSpecificProperties struct {
	socket int
}

func (d *Detector) isRoutingChanged() (bool, error) {
	if d.interfaceToProtect == nil {
		log.Error("failed to check route change. Initial interface not defined")
		return false, nil
	}

	isDefaultRoute, err := netinfo.IsDefaultRoutingInterface(d.interfaceToProtect.Name)

	if err != nil {
		log.Error("Failed to check route change:", err)
		return false, err
	}

	return !isDefaultRoute, nil

}

func (d *Detector) doStart() {
	sock, err := syscall.Socket(syscall.AF_ROUTE, syscall.SOCK_RAW, syscall.AF_UNSPEC)
	if err != nil {
		log.Error("Failed to start route change detector:", err)
		return
	}
	d.props.socket = sock

	log.Info("Route change detector started")
	defer func() {
		log.Info("Route change detector stopped")
		d.doStop()
	}()

	// Loop waiting for messages.
	b := make([]byte, os.Getpagesize())
	for {
		nr, err := syscall.Read(d.props.socket, b)
		if err != nil {
			if d.props.socket == 0 {
				break // Manually stopped
			}
			if err == syscall.EINTR {
				continue // Interrupted by a signal, retry the read
			}
			log.Error("Route change detector (error on socket read):", err)
			return
		}

		messages, err := route.ParseRIB(0, b[:nr])
		if err != nil {
			continue
		}

		for _, msg := range messages {
			switch rmsg := msg.(type) {
			case *route.RouteMessage:
				switch rmsg.Type {
				case syscall.RTM_ADD, syscall.RTM_CHANGE, syscall.RTM_DELETE:

					if newGw := checkMsgIsDefaultIPv4RouteAdded(rmsg); newGw != nil {
						//log.Debug("----------- ======== DEFAULT ROUTE ADD  =========== ---------", newGw.String())
						var msg RouteChangeMessage
						msg.newDefaultGateway = newGw
						msg.interfaceLeakDetected, _ = d.isRoutingChanged()
						d.notifyRoutingChangeEx(msg)
					} else {
						d.notifyRoutingChangeWithDelay()
					}

				}
			}
		}
	}
}

func (d *Detector) doStop() {
	s := d.props.socket
	d.props.socket = 0
	if s != 0 {
		syscall.Close(s)
	}
}

// checkMsgIsDefaultIPv4RouteAdded - check if the message is about adding the default IPv4 route.
// Return new default IPv4 Gateway IP address, nil otherwise.
func checkMsgIsDefaultIPv4RouteAdded(rmsg *route.RouteMessage) net.IP {
	if rmsg == nil {
		return nil
	}
	// Ignore ifscope routes
	if rmsg.Flags&syscall.RTF_IFSCOPE != 0 {
		return nil
	}

	if rmsg.Type != syscall.RTM_ADD { // rmsg.Type != syscall.RTM_CHANGE
		return nil
	}

	const (
		RTAX_DST     = 0 // destination sockaddr present
		RTAX_GATEWAY = 1 // gateway sockaddr present
		RTAX_NETMASK = 2 // netmask sockaddr present
	)

	// Check DST address is 0.0.0.0
	if len(rmsg.Addrs) > RTAX_DST && rmsg.Addrs[RTAX_DST] != nil {
		if a, ok := rmsg.Addrs[RTAX_DST].(*route.Inet4Addr); !ok || !net.IP(a.IP[:]).Equal(net.IPv4zero) {
			return nil
		}
	} else {
		return nil
	}
	// Check Netmask is 0.0.0.0/0
	if len(rmsg.Addrs) > RTAX_NETMASK && rmsg.Addrs[RTAX_NETMASK] != nil {
		if a, ok := rmsg.Addrs[RTAX_NETMASK].(*route.Inet4Addr); !ok || !net.IP(a.IP[:]).Equal(net.IPv4zero) {
			return nil
		}
	} else {
		return nil
	}
	// Return Gateway IP address
	if len(rmsg.Addrs) > RTAX_GATEWAY && rmsg.Addrs[RTAX_GATEWAY] != nil {
		if a, ok := rmsg.Addrs[RTAX_GATEWAY].(*route.Inet4Addr); ok {
			return net.IP(a.IP[:])
		}
	}

	return nil
}
