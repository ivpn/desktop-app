// +build windows,debug

package platform

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
func doInitConstants() {
	servicePortFile = `C:/Program Files/IVPN Client/etc/port.txt`
	if err = os.MkdirAll(filepath.Dir(servicePortFile), os.ModePerm); err != nil {
		fmt.Printf("!!! DEBUG VERSION !!! ERROR	: '%s'\n", servicePortFile)
		servicePortFile = ""
	}
}

func doOsInitForBuild() (instDir string) {
	installDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(fmt.Sprintf("Failed to obtain folder of current binary: %s", err.Error()))
	}

	// When running tests, the installDir is detected as a dir where test located
	// we need to point installDir to project root
	// Therefore, we cutting rest after "desktop-app-daemon"
	rootDir := "desktop-app-daemon"
	if idx := strings.LastIndex(installDir, rootDir); idx > 0 {
		installDir = installDir[:idx+len(rootDir)]
	}

	installDir = strings.ReplaceAll(installDir, `\`, `/`)
	installDir = path.Join(installDir, "References/Windows")

	wfpDllPath = path.Join(installDir, `Native Projects/bin/Release/IVPN Firewall Native x64.dll`)
	nativeHelpersDllPath = path.Join(installDir, `Native Projects/bin/Release/IVPN Firewall Native x64.dll`)

	if Is64Bit() == false {
		wfpDllPath = path.Join(installDir, `Native Projects/bin/Release/IVPN Firewall Native.dll`)
		nativeHelpersDllPath = path.Join(installDir, `Native Projects/bin/Release/IVPN Firewall Native.dll`)
	}

	fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	fmt.Printf("!!! DEBUG VERSION !!! wfpDllPath            : '%s'\n", wfpDllPath)
	fmt.Printf("!!! DEBUG VERSION !!! nativeHelpersDllPath  : '%s'\n", nativeHelpersDllPath)
	fmt.Printf("!!! DEBUG VERSION !!! servicePortFile	: '%s'\n", servicePortFile)
	fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")

	return installDir
}
