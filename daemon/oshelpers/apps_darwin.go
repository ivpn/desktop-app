//+build darwin

package oshelpers

import (
	"fmt"
)

func implGetInstalledApps() (apps []AppInfo, error) {
	return nil, fmt.Errorf("not implemented for macOS")
}

func implGetBinaryIconBase64Png(binaryPath string) (icon string, err error) {
	return "", fmt.Errorf("not implemented for macOS")
}
