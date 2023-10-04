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

package types

import (
	"fmt"

	"github.com/ivpn/desktop-app/daemon/api/types"
	api_types "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/obfsproxy"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	service_types "github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/v2r"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("prttyp")
}

type ErrorType int

const (
	ErrorUnknown                   ErrorType = iota
	ErrorParanoidModePasswordError ErrorType = iota
)

// ErrorResp response of error
type ErrorResp struct {
	CommandBase
	ErrorMessage string
	ErrorTitle   string
	ErrorType    ErrorType
}

func (e ErrorResp) Error() string {
	return e.ErrorMessage
}

// ErrorRespDelayed - error info which had happened in the past
type ErrorRespDelayed struct {
	ErrorResp
}

// EmptyResp empty response on request
type EmptyResp struct {
	CommandBase
}

// ServiceExitingResp service is going to exit response
type ServiceExitingResp struct {
	CommandBase
}

type DisabledFunctionalityLinux struct {
	// If not empty - it is not possible to use the old way of DNS management
	// (which is based on a direct change of '/etc/resolv.conf')
	// For example: It could be because of snap environment (it does not allow to modify '/etc/resolv.conf')
	DnsMgmtOldResolvconfError string

	// If not empty - it is not possible to use modern way of DNS management
	// (based on communicationd with 'resolved' using 'resolvectl')
	// There could be different reasons of it:
	//	- there is no 'resolvectl' binary on target system
	//	- 'resolvectl' initialisation try was failed
	DnsMgmtNewResolvectlError string
}

type DisabledFunctionalityForPlatform struct {
	// Linux specific functionality which is disabled
	Linux DisabledFunctionalityLinux

	// Windows ...

	// macOS ...
}

// DisabledFunctionality Some functionality can be not accessible
// It can happen, for example, if some external binaries not installed
// (e.g. obfsproxy or WireGaurd on Linux)
type DisabledFunctionality struct {
	WireGuardError          string // WireGuard is not supported on this platform
	OpenVPNError            string // OpenVPN is not supported on this platform
	ObfsproxyError          string // Obfsproxy is not supported on this platform
	V2RayError              string // V2Ray is not supported on this platform
	SplitTunnelError        string // SplitTunneling is not supported on this platform
	SplitTunnelInverseError string // Inversed SplitTunneling is not supported on this platform

	// Linux specific functionality which is disabled
	Platform DisabledFunctionalityForPlatform
}

type DnsAbilities struct {
	CanUseDnsOverTls   bool
	CanUseDnsOverHttps bool
}

type ParanoidModeStatus struct {
	IsEnabled bool
}

type SettingsResp struct {
	CommandBase

	IsAutoconnectOnLaunch       bool
	IsAutoconnectOnLaunchDaemon bool
	UserDefinedOvpnFile         string
	UserPrefs                   preferences.UserPreferences
	WiFi                        preferences.WiFiParams
	IsLogging                   bool
	AntiTracker                 service_types.AntiTrackerMetadata

	// TODO: implement the rest of daemon settings
	// IsFwPersistant        bool
	// IsFwAllowLAN          bool
	// IsFwAllowLANMulticast bool
	// IsFwAllowApiServers   bool
	// FwUserExceptions      string
	// IsSplitTunnel         bool
	// SplitTunnelApps       []string
}

// HelloResp response on initial request
type HelloResp struct {
	CommandBase
	Version           string
	ProcessorArch     string
	Session           SessionResp
	Account           preferences.AccountStatus
	DisabledFunctions DisabledFunctionality
	Dns               DnsAbilities

	// SettingsSessionUUID is unique for Preferences object
	// It allow to detect situations when settings was erased (created new Preferences object)
	SettingsSessionUUID string

	ParanoidMode ParanoidModeStatus

	DaemonSettings SettingsResp
}

// SessionResp information about session
type SessionResp struct {
	AccountID          string
	Session            string
	WgPublicKey        string
	WgLocalIP          string
	WgKeyGenerated     int64 // Unix time
	WgKeysRegenInerval int64 // seconds
	WgUsePresharedKey  bool
}

// CreateSessionResp create new session info object to send to client
func CreateSessionResp(s preferences.SessionStatus) SessionResp {
	return SessionResp{
		AccountID:          s.AccountID,
		Session:            s.Session,
		WgPublicKey:        s.WGPublicKey,
		WgLocalIP:          s.WGLocalIP,
		WgKeyGenerated:     s.WGKeyGenerated.Unix(),
		WgKeysRegenInerval: int64(s.WGKeysRegenInerval.Seconds()),
		WgUsePresharedKey:  len(s.WGPresharedKey) > 0}
}

