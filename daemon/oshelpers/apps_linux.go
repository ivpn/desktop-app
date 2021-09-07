//+build linux

package oshelpers

import (
	"fmt"
)

func implGetInstalledApps(extraArgsJSON string) (apps []AppInfo, error) {
	return nil, fmt.Errorf("not implemented for Linux")
}

func implGetBinaryIconBase64Png(binaryPath string) (icon string, err error) {
	return "", fmt.Errorf("not implemented for Linux")
}
