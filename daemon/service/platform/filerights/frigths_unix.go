//go:build darwin || linux
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
	// permsForExecutableNotAcceptable - file permissions mask for executables which are not allowed.
	// Executable files should not have write access for someone else except root
	permsForExecutableNotAcceptable os.FileMode = 0022

	// permsForConfig - mutable config files should have permissions read/write ONLY for owner (root)
	permsForConfig os.FileMode = 0600

	// un-mutable config files should have read rights ONLY for owner (root)
	permsForStaticConfigRequired      os.FileMode = 0400
	permsForStaticConfigNotAcceptable os.FileMode = 0077
)

// DefaultFilePermissionsForConfig - returns default file permissions to save config files
func DefaultFilePermissionsForConfig() os.FileMode { return permsForConfig }

// CheckFileAccessRightsConfig ensures if given file has correct rights for mutable config file
func CheckFileAccessRightsConfig(file string) error {
	return checkFileAccessRights(file, permsForConfig, 0, 0)
}

// CheckFileAccessRightsStaticConfig ensures if given file has correct rights for un-mutable config file
func CheckFileAccessRightsStaticConfig(file string) error {
	return checkFileAccessRights(file, 0, permsForStaticConfigRequired, permsForStaticConfigNotAcceptable)
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

	return checkFileAccessRights(file, 0, 0, permsForExecutableNotAcceptable)
}

func checkFileAccessRights(file string, fmodeOnly os.FileMode, fmodeRequired os.FileMode, fmodeNotAcceptable os.FileMode) error {
	if fmodeOnly == 0 && fmodeRequired == 0 && fmodeNotAcceptable == 0 {
		return fmt.Errorf("INTERNAL ERROR: parameters not defined")
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

	mode := stat.Mode()
	if fmodeOnly > 0 {
		if mode != fmodeOnly {
			return fmt.Errorf(fmt.Sprintf("file '%s' has wrong access permissions (%03o but expected %03o)", file, mode, fmodeOnly))
		}
		return nil
	}

	// check file REQUIRED access rights
	if fmodeRequired > 0 {
		if (mode & fmodeRequired) == 0 {
			return fmt.Errorf(fmt.Sprintf("file '%s' has wrong access permissions (%03o but required %03o)", file, mode, fmodeRequired))
		}
	}

	// check file NOT ACCEPTABLE access rights
	if fmodeNotAcceptable > 0 {
		if (mode & fmodeNotAcceptable) > 0 {
			return fmt.Errorf(fmt.Sprintf("file '%s' has wrong access permissions (%03o but not applicable perms mask is %03o)", file, mode, fmodeNotAcceptable))
		}
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
		return fmt.Errorf("wrong owner for a file (UID:%d). Expected a privileged user as owner (UID:%d)", fileOwnerUID, curUserID)
	}
	return nil
}

// WindowsChmod - changing file permissions in Windows style
// (applicable only for Windows)
func WindowsChmod(name string, fileMode os.FileMode) error {
	return nil // do nothing for darwin or linux
}
