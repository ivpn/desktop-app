// +build linux,debug

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

	installDir = path.Join(installDir, "References/Linux")

	logDir := "/opt/ivpn/log"

	firewallScript = path.Join(installDir, "etc/firewall.sh")
	openvpnCaKeyFile = path.Join(installDir, "etc/ca.crt")
	openvpnTaKeyFile = path.Join(installDir, "etc/ta.key")
	openvpnUpScript = path.Join(installDir, "etc/client.up")
	openvpnDownScript = path.Join(installDir, "etc/client.down")

	settingsFile = path.Join(installDir, "etc/settings.json")
	serversFile = path.Join(installDir, "etc/servers.json")
	openvpnConfigFile = path.Join(installDir, "etc/openvpn.cfg")
	openvpnProxyAuthFile = path.Join(installDir, "etc/proxyauth.txt")
	wgConfigFilePath = path.Join(installDir, "etc/wgivpn.conf")

	logFile = path.Join(logDir, "IVPN Agent.log")
	openvpnLogFile = path.Join(logDir, "openvpn.log")
}
