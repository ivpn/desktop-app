//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2021 IVPN Limited.
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

func implInitialize() error {
	return fmt.Errorf("Split-Tunnelling is not implemented for macOS")
}

func implFuncNotAvailableError() error {
	return fmt.Errorf("Split-Tunnelling is not implemented for macOS")
}

func implReset() error {
	return fmt.Errorf("Split-Tunnelling is not implemented for macOS")
}

func implApplyConfig(isStEnabled bool, isVpnEnabled bool, addrConfig ConfigAddresses, splitTunnelApps []string) error {
	return fmt.Errorf("Split-Tunnelling is not implemented for macOS")
}

func implAddPid(pid int, commandToExecute string) error {
	return fmt.Errorf("Split-Tunnelling is not implemented for macOS")
}

func implRemovePid(pid int) error {
	return fmt.Errorf("Split-Tunnelling is not implemented for macOS")
}

func implGetRunningApps() ([]RunningApp, error) {
	return nil, fmt.Errorf("Split-Tunnelling is not implemented for macOS")
}
