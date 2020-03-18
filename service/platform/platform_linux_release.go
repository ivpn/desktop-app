// +build linux,!debug

package platform

import (
	"path"
)

func doOsInitForBuild() {
	installDir := "/opt/ivpn"
	logDir := "/opt/ivpn/log"

	firewallScript = path.Join(installDir, "etc/firewall.sh")
	openvpnCaKeyFile = path.Join(installDir, "etc/ca.crt")
	openvpnTaKeyFile = path.Join(installDir, "etc/ta.key")
	openvpnUpScript = path.Join(installDir, "etc/client.up")
	openvpnDownScript = path.Join(installDir, "etc/client.down")
	obfsproxyStartScript = path.Join(installDir, "obfsproxy/obfsproxy.sh")

	settingsFile = path.Join(installDir, "etc/settings.json")
	serversFile = path.Join(installDir, "etc/servers.json")
	openvpnConfigFile = path.Join(installDir, "etc/openvpn.cfg")
	openvpnProxyAuthFile = path.Join(installDir, "etc/proxyauth.txt")
	wgConfigFilePath = path.Join(installDir, "etc/wgivpn.conf")

	logFile = path.Join(logDir, "IVPN Agent.log")
	openvpnLogFile = path.Join(logDir, "openvpn.log")
}
