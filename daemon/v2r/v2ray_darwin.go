//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 Privatus Limited.
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

package v2r

import (
	"fmt"
	"net"

	"github.com/ivpn/desktop-app/daemon/shell"
)

func implInit() {
	// nothing to do here for macOS
}

func (v *V2RayWrapper) implSetMainRoute(defaultGateway net.IP) error {
	remoteHost, _, err := v.getRemoteEndpoint()
	if err != nil {
		return fmt.Errorf("getting remote endpoint error : %w", err)
	}

	// ip route add 144.217.233.114/32 via 192.168.0.1 dev eth0
	if err := shell.Exec(log, "/sbin/route", "-n", "add", "-inet", "-net", remoteHost.String(), defaultGateway.String(), "255.255.255.255"); err != nil {
		return fmt.Errorf("adding route shell comand error : %w", err)
	}

	return nil
}

func (v *V2RayWrapper) implDeleteMainRoute() error {
	remoteHost, _, err := v.getRemoteEndpoint()
	if err != nil {
		return fmt.Errorf("getting remote endpoint error : %w", err)
	}

	shell.Exec(log, "/sbin/route", "-n", "delete", "-inet", "-net", remoteHost.String())
	return nil
}
