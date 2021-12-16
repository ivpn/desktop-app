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
	"path/filepath"
	"regexp"
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

// Information about added running process to the ST (by implAddPid())
// (map[<PID>]<command>)
var _addedRootProcesses map[int]string = map[int]string{}

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
	if pid <= 0 {
		return fmt.Errorf("PID is not defined")
	}
	log.Info(fmt.Sprintf("Adding PID:%d", pid))

	enabled, err := isEnabled()
	if err != nil {
		return fmt.Errorf("unable to check Split Tunneling status")
	}
	if !enabled {
		return fmt.Errorf("the Split Tunneling not enabled")
	}

	err = shell.Exec(nil, stScriptPath, "addpid", strconv.Itoa(pid))
	if err == nil {
		_addedRootProcesses[pid] = commandToExecute
	}
	return err
}

func implRemovePid(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("PID is not defined")
	}
	var retErr error

	// Remove PID and all it's child processes
	pids := make(map[int]struct{}, 0)
	pids[pid] = struct{}{}

	// looking for all childs
	if runningApps, err := implGetRunningApps(); err == nil {
		for _, app := range runningApps {
			if _, ok := pids[app.Ppid]; ok {
				pids[app.Pid] = struct{}{}
			}
		}
	} else {
		retErr = err
	}

	// remove all required pids
	for pidToRemove := range pids {
		log.Info(fmt.Sprintf("Removing PID:%d", pidToRemove))
		err := shell.Exec(nil, stScriptPath, "removepid", strconv.Itoa(pidToRemove))
		if err != nil && retErr == nil {
			retErr = err
		}
		if err == nil {
			delete(_addedRootProcesses, pidToRemove)
		}
	}
	return retErr
}

func implGetRunningApps() (allProcesses []RunningApp, err error) {
	// TODO: https://man7.org/linux/man-pages/man5/proc.5.html
	// '/proc/[pid]/stat' - Contains info about PPID

	// read all PIDs which are active in ST environment
	bytes, err := os.ReadFile(stPidsFile)
	if err != nil {
		return nil, err
	}

	pidStrings := strings.Split(string(bytes), "\n")

	retMapAll := make(map[int]RunningApp, len(pidStrings))

	regexpStat := regexp.MustCompile(`^([0-9]*) (\([\S ]*\)) \S ([0-9]+) ([0-9]+) ([0-9]+)`)

	for _, s := range pidStrings {
		if len(s) <= 0 {
			continue
		}

		pid, err := strconv.Atoi(s)
		if err != nil {
			log.Warning(err)
			continue
		}

		// read PPID, ProgessGroup, Session for each pid
		ppid := 0
		pgrp := 0
		psess := 0
		statBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			log.Warning(err)
		} else {
			statCols := regexpStat.FindStringSubmatch(string(statBytes))
			if len(statCols) >= 4 {
				if v, err := strconv.Atoi(statCols[3]); err == nil {
					ppid = v
				}
			}
			if len(statCols) >= 5 {
				if v, err := strconv.Atoi(statCols[4]); err == nil {
					pgrp = v
				}
			}
			if len(statCols) >= 6 {
				if v, err := strconv.Atoi(statCols[5]); err == nil {
					psess = v
				}
			}
		}

		// Read the actual pathname of the executed command
		exe := ""
		if sl, err := filepath.EvalSymlinks(fmt.Sprintf("/proc/%d/exe", pid)); err == nil && len(sl) > 0 {
			exe = sl
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
		// TODO: do not forget update prefices in case if CLI interface change
		cmdline = strings.TrimPrefix(cmdline, "/usr/bin/ivpn exclude ")
		cmdline = strings.TrimPrefix(cmdline, "/usr/bin/ivpn splittun -execute ")
		cmdline = strings.TrimPrefix(cmdline, "ivpn exclude ")
		cmdline = strings.TrimPrefix(cmdline, "ivpn splittun -execute ")
		cmdline = strings.TrimSpace(cmdline)
		retMapAll[pid] = RunningApp{
			Pid:     pid,
			Ppid:    ppid,
			Pgrp:    pgrp,
			Session: psess,
			Cmdline: cmdline,
			Exe:     exe}
	}

	// remove from _addedRootProcesses PIDs which are not exists anumore
	toRemoveRootPids := make([]int, 0)
	for p := range _addedRootProcesses {
		if _, ok := retMapAll[p]; !ok {
			toRemoveRootPids = append(toRemoveRootPids, p)
		}
	}
	for _, p := range toRemoveRootPids {
		delete(_addedRootProcesses, p)
	}

	// recursive anonymous function to find root PID
	var funcIsHasRoot func(ppid int) bool
	funcIsHasRoot = func(ppid int) bool {
		if _, ok := _addedRootProcesses[ppid]; ok {
			return true
		}
		if parentProc, ok := retMapAll[ppid]; ok {
			if ppid <= parentProc.Ppid {
				return false //just to ensure there is no infinite recursion
			}
			return funcIsHasRoot(parentProc.Ppid)
		}
		return false
	}

	// make result slice and mark required elements as 'ExtIsChild'
	retAll := make([]RunningApp, 0, len(retMapAll))
	for _, value := range retMapAll {
		value.ExtIsChild = funcIsHasRoot(value.Ppid)
		// for know root processes - replace command by the original command used to run process
		if cmdLine, ok := _addedRootProcesses[value.Pid]; ok {
			value.Cmdline = cmdLine
		}

		retAll = append(retAll, value)
	}

	return retAll, nil
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
