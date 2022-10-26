//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2022 Privatus Limited.
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

package service

import (
	"fmt"

	protocolTypes "github.com/ivpn/desktop-app/daemon/protocol/types"
)

type wifiStatus struct {
	WifiSsid       string
	WifiIsInsecure bool
}

type autoConnectReason int

const (
	OnDaemonStarted     autoConnectReason = iota
	OnUiClientConnected autoConnectReason = iota
	OnWifiChanged       autoConnectReason = iota
)

type trustedWiFiActionType int

const (
	NoAction trustedWiFiActionType = iota
	On       trustedWiFiActionType = iota
	Off      trustedWiFiActionType = iota
)

type automaticAction struct {
	Vpn      trustedWiFiActionType
	Firewall trustedWiFiActionType
}

func (a automaticAction) IsHasAction() bool {
	return a.Firewall != NoAction || a.Vpn != NoAction
}

func (s *Service) OnAuthenticatedClient(t protocolTypes.ClientTypeEnum) {
	if t != protocolTypes.ClientUi {
		// "auto-connect on app launch" is applicable only for UI client
		return
	}
	s.autoConnectIfRequired(OnUiClientConnected, nil)
}

// autoConnectIfRequired - checks if automatic connection required
// If connection needed - it calls _evtReceiver.RegisterConnectionRequest() which will be processed by the 'protocol'
// Params:
//
//	reason - the reason why this method is called
//	wifiInfo - current WiFi info. It can be 'nil', in this case, the function will check the WiFi info itself
func (s *Service) autoConnectIfRequired(reason autoConnectReason, wifiInfo *wifiStatus) error {

	prefs := s.Preferences()
	if !prefs.Session.IsLoggedIn() {
		return nil
	}

	// Check WiFi status (if not defined)
	if wifiInfo == nil {
		ssid, isInsecure := s.GetWiFiCurrentState()
		wifiInfo = &wifiStatus{WifiSsid: ssid, WifiIsInsecure: isInsecure}
	}

	//
	// Checking if new connection required
	//

	// Check "Trusted WiFi" actions
	action := s.getActionForWifiNetwork(*wifiInfo)
	if action.IsHasAction() {
		log.Info("Automatic connection manager: applying 'Trusted-WIFI' action...")
	}

	// Check "Auto-connect on APP/daemon launch" action
	// (skip when we are connected to a trusted network with "Disconnect VPN" action)
	if prefs.IsAutoconnectOnLaunch && action.Vpn != Off {
		if reason == OnDaemonStarted && prefs.IsAutoconnectOnLaunchDaemon {
			log.Info("Automatic connection manager: applying Auto-Connect on daemon Launch action...")
			action.Vpn = On
		} else if reason == OnUiClientConnected && !prefs.IsAutoconnectOnLaunchDaemon {
			log.Info("Automatic connection manager: applying Auto-Connect on app Launch action...")
			action.Vpn = On
		}
	}

	// Check Auto-connect 'On joining WiFi networks without encryption'
	if action.Vpn == Off &&
		prefs.WiFiControl.ConnectVPNOnInsecureNetwork &&
		wifiInfo.WifiIsInsecure &&
		s.isCanApplyWiFiActions() {

		log.Info("Automatic connection manager: applying Auto-Connect 'On joining WiFi networks without encryption'  action...")
		action.Vpn = On
	}

	//
	// Request new connection
	//

	var retErr error = nil
	connParams := prefs.LastConnectionParams

	// Firewall
	switch action.Firewall {
	case Off:
		log.Info("Automatic connection manager: disabling Firewall")
		s.SetKillSwitchState(false)
		connParams.FirewallOn = false // Ensure Firewall connection params is the same as in action
	case On:
		log.Info("Automatic connection manager: enabling Firewall")
		s.SetKillSwitchState(true)
		connParams.FirewallOn = true // Ensure Firewall connection params is the same as in action
	default:
	}

	// Vpn
	switch action.Vpn {
	case Off:
		if s.Connected() {
			log.Info("Automatic connection manager: disconnecting VPN")
			s.Disconnect()
		}
	case On:
		if !s.Connected() {
			log.Info("Automatic connection manager: connecting VPN")
			var err error
			const canFixParams bool = true
			connParams, err = s.ValidateConnectionParameters(connParams, canFixParams)
			if err != nil {
				log.Error(fmt.Sprintf("Auto-connection failed (bad connection parameters): %v", err))
			}

			retErr = s._evtReceiver.RegisterConnectionRequest(connParams)
		}
	default:
	}

	return retErr
}

func (s *Service) isCanApplyWiFiActions() bool {
	prefs := s.Preferences()
	if !prefs.WiFiControl.CanApplyInBackground && !s._evtReceiver.IsAnyAuthenticatedClientConnected() {
		// WiFi action not allowed: no UI client connected (CanApplyInBackground == false)
		return false
	}
	return true
}

func (s *Service) getActionForWifiNetwork(wifiInfo wifiStatus) (retAction automaticAction) {
	prefs := s.Preferences()
	if !prefs.Session.IsLoggedIn() {
		return
	}

	if !s.isCanApplyWiFiActions() {
		return
	}

	wifiParams := prefs.WiFiControl
	if !wifiParams.TrustedNetworksControl || wifiInfo.WifiSsid == "" {
		return
	}

	var isNetworkTrusted *bool // nil - no action

	// get config for ssid
	for _, w := range wifiParams.Networks {
		if w.SSID != wifiInfo.WifiSsid {
			continue
		}

		isNetworkTrusted = &w.IsTrusted
		break
	}

	if isNetworkTrusted == nil {
		// network not defined in settings. Using default configuration
		isNetworkTrusted = wifiParams.DefaultTrustStatusTrusted
	}

	if isNetworkTrusted == nil {
		return
	}

	if !*isNetworkTrusted {
		// UnTrusted
		if wifiParams.Actions.UnTrustedConnectVpn {
			retAction.Vpn = On
		}
		if wifiParams.Actions.UnTrustedEnableFirewall {
			retAction.Firewall = On
		}
	} else {
		// Trusted
		if wifiParams.Actions.TrustedDisconnectVpn {
			retAction.Vpn = Off
		}
		if wifiParams.Actions.TrustedDisableFirewall {
			retAction.Firewall = Off
		}
	}

	return
}
