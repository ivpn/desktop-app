package vpn

import "net"

// Type - VPN type
type Type int

// Supported VPN protocols
const (
	OpenVPN   Type = iota
	WireGuard Type = iota
)

// State - state of VPN
type State int

// Possible VPN state values (must be applicable for all protocols)
// Such stetes MUST be in use by ALL supportded VPN protocols:
// 		DISCONNECTED
// 		CONNECTING
// 		CONNECTED
// 		EXITING
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
	EXITING      State = iota // A graceful exit is in progress.
)

func (s State) String() string {
	if s < DISCONNECTED || s > EXITING {
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
		"EXITING"}[s]
}

// StateInfo - VPN state + additional information
type StateInfo struct {
	State       State
	Description string

	ClientIP    net.IP // applicable only for 'CONNECTED' state
	ServerIP    net.IP // applicable only for 'CONNECTED' state
	IsAuthError bool   // applicable only for 'EXITING' state
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
func NewStateInfoConnected(clientIP net.IP, serverIP net.IP) StateInfo {
	return StateInfo{
		State:       CONNECTED,
		Description: "",
		ClientIP:    clientIP,
		ServerIP:    serverIP,
		IsAuthError: false}
}

// Process represents VPN object operations
type Process interface {
	// Connect - SYNCHRONOUSLY execute openvpn process (wait untill it finished)
	Connect(stateChan chan<- StateInfo) error
	Disconnect() error
	Pause() error
	Resume() error
	IsPaused() bool

	SetManualDNS(addr net.IP) error
	ResetManualDNS() error

	// DestinationIPs -  Get destination IPs (VPN host server or proxy server IP address)
	// This information if required, for example, to allow this address in firewall
	DestinationIPs() []net.IP
}
