//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2022 Privatus Limited.
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

package types

import (
	api_types "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

type ServerSelectionEnum int

const (
	Default ServerSelectionEnum = iota // Server is manually defined
	Fastest ServerSelectionEnum = iota // Fastest server in use (only for 'Entry' server)
	Random  ServerSelectionEnum = iota // Random server in use
)

type ConnectMetadata struct {
	// How the entry server was chosen
	ServerSelectionEntry ServerSelectionEnum
	// How the exit server was chosen ('Fastest' is not applicable for 'Exit' server)
	ServerSelectionExit ServerSelectionEnum
}

// Connect request to establish new VPN connection
type ConnectionParams struct {
	Metadata ConnectMetadata

	// Can use IPv6 connection inside tunnel
	// The hosts which support IPv6 have higher priority,
	// but if there are no IPv6 hosts - we will use the IPv4 host.
	IPv6 bool
	// Use ONLY IPv6 hosts (ignored when IPv6!=true)
	IPv6Only  bool
	VpnType   vpn.Type
	ManualDNS dns.DnsSettings

	// Enable firewall before connection
	// (if true - the parameter 'firewallDuringConnection' will be ignored)
	FirewallOn bool
	// Enable firewall before connection and disable after disconnection
	// (has effect only if Firewall not enabled before)
	FirewallOnDuringConnection bool

	WireGuardParameters struct {
		// Port in use only for Single-Hop connections
		Port struct {
			Port int
		}

		EntryVpnServer struct {
			Hosts []api_types.WireGuardServerHostInfo
		}

		MultihopExitServer MultiHopExitServer_WireGuard

		Mtu int // Set 0 to use default MTU value
	}

	OpenVpnParameters struct {
		EntryVpnServer struct {
			Hosts []api_types.OpenVPNServerHostInfo
		}

		MultihopExitServer MultiHopExitServer_OpenVpn

		Proxy struct {
			Type     string
			Address  string
			Port     int
			Username string
			Password string
		}

		Port struct {
			Protocol int
			// Port number in use only for Single-Hop connections
			Port int
		}
	}
}

type MultiHopExitServer_WireGuard struct {
	// ExitSrvID (geteway ID) just in use to keep clients notified about connected MH exit server
	// Example: "gateway":"zz.wg.ivpn.net" => "zz"
	ExitSrvID string
	Hosts     []api_types.WireGuardServerHostInfo
}

type MultiHopExitServer_OpenVpn struct {
	// ExitSrvID (gateway ID) just in use to keep clients notified about connected MH exit server
	// Example: "gateway":"zz.wg.ivpn.net" => "zz"
	ExitSrvID string
	Hosts     []api_types.OpenVPNServerHostInfo
}
