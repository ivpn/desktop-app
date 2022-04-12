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
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/platform"
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

func (p *Protocol) sendErrorResponse(conn net.Conn, request types.RequestBase, err error) {
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
		log.Info(fmt.Sprintf("[-->] %s", p.connLogID(conn)), reqType, fmt.Sprintf(" [%d]", idx))
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

	dnsOverHttps, dnsOverTls, err := dns.EncryptionAbilities()
	if err != nil {
		dnsOverHttps = false
		dnsOverTls = false
		log.Error(err)
	}

	// send back Hello message with account session info
	helloResp := types.HelloResp{
		ParanoidMode:        types.ParanoidModeStatus{IsEnabled: p.paranoidModeIsEnabled()},
		Version:             version.Version(),
		Session:             types.CreateSessionResp(prefs.Session),
		SettingsSessionUUID: prefs.SettingsSessionUUID,
		DisabledFunctions: types.DisabledFunctionality{
			WireGuardError:   wgErr,
			OpenVPNError:     ovpnErr,
			ObfsproxyError:   obfspErr,
			SplitTunnelError: stErr},
		Dns: types.DnsAbilities{
			CanUseDnsOverTls:   dnsOverTls,
			CanUseDnsOverHttps: dnsOverHttps,
		},
	}
	return &helloResp
}

func (p *Protocol) createConnectedResponse(state vpn.StateInfo) *types.ConnectedResp {
	ipv6 := ""
	if state.ClientIPv6 != nil {
		ipv6 = state.ClientIPv6.String()
	}
	ret := &types.ConnectedResp{
		TimeSecFrom1970: state.Time,
		ClientIP:        state.ClientIP.String(),
		ClientIPv6:      ipv6,
		ServerIP:        state.ServerIP.String(),
		VpnType:         state.VpnType,
		ExitServerID:    state.ExitServerID,
		ManualDNS:       dns.GetLastManualDNS(),
		IsCanPause:      state.IsCanPause}

	return ret
}

// --------------------- paranoid mode ------------------------

//  key for encryption secret string (just to make it a little bit harder to read the file for humans).
const paranoidModeSecretEncryptionKey = "Fx>PT/*fllA3yr3}Jn+k(?h<~4%lJm$Y"

func (p *Protocol) paranoidModeSecret() (retSecret string, retErr error) {
	file := platform.ParanoidModeSecretFile()
	if len(file) <= 0 {
		return "", nil // paranoid mode not implemented for this platform
	}

	f, err := os.Open(platform.ParanoidModeSecretFile())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // paranoid mode disabled
		}
		return "", fmt.Errorf("the Enhanced App Protection file open error : %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("the Enhanced App Protection file status check error : %w", err)
	}

	// check file access rights
	mode := stat.Mode()
	expectedMode := os.FileMode(0600) // read only for privilaged user
	if mode != expectedMode {
		return "", fmt.Errorf(fmt.Sprintf("the Enhanced App Protection file has wrong access permissions (%o but expected %o)", mode, expectedMode))
	}

	// read file
	if stat.Size() > 1024*5 {
		return "", fmt.Errorf("the Enhanced App Protection file too big")
	}
	buff := make([]byte, stat.Size())
	_, err = f.Read(buff)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read Enhanced App Protection file: %w", err)
	}

	// decode
	data, err := helpers.DecryptString([]byte(paranoidModeSecretEncryptionKey), string(buff))
	if err != nil {
		return "", fmt.Errorf("failed to decode EAP secret: %w", err)
	}

	// check first line
	lines := strings.Split(data, "\n")
	if len(lines) != 1 {
		return "", fmt.Errorf("wrong data in Enhanced App Protection file (expected one line)")
	}
	secret := strings.TrimSpace(lines[0])
	if len(secret) <= 0 {
		return "", fmt.Errorf("wrong data in Enhanced App Protection file (secret is empty)")
	}

	return secret, nil
}

func (p *Protocol) paranoidModeIsEnabled() bool {
	secret, err := p.paranoidModeSecret()
	return err == nil && len(secret) > 0
}

