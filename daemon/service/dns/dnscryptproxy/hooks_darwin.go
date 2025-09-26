//go:build darwin
// +build darwin

//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2025 IVPN Limited.
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

package dnscryptproxy

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

func init() {
	preStartHook = thePreStartHook
}

func thePreStartHook(p *DnsCryptProxy) error {
	if !p.listenAddr.IsLoopback() || p.listenAddr.IsUnspecified() {
		return nil
	}

	if p.listenAddr.Equal(net.IPv4(127, 0, 0, 1)) {
		return nil
	}

	// On macOS, if we plan to listen on 127.0.0.x (x != 1), we need to add an alias
	// to the loopback interface first
	cmd := []string{"ifconfig", "lo0", "alias", p.listenAddr.String()}

	log.Debug("Adding loopback alias: ", fmt.Sprintf("%v", cmd))
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	if output, err := execCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add loopback alias %s: %v - %s", p.listenAddr.String(), err, strings.TrimSpace(string(output)))
	}

	return nil
}
