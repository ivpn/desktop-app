package ivpnclient

// HelloResp is a response to Hello command with general from daemon
type HelloResp struct {
	CommandBase
	Version       string
	ServiceBinary string `json:",omitempty"`
	ProcessorArch string
	// NOTE: this is not all fields that IVPN client sends, add more if needed
}

// ConnectedResp is a notification about successful connection to the VPN server.
// It is sent by the daemon to the client when a connection is established.
type ConnectedResp struct {
	CommandBase
	TimeSecFrom1970 int64
	ClientIP        string
	ClientIPv6      string
	ServerIP        string
	ServerPort      int
	IsTCP           bool
	Mtu             int    // (for WireGuard connections)
	IsPaused        bool   // When "true" - the actual connection may be "disconnected" (depending on the platform and VPN protocol), but the daemon responds "connected"
	PausedTill      string // pausedTill.Format(time.RFC3339)
	IsUnhealthy     bool   // When "true" - WireGuard tunnel health checks failed; no peer response detected
	VpnType         VpnType
	// NOTE: this is not all fields that IVPN client sends, add more if needed
}

type DisconnectionReason int

// Disconnection reason types
const (
	Unknown             DisconnectionReason = iota
	AuthenticationError DisconnectionReason = iota
	DisconnectRequested DisconnectionReason = iota
)

// DisconnectedResp notifying about stopped VPN connection
type DisconnectedResp struct {
	CommandBase
	Failure           bool
	Reason            DisconnectionReason
	ReasonDescription string
	IsStateInfo       bool // if 'true' - it is not an disconnection event, it is just status info "disconnected"
}

type DnsEncryption int

const (
	EncryptionNone         DnsEncryption = 0
	EncryptionDnsOverTls   DnsEncryption = 1
	EncryptionDnsOverHttps DnsEncryption = 2
)

type DnsServerConfig struct {
	Address    string        // IP address of the DNS server
	Encryption DnsEncryption // Encryption type (None, DoH, DoT)
	Template   string        // DoH/DoT template
}

type DnsSettings struct {
	// List of DNS servers specified in order of preference
	Servers []DnsServerConfig
}

func (d DnsSettings) IsEmpty() bool {
	return len(d.Servers) == 0
}

type AntiTrackerMetadata struct {
	Enabled                  bool
	Hardcore                 bool
	AntiTrackerBlockListName string
}

// SetAlternateDns request to set custom DNS
type SetAlternateDns struct {
	RequestBase
	AntiTracker AntiTrackerMetadata // takes precedence over ManualDNS but can be overridden by TempPrioritizedDns
	Dns         DnsSettings         // user-defined manual DNS settings
}

// SetDnsOverride sets DNS override parameters for the current VPN connection.
//
// DNS priority order (highest to lowest):
//   - antiTracker       — if enabled, uses AntiTracker DNS (overrides dnsCfg)
//   - dnsCfg            — user-defined manual DNS
//   - (default)         — VPN connection's own DNS
//
// Note: The Temporary Prioritized DNS has the highest priority and overrides all other DNS settings, but it is not set via this method.
//
// All three parameters are saved to service preferences, so all must be provided
// even if only one needs to change. Recommended usage:
//  1. Read current DNS state (dnsCfg, antiTracker)
//  2. Modify the desired parameter(s)
//  3. Call SetDnsOverride with the full updated set
func (c *Client) SetManualDNS(dnsCfg DnsSettings, antiTracker AntiTrackerMetadata) error {
	req := SetAlternateDns{Dns: dnsCfg, AntiTracker: antiTracker}
	var resp EmptyResp
	if err := c.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

// Temporary prioritized DNS settings
type SetTempPrioritizedDns struct {
	RequestBase
	Dns DnsSettings
	// Description of the issuer of the temporary prioritized DNS settings (e.g. "Portmaster").
	// To be displayed in the UI to provide additional context to the user about the active DNS configuration.
	Description string
}

// SetTempPrioritizedDns sets temporary prioritized DNS settings that take precedence over all other DNS settings (including AntiTracker).
// Intended for clients that manage DNS only while connected (e.g. Portmaster).
// Not persistent: resets to empty on next application start and is cleared automatically on client disconnect.
// Parameters:
//   - dnsCfg: DNS settings to be applied with the highest priority (overriding all other DNS settings, including AntiTracker).
//     Setting empty dnsCfg will disable prioritized DNS and restore the priority of other DNS settings (AntiTracker, user-defined manual DNS, VPN connection's own DNS).
//   - description: Description of the issuer of the temporary prioritized DNS settings (e.g. "Portmaster") to be displayed in the UI to provide additional context to the user about the active DNS configuration.
func (c *Client) SetTempPrioritizedDns(dnsCfg DnsSettings, description string) error {
	req := SetTempPrioritizedDns{Dns: dnsCfg, Description: description}
	var resp EmptyResp
	if err := c.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

//
// TODO: more client commands can be added here in the future ...
//
