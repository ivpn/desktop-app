//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
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
	"strings"

	"github.com/ivpn/desktop-app-cli/flags"
	apitypes "github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/service"
	"github.com/ivpn/desktop-app-daemon/vpn"
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

	serverInfo := ""
	exitServerInfo := ""

	var servers apitypes.ServersInfoResponse
	if state == vpn.CONNECTED {
		servers, err = _proto.GetServers()
		if err == nil {
			slist := serversListByVpnType(servers, connected.VpnType)

			serverInfo = getServerInfoByIP(slist, connected.ServerIP)
			exitServerInfo = getServerInfoByID(slist, connected.ExitServerID)
		}
	}

	w := printAccountInfo(nil, _proto.GetHelloResponse().Session.AccountID)
	printState(w, state, connected, serverInfo, exitServerInfo)
	if state == vpn.CONNECTED {
		printDNSState(w, connected.ManualDNS, &servers)
	}
	printFirewallState(w, fwstate.IsEnabled, fwstate.IsPersistent, fwstate.IsAllowLAN, fwstate.IsAllowMulticast)
	w.Flush()

	// TIPS
	tips := make([]TipType, 0, 3)
	if len(_proto.GetHelloResponse().Session.Session) == 0 {
		tips = append(tips, TipLogin)
	}
	if state == vpn.CONNECTED {
		tips = append(tips, TipDisconnect)
		if fwstate.IsEnabled == false {
			tips = append(tips, TipFirewallEnable)
		}
	} else if fwstate.IsEnabled {
		tips = append(tips, TipFirewallDisable)
	}
	if len(tips) > 0 {
		PrintTips(tips)
	}

	if len(_proto.GetHelloResponse().Session.Session) == 0 {
		return service.ErrorNotLoggedIn{}
	}
	return nil
}

func getServerInfoByIP(servers []serverDesc, ip string) string {
	ip = strings.TrimSpace(ip)
	for _, s := range servers {
		for h := range s.hosts {
			if ip == strings.TrimSpace(h) {
				return s.String()
			}
		}
	}
	return ""
}

func getServerInfoByID(servers []serverDesc, id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	if len(id) == 0 {
		return ""
	}

	for _, s := range servers {
		sID := strings.ToLower(strings.TrimSpace(s.gateway))
		if strings.HasPrefix(sID, id) {
			return s.String()
		}
	}
	return ""
}
