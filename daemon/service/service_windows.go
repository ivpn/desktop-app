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

//go:build windows
// +build windows

package service

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	protocolTypes "github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
)

func (s *Service) implIsCanApplyUserPreferences(userPrefs preferences.UserPreferences) error {
	return nil
}

func (s *Service) implGetDisabledFuncForPlatform() protocolTypes.DisabledFunctionalityForPlatform {
	return protocolTypes.DisabledFunctionalityForPlatform{}
}

func (s *Service) implPingServersStarting(hosts []net.IP) error {
	// nothing to do for Windows implementation
	// firewall configured to allow all connectivity for service
	return nil
}
func (s *Service) implPingServersStopped(hosts []net.IP) error {
	// nothing to do for Windows implementation
	// firewall configured to allow all connectivity for service
	return nil
}

func (s *Service) implSplitTunnelling_AddApp(binaryFile string) (requiredCmdToExec string, isAlreadyRunning bool, err error) {
	binaryFile = strings.TrimSpace(binaryFile)
	if len(binaryFile) <= 0 {
		return "", false, nil
	}

	prefs := s._preferences
	// current binary folder path
	var exeDir string
	if ex, err := os.Executable(); err == nil && len(ex) > 0 {
		exeDir = filepath.Dir(ex)
	}

	// Ensure no binaries from IVPN package is included into apps list to Split-Tunnel
	if strings.HasPrefix(binaryFile, exeDir) {
		return "", false, fmt.Errorf("Split-Tunnelling for IVPN binaries is forbidden (%s)", binaryFile)
	}
	// Ensure file is exists
	if _, err := os.Stat(binaryFile); os.IsNotExist(err) {
		return "", false, err
	}

	binaryPathLowCase := strings.ToLower(binaryFile)
	for _, a := range prefs.SplitTunnelApps {
		if strings.ToLower(a) == binaryPathLowCase {
			// the binary is already in configuration
			return "", false, nil
		}
	}

	prefs.SplitTunnelApps = append(prefs.SplitTunnelApps, binaryFile)
	s.setPreferences(prefs)

	return "", false, nil
}

func (s *Service) implSplitTunnelling_RemoveApp(pid int, binaryPath string) (err error) {
	binaryPath = strings.TrimSpace(binaryPath)
	if len(binaryPath) <= 0 {
		return nil
	}

	prefs := s._preferences
	newStApps := make([]string, 0, len(prefs.SplitTunnelApps))
	binaryPathLowCase := strings.ToLower(binaryPath)

	for _, a := range prefs.SplitTunnelApps {
		if strings.ToLower(a) == binaryPathLowCase {
			continue
		}
		newStApps = append(newStApps, a)
	}

	prefs.SplitTunnelApps = newStApps
	s.setPreferences(prefs)

	return nil
}

func (s *Service) implSplitTunnelling_AddedPidInfo(pid int, exec string, cmdToExecute string) error {
	return fmt.Errorf("function not applicable for this platform")
}

func (s *Service) implGetDiagnosticExtraInfo() (string, error) {
	ifconfig := s.diagnosticGetCommandOutput("ipconfig", "/all")
	route := s.diagnosticGetCommandOutput("route", "print")

	return fmt.Sprintf("%s\n%s", ifconfig, route), nil
}
