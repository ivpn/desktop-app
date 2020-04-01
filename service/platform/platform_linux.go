package platform

import (
	"path"
)

var (
	firewallScript string
	logDir         string = "/opt/ivpn/log"
	tmpDir         string = "/opt/ivpn/mutable"
)

// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
func doInitConstants() {
	servicePortFile = path.Join(tmpDir, "port.txt")
}

func doOsInit() (warnings []string, errors []error) {
	openVpnBinaryPath = path.Join("/usr/sbin", "openvpn")
	obfsproxyStartScript = "/usr/bin/obfsproxy"
	wgBinaryPath = path.Join("/usr/bin", "wg-quick")
	wgToolBinaryPath = path.Join("/usr/bin", "wg")

	return doOsInitForBuild()
}

// FirewallScript returns path to firewal script
func FirewallScript() string {
	return firewallScript
}
