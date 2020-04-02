// +build darwin,!debug

package platform

import "path"

func doOsInitForBuild() (warnings []string, errors []error) {
	// macOS-specific variable initialization
	firewallScript = "/Applications/IVPN.app/Contents/Resources/etc/firewall.sh"
	dnsScript = "/Applications/IVPN.app/Contents/Resources/etc/dns.sh"

	// common variables initialization
	settingsDir = "/Library/Application Support/IVPN"
	settingsFile = path.Join(settingsDir, "settings.json")
	serversFile = path.Join(settingsDir, "servers.json")
	openvpnConfigFile = path.Join(settingsDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(settingsDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(settingsDir, "wireguard.conf")

	logDir = "/Library/Logs/"
	logFile = path.Join(logDir, "IVPN Agent.log")
	openvpnLogFile = path.Join(logDir, "openvpn.log")

	openVpnBinaryPath = "/Applications/IVPN.app/Contents/MacOS/openvpn"
	openvpnCaKeyFile = "/Applications/IVPN.app/Contents/Resources/etc/ca.crt"
	openvpnTaKeyFile = "/Applications/IVPN.app/Contents/Resources/etc/ta.key"
	openvpnUpScript = "/Applications/IVPN.app/Contents/Resources/etc/dns.sh -up"
	openvpnDownScript = "/Applications/IVPN.app/Contents/Resources/etc/dns.sh -down"

	obfsproxyStartScript = "/Applications/IVPN.app/Contents/Resources/obfsproxy/obfsproxy.sh"

	wgBinaryPath = "/Applications/IVPN.app/Contents/MacOS/WireGuard/wireguard-go"
	wgToolBinaryPath = "/Applications/IVPN.app/Contents/MacOS/WireGuard/wg"

	return nil, nil
}
