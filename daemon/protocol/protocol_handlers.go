//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
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

package protocol

import (
	"net"

	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
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

// OnDNSChanged - DNS changed handler
func (p *Protocol) OnDNSChanged(dns net.IP) {
	// notify all clients
	if dns == nil {
		p.notifyClients(&types.SetAlternateDNSResp{IsSuccess: true, ChangedDNS: ""})
	} else {
		p.notifyClients(&types.SetAlternateDNSResp{IsSuccess: true, ChangedDNS: dns.String()})
	}
}

// OnKillSwitchStateChanged - Firewall change handler
func (p *Protocol) OnKillSwitchStateChanged() {
	// notify all clients about KillSwitch status
	if isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast, isAllowApiServers, err := p._service.KillSwitchState(); err != nil {
		log.Error(err)
	} else {
		p.notifyClients(&types.KillSwitchStatusResp{IsEnabled: isEnabled, IsPersistent: isPersistant, IsAllowLAN: isAllowLAN, IsAllowMulticast: isAllowLanMulticast, IsAllowApiServers: isAllowApiServers})
	}
}

// OnWiFiChanged - handler of WiFi status change. Notifying clients.
func (p *Protocol) OnWiFiChanged(ssid string, isInsecureNetwork bool) {
	p.notifyClients(&types.WiFiCurrentNetworkResp{
		SSID:              ssid,
		IsInsecureNetwork: isInsecureNetwork})
}

// OnPingStatus - servers ping status
func (p *Protocol) OnPingStatus(retMap map[string]int) {
	var results []types.PingResultType
	for k, v := range retMap {
		results = append(results, types.PingResultType{Host: k, Ping: v})
	}
	p.notifyClients(&types.PingServersResp{PingResults: results})
}

func (p *Protocol) OnServersUpdated(serv *apitypes.ServersInfoResponse) {
	if serv == nil {
		return
	}
	p.notifyClients(&types.ServerListResp{VpnServers: *serv})
}

func (p *Protocol) OnSplitTunnelConfigChanged() {
	var prefs = p._service.Preferences()
	p.notifyClients(&types.SplitTunnelConfig{IsEnabled: prefs.IsSplitTunnel, SplitTunnelApps: prefs.SplitTunnelApps})
}
