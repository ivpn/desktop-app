package types

// APIResponse - generic API response
type APIResponse struct {
	Status int `json:"status"` // status code
}

// APIErrorResponse generic IVPN API error
type APIErrorResponse struct {
	APIResponse
	Message string `json:"message,omitempty"` // Text description of the message
}

// ServiceStatusAPIResp account info
type ServiceStatusAPIResp struct {
	Active         bool     `json:"is_active"`
	ActiveUntil    int64    `json:"active_until"`
	CurrentPlan    string   `json:"current_plan"`
	PaymentMethod  string   `json:"payment_method"`
	IsRenewable    bool     `json:"is_renewable"`
	WillAutoRebill bool     `json:"will_auto_rebill"`
	IsFreeTrial    bool     `json:"is_on_free_trial"`
	Capabilities   []string `json:"capabilities"`
	Upgradable     bool     `json:"upgradable"`
	UpgradeToPlan  string   `json:"upgrade_to_plan"`
	UpgradeToURL   string   `json:"upgrade_to_url"`
	Limit          int      `json:"limit"` // applicable for 'session limit' error
}

// SessionNewResponse information about created session
type SessionNewResponse struct {
	APIErrorResponse
	Token       string `json:"token"`
	VpnUsername string `json:"vpn_username"`
	VpnPassword string `json:"vpn_password"`

	ServiceStatus ServiceStatusAPIResp `json:"service_status"`

	WireGuard struct {
		Status    int    `json:"status"`
		Message   string `json:"message,omitempty"`
		IPAddress string `json:"ip_address,omitempty"`
	} `json:"wireguard"`
}

// SessionNewErrorLimitResponse information about session limit error
type SessionNewErrorLimitResponse struct {
	APIErrorResponse
	SessionLimitData ServiceStatusAPIResp `json:"data"`
}

// SessionsWireGuardResponse Sessions WireGuard response
type SessionsWireGuardResponse struct {
	APIErrorResponse
	IPAddress string `json:"ip_address,omitempty"`
}

// SessionStatusResponse session status response
type SessionStatusResponse struct {
	APIErrorResponse
	ServiceStatus ServiceStatusAPIResp `json:"service_status"`
}
