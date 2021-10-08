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
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/version"
	"github.com/ivpn/desktop-app/daemon/vpn"
	"github.com/ivpn/desktop-app/daemon/vpn/openvpn"
	"github.com/ivpn/desktop-app/daemon/vpn/wireguard"
)

// connID returns connection info (required to distinguish communication between several connections in log)
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
	delete(p._connections, c)
	c.Close()
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
	p._connections = make(map[net.Conn]struct{})
}

// -------------- sending responses ---------------
func (p *Protocol) sendError(conn net.Conn, errorText string, cmdIdx int) {
	log.Error(errorText)
	p.sendResponse(conn, &types.ErrorResp{ErrorMessage: errorText}, cmdIdx)
}

func (p *Protocol) sendErrorResponse(conn net.Conn, request types.CommandBase, err error) {
	log.Error(fmt.Sprintf("%sError processing request '%s': %s", p.connLogID(conn), request.Command, err))
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

	wg, ovpn, obfsp, splitTun := p._service.GetDisabledFunctions()
	var (
		wgErr    string
		ovpnErr  string
		obfspErr string
		stErr    string
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
	if splitTun != nil {
		stErr = splitTun.Error()
	}

	// send back Hello message with account session info
	helloResp := types.HelloResp{
		Version:             version.Version(),
		Session:             types.CreateSessionResp(prefs.Session),
		SettingsSessionUUID: prefs.SettingsSessionUUID,
		DisabledFunctions: types.DisabledFunctionality{
			WireGuardError:   wgErr,
			OpenVPNError:     ovpnErr,
			ObfsproxyError:   obfspErr,
			SplitTunnelError: stErr}}
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
		// PARAMETERS VALIDATION
		// parsing hosts
		var hosts []net.IP
		for _, v := range r.OpenVpnParameters.EntryVpnServer.IPAddresses {
			hosts = append(hosts, net.ParseIP(v))
		}
		if len(hosts) < 1 {
			return fmt.Errorf("VPN host not defined")
		}
		// in case of multiple hosts - take random host from the list
		host := hosts[rand.Intn(len(hosts))]

		// only one-line parameter is allowed
		multihopExitSrvID := strings.Split(r.OpenVpnParameters.MultihopExitSrvID, "\n")[0]
		// nothing from supported proxy types should be in this parameter
		proxyType := r.OpenVpnParameters.ProxyType
		if len(proxyType) > 0 && proxyType != "http" && proxyType != "socks" {
			proxyType = ""
		}
		// only one-line parameter is allowed
		proxyUsername := strings.Split(r.OpenVpnParameters.ProxyUsername, "\n")[0]
		proxyPassword := strings.Split(r.OpenVpnParameters.ProxyPassword, "\n")[0]

		// CONNECTION
		// OpenVPN connection parameters
		connectionParams := openvpn.CreateConnectionParams(
			multihopExitSrvID,
			r.OpenVpnParameters.Port.Protocol > 0, // is TCP
			r.OpenVpnParameters.Port.Port,
			host,
			proxyType,
			net.ParseIP(r.OpenVpnParameters.ProxyAddress),
			r.OpenVpnParameters.ProxyPort,
			proxyUsername,
			proxyPassword)

		return p._service.ConnectOpenVPN(connectionParams, retManualDNS, r.FirewallOn, r.FirewallOnDuringConnection, stateChan)

	} else if vpn.Type(r.VpnType) == vpn.WireGuard {
		hosts := r.WireGuardParameters.EntryVpnServer.Hosts

		// filter hosts: use IPv6 hosts
		if r.IPv6 {
			ipv6Hosts := append(hosts[:0:0], hosts...)
			n := 0
			for _, h := range ipv6Hosts {
				if h.IPv6.LocalIP != "" {
					ipv6Hosts[n] = h
					n++
				}
			}
			if n == 0 {
				if r.IPv6Only {
					return fmt.Errorf("unable to make IPv6 connection inside tunnel. Server does not support IPv6")
				}
			} else {
				hosts = ipv6Hosts[:n]
			}
		}
		hostValue := hosts[rand.Intn(len(hosts))]

		// prevent user-defined data injection: ensure that nothing except the base64 public key will be stored in the configuration
		if !helpers.ValidateBase64(hostValue.PublicKey) {
			return fmt.Errorf("WG public key is not base64 string")
		}

		hostLocalIP := net.ParseIP(strings.Split(hostValue.LocalIP, "/")[0])
		ipv6Prefix := ""
		if r.IPv6 {
			ipv6Prefix = strings.Split(hostValue.IPv6.LocalIP, "/")[0]
		}

		connectionParams := wireguard.CreateConnectionParams(
			r.WireGuardParameters.Port.Port,
			net.ParseIP(hostValue.Host),
			hostValue.PublicKey,
			hostLocalIP,
			ipv6Prefix)

		return p._service.ConnectWireGuard(connectionParams, retManualDNS, r.FirewallOn, r.FirewallOnDuringConnection, stateChan)

	}

	return fmt.Errorf("unexpected VPN type to connect (%v)", r.VpnType)
}
