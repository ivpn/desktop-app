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

package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	apiTypes "github.com/ivpn/desktop-app/daemon/api/types"
	protocolTypes "github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	"github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/vpn"
	"github.com/ivpn/desktop-app/daemon/wifiNotifier"
)

type autoConnectReason int

const (
	OnDaemonStarted     autoConnectReason = iota
	OnUiClientConnected autoConnectReason = iota
	OnSessionLogon      autoConnectReason = iota
	OnWifiChanged       autoConnectReason = iota
)

func (cr autoConnectReason) ToString() string {
	switch cr {
	case OnDaemonStarted:
		return "DaemonLaunch"
	case OnUiClientConnected:
		return "UIAppLaunch"
	case OnWifiChanged:
		return "WiFiChanged"
	case OnSessionLogon:
		return "UserSessionLogon"
	default:
		return "<unknown>"
	}
}

type actionTypeVpn int

const (
	VPN_NoAction actionTypeVpn = iota
	VPN_On       actionTypeVpn = iota
	VPN_Off      actionTypeVpn = iota
)

type actionTypeFirewall int

const (
	FW_NoAction        actionTypeFirewall = iota
	FW_On              actionTypeFirewall = iota
	FW_Off             actionTypeFirewall = iota
	FW_On_and_blockLan actionTypeFirewall = iota
)

type automaticAction struct {
	Vpn      actionTypeVpn
	Firewall actionTypeFirewall
}

type lastProcessedWiFiInfo struct {
	wifi   wifiNotifier.WifiInfo
	params preferences.WiFiParams
}

var autoconnectLastProcessedWifi lastProcessedWiFiInfo

func (a automaticAction) IsHasAction() bool {
	return a.Firewall != FW_NoAction || a.Vpn != VPN_NoAction
}

func (s *Service) OnAuthenticatedClient(t protocolTypes.ClientTypeEnum) {
	if t != protocolTypes.ClientUi {
		// "auto-connect on app launch" is applicable only for UI client
		return
	}
	s.autoConnectIfRequired(OnUiClientConnected, nil)
}

func (s *Service) isTrustedWifiForcingToBlockLan(wifiInfoPtr *wifiNotifier.WifiInfo) bool {
	prefs := s.Preferences()
	if !prefs.Session.IsLoggedIn() {
		return false
	}

	if !s._evtReceiver.IsCanDoBackgroundAction() {
		return false
	}

	var wifiInfo wifiNotifier.WifiInfo
	// Check WiFi status (if not defined)
	if wifiInfoPtr == nil {
		wifiInfo = s.GetWiFiCurrentState()
	} else {
		wifiInfo = *wifiInfoPtr
	}

	action := s.getActionForWifiNetwork(wifiInfo)

	return action.Firewall == FW_On_and_blockLan
}

