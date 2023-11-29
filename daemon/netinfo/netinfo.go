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

var (
	// Define the non-routable address ranges
	_localNonRoutableRanges []net.IPNet = []net.IPNet{
		{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},     // IPv4 private range (RFC 1918)
		{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},  // IPv4 private range (RFC 1918)
		{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)}, // IPv4 private range (RFC 1918)
		{IP: net.ParseIP("169.254.0.0"), Mask: net.CIDRMask(16, 32)}, // IPv4 auto-IP range (RFC 3927)
		{IP: net.ParseIP("fc00::"), Mask: net.CIDRMask(7, 128)},      // IPv6 Unique Local Address (ULA) (RFC 4193)
		{IP: net.ParseIP("fe80::"), Mask: net.CIDRMask(10, 128)},     // IPv6 Link-Local Address (RFC 4291)
	}

	_multicastAddresses []net.IPNet = []net.IPNet{
		{IP: net.ParseIP("224.0.0.0"), Mask: net.CIDRMask(4, 32)}, //IPv4 Multicast Addresses (RFC 5771)
		{IP: net.ParseIP("ff00::"), Mask: net.CIDRMask(8, 128)},   //IPv6 Multicast Addresses (RFC 4291; RFC 3306)
	}
)

// IsLocalNonRoutableIP checks if an 'ip' address is within one of the local non-routable ranges.
func IsLocalNonRoutableIP(ip net.IP) bool {
	for _, localRange := range _localNonRoutableRanges {
		if localRange.Contains(ip) {
			return true
		}
	}
	return false
}

// GetNonRoutableLocalAddrRanges retrieves all the non-routable address ranges.
func GetNonRoutableLocalAddrRanges() []net.IPNet {
	return append([]net.IPNet{}, _localNonRoutableRanges...)
}

// GetMulticastAddresses retrieves all the multicast address ranges.
func GetMulticastAddresses() []net.IPNet {
	return append([]net.IPNet{}, _multicastAddresses...)
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

func GetLoopbackInterface(isIpv6 bool) (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, ifs := range ifaces {
		// check only loopback interfaces
		if ifs.Flags&net.FlagLoopback == 0 || ifs.Flags&net.FlagUp == 0 {
			continue
		}

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

			// loopback interface must have required IP address type (IPv6/IPv4)
			if ip != nil {
				ipv6 := ip.To4() == nil
				if ipv6 == isIpv6 {
					return &ifs, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("not found loopback network interface (ipv6=%v)", isIpv6)
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
// Note: it returns only non-routable local addresses!
func GetAllLocalV4Addresses() ([]net.IPNet, error) {
	return getAllLocalAddresses(nil, false)
}

// GetAllLocalV6Addresses - returns IPv6 addresses of all local interfaces
// Note: it returns only non-routable local addresses!
func GetAllLocalV6Addresses() ([]net.IPNet, error) {
	return getAllLocalAddresses(nil, true)
}

// getAllLocalAddresses - returns all local addresses of all available interfaces.
//
// Note: it returns only non-routable local addresses!
// (this prevents potential vulnerabilities when attacker can manipulate with routing table)
//
// * ifaces - list of interfaces to check, if nil then all available interfaces will be checked;
// * if isIPv6 is true, then IPv6 addresses will be returned, otherwise IPv4;
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

			if ip == nil || ip.IP.IsUnspecified() {
				continue
			}

			// Ensure that IP is local NON-routable address
			// This prevents potential vulnerability when attacker can manipulate with routing table
			if !IsLocalNonRoutableIP(ip.IP) {
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
