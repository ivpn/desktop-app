package types

import (
	commonTypes "github.com/ivpn/desktop-app-daemon/api/common/types"
)

// SessionsAuthenticateFullResponse - full Sessions Authenticate response
type SessionsAuthenticateFullResponse struct {
	commonTypes.SessionsAuthenticateResponse
	ErrorMessage     string                `json:"message,omitempty"`
	SessionLimitData SessionLimitErrorData `json:"data,omitempty"`
}

// SessionLimitErrorData - full session limit error description
type SessionLimitErrorData struct {
	Limit         int    `json:"limit"`
	CurrentPlan   string `json:"current_plan"`
	PaymentMethod string `json:"payment_method"`
	Upgradable    bool   `json:"upgradable"`
	UpgradeToPlan string `json:"upgrade_to_plan"`
	UpgradeToURL  string `json:"upgrade_to_url"`
}