// autoConnectIfRequired - checks if automatic connection required
// If connection needed - it calls _evtReceiver.RegisterConnectionRequest() which will be processed by the 'protocol'
// Params:
//
//	reason - the reason why this method is called
//	wifiInfo - current WiFi info. It can be 'nil', in this case, the function will check the WiFi info itself
func (s *Service) autoConnectIfRequired(reason autoConnectReason, wifiInfoPtr *wifiNotifier.WifiInfo) error {
	prefs := s.Preferences()
	if !prefs.Session.IsLoggedIn() {
		return nil
	}

	if !s._evtReceiver.IsCanDoBackgroundAction() {
		return nil // background actions not allowed (e.g. due to EAA enabled)
	}

	var wifiInfo wifiNotifier.WifiInfo
	// Check WiFi status (if not defined)
	if wifiInfoPtr == nil {
		wifiInfo = s.GetWiFiCurrentState()
	} else {
		wifiInfo = *wifiInfoPtr
	}

	// Check if WiFi already processed
	isWifiProcessedAlready := false

	currWiFi := lastProcessedWiFiInfo{wifi: wifiInfo, params: prefs.WiFiControl}
	lastWifi := autoconnectLastProcessedWifi

	if reflect.DeepEqual(lastWifi, currWiFi) {
		// this wifi network change has been processed already
		if reason == OnWifiChanged {
			return nil
		}
		isWifiProcessedAlready = true
	}
	autoconnectLastProcessedWifi = currWiFi

	isLanActionsProcessed := false
	defer func() {
		if isLanActionsProcessed {
			return
		}
		// Check if the untrusted WiFi settings were forced to block LAN.
		// We have to restore LAN connectivity if there is not required to block LAN for current network
		prevSettingsBlockLan := lastWifi.params.TrustedNetworksControl && lastWifi.params.Actions.UnTrustedBlockLan
		currSettingsBlockLan := prefs.WiFiControl.TrustedNetworksControl && prefs.WiFiControl.Actions.UnTrustedBlockLan
		if prefs.IsFwAllowLAN && (prevSettingsBlockLan || currSettingsBlockLan) {
			if err := s.applyKillSwitchAllowLAN(&wifiInfo); err != nil {
				log.Info(fmt.Sprintf("Automatic connection manager: failed to restore Firewall rules to allow LAN: %s", err.Error()))
			}
		}
	}()

	//
	// Checking if new connection required
	//

	// Check "Trusted WiFi" actions
	action := s.getActionForWifiNetwork(wifiInfo)

	isVpnOffRequired := false
	if isWifiProcessedAlready {
		// For already processed networks - keep only 'vpn-off' action (it works in combination with 'IsAutoconnectOnLaunch')
		// Clean other actions for this network (because they were applied already)
		isVpnOffRequired = action.Vpn == VPN_Off
		action = automaticAction{}
	}

	if action.IsHasAction() {
		log.Info("Automatic connection manager: applying 'Trusted-WiFi' action...")
	}

	// Check "Auto-connect on APP/daemon launch" action
	// (skip when we are connected to a trusted network with "Disconnect VPN" action)
	if prefs.IsAutoconnectOnLaunch && !isVpnOffRequired {
		if !s.Connected() && action.Vpn != VPN_On {
			if (reason == OnDaemonStarted || reason == OnSessionLogon) && prefs.IsAutoconnectOnLaunchDaemon {
				log.Info(fmt.Sprintf("Automatic connection manager: applying Auto-Connect action on '%s' ...", reason.ToString()))
				action.Vpn = VPN_On
			} else if reason == OnUiClientConnected {
				log.Info(fmt.Sprintf("Automatic connection manager: applying Auto-Connect action on '%s' ...", reason.ToString()))
				action.Vpn = VPN_On
			}
		}
	}

	// Check Auto-connect 'On joining WiFi networks without encryption'
	if !isWifiProcessedAlready &&
		action.Vpn == VPN_NoAction &&
		prefs.WiFiControl.ConnectVPNOnInsecureNetwork &&
		wifiInfo.IsInsecure {

		if s.isCanApplyWiFiActions() {
			log.Info("Automatic connection manager: applying Auto-Connect 'On joining WiFi networks without encryption' action...")
			action.Vpn = VPN_On
		}
	}

	if !action.IsHasAction() {
		// No actions defined. Nothing to do here.
		return nil
	}

	//
	// Apply actions (Firewall, VPN ...)
	//

	var retErr error = nil
	connParams := prefs.LastConnectionParams

	// Firewall
	switch action.Firewall {
	case FW_Off:
		log.Info("Automatic connection manager: disabling Firewall")
		if retErr = s.SetKillSwitchState(false); retErr != nil {
			log.Error("Auto connection: disabling Firewall: ", retErr)
		}
		connParams.FirewallOn = false // Ensure Firewall connection params is the same as in action
		connParams.FirewallOnDuringConnection = false
	case FW_On:
		log.Info("Automatic connection manager: enabling Firewall")
		if retErr = s.SetKillSwitchState(true); retErr != nil {
			log.Error("Auto connection: enabling Firewall: ", retErr)
		}
		connParams.FirewallOn = true // Ensure Firewall connection params is the same as in action
	case FW_On_and_blockLan:
		log.Info("Automatic connection manager: enabling Firewall and block LAN")
		if retErr = s.SetKillSwitchState(true); retErr != nil {
			log.Error("Auto connection: enabling Firewall: ", retErr)
		}
		if retErr = s.applyKillSwitchAllowLAN(&wifiInfo); retErr != nil {
			log.Error("Auto connection: Firewall (block LAN): ", retErr)
		}
		connParams.FirewallOn = true // Ensure Firewall connection params is the same as in action
		isLanActionsProcessed = true // for defferred function: to not re-appy LAN settings

	default:
	}

	// Vpn
	switch action.Vpn {
	case VPN_Off:
		if s.Connected() {
			log.Info("Automatic connection manager: disconnecting VPN")
			if retErr = s.Disconnect(); retErr != nil {
				log.Error("Auto connection: disconnecting: ", retErr)
			}
		}
	case VPN_On:
		if !s.Connected() {
			log.Info("Automatic connection manager: connecting VPN")

			connParams, retErr = s.updateParamsAccordingToMetadata(connParams)
			if retErr != nil {
				log.Info("[WARNING] Auto connection: failed updating connection parameters: ", retErr)
			}

			const canFixParams bool = true
			if connParams, retErr = s.ValidateConnectionParameters(connParams, canFixParams); retErr != nil {
				log.Error("Auto connection: error validating connection parameters: ", retErr)
				return retErr
			}

			if retErr = s._evtReceiver.RegisterConnectionRequest(connParams); retErr != nil {
				log.Error("Auto connection: connecting: ", retErr)
			}

		}
	default:
	}

	return retErr
}

