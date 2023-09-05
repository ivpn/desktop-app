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

//go:build linux
// +build linux

package service

import (
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	protocolTypes "github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/firewall"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	"github.com/ivpn/desktop-app/daemon/shell"
	"github.com/ivpn/desktop-app/daemon/splittun"
)

func (s *Service) implIsCanApplyUserPreferences(userPrefs preferences.UserPreferences) error {
	if s.Connected() {
		return fmt.Errorf("unable to change settings in the connected state")
	}
	if userPrefs.Linux.IsDnsMgmtOldStyle {
		disabledFuncs := s.GetDisabledFunctions()
		dnsMgmtOldErr := disabledFuncs.Platform.Linux.DnsMgmtOldResolvconfError
		if len(dnsMgmtOldErr) > 0 {
			return fmt.Errorf("the old-style DNS management is not applicable to the current environment: %s", dnsMgmtOldErr)
		}
	}
	return nil
}

func (s *Service) implGetDisabledFuncForPlatform() protocolTypes.DisabledFunctionalityForPlatform {
	var linuxFuncs protocolTypes.DisabledFunctionalityLinux

	if len(platform.ResolvectlBinPath()) <= 0 {
		linuxFuncs.DnsMgmtNewResolvectlError = "the 'resolvectl' is not applicable or missing"
	}
	if envs := platform.GetSnapEnvs(); envs != nil {
		linuxFuncs.DnsMgmtOldResolvconfError = "it is not allowed to modify 'resolv.conf' from the snap environment"
	}

	return protocolTypes.DisabledFunctionalityForPlatform{Linux: linuxFuncs}
}

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
		return "", false, fmt.Errorf("unable to run application in Split Tunnel environment: Split Tunnel is disabled")
	}
	execCmd = strings.TrimSpace(execCmd)
	if len(execCmd) <= 0 {
		return "", false, nil
	}

	isRunning, err := isAbleToAddAppToConfig(execCmd)
	if err != nil {
		return "", isRunning, err
	}

	// ensure ST is initialized
	err = s.splitTunnelling_ApplyConfig()
	if err != nil {
		return "", false, err
	}

	return fmt.Sprintf("ivpn exclude %s", execCmd), isRunning, nil
}

func (s *Service) implSplitTunnelling_RemoveApp(pid int, binaryPath string) (err error) {
	return splittun.RemovePid(pid)
}

// Inform the daemon about started process in ST environment
// Parameters:
// pid 			- process PID
// exec 		- Command executed in ST environment (e.g. binary + arguments)
//
//	(identical to SplitTunnelAddApp.Exec and SplitTunnelAddAppCmdResp.Exec)
//
// cmdToExecute - Shell command used to perform this operation
func (s *Service) implSplitTunnelling_AddedPidInfo(pid int, exec string, cmdToExecute string) error {
	return splittun.AddPid(pid, exec)
}

func (s *Service) implGetDiagnosticExtraInfo() (string, error) {
	ifconfig := s.diagnosticGetCommandOutput("ifconfig")
	netstat := s.diagnosticGetCommandOutput("netstat", "-nr", "--protocol", "inet,inet6")
	resolvectl := s.diagnosticGetCommandOutput("resolvectl", "status")
	resolvconf := s.diagnosticGetCommandOutput("cat", "/etc/resolv.conf")

	return fmt.Sprintf("%s\n%s\n%s\n%s", ifconfig, netstat, resolvectl, resolvconf), nil
}

