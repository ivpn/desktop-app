//+build darwin

package oshelpers

import (
	"fmt"
)

func implGetInstalledApps() (map[string]string, error) {
	return nil, fmt.Errorf("not implemented for macOS")
}
