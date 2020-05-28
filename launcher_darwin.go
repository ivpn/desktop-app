//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
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

package main

import (
	"os"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/oshelpers/macos/libivpn"
	"github.com/ivpn/desktop-app-daemon/shell"
)

// Prepare to start IVPN daemon for macOS
func doPrepareToRun() error {
	// create symlink to 'ivpn' cli client

	linkpath := "/usr/local/bin/ivpn"
	if _, err := os.Stat(linkpath); err != nil {
		// FIXME: we always getting error (even if symlink is exists)
		if os.IsNotExist(err) {
			log.Info("Creating symlink to IVPN CLI: ", linkpath)
			err := shell.Exec(log, "ln", "-fs", "/Applications/IVPN.app/Contents/MacOS/cli/ivpn", linkpath)
			if err != nil {
				log.Error("Failed to create symlink to IVPN CLI: ", err)
			}
		}
	}

	return nil
}

// inform OS-specific implementation about listener port
func doStartedOnPort(openedPort int, secret uint64) {
	libivpn.StartXpcListener(openedPort, secret)
}

// OS-specific service finalizer
func doStopped() {
	// do not forget to close 'libivpn' dynamic library
	logger.Debug("Unloading libivpn...")
	libivpn.Unload()
	logger.Debug("Unloaded libivpn")
}

// checkIsAdmin - check is application running with root privileges
func doCheckIsAdmin() bool {
	uid := os.Geteuid()
	if uid != 0 {
		return false
	}

	return true
}
