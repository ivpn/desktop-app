//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
//
//  This file is part of the IVPN command line interface.
//
//  The IVPN command line interface is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The IVPN command line interface is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the IVPN command line interface. If not, see <https://www.gnu.org/licenses/>.
//

package commands

import (
	"fmt"
	"strings"

	"github.com/ivpn/desktop-app/cli/flags"
	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/service/srverrors"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

type CmdState struct {
	flags.CmdInfo
}

func (c *CmdState) Init() {
	c.Initialize("status", "Prints full info about IVPN state")
}
func (c *CmdState) Run() error {
	return showState()
}

func showState() error {
	fwstate, err := _proto.FirewallStatus()
	if err != nil {
		return err
	}

	state, connected, err := _proto.GetVPNState()
	if err != nil {
		return err
	}

	stStatus, err := _proto.GetSplitTunnelStatus()
	if err != nil {
		return err
	}

	serverInfo := ""
	exitServerInfo := ""

	var servers apitypes.ServersInfoResponse
	if state == vpn.CONNECTED {
		servers, err = _proto.GetServers()
		if err == nil {
			slist := serversListByVpnType(servers, connected.VpnType)

			serverInfo = getServerInfoByIP(slist, connected.ServerIP)
			exitServerInfo = getServerInfoByHostName(slist, connected.ExitHostname)
		}
	}

	w := printAccountInfo(nil, _proto.GetHelloResponse().Session.AccountID)
	printState(w, state, connected, serverInfo, exitServerInfo, _proto.GetHelloResponse())
	if state == vpn.CONNECTED {
		printDNSState(w, connected.Dns, &servers)
	}
	if !stStatus.IsFunctionalityNotAvailable {
		printSplitTunState(w, true, false, stStatus.IsEnabled, stStatus.SplitTunnelApps, stStatus.RunningApps)
	}
	printFirewallState(w, fwstate.IsEnabled, fwstate.IsPersistent, fwstate.IsAllowLAN, fwstate.IsAllowMulticast, fwstate.IsAllowApiServers, fwstate.UserExceptions, &state)
	w.Flush()

	// TIPS
	tips := make([]TipType, 0, 3)
	if len(_proto.GetHelloResponse().Session.Session) == 0 {
		tips = append(tips, TipLogin)
	}
	if state == vpn.CONNECTED {
		tips = append(tips, TipDisconnect)
		if !fwstate.IsEnabled {
			tips = append(tips, TipFirewallEnable)
		}
	} else if fwstate.IsEnabled {
		tips = append(tips, TipFirewallDisable)
	}
	if len(tips) > 0 {
		PrintTips(tips)
	}

	if len(_proto.GetHelloResponse().Session.Session) == 0 {
		return srverrors.ErrorNotLoggedIn{}
	}
	return nil
}

func getServerInfoByIP(servers []serverDesc, ip string) string {
	ip = strings.TrimSpace(ip)
	for _, s := range servers {
		for _, h := range s.hosts {
			if ip == strings.TrimSpace(h.host) {
				return ConnectedServerInfo(s, h)
			}
		}
	}
	return ""
}

func getServerInfoByHostName(servers []serverDesc, hostname string) string {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if len(hostname) == 0 {
		return ""
	}

	for _, s := range servers {
		for _, h := range s.hosts {
			hn := strings.ToLower(strings.TrimSpace(h.hostname))
			if hn == hostname {
				return ConnectedServerInfo(s, h)
			}
		}
	}
	return ""
}

func ConnectedServerInfo(s serverDesc, host hostDesc) string {
	return fmt.Sprintf("%s [%s], %s (%s), %s", s.gateway, host.hostname, s.city, s.countryCode, s.country)
}
