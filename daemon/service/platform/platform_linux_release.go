//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
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

// +build linux,!debug

package platform

import (
	"path"
)

func doOsInitForBuild() (warnings []string, errors []error) {
	installDir := "/opt/ivpn"

	firewallScript = path.Join(installDir, "etc/firewall.sh")
	openvpnCaKeyFile = path.Join(installDir, "etc/ca.crt")
	openvpnTaKeyFile = path.Join(installDir, "etc/ta.key")
	openvpnUpScript = path.Join(installDir, "etc/client.up")
	openvpnDownScript = path.Join(installDir, "etc/client.down")
	serversFile = path.Join(installDir, "etc/servers.json")
	
	obfsproxyStartScript = path.Join(installDir, "obfsproxy/obfs4proxy")

	settingsFile = path.Join(tmpDir, "settings.json")
	openvpnConfigFile = path.Join(tmpDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(tmpDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(tmpDir, "wgivpn.conf")

	return nil, nil
}
