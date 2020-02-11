// +build darwin,debug

package main

func isNeedToSavePortInFile() bool {
	// only in debug mode (for macOS): save port info into file to be able debug project from IDE
	return true
}
