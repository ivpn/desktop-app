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

package types

import (
	"github.com/ivpn/desktop-app/daemon/obfsproxy"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	service_types "github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

type EmptyReq struct {
	RequestBase
}

type ClientTypeEnum int

const (
	ClientUi  ClientTypeEnum = iota // 0
	ClientCli ClientTypeEnum = iota // 1
)

// Hello is an initial request
type Hello struct {
	RequestBase

	// connected client type
	ClientType ClientTypeEnum
	// connected client version
	Version string

	Secret uint64

	// when 'true' - send HelloResp to all connected clients
	SendResponseToAllClients bool

	// GetServersList == true - client requests to send back info about all servers
	GetServersList bool

	// GetStatus == true - client requests current status (Vpn connection, Firewal... etc.)
	GetStatus bool

	// GetSplitTunnelStatus == true - client requests configuration of SplitTunnelling
	GetSplitTunnelStatus bool

	// GetWiFiCurrentState == true - client requests info about current WiFi
	GetWiFiCurrentState bool
}

// GetServers request servers list
type GetServers struct {
	RequestBase
	// Force to update servers from the backend (locations, hosts and hosts load)
	RequestServersUpdate bool
}

// PingServers request to ping servers
//
// Pinging operation separated on few phases:
//  1. Fast ping:  ping one host for each nearest location (locations only for specified VPN type when VpnTypePrioritization==true)
//     Operation ends after 'TimeOutMs'. Then daemon sends response PingServersResp with all data were collected.
//  2. Full ping: Pinging all hosts for all locations. There is no time limit for this operation. It runs in background.
//     2.1) Ping all hosts only for specified VPN type (when VpnTypePrioritization==true) or for all VPN types (when VpnTypePrioritization==false)
//     2.1.1) Ping one host for all locations (for prioritized VPN protocol)
//     2.1.2) Ping the rest hosts for all locations (for prioritized VPN protocol)
//     2.2) (when VpnTypePrioritization==true) Ping all hosts for the rest protocols
//
// If PingAllHostsOnFirstPhase==true - daemon will ping all hosts for nearest locations on the phase (1)
// If SkipSecondPhase==true - phase (2) will be skipped
type PingServers struct {
	RequestBase
	TimeOutMs                int
	VpnTypePrioritized       vpn.Type // hosts for this VPN type will be pinged first (only if VpnTypePrioritization == true)
	VpnTypePrioritization    bool
	PingAllHostsOnFirstPhase bool
	SkipSecondPhase          bool
}

// KillSwitchSetAllowLANMulticast enable\disable LAN multicast acces for kill-switch
type KillSwitchSetAllowLANMulticast struct {
	RequestBase
	AllowLANMulticast bool
}

// KillSwitchSetAllowLAN enable\disable LAN acces for kill-switch
type KillSwitchSetAllowLAN struct {
	RequestBase
	AllowLAN bool
}

// KillSwitchSetUserExceptions set ip masks to exclude from firewall blocking rules
type KillSwitchSetUserExceptions struct {
	CommandBase
	// Firewall exceptions: comma separated list of IP addresses (masks) in format: x.x.x.x[/xx]
	UserExceptions     string
	FailOnParsingError bool
}

type KillSwitchSetAllowApiServers struct {
	RequestBase
	IsAllowApiServers bool
}

// KillSwitchSetEnabled request to enable\disable kill-switch
type KillSwitchSetEnabled struct {
	RequestBase
	IsEnabled bool
}

// KillSwitchGetStatus get full killswitch status
type KillSwitchGetStatus struct {
	RequestBase
}

// KillSwitchSetIsPersistent request to mark kill-switch persistant
type KillSwitchSetIsPersistent struct {
	RequestBase
	IsPersistent bool
}

// SetPreference sets daemon configuration parameter
// (This is an old implementation. It is necessary to use 'SetUserPreferences/SettingsResp' for future extensions)
type SetPreference struct {
	RequestBase
	Key   string
	Value string
}

// SetUserPreferences sets daemon configuration parameters (the 'SettingsResp' is in use to send this settings to client)
type SetUserPreferences struct {
	RequestBase
	UserPrefs preferences.UserPreferences
}

// SetObfsProxy sets obfsproxy configuration
type SetObfsProxy struct {
	RequestBase
	ObfsproxyConfig obfsproxy.Config
}

// SetAlternateDns request to set custom DNS
type SetAlternateDns struct {
	RequestBase
	Dns dns.DnsSettings
}

// GetDnsPredefinedConfigs request to get list of predefined DoH/DoT configurations (if exists)
type GetDnsPredefinedConfigs struct {
	RequestBase
}

// WiFiAvailableNetworks - get list of available WIFI networks
type WiFiAvailableNetworks struct {
	RequestBase
}

// WiFiCurrentNetwork - request info about connected WIFI
type WiFiCurrentNetwork struct {
	RequestBase
}

// WiFiSettings - set wifi configuration
type WiFiSettings struct {
	RequestBase
	Params preferences.WiFiParams
}

// ConnectSettings contains same data as 'Connect' request but this command not start the connection.
// UI client have to notify daemon about changes in connection settings.
// It is required for automatic connection on daemon's side (e.g. 'Auto-connect on Launch' or 'Trusted WiFi' functionality)
type ConnectSettings struct {
	RequestBase
	Params service_types.ConnectionParams
}

// Connect request to establish new VPN connection
type Connect struct {
	RequestBase
	Params service_types.ConnectionParams
}

// Disconnect disconnect active VPN connection
type Disconnect struct {
	RequestBase
}

// GetVPNState request daemon to provive current VPN connection state
type GetVPNState struct {
	RequestBase
}

// SessionNew - create new session
//
// When force is set to true - all active sessions will be deleted prior to creating a new one if user reached session limit.
// Initial call to /sessin/new should always be performed with force set to false, to display special form, when sessions limit is reached.
// IVPN client apps have to set force to true only when customer clicks Log all other clients button.
type SessionNew struct {
	RequestBase
	AccountID  string
	ForceLogin bool

	CaptchaID       string
	Captcha         string
	Confirmation2FA string
}

// SessionDelete logout from current device
type SessionDelete struct {
	RequestBase
	NeedToResetSettings   bool
	NeedToDisableFirewall bool
	// If IsCanDeleteSessionLocally==true: the account will be logged out
	// even if there is no connectivity to API server
	IsCanDeleteSessionLocally bool
}

// AccountStatus get account status
type AccountStatus struct {
	RequestBase
}

// WireGuardGenerateNewKeys - generate WG keys
type WireGuardGenerateNewKeys struct {
	RequestBase
	OnlyUpdateIfNecessary bool
}

// WireGuardSetKeysRotationInterval -  change WG keys rotation interval
type WireGuardSetKeysRotationInterval struct {
	RequestBase
	Interval int64
}

// IPProtocol - VPN type
type RequiredIPProtocol int

const (
	IPvAny RequiredIPProtocol = 0
	IPv4   RequiredIPProtocol = 1
	IPv6   RequiredIPProtocol = 2
)

// APIRequest do custom request to API
type APIRequest struct {
	RequestBase
	APIPath            string
	IPProtocolRequired RequiredIPProtocol
}

// paranoid mode

type ParanoidModeSetPasswordReq struct {
	RequestBase
	NewSecret string
}
