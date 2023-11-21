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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"os"

	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/obfsproxy"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	service_types "github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/version"
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

type LinuxSpecificUserPrefs struct {
	// If true - use old style DNS management mechanism
	// by direct modifying file '/etc/resolv.conf'
	IsDnsMgmtOldStyle bool
}

// UserPreferences - IVPN service preferences which can be exposed to client
type UserPreferences struct {
	// NOTE: update this type when adding new preferences which can be exposed for clients
	// ...

	// The platform-specific preferences
	Linux LinuxSpecificUserPrefs
}

// Preferences - IVPN service preferences
type Preferences struct {
	// The daemon version that saved this data.
	// Can be used to determine the format version (e.g., on the first app start after an upgrade).
	Version string
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

	// IsAutoconnectOnLaunch: if 'true' - daemon will perform automatic connection (see 'IsAutoconnectOnLaunchDaemon' for details)
	IsAutoconnectOnLaunch bool
	// IsAutoconnectOnLaunchDaemon:
	//	false - means the daemon applies operation 'IsAutoconnectOnLaunch' only when UI app connected
	//	true - means the daemon applies operation 'IsAutoconnectOnLaunch':
	//		-	when UI app connected
	//		-	after daemon initialization
	//		-	on user session LogOn
	IsAutoconnectOnLaunchDaemon bool

	// split-tunnelling
	IsSplitTunnel             bool // Split Tunnel on/off
	SplitTunnelApps           []string
	SplitTunnelInversed       bool // Inverse Split Tunnel: only 'splitted' apps use VPN tunnel (applicable only when IsSplitTunnel=true)
	SplitTunnelAnyDns         bool // (only for Inverse Split Tunnel) When false: Allow only DNS servers specified by the IVPN application
	SplitTunnelAllowWhenNoVpn bool // (only for Inverse Split Tunnel) Allow connectivity for Split Tunnel apps when VPN is disabled

	// last known account status
	Session SessionStatus
	Account AccountStatus

	// NOTE: update this type when adding new preferences which can be exposed to clients
	UserPrefs UserPreferences

	LastConnectionParams service_types.ConnectionParams
	WiFiControl          WiFiParams
}

func Create() *Preferences {
	// init default values
	return &Preferences{
		// SettingsSessionUUID is unique for Preferences object
		// It allow to detect situations when settings was erased (created new Preferences object)
		SettingsSessionUUID: uuid.New().String(),
		IsFwAllowApiServers: true,
		WiFiControl:         WiFiParamsCreate(),
	}
}

// IsInverseSplitTunneling returns:
// 'true' (default behavior) - when the VPN connection should be configured as the default route on a system,
// 'false' - when the default route should remain unchanged	(e.g., for inverse split-tunneling,	when the VPN tunnel is used only by 'split' apps).
func (p *Preferences) IsInverseSplitTunneling() bool {
	if !p.IsSplitTunnel {
		return false
	}

	return p.SplitTunnelInversed
}

// SetSession save account credentials
func (p *Preferences) SetSession(accountInfo AccountStatus,
	accountID string,
	session string,
	vpnUser string,
	vpnPass string,
	wgPublicKey string,
	wgPrivateKey string,
	wgLocalIP string,
	wgPreSharedKey string) {

	if len(session) == 0 || len(accountID) == 0 {
		p.Account = AccountStatus{}
	} else {
		p.Account = accountInfo
	}

	p.setSession(accountID, session, vpnUser, vpnPass, wgPublicKey, wgPrivateKey, wgLocalIP, wgPreSharedKey)
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
func (p *Preferences) UpdateWgCredentials(wgPublicKey string, wgPrivateKey string, wgLocalIP string, wgPresharedKey string) {
	p.Session.updateWgCredentials(wgPublicKey, wgPrivateKey, wgLocalIP, wgPresharedKey)
	p.SavePreferences()
}

func (p *Preferences) getTempFilePath() string {
	return platform.SettingsFile() + ".tmp"
}

// SavePreferences saves preferences
func (p *Preferences) SavePreferences() error {
	mutexRW.Lock()
	defer mutexRW.Unlock()

	p.Version = version.Version()

	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to save preferences file (json marshal error): %w", err)
	}

	settingsFile := platform.SettingsFile()
	settingsFileMode := os.FileMode(0600) // read\write only for privileged user

	// Save the settings file to a temporary file. This is necessary to prevent data loss in case of a power failure
	// or other system operations that could interrupt the saving process (e.g., a crash or process termination).
	// If the settings file becomes corrupted, the daemon will attempt to restore it from the temporary file.
	settingsFileTmp := p.getTempFilePath()
	if err := helpers.WriteFile(p.getTempFilePath(), data, settingsFileMode); err != nil { // read\write only for privileged user
		return err
	}

	// save settings file
	if err := helpers.WriteFile(settingsFile, data, settingsFileMode); err != nil { // read\write only for privileged user
		return err
	}

	// Remove temp file after successful saving
	os.Remove(settingsFileTmp)

	return nil
}

