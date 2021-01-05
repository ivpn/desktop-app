// +build darwin,!libivpn

package main

// inform OS-specific implementation about listener port
func implStartedOnPort(openedPort int, secret uint64) {
	// nothing to do here
}

// OS-specific service finalizer
func implStopped() {
	// nothing to do here
}
