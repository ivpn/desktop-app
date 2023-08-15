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

//go:build darwin
// +build darwin

package service

import (
	"fmt"
	"net"

	protocolTypes "github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/firewall"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
)

func (s *Service) implIsCanApplyUserPreferences(userPrefs preferences.UserPreferences) error {
	return nil
}

func (s *Service) implGetDisabledFuncForPlatform() protocolTypes.DisabledFunctionalityForPlatform {
	return protocolTypes.DisabledFunctionalityForPlatform{}
}

func (s *Service) implPingServersStarting(hosts []net.IP) error {
	const onlyForICMP = true
	const isPersistent = false
	return firewall.AddHostsToExceptions(hosts, onlyForICMP, isPersistent)
}
func (s *Service) implPingServersStopped(hosts []net.IP) error {
	const onlyForICMP = true
	const isPersistent = false
	return firewall.RemoveHostsFromExceptions(hosts, onlyForICMP, isPersistent)
}

func (s *Service) implSplitTunnelling_AddApp(binaryFile string) (requiredCmdToExec string, isAlreadyRunning bool, err error) {
	// Split Tunneling is not implemented for macOS
	return "", false, nil
}
func (s *Service) implSplitTunnelling_RemoveApp(pid int, binaryPath string) (err error) {
	// Split Tunneling is not implemented for macOS
	return nil
}
func (s *Service) implSplitTunnelling_AddedPidInfo(pid int, exec string, cmdToExecute string) error {
	return fmt.Errorf("function not applicable for this platform")
}
func (s *Service) implGetDiagnosticExtraInfo() (string, error) {
	ifconfig := s.diagnosticGetCommandOutput("ifconfig")
	netstat := s.diagnosticGetCommandOutput("netstat", "-nr")
	scutil := s.diagnosticGetCommandOutput("scutil", "--dns")

	return fmt.Sprintf("%s\n%s\n%s", ifconfig, netstat, scutil), nil
}
