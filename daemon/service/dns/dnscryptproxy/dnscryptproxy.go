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

package dnscryptproxy

import (
	"fmt"
	"net"

	"github.com/ivpn/desktop-app/daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("dnscrt")
}

type DnsCryptProxy struct {
	binaryPath     string
	configFilePath string
	listenAddr     net.IP      // address where dnscrypt-proxy will listen
	extra          extraParams // platform-specific extra params
}

var hookInitLoopbackIP func(loopbackIP net.IP) error

// Start - create and start dnscrypt-proxy instance asynchronously
func Start(binaryPath, configFile string, listenAddr net.IP) (obj *DnsCryptProxy, retErr error) {
	p := &DnsCryptProxy{
		binaryPath:     binaryPath,
		configFilePath: configFile,
		listenAddr:     listenAddr,
	}

	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("error starting dnscrypt-proxy: %w", retErr)
			p.implStop()
		}
	}()

	return p, p.implStart()
}
func (p *DnsCryptProxy) Stop() (err error) {
	if p == nil {
		return nil
	}

	log.Info("Stopping dnscrypt-proxy")
	defer func() {
		if err != nil {
			err = fmt.Errorf("error stopping dnscrypt-proxy: %w", err)
		}
	}()

	ret := p.implStop()
	if ret == nil {
		p = nil
	}
	return ret
}

// getFreeLocalAddressForDNS searches for a free TCP/UDP port pairs on loopback addresses
// in the 127.0.0.x range, starting from the specified address.
// It returns the first found local IP address where both TCP:53 and UDP:53 ports are available.
//
// Note: This function has a potential race condition. Since it only checks for port
// availability and doesn't hold the port, another process could bind to the address
// in the interval between this check and its use. This is considered an acceptable
// risk for its intended purpose.
//
// Parameters:
//   - startAddress: IPv4 address in 127.0.0.x range to start searching from (nil defaults to 127.0.0.1)
//   - desiredPort: Port number to check for availability (typically 53 for DNS)
//
// Returns:
//   - net.IP: Local IP address with the 53 port available on both TCP and UDP
//   - error: Error if no free port found, invalid parameters, or other failure occurs
func GetFreeLocalAddressForDNS(startAddress net.IP) (net.IP, error) {
	// Set default start address if not provided
	if startAddress == nil || startAddress.To4() == nil {
		startAddress = net.IPv4(127, 0, 0, 1)
	}

	start := startAddress.To4() // convert IP address 4-byte form

	const maxLastByte = 16 // limit search to 127.0.0.1 - 127.0.0.16
	if start[0] != 127 || start[1] != 0 || start[2] != 0 || start[3] < 1 || start[3] > maxLastByte {
		return nil, fmt.Errorf("start address must be in 127.0.0.1-127.0.0.%d range", maxLastByte)
	}

	for i := start[3]; i <= maxLastByte; i++ {
		ip := net.IPv4(127, 0, 0, i)

		if hookInitLoopbackIP != nil {
			if err := hookInitLoopbackIP(ip); err != nil {
				log.Warning("Failed to init loopback IP ", ip.String(), ": ", err.Error())
			}
		}

		// Check UDP port availability
		udpAddr := &net.UDPAddr{IP: ip, Port: 53}
		udpConn, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			continue // UDP port not available on this IP
		}
		udpConn.Close()

		// Check TCP port availability
		tcpAddr := &net.TCPAddr{IP: ip, Port: 5}
		tcpConn, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			continue // TCP port not available on this IP
		}
		tcpConn.Close()

		return ip, nil // found free port on both UDP and TCP for this IP
	}
	return nil, fmt.Errorf("no free local addresses found for port 53 in 127.0.0.1-127.0.0.%d range", maxLastByte)
}
