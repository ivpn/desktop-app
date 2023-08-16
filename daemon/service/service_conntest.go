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

package service

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/api/types"
)

type connTest struct {
	locker       sync.Mutex
	lockerResult sync.Mutex
	result       *connTestResult
}

type connTestResult struct {
	tested    []types.PortInfo
	result    []types.PortInfo // accessible ports list
	resultErr error
}

// DetectAccessiblePorts runs test to detect accessible ports
func (s *Service) DetectAccessiblePorts(portsToTest []types.PortInfo) ([]types.PortInfo, error) {
	ct := &s._connectionTest

	ct.lockerResult.Lock()
	// If 'ct.result' not null - it means the previous test still running.
	// The 'ct.result.tested' contains the info about ports currently in test
	runningTest := ct.result
	ct.lockerResult.Unlock()

	{
		// Limiting to run 'doDetectAccessiblePorts()':
		// do not allow to run multiple calls of function in parallel
		ct.locker.Lock()
		defer ct.locker.Unlock()

		// If there were previously running test - check if the tested ports are the same as we need;
		// if so - do not perform new round of test and take this results.
		if runningTest != nil {
			if isPortsEquals(portsToTest, runningTest.tested) {
				return runningTest.result, runningTest.resultErr
			}
		}

		// mark "the test is running"
		ct.lockerResult.Lock()
		ct.result = &connTestResult{tested: portsToTest}
		ct.lockerResult.Unlock()

		// test ports
		r, err := s.doDetectAccessiblePorts(portsToTest)

		// save results ('waiting' routines will access this info over 'runningTest')
		ct.result.result, ct.result.resultErr = r, err
		// mark: "no tests currently running"
		ct.result = nil

		return r, err
	}
}

func (s *Service) doDetectAccessiblePorts(portsToTest []types.PortInfo) ([]types.PortInfo, error) {
	serversInfo, err := s.ServersList()
	if err != nil {
		return []types.PortInfo{}, err
	}
	if len(serversInfo.Config.Ports.Test) == 0 {
		return []types.PortInfo{}, nil
	}

	remoteEchoServerIP := serversInfo.Config.Ports.Test[0].EchoServer // use first echo-server

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
		allPorts[types.PortInfo{PortInfoBase: types.PortInfoBase{Port: svrs.Config.Ports.Obfs3.Port, Type: "TCP"}}] = struct{}{}
		allPorts[types.PortInfo{PortInfoBase: types.PortInfoBase{Port: svrs.Config.Ports.Obfs4.Port, Type: "TCP"}}] = struct{}{}
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
	for k := range allPorts {

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

// Function to test unordered slices of 'types.PortInfo' on equality
func isPortsEquals(a []types.PortInfo, b []types.PortInfo) bool {
	if len(a) != len(b) {
		return false
	}
	hashedA := make(map[types.PortInfo]struct{})
	for _, p := range a {
		hashedA[p] = struct{}{}
	}
	for _, p := range b {
		if _, ok := hashedA[p]; !ok {
			return false
		}
	}
	return true
}
