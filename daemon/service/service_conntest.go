//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 Privatus Limited.
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

package service

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/api/types"
)

func (s *Service) DetectAccessiblePorts(portsToTest []types.PortInfo) ([]types.PortInfo, error) {
	// TODO: use real IP address of echo-server
	const remoteEchoServerIP = ""

	if len(remoteEchoServerIP) == 0 {
		return []types.PortInfo{}, nil
	}

	allPorts := make(map[types.PortInfo]struct{})
	funcAddAllPorts := func(pts []types.PortInfo) {
		for _, p := range pts {
			if p.Port == 0 {
				continue
			}
			allPorts[p] = struct{}{}
		}
	}

	if len(portsToTest) > 0 {
		funcAddAllPorts(portsToTest)
	} else {
		svrs, err := s.ServersList()
		if err != nil {
			return nil, err
		}
		funcAddAllPorts(svrs.Config.Ports.WireGuard)
		funcAddAllPorts(svrs.Config.Ports.OpenVPN)
		allPorts[types.PortInfo{Port: svrs.Config.Ports.Obfs3.Port, Type: "TCP"}] = struct{}{}
		allPorts[types.PortInfo{Port: svrs.Config.Ports.Obfs4.Port, Type: "TCP"}] = struct{}{}
	}
	if len(allPorts) == 0 {
		return []types.PortInfo{}, nil
	}

	log.Info("Testing accessible ports...")

	const theTimeout = time.Millisecond * 500

	checkPort := func(p types.PortInfo) error {
		servAddr := fmt.Sprintf("%s:%d", remoteEchoServerIP, p.Port)

		if p.IsUDP() {
			conn, err := net.Dial("udp", servAddr)
			if err != nil {
				return err
			}
			defer conn.Close()

			conn.SetDeadline(time.Now().Add(theTimeout))
			if _, err := conn.Write([]byte("Hi!")); err != nil {
				return err
			}

			buff := make([]byte, 64)
			if _, err := conn.Read(buff); err != nil {
				return err
			}
		} else {
			conn, err := net.DialTimeout("tcp", servAddr, theTimeout)
			if err != nil {
				return err // seems, port 'p' is not accessible
			}
			defer conn.Close()
		}

		return nil // port is accessible
	}

	accessiblePortsLocker := sync.Mutex{}
	accessiblePorts := make([]types.PortInfo, 0, len(allPorts))

	limitChan := make(chan struct{}, 30) // limit max simulteneous requests

	wg := sync.WaitGroup{}
	wg.Add(len(allPorts))
	for k, _ := range allPorts {

		limitChan <- struct{}{}

		go func(p types.PortInfo) {
			defer func() {
				<-limitChan
				wg.Done()
			}()
			if err := checkPort(p); err != nil {
				// port not accessible
				// log.Error(fmt.Sprintf("Port %s access error: %s", p.String(), err))
				return
			}

			accessiblePortsLocker.Lock()
			accessiblePorts = append(accessiblePorts, p)
			accessiblePortsLocker.Unlock()
		}(k)
	}

	wg.Wait()

	log.Info(fmt.Sprintf("Testing accessible ports done: %d of %d is OK", len(accessiblePorts), len(allPorts)))
	return accessiblePorts, nil
}
