package types

// PingServers request to ping servers
type PingServers struct {
	RetryCount int `json:"retryCount"`
	TimeOutMs  int `json:"timeOutMs"`
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
