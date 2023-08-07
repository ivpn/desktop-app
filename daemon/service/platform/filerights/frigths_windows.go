//go:build windows
// +build windows

package filerights

import (
	"fmt"
	"os"
	"strings"

	"github.com/ivpn/desktop-app/daemon/oshelpers/windows/go-acl"
)

var envVarProgramFiles string
var envVarSystemroot string
var isDebug bool = false

func init() {
	envVarProgramFiles = strings.ToLower(os.Getenv("ProgramFiles"))
	if len(envVarProgramFiles) == 0 {
		fmt.Println("!!! ERROR !!! Unable to determine 'ProgramFiles' environment variable")
	}
	envVarSystemroot = strings.ToLower(os.Getenv("SYSTEMROOT"))
	if len(envVarSystemroot) == 0 {
		fmt.Println("!!! ERROR !!! Unable to determine 'SYSTEMROOT' environment variable")
	}
}

// DefaultFilePermissionsForConfig - returns default file permissions to save config files
func DefaultFilePermissionsForConfig() os.FileMode { return 0600 }

// CheckFileAccessRightsConfig ensures if given file has correct rights for mutable config file
func CheckFileAccessRightsConfig(file string) error {
	// No file rights check for Windows
	// Application is installed to a '%PROGRAMFILES%' which is write-accessible only for admins
	return isFileInProgramFiles(file)
}

// CheckFileAccessRightsStaticConfig ensures if given file has correct rights for unmutable config file
func CheckFileAccessRightsStaticConfig(file string) error {
	// No file rights check for Windows
	// Application is installed to a '%PROGRAMFILES%' which is write-accessible only for admins
	return isFileInProgramFiles(file)
}

// CheckFileAccessRightsExecutable checks if file has correct access-permission for executable
// If file does not exist or it can be writable by someone else except root - return error
func CheckFileAccessRightsExecutable(file string) error {
	// No file rights check for Windows
	// Application is installed to a '%PROGRAMFILES%' which is write-accessible only for admins
	// External binaries can be started only from '%SYSTEMROOT%' which is also write-accessible only for admins

	file = strings.ToLower(strings.ReplaceAll(file, "/", "\\"))
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file '%s' does not exists", file)
		}
	}
	if strings.HasPrefix(file, envVarProgramFiles) {
		return isFileInProgramFiles(file)
	}
	if strings.HasPrefix(file, envVarSystemroot) {
		return isFileInSystemroot(file)
	}

	if isDebug {
		return nil
	}

	return fmt.Errorf("file '%s' is not in folders '%s' or '%s'", file, envVarProgramFiles, envVarSystemroot)
}

func isFileInProgramFiles(file string) error {
	if err := isFileExists(file); err != nil {
		return err
	}

	if len(envVarProgramFiles) == 0 {
		return fmt.Errorf("the 'ProgramFiles' environment variable not initialized")
	}

	if isDebug == false {
		if strings.HasPrefix(strings.ToLower(strings.ReplaceAll(file, "/", "\\")), envVarProgramFiles) == false {
			return fmt.Errorf("file '%s' is not in folder '%s'", file, envVarProgramFiles)
		}
	}
	return nil
}

func isFileInSystemroot(file string) error {
	if err := isFileExists(file); err != nil {
		return err
	}

	if len(envVarSystemroot) == 0 {
		return fmt.Errorf("the 'SYSTEMROOT' environment variable not initialized")
	}

	if isDebug == false {
		if strings.HasPrefix(strings.ToLower(strings.ReplaceAll(file, "/", "\\")), envVarSystemroot) == false {
			return fmt.Errorf("file '%s' is not in folder '%s'", file, envVarSystemroot)
		}
	}
	return nil
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

// WindowsChmod - changing file permissions in Windows style
// (Golang is not able to change file permissins  in Windows style)
func WindowsChmod(name string, fileMode os.FileMode) error {
	return acl.Chmod(name, fileMode)
}
