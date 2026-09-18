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
	"os"
	"strings"

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

// macOS is path-based, like Windows: it can exclude already-running apps by
// executable/bundle path, unlike Linux's launch-based (cgroup) model.

// macOwnAppBundlePathPrefix mirrors the internal bypass list hardcoded in the
// system extension itself (kInternalBypassPathPrefix in STProxyProvider.m) -
// the extension enforces this unconditionally regardless of what's stored
// here; this check just keeps the daemon from offering it in the first place.
const macOwnAppBundlePathPrefix = "/Applications/IVPN.app"

// macAppBundlePath walks up from path to the nearest ancestor ending in
// ".app", if any - so passing either the bundle itself or a path to an inner
// executable (e.g. "Contents/MacOS/firefox") resolves to the same bundle path.
// A bare executable with no ".app" ancestor (e.g. "/usr/local/bin/mytool") is
// returned unchanged, matching the extension's own exact-path fallback.
func macAppBundlePath(path string) string {
	for dir := path; len(dir) > 1 && dir != "/" && dir != "."; {
		if strings.HasSuffix(dir, ".app") {
			return dir
		}
		parent := dir[:strings.LastIndex(dir, "/")]
		if len(parent) == 0 || parent == dir {
			break
		}
		dir = parent
	}
	return path
}

func (s *Service) implSplitTunnelling_AddApp(binaryFile string) (requiredCmdToExec string, isAlreadyRunning bool, err error) {
	binaryFile = strings.TrimSpace(binaryFile)
	// Strip a single surrounding quote pair (e.g. paths with spaces sent by CLI as "path to binary")
	if len(binaryFile) >= 2 && binaryFile[0] == '"' && binaryFile[len(binaryFile)-1] == '"' {
		binaryFile = binaryFile[1 : len(binaryFile)-1]
	}
	if len(binaryFile) == 0 {
		return "", false, nil
	}

	// Store the '.app' bundle path, not an inner executable: the extension
	// matches by bundle-path prefix, so a single entry covers every
	// helper/XPC binary inside the bundle.
	binaryFile = macAppBundlePath(binaryFile)

	if strings.HasPrefix(binaryFile, macOwnAppBundlePathPrefix) {
		return "", false, fmt.Errorf("Split-Tunnelling for IVPN binaries is forbidden (%s)", binaryFile)
	}
	if _, err := os.Stat(binaryFile); os.IsNotExist(err) {
		return "", false, err
	}

	prefs := s._preferences
	for _, a := range prefs.SplitTunnelApps {
		if a == binaryFile {
			return "", false, nil // already in configuration
		}
	}
	prefs.SplitTunnelApps = append(prefs.SplitTunnelApps, binaryFile)
	s.setPreferences(prefs)

	return "", false, nil
}
func (s *Service) implSplitTunnelling_RemoveApp(pid int, binaryPath string) (err error) {
	binaryPath = strings.TrimSpace(binaryPath)
	if len(binaryPath) == 0 {
		return nil
	}

	prefs := s._preferences
	newStApps := make([]string, 0, len(prefs.SplitTunnelApps))
	for _, a := range prefs.SplitTunnelApps {
		if a == binaryPath {
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
	ifconfig, _ := s.diagnosticGetCommandOutput("ifconfig")
	netstat, _ := s.diagnosticGetCommandOutput("netstat", "-nr")
	scutil, _ := s.diagnosticGetCommandOutput("scutil", "--dns")

	return fmt.Sprintf("%s\n%s\n%s", ifconfig, netstat, scutil), nil
}
