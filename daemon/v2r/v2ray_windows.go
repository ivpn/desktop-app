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

package v2r

import (
	"fmt"
	"net"
	"os"
	"path"
	"strings"

	"github.com/ivpn/desktop-app/daemon/shell"
)

var routeBinaryPath string

func implInit() {
	envVarSystemroot := strings.ToLower(os.Getenv("SYSTEMROOT"))
	if len(envVarSystemroot) == 0 {
		log.Error("!!! ERROR !!! Unable to determine 'SYSTEMROOT' environment variable!")
	} else {
		routeBinaryPath = strings.ReplaceAll(path.Join(envVarSystemroot, "system32", "route.exe"), "/", "\\")
	}
}

func (v *V2RayWrapper) implSetMainRoute(defaultGateway net.IP) error {
	if routeBinaryPath == "" {
		return fmt.Errorf("route.exe location not specified")
	}

	remoteHost, _, err := v.getRemoteEndpoint()
	if err != nil {
		return fmt.Errorf("getting remote endpoint error : %w", err)
	}

	// route.exe add 144.217.233.114 mask 255.255.255.255 192.168.0.1
	if err := shell.Exec(log, routeBinaryPath, "add", remoteHost.String(), "mask", "255.255.255.255", defaultGateway.String()); err != nil {
		return fmt.Errorf("adding route shell comand error : %w", err)
	}

	return nil
}

func (v *V2RayWrapper) implDeleteMainRoute() error {
	if routeBinaryPath == "" {
		return fmt.Errorf("route.exe location not specified")
	}

	remoteHost, _, err := v.getRemoteEndpoint()
	if err != nil {
		return fmt.Errorf("getting remote endpoint error : %w", err)
	}

	shell.Exec(log, routeBinaryPath, "delete", remoteHost.String())
	return nil
}
