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
	"path"
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
// The function must return 'true' when the same command is already running.
// IMPORTANT: The result is NOT RELIABLE! In most standard cases, it works. But there is no guarantee that it will work on every Linux distributive with all commands!
func isAbleToAddAppToConfig(cmd string) (isAlreadyRunning bool, notAbleToRunError error) {
	cmd = strings.TrimSpace(cmd)
	if len(cmd) <= 0 {
		return false, fmt.Errorf("empty command")
	}

	// Here are implemented two machanisms for non-reliable detection of already started applications:
	// 1. Detection based on binary location (e.g. applicable for Ubuntu):
	//		1.1. Get full paths for all binaries in the command
	//		1.2. Get first 3 parent directories from the binary path
	//		1.3. Check is there any running processes from the path detected in previous step (`ps -aux`)
	//		    1.3.1. only for the paths  "/usr/..." , check filter is "/usr/*/<xxx>"
	//
	//		Example: command "google-chrome"
	//				1.1) binary (script) path: "/opt/google/chrome/google-chrome"
	//				1.2) first three parent directories: "/opt/google/chrome/"
	//				1.3) list of all running processes and CHECK if there any processes from "/opt/google/chrome/"
	//		Example: command "firefox"
	//				1.1) binary (script) path: "/usr/lib/firefox/firefox.sh"
	//				1.2) first three parent directories: "/usr/lib/firefox/"
	//				1.3) list of all running processes and CHECK if there any processes from "/usr/*/firefox/"
	//					1.3.1) because binary path is in "/usr/..." the filter is "/usr/*/firefox/"
	//
	// 2. Detection based on the list of opened GUI windows in the system and binary filename
	//		2.1. Get full paths for all binaries in the command
	//		2.2. Get binary file name
	//		2.3. Check is there any running GUI window in the system with the name same as binary file name (`xwininfo -root -children`)
	//
	//		Example: command "google-chrome"
	//				2.1) binary (script) path: "/opt/google/chrome/google-chrome"
	//				2.2) binary file name: "google-chrome"
	//				2.3) list of all running windows and CHECK if there any window has name "google-chrome"

	// Step 1.1 / 2.1 : Get full paths for all binaries in the command

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

	for _, arg := range strings.Split(cols[2], " ") {
		fpath := getBinaryOriginalLocation(arg)
		if len(fpath) > 0 {
			binPathsToCheck = append(binPathsToCheck, fpath)
		}
	}

	// Step 1.2 : first three parent directories
	grepParam := ""
	for _, path := range binPathsToCheck {
		dirs := strings.Split(path, "/")
		if len(dirs) < 4 {
			continue
		}
		if len(grepParam) > 0 {
			grepParam += `\|`
		}
				
		// Step 1.2 : first three parent directories
		//	* 1.3.1 : Only for the paths  "/usr/..." -> check filter is "/usr/*/<xxx>"
		if dirs[1] == "usr" {
			grepParam += `[ \t]/usr/[^/ ]\+/` + dirs[3] + "/"
		} else {
			grepParam += "[ \t]/" + dirs[1] + "/" + dirs[2] + "/" + dirs[3] + "/"
		}
	}

	// Step 1.3 : list of all running processes and CHECK if there any processes from the directories detected in previous step
	retIsAlreadyRunning := false
	if len(grepParam) > 0 {
		err := shell.Exec(nil, "bash", "-c", "ps -aux | grep '"+grepParam+"' | grep -v grep &>/dev/null")
		if err == nil {
			retIsAlreadyRunning = true
		}
	}
	
	if !retIsAlreadyRunning {
		// Step 2.2 : binary file name: "google-chrome"
		grepParam := ""
		for _, fpath := range binPathsToCheck {
			_, file := path.Split(fpath)
			file = strings.TrimSpace(strings.TrimSuffix(file, filepath.Ext(file)))

			if len(grepParam) > 0 {
				grepParam += `\|`
			}
			grepParam += "\"" + file + "\""
		}

		//Step 2.3 : Check is there any running GUI window in the system with the name same as binary file name (`xwininfo -root -children`)

		// xwininfo -root -children | grep --ignore-case '"google-chroMe"\|"Atom"\|("firefOx"'
		if len(grepParam) > 0 {
			err := shell.Exec(nil, "bash", "-c", "xwininfo -root -children | grep --ignore-case '"+grepParam+"' | grep -v grep &>/dev/null")
			if err == nil {
				retIsAlreadyRunning = true
			}
		}
	}

	return retIsAlreadyRunning, nil
}
