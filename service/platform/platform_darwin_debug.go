// +build darwin,debug

package platform

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

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

	// macOS-specivic variable initialization
	firewallScript = path.Join(installDir, "References/macOS/etc/firewall.sh")
	ensureFileExists("firewallScript", firewallScript)

	dnsScript = path.Join(installDir, "References/macOS/etc/dns.sh")
	ensureFileExists("firewallScript", dnsScript)

	// common variables initialization
	settingsDir = "/Library/Application Support/IVPN"
	settingsFile = path.Join(settingsDir, "settings.json")
	servicePortFile = path.Join(settingsDir, "port.txt")
	serversFile = path.Join(settingsDir, "servers.json") // path.Join(installDir, "References/macOS/etc/servers.json")
	openvpnConfigFile = path.Join(settingsDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(settingsDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(settingsDir, "wireguard.conf")

	logDir = "/Library/Logs/"
	logFile = path.Join(logDir, "IVPN Agent.log")
	openvpnLogFile = path.Join(logDir, "openvpn.log")

	openVpnBinaryPath = path.Join(installDir, "References/macOS/_deps/openvpn_inst/bin/openvpn")
	openvpnCaKeyFile = path.Join(installDir, "References/macOS/etc/ca.crt")
	openvpnTaKeyFile = path.Join(installDir, "References/macOS/etc/ta.key")
	openvpnUpScript = path.Join(installDir, "References/macOS/etc/dns.sh -up")
	openvpnDownScript = path.Join(installDir, "References/macOS/etc/dns.sh -down")

	obfsproxyStartScript = path.Join(installDir, "References/macOS/obfsproxy/obfsproxy.sh")

	wgBinaryPath = path.Join(installDir, "References/macOS/_deps/wg_inst/wireguard-go")
	wgToolBinaryPath = path.Join(installDir, "References/macOS/_deps/wg_inst/wg")
}