func (s *Service) isCanApplyWiFiActions() bool {
	prefs := s.Preferences()
	const onlyUiClients = true
	if !prefs.WiFiControl.CanApplyInBackground && !s._evtReceiver.IsClientConnected(onlyUiClients) {
		// WiFi action not allowed: no UI client connected (CanApplyInBackground == false)
		return false
	}
	return true
}

func (s *Service) getActionForWifiNetwork(wifiInfo wifiNotifier.WifiInfo) (retAction automaticAction) {
	prefs := s.Preferences()
	if !prefs.Session.IsLoggedIn() {
		return
	}

	if !s.isCanApplyWiFiActions() {
		return
	}

	wifiParams := prefs.WiFiControl
	if !wifiParams.TrustedNetworksControl || wifiInfo.SSID == "" {
		return
	}

	var isNetworkTrusted *bool // nil - no action

	// get config for ssid
	for _, w := range wifiParams.Networks {
		if w.SSID != wifiInfo.SSID {
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
			retAction.Vpn = VPN_On
		}
		if wifiParams.Actions.UnTrustedEnableFirewall {
			retAction.Firewall = FW_On
		}
		if wifiParams.Actions.UnTrustedBlockLan {
			retAction.Firewall = FW_On_and_blockLan
		}
	} else {
		// Trusted
		if wifiParams.Actions.TrustedDisconnectVpn {
			retAction.Vpn = VPN_Off
		}
		if wifiParams.Actions.TrustedDisableFirewall {
			retAction.Firewall = FW_Off
		}
	}

	return
}

