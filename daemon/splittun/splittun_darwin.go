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

package splittun

import (
	"fmt"
)

var (
	notImplementedError = fmt.Errorf("Split-Tunnelling is not implemented for this platform")
)

func implInitialize() error {
	return notImplementedError
}

func implFuncNotAvailableError() (generalStError, inversedStError error) {
	return notImplementedError, fmt.Errorf("Inversed Split-Tunnelling is not implemented for this platform")
}

func implReset() error {
	return notImplementedError
}

func implApplyConfig(isStEnabled, isStInversed, isStInverseAllowWhenNoVpn, isVpnEnabled bool, addrConfig ConfigAddresses, splitTunnelApps []string) error {
	return notImplementedError
}

func implAddPid(pid int, commandToExecute string) error {
	return notImplementedError
}

func implRemovePid(pid int) error {
	return notImplementedError
}

func implGetRunningApps() ([]RunningApp, error) {
	return nil, notImplementedError
}
