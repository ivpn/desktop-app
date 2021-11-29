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

package splittun

import (
	"fmt"

	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/shell"
)

var logDebug *logger.Logger

var (
	// error describing details if functionality not available
	funcNotAvailableError error
	stScriptPath          string
)

func implInitialize() error {
	logDebug = logger.NewLogger("stdbg")

	stScriptPath = platform.SplitTunScript()
	if len(stScriptPath) <= 0 {
		funcNotAvailableError = fmt.Errorf("Split-Tunnelling script is not defined")
		return funcNotAvailableError
	}

	return nil
}

func implFuncNotAvailableError() error {
	return funcNotAvailableError
}

func implApplyConfig(isStEnabled bool, isVpnEnabled bool, addrConfig ConfigAddresses, splitTunnelApps []string) error {
	if !isStEnabled {
		return enable(false)
	}
	return nil
}

func implRunCmdInSplittunEnvironment(commandToExecute, osUser string) error {
	err := enable(true)
	if err != nil {
		return err
	}
	if len(osUser) <= 0 {
		return fmt.Errorf("username not defined")
	}
	if len(commandToExecute) <= 0 {
		return fmt.Errorf("command not defined")
	}

	go runCommand(commandToExecute, osUser)
	return nil
}

func isEnabled() (bool, error) {
	err := shell.Exec(logDebug, stScriptPath, "status")

	if err != nil {
		exitCode, err := shell.GetCmdExitCode(err)
		if err != nil {
			return false, fmt.Errorf("failed to get Cmd exit code: %w", err)
		}
		if exitCode == 0 {
			return true, nil
		}
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
		err = shell.Exec(logDebug, stScriptPath, "stop")
		if err != nil {
			return fmt.Errorf("failed to disable Split Tunneling: %w", err)
		}
	} else {
		enabled, err := isEnabled()
		if err != nil {
			return fmt.Errorf("failed to enable Split Tunneling (unable to obtain ST status): %w", err)
		}

		if enabled {
			return nil
		}
		err = shell.Exec(logDebug, stScriptPath, "start")
		if err != nil {
			return fmt.Errorf("failed to enable Split Tunneling: %w", err)
		}
	}
	return nil
}

func runCommand(command, user string) {
	log.Info(fmt.Sprintf("Starting command '%s' (user '%s')", command, user[0:1]+"***"))
	defer log.Info(fmt.Sprintf("Stopped command '%s' (user '%s')", command, user[0:1]+"***"))

	outProcessFunc := func(text string, isError bool) {
		if isError {
			logDebug.Info("CMD (error)>>", text)
		} else {
			logDebug.Info("CMD >>", text)
		}
	}
	err := shell.ExecAndProcessOutput(logDebug, outProcessFunc, "", stScriptPath, "run", "-u", user, command)
	//err := shell.Exec(logDebug, stScriptPath, "run", "-u", user, command)
	if err != nil {
		log.Error(fmt.Errorf("fcommand execution error: %w", err))
	}
}