// SessionNewResp - information about created session (or error info)
type SessionNewResp struct {
	CommandBase
	APIStatus       int
	APIErrorMessage string
	Session         SessionResp
	Account         preferences.AccountStatus
	RawResponse     string
}

// AccountStatusResp - information about account status (or error info)
type AccountStatusResp struct {
	CommandBase
	APIStatus       int
	APIErrorMessage string
	SessionToken    string
	Account         preferences.AccountStatus
}

// KillSwitchStatusResp returns kill-switch status
type KillSwitchStatusResp struct {
	CommandBase
	service_types.KillSwitchStatus
}

// KillSwitchGetIsPestistentResp returns kill-switch persistance status
type KillSwitchGetIsPestistentResp struct {
	CommandBase
	IsPersistent bool
}

// DiagnosticsGeneratedResp returns info from daemon logs
type DiagnosticsGeneratedResp struct {
	CommandBase
	Log0_Old    string // previous daemon session log
	Log1_Active string // active daemon log
	ExtraInfo   string // Extra info for logging (e.g. ifconfig, netstat -nr ... etc.)
}

type DnsStatus struct {
	Dns               dns.DnsSettings
	AntiTrackerStatus service_types.AntiTrackerMetadata
}

// SetAlternateDNSResp returns status of changing DNS
type SetAlternateDNSResp struct {
	CommandBase
	Dns DnsStatus
}

// DnsPredefinedConfigsResp list of predefined DoH/DoT configurations (if exists)
type DnsPredefinedConfigsResp struct {
	CommandBase
	DnsConfigs []dns.DnsSettings
}

// ConnectedResp notifying about established connection
type ConnectedResp struct {
	CommandBase
	VpnType         vpn.Type
	TimeSecFrom1970 int64
	ClientIP        string
	ClientIPv6      string
	ServerIP        string
	ServerPort      int
	ExitHostname    string // multi-hop exit hostname (e.g. "us-tx1.wg.ivpn.net")
	Dns             DnsStatus
	IsTCP           bool
	Mtu             int                    // (for WireGuard connections)
	V2RayProxy      v2r.V2RayTransportType // applicable only for 'CONNECTED' state
	Obfsproxy       obfsproxy.Config       // applicable only for 'CONNECTED' state (OpenVPN)
	IsPaused        bool                   // When "true" - the actual connection may be "disconnected" (depending on the platform and VPN protocol), but the daemon responds "connected"
	PausedTill      string                 // pausedTill.Format(time.RFC3339)
}

// DisconnectionReason - disconnection reason
type DisconnectionReason int

// Disconnection reason types
const (
	Unknown             DisconnectionReason = iota
	AuthenticationError DisconnectionReason = iota
	DisconnectRequested DisconnectionReason = iota
)

// DisconnectedResp notifying about stopped connetion
type DisconnectedResp struct {
	CommandBase
	Failure           bool
	Reason            DisconnectionReason //int
	ReasonDescription string
	IsStateInfo       bool // if 'true' - it is not an disconneection event, it is just status info "disconnected"
}

// VpnStateResp returns VPN connection state
type VpnStateResp struct {
	CommandBase
	// TODO: remove 'State' field. Use only 'StateVal'
	State               string
	StateVal            vpn.State
	StateAdditionalInfo string
}

// ServerListResp returns list of servers
type ServerListResp struct {
	CommandBase
	VpnServers types.ServersInfoResponse
}

// PingResultType represents information ping TTL for a host (is a part of 'PingServersResp')
type PingResultType struct {
	Host string
	Ping int
}

// PingServersResp returns average ping time for servers
type PingServersResp struct {
	CommandBase
	PingResults []PingResultType
}

// WiFiNetworkInfo - information about WIFI network
type WiFiNetworkInfo struct {
	SSID string
}

// WiFiAvailableNetworksResp - contains information about available WIFI networks
type WiFiAvailableNetworksResp struct {
	CommandBase
	Networks []WiFiNetworkInfo
}

// WiFiCurrentNetworkResp contains the information about currently connected WIFI
type WiFiCurrentNetworkResp struct {
	CommandBase
	SSID              string
	IsInsecureNetwork bool
}

// APIResponse contains the raw data of response to custom API request
type APIResponse struct {
	CommandBase
	APIPath      string
	ResponseData string
	Error        string
}

func (r APIResponse) LogExtraInfo() string {
	if len(r.Error) > 0 {
		return fmt.Sprint(r.APIPath, " Error!")
	}
	return fmt.Sprint(r.APIPath)
}

type CheckAccessiblePortsResponse struct {
	RequestBase
	Ports []api_types.PortInfo
}
