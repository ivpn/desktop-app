package darwinhelpers

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func GetOsMajorVersion() (int, error) {
	// Checking macOS version
	var uts unix.Utsname
	if err := unix.Uname(&uts); err != nil {
		return 0, fmt.Errorf("Can not obtain macOS version: %w", err)
	}
	release := unix.ByteSliceToString(uts.Release[:])
	dotPos := strings.Index(release, ".")
	if dotPos == -1 {
		return 0, fmt.Errorf("Can not obtain macOS version")
	}
	major := release[:dotPos]
	majorVersion, err := strconv.Atoi(major)
	if err != nil {
		return 0, fmt.Errorf("Can not obtain macOS version: %w", err)
	}
	return majorVersion, nil
}
