//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package preferences

import (
	"fmt"
	"strings"
)

type Capability string

const (
	MultiHop Capability = "multihop"
)

// AccountStatus contains information about current account
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

func (a AccountStatus) IsInitialized() bool {
	return len(a.CurrentPlan) > 0 || len(a.Capabilities) > 0
}

func (a AccountStatus) IsHasCapability(cap Capability) bool {
	for _, c := range a.Capabilities {
		if strings.ToLower(c) == string(cap) {
			return true
		}
	}
	return false
}

func (a AccountStatus) IsCanConnectMultiHop() error {
	if !a.IsInitialized() {
		// It could be that account status is not known. We allow MH in this case.
		// It can happen on upgrading from an old version (which did not keep s._preferences.Account)
		return nil
	}

	if a.IsHasCapability(MultiHop) {
		return nil
	}
	return fmt.Errorf("MultiHop connections are not allowed for the current subscription plan. Please upgrade your subscription to Pro")
}
