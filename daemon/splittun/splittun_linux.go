//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2021 Privatus Limited.
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

// +build linux

package splittun

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/shell"
)

var (
	// error describing details if functionality not available
	funcNotAvailableError error
	stScriptPath          string
)

const stPidsFile = "/sys/fs/cgroup/net_cls/ivpn-exclude/cgroup.procs"

func implInitialize() error {
	funcNotAvailableError = nil

	stScriptPath = platform.SplitTunScript()
	if len(stScriptPath) <= 0 {
		funcNotAvailableError = fmt.Errorf("Split-Tunnelling script is not defined")
		return funcNotAvailableError
	}

	// check if ST functionality accessible
	outProcessFunc := func(text string, isError bool) {
		if isError {
			log.Error("Split Tunneling test: " + text)
		} else {
			log.Info("Split Tunneling test: " + text)
		}
	}
	err := shell.ExecAndProcessOutput(nil, outProcessFunc, "", stScriptPath, "test")
	if err != nil {
		funcNotAvailableError = err
	}

	// Ensure that ST is disable on daemon startup
	enable(false)

	return funcNotAvailableError
}

func implFuncNotAvailableError() error {
	return funcNotAvailableError
}

func implApplyConfig(isStEnabled bool, isVpnEnabled bool, addrConfig ConfigAddresses, splitTunnelApps []string) error {
	return enable(isStEnabled)
}

func implAddPid(pid int, commandToExecute string) error {
	enabled, err := isEnabled()
	if err != nil {
		return fmt.Errorf("unable to check Split Tunneling status")
	}
	if !enabled {
		return fmt.Errorf("the Split Tunneling not enabled")
	}

	return shell.Exec(nil, stScriptPath, "addpid", strconv.Itoa(pid))
}

func implGetRunningApps() ([]RunningApp, error) {
	// read all PIDs which are active in ST environment
	bytes, err := os.ReadFile(stPidsFile)
	if err != nil {
		return nil, err
	}

	pidStrings := strings.Split(string(bytes), "\n")

	ret := make([]RunningApp, 0, len(pidStrings))

	for _, s := range pidStrings {
		if len(s) <= 0 {
			continue
		}

		pid, err := strconv.Atoi(s)
		if err != nil {
			log.Warning(err)
			continue
		}

		// read cmdline for each pid
		cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			log.Warning(err)
			continue
		}
		for i, b := range cmdlineBytes {
			if b == 0 {
				cmdlineBytes[i] = ' '
			}
		}
		cmdline := string(cmdlineBytes)
		// TODO: do not forget update prefices in cese if CLI interface change
		cmdline = strings.TrimPrefix(cmdline, "ivpn exclude ")
		cmdline = strings.TrimPrefix(cmdline, "ivpn splittun -execute ")
		ret = append(ret, RunningApp{Pid: pid, Cmdline: cmdline})
	}

	return ret, nil
}

func isEnabled() (bool, error) {
	err := shell.Exec(nil, stScriptPath, "status")
	if err != nil {
		return false, nil
	}
	return true, nil
}

func enable(isEnable bool) error {

	if !isEnable {

		enabled, err := isEnabled()
		if err == nil && !enabled {
			return nil
		}
		err = shell.Exec(nil, stScriptPath, "stop")
		if err != nil {
			return fmt.Errorf("failed to disable Split Tunneling: %w", err)
		}
		log.Info("Split Tunneling disabled")
	} else {
		enabled, err := isEnabled()
		if err != nil {
			return fmt.Errorf("failed to enable Split Tunneling (unable to obtain ST status): %w", err)
		}

		if enabled {
			return nil
		}
		err = shell.Exec(nil, stScriptPath, "start")
		if err != nil {
			// if ST start failed - clean everything (by command 'stop')
			shell.Exec(nil, stScriptPath, "stop")

			return fmt.Errorf("failed to enable Split Tunneling: %w", err)
		}
		log.Info("Split Tunneling enabled")
	}
	return nil
}
