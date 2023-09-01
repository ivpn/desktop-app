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

package vpn

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/ivpn/desktop-app/daemon/obfsproxy"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/v2r"
)

// Type - VPN type
type Type int

// Supported VPN protocols
const (
	OpenVPN   Type = iota
	WireGuard Type = iota
)

func (t Type) String() string {
	switch t {
	case OpenVPN:
		return "OpenVPN"
	case WireGuard:
		return "WireGuard"
	}
	return "<Unknown>"
}

// State - state of VPN
type State int

// Possible VPN state values (must be applicable for all protocols)
// Such states MUST be in use by ALL supported VPN protocols:
//
//	DISCONNECTED
//	CONNECTING
//	CONNECTED
//	EXITING
const (
	DISCONNECTED State = iota
	CONNECTING   State = iota // OpenVPN's initial state.
	WAIT         State = iota // (Client only) Waiting for initial response from server.
	AUTH         State = iota // (Client only) Authenticating with server.
	GETCONFIG    State = iota // (Client only) Downloading configuration options from server.
	ASSIGNIP     State = iota // Assigning IP address to virtual network interface.
	ADDROUTES    State = iota // Adding routes to system.
	CONNECTED    State = iota // Initialization Sequence Completed.
	RECONNECTING State = iota // A restart has occurred.
	TCP_CONNECT  State = iota // TCP_CONNECT
	EXITING      State = iota // A graceful exit is in progress.
	INITIALISED  State = iota // Interface initialised (WireGuard: but connection handshake still not detected)
)

func (s State) String() string {
	if s < DISCONNECTED || s > INITIALISED {
		return "<Unknown>"
	}

	return []string{
		"DISCONNECTED",
		"CONNECTING",
		"WAIT",
		"AUTH",
		"GETCONFIG",
		"ASSIGNIP",
		"ADDROUTES",
		"CONNECTED",
		"RECONNECTING",
		"TCP_CONNECT",
		"EXITING",
		"INITIALISED"}[s]
}

// ParseState - Converts string representation of OpenVPN state to vpn.State
func ParseState(stateStr string) (State, error) {
	stateStr = strings.Trim(stateStr, " \t;,.")
	switch stateStr {
	case "CONNECTING":
		return CONNECTING, nil
	case "WAIT":
		return WAIT, nil
	case "AUTH":
		return AUTH, nil
	case "GET_CONFIG":
		return GETCONFIG, nil
	case "ASSIGN_IP":
		return ASSIGNIP, nil
	case "ADD_ROUTES":
		return ADDROUTES, nil
	case "CONNECTED":
		return CONNECTED, nil
	case "RECONNECTING":
		return RECONNECTING, nil
	case "TCP_CONNECT":
		return TCP_CONNECT, nil
	case "EXITING":
		return EXITING, nil
	case "INITIALISED":
		return INITIALISED, nil
	default:
		return DISCONNECTED, errors.New("unexpected state:" + stateStr)
	}
}

// StateInfo - VPN state + additional information
type StateInfo struct {
	State       State
	Description string

	VpnType      Type
	Time         int64                  // unix time (seconds)
	IsTCP        bool                   // applicable only for 'CONNECTED' state
	ClientIP     net.IP                 // applicable only for 'CONNECTED' state
	ClientIPv6   net.IP                 // applicable only for 'CONNECTED' state. Initialized only if protocol supports IPv6 inside tunnel
	ClientPort   int                    // applicable only for 'CONNECTED' state (source port)
	ServerIP     net.IP                 // applicable only for 'CONNECTED' state
	ServerPort   int                    // applicable only for 'CONNECTED' state (destination port)
	V2RayProxy   v2r.V2RayTransportType // applicable only for 'CONNECTED' state
	Obfsproxy    obfsproxy.Config       // applicable only for 'CONNECTED' state (OpenVPN)
	ExitHostname string                 // applicable only for 'CONNECTED' state
	Mtu          int                    // applicable only for 'CONNECTED' state (WireGuard)
	IsAuthError  bool                   // applicable only for 'EXITING' state

	// TODO: try to avoid using this protocol-specific parameter in future
	// Currently, in use by OpenVPN connection to inform about "RECONNECTING" reason (e.g. "tls-error", "init_instance"...)
	// UI client using this info in order to determine is it necessary to try to connect with another port
	StateAdditionalInfo string
}

// NewStateInfo - create new state object (not applicable for CONNECTED state)
func NewStateInfo(state State, description string) StateInfo {
	return StateInfo{
		State:       state,
		Description: description,
		ClientIP:    nil,
		ServerIP:    nil,
		IsAuthError: false}
}

// NewStateInfoConnected - create new state object for CONNECTED state
func NewStateInfoConnected(isTCP bool, clientIP net.IP, clientIPv6 net.IP, localPort int, serverIP net.IP, destPort int, mtu int) StateInfo {
	return StateInfo{
		State:       CONNECTED,
		Description: "",
		IsTCP:       isTCP,
		ClientIP:    clientIP,
		ClientIPv6:  clientIPv6,
		ClientPort:  localPort,
		ServerIP:    serverIP,
		ServerPort:  destPort,
		IsAuthError: false,
		Mtu:         mtu,
	}
}

// Process represents VPN object operations
type Process interface {
	// Type just returns VPN type
	Type() Type
	// Init performs basic initializations before connection
	// It is usefull, for example, for WireGuard(Windows) - to ensure that WG service is fully uninstalled
	// (currently, in use by WireGuard(Windows))
	Init() error

	// Connect - SYNCHRONOUSLY execute openvpn process (wait until it finished)
	Connect(stateChan chan<- StateInfo) error
	Disconnect() error
	Pause() error
	Resume() error
	IsPaused() bool

	DefaultDNS() net.IP
	SetManualDNS(dnsCfg dns.DnsSettings) error
	ResetManualDNS() error

	// DestinationIP -  Get destination IP (VPN host server or proxy server IP address)
	// This information if required, for example, to allow this address in firewall
	DestinationIP() net.IP

	IsIPv6InTunnel() bool

	OnRoutingChanged() error
}

// ReconnectionRequiredError object can be returned by vpn.Process.Connect() function
// which means that it requesting to do re-connect immediately
type ReconnectionRequiredError struct {
	Err error
}

func (e *ReconnectionRequiredError) Error() string {
	mes := "re-connection required"
	if e.Err == nil {
		return mes
	}
	return fmt.Sprintf("%s: %s", mes, e.Err.Error())
}

// Unwrap returns inner error
func (e *ReconnectionRequiredError) Unwrap() error { return e.Err }
