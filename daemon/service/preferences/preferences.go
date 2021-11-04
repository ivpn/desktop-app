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
	"time"

	"github.com/google/uuid"

	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/service/platform/filerights"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("sprefs")
}

const (
	// DefaultWGKeysInterval - Default WireGuard keys rotation interval
	DefaultWGKeysInterval = time.Hour * 24 * 1
)

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
	IsStopOnClientDisconnect bool
	IsObfsproxy              bool

	// split-tunnelling
	IsSplitTunnel   bool
	SplitTunnelApps []string

	// last known account status
	Session SessionStatus
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
func (p *Preferences) SetSession(accountID string,
	session string,
	vpnUser string,
	vpnPass string,
	wgPublicKey string,
	wgPrivateKey string,
	wgLocalIP string) {

	p.setSession(accountID, session, vpnUser, vpnPass, wgPublicKey, wgPrivateKey, wgLocalIP)
	p.SavePreferences()
}

// UpdateWgCredentials save wireguard credentials
func (p *Preferences) UpdateWgCredentials(wgPublicKey string, wgPrivateKey string, wgLocalIP string) {
	p.Session.updateWgCredentials(wgPublicKey, wgPrivateKey, wgLocalIP)
	p.SavePreferences()
}

// SavePreferences saves preferences
func (p *Preferences) SavePreferences() error {
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to save preferences file (json marshal error): %w", err)
	}

	settingsFile := platform.SettingsFile()
	if err := ioutil.WriteFile(settingsFile, data, 0600); err != nil { // read\write only for privileged user
		return err
	}

	// only for Windows: Golang is not able to change file permissins in Windows style
	if err := filerights.WindowsChmod(settingsFile, 0600); err != nil { // read\write only for privileged user
		return fmt.Errorf("failed to change settings-file permissions: %w", err)
	}

	return nil
}

// LoadPreferences loads preferences
func (p *Preferences) LoadPreferences() error {
	data, err := ioutil.ReadFile(platform.SettingsFile())

	if err != nil {
		return fmt.Errorf("failed to read preferences file: %w", err)
	}

	dataStr := string(data)
	if strings.Contains(dataStr, `"firewall_is_persistent"`) {
		// TODO: remove this old code
		// It is a first time loading preferences after IVPN Client upgrade from old version (<= v2.10.9)
		// Loading preferences with an old parameter names and types:
		type PreferencesOld struct {
			IsLogging                string `json:"enable_logging"`
			IsFwPersistant           string `json:"firewall_is_persistent"`
			IsFwAllowLAN             string `json:"firewall_allow_lan"`
			IsFwAllowLANMulticast    string `json:"firewall_allow_lan_multicast"`
			IsStopOnClientDisconnect string `json:"is_stop_server_on_client_disconnect"`
			IsObfsproxy              string `json:"enable_obfsproxy"`
		}
		oldStylePrefs := &PreferencesOld{}

		if err := json.Unmarshal(data, oldStylePrefs); err != nil {
			return err
		}

		p.IsLogging = oldStylePrefs.IsLogging == "1"
		p.IsFwPersistant = oldStylePrefs.IsFwPersistant == "1"
		p.IsFwAllowLAN = oldStylePrefs.IsFwAllowLAN == "1"
		p.IsFwAllowLANMulticast = oldStylePrefs.IsFwAllowLANMulticast == "1"
		p.IsStopOnClientDisconnect = oldStylePrefs.IsStopOnClientDisconnect == "1"
		p.IsObfsproxy = oldStylePrefs.IsObfsproxy == "1"

		return nil
	}

	//	We need to determine if some of variables are present in json at all.
	//	After the app upgrade new configuration parameters will not be saved in JSON jet.
	// 	In this case we will understand that we are able to set initial parameters for this values (e.g. platform-specific values)
	//
	//	The 'PreferencesValsCheck' contains SAME PROPERTIES NAMES which original properties object, but fields is pointers
	//	If field==nil => it is not exists in json.
	type PreferencesValsCheck struct {
		IsFwAllowApiServers *bool
	}
	var testJsonObj PreferencesValsCheck
	err = json.Unmarshal(data, &testJsonObj)
	if err != nil {
		return err
	}

	// Parse json onto preferences object
	err = json.Unmarshal(data, p)
	if err != nil {
		return err
	}

	// set initial values for a properties (if necessary)
	if testJsonObj.IsFwAllowApiServers == nil {
		// IsFwAllowApiServers was not initialized (it is a first start after application update)
		p.IsFwAllowApiServers = platform.FwInitialValueAllowApiServers()
	}

	// init WG properties
	if len(p.Session.WGPublicKey) == 0 || len(p.Session.WGPrivateKey) == 0 || len(p.Session.WGLocalIP) == 0 {
		p.Session.WGKeyGenerated = time.Time{}
	}

	if p.Session.WGKeysRegenInerval <= 0 {
		p.Session.WGKeysRegenInerval = DefaultWGKeysInterval
		log.Info(fmt.Sprintf("default value for preferences: WgKeysRegenIntervalDays=%v", p.Session.WGKeysRegenInerval))
		p.SavePreferences()
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
