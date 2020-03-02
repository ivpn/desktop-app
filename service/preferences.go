package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ivpn/desktop-app-daemon/service/platform"
)

// Preferences - IVPN service preferences
type Preferences struct {
	IsLogging                bool
	IsFwPersistant           bool
	IsFwAllowLAN             bool
	IsFwAllowLANMulticast    bool
	IsStopOnClientDisconnect bool
	IsObfsproxy              bool
	OpenVpnExtraParameters   string

	AccountID string
	Session   string
	VPNUser   string
	VPNPass   string

	WGPublicKey  string
	WGPrivateKey string
	WGLocalIP    string
}

// SetCredentials save account credentials
func (s *Preferences) SetCredentials(accountID string,
	session string,
	vpnUser string,
	vpnPass string,
	wgPublicKey string,
	wgPrivateKey string,
	wgLocalIP string) {

	s.AccountID = strings.TrimSpace(accountID)
	s.Session = strings.TrimSpace(session)
	s.VPNUser = strings.TrimSpace(vpnUser)
	s.VPNPass = strings.TrimSpace(vpnPass)

	s.WGPublicKey = strings.TrimSpace(wgPublicKey)
	s.WGPrivateKey = strings.TrimSpace(wgPrivateKey)
	s.WGLocalIP = strings.TrimSpace(wgLocalIP)
	s.savePreferences()
}

// Credentials account credentials
func (s *Preferences) Credentials() (accountID string,
	session string,
	vpnUser string,
	vpnPass string,
	wgPublicKey string,
	wgPrivateKey string) {

	return s.AccountID, s.Session, s.VPNUser, s.VPNPass, s.WGPublicKey, s.WGPrivateKey
}

// UpdateWgCredentials save wireguard credentials
func (s *Preferences) UpdateWgCredentials(wgPublicKey string, wgPrivateKey string) {
	s.WGPublicKey = wgPublicKey
	s.WGPrivateKey = wgPrivateKey

	s.savePreferences()
}

func (s *Preferences) savePreferences() error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to save preferences file (json marshal error): %w", err)
	}

	return ioutil.WriteFile(platform.SettingsFile(), data, 0644)
}

func (s *Preferences) loadPreferences() error {
	data, err := ioutil.ReadFile(platform.SettingsFile())

	if err != nil {
		return fmt.Errorf("failed to read preferences file: %w", err)
	}

	dataStr := string(data)
	if strings.Contains(dataStr, `"firewall_is_persistent"`) {
		// It is a first time loading preferences after IVPN Client upgrade from old version (<= v2.10.9)
		// Loading preferences with an old parameter names and types:
		type PreferencesOld struct {
			IsLogging                string `json:"enable_logging"`
			IsFwPersistant           string `json:"firewall_is_persistent"`
			IsFwAllowLAN             string `json:"firewall_allow_lan"`
			IsFwAllowLANMulticast    string `json:"firewall_allow_lan_multicast"`
			IsStopOnClientDisconnect string `json:"is_stop_server_on_client_disconnect"`
			IsObfsproxy              string `json:"enable_obfsproxy"`
			OpenVpnExtraParameters   string `json:"open_vpn_extra_parameters"`
		}
		oldStylePrefs := &PreferencesOld{}

		if err := json.Unmarshal(data, oldStylePrefs); err != nil {
			return err
		}

		s.IsLogging = oldStylePrefs.IsLogging == "1"
		s.IsFwPersistant = oldStylePrefs.IsFwPersistant == "1"
		s.IsFwAllowLAN = oldStylePrefs.IsFwAllowLAN == "1"
		s.IsFwAllowLANMulticast = oldStylePrefs.IsFwAllowLANMulticast == "1"
		s.IsStopOnClientDisconnect = oldStylePrefs.IsStopOnClientDisconnect == "1"
		s.IsObfsproxy = oldStylePrefs.IsObfsproxy == "1"
		s.OpenVpnExtraParameters = oldStylePrefs.OpenVpnExtraParameters

		return nil
	}

	return json.Unmarshal(data, s)
}
