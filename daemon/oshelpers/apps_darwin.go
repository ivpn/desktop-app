//+build darwin

package oshelpers

import (
	"fmt"
)

func implGetInstalledApps(extraArgsJSON string) ([]AppInfo, error) {
	return nil, fmt.Errorf("not implemented for macOS")
}

func implGetFunc_BinaryIconBase64(binaryPath string) (icon string, err error) {
	return nil
}
