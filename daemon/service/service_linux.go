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
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ivpn/desktop-app/daemon/service/firewall"
	"github.com/ivpn/desktop-app/daemon/shell"
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

func (s *Service) implSplitTunnelling_AddApp(execCmd string) (requiredCmdToExec string, isAlreadyRunning bool, err error) {
	if !s._preferences.IsSplitTunnel {
		return "", false, fmt.Errorf("unable to run application in Split Tunneling environment: Split Tunneling is disabled")
	}
	execCmd = strings.TrimSpace(execCmd)
	if len(execCmd) <= 0 {
		return "", false, nil
	}

	isRunning, err := isAbleToAddAppToConfig(execCmd)
	if err != nil {
		return "", isRunning, err
	}

	return fmt.Sprintf("ivpn splittun -execute %s", execCmd), isRunning, nil
}

func (s *Service) implSplitTunnelling_RemoveApp(pid int, binaryPath string) (err error) {
	return splittun.RemovePid(pid)
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

// Function: Check if the same process already running
// IMPORTANT: The result is NOT RELIABLE!
// The function must return 'true' when the same binary is already launched.
// In most standard cases, it works. But there is no guarantee that it will work everywhere with all commands!
func isAbleToAddAppToConfig(cmd string) (isAlreadyRunning bool, notAbleToRunError error) {
	cmd = strings.TrimSpace(cmd)
	if len(cmd) <= 0 {
		return false, fmt.Errorf("empty command")
	}

	// Function is trying to get the real path to binary
	getBinaryOriginalLocation := func(bin string) string {
		binPath, err := exec.LookPath(bin)
		if err != nil {
			return ""
		}
		realpath, err := filepath.EvalSymlinks(binPath)
		if err != nil {
			return ""
		}
		return realpath
	}

	// list of binaries to check
	binPathsToCheck := make([]string, 0, 2)

	// get app absolute path and arguments
	binaryArgsRegexp := regexp.MustCompile("(\".*\"|\\S*)(.*)")
	cols := binaryArgsRegexp.FindStringSubmatch(cmd)
	if len(cols) != 3 {
		return false, fmt.Errorf("failed to parse command")
	}

	execBin := strings.Trim(cols[1], "\"")
	binPath, err := exec.LookPath(execBin)
	if err != nil {
		// do not allow to run app if no binary found
		return false, err
	}
	fpath, err := filepath.EvalSymlinks(binPath)
	if err == nil {
		binPathsToCheck = append(binPathsToCheck, fpath)
	}

	for _, arg := range strings.Split(cols[2], " ") {
		fpath := getBinaryOriginalLocation(arg)
		if len(fpath) > 0 {
			binPathsToCheck = append(binPathsToCheck, fpath)
		}
	}

	grepParam := ""
	for _, path := range binPathsToCheck {
		dirs := strings.Split(path, "/")
		if len(dirs) < 4 {
			continue
		}
		if len(grepParam) > 0 {
			grepParam += `\|`
		}
		grepParam += "/" + dirs[1] + "/" + dirs[2] + "/" + dirs[3]
	}
	retIsAlreadyRunning := false
	if len(grepParam) > 0 {
		err := shell.Exec(nil, "/usr/bin/bash", "-c", "/usr/bin/ps -aux | /usr/bin/grep '"+grepParam+"' | /usr/bin/grep -v /usr/bin/grep &>/dev/null")
		if err == nil {
			retIsAlreadyRunning = true
		}
	}

	return retIsAlreadyRunning, nil
}
