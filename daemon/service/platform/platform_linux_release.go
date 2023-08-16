//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

//go:build linux && !debug
// +build linux,!debug

package platform

import (
	"path"
)

func doOsInitForBuild() (warnings []string, errors []error, logInfo []string) {
	installDir := "/opt/ivpn"

	// check if we are running in snap environment
	if envs := GetSnapEnvs(); envs != nil {
		installDir = path.Join(envs.SNAP, "/opt/ivpn")

		if logInfo == nil {
			logInfo = make([]string, 0)
		}
		logInfo = append(logInfo, "Running in SNAP environment!")
	}

	firewallScript = path.Join(installDir, "etc/firewall.sh")
	splitTunScript = path.Join(installDir, "etc/splittun.sh")
	openvpnCaKeyFile = path.Join(installDir, "etc/ca.crt")
	openvpnTaKeyFile = path.Join(installDir, "etc/ta.key")
	openvpnUpScript = path.Join(installDir, "etc/client.up")
	openvpnDownScript = path.Join(installDir, "etc/client.down")
	serversFileBundled = path.Join(installDir, "etc/servers.json")

	obfsproxyStartScript = path.Join(installDir, "obfsproxy/obfs4proxy")

	v2rayBinaryPath = path.Join(installDir, "v2ray/v2ray")
	v2rayConfigTmpFile = path.Join(tmpDir, "v2ray.json")

	wgBinaryPath = path.Join(installDir, "wireguard-tools/wg-quick")
	wgToolBinaryPath = path.Join(installDir, "wireguard-tools/wg")

	dnscryptproxyBinPath = path.Join(installDir, "dnscrypt-proxy/dnscrypt-proxy")
	dnscryptproxyConfigTemplate = path.Join(installDir, "etc/dnscrypt-proxy-template.toml")
	dnscryptproxyConfig = path.Join(tmpDir, "dnscrypt-proxy.toml")

	kemHelperBinaryPath = path.Join(installDir, "kem/kem-helper")

	settingsFile = path.Join(tmpDir, "settings.json")
	openvpnConfigFile = path.Join(tmpDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(tmpDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(tmpDir, "wgivpn.conf")

	return nil, nil, logInfo
}
