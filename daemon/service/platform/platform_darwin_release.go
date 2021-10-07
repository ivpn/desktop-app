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

// +build darwin,!debug

package platform

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/ivpn/desktop-app/daemon/service/platform/filerights"
)

func doOsInitForBuild() (warnings []string, errors []error) {
	// macOS-specific variable initialization
	firewallScript = "/Applications/IVPN.app/Contents/Resources/etc/firewall.sh"
	dnsScript = "/Applications/IVPN.app/Contents/Resources/etc/dns.sh"

	// common variables initialization
	settingsDir := "/Library/Application Support/IVPN"
	settingsFile = path.Join(settingsDir, "settings.json")
	serversFile = path.Join(settingsDir, "servers.json")
	openvpnConfigFile = path.Join(settingsDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(settingsDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(settingsDir, "wireguard.conf")

	openVpnBinaryPath = "/Applications/IVPN.app/Contents/MacOS/openvpn"
	openvpnCaKeyFile = "/Applications/IVPN.app/Contents/Resources/etc/ca.crt"
	openvpnTaKeyFile = "/Applications/IVPN.app/Contents/Resources/etc/ta.key"
	openvpnUpScript = "/Applications/IVPN.app/Contents/Resources/etc/dns.sh -up"
	openvpnDownScript = "/Applications/IVPN.app/Contents/Resources/etc/dns.sh -down"

	obfsproxyStartScript = "/Applications/IVPN.app/Contents/Resources/obfsproxy/obfs4proxy"

	wgBinaryPath = "/Applications/IVPN.app/Contents/MacOS/WireGuard/wireguard-go"
	wgToolBinaryPath = "/Applications/IVPN.app/Contents/MacOS/WireGuard/wg"

	return nil, nil
}

func doInitOperations() (w string, e error) {
	serversFile := ServersFile()
	if _, err := os.Stat(serversFile); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("File '%s' does not exists. Copying from bundle...\n", serversFile)
			// Servers file is not exists on required place
			// Probably, it is first start after clean install
			// Copying it from a bundle
			os.MkdirAll(filepath.Base(serversFile), os.ModePerm)
			if _, err = copyFile("/Applications/IVPN.app/Contents/Resources/etc/servers.json", serversFile); err != nil {
				return err.Error(), nil
			}
			return "", nil
		}

		return err.Error(), nil
	}
	return "", nil
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	destination.Chmod(filerights.DefaultFilePermissionsForConfig())
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
