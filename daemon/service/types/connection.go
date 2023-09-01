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

package types

import (
	"crypto/rand"
	"fmt"
	"math/big"

	api_types "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/obfsproxy"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/v2r"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

type ServerSelectionEnum int

const (
	Default ServerSelectionEnum = iota // Server is manually defined
	Fastest ServerSelectionEnum = iota // Fastest server in use (only for 'Entry' server)
	Random  ServerSelectionEnum = iota // Random server in use
)

type AntiTrackerMetadata struct {
	Enabled                  bool
	Hardcore                 bool
	AntiTrackerBlockListName string
}

func (a AntiTrackerMetadata) IsEnabled() bool {
	return a.Enabled
}

func (a AntiTrackerMetadata) Equal(b AntiTrackerMetadata) bool {
	return a.Enabled == b.Enabled && a.Hardcore == b.Hardcore && a.AntiTrackerBlockListName == b.AntiTrackerBlockListName
}

type ConnectMetadata struct {
	// How the entry server was chosen
	ServerSelectionEntry ServerSelectionEnum
	// How the exit server was chosen ('Fastest' is not applicable for 'Exit' server)
	ServerSelectionExit ServerSelectionEnum

	AntiTracker AntiTrackerMetadata

	// (only if Fastest server in use) List of fastest servers which must be ignored (only gateway ID in use: e.g."us-tx.wg.ivpn.net" => "us-tx")
	FastestGatewaysExcludeList []string
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
			Protocol int // by default, it must be UDP (0) for WireGuard. But for V2Ray connections it can be UDP or TCP
			Port     int
		}

		EntryVpnServer struct {
			Hosts []api_types.WireGuardServerHostInfo
		}

		MultihopExitServer MultiHopExitServer_WireGuard

		Mtu int // Set 0 to use default MTU value

		V2RayProxy v2r.V2RayTransportType // V2Ray config
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

		Obfs4proxy obfsproxy.Config       // Obfsproxy config (ignored when 'V2RayProxy' defined)
		V2RayProxy v2r.V2RayTransportType // V2Ray config (this option takes precedence over the 'Obfs4proxy')
	}
}

func (p ConnectionParams) IsMultiHop() bool {
	if p.VpnType == vpn.OpenVPN {
		return len(p.OpenVpnParameters.MultihopExitServer.Hosts) > 0
	}
	return len(p.WireGuardParameters.MultihopExitServer.Hosts) > 0
}

func (p ConnectionParams) CheckIsDefined() error {
	if p.VpnType == vpn.WireGuard {
		if len(p.WireGuardParameters.EntryVpnServer.Hosts) <= 0 {
			return fmt.Errorf("no hosts defined for WireGuard connection")
		}
	} else {
		if len(p.OpenVpnParameters.EntryVpnServer.Hosts) <= 0 {
			return fmt.Errorf("no hosts defined for OpenVPN connection")
		}
	}
	return nil
}

func (p ConnectionParams) Port() (port int, isTcp bool) {
	if p.VpnType == vpn.WireGuard {
		return p.WireGuardParameters.Port.Port, p.WireGuardParameters.Port.Protocol > 0 // is TCP
	}
	return p.OpenVpnParameters.Port.Port, p.OpenVpnParameters.Port.Protocol > 0 // is TCP
}

func (p ConnectionParams) V2Ray() v2r.V2RayTransportType {
	if p.VpnType == vpn.WireGuard {
		return p.WireGuardParameters.V2RayProxy
	}
	return p.OpenVpnParameters.V2RayProxy
}

