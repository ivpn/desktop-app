package preferences

// AccountStatus conatins information about current account
type AccountStatus struct {
	Active         bool
	ActiveUntil    int64
	CurrentPlan    string
	PaymentMethod  string
	IsRenewable    bool
	WillAutoRebill bool
	IsFreeTrial    bool
	Capabilities   []string
	Upgradable     bool
	UpgradeToPlan  string
	UpgradeToURL   string
	Limit          int
}
