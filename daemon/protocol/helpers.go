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
	"runtime"
	"time"

	"github.com/ivpn/desktop-app/daemon/protocol/ivpnclient"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/version"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

func (p *Protocol) createSettingsResponse() *types.SettingsResp {
	prefs := p._service.Preferences()
	return &types.SettingsResp{
		IsAutoconnectOnLaunch:       prefs.IsAutoconnectOnLaunch,
		IsAutoconnectOnLaunchDaemon: prefs.IsAutoconnectOnLaunchDaemon,
		UserDefinedOvpnFile:         platform.OpenvpnUserParamsFile(),
		UserPrefs:                   prefs.UserPrefs,
		WiFi:                        prefs.WiFiControl,
		IsLogging:                   prefs.IsLogging,
		AntiTracker:                 p._service.GetSettingsAntiTracker(),
		// TODO: implement the rest of daemon settings
	}
}

func (p *Protocol) createHelloResponse() *types.HelloResp {
	prefs := p._service.Preferences()

	disabledFuncs := p._service.GetDisabledFunctions()

	dnsOverHttps, dnsOverTls, err := dns.EncryptionAbilities()
	if err != nil {
		dnsOverHttps = false
		dnsOverTls = false
		log.Error(err)
	}

	// send back Hello message with account session info
	helloResp := types.HelloResp{
		ParanoidMode: types.ParanoidModeStatus{IsEnabled: p._eaa.IsEnabled()},
		HelloResp: ivpnclient.HelloResp{
			Version:       version.Version(),
			ProcessorArch: runtime.GOARCH,
		},
		Session:             types.CreateSessionResp(prefs.Session),
		Account:             prefs.Account,
		SettingsSessionUUID: prefs.SettingsSessionUUID,
		DisabledFunctions:   disabledFuncs,
		Dns: types.DnsAbilities{
			CanUseDnsOverTls:   dnsOverTls,
			CanUseDnsOverHttps: dnsOverHttps,
		},
		DaemonSettings: *p.createSettingsResponse(),
	}
	return &helloResp
}

func (p *Protocol) createConnectedResponse(state vpn.StateInfo) *types.ConnectedResp {
	ipv6 := ""
	if state.ClientIPv6 != nil {
		ipv6 = state.ClientIPv6.String()
	}

	pausedTill := p._service.PausedTill()
	pausedTillStr := pausedTill.Format(time.RFC3339)
	if pausedTill.IsZero() {
		pausedTillStr = ""
	}

	ret := &types.ConnectedResp{
		ConnectedResp: ivpnclient.ConnectedResp{
			TimeSecFrom1970: state.Time,
			ClientIP:        state.ClientIP.String(),
			ClientIPv6:      ipv6,
			ServerIP:        state.ServerIP.String(),
			ServerPort:      state.ServerPort,
			IsTCP:           state.IsTCP,
			Mtu:             state.Mtu,
			IsPaused:        p._service.IsPaused(),
			PausedTill:      pausedTillStr,
			VpnType:         ivpnclient.VpnType(state.VpnType),
		},

		ExitHostname: state.ExitHostname,
		Dns:          p.createDnsStatus(),

		V2RayProxy: state.V2RayProxy,
		Obfsproxy:  state.Obfsproxy,
	}

	return ret
}

func (p *Protocol) createDnsStatus() types.DnsStatus {
	return types.DnsStatus{
		Dns:                p._service.GetSettingsManualDNS(),
		AntiTrackerStatus:  p._service.GetSettingsAntiTracker(),
		TempPrioritizedDns: p._service.GetSettingsTempPrioritizedDNS(),
	}
}

func (p *Protocol) createAlternateDNSResponse() *types.SetAlternateDNSResp {
	return &types.SetAlternateDNSResp{Dns: p.createDnsStatus()}
}
