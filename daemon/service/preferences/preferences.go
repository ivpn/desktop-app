//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/service/platform"
)

var log *logger.Logger
var mutexRW sync.RWMutex

func init() {
	log = logger.NewLogger("sprefs")
}

const (
	// DefaultWGKeysInterval - Default WireGuard keys rotation interval
	DefaultWGKeysInterval = time.Hour * 24 * 1
)

// UserPreferences - IVPN service preferences which can be exposed to client
type UserPreferences struct {
	// NOTE: update this type when adding new preferenvces which can be exposed for clients
	// ...

	// The platform-specific preferences
	Platform PlatformSpecificUserPrefs
}

// Preferences - IVPN service preferences
type Preferences struct {
	// SettingsSessionUUID is unique for Preferences object
	// It allow to detect situations when settings was erased (created new Preferences object)
	SettingsSessionUUID      string
	IsLogging                bool
	IsFwPersistant           bool
	IsFwAllowLAN             bool
	IsFwAllowLANMulticast    bool
	IsFwAllowApiServers      bool
	FwUserExceptions         string // Firewall exceptions: comma separated list of IP addresses (masks) in format: x.x.x.x[/xx]
	IsStopOnClientDisconnect bool
	IsObfsproxy              bool
	IsAutoconnectOnLaunch    bool // when 'true' - UI app (not the daemon!) will perform automation connection on app launch

	// split-tunnelling
	IsSplitTunnel   bool
	SplitTunnelApps []string

	// last known account status
	Session SessionStatus
	Account AccountStatus

	// NOTE: update this type when adding new preferences which can be exposed to clients
	UserPrefs UserPreferences
}

func Create() *Preferences {
	// init default values
	return &Preferences{
		// SettingsSessionUUID is unique for Preferences object
		// It allow to detect situations when settings was erased (created new Preferences object)
		SettingsSessionUUID: uuid.New().String(),
		IsFwAllowApiServers: true,
	}
}

// SetSession save account credentials
func (p *Preferences) SetSession(accountInfo AccountStatus,
	accountID string,
	session string,
	vpnUser string,
	vpnPass string,
	wgPublicKey string,
	wgPrivateKey string,
	wgLocalIP string) {

	if len(session) == 0 || len(accountID) == 0 {
		p.Account = AccountStatus{}
	} else {
		p.Account = accountInfo
	}

	p.setSession(accountID, session, vpnUser, vpnPass, wgPublicKey, wgPrivateKey, wgLocalIP)
	p.SavePreferences()
}

func (p *Preferences) UpdateAccountInfo(acc AccountStatus) {
	if len(p.Session.AccountID) == 0 || len(p.Session.Session) == 0 {
		acc = AccountStatus{}
	}
	p.Account = acc
	p.SavePreferences()
}

// UpdateWgCredentials save wireguard credentials
func (p *Preferences) UpdateWgCredentials(wgPublicKey string, wgPrivateKey string, wgLocalIP string) {
	p.Session.updateWgCredentials(wgPublicKey, wgPrivateKey, wgLocalIP)
	p.SavePreferences()
}

// SavePreferences saves preferences
func (p *Preferences) SavePreferences() error {
	mutexRW.Lock()
	defer mutexRW.Unlock()

	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to save preferences file (json marshal error): %w", err)
	}

	settingsFile := platform.SettingsFile()
	if err := helpers.WriteFile(settingsFile, data, 0600); err != nil { // read\write only for privileged user
		return err
	}

	return nil
}

// LoadPreferences loads preferences
func (p *Preferences) LoadPreferences() error {
	mutexRW.RLock()
	defer mutexRW.RUnlock()

	data, err := ioutil.ReadFile(platform.SettingsFile())

	if err != nil {
		return fmt.Errorf("failed to read preferences file: %w", err)
	}

	// Parse json onto preferences object
	err = json.Unmarshal(data, p)
	if err != nil {
		return err
	}

	// init WG properties
	if len(p.Session.WGPublicKey) == 0 || len(p.Session.WGPrivateKey) == 0 || len(p.Session.WGLocalIP) == 0 {
		p.Session.WGKeyGenerated = time.Time{}
	}

	if p.Session.WGKeysRegenInerval <= 0 {
		p.Session.WGKeysRegenInerval = DefaultWGKeysInterval
		log.Info(fmt.Sprintf("default value for preferences: WgKeysRegenIntervalDays=%v", p.Session.WGKeysRegenInerval))
	}

	return nil
}

func (p *Preferences) setSession(accountID string,
	session string,
	vpnUser string,
	vpnPass string,
	wgPublicKey string,
	wgPrivateKey string,
	wgLocalIP string) {

	p.Session = SessionStatus{
		AccountID:          strings.TrimSpace(accountID),
		Session:            strings.TrimSpace(session),
		OpenVPNUser:        strings.TrimSpace(vpnUser),
		OpenVPNPass:        strings.TrimSpace(vpnPass),
		WGKeysRegenInerval: p.Session.WGKeysRegenInerval} // keep 'WGKeysRegenInerval' from previous Session object

	if p.Session.WGKeysRegenInerval <= 0 {
		p.Session.WGKeysRegenInerval = DefaultWGKeysInterval
	}

	p.Session.updateWgCredentials(wgPublicKey, wgPrivateKey, wgLocalIP)
}
