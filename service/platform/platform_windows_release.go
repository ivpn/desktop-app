// +build windows,!debug

package platform

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
func doInitConstants() {
	installDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(fmt.Sprintf("Failed to obtain folder of current binary: %s", err.Error()))
	}
	// TODO: use standard user setting folder (here must be a constant path to 'servicePortFile')
	if len(servicePortFile) <= 0 {
		servicePortFile = path.Join(installDir, "etc/port.txt")
	} else {
		// debug version can have different port file value
		fmt.Println("!!! WARNING!!! Non-standard service port file: ", servicePortFile)
	}
}

func doOsInitForBuild() (instDir string) {
	installDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(fmt.Sprintf("Failed to obtain folder of current binary: %s", err.Error()))
	}

	wfpDllPath = path.Join(installDir, "IVPN Firewall Native x64.dll")
	nativeHelpersDllPath = path.Join(installDir, "IVPN Helpers Native x64.dll")
	if Is64Bit() == false {
		wfpDllPath = path.Join(installDir, "IVPN Firewall Native.dll")
		nativeHelpersDllPath = path.Join(installDir, "IVPN Helpers Native.dll")
	}

	return installDir
}
