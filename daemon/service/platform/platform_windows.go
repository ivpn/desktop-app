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

package platform

import (
	"fmt"
	"os"
	"path"
	"strings"
)

var (
	wfpDllPath           string
	nativeHelpersDllPath string
	splitTunDriverPath   string
)

func doInitConstants() {
	fwInitialValueAllowApiServers = true
	doInitConstantsForBuild()

	installDir := getInstallDir()
	if len(servicePortFile) <= 0 {
		servicePortFile = path.Join(installDir, "etc/port.txt")
	} else {
		// debug version can have different port file value
		fmt.Println("!!! WARNING !!! Non-standard service port file: ", servicePortFile)
	}

	logFile = path.Join(installDir, "log/IVPN Agent.log")
	openvpnLogFile = path.Join(installDir, "log/openvpn.log")

	openvpnUserParamsFile = path.Join(installDir, "mutable/ovpn_extra_params.txt")
}

func doOsInit() (warnings []string, errors []error) {
	SYSTEMROOT := os.Getenv("SYSTEMROOT")
	if len(SYSTEMROOT) > 0 {
		routeCommand = strings.ReplaceAll(path.Join(SYSTEMROOT, "System32", "ROUTE.EXE"), "/", "\\")
	}

	doOsInitForBuild()
	_installDir := getInstallDir()

	_archDir := "x86_64"
	if !Is64Bit() {
		_archDir = "x86"
	}

	if warnings == nil {
		warnings = make([]string, 0)
	}
	if errors == nil {
		errors = make([]error, 0)
	}

	// common variables initialization
	settingsDir := path.Join(_installDir, "etc")
	settingsFile = path.Join(settingsDir, "settings.json")

	serversFile = path.Join(settingsDir, "servers.json")
	openvpnConfigFile = path.Join(settingsDir, "openvpn.cfg")
	openvpnProxyAuthFile = path.Join(settingsDir, "proxyauth.txt")
	wgConfigFilePath = path.Join(settingsDir, "IVPN.conf") // will be used also for WireGuard service name (e.g. "WireGuardTunnel$IVPN")

	openVpnBinaryPath = path.Join(_installDir, "OpenVPN", _archDir, "openvpn.exe")
	openvpnCaKeyFile = path.Join(settingsDir, "ca.crt")
	openvpnTaKeyFile = path.Join(settingsDir, "ta.key")
	openvpnUpScript = ""
	openvpnDownScript = ""

	obfsproxyStartScript = path.Join(_installDir, "OpenVPN", "obfsproxy", "obfs4proxy.exe")

	_wgArchDir := "x86_64"
	if _, err := os.Stat(path.Join(_installDir, "WireGuard", _wgArchDir, "wireguard.exe")); err != nil {
		_wgArchDir = "x86"
		if _, err := os.Stat(path.Join(_installDir, "WireGuard", _wgArchDir, "wireguard.exe")); err != nil {
			errors = append(errors, fmt.Errorf("unabale to find WireGuard binary: %s ..<x86_64\\x86>", path.Join(_installDir, "WireGuard")))
		}
	}
	wgBinaryPath = path.Join(_installDir, "WireGuard", _wgArchDir, "wireguard.exe")
	wgToolBinaryPath = path.Join(_installDir, "WireGuard", _wgArchDir, "wg.exe")

	if _, err := os.Stat(wfpDllPath); err != nil {
		errors = append(errors, fmt.Errorf("file not exists: '%s'", wfpDllPath))
	}
	if _, err := os.Stat(nativeHelpersDllPath); err != nil {
		errors = append(errors, fmt.Errorf("file not exists: '%s'", nativeHelpersDllPath))
	}
	if _, err := os.Stat(splitTunDriverPath); err != nil {
		warnings = append(warnings, fmt.Errorf("file not exists: '%s'", splitTunDriverPath).Error())
	}

	return warnings, errors
}

func doInitOperations() (w string, e error) { return "", nil }

// WindowsWFPDllPath - Path to Windows DLL with helper methods for WFP (Windows Filtering Platform)
func WindowsWFPDllPath() string {
	return wfpDllPath
}

// WindowsNativeHelpersDllPath - Path to Windows DLL with helper methods (native DNS implementation... etc.)
func WindowsNativeHelpersDllPath() string {
	return nativeHelpersDllPath
}

// WindowsSplitTunnelDriverPath - path to *.sys binary of Split-Tunnel driver
func WindowsSplitTunnelDriverPath() string {
	return splitTunDriverPath
}
