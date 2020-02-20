package types

// PingServers request to ping servers
type PingServers struct {
	RetryCount int
	TimeOutMs  int
}

// KillSwitchSetAllowLANMulticastRequest enable\disable LAN multicast acces for kill-switch
type KillSwitchSetAllowLANMulticastRequest struct {
	AllowLANMulticast bool
}

// KillSwitchSetAllowLANRequest enable\disable LAN acces for kill-switch
type KillSwitchSetAllowLANRequest struct {
	AllowLAN bool
}

// SetPreferenceRequest sets daemon configuration parameter
type SetPreferenceRequest struct {
	Key   string
	Value string
}

// SetAlternateDNS request to set custom DNS
type SetAlternateDNS struct {
	DNS string
}

// KillSwitchSetEnabledRequest request to enable\disable kill-switch
type KillSwitchSetEnabledRequest struct {
	IsEnabled bool
}

// KillSwitchSetIsPersistentRequest request to mark kill-switch persistant
type KillSwitchSetIsPersistentRequest struct {
	IsPersistent bool
}

// Connect request to establish new VPN connection
type Connect struct {
	VpnType    int
	CurrentDNS string

	WireGuardParameters struct {
		InternalClientIP string
		LocalPrivateKey  string

		Port struct {
			Port int
		}

		EntryVpnServer struct {
			Hosts []struct {
				Host      string
				PublicKey string `json:"public_key"`
				LocalIP   string `json:"local_ip"`
			}
		}
	}

	OpenVpnParameters struct {
		EntryVpnServer struct {
			IPAddresses []string `json:"ip_addresses"`
		} 

		Username      string
		Password      string
		ProxyType     string
		ProxyAddress  string
		ProxyPort     int
		ProxyUsername string
		ProxyPassword string

		Port struct {
			Port     int
			Protocol int
		}
	}
}
