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

// checkIsAdmin - check is application running with root privilages
func doCheckIsAdmin() bool {
	uid := os.Geteuid()
	if uid != 0 {
		return false
	}

	return true
}
