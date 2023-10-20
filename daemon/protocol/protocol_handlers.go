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

package protocol

import (
	api_types "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	"github.com/ivpn/desktop-app/daemon/wifiNotifier"
)

// OnServiceSessionChanged - SessionChanged handler
func (p *Protocol) OnServiceSessionChanged() {
	// send back Hello message with account session info
	helloResp := p.createHelloResponse()
	p.notifyClients(helloResp)
}

// OnAccountStatus - handler of account status info. Notifying clients.
func (p *Protocol) OnAccountStatus(sessionToken string, accountInfo preferences.AccountStatus) {
	if len(sessionToken) == 0 {
		return
	}

	p.notifyClients(&types.AccountStatusResp{
		SessionToken: sessionToken,
		Account:      accountInfo})
}

// OnKillSwitchStateChanged - Firewall change handler
func (p *Protocol) OnKillSwitchStateChanged() {
	if p._service == nil {
		return
	}

	// notify all clients about KillSwitch status
	if status, err := p._service.KillSwitchState(); err != nil {
		log.Error(err)
	} else {
		p.notifyClients(&types.KillSwitchStatusResp{KillSwitchStatus: status})
	}
}

// OnWiFiChanged - handler of WiFi status change. Notifying clients.
func (p *Protocol) OnWiFiChanged(info wifiNotifier.WifiInfo) {
	p.notifyClients(&types.WiFiCurrentNetworkResp{
		SSID:              info.SSID,
		IsInsecureNetwork: info.IsInsecure})
}

// OnPingStatus - servers ping status
func (p *Protocol) OnPingStatus(retMap map[string]int) {
	var results []types.PingResultType
	for k, v := range retMap {
		results = append(results, types.PingResultType{Host: k, Ping: v})
	}
	p.notifyClients(&types.PingServersResp{PingResults: results})
}

func (p *Protocol) OnServersUpdated(serv *api_types.ServersInfoResponse) {
	if serv == nil {
		return
	}
	p.notifyClients(&types.ServerListResp{VpnServers: *serv})
}

func (p *Protocol) OnSplitTunnelStatusChanged() {
	if p._service == nil {
		return
	}
	status, err := p._service.SplitTunnelling_GetStatus()
	if err != nil {
		return
	}
	p.notifyClients(&status)
}
