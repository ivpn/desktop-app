//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2026 IVPN Limited.
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

package splittun

import (
	"fmt"
	"sync"

	"github.com/ivpn/desktop-app/daemon/oshelpers/macos/darwinhelpers"
	"github.com/ivpn/desktop-app/daemon/service/firewall"
)

// macOS does not implement Split Tunnel in the daemon itself: the actual
// interception/relaying happens in a NETransparentProxyProvider system
// extension, driven from the Electron main process (ui/addons/split-tunnel-macos)
// because OSSystemExtensionManager/NETransparentProxyManager require a
// user-session process inside an app bundle - the root daemon cannot call
// either API. Everything the extension needs (enabled state, inverse flag,
// app list) already flows to the UI unchanged via the existing
// cross-platform SplitTunnelStatus fields, so this file has nothing left to
// resolve - it only checks OS-version availability and tells the firewall to
// adjust its intentional-routing rules (see implApplyConfig() below and
// firewall_darwin.go's ApplySplitTunnelRouting()).

var (
	mutexMac sync.Mutex

	osVersionError error // set once by implInitialize(), nil if the OS is new enough

	// Milestone 1 ships exclusion mode only - inverse mode is a separate,
	// later milestone.
	inverseModeNotAvailableError = fmt.Errorf("Inverse Split Tunnel is not yet supported on macOS")
)

func implInitialize() error {
	mutexMac.Lock()
	defer mutexMac.Unlock()

	majorVer, err := darwinhelpers.GetOsMajorVersion()
	if err != nil {
		osVersionError = fmt.Errorf("Split Tunnel: unable to determine macOS version: %w", err)
		return osVersionError
	}
	if majorVer < 12 {
		osVersionError = fmt.Errorf("Split Tunnel requires macOS 12 or later (detected major version %d)", majorVer)
		return osVersionError
	}
	osVersionError = nil

	return nil
}

func implFuncNotAvailableError() (generalStError, inversedStError error) {
	mutexMac.Lock()
	defer mutexMac.Unlock()

	// Extension/session readiness is not tracked here - it's reported by the
	// UI-side addon via SplitTunnelMacExtensionState and routed into the
	// daemon's existing SplitTunnelling_SetDisabledReason() mechanism (the
	// same one already used e.g. for the Portmaster-conflict check), so it
	// surfaces through SplitTunnelStatus.NoFuncReason instead of a second,
	// parallel availability concept here.
	return osVersionError, inverseModeNotAvailableError
}

func implReset() error {
	return nil
}

// The extension is driven entirely from the Electron main process
// (ui/addons/split-tunnel-macos), which already gets the enabled/inverse
// flags and app list via the existing SplitTunnelStatus fields - there is
// nothing left for the daemon to resolve or store here. The only real
// daemon-side effect is telling the firewall to skip its intentional-routing
// NAT/route-to rules while Split Tunnel is enabled (see firewall_darwin.go).
func implApplyConfig(isStEnabled, isStInversed, isStInverseAllowWhenNoVpn, isVpnEnabled bool, addrConfig ConfigAddresses, splitTunnelApps []string) error {
	return firewall.ApplySplitTunnelRouting(isStEnabled)
}

// Linux-only by contract - macOS is path-based (like Windows), not launch-based.
func implAddPid(pid int, commandToExecute string) error {
	return fmt.Errorf("function not applicable for this platform")
}
func implRemovePid(pid int) error {
	return fmt.Errorf("function not applicable for this platform")
}
func implGetRunningApps() ([]RunningApp, error) {
	return nil, nil
}