// Function: Check if the same process already running
// The function must return 'true' when the same command is already running.
// IMPORTANT: The result is NOT RELIABLE! In most standard cases, it works. But there is no guarantee that it will work on every Linux distributive with all commands!
func isAbleToAddAppToConfig(cmd string) (isAlreadyRunning bool, notAbleToRunError error) {
	cmd = strings.TrimSpace(cmd)
	if len(cmd) <= 0 {
		return false, fmt.Errorf("empty command")
	}

	// 1. Detection based on binary location (e.g. applicable for Ubuntu):
	//		1.1. Get full paths for all binaries in the command
	//		1.2. Find running processes by mask (`ps -aux`):
	//			1.2.1.	mask: full path to binary
	//					Example: file "/usr/bin/atom" mask "/usr/bin/atom"
	//			1.2.2.	mask: (if binary starts from "/opt/") "/opt/<dir2>/<dir3>/"
	//					Example: file "/opt/google/chrome/google-chrome" mask "/opt/google/chrome/"
	//			1.2.3.	mask: (if binary starts from "/usr/") "/usr/<anything>/<filename>"
	//					Example: file "/usr/bin/firefox" mask "/usr/*/firefox"
	//			1.2.4.	mask: (if symlink to a binary starts from "/snap/") " /snap/<filename>/"
	//					Example: file "/snap/bin/git-cola" mask " /snap/git-cola/"

	// Step 1.1 sssss: Get full paths for all binaries in the command

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
		binPathsToCheck = append(binPathsToCheck, strings.TrimSuffix(fpath, filepath.Ext(fpath)))
	}

	// Function is trying to get the real path to binary
	getBinaryOriginalLocation := func(bin string) string {
		binPath, err := exec.LookPath(bin)
		if err != nil {
			return ""
		}
		if strings.HasPrefix(binPath, "/snap/") {
			return binPath
		}
		realpath, err := filepath.EvalSymlinks(binPath)
		if err != nil {
			return ""
		}
		return realpath
	}

	for _, arg := range strings.Split(cols[2], " ") {
		if len(arg) <= 0 {
			continue
		}
		fpath := getBinaryOriginalLocation(arg)
		if len(fpath) > 0 {
			binPathsToCheck = append(binPathsToCheck, strings.TrimSuffix(fpath, filepath.Ext(fpath)))
		}
	}

	// prepare search masks
	regexParam := ""
	for _, path := range binPathsToCheck {
		if len(regexParam) > 0 {
			regexParam += `|`
		}
		//	1.2.1.	mask: full path to binary
		//	Example: file "/usr/bin/atom" mask "/usr/bin/atom"
		regexParam += `(\s` + path + `\s)`

		dirs := strings.Split(path, "/")

		if len(dirs) >= 4 {
			if strings.HasPrefix(path, "/opt/") {
				//	1.2.2.	mask: (if binary starts from "/opt/") "/opt/<dir2>/<dir3>/"
				//	Example: file "/opt/google/chrome/google-chrome" mask " /opt/google/chrome/"
				regexParam += `|(\s/opt/` + dirs[2] + "/" + dirs[3] + "/)"
			} else if strings.HasPrefix(path, "/usr/") {
				//	1.2.3.	mask: (if binary starts from "/usr/") " /usr/<anything>/<filename> "
				//	Example: file "/usr/bin/firefox" mask " /usr/*/firefox "
				regexParam += `|(\s/usr/[^ ]+/` + dirs[len(dirs)-1] + `(\s|$))`
			} else if strings.HasPrefix(path, "/snap/") {
				//	1.2.4.	mask: (if symlink to a binary starts from "/snap/") " /snap/<filename>/"
				//	Example: file "/snap/bin/git-cola" mask " /snap/git-cola/"
				regexParam += `|(\s/snap/` + dirs[len(dirs)-1] + `/)`
			}
		}
	}

	//	Step 1.2 : Find running processes by mask (`ps -aux`):
	retIsAlreadyRunning := false
	if len(regexParam) > 0 {
		outRegexp := regexp.MustCompile(regexParam)
		outProcessFunc := func(text string, isError bool) {
			if isError || retIsAlreadyRunning {
				return
			}
			found := outRegexp.FindString(text)
			if len(found) > 0 {
				retIsAlreadyRunning = true
				// log.Debug("(running app detection: looks like the application is already started) found: ", found)
			}
		}

		err := shell.ExecAndProcessOutput(nil, outProcessFunc, "", "ps", "-aux")
		if err != nil {
			log.Debug("(running app detection ERROR): ", err)
		}
	}

	/*
		// INFO: running commands by the daemon (as a service) 'xlsclients' or 'xwininfo' failing
		// Therefore we do not use this mechanism
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
		if !retIsAlreadyRunning {
			// Step 2.2 : binary file name: like "google-chrome"

			regexParam := ""
			for _, fpath := range binPathsToCheck {
				_, file := path.Split(fpath)
				file = strings.TrimSpace(strings.TrimSuffix(file, filepath.Ext(file)))

				if len(regexParam) > 0 {
					regexParam += `|`
				}
				regexParam += `\s` + file + `(\s|$)`
			}


			// Step 2.3 : Check is there any running GUI window in the system with the name same as binary file name (`xlsclients -a`)
			// xwininfo -root -children | grep --ignore-case '"google-chroMe"\|"Atom"\|("firefOx"'
			// xlsclients -a
			if len(regexParam) > 0 {
				outRegexp := regexp.MustCompile(regexParam)
				outProcessFunc := func(text string, isError bool) {
					if isError || retIsAlreadyRunning{
						return
					}
					log.Debug(isError, " (xlsclients -a):", text)
					found  := outRegexp.FindString(text)
					if len(found) > 0 {
						retIsAlreadyRunning = true
						log.Debug("**** FOUND! *****", found)
					}
				}

				log.Debug("REGEXP (xlsclients -a): ", "'"+regexParam+"'")
				err := shell.ExecAndProcessOutput(nil, outProcessFunc, "", "xlsclients", "-a")
				if err != nil {
					log.Debug("EXEC ERROR (xlsclients -a): ", err)
				}
			}
		}
	*/

	return retIsAlreadyRunning, nil
}
