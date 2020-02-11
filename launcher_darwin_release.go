// +build darwin,!debug

package main

func isNeedToSavePortInFile() bool {
	// macoOS release implementation does not need to save port info into file
	return false
}