// LoadPreferences loads preferences
func (p *Preferences) LoadPreferences() error {
	mutexRW.RLock()
	defer mutexRW.RUnlock()

	funcReadPreferences := func(filePath string) (data []byte, err error) {
		data, err = os.ReadFile(filePath)
		if err != nil {
			return data, fmt.Errorf("failed to read preferences file: %w", err)
		}

		// Parse json into preferences object
		err = json.Unmarshal(data, p)
		if err != nil {
			return data, err
		}
		return data, nil
	}

	data, err := funcReadPreferences(platform.SettingsFile())
	if err != nil {
		log.Error(fmt.Sprintf("failed to read preferences file: %v", err))
		// Try to read from temp file, if exists (this is necessary to prevent data loss in case of a power failure)
		var errTmp error
		data, errTmp = funcReadPreferences(p.getTempFilePath())
		if errTmp != nil {
			return err // return original error
		}
		log.Info("Preferences file was restored from temporary file")
	}

	// init WG properties
	if len(p.Session.WGPublicKey) == 0 || len(p.Session.WGPrivateKey) == 0 || len(p.Session.WGLocalIP) == 0 {
		p.Session.WGKeyGenerated = time.Time{}
	}

	if p.Session.WGKeysRegenInerval <= 0 {
		p.Session.WGKeysRegenInerval = DefaultWGKeysInterval
		log.Info(fmt.Sprintf("default value for preferences: WgKeysRegenIntervalDays=%v", p.Session.WGKeysRegenInerval))
	}

	// *** Compatibility with old versions ***

	// Convert parameters from v3.10.23 (and releases older than 2023-05-15)
	// The default antitracker blocklist was "OSID Big". So keep it for old users who upgrade.
	//
	// We are here because the preferences file was exists, so it is not a new installation	(it is upgrade),
	// and if the AntiTrackerBlockListName is empty - it means that it is first upgrade to version which support multiple blocklists.
	if p.LastConnectionParams.Metadata.AntiTracker.AntiTrackerBlockListName == "" {
		log.Info("It looks like this is the first upgrade to the version which supports AntiTracker blocklists. Keep the old default blocklist name 'Oisdbig'.")
		p.LastConnectionParams.Metadata.AntiTracker.AntiTrackerBlockListName = "Oisdbig"
	}

	// Convert parameters from v3.11.15 (and releases older than 2023-08-07)
	if compareVersions(p.Version, "3.11.15") <= 0 {
		// if upgrading from "3.11.15" or older version

		// A new option, WiFiControl.Actions.UnTrustedBlockLan, was introduced.
		// It is 'true' by default. However, older versions did not have this functionality.
		// Therefore, for users upgrading from v3.11.15, it must be disabled.
		p.WiFiControl.Actions.UnTrustedBlockLan = false

		// Obfsproxy configuration was moved to 'LastConnectionParams->OpenVpnParameters' section
		type tmp_type_Settings_v3_11_15 struct {
			Obfs4proxy struct {
				Obfs4Iat obfsproxy.Obfs4IatMode
				Version  obfsproxy.ObfsProxyVersion
			}
		}
		var tmp_Settings_v3_11_15 tmp_type_Settings_v3_11_15
		err = json.Unmarshal(data, &tmp_Settings_v3_11_15)
		if err == nil && tmp_Settings_v3_11_15.Obfs4proxy.Version > obfsproxy.None {
			p.LastConnectionParams.OpenVpnParameters.Obfs4proxy = obfsproxy.Config{
				Version:  tmp_Settings_v3_11_15.Obfs4proxy.Version,
				Obfs4Iat: tmp_Settings_v3_11_15.Obfs4proxy.Obfs4Iat,
			}

		}
	}

	return nil
}

func (p *Preferences) setSession(accountID string,
	session string,
	vpnUser string,
	vpnPass string,
	wgPublicKey string,
	wgPrivateKey string,
	wgLocalIP string,
	wgPreSharedKey string) {

	p.Session = SessionStatus{
		AccountID:          strings.TrimSpace(accountID),
		Session:            strings.TrimSpace(session),
		OpenVPNUser:        strings.TrimSpace(vpnUser),
		OpenVPNPass:        strings.TrimSpace(vpnPass),
		WGKeysRegenInerval: p.Session.WGKeysRegenInerval} // keep 'WGKeysRegenInerval' from previous Session object

	if p.Session.WGKeysRegenInerval <= 0 {
		p.Session.WGKeysRegenInerval = DefaultWGKeysInterval
	}

	p.Session.updateWgCredentials(wgPublicKey, wgPrivateKey, wgLocalIP, wgPreSharedKey)
}

// compareVersions compares two version strings in the format "XX.XX.XX..."
// and returns -1 if version1 is older, 1 if version1 is newer,
// and 0 if both versions are equal.
func compareVersions(version1, version2 string) int {
	v1Parts := strings.Split(version1, ".")
	v2Parts := strings.Split(version2, ".")

	for i := 0; i < len(v1Parts) && i < len(v2Parts); i++ {
		v1Part, _ := strconv.Atoi(v1Parts[i])
		v2Part, _ := strconv.Atoi(v2Parts[i])

		if v1Part < v2Part {
			return -1
		} else if v1Part > v2Part {
			return 1
		}
	}

	if len(v1Parts) < len(v2Parts) {
		return -1
	} else if len(v1Parts) > len(v2Parts) {
		return 1
	}

	return 0
}
