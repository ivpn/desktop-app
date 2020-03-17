package platform

import (
	"path"
)

var (
	firewallScript string
)

// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
func doInitConstants() {
	servicePortFile = "/opt/ivpn/port.txt"
}

func doOsInit() {
	openVpnBinaryPath = path.Join("/usr/sbin", "openvpn")
	wgBinaryPath = path.Join("/usr/bin", "wg-quick")
	wgToolBinaryPath = path.Join("/usr/bin", "wg")

	doOsInitForBuild()
}

// FirewallScript returns path to firewal script
func FirewallScript() string {
	return firewallScript
}
