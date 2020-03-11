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
	servicePortFile = "/tmp/ivpn/port.txt"
}

func doOsInit() {
	doOsInitForBuild()
}

func doOsInitForBuild() {
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

	// common variables initialization
	settingsDir = "/tmp/ivpn" //path.Join(installDir, "References/linux/tmp")
	settingsFile = path.Join(settingsDir, "settings.json")
	serversFile = path.Join(settingsDir, "servers.json") // path.Join(installDir, "References/macOS/etc/servers.json")
	openvpnConfigFile = path.Join(settingsDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(settingsDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(settingsDir, "wireguard.conf")

	logDir = path.Join(rootDir, "References/linux/tmp")
	logFile = path.Join(logDir, "IVPN Agent.log")
	openvpnLogFile = path.Join(logDir, "openvpn.log")

	openVpnBinaryPath = path.Join("/usr/sbin", "openvpn") //path.Join(installDir, "References/macOS/_deps/openvpn_inst/bin/openvpn")
	openvpnCaKeyFile = path.Join(installDir, "References/linux/etc/ca.crt")
	openvpnTaKeyFile = path.Join(installDir, "References/linux/etc/ta.key")
	openvpnUpScript = path.Join(installDir, "References/linux/etc/client.up")
	openvpnDownScript = path.Join(installDir, "References/linux/etc/client.down")

	obfsproxyStartScript = path.Join(installDir, "References/linux/obfsproxy/obfsproxy.sh")

	wgBinaryPath = path.Join("/usr/bin", "wg-quick") //path.Join(installDir, "References/macOS/_deps/wg_inst/wireguard-go")
	wgToolBinaryPath = path.Join("/usr/bin", "wg")   // path.Join(installDir, "References/macOS/_deps/wg_inst/wg")
}
