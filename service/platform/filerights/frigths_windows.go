// +build windows

package filerights

import (
	"fmt"
	"os"
)

// DefaultFilePermissionsForConfig - returns default file permissions to save config files
func DefaultFilePermissionsForConfig() os.FileMode { return 0600 }

// CheckFileAccessRigthsConfig ensures if given file has correct rights for mutable config file
func CheckFileAccessRigthsConfig(file string) error {
	// No file rigths check for Windows
	// Application is installed to a '%PROGRAMFILES%' which is write-accessible only for admins
	return isFileExists(file)
}

// CheckFileAccessRigthsStaticConfig ensures if given file has correct rights for unmutable config file
func CheckFileAccessRigthsStaticConfig(file string) error {
	// No file rigths check for Windows
	// Application is installed to a '%PROGRAMFILES%' which is write-accessible only for admins
	return isFileExists(file)
}

// CheckFileAccessRigthsExecutable checks if file has correct access-permission for executable
// If file does not exist or it can be writable by someone alse except root - retun error
func CheckFileAccessRigthsExecutable(file string) error {
	// No file rigths check for Windows
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
