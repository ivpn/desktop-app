//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
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

package dns

import (
	"fmt"
	"net"

	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/shell"
)

// implInitialise doing initialisation stuff (called on application start)
func implInitialise() error {
	return nil
}

func implPause() error {
	err := shell.Exec(log, platform.DNSScript(), "-pause")
	if err != nil {
		return fmt.Errorf("DNS pause: Failed to change DNS: %w", err)
	}
	return nil
}

func implResume() error {
	err := shell.Exec(log, platform.DNSScript(), "-resume")
	if err != nil {
		return fmt.Errorf("DNS resume: Failed to change DNS: %w", err)
	}

	return nil
}

// Set manual DNS.
// 'addr' parameter - DNS IP value
// 'localInterfaceIP' - not in use for macOS implementation
func implSetManual(addr net.IP, localInterfaceIP net.IP) error {
	err := shell.Exec(log, platform.DNSScript(), "-set_alternate_dns", addr.String())
	if err != nil {
		return fmt.Errorf("set manual DNS: Failed to change DNS: %w", err)
	}

	return nil
}

// DeleteManual - reset manual DNS configuration to default (DHCP)
// 'localInterfaceIP' (obligatory only for Windows implementation) - local IP of VPN interface
func implDeleteManual(localInterfaceIP net.IP) error {
	err := shell.Exec(log, platform.DNSScript(), "-delete_alternate_dns")
	if err != nil {
		return fmt.Errorf("reset manual DNS: Failed to change DNS: %w", err)
	}

	return nil
}
