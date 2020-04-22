package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"

	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/vpn"
	"github.com/ivpn/desktop-app-daemon/vpn/openvpn"
	"github.com/ivpn/desktop-app-daemon/vpn/wireguard"
)

func (p *Protocol) processConnectRequest(messageData []byte, stateChan chan<- vpn.StateInfo) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC on connect: ", r)
			// changing return values of main method
			err = errors.New("panic on connect: " + fmt.Sprint(r))
		}
	}()

	if p._disconnectRequestCmdIdx > 0 {
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

func (p *Protocol) sendResponse(cmd interface{}, idx int) error {
	if p._clientIsAuthenticated == false {
		return fmt.Errorf("client is not authenticated")
	}

	err := sendResponse(p.clientConnection(), cmd, idx)
	if err != nil {
		log.Error(err)
	}
	return err
}

func (p *Protocol) sendErrorResponse(request types.CommandBase, err error) {
	log.Error(fmt.Sprintf("Error processing request '%s': %s", request.Command, err))
	sendResponse(p.clientConnection(), &types.ErrorResp{ErrorMessage: err.Error()}, request.Idx)
}

func sendResponse(conn net.Conn, cmd interface{}, idx int) (retErr error) {
	if conn == nil {
		return fmt.Errorf("response not sent (no connection to client)")
	}

	if err := types.Send(conn, cmd, idx); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	// Just for logging
	if reqType := types.GetTypeName(cmd); len(reqType) > 0 {
		log.Info("[-->] ", reqType)
	} else {
		return fmt.Errorf("protocol error: BAD DATA SENT")
	}

	return nil
}
