// +build linux,!debug

package platform

import (
	"path"
)

func doOsInitForBuild() {
	installDir := "/opt/ivpn"

	firewallScript = path.Join(installDir, "etc/firewall.sh")
	openvpnCaKeyFile = path.Join(installDir, "etc/ca.crt")
	openvpnTaKeyFile = path.Join(installDir, "etc/ta.key")
	openvpnUpScript = path.Join(installDir, "etc/client.up")
	openvpnDownScript = path.Join(installDir, "etc/client.down")
	settingsFile = path.Join(installDir, "etc/settings.json")
	serversFile = path.Join(installDir, "etc/servers.json")

	openvpnConfigFile = path.Join(tmpDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(tmpDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(tmpDir, "wgivpn.conf")

	logFile = path.Join(logDir, "IVPN Agent.log")
	openvpnLogFile = path.Join(logDir, "openvpn.log")
}