// updateParamsAccordingToMetadata - update Entry/Exit servers if connection requires 'Fastest' or 'Random'
func (s *Service) updateParamsAccordingToMetadata(params types.ConnectionParams) (types.ConnectionParams, error) {
	if params.Metadata.ServerSelectionEntry == types.Default && params.Metadata.ServerSelectionExit == types.Default {
		return params, nil
	}

	allServers, err := s.ServersList()
	if err != nil {
		return params, err
	}

	// ENTRY server
	if params.Metadata.ServerSelectionEntry != types.Default {
		// Get countryCode of exit server (do not choose exit server from same country)
		exitSvrCountryCode := ""
		if params.IsMultiHop() && params.Metadata.ServerSelectionExit != types.Default {
			exitSvrCountryCode = s.getServerCountryCode(params, false)
		}

		if params.VpnType == vpn.OpenVPN {
			//OpenVPN
			applicableEntryServers := []apiTypes.OpenvpnServerInfo{}
			if exitSvrCountryCode == "" {
				applicableEntryServers = allServers.OpenvpnServers
			} else {
				for _, s := range allServers.OpenvpnServers {
					if s.CountryCode == exitSvrCountryCode {
						continue // exclude exit server from the same country as Exit server
					}
					applicableEntryServers = append(applicableEntryServers, s)
				}
			}
			// Random/Fastest
			switch params.Metadata.ServerSelectionEntry {
			case types.Random: // RANDOM SERVER (OpenVPN)
				rndIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(applicableEntryServers))))
				if err != nil {
					return params, err
				}
				params.OpenVpnParameters.EntryVpnServer.Hosts = applicableEntryServers[rndIdx.Int64()].Hosts
			case types.Fastest: // FASTEST SERVER (OpenVPN)
				fastestSvr, err := getFastestServer(s, vpn.OpenVPN, applicableEntryServers, params.Metadata.FastestGatewaysExcludeList)
				if err != nil {
					return params, err
				}
				params.OpenVpnParameters.EntryVpnServer.Hosts = fastestSvr.Hosts
			default:
			}
		} else {
			// WireGuard
			applicableEntryServers := []apiTypes.WireGuardServerInfo{}
			if exitSvrCountryCode == "" {
				applicableEntryServers = allServers.WireguardServers
			} else {
				for _, s := range allServers.WireguardServers {
					if s.CountryCode == exitSvrCountryCode {
						continue // exclude exit server from the same country as Exit server
					}
					applicableEntryServers = append(applicableEntryServers, s)
				}
			}
			// Random/Fastest
			switch params.Metadata.ServerSelectionEntry {
			case types.Random: // RANDOM SERVER (WireGuard)
				rndIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(applicableEntryServers))))
				if err != nil {
					return params, err
				}
				params.WireGuardParameters.EntryVpnServer.Hosts = applicableEntryServers[rndIdx.Int64()].Hosts
			case types.Fastest: // FASTEST SERVER (WireGuard)
				fastestSvr, err := getFastestServer(s, vpn.WireGuard, applicableEntryServers, params.Metadata.FastestGatewaysExcludeList)
				if err != nil {
					return params, err
				}
				params.WireGuardParameters.EntryVpnServer.Hosts = fastestSvr.Hosts
			default:
			}
		}
	}

	// EXIT server (Fastest server is not applicable for exit server)
	if params.IsMultiHop() && params.Metadata.ServerSelectionExit == types.Random {

		// Get countryCode of exit server (do not choose exit server from same country)
		entrySvrCountryCode := s.getServerCountryCode(params, true)

		if params.VpnType == vpn.OpenVPN {
			//OpenVPN
			applicableExitServers := []apiTypes.OpenvpnServerInfo{}
			if entrySvrCountryCode == "" {
				applicableExitServers = allServers.OpenvpnServers
			} else {
				for _, s := range allServers.OpenvpnServers {
					if s.CountryCode == entrySvrCountryCode {
						continue // exclude exit server from the same country as Exit server
					}
					applicableExitServers = append(applicableExitServers, s)
				}
			}
			// Random/Fastest
			switch params.Metadata.ServerSelectionEntry {
			case types.Random: // RANDOM SERVER (OpenVPN)
				rndIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(applicableExitServers))))
				if err != nil {
					return params, err
				}
				params.OpenVpnParameters.MultihopExitServer.Hosts = applicableExitServers[rndIdx.Int64()].Hosts

			case types.Fastest: // FASTEST SERVER (OpenVPN)
				// fastest server not applicable for ExitServer
			default:
			}
		} else {
			// WireGuard

			applicableExitServers := []apiTypes.WireGuardServerInfo{}
			if entrySvrCountryCode == "" {
				applicableExitServers = allServers.WireguardServers
			} else {
				for _, s := range allServers.WireguardServers {
					if s.CountryCode == entrySvrCountryCode {
						continue // exclude exit server from the same country as Exit server
					}
					applicableExitServers = append(applicableExitServers, s)
				}
			}
			// Random/Fastest
			switch params.Metadata.ServerSelectionEntry {
			case types.Random: // RANDOM SERVER (WireGuard)
				rndIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(applicableExitServers))))
				if err != nil {
					return params, err
				}
				params.WireGuardParameters.MultihopExitServer.Hosts = applicableExitServers[rndIdx.Int64()].Hosts

			case types.Fastest: // FASTEST SERVER (WireGuard)
				// fastest server not applicable for ExitServer
			default:
			}
		}
	}

	return params, nil
}

