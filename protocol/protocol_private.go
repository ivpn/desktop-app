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

package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/version"
	"github.com/ivpn/desktop-app-daemon/vpn"
	"github.com/ivpn/desktop-app-daemon/vpn/openvpn"
	"github.com/ivpn/desktop-app-daemon/vpn/wireguard"
)

// connID returns connection info (required to destinguish communication between several connections in log)
func (p *Protocol) connLogID(c net.Conn) string {
	if c == nil {
		return ""
	}
	//return ""
	// not necessary to print additional data into a log when only one connection available
	numConnections := 0
	func() {
		p._connectionsMutex.RLock()
		defer p._connectionsMutex.RUnlock()
		numConnections = len(p._connections)
	}()
	if numConnections <= 1 {
		return ""
	}

	ret := strings.Replace(c.RemoteAddr().String(), "127.0.0.1:", "", 1)
	return fmt.Sprintf("%s ", ret)
}

// -------------- send message to all active connections ---------------
func (p *Protocol) notifyClients(cmd interface{}) {
	p._connectionsMutex.RLock()
	defer p._connectionsMutex.RUnlock()
	for conn := range p._connections {
		p.sendResponse(conn, cmd, 0)
	}
}

// -------------- clients connections ---------------
func (p *Protocol) clientConnected(c net.Conn) {
	p._connectionsMutex.Lock()
	defer p._connectionsMutex.Unlock()
	p._connections[c] = struct{}{}
}

func (p *Protocol) clientDisconnected(c net.Conn) {
	p._connectionsMutex.Lock()
	defer p._connectionsMutex.Unlock()
	if _, ok := p._connections[c]; ok {
		delete(p._connections, c)
	}
	c.Close()
}

// Notifying clients "service is going to stop" (client application (UI) will close)
// Closing and erasing all clients connections
func (p *Protocol) notifyClientsDaemonExiting() {
	func() {
		p._connectionsMutex.RLock()
		defer p._connectionsMutex.RUnlock()
		for conn := range p._connections {
			// notifying client "service is going to stop" (client application (UI) will close)
			p.sendResponse(conn, types.ServiceExitingResp{}, 0)
			// closing current connection with a client
			conn.Close()
		}
	}()

	// erasing clients connections
	p._connectionsMutex.Lock()
	defer p._connectionsMutex.Unlock()
	p._connections = make(map[net.Conn]struct{})
}

// -------------- sending responses ---------------
func (p *Protocol) sendErrorResponse(conn net.Conn, request types.CommandBase, err error) {
	log.Error("%sError processing request '%s': %s", p.connLogID(conn), request.Command, err)
	p.sendResponse(conn, &types.ErrorResp{ErrorMessage: err.Error()}, request.Idx)
}

func (p *Protocol) sendResponse(conn net.Conn, cmd interface{}, idx int) (retErr error) {
	if conn == nil {
		return fmt.Errorf("%sresponse not sent (no connection to client)", p.connLogID(conn))
	}

	if err := types.Send(conn, cmd, idx); err != nil {
		return fmt.Errorf("%sfailed to send command: %w", p.connLogID(conn), err)
	}

	// Just for logging
	if reqType := types.GetTypeName(cmd); len(reqType) > 0 {
		log.Info(fmt.Sprintf("[-->] %s", p.connLogID(conn)), reqType)
	} else {
		return fmt.Errorf("%sprotocol error: BAD DATA SENT", p.connLogID(conn))
	}

	return nil
}

// -------------- VPN connection requests counter ---------------
func (p *Protocol) vpnConnectReqCounter() (int, time.Time) {
	p._connectRequestsMutex.Lock()
	defer p._connectRequestsMutex.Unlock()

	return p._connectRequests, p._connectRequestLastTime
}
func (p *Protocol) vpnConnectReqCounterIncrease() time.Time {
	p._connectRequestsMutex.Lock()
	defer p._connectRequestsMutex.Unlock()

	p._connectRequestLastTime = time.Now()
	p._connectRequests++
	return p._connectRequestLastTime
}
func (p *Protocol) vpnConnectReqCounterDecrease() {
	p._connectRequestsMutex.Lock()
	defer p._connectRequestsMutex.Unlock()

	p._connectRequests--
}

func (p *Protocol) createHelloResponse() *types.HelloResp {
	prefs := p._service.Preferences()

	wg, ovpn, obfsp := p._service.GetDisabledFunctions()
	var (
		wgErr    string
		ovpnErr  string
		obfspErr string
	)
	if wg != nil {
		wgErr = wg.Error()
	}
	if ovpn != nil {
		ovpnErr = ovpn.Error()
	}
	if obfsp != nil {
		obfspErr = obfsp.Error()
	}
	// send back Hello message with account session info
	helloResp := types.HelloResp{
		Version: version.Version(),
		Session: types.CreateSessionResp(prefs.Session),
		DisabledFunctions: types.DisabledFunctionality{
			WireGuardError: wgErr,
			OpenVPNError:   ovpnErr,
			ObfsproxyError: obfspErr}}
	return &helloResp
}

// -------------- processing connection request ---------------
func (p *Protocol) processConnectRequest(messageData []byte, stateChan chan<- vpn.StateInfo) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC on connect: ", r)
			// changing return values of main method
			err = errors.New("panic on connect: " + fmt.Sprint(r))
		}
	}()

	if p._disconnectRequested {
		log.Info("Disconnection was requested. Canceling connection.")
		return p._service.Disconnect()
	}

	var r types.Connect
	if err := json.Unmarshal(messageData, &r); err != nil {
		return fmt.Errorf("failed to unmarshal json 'Connect' request: %w", err)
	}

	retManualDNS := net.ParseIP(r.CurrentDNS)

	if vpn.Type(r.VpnType) == vpn.OpenVPN {
		var hosts []net.IP
		for _, v := range r.OpenVpnParameters.EntryVpnServer.IPAddresses {
			hosts = append(hosts, net.ParseIP(v))
		}

		connectionParams := openvpn.CreateConnectionParams(
			r.OpenVpnParameters.MultihopExitSrvID,
			r.OpenVpnParameters.Port.Protocol > 0, // is TCP
			r.OpenVpnParameters.Port.Port,
			hosts,
			r.OpenVpnParameters.ProxyType,
			net.ParseIP(r.OpenVpnParameters.ProxyAddress),
			r.OpenVpnParameters.ProxyPort,
			r.OpenVpnParameters.ProxyUsername,
			r.OpenVpnParameters.ProxyPassword)

		return p._service.ConnectOpenVPN(connectionParams, retManualDNS, r.FirewallOnDuringConnection, stateChan)

	} else if vpn.Type(r.VpnType) == vpn.WireGuard {
		hostValue := r.WireGuardParameters.EntryVpnServer.Hosts[rand.Intn(len(r.WireGuardParameters.EntryVpnServer.Hosts))]

		connectionParams := wireguard.CreateConnectionParams(
			r.WireGuardParameters.Port.Port,
			net.ParseIP(hostValue.Host),
			hostValue.PublicKey,
			net.ParseIP(strings.Split(hostValue.LocalIP, "/")[0]))

		return p._service.ConnectWireGuard(connectionParams, retManualDNS, r.FirewallOnDuringConnection, stateChan)

	}

	return fmt.Errorf("unexpected VPN type to connect (%v)", r.VpnType)
}
