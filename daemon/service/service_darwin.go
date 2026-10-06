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
	"path/filepath"
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
	for dir := filepath.Clean(path); dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		if strings.HasSuffix(dir, ".app") {
			return dir
		}
	}
	return path
}

// isMacOwnAppBundlePath reports whether path is our own bundle or something
// inside it. The separator matters: a plain prefix test would also reject
// unrelated paths like "/Applications/IVPN.app.bak".
func isMacOwnAppBundlePath(path string) bool {
	return path == macOwnAppBundlePathPrefix ||
		strings.HasPrefix(path, macOwnAppBundlePathPrefix+"/")
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

	if isMacOwnAppBundlePath(binaryFile) {
		return "", false, fmt.Errorf("Split-Tunnelling for IVPN binaries is forbidden (%s)", binaryFile)
	}
	// Accept only an application bundle or an executable file. The extension
	// applies a prefix match to bundles, so a plain directory (e.g. "/") in the
	// list would exclude every process on the machine.
	fi, err := os.Stat(binaryFile)
	if err != nil {
		return "", false, err
	}
	isBundle := fi.IsDir() && strings.HasSuffix(binaryFile, ".app")
	isExecutable := fi.Mode().IsRegular() && fi.Mode()&0111 != 0
	if !isBundle && !isExecutable {
		return "", false, fmt.Errorf("not an application bundle or an executable file: %s", binaryFile)
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