func (s *Service) getServerCountryCode(params types.ConnectionParams, isEntryServer bool) string {
	allServers, err := s.ServersList()
	if err != nil {
		return ""
	}

	if params.VpnType == vpn.OpenVPN {
		if isEntryServer {
			return getServerCountryCode(s, params.OpenVpnParameters.EntryVpnServer.Hosts, allServers.OpenvpnServers)
		}
		if params.IsMultiHop() {
			return getServerCountryCode(s, params.OpenVpnParameters.MultihopExitServer.Hosts, allServers.OpenvpnServers)
		}
	} else {
		if isEntryServer {
			return getServerCountryCode(s, params.WireGuardParameters.EntryVpnServer.Hosts, allServers.WireguardServers)
		}
		if params.IsMultiHop() {
			return getServerCountryCode(s, params.WireGuardParameters.MultihopExitServer.Hosts, allServers.WireguardServers)
		}
	}
	return ""
}

type hostBaseInterface interface {
	apiTypes.OpenVPNServerHostInfo | apiTypes.WireGuardServerHostInfo
	GetHostInfoBase() apiTypes.HostInfoBase
}

type serverBaseInterface interface {
	apiTypes.WireGuardServerInfo | apiTypes.OpenvpnServerInfo
	GetServerInfoBase() apiTypes.ServerInfoBase
	GetHostsInfoBase() []apiTypes.HostInfoBase
}

// Return country code of server
func getServerCountryCode[S serverBaseInterface, H hostBaseInterface](service *Service, serverHosts []H, allServers []S) string {
	for _, pHost := range serverHosts {
		for _, s := range allServers {
			for _, h := range s.GetHostsInfoBase() {
				if pHost.GetHostInfoBase().Host == h.Host {
					return s.GetServerInfoBase().CountryCode
				}
			}
		}
	}
	return ""
}

func getFastestServer[S serverBaseInterface](service *Service, vpnTypePrioritized vpn.Type, servers []S, excludedGateways []string) (ret S, err error) {
	hosts, err := service.PingServers(4000, vpnTypePrioritized, true)
	if err != nil {
		return ret, err
	}
	// looking for IP with minimum ping time
	minPingTime := -1
	minPingTimeIp := ""
	for ip, msTime := range hosts {
		if minPingTime == -1 || minPingTime > msTime {
			minPingTime = msTime
			minPingTimeIp = ip
		}
	}

	// Remove everything after symbol '.': "us-tx.wg.ivpn.net" => "us-tx"; or "us-tx" => "us-tx"
	normalizeGwId := func(gwId string) string {
		return strings.Split(gwId, ".")[0]
	}
	// ignored gateways in hashed map
	excludedGatewaysHashed := make(map[string]struct{})
	if len(excludedGateways) > 0 {
		for _, gw := range excludedGateways {
			excludedGatewaysHashed[normalizeGwId(gw)] = struct{}{}
		}
	}

	// looking for server info which contains host with lower ping time
	for _, s := range servers {
		if len(excludedGatewaysHashed) > 0 {
			gw := normalizeGwId(s.GetServerInfoBase().Gateway)
			if _, ok := excludedGatewaysHashed[gw]; ok {
				continue
			}
		}

		for _, h := range s.GetHostsInfoBase() {
			if h.Host == minPingTimeIp {
				return s, nil
			}
		}
	}
	return ret, fmt.Errorf("unable to determine servers latency")
}
