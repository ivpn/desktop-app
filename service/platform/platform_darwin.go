//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
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
	"path"
)

var (
	firewallScript string
	dnsScript      string
)

// initialize all constant values (e.g. servicePortFile) which can be used in external projects (IVPN CLI)
func doInitConstants() {
	servicePortFile = "/Library/Application Support/IVPN/port.txt"
	openvpnUserParamsFile = "/Library/Application Support/IVPN/OpenVPN/ovpn_extra_params.txt"

	logDir := "/Library/Logs/"
	logFile = path.Join(logDir, "IVPN Agent.log")
	openvpnLogFile = path.Join(logDir, "openvpn.log")
}

func doOsInit() (warnings []string, errors []error) {
	warnings, errors = doOsInitForBuild()

	if errors == nil {
		errors = make([]error, 0)
	}

	if err := checkFileAccessRightsExecutable("firewallScript", firewallScript); err != nil {
		errors = append(errors, err)
	}
	if err := checkFileAccessRightsExecutable("dnsScript", dnsScript); err != nil {
		errors = append(errors, err)
	}

	return warnings, errors
}

// FirewallScript returns path to firewal script
func FirewallScript() string {
	return firewallScript
}

// DNSScript returns path to DNS script
func DNSScript() string {
	return dnsScript
}

// AllowedClients returns list of clients which are allowed to connect to daemon
// (list of absolute paths to binaries)
func implAllowedClients() []string {
	// allow connection only for IVPN UI and IVPN CLI
	return []string{"/Applications/IVPN.app/Contents/MacOS/ivpn-ui", "/Applications/IVPN.app/Contents/MacOS/cli/ivpn"}
}