func (p *Protocol) paranoidModeForceDisable() error {
	file := platform.ParanoidModeSecretFile()
	if len(file) <= 0 {
		return nil // paranoid mode not implemented for this platform
	}

	// disable paranoid mode
	if err := os.Remove(file); err != nil {
		return fmt.Errorf("failed to disable Enhanced App Protection: %w", err)
	}
	p._paranoidModeLastFailedAttempts = nil

	return nil
}

func (p *Protocol) paranoidModeSetSecret(oldSecret, newSecret string) error {
	file := platform.ParanoidModeSecretFile()
	if len(file) <= 0 {
		return nil // paranoid mode not implemented for this platform
	}

	secret, err := p.paranoidModeSecret()
	isPmEnabled := err == nil && len(secret) > 0
	if isPmEnabled && oldSecret != secret {
		return fmt.Errorf("the current shared secret for EAP does not match")
	}

	if strings.TrimSpace(newSecret) != newSecret {
		return fmt.Errorf("please avoid using space symbols in EAP secret")
	}

	if len(strings.Split(newSecret, "\n")) != 1 {
		return fmt.Errorf("new shared secret for EAP should contain only one line")
	}

	if len(newSecret) == 0 {
		if isPmEnabled {
			// disable paranoid mode
			if err := p.paranoidModeForceDisable(); err != nil {
				return fmt.Errorf("failed to disable EAP: %w", err)
			}
		}
		return nil
	}

	//preferences := p._service.Preferences()
	//if !preferences.Session.IsLoggedIn() {
	//	return fmt.Errorf("can't set up EAP: please login to your account first")
	//}

	encrypted, err := helpers.EncryptString([]byte(paranoidModeSecretEncryptionKey), newSecret)
	if err != nil {
		return fmt.Errorf("failed to encode EAP secret: %w", err)
	}

	bytesToWrite := []byte(encrypted)
	if len(bytesToWrite) > 1024*10 {
		return fmt.Errorf("password too long")
	}

	// save data
	if err := os.WriteFile(file, bytesToWrite, 0600); err != nil {
		p.paranoidModeForceDisable()
		return fmt.Errorf("failed to enable EAP (FileWrite error): %w", err)
	}

	// ensure data were saved correctly
	secretConfirm, err := p.paranoidModeSecret()
	if err != nil {
		p.paranoidModeForceDisable()
		return fmt.Errorf("failed to enable EAP: %w", err)
	}
	if secretConfirm != newSecret {
		p.paranoidModeForceDisable()
		return fmt.Errorf("failed to enable EAP: internal error during confirmation")
	}

	return nil
}