// NormalizeHosts - normalize hosts list
// 1) in case of multiple entry hosts - take random host from the list
// 2) in case of multiple exit hosts - take random host from the list
// 3) (WireGuard) filter entry hosts: use IPv6 hosts
// 4) (WireGuard) filter exit servers (Multi-Hop connection):
// 4.1) each exit server must have initialized 'multihop_port' field
// 4.2) (in case of IPv6Only) IPv6 local address should be defined
func (p *ConnectionParams) NormalizeHosts() error {

	if vpn.Type(p.VpnType) == vpn.OpenVPN {
		// in case of multiple entry hosts - take random host from the list
		entryHosts := p.OpenVpnParameters.EntryVpnServer.Hosts
		if len(entryHosts) > 1 {
			rndHost := entryHosts[0]
			if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(entryHosts)))); err == nil {
				rndHost = entryHosts[rnd.Int64()]
			}
			p.OpenVpnParameters.EntryVpnServer.Hosts = []api_types.OpenVPNServerHostInfo{rndHost}
		}

		// in case of multiple exit hosts - take random host from the list
		exitHosts := p.OpenVpnParameters.MultihopExitServer.Hosts
		if len(exitHosts) > 1 {
			rndHost := exitHosts[0]
			if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(exitHosts)))); err == nil {
				rndHost = exitHosts[rnd.Int64()]
			}
			p.OpenVpnParameters.MultihopExitServer.Hosts = []api_types.OpenVPNServerHostInfo{rndHost}
		}

	} else if vpn.Type(p.VpnType) == vpn.WireGuard {
		// filter entry hosts: use IPv6 hosts
		if p.IPv6 {
			hosts := p.WireGuardParameters.EntryVpnServer.Hosts
			var ipv6Hosts []api_types.WireGuardServerHostInfo
			for _, h := range hosts {
				if h.IPv6.LocalIP != "" {
					ipv6Hosts = append(ipv6Hosts, h)
				}
			}
			if len(ipv6Hosts) == 0 {
				if p.IPv6Only {
					return fmt.Errorf("unable to make IPv6 connection inside tunnel. Server does not support IPv6")
				}
			} else {
				p.WireGuardParameters.EntryVpnServer.Hosts = ipv6Hosts
			}
		}

		// filter exit servers (Multi-Hop connection):
		// 1) each exit server must have initialized 'multihop_port' field
		// 2) (in case of IPv6Only) IPv6 local address should be defined
		multihopExitHosts := p.WireGuardParameters.MultihopExitServer.Hosts
		if len(multihopExitHosts) > 0 {
			isHasMHPort := false
			//filteredExitHosts := append(multihopExitHosts[0:0], multihopExitHosts...)
			var filteredExitHosts []api_types.WireGuardServerHostInfo
			for _, h := range multihopExitHosts {
				if h.MultihopPort == 0 {
					continue
				}
				isHasMHPort = true
				if p.IPv6 && h.IPv6.LocalIP == "" {
					continue
				}
				filteredExitHosts = append(filteredExitHosts, h)
			}
			if len(filteredExitHosts) == 0 {
				if !isHasMHPort {
					return fmt.Errorf("unable to make Multi-Hop connection inside tunnel. Exit server does not support Multi-Hop")
				}
				if p.IPv6Only {
					return fmt.Errorf("unable to make IPv6 Multi-Hop connection inside tunnel. Exit server does not support IPv6")
				}
			} else {
				p.WireGuardParameters.MultihopExitServer.Hosts = filteredExitHosts
			}
		}

		// in case of multiple entry hosts - take random host from the list
		if len(p.WireGuardParameters.EntryVpnServer.Hosts) > 1 {
			rndHost := p.WireGuardParameters.EntryVpnServer.Hosts[0]
			if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(p.WireGuardParameters.EntryVpnServer.Hosts)))); err == nil {
				rndHost = p.WireGuardParameters.EntryVpnServer.Hosts[rnd.Int64()]
			}
			p.WireGuardParameters.EntryVpnServer.Hosts = []api_types.WireGuardServerHostInfo{rndHost}
		}

		// in case of multiple exit hosts - take random host from the list
		if len(p.WireGuardParameters.MultihopExitServer.Hosts) > 1 {
			rndHost := p.WireGuardParameters.MultihopExitServer.Hosts[0]
			if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(p.WireGuardParameters.MultihopExitServer.Hosts)))); err == nil {
				rndHost = p.WireGuardParameters.MultihopExitServer.Hosts[rnd.Int64()]
			}
			p.WireGuardParameters.MultihopExitServer.Hosts = []api_types.WireGuardServerHostInfo{rndHost}
		}

	} else {
		return fmt.Errorf("unknown VPN type: %d", p.VpnType)
	}

	return nil
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
