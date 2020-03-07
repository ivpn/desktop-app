package types

import "github.com/ivpn/desktop-app-daemon/vpn"

// Hello is an initial request
type Hello struct {
	CommandBase
	// connected client version
	Version string
	Secret  uint64

	// GetServersList == true - client requests to send back info about all servers
	GetServersList bool

	// GetStatus == true - client requests current status (Vpn connection, Firewal... etc.)
	GetStatus bool

	//	KeepDaemonAlone == false (default) - VPN disconnects when client disconnects from a daemon
	//	KeepDaemonAlone == true - do nothing when client disconnects from a daemon (if VPN is connected - do not disconnect)
	KeepDaemonAlone bool
}

// GetServers request servers list
type GetServers struct {
	CommandBase
}

// PingServers request to ping servers
type PingServers struct {
	CommandBase
	RetryCount int
	TimeOutMs  int
}

// KillSwitchSetAllowLANMulticast enable\disable LAN multicast acces for kill-switch
type KillSwitchSetAllowLANMulticast struct {
	CommandBase
	AllowLANMulticast bool
}

// KillSwitchSetAllowLAN enable\disable LAN acces for kill-switch
type KillSwitchSetAllowLAN struct {
	CommandBase
	AllowLAN bool
}

// KillSwitchSetEnabled request to enable\disable kill-switch
type KillSwitchSetEnabled struct {
	CommandBase
	IsEnabled bool
}

// KillSwitchGetStatus get full killswitch status
type KillSwitchGetStatus struct {
	CommandBase
}

// KillSwitchSetIsPersistent request to mark kill-switch persistant
type KillSwitchSetIsPersistent struct {
	CommandBase
	IsPersistent bool
}

// SetPreferenceRequest sets daemon configuration parameter
type SetPreferenceRequest struct {
	CommandBase
	Key   string
	Value string
}

// SetAlternateDNS request to set custom DNS
type SetAlternateDNS struct {
	CommandBase
	DNS string
}

// WGHost is a WireGuard host description
type WGHost struct {
	Host      string
	PublicKey string `json:"public_key"`
	LocalIP   string `json:"local_ip"`
}

// Connect request to establish new VPN connection
type Connect struct {
	CommandBase
	VpnType    vpn.Type
	CurrentDNS string

	WireGuardParameters struct {
		Port struct {
			Port int
		}

		EntryVpnServer struct {
			Hosts []WGHost
		}
	}

	OpenVpnParameters struct {
		EntryVpnServer struct {
			IPAddresses []string `json:"ip_addresses"`
		}

		MultihopExitSrvID string
		ProxyType         string
		ProxyAddress      string
		ProxyPort         int
		ProxyUsername     string
		ProxyPassword     string

		Port struct {
			Port     int
			Protocol int
		}
	}
}

// Disconnect disconnect active VPN connection
type Disconnect struct {
	CommandBase
}

// GetVPNState request daemon to provive current VPN connection state
type GetVPNState struct {
	CommandBase
}

// SessionNew - create new session
//
// When force is set to true - all active sessions will be deleted prior to creating a new one if user reached session limit.
// Initial call to /sessin/new should always be performed with force set to false, to display special form, when sessions limit is reached.
// IVPN client apps have to set force to true only when customer clicks Log all other clients button.
type SessionNew struct {
	CommandBase
	AccountID  string
	ForceLogin bool
}

// SessionDelete logout from current device
type SessionDelete struct {
	CommandBase
}

// WireGuardGenerateNewKeys - generate WG keys
type WireGuardGenerateNewKeys struct {
	CommandBase
	OnlyUpdateIfNecessary bool
}

// WireGuardSetKeysRotationInterval -  change WG keys rotation interval
type WireGuardSetKeysRotationInterval struct {
	CommandBase
	Interval int64
}
