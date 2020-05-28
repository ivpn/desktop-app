// +build windows

package filerights

import (
	"fmt"
	"os"
)

// DefaultFilePermissionsForConfig - returns default file permissions to save config files
func DefaultFilePermissionsForConfig() os.FileMode { return 0600 }

// CheckFileAccessRightsConfig ensures if given file has correct rights for mutable config file
func CheckFileAccessRightsConfig(file string) error {
	// No file rights check for Windows
	// Application is installed to a '%PROGRAMFILES%' which is write-accessible only for admins
	return isFileExists(file)
}

// CheckFileAccessRightsStaticConfig ensures if given file has correct rights for unmutable config file
func CheckFileAccessRightsStaticConfig(file string) error {
	// No file rights check for Windows
	// Application is installed to a '%PROGRAMFILES%' which is write-accessible only for admins
	return isFileExists(file)
}

// CheckFileAccessRightsExecutable checks if file has correct access-permission for executable
// If file does not exist or it can be writable by someone else except root - return error
func CheckFileAccessRightsExecutable(file string) error {
	// No file rights check for Windows
	// Application is installed to a '%PROGRAMFILES%' which is write-accessible only for admins
	return isFileExists(file)
}

func isFileExists(file string) error {
	stat, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("file existing check error '%s' : %w", file, err)
	}

	if stat.IsDir() {
		return fmt.Errorf("'%s' is directory", file)
	}
	return nil
}
