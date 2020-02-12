package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

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

	return json.Unmarshal(data, s)
}

/*
func (s *preferences) IsEnabledLogging() bool {
	return s.isEnabledLogging
}
func (s *preferences) SetIsEnabledLogging(val bool) {
	s.isEnabledLogging = val
}*/
