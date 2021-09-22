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

// +build darwin,debug

package platform

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func doOsInitForBuild() (warnings []string, errors []error) {
	installDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(fmt.Sprintf("Failed to obtain folder of current binary: %s", err.Error()))
	}

	// When running tests, the installDir is detected as a dir where test located
	// we need to point installDir to project root
	// Therefore, we cutting rest after "desktop-app/daemon"
	rootDir := "desktop-app/daemon"
	if idx := strings.LastIndex(installDir, rootDir); idx > 0 {
		installDir = installDir[:idx+len(rootDir)]
	}

	// macOS-specific variable initialization
	firewallScript = path.Join(installDir, "References/macOS/etc/firewall.sh")
	dnsScript = path.Join(installDir, "References/macOS/etc/dns.sh")

	// common variables initialization
	settingsDir := "/Library/Application Support/IVPN"
	settingsFile = path.Join(settingsDir, "settings.json")
	serversFile = path.Join(settingsDir, "servers.json")
	openvpnConfigFile = path.Join(settingsDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(settingsDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(settingsDir, "wireguard.conf")

	openVpnBinaryPath = path.Join(installDir, "References/macOS/_deps/openvpn_inst/bin/openvpn")
	openvpnCaKeyFile = path.Join(installDir, "References/macOS/etc/ca.crt")
	openvpnTaKeyFile = path.Join(installDir, "References/macOS/etc/ta.key")
	openvpnUpScript = path.Join(installDir, "References/macOS/etc/dns.sh -up")
	openvpnDownScript = path.Join(installDir, "References/macOS/etc/dns.sh -down")

	obfsproxyStartScript = path.Join(installDir, "References/macOS/_deps/obfs4proxy_inst/obfs4proxy")

	wgBinaryPath = path.Join(installDir, "References/macOS/_deps/wg_inst/wireguard-go")
	wgToolBinaryPath = path.Join(installDir, "References/macOS/_deps/wg_inst/wg")

	return nil, nil
}

func doInitOperations() (w string, e error) {
	serversFile := ServersFile()
	if _, err := os.Stat(serversFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("!!!DEBUG!!! File '%s' not exists (will be downloaded from backend; will lead to errors in case of failed download!)", serversFile), nil
		}
		return "", err
	}
	return "", nil
}
