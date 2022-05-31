// +build darwin linux

package filerights

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

var (
	isDebug bool
)

const (
	// WrongExecutableFilePermissionsMask - file permissions mask for executables which are not allowed.
	// Executable files should not have write access for someone else except root
	wrongExecutableFilePermissionsMask os.FileMode = 0022
	// DefaultFilePermissionForConfig - mutable config files should have permissions read/write only for owner (root)
	defaultFilePermissionForConfig os.FileMode = 0600
	// DefaultFilePermissionForStaticConfig - unmutable config files should have permissions read/write only for owner (root)
	defaultFilePermissionForStaticConfig os.FileMode = 0400
)

// DefaultFilePermissionsForConfig - returns default file permissions to save config files
func DefaultFilePermissionsForConfig() os.FileMode { return defaultFilePermissionForConfig }

// CheckFileAccessRightsConfig ensures if given file has correct rights for mutable config file
func CheckFileAccessRightsConfig(file string) error {
	return ensureFileAccessRights(file, defaultFilePermissionForConfig)
}

// CheckFileAccessRightsStaticConfig ensures if given file has correct rights for unmutable config file
func CheckFileAccessRightsStaticConfig(file string) error {
	return ensureFileAccessRights(file, defaultFilePermissionForStaticConfig)
}

// CheckFileAccessRightsExecutable checks if file has correct access-permission for executable
// If file does not exist or it can be writable by someone else except root - return error
func CheckFileAccessRightsExecutable(file string) error {
	if len(file) > 0 {
		// 'file' can be presented as executable with arguments (e.g. 'dns.sh -up')
		// Trying here to take only executable file path without arguments
		file = strings.Split(file, " -")[0]
		file = strings.Split(file, "\t-")[0]
	}

	// check is file exists
	stat, err := getFileStat(file)
	if err != nil {
		return err
	}

	if isDebug {
		fmt.Println("WARNING! DEBUG MODE : permissions check skipped for file: ", file)
		return nil
	}

	// check file owner
	if err := ensureFileOwner(stat); err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}

	// check file access rights
	mode := stat.Mode()
	if (mode & wrongExecutableFilePermissionsMask) > 0 {
		return fmt.Errorf("file '%s' has wrong permissions (it can be modified not only by owner [%o])", file, mode)
	}
	return nil
}

func ensureFileAccessRights(file string, fmode os.FileMode) error {
	// check is file exists
	stat, err := getFileStat(file)
	if err != nil {
		return err
	}

	if isDebug {
		fmt.Println("WARNING! DEBUG MODE : permissions check skipped for file: ", file)
		return nil
	}

	// check file owner
	if err := ensureFileOwner(stat); err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}

	// check file access rights
	mode := stat.Mode()
	if mode != fmode {
		return fmt.Errorf(fmt.Sprintf("file '%s' has wrong access permissions (%o but expected %o)", file, mode, fmode))
	}

	return nil
}

func getFileStat(file string) (os.FileInfo, error) {
	stat, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return stat, err
		}
		return stat, fmt.Errorf("file existing check error '%s' : %w", file, err)
	}

	if stat.IsDir() {
		return stat, fmt.Errorf("'%s' is directory", file)
	}
	return stat, nil
}

func ensureFileOwner(finfo os.FileInfo) error {
	// check if current user (root, by default) is owner for a file
	fileOwnerUID := finfo.Sys().(*syscall.Stat_t).Uid
	curUserID := uint32(os.Getuid())
	if fileOwnerUID != curUserID {
		return fmt.Errorf("wrong owner for a file (UID:%d). Expected a privilaged user as owner (UID:%d)", fileOwnerUID, curUserID)
	}
	return nil
}

// WindowsChmod - changing file permissions in Windows style
// (applicable only for Windows)
func WindowsChmod(name string, fileMode os.FileMode) error {
	return nil // do nothing for darwin or linux
}
