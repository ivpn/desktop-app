package types

// SessionsAuthenticateResponse Sessions Authenticate response
type SessionsAuthenticateResponse struct {
	Status        int                       `json:"status"`
	Token         string                    `json:"token"`
	VpnUsername   string                    `json:"vpn_username"`
	VpnPassword   string                    `json:"vpn_password"`
	ServiceStatus SessionStatusResponse     `json:"service_status"`
	WireGuard     SessionsWireGuardResponse `json:"wireguard"`
}

// SessionsWireGuardResponse Sessions WireGuard response
type SessionsWireGuardResponse struct {
	Status    int    `json:"status"`
	Message   string `json:"message,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// SessionsStatusResponse Sessions Status response
type SessionsStatusResponse struct {
	Status        int                   `json:"status"`
	ServiceStatus SessionStatusResponse `json:"service_status"`
}

// SessionStatusResponse Sessions Status response
type SessionStatusResponse struct {
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
}

// SessionSuccessResponse Sessions Delete response
type SessionSuccessResponse struct {
	Status int `json:"status"`
}
