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

//go:build linux && debug
// +build linux,debug

package platform

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func doOsInitForBuild() (warnings []string, errors []error, logInfo []string) {
	installDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(fmt.Sprintf("Failed to obtain folder of current binary: %s", err.Error()))
	}

	if len(os.Args) > 2 {
		firstArg := strings.Split(os.Args[1], "=")
		if len(firstArg) == 2 && firstArg[0] == "-debug_install_dir" {
			installDir = firstArg[1]
		}
	}

	// When running tests, the installDir is detected as a dir where test located
	// we need to point installDir to project root
	// Therefore, we cutting rest after "desktop-app/daemon"
	rootDir := "desktop-app/daemon"
	if idx := strings.LastIndex(installDir, rootDir); idx > 0 {
		installDir = installDir[:idx+len(rootDir)]
	}

	etcDir := path.Join(installDir, "References/Linux/etc")
	etcDirCommon := path.Join(installDir, "References/common/etc")
	installDir = path.Join(installDir, "References/Linux")

	firewallScript = path.Join(etcDir, "firewall.sh")
	splitTunScript = path.Join(etcDir, "splittun.sh")
	openvpnCaKeyFile = path.Join(etcDirCommon, "ca.crt")
	openvpnTaKeyFile = path.Join(etcDirCommon, "ta.key")
	openvpnUpScript = path.Join(etcDir, "client.up")
	openvpnDownScript = path.Join(etcDir, "client.down")
	serversFileBundled = path.Join(etcDirCommon, "servers.json")

	obfsproxyStartScript = path.Join(installDir, "_deps/obfs4proxy_inst/obfs4proxy")

	v2rayBinaryPath = path.Join(installDir, "_deps/v2ray_inst/v2ray")
	v2rayConfigTmpFile = path.Join(tmpDir, "v2ray.json")

	wgBinaryPath = path.Join(installDir, "_deps/wireguard-tools_inst/wg-quick")
	wgToolBinaryPath = path.Join(installDir, "_deps/wireguard-tools_inst/wg")

	dnscryptproxyBinPath = path.Join(installDir, "_deps/dnscryptproxy_inst/dnscrypt-proxy")
	dnscryptproxyConfigTemplate = path.Join(etcDirCommon, "dnscrypt-proxy-template.toml")
	dnscryptproxyConfig = path.Join(tmpDir, "dnscrypt-proxy.toml")

	kemHelperBinaryPath = path.Join(installDir, "_deps/kem-helper/kem-helper-bin/kem-helper")

	settingsFile = path.Join(tmpDir, "settings.json")
	openvpnConfigFile = path.Join(tmpDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(tmpDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(tmpDir, "wgivpn.conf")

	return nil, nil, nil
}
