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
	"errors"
	"fmt"
	"os/user"
	"strconv"
	"sync"

	"github.com/ivpn/desktop-app/daemon/oshelpers/macos/darwinhelpers"
	"github.com/ivpn/desktop-app/daemon/service/firewall"
	"github.com/ivpn/desktop-app/daemon/shell"
)

// On macOS, Split Tunnel is implemented by a NETransparentProxyProvider system extension
// driven from the Electron main process (ui/addons/split-tunnel-macos): the
// OSSystemExtensionManager/NETransparentProxyManager APIs require a user-session process
// inside an app bundle, which the root daemon is not. The extension receives its
// configuration through the cross-platform SplitTunnelStatus. The daemon only validates
// the OS version and adjusts the firewall rules (firewall.ApplySplitTunnelRouting).

// The Split Tunnel extension changes its group to this one on start. It is the only
// way for the firewall to distinguish the traffic relayed by the extension (which has
// to leave over the physical interface) from the traffic of any other process.
const extensionGroupName = "ivpn-st"

// Split Tunnel needs macOS 12 (Monterey). GetOsMajorVersion() reports the Darwin
// kernel major, which is 21 for macOS 12.
const minDarwinMajorVersion = 21

var (
	mutexMac sync.Mutex

	osVersionError error // set once by implInitialize(), nil if the OS is new enough

	// GID of 'extensionGroupName' (0 if the group is not available)
	extensionGroupId int

	// Inverse mode is not implemented on macOS.
	inverseModeNotAvailableError = errors.New("Inverse Split Tunnel mode is not supported on macOS")
)

func implInitialize() error {
	mutexMac.Lock()
	defer mutexMac.Unlock()

	majorVer, err := darwinhelpers.GetOsMajorVersion()
	if err != nil {
		osVersionError = fmt.Errorf("Split Tunnel: unable to determine macOS version: %w", err)
		return osVersionError
	}
	if majorVer < minDarwinMajorVersion {
		osVersionError = fmt.Errorf("Split Tunnel requires macOS 12 or later (detected Darwin kernel version %d)", majorVer)
		return osVersionError
	}
	osVersionError = nil

	// Not a fatal error: without the group the firewall can not tell the extension's traffic
	// apart, so the intentional-routing rules just stay applied (excluded apps keep using the
	// tunnel instead of bypassing it) - which is the safe fallback.
	if extensionGroupId, err = ensureExtensionGroup(); err != nil {
		log.Error(fmt.Errorf("Split Tunnel: unable to prepare the '%s' group: %w", extensionGroupName, err))
	}

	return nil
}

// ensureExtensionGroup creates 'extensionGroupName' (if not created yet) and returns its GID.
func ensureExtensionGroup() (int, error) {
	if gid, err := extensionGroupIdByName(); err == nil {
		return gid, nil
	}

	if err := shell.Exec(nil, "/usr/sbin/dseditgroup", "-o", "create", "-q", extensionGroupName); err != nil {
		return 0, fmt.Errorf("failed to create group: %w", err)
	}

	return extensionGroupIdByName()
}

func extensionGroupIdByName() (int, error) {
	group, err := user.LookupGroup(extensionGroupName)
	if err != nil {
		return 0, err
	}
	gid, err := strconv.Atoi(group.Gid)
	if err != nil || gid <= 0 {
		return 0, fmt.Errorf("unexpected GID value '%s'", group.Gid)
	}
	return gid, nil
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

// The only daemon-side effect of the configuration is the firewall rule set for the
// extension's traffic (see firewall_darwin.go); the extension itself is configured by the UI.
func implApplyConfig(isStEnabled, isStInversed, isStInverseAllowWhenNoVpn, isVpnEnabled bool, addrConfig ConfigAddresses, splitTunnelApps []string) error {
	mutexMac.Lock()
	stGroupId := 0
	if isStEnabled {
		stGroupId = extensionGroupId
	}
	mutexMac.Unlock()

	return firewall.ApplySplitTunnelRouting(stGroupId)
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
