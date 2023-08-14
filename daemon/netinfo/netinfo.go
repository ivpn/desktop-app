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
	"errors"
	"fmt"
	"net"

	"github.com/ivpn/desktop-app/daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("netinf")
}

// DefaultGatewayIP - returns: default gatewat IP
func DefaultGatewayIP() (defGatewayIP net.IP, err error) {
	// method should be implemented in platform-specific file
	return doDefaultGatewayIP()
}

func GetOutboundIP(isIPv6 bool) (net.IP, error) {
	if isIPv6 {
		return GetOutboundIPEx(net.ParseIP("2a00:1450:400d:80a::200e"))
	}
	return GetOutboundIPEx(net.ParseIP("8.8.8.8"))
}

func GetOutboundIPEx(addr net.IP) (net.IP, error) {
	addrStr := ""
	if addr.To4() != nil {
		// IPv4
		addrStr = addr.String() + ":80"
	} else {
		// IPv6
		addrStr = "[" + addr.String() + "]:80"
	}

	conn, err := net.Dial("udp", addrStr)
	if err != nil {
		return net.IP{}, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

// InterfaceByIPAddr - Get network interface object by it's local IP address
func InterfaceByIPAddr(localAddr net.IP) (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, ifs := range ifaces {

		addrs, _ := ifs.Addrs()
		if addrs == nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil {
				if ip.Equal(localAddr) {
					return &ifs, nil
				}
			}
		}
	}
	return nil, errors.New("not found network interface with address:" + localAddr.String())
}

// GetFreePort - get unused local port
// Note there is no guarantee that port will not be in use right after finding it
func GetFreePort(isTCP bool) (int, error) {
	if isTCP {
		return GetFreeTCPPort()
	}
	return GetFreeUDPPort()
}

// GetFreeTCPPort - get unused TCP local port
// Note there is no guarantee that port will not be in use right after finding it
func GetFreeTCPPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to resolve TCP address: %w", err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("failed to obtain free local TCP port: %w", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// GetFreeUDPPort - get unused UDP local port
// Note there is no guarantee that port will not be in use right after finding it
func GetFreeUDPPort() (int, error) {
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to resolve TCP address: %w", err)
	}

	l, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, fmt.Errorf("failed to obtain free local UDP port: %w", err)
	}
	defer l.Close()
	return l.LocalAddr().(*net.UDPAddr).Port, nil
}

// GetInterfaceByIndex - get interface info by its index
func GetInterfaceByIndex(index int) (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, ifs := range ifaces {
		if ifs.Index == index {
			return &ifs, nil
		}
	}
	return nil, nil
}

// GetAllLocalV4Addresses - returns IPv4 addresses of all local interfaces
func GetAllLocalV4Addresses() ([]net.IPNet, error) {
	return getAllLocalAddresses(nil, false)
}

// GetAllLocalV6Addresses - returns IPv6 addresses of all local interfaces
func GetAllLocalV6Addresses() ([]net.IPNet, error) {
	return getAllLocalAddresses(nil, true)
}

/*
// GetInterfaceV4Addresses - returns IPv4 addresses of the local interface
func GetInterfaceV4Addresses(inf net.Interface) ([]net.IPNet, error) {
	return getAllLocalAddresses([]net.Interface{inf}, false)
}

// GetInterfaceV6Addresses - returns IPv6 addresses of the local interfaces
func GetInterfaceV6Addresses(inf net.Interface) ([]net.IPNet, error) {
	return getAllLocalAddresses([]net.Interface{inf}, true)
}*/

func getAllLocalAddresses(ifaces []net.Interface, isIPv6 bool) ([]net.IPNet, error) {
	ret := make([]net.IPNet, 0, 8)

	var err error
	if len(ifaces) == 0 {
		ifaces, err = net.Interfaces()
		if err != nil {
			return ret, fmt.Errorf("failed to get network interfaces: %w", err)
		}
	}

	for _, ifs := range ifaces {
		if ifs.Flags&net.FlagUp == 0 {
			continue
		}

		//if (ifs.Flags & net.FlagLoopback) > 0 {
		//	continue
		//}

		addrs, _ := ifs.Addrs()
		if addrs == nil {
			continue
		}

		for _, addr := range addrs {
			var ip *net.IPNet

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v
			}

			if ip == nil {
				continue
			}

			isIPv6Addr := ip.IP.To4() == nil // check address is IPv6
			if isIPv6Addr != isIPv6 {
				continue // ignore unexpected IP address family (v4 or v6)
			}

			ret = append(ret, *ip)
		}
	}
	return ret, nil
}
