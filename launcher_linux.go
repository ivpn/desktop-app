package main

import "os"

func doPrepareToRun() error {
	return nil
}

func doStopped() {
}

func doCheckIsAdmin() bool {
	uid := os.Geteuid()
	if uid != 0 {
		return false
	}

	return true
}

func doStartedOnPort(port int, secret uint64) {
}

func isNeedToSavePortInFile() bool {
	return true
}
