package types

import (
	"github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/service/preferences"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("prttyp")
}

// ErrorResp response of error
type ErrorResp struct {
	CommandBase
	ErrorMessage string
}

// EmptyResp empty response on request
type EmptyResp struct {
	CommandBase
}

// ServiceExitingResp service is going to exit response
type ServiceExitingResp struct {
	CommandBase
}

// DisabledFunctionality Some functionality can be not accessible
// It can happen, for example, if some external binaries not installed
// (e.g. obfsproxy or WireGaurd on Linux)
type DisabledFunctionality struct {
	WireGuardError string
	OpenVPNError   string
	ObfsproxyError string
}

// HelloResp response on initial request
type HelloResp struct {
	CommandBase
	Version           string
	Session           SessionResp
	DisabledFunctions DisabledFunctionality
	//Account preferences.AccountStatus
}

// SessionResp information about session
type SessionResp struct {
	AccountID          string
	Session            string
	WgPublicKey        string
	WgLocalIP          string
	WgKeyGenerated     int64 // Unix time
	WgKeysRegenInerval int64 // seconds
}

// CreateSessionResp create new session info object to send to client
func CreateSessionResp(s preferences.SessionStatus) SessionResp {
	return SessionResp{
		AccountID:          s.AccountID,
		Session:            s.Session,
		WgPublicKey:        s.WGPublicKey,
		WgLocalIP:          s.WGLocalIP,
		WgKeyGenerated:     s.WGKeyGenerated.Unix(),
		WgKeysRegenInerval: int64(s.WGKeysRegenInerval.Seconds())}
}

// SessionNewResp - information about created session (or error info)
type SessionNewResp struct {
	CommandBase
	APIStatus       int
	APIErrorMessage string
	Session         SessionResp
	Account         preferences.AccountStatus
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
	IsEnabled        bool
	IsPersistent     bool
	IsAllowLAN       bool
	IsAllowMulticast bool
}

// KillSwitchGetIsPestistentResp returns kill-switch persistance status
type KillSwitchGetIsPestistentResp struct {
	CommandBase
	IsPersistent bool
}

// DiagnosticsGeneratedResp returns info from daemon logs
type DiagnosticsGeneratedResp struct {
	CommandBase
	ServiceLog     string
	ServiceLog0    string
	OpenvpnLog     string
	OpenvpnLog0    string
	EnvironmentLog string
}

// SetAlternateDNSResp returns status of changing DNS
type SetAlternateDNSResp struct {
	CommandBase
	IsSuccess  bool
	ChangedDNS string
}

// ConnectedResp notifying about established connection
type ConnectedResp struct {
	CommandBase
	VpnType         vpn.Type
	TimeSecFrom1970 int64
	ClientIP        string
	ServerIP        string
	ExitServerID    string
	ManualDNS       string
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

//PingResultType represents information ping TTL for a host (is a part of 'PingServersResp')
type PingResultType struct {
	Host string
	Ping int
}

// PingServersResp returns average ping time for servers
type PingServersResp struct {
	CommandBase
	PingResults []PingResultType
}
