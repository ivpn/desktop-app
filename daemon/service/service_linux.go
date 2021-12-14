//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
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

package service

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ivpn/desktop-app/daemon/service/firewall"
	"github.com/ivpn/desktop-app/daemon/splittun"
)

func (s *Service) implPingServersStarting(hosts []net.IP) error {
	const onlyForICMP = true
	const isPersistent = false
	return firewall.AddHostsToExceptions(hosts, onlyForICMP, isPersistent)
}
func (s *Service) implPingServersStopped(hosts []net.IP) error {
	const onlyForICMP = true
	const isPersistent = false
	return firewall.RemoveHostsFromExceptions(hosts, onlyForICMP, isPersistent)
}

func (s *Service) implSplitTunnelling_AddApp(execCmd string) (requiredCmdToExec string, err error) {
	if !s._preferences.IsSplitTunnel {
		return "", fmt.Errorf("unable to run application in Split Tunneling environment: Split Tunneling is disabled")
	}
	execCmd = strings.TrimSpace(execCmd)
	if len(execCmd) <= 0 {
		return "", nil
	}

	isAbleToAddAppToConfig := func(app string) error {
		binaryArgsRegexp := regexp.MustCompile("(\".*\"|\\S*)(.*)")
		// get app absolute path and arguments
		cols := binaryArgsRegexp.FindStringSubmatch(app)
		if len(cols) != 3 {
			return fmt.Errorf("failed to parse command")
		}

		execBin := strings.Trim(cols[1], "\"")
		_, err := exec.LookPath(execBin)

		return err
	}

	if err := isAbleToAddAppToConfig(execCmd); err != nil {
		return "", err
	}

	return fmt.Sprintf("ivpn splittun -execute %s", execCmd), nil
}

func (s *Service) implSplitTunnelling_RemoveApp(pid int, binaryPath string) (err error) {

	// TODO: not implemented yet
	return fmt.Errorf("not implemented")
}

// Inform the daemon about started process in ST environment
// Parameters:
// pid 			- process PID
// exec 		- Command executed in ST environment (e.g. binary + arguments)
// 				  (identical to SplitTunnelAddApp.Exec and SplitTunnelAddAppCmdResp.Exec)
// cmdToExecute - Shell command used to perform this operation
func (s *Service) implSplitTunnelling_AddedPidInfo(pid int, exec string, cmdToExecute string) error {
	return splittun.AddPid(pid, exec)
}
