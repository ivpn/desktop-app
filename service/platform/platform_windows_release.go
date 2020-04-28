// +build windows,!debug

package platform

import (
	"fmt"
	"path"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
func doInitConstantsForBuild() {
}

func doOsInitForBuild() {
	installDir := getInstallDir()
	wfpDllPath = path.Join(installDir, "IVPN Firewall Native x64.dll")
	nativeHelpersDllPath = path.Join(installDir, "IVPN Helpers Native x64.dll")
	if Is64Bit() == false {
		wfpDllPath = path.Join(installDir, "IVPN Firewall Native.dll")
		nativeHelpersDllPath = path.Join(installDir, "IVPN Helpers Native.dll")
	}
}

func getInstallDir() string {
	ret := ""

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\IVPN Client`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		fmt.Println("ERROR: ", err)
	}
	defer k.Close()

	if err == nil {
		ret, _, err = k.GetStringValue("")
		if err != nil {
			fmt.Println("ERROR: ", err)
		}
	}

	if len(ret) == 0 {
		fmt.Println("WARNING: There is no info about IVPN Client install folder in the registry. Is IVPN Client installed?")
		return ""
	}

	return strings.ReplaceAll(ret, `\`, `/`)
}
