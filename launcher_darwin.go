package main

import (
	"os"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/oshelpers/macos/libivpn"
)

// Prepare to start IVPN daemon for macOS
func doPrepareToRun() error {
	return nil
}

// inform OS-specific implementation about listener port
func doStartedOnPort(openedPort int) {
	libivpn.StartXpcListener(openedPort)
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
	//logger.Debug("SUDO_UID:", os.Getenv("SUDO_UID"))
	//logger.Debug("SUDO_GID:", os.Getenv("SUDO_GID"))
	//logger.Debug("uid:", os.Geteuid())

	uid := os.Geteuid()
	if uid != 0 {
		return false
	}

	return true
}
