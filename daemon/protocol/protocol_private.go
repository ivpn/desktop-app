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
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/version"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

// ----------------------------------------------------------------------
func getConnectionName(c net.Conn) string {
	return strings.TrimSpace(strings.Replace(c.RemoteAddr().String(), "127.0.0.1:", "", 1))
}

func (p *Protocol) connLogID(c net.Conn) string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s ", getConnectionName(c))
}

// -------------- send message to all active connections ---------------
func (p *Protocol) notifyClients(cmd types.ICommandBase) {
	p._connectionsMutex.RLock()
	defer p._connectionsMutex.RUnlock()
	for conn := range p._connections {
		p.sendResponse(conn, cmd, 0)
	}
}

// -------------- clients connections ---------------
// IsClientConnected checks is any authenticated connection available of specific client type
func (p *Protocol) IsClientConnected(checkOnlyUiClients bool) bool {
	p._connectionsMutex.RLock()
	defer p._connectionsMutex.RUnlock()

	for _, val := range p._connections {
		if val.IsAuthenticated {
			if checkOnlyUiClients {
				if val.Type == types.ClientUi {
					return true
				}
			} else {
				return true
			}
		}
	}
	return false
}

// IsCanDoBackgroundAction returns 'false' when no background action allowed (e.g. EAA enabled but no authenticated clients connected)
func (p *Protocol) IsCanDoBackgroundAction() bool {
	if p._eaa.IsEnabled() {
		const checkOnlyUiClients = true
		return p.IsClientConnected(checkOnlyUiClients)
	}
	return true
}

func (p *Protocol) clientConnected(c net.Conn, cType types.ClientTypeEnum) {
	p._connectionsMutex.Lock()
	defer p._connectionsMutex.Unlock()
	p._connections[c] = connectionInfo{Type: cType}
}

func (p *Protocol) clientDisconnected(c net.Conn) (disconnectedClientInfo *connectionInfo) {
	p._connectionsMutex.Lock()
	defer p._connectionsMutex.Unlock()

	if ci, ok := p._connections[c]; ok {
		disconnectedClientInfo = &ci
	}

	delete(p._connections, c)
	c.Close()

	return disconnectedClientInfo
}

func (p *Protocol) clientsConnectedCount() int {
	p._connectionsMutex.Lock()
	defer p._connectionsMutex.Unlock()
	return len(p._connections)
}

// Notifying clients "service is going to stop" (client application (UI) will close)
// Closing and erasing all clients connections
func (p *Protocol) notifyClientsDaemonExiting() {
	func() {
		p._connectionsMutex.RLock()
		defer p._connectionsMutex.RUnlock()

		if len(p._connections) > 0 {
			log.Info("Notifying clients: 'daemon is stopping'...")
		}

		for conn := range p._connections {
			// notifying client "service is going to stop" (client application (UI) will close)
			p.sendResponse(conn, &types.ServiceExitingResp{}, 0)
			// closing current connection with a client
			conn.Close()
		}
	}()

	// erasing clients connections
	p._connectionsMutex.Lock()
	defer p._connectionsMutex.Unlock()
	p._connections = make(map[net.Conn]connectionInfo)
}

func (p *Protocol) clientSetAuthenticated(c net.Conn) {
	// contains information about just connected client (first authentication) or nil
	var justConnectedClientInfo *connectionInfo

	// separate anonymous function for correct mutex unlock
	func() {
		p._connectionsMutex.Lock()
		defer p._connectionsMutex.Unlock()

		if cInfo, ok := p._connections[c]; ok {
			if !cInfo.IsAuthenticated {
				cInfo.IsAuthenticated = true
				p._connections[c] = cInfo

				justConnectedClientInfo = &cInfo
			}
		}
	}()

	if justConnectedClientInfo != nil {
		go func() {
			p._service.OnAuthenticatedClient(justConnectedClientInfo.Type)
		}()
	}

	if len(p._lastConnectionErrorToNotifyClient) > 0 {
		log.Info("Sending delayed error to client: ", p._lastConnectionErrorToNotifyClient)
		delayedErr := types.ErrorRespDelayed{}
		delayedErr.ErrorMessage = p._lastConnectionErrorToNotifyClient
		p.sendResponse(c, &delayedErr, 0)
	}
	p._lastConnectionErrorToNotifyClient = ""
}

// -------------- sending responses ---------------
func (p *Protocol) sendError(conn net.Conn, errorText string, cmdIdx int) {
	log.Error(errorText)
	p.sendResponse(conn, &types.ErrorResp{ErrorMessage: errorText}, cmdIdx)
}

func (p *Protocol) sendErrorResponse(conn net.Conn, request types.RequestBase, err error) {
	log.Error(fmt.Sprintf("%sError processing request '%s': %s", p.connLogID(conn), request.Command, err))
	p.sendResponse(conn, &types.ErrorResp{ErrorMessage: helpers.CapitalizeFirstLetter(err.Error())}, request.Idx)
}

func (p *Protocol) sendResponse(conn net.Conn, cmd types.ICommandBase, idx int) (retErr error) {
	if conn == nil {
		return fmt.Errorf("%sresponse not sent (no connection to client)", p.connLogID(conn))
	}

	if err := types.Send(conn, cmd, idx); err != nil {
		return fmt.Errorf("%sfailed to send command: %w", p.connLogID(conn), err)
	}

	// Just for logging
	if reqType := types.GetTypeName(cmd); len(reqType) > 0 {
		log.Info(fmt.Sprintf("[-->] %s", p.connLogID(conn)), reqType, fmt.Sprintf(" [%d]", idx), " ", cmd.LogExtraInfo())
	} else {
		return fmt.Errorf("%sprotocol error: BAD DATA SENT", p.connLogID(conn))
	}

	return nil
}

// -------------- Initialize response objects ---------------
func (p *Protocol) createSettingsResponse() *types.SettingsResp {
	prefs := p._service.Preferences()
	return &types.SettingsResp{
		IsAutoconnectOnLaunch:       prefs.IsAutoconnectOnLaunch,
		IsAutoconnectOnLaunchDaemon: prefs.IsAutoconnectOnLaunchDaemon,
		UserDefinedOvpnFile:         platform.OpenvpnUserParamsFile(),
		UserPrefs:                   prefs.UserPrefs,
		WiFi:                        prefs.WiFiControl,
		IsLogging:                   prefs.IsLogging,
		AntiTracker:                 p._service.GetAntiTrackerStatus(),
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
		ParanoidMode:        types.ParanoidModeStatus{IsEnabled: p._eaa.IsEnabled()},
		Version:             version.Version(),
		ProcessorArch:       runtime.GOARCH,
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

	manualDns := dns.GetLastManualDNS()

	ret := &types.ConnectedResp{
		TimeSecFrom1970: state.Time,
		ClientIP:        state.ClientIP.String(),
		ClientIPv6:      ipv6,
		ServerIP:        state.ServerIP.String(),
		ServerPort:      state.ServerPort,
		VpnType:         state.VpnType,
		ExitHostname:    state.ExitHostname,
		Dns:             types.DnsStatus{Dns: manualDns, AntiTrackerStatus: p._service.GetAntiTrackerStatus()},
		IsTCP:           state.IsTCP,
		Mtu:             state.Mtu,
		V2RayProxy:      state.V2RayProxy,
		Obfsproxy:       state.Obfsproxy,
		IsPaused:        p._service.IsPaused(),
		PausedTill:      pausedTillStr,
	}

	return ret
}