func (p *Protocol) paranoidModeCheckSecret(secretToCheck string) (retVal bool, err error) {
	// some protection from brute force attack
	defer func() {
		if retVal {
			p._paranoidModeLastFailedAttempts = nil
		} else {
			p._paranoidModeLastFailedAttempts = append(p._paranoidModeLastFailedAttempts, time.Now())
		}
	}()
	const maxAttemptsCnt = 4
	const maxDuration = time.Minute
	cntAttempts := len(p._paranoidModeLastFailedAttempts)
	if cntAttempts >= maxAttemptsCnt {
		if cntAttempts > maxAttemptsCnt {
			p._paranoidModeLastFailedAttempts = p._paranoidModeLastFailedAttempts[cntAttempts-maxAttemptsCnt : cntAttempts]
		}
		if p._paranoidModeLastFailedAttempts[0].Add(maxDuration).After(time.Now()) {
			return false, fmt.Errorf("You have exceeded the allowed number of requests. Please wait 1 minute and try again.")
		}
	}

	// compare secrets
	secret, err := p.paranoidModeSecret()
	isPModeEnabled := err == nil && len(secret) > 0
	if !isPModeEnabled {
		return true, nil
	}
	return secret == secretToCheck, nil
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

	retManualDNS := r.ManualDNS

	if vpn.Type(r.VpnType) == vpn.OpenVPN {
		// PARAMETERS VALIDATION
		// parsing hosts
		var hosts []net.IP
		for _, v := range r.OpenVpnParameters.EntryVpnServer.Hosts {
			hosts = append(hosts, net.ParseIP(v.Host))
		}
		if len(hosts) < 1 {
			return fmt.Errorf("VPN host not defined")
		}
		// in case of multiple hosts - take random host from the list
		host := hosts[0]
		if len(hosts) > 1 {
			if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(hosts)))); err == nil {
				host = hosts[rnd.Int64()]
			}
		}

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
		multihopExitHosts := r.WireGuardParameters.MultihopExitServer.Hosts

		// filter hosts: use IPv6 hosts
		if r.IPv6 {
			ipv6Hosts := append(hosts[0:0], hosts...)
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

		// filter exit servers (Multi-Hop connection):
		// 1) each exit server must have initialized 'multihop_port' field
		// 2) (in case of IPv6Only) IPv6 local address should be defined
		if len(multihopExitHosts) > 0 {
			isHasMHPort := false
			ipv6ExitHosts := append(multihopExitHosts[0:0], multihopExitHosts...)
			n := 0
			for _, h := range ipv6ExitHosts {
				if h.MultihopPort == 0 {
					continue
				}
				isHasMHPort = true
				if r.IPv6 && h.IPv6.LocalIP == "" {
					continue
				}

				ipv6ExitHosts[n] = h
				n++
			}
			if n == 0 {
				if !isHasMHPort {
					return fmt.Errorf("unable to make Multi-Hop connection inside tunnel. Exit server does not support Multi-Hop")
				}
				if r.IPv6Only {
					return fmt.Errorf("unable to make IPv6 Multi-Hop connection inside tunnel. Exit server does not support IPv6")
				}
			} else {
				multihopExitHosts = ipv6ExitHosts[:n]
			}
		}

		hostValue := hosts[0]
		if len(hosts) > 1 {
			if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(hosts)))); err == nil {
				hostValue = hosts[rnd.Int64()]
			}
		}

		var exitHostValue *apitypes.WireGuardServerHostInfo
		if len(multihopExitHosts) > 0 {
			exitHostValue = &multihopExitHosts[0]
			if len(multihopExitHosts) > 1 {
				if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(multihopExitHosts)))); err == nil {
					exitHostValue = &multihopExitHosts[rnd.Int64()]
				}
			}
		}
		// only one-line parameter is allowed
		multihopExitSrvID := strings.Split(r.WireGuardParameters.MultihopExitServer.ExitSrvID, "\n")[0]

		// prevent user-defined data injection: ensure that nothing except the base64 public key will be stored in the configuration
		if !helpers.ValidateBase64(hostValue.PublicKey) {
			return fmt.Errorf("WG public key is not base64 string")
		}

		hostLocalIP := net.ParseIP(strings.Split(hostValue.LocalIP, "/")[0])
		ipv6Prefix := ""
		if r.IPv6 {
			ipv6Prefix = strings.Split(hostValue.IPv6.LocalIP, "/")[0]
		}

		var connectionParams wireguard.ConnectionParams
		if exitHostValue != nil && len(multihopExitSrvID) > 0 {
			// Multi-Hop
			connectionParams = wireguard.CreateConnectionParams(
				multihopExitSrvID,
				exitHostValue.MultihopPort,
				net.ParseIP(hostValue.Host),
				exitHostValue.PublicKey,
				hostLocalIP,
				ipv6Prefix)
		} else {
			// Single-Hop
			connectionParams = wireguard.CreateConnectionParams(
				"",
				r.WireGuardParameters.Port.Port,
				net.ParseIP(hostValue.Host),
				hostValue.PublicKey,
				hostLocalIP,
				ipv6Prefix)
		}

		return p._service.ConnectWireGuard(connectionParams, retManualDNS, r.FirewallOn, r.FirewallOnDuringConnection, stateChan)

	}

	return fmt.Errorf("unexpected VPN type to connect (%v)", r.VpnType)
}
