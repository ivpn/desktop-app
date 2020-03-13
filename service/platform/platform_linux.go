package platform

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	firewallScript string
)

var usrCfgDir string

// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
func doInitConstants() {
	usrDir, err := os.UserConfigDir()
	if err != nil {
		panic("Unable to determine user's configuration dirrectory: " + err.Error())
	}
	usrCfgDir = usrDir
	settingsDir = path.Join(usrCfgDir, "ivpn")
	servicePortFile = path.Join(settingsDir, "port.txt")
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

	// Linux-specific variable initialization
	firewallScript = path.Join(installDir, "References/linux/etc/firewall.sh")

	// common variables initialization
	settingsFile = path.Join(settingsDir, "settings.json")
	serversFile = path.Join(settingsDir, "servers.json") // path.Join(installDir, "References/macOS/etc/servers.json")

	//tmpDir := "/tmp/ivpn"
	openvpnConfigFile = path.Join(settingsDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(settingsDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(settingsDir, "wgivpn.conf")

	logDir = path.Join(rootDir, "References/linux/tmp")
	logFile = path.Join(logDir, "IVPN Agent.log")
	openvpnLogFile = path.Join(logDir, "openvpn.log")

	openVpnBinaryPath = path.Join("/usr/sbin", "openvpn") //path.Join(installDir, "References/macOS/_deps/openvpn_inst/bin/openvpn")
	openvpnCaKeyFile = path.Join(installDir, "References/linux/etc/ca.crt")
	openvpnTaKeyFile = path.Join(installDir, "References/linux/etc/ta.key")
	openvpnUpScript = path.Join(installDir, "References/linux/etc/client.up")
	openvpnDownScript = path.Join(installDir, "References/linux/etc/client.down")

	obfsproxyStartScript = path.Join(installDir, "References/linux/obfsproxy/obfsproxy.sh")

	wgBinaryPath = path.Join("/usr/bin", "wg-quick")
	wgToolBinaryPath = path.Join("/usr/bin", "wg")
}

// FirewallScript returns path to firewal script
func FirewallScript() string {
	return firewallScript
}
