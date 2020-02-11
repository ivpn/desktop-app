// +build windows,!debug

package platform

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

func doOsInitForBuild() (instDir string){
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
