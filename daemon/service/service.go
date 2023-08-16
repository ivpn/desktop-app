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
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/api"
	api_types "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/kem"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/oshelpers"
	protocolTypes "github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/firewall"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/service/platform/filerights"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	"github.com/ivpn/desktop-app/daemon/service/srverrors"
	"github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/shell"
	"github.com/ivpn/desktop-app/daemon/splittun"
	"github.com/ivpn/desktop-app/daemon/vpn"
	"github.com/ivpn/desktop-app/daemon/vpn/wireguard"

	syncSemaphore "golang.org/x/sync/semaphore"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("servc")
}

// RequiredState VPN state which service is going to reach
type RequiredState int

// Requested VPN states
const (
	Disconnect     RequiredState = 0
	Connect        RequiredState = 1
	KeepConnection RequiredState = 2
)

const (
	// SessionCheckInterval - the interval for periodical check session status
	SessionCheckInterval time.Duration = time.Hour * 1
)

// Service - IVPN service
type Service struct {
	_evtReceiver       IServiceEventsReceiver
	_api               *api.API
	_serversUpdater    IServersUpdater
	_netChangeDetector INetChangeDetector
	_wgKeysMgr         IWgKeysManager
	_vpn               vpn.Process
	_preferences       preferences.Preferences
	_connectMutex      sync.Mutex

	// Additional information about current VPN connection
	// Use GetVpnSessionInfo()/SetVpnSessionInfo() to access this data
	_vpnSessionInfo      VpnSessionInfo
	_vpnSessionInfoMutex sync.Mutex

	// Required VPN state which service is going to reach (disconnect->keep connection->connect)
	// When KeepConnection - reconnects immediately after disconnection
	_requiredVpnState RequiredState

	// Note: Disconnect() function will wait until VPN fully disconnects
	_done chan struct{}

	// nil - when session checker stopped
	// to stop -> write to channel (it is synchronous channel)
	_sessionCheckerStopChn chan struct{}

	// when true - necessary to update account status as soon as it will be possible (e.g. on firewall disconnected)
	_isNeedToUpdateSessionInfo bool

	_globalEvents <-chan ServiceEventType

	_systemLog chan<- SystemLogMessage

	_ping struct {
		_results_mutex               sync.RWMutex
		_result                      map[string]int //[host]latency
		_singleRequestLimitSemaphore *syncSemaphore.Weighted
	}

	// variables needed for automatic resume
	_pause struct {
		_mutex           sync.Mutex
		_pauseTill       time.Time // time when connection will be resumed automatically (if not paused - will be zero)
		_killSwitchState bool      // killswitch state before pause (to be able to restore it)
	}

	// variables related to connection test (e.g. ports accessibility test)
	_connectionTest connTest
}

// VpnSessionInfo - Additional information about current VPN connection
type VpnSessionInfo struct {
	// The outbound IP addresses on the moment BEFORE the VPN connection
	OutboundIPv4 net.IP
	OutboundIPv6 net.IP
	// local VPN addresses (outbound IPs)
	VpnLocalIPv4 net.IP
	VpnLocalIPv6 net.IP
}

// CreateService - service constructor
func CreateService(evtReceiver IServiceEventsReceiver, api *api.API, updater IServersUpdater, netChDetector INetChangeDetector, wgKeysMgr IWgKeysManager, globalEvents <-chan ServiceEventType, systemLog chan<- SystemLogMessage) (*Service, error) {
	if updater == nil {
		return &Service{}, fmt.Errorf("ServersUpdater is not defined")
	}

	serv := &Service{
		_preferences:       *preferences.Create(),
		_evtReceiver:       evtReceiver,
		_api:               api,
		_serversUpdater:    updater,
		_netChangeDetector: netChDetector,
		_wgKeysMgr:         wgKeysMgr,
		_globalEvents:      globalEvents,
		_systemLog:         systemLog,
	}

	serv._ping._singleRequestLimitSemaphore = syncSemaphore.NewWeighted(1)

	// register the current service as a 'Connectivity checker' for API object
	serv._api.SetConnectivityChecker(serv)

	if err := serv.init(); err != nil {
		return nil, fmt.Errorf("service initialization error : %w", err)
	}

	return serv, nil
}

func (s *Service) init() error {
	// Start waiting for IP stack initialization
	//
	// _ipStackInitializationWaiter - channel closes as soon as IP stack initialized OR after timeout
	_ipStackInitializationWaiter := make(chan struct{})
	go func() {
		defer close(_ipStackInitializationWaiter) // ip stack initialized (or timeout)
		log.Info("Waiting for IP stack initialization ...")
		endTime := time.Now().Add(time.Minute * 2)
		for {
			ipv4, err4 := netinfo.GetOutboundIP(false)
			ipv6, err6 := netinfo.GetOutboundIP(true)
			if (!ipv4.IsUnspecified() && err4 == nil) || (!ipv6.IsUnspecified() && err6 == nil) {
				log.Info("IP stack initializaed")
				return
			}
			if time.Now().After(endTime) {
				log.Info("WARNING! Timeout waiting for IP stack initialization!")
				return
			}
			time.Sleep(time.Millisecond * 200)
		}
	}()

	// Start periodically updating (downloading) servers in background
	go func() {
		<-_ipStackInitializationWaiter // Wait for IP stack initialization
		if err := s._serversUpdater.StartUpdater(); err != nil {
			log.Error("Failed to start servers-list updater: ", err)
		}
	}()

	if err := s._preferences.LoadPreferences(); err != nil {
		log.Error("Failed to load service preferences: ", err)

		log.Warning("Saving default values for preferences")
		s._preferences.SavePreferences()
	}

	// initialize firewall functionality
	if err := firewall.Initialize(); err != nil {
		return fmt.Errorf("service initialization error : %w", err)
	}

	// initialize dns functionality
	funcGetDnsExtraSettings := func() dns.DnsExtraSettings {
		return dns.DnsExtraSettings{Linux_IsDnsMgmtOldStyle: s._preferences.UserPrefs.Linux.IsDnsMgmtOldStyle}
	}
	if err := dns.Initialize(firewall.OnChangeDNS, funcGetDnsExtraSettings); err != nil {
		log.Error(fmt.Sprintf("failed to initialize DNS : %s", err))
	}

	// initialize split-tunnel functionality
	go func() {
		<-_ipStackInitializationWaiter // Wait for IP stack initialization
		if err := splittun.Initialize(); err != nil {
			log.Warning(fmt.Errorf("Split-Tunnelling initialization error : %w", err))
		} else {
			// apply Split Tunneling configuration
			s.splitTunnelling_ApplyConfig()
		}
	}()

	// Logging mus be already initialized (by launcher). Do nothing here.
	// Init logger (if not initialized before)
	//logger.Enable(s._preferences.IsLogging)

	// firewall initial values
	if err := firewall.AllowLAN(s._preferences.IsFwAllowLAN, s._preferences.IsFwAllowLANMulticast); err != nil {
		log.Error("Failed to initialize firewall with AllowLAN preference value: ", err)
	}

	//log.Info("Applying firewal exceptions (user configuration)")
	if err := firewall.SetUserExceptions(s._preferences.FwUserExceptions, true); err != nil {
		log.Error("Failed to apply firewall exceptions: ", err)
	}

	if s._preferences.IsFwPersistant {
		log.Info("Enabling firewal (persistant configuration)")
		if err := firewall.SetPersistant(true); err != nil {
			log.Error("Failed to enable firewall: ", err)
		}
	}

	// start WireGuard keys rotation
	if err := s._wgKeysMgr.Init(s); err != nil {
		log.Error("Failed to initialize WG keys rotation:", err)
	} else {

		go func() {
			<-_ipStackInitializationWaiter // Wait for IP stack initialization

			if err := s._wgKeysMgr.StartKeysRotation(); err != nil {
				log.Error("Failed to start WG keys rotation:", err)
			}
		}()
	}

	if err := s.initWiFiFunctionality(); err != nil {
		log.Error("Failed to init WiFi functionality:", err)
	}

	// Start session status checker
	go func() {
		<-_ipStackInitializationWaiter // Wait for IP stack initialization
		s.startSessionChecker()
	}()

	s.updateAPIAddrInFWExceptions()
	// servers updated notifier
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("PANIC in Servers update notifier!: ", r)
				if err, ok := r.(error); ok {
					log.ErrorTrace(err)
				}
			}
		}()

		log.Info("Servers update notifier started")
		for {
			// wait for 'servers updated' event
			<-s._serversUpdater.UpdateNotifierChannel()
			// notify clients
			svrs, _ := s.ServersList()
			s._evtReceiver.OnServersUpdated(svrs)
			// update firewall rules: notify firewall about new IP addresses of IVPN API
			s.updateAPIAddrInFWExceptions()
		}
	}()

	// 'Auto-connect on launch' functionality: auto-connect if necessary
	// 'trusted-wifi' functionality: auto-connect if necessary
	go func() {
		<-_ipStackInitializationWaiter // Wait for IP stack initialization
		s.autoConnectIfRequired(OnDaemonStarted, nil)
	}()

	// Start processing power events in separate routine (Windows)
	s.startProcessingPowerEvents()

	return nil
}

// IsConnectivityBlocked - returns nil if connectivity NOT blocked
func (s *Service) IsConnectivityBlocked() error {
	preferences := s._preferences
	if !preferences.IsFwAllowApiServers &&
		preferences.Session.IsLoggedIn() &&
		(!s.Connected() || s.IsPaused()) {
		enabled, err := s.FirewallEnabled()
		if err != nil {
			return fmt.Errorf("access to IVPN servers is blocked: %w", err)
		}
		if err == nil && enabled {
			return fmt.Errorf("access to IVPN servers is blocked (check IVPN Firewall settings)")
		}
	}
	return nil
}

func (s *Service) GetVpnSessionInfo() VpnSessionInfo {
	s._vpnSessionInfoMutex.Lock()
	defer s._vpnSessionInfoMutex.Unlock()
	return s._vpnSessionInfo
}

func (s *Service) SetVpnSessionInfo(i VpnSessionInfo) {
	s._vpnSessionInfoMutex.Lock()
	defer s._vpnSessionInfoMutex.Unlock()
	s._vpnSessionInfo = i
}

func (s *Service) updateAPIAddrInFWExceptions() {
	svrs, _ := s.ServersList()

	ivpnAPIAddr := svrs.Config.API.IPAddresses

	if len(ivpnAPIAddr) <= 0 {
		return
	}

	apiAddrs := make([]net.IP, 0, len(ivpnAPIAddr))
	for _, ipStr := range ivpnAPIAddr {
		apiIP := net.ParseIP(ipStr)
		if apiIP != nil {
			apiAddrs = append(apiAddrs, apiIP)
		}
	}

	if len(apiAddrs) > 0 {
		const onlyForICMP = false
		const isPersistent = true
		prefs := s.Preferences()
		if prefs.IsFwAllowApiServers {
			firewall.AddHostsToExceptions(apiAddrs, onlyForICMP, isPersistent)
		} else {
			firewall.RemoveHostsFromExceptions(apiAddrs, onlyForICMP, isPersistent)
		}
	}
}

// ServersList returns servers info
// (if there is a cached data available - will be returned data from cache)
func (s *Service) ServersList() (*api_types.ServersInfoResponse, error) {
	return s._serversUpdater.GetServers()
}

func (s *Service) findOpenVpnHost(hostname string, ip net.IP, svrs []api_types.OpenvpnServerInfo) (api_types.OpenVPNServerHostInfo, error) {
	if ((len(hostname) > 0) || (ip != nil && !ip.IsUnspecified())) && svrs != nil {
		for _, svr := range svrs {
			for _, host := range svr.Hosts {
				if (len(hostname) <= 0 || !strings.EqualFold(host.Hostname, hostname)) && (ip == nil || ip.IsUnspecified() || !ip.Equal(net.ParseIP(host.Host))) {
					continue
				}
				return host, nil
			}
		}
	}

	return api_types.OpenVPNServerHostInfo{}, fmt.Errorf(fmt.Sprintf("host '%s' not found", hostname))
}

// ServersListForceUpdate returns servers list info.
// The daemon will make request to update servers from the backend.
// The cached data will be ignored in this case.
func (s *Service) ServersListForceUpdate() (*api_types.ServersInfoResponse, error) {
	return s._serversUpdater.GetServersForceUpdate()
}

// APIRequest do custom request to API
func (s *Service) APIRequest(apiAlias string, ipTypeRequired protocolTypes.RequiredIPProtocol) (responseData []byte, err error) {

	if ipTypeRequired == protocolTypes.IPv6 {
		// IPV6-LOC-200 - IVPN Apps should request only IPv4 location information when connected  to the gateway, which doesn’t support IPv6
		vpn := s._vpn
		if vpn != nil && !vpn.IsPaused() && !vpn.IsIPv6InTunnel() {
			return nil, fmt.Errorf("no IPv6 support inside tunnel for current connection")
		}
	}

	return s._api.DoRequestByAlias(apiAlias, ipTypeRequired)
}

// GetDisabledFunctions returns info about functions which are disabled
// Some functionality can be not accessible
// It can happen, for example, if some external binaries not installed
// (e.g. obfsproxy or WireGuard on Linux)
func (s *Service) GetDisabledFunctions() protocolTypes.DisabledFunctionality {
	var ovpnErr, obfspErr, v2rayErr, wgErr, splitTunErr error

	if err := filerights.CheckFileAccessRightsExecutable(platform.OpenVpnBinaryPath()); err != nil {
		ovpnErr = fmt.Errorf("OpenVPN binary: %w", err)
	}

	if err := filerights.CheckFileAccessRightsExecutable(platform.ObfsproxyStartScript()); err != nil {
		obfspErr = fmt.Errorf("obfsproxy binary: %w", err)
	}

	if err := filerights.CheckFileAccessRightsExecutable(platform.V2RayBinaryPath()); err != nil {
		v2rayErr = fmt.Errorf("V2Ray binary: %w", err)
	} else if platform.V2RayConfigFile() == "" {
		v2rayErr = fmt.Errorf("V2Ray config file path not defined")
	}

	if err := filerights.CheckFileAccessRightsExecutable(platform.WgBinaryPath()); err != nil {
		wgErr = fmt.Errorf("WireGuard binary: %w", err)
	} else {
		if err := filerights.CheckFileAccessRightsExecutable(platform.WgToolBinaryPath()); err != nil {
			wgErr = fmt.Errorf("WireGuard tools binary: %w", err)
		}
	}

	// returns non-nil error object if Split-Tunneling functionality not available
	splitTunErr = splittun.GetFuncNotAvailableError()

	if errors.Is(ovpnErr, os.ErrNotExist) {
		ovpnErr = fmt.Errorf("%w. Please install OpenVPN", ovpnErr)
	}
	if errors.Is(obfspErr, os.ErrNotExist) {
		obfspErr = fmt.Errorf("%w. Please install obfsproxy binary", obfspErr)
	}
	if errors.Is(wgErr, os.ErrNotExist) {
		wgErr = fmt.Errorf("%w. Please install WireGuard", wgErr)
	}

	var ret protocolTypes.DisabledFunctionality

	if wgErr != nil {
		ret.WireGuardError = wgErr.Error()
	}
	if ovpnErr != nil {
		ret.OpenVPNError = ovpnErr.Error()
	}
	if obfspErr != nil {
		ret.ObfsproxyError = obfspErr.Error()
	}
	if v2rayErr != nil {
		ret.V2RayError = v2rayErr.Error()
	}
	if splitTunErr != nil {
		ret.SplitTunnelError = splitTunErr.Error()
	}

	ret.Platform = s.implGetDisabledFuncForPlatform()

	return ret
}

func (s *Service) IsCanConnectMultiHop() error {
	return s._preferences.Account.IsCanConnectMultiHop()
}

func (s *Service) reconnect() {
	// Just call disconnect
	// The reconnection will be performed automatically in method 'keepConnection(...)'
	// (according to s._requiredVpnState value == KeepConnection)
	s.disconnect()
}

// Disconnect disconnect vpn
func (s *Service) Disconnect() error {
	s._requiredVpnState = Disconnect
	// Resume connection (but do not notify "Connection resumed" status)
	if err := s.resume(); err != nil {
		log.Error("Resume failed:", err)
	}
	return s.disconnect()
}

func (s *Service) disconnect() error {
	vpn := s._vpn
	if vpn == nil {
		return nil
	}

	done := s._done
	if s._requiredVpnState == KeepConnection {
		log.Info("Disconnecting (going to reconnect)...")
	} else {
		log.Info("Disconnecting...")
	}

	// stop detections for routing changes
	s._netChangeDetector.Stop()

	// stop VPN
	if err := vpn.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect VPN: %w", err)
	}

	// wait for stop
	if done != nil {
		<-done
	}

	return nil
}

// Connected returns 'true' if VPN connected
func (s *Service) Connected() bool {
	return s._vpn != nil
}

// ConnectedType returns connected VPN type (only if VPN connected!)
func (s *Service) ConnectedType() (isConnected bool, connectedVpnType vpn.Type) {
	vpnObj := s._vpn
	if vpnObj == nil {
		return false, 0
	}
	return true, vpnObj.Type()
}

// FirewallEnabled returns firewall state (enabled\disabled)
// (in use, for example, by WireGuard keys manager, to know is it have sense to make API requests.)
func (s *Service) FirewallEnabled() (bool, error) {
	return firewall.GetEnabled()
}

// Pause pause vpn connection
func (s *Service) Pause(durationSeconds uint32) error {
	vpn := s._vpn
	if vpn == nil {
		return fmt.Errorf("VPN not connected")
	}

	if durationSeconds <= 0 {
		return fmt.Errorf("the duration of the pause has not been specified")
	}

	defer s._evtReceiver.OnVpnPauseChanged()

	s._pause._mutex.Lock()
	defer s._pause._mutex.Unlock()

	fwIsEnabled, isPersistant, _, _, _, _, err := s.KillSwitchState()
	if err != nil {
		return fmt.Errorf("failed to check KillSwitch status: %w", err)
	}
	s._pause._killSwitchState = fwIsEnabled
	if fwIsEnabled && !isPersistant {
		if err := s.SetKillSwitchState(false); err != nil {
			return err
		}
	}

	log.Info("Pausing...")
	firewall.ClientPaused()

	if err = vpn.Pause(); err != nil {
		return err
	}

	s._pause._pauseTill = time.Now().Add(time.Second * time.Duration(durationSeconds))
	log.Info(fmt.Sprintf("Paused on %v (till %v)", time.Second*time.Duration(durationSeconds), s._pause._pauseTill.Format(time.Stamp)))

	go func() {
		// Pause resumer: Every second checks if it is time to resume VPN connection.
		// Info: We can not use 'time.AfterFunc()' because
		// it does not take into account the time when the system was in sleep mode.
		defer log.Info("Resumed")
		for {
			time.Sleep(time.Second * 1)

			if !s.IsPaused() {
				s._pause._pauseTill = time.Time{} // reset pause time (to indicate that connection is not paused, just in case)
				break
			} else {
				// Note! In order to avoid any potential issues with location or changes with system clock, we must use "monotonic clock" time (Unix()).
				if time.Now().Unix()-s.PausedTill().Unix() >= 0 {
					log.Info(fmt.Sprintf("Automatic resuming after %v ...", time.Second*time.Duration(durationSeconds)))
					if err := s.Resume(); err != nil {
						log.Error(fmt.Errorf("Resume failed: %w", err))
					}
					break
				}
			}
		}
	}()

	return nil
}

// Resume resume vpn connection
func (s *Service) Resume() error {
	defer s._evtReceiver.OnVpnPauseChanged()

	vpn := s._vpn
	if vpn == nil || !vpn.IsPaused() {
		return fmt.Errorf("VPN not paused")
	}

	return s.resume()
}

// Resume resume vpn connection
func (s *Service) resume() error {
	s._pause._mutex.Lock()
	defer s._pause._mutex.Unlock()
	s._pause._pauseTill = time.Time{} // reset pause time (to indicate that connection is not paused)

	vpn := s._vpn
	if vpn == nil {
		return nil
	}
	if !vpn.IsPaused() {
		return nil
	}

	log.Info("Resuming...")
	firewall.ClientResumed()
	if err := vpn.Resume(); err != nil {
		return err
	}

	fwIsEnabled, isPersistant, _, _, _, _, err := s.KillSwitchState()
	if err != nil {
		log.Error(fmt.Errorf("failed to check KillSwitch status: %w", err))
	} else {
		if !isPersistant && fwIsEnabled != s._pause._killSwitchState {
			if err := s.SetKillSwitchState(s._pause._killSwitchState); err != nil {
				log.Error("failed to restore KillSwitch status: %w", err)
			}
		}
	}

	return nil
}

// IsPaused returns 'true' if current vpn connection is in paused state
func (s *Service) IsPaused() bool {
	vpn := s._vpn
	if vpn == nil {
		return false
	}

	return vpn.IsPaused() && !s.PausedTill().IsZero()
}

func (s *Service) PausedTill() time.Time {
	return s._pause._pauseTill
}

func (s *Service) saveDefaultDnsParams(dnsCfg dns.DnsSettings, antiTrackerCfg types.AntiTrackerMetadata) (retErr error) {
	defaultParams := s.GetConnectionParams()

	if defaultParams.ManualDNS.Equal(dnsCfg) && defaultParams.Metadata.AntiTracker.Equal(antiTrackerCfg) {
		return nil
	}

	// save DNS and AntiTracker default metadata
	defaultParams.ManualDNS = dnsCfg
	defaultParams.Metadata.AntiTracker = antiTrackerCfg

	return s.setConnectionParams(defaultParams)
}

// GetDefaultDnsParams returns default DNS parameters
// Returns:
//
//	dnsCfg - default DNS parameters
//	antiTrackerCfg - default AntiTracker parameters
//	realDnsValue - real DNS value (if 'antiTracker' is enabled - it will contain DNS of AntiTracker server)
func (s *Service) GetDefaultDnsParams() (dnsCfg dns.DnsSettings, antiTrackerCfg types.AntiTrackerMetadata, realDnsValue dns.DnsSettings, err error) {
	defaultParams := s.GetConnectionParams()

	dnsCfg = defaultParams.ManualDNS
	realDnsValue = defaultParams.ManualDNS
	antiTrackerCfg = defaultParams.Metadata.AntiTracker

	if antiTrackerCfg.Enabled {
		realDnsValue, err = s.getAntiTrackerDns(antiTrackerCfg.Hardcore, antiTrackerCfg.AntiTrackerBlockListName)
	}

	return dnsCfg, antiTrackerCfg, realDnsValue, err
}

// SetManualDNS update default DNS parameters AND apply new DNS value for current VPN connection
// If 'antiTracker' is enabled - the 'dnsCfg' will be ignored
func (s *Service) SetManualDNS(dnsCfg dns.DnsSettings, antiTracker types.AntiTrackerMetadata) (changedDns dns.DnsSettings, retErr error) {
	// Update default metadata
	defaultParams := s.GetConnectionParams()
	isChanged := false
	// save DNS and AntiTracker default metadata
	if !defaultParams.ManualDNS.Equal(dnsCfg) {
		defaultParams.ManualDNS = dnsCfg
		isChanged = true
	}
	if !defaultParams.Metadata.AntiTracker.Equal(antiTracker) {
		at, err := s.normalizeAntiTrackerBlockListName(antiTracker)
		if err != nil {
			return changedDns, err
		}
		defaultParams.Metadata.AntiTracker = at
		isChanged = true
	}
	if isChanged {
		s.setConnectionParams(defaultParams)
	}

	// Get anti-tracker DNS settings
	changedDns = dnsCfg
	if antiTracker.Enabled {
		atDns, err := s.getAntiTrackerDns(antiTracker.Hardcore, antiTracker.AntiTrackerBlockListName)
		if err != err {
			return dns.DnsSettings{}, err
		}
		changedDns = atDns
	}

	vpn := s._vpn
	if vpn == nil {
		// no active VPN connection
		return changedDns, nil
	}

	if dnsCfg.IsEmpty() && !antiTracker.Enabled {
		return dns.DnsSettings{}, vpn.ResetManualDNS()
	}
	return changedDns, vpn.SetManualDNS(changedDns)
}

func (s *Service) GetAntiTrackerStatus() (types.AntiTrackerMetadata, error) {
	// Get AntiTracker DNS settings. If error - use default date and ignore error
	retAtMetadata, err := s.normalizeAntiTrackerBlockListName(s.GetConnectionParams().Metadata.AntiTracker)
	if err != nil {
		log.Error(fmt.Sprintf("failed to normalize AntiTracker block list name: %v (using '%s')", err, retAtMetadata.AntiTrackerBlockListName))
	}
	return retAtMetadata, nil
}

// Normze AntiTracker block list name:
// - if antiTrackerPlusList not defined - return default value
// - if antiTrackerPlusList defined - check if it is valid; if not valid - return default value and error
func (s *Service) normalizeAntiTrackerBlockListName(antiTracker types.AntiTrackerMetadata) (types.AntiTrackerMetadata, error) {
	var retError error

	atBlistName := strings.ToLower(strings.TrimSpace(antiTracker.AntiTrackerBlockListName))
	// check if block list name is known
	if atBlistName != "" {
		servers, err := s.ServersList()
		if err == nil {
			for _, atp_svr := range servers.Config.AntiTrackerPlus.DnsServers {
				if strings.ToLower(strings.TrimSpace(atp_svr.Name)) == atBlistName {
					// Block-list name is OK. Just ensure to use correct case
					antiTracker.AntiTrackerBlockListName = strings.TrimSpace(atp_svr.Name)
					return antiTracker, nil
				}
			}
		}

		retError = fmt.Errorf("unexpected DNS block list name: '%s'", antiTracker.AntiTrackerBlockListName)
	}

	// Set default block list name (if empty)
	if tmpDns, err := s.getAntiTrackerDns(antiTracker.Hardcore, ""); err == nil {
		if tmpAt, err := s.getAntiTrackerInfo(tmpDns); err == nil {
			antiTracker.AntiTrackerBlockListName = tmpAt.AntiTrackerBlockListName
		}
	}

	return antiTracker, retError
}

// Get DNS server according to AntiTracker parameters
func (s *Service) getAntiTrackerDns(isHardcore bool, antiTrackerPlusList string) (dnsCfg dns.DnsSettings, err error) {
	defer func() {
		if dnsCfg.IsEmpty() && err == nil {
			err = fmt.Errorf("unable to determine AntiTracker DNS")
		}
	}()
	servers, err := s.ServersList()
	if err != nil {
		return dns.DnsSettings{}, fmt.Errorf("failed to determine AntiTracker parameters: %w", err)
	}

	// AntiTracker Plus list
	atListName := strings.ToLower(strings.TrimSpace(antiTrackerPlusList))
	if len(atListName) == 0 {
		// if block list name not defined - use default AntiTracker block list "Basic"
		atListName = "basic"
	}

	if len(atListName) > 0 {
		for _, atp_svr := range servers.Config.AntiTrackerPlus.DnsServers {
			if strings.ToLower(strings.TrimSpace(atp_svr.Name)) == atListName {
				if isHardcore {
					return dns.DnsSettings{DnsHost: atp_svr.Hardcore}, nil
				}
				return dns.DnsSettings{DnsHost: atp_svr.Normal}, nil
			}
		}
	}

	// If AntiTracker Plus block list not found - ignore 'antiTrackerPlusList' and use old-style AntiTracker DNS
	if isHardcore {
		return dns.DnsSettings{DnsHost: servers.Config.Antitracker.Hardcore.IP}, nil
	}
	return dns.DnsSettings{DnsHost: servers.Config.Antitracker.Default.IP}, nil
}

// Get AntiTracker info according to DNS settings
func (s *Service) getAntiTrackerInfo(dnsVal dns.DnsSettings) (types.AntiTrackerMetadata, error) {
	if dnsVal.IsEmpty() || dnsVal.Encryption != dns.EncryptionNone {
		return types.AntiTrackerMetadata{}, nil
	}

	servers, err := s.ServersList()
	if err != nil {
		return types.AntiTrackerMetadata{}, fmt.Errorf("failed to determine AntiTracker parameters: %w", err)
	}

	dnsHost := strings.ToLower(strings.TrimSpace(dnsVal.DnsHost))
	if dnsHost == "" {
		return types.AntiTrackerMetadata{}, nil
	}

	// Check AntiTracker Plus lists
	for _, atp_svr := range servers.Config.AntiTrackerPlus.DnsServers {
		if strings.EqualFold(dnsHost, strings.TrimSpace(atp_svr.Normal)) {
			return types.AntiTrackerMetadata{Enabled: true, Hardcore: false, AntiTrackerBlockListName: atp_svr.Name}, nil
		}
		if strings.EqualFold(dnsHost, strings.TrimSpace(atp_svr.Hardcore)) {
			return types.AntiTrackerMetadata{Enabled: true, Hardcore: true, AntiTrackerBlockListName: atp_svr.Name}, nil
		}
	}

	// Check AntiTracker values
	if strings.EqualFold(dnsHost, strings.TrimSpace(servers.Config.Antitracker.Default.IP)) {
		return types.AntiTrackerMetadata{Enabled: true, Hardcore: false}, nil
	}
	if strings.EqualFold(dnsHost, strings.TrimSpace(servers.Config.Antitracker.Hardcore.IP)) {
		return types.AntiTrackerMetadata{Enabled: true, Hardcore: true}, nil
	}

	return types.AntiTrackerMetadata{}, nil
}

// ////////////////////////////////////////////////////////
// KillSwitch
// ////////////////////////////////////////////////////////
func (s *Service) onKillSwitchStateChanged() {
	s._evtReceiver.OnKillSwitchStateChanged()

	// check if we need try to update account info
	if s._isNeedToUpdateSessionInfo {
		go s.RequestSessionStatus()
	}
}

// SetKillSwitchState enable\disable kill-switch
func (s *Service) SetKillSwitchState(isEnabled bool) error {

	if !isEnabled && s._preferences.IsFwPersistant {
		return fmt.Errorf("unable to disable Firewall in 'Persistent' state. Please, disable 'Always-on firewall' first")
	}
	if s.IsPaused() {
		return fmt.Errorf("unable to change the firewall state while connection is paused, please resume the connection first")
	}

	err := firewall.SetEnabled(isEnabled)
	if err == nil {
		s.onKillSwitchStateChanged()
		// If no any clients connected - connection notification will not be passed to user
		// In this case we are trying to save info message into system log
		if !s._evtReceiver.IsClientConnected(false) {
			if isEnabled {
				s.systemLog(Info, "IVPN Firewall enabled")
			} else {
				s.systemLog(Info, "IVPN Firewall disabled")
			}
		}
	}
	return err
}

// KillSwitchState returns kill-switch state
func (s *Service) KillSwitchState() (isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast, isAllowApiServers bool, fwUserExceptions string, err error) {
	prefs := s._preferences
	enabled, err := firewall.GetEnabled()
	return enabled, prefs.IsFwPersistant, prefs.IsFwAllowLAN, prefs.IsFwAllowLANMulticast, prefs.IsFwAllowApiServers, prefs.FwUserExceptions, err
}

// SetKillSwitchIsPersistent change kill-switch value
func (s *Service) SetKillSwitchIsPersistent(isPersistant bool) error {
	if s.IsPaused() {
		return fmt.Errorf("unable to change the firewall state while connection is paused, please resume the connection first")
	}

	prefs := s._preferences
	prefs.IsFwPersistant = isPersistant
	s.setPreferences(prefs)

	err := firewall.SetPersistant(isPersistant)
	if err == nil {
		s.onKillSwitchStateChanged()
	}
	return err
}

// SetKillSwitchAllowLAN change kill-switch value
func (s *Service) SetKillSwitchAllowLAN(isAllowLan bool) error {
	return s.setKillSwitchAllowLAN(isAllowLan, s._preferences.IsFwAllowLANMulticast)
}

// SetKillSwitchAllowLANMulticast change kill-switch value
func (s *Service) SetKillSwitchAllowLANMulticast(isAllowLanMulticast bool) error {
	return s.setKillSwitchAllowLAN(s._preferences.IsFwAllowLAN, isAllowLanMulticast)
}

func (s *Service) setKillSwitchAllowLAN(isAllowLan bool, isAllowLanMulticast bool) error {
	prefs := s._preferences
	prefs.IsFwAllowLAN = isAllowLan
	prefs.IsFwAllowLANMulticast = isAllowLanMulticast
	s.setPreferences(prefs)

	err := firewall.AllowLAN(prefs.IsFwAllowLAN, prefs.IsFwAllowLANMulticast)
	if err == nil {
		s.onKillSwitchStateChanged()
	}
	return err
}

func (s *Service) SetKillSwitchAllowAPIServers(isAllowAPIServers bool) error {
	if !isAllowAPIServers {
		// Do not allow to disable access to IVPN API server if user logged-out
		// Otherwise, we will not have possibility to login
		session := s.Preferences().Session
		if !session.IsLoggedIn() {
			return srverrors.ErrorNotLoggedIn{}
		}
	}

	prefs := s._preferences
	prefs.IsFwAllowApiServers = isAllowAPIServers
	s.setPreferences(prefs)
	s.onKillSwitchStateChanged()
	s.updateAPIAddrInFWExceptions()
	return nil
}

// SetKillSwitchUserExceptions set ip/mask to be excluded from FW block
// Parameters:
//   - exceptions - comma separated list of IP addresses in format: x.x.x.x[/xx]
func (s *Service) SetKillSwitchUserExceptions(exceptions string, ignoreParsingErrors bool) error {
	prefs := s._preferences
	prefs.FwUserExceptions = exceptions
	s.setPreferences(prefs)

	err := firewall.SetUserExceptions(exceptions, ignoreParsingErrors)
	if err == nil {
		s.onKillSwitchStateChanged()
	}
	return err
}

//////////////////////////////////////////////////////////
// PREFERENCES
//////////////////////////////////////////////////////////

// SetPreference set preference value
func (s *Service) SetPreference(key protocolTypes.ServicePreference, val string) (isChanged bool, err error) {
	prefs := s._preferences
	isChanged = false

	switch key {
	case protocolTypes.Prefs_IsEnableLogging:
		if val, err := strconv.ParseBool(val); err == nil {
			isChanged = val != prefs.IsLogging
			prefs.IsLogging = val
			logger.Enable(val)
		}

	case protocolTypes.Prefs_IsAutoconnectOnLaunch:
		if val, err := strconv.ParseBool(val); err == nil {
			isChanged = val != prefs.IsAutoconnectOnLaunch
			prefs.IsAutoconnectOnLaunch = val
		}

	case protocolTypes.Prefs_IsAutoconnectOnLaunch_Daemon:
		if val, err := strconv.ParseBool(val); err == nil {
			if val {
				if e := prefs.LastConnectionParams.CheckIsDefined(); e != nil {
					return false, srverrors.ErrorBackgroundConnectionNoParams{}
				}
			}
			isChanged = val != prefs.IsAutoconnectOnLaunchDaemon
			prefs.IsAutoconnectOnLaunchDaemon = val
		}

	default:
		log.Warning(fmt.Sprintf("Preference key '%s' not supported", key))
	}

	s.setPreferences(prefs)

	if isChanged {
		log.Info(fmt.Sprintf("(prefs '%s' changed) %s", key, val))
	}

	return isChanged, nil
}

// SetPreference set preference value
func (s *Service) SetUserPreferences(userPrefs preferences.UserPreferences) error {
	// platform-specific check if we can apply this preferences
	if err := s.implIsCanApplyUserPreferences(userPrefs); err != nil {
		return err
	}

	prefs := s._preferences
	prefs.UserPrefs = userPrefs
	s.setPreferences(prefs)

	return nil
}

// Preferences returns preferences
func (s *Service) Preferences() preferences.Preferences {
	return s._preferences
}

func (s *Service) ResetPreferences() error {
	s._preferences = *preferences.Create()

	// erase ST config
	s.SplitTunnelling_SetConfig(false, true)
	return nil
}

func (s *Service) GetConnectionParams() types.ConnectionParams {
	return s._preferences.LastConnectionParams
}

func (s *Service) SetConnectionParams(params types.ConnectionParams) error {
	prefs := s._preferences

	isOldParamsDefined := prefs.LastConnectionParams.CheckIsDefined() == nil

	retErr := s.setConnectionParams(params)

	if !isOldParamsDefined && prefs.LastConnectionParams.CheckIsDefined() != nil {
		// if it is first initialization of connection parameters - run auto-connection rules
		// (seems, it is first start after app version upgrade)

		prefs := s.Preferences()
		const checkOnlyUiClients = true
		if prefs.Session.IsLoggedIn() && s._evtReceiver.IsClientConnected(checkOnlyUiClients) {
			log.Info("Applying auto-connection rules (reason: first initialization of connection parameters) ...")
			s.autoConnectIfRequired(OnUiClientConnected, nil)
		}
	}

	return retErr
}

func (s *Service) setConnectionParams(params types.ConnectionParams) error {
	prefs := s._preferences

	prefs.LastConnectionParams = params
	s.setPreferences(prefs)

	return nil
}

func (s *Service) SetWiFiSettings(params preferences.WiFiParams) error {
	if params.CanApplyInBackground {
		prefs := s._preferences
		if e := prefs.LastConnectionParams.CheckIsDefined(); e != nil {
			return srverrors.ErrorBackgroundConnectionNoParams{}
		}
	}

	// remove duplicate networks from 'trusted' list
	newNets := []preferences.WiFiNetwork{}
	keys := make(map[string]struct{})
	for _, n := range params.Networks {
		if _, exists := keys[n.SSID]; !exists && len(n.SSID) > 0 {
			newNets = append(newNets, n)
			keys[n.SSID] = struct{}{}
		}
	}
	params.Networks = newNets

	// save settings
	prefs := s._preferences
	prefs.WiFiControl = params
	s.setPreferences(prefs)

	// 'trusted-wifi' functionality: auto-connect if necessary
	s.autoConnectIfRequired(OnWifiChanged, nil)
	return nil
}

//////////////////////////////////////////////////////////
// SPLIT TUNNEL
//////////////////////////////////////////////////////////

func (s *Service) GetInstalledApps(extraArgsJSON string) ([]oshelpers.AppInfo, error) {
	return oshelpers.GetInstalledApps(extraArgsJSON)
}

func (s *Service) GetBinaryIcon(binaryPath string) (string, error) {
	return oshelpers.GetBinaryIconBase64(binaryPath)
}

func (s *Service) SplitTunnelling_GetStatus() (protocolTypes.SplitTunnelStatus, error) {
	var prefs = s.Preferences()
	runningProcesses, err := splittun.GetRunningApps()
	if err != nil {
		runningProcesses = []splittun.RunningApp{}
	}

	ret := protocolTypes.SplitTunnelStatus{
		IsFunctionalityNotAvailable: splittun.GetFuncNotAvailableError() != nil,
		IsEnabled:                   prefs.IsSplitTunnel,
		IsCanGetAppIconForBinary:    oshelpers.IsCanGetAppIconForBinary(),
		SplitTunnelApps:             prefs.SplitTunnelApps,
		RunningApps:                 runningProcesses}

	return ret, nil
}

func (s *Service) SplitTunnelling_SetConfig(isEnabled bool, reset bool) error {
	if reset || splittun.GetFuncNotAvailableError() != nil {
		return s.splitTunnelling_Reset()
	}

	prefs := s._preferences
	prefs.IsSplitTunnel = isEnabled
	s.setPreferences(prefs)

	return s.splitTunnelling_ApplyConfig()
}
func (s *Service) splitTunnelling_Reset() error {
	prefs := s._preferences
	prefs.IsSplitTunnel = false
	prefs.SplitTunnelApps = make([]string, 0)
	s.setPreferences(prefs)

	splittun.Reset()

	return s.splitTunnelling_ApplyConfig()
}
func (s *Service) splitTunnelling_ApplyConfig() error {
	// notify changed ST configuration status (even if functionality not available)
	defer s._evtReceiver.OnSplitTunnelStatusChanged()

	if splittun.GetFuncNotAvailableError() != nil {
		// Split-Tunneling not accessible (not able to connect to a driver or not implemented for current platform)
		return nil
	}

	prefs := s.Preferences()
	sInf := s.GetVpnSessionInfo()

	addressesCfg := splittun.ConfigAddresses{
		IPv4Tunnel: sInf.VpnLocalIPv4,
		IPv4Public: sInf.OutboundIPv4,
		IPv6Tunnel: sInf.VpnLocalIPv6,
		IPv6Public: sInf.OutboundIPv6}

	return splittun.ApplyConfig(prefs.IsSplitTunnel, s.Connected(), addressesCfg, prefs.SplitTunnelApps)
}

func (s *Service) SplitTunnelling_AddApp(exec string) (cmdToExecute string, isAlreadyRunning bool, err error) {
	if !s._preferences.IsSplitTunnel {
		return "", false, fmt.Errorf("unable to run application in Split Tunneling environment: Split Tunneling is disabled")
	}
	// apply ST configuration after function ends
	defer s.splitTunnelling_ApplyConfig()
	return s.implSplitTunnelling_AddApp(exec)
}

func (s *Service) SplitTunnelling_RemoveApp(pid int, exec string) (err error) {
	// apply ST configuration after function ends
	defer s.splitTunnelling_ApplyConfig()
	return s.implSplitTunnelling_RemoveApp(pid, exec)
}

// Inform the daemon about started process in ST environment
// Parameters:
// pid 			- process PID
// exec 		- Command executed in ST environment (e.g. binary + arguments)
//
//	(identical to SplitTunnelAddApp.Exec and SplitTunnelAddAppCmdResp.Exec)
//
// cmdToExecute - Shell command used to perform this operation
func (s *Service) SplitTunnelling_AddedPidInfo(pid int, exec string, cmdToExecute string) error {
	// notify changed ST configuration status
	defer s._evtReceiver.OnSplitTunnelStatusChanged()
	return s.implSplitTunnelling_AddedPidInfo(pid, exec, cmdToExecute)
}

//////////////////////////////////////////////////////////
// SESSIONS
//////////////////////////////////////////////////////////

func (s *Service) setCredentials(accountInfo preferences.AccountStatus, accountID, session, vpnUser, vpnPass, wgPublicKey, wgPrivateKey, wgLocalIP string, wgKeyGenerated int64, wgPreSharedKey string) error {
	// save session info
	s._preferences.SetSession(accountInfo,
		accountID,
		session,
		vpnUser,
		vpnPass,
		wgPublicKey,
		wgPrivateKey,
		wgLocalIP,
		wgPreSharedKey)

	// manually set info about WG keys timestamp
	if wgKeyGenerated > 0 {
		s._preferences.Session.WGKeyGenerated = time.Unix(wgKeyGenerated, 0)
		s._preferences.SavePreferences()
	}

	// notify clients about session update
	s._evtReceiver.OnServiceSessionChanged()

	// start session checker
	s.startSessionChecker()

	// start WireGuard keys rotation
	if err := s._wgKeysMgr.StartKeysRotation(); err != nil {
		log.Error(fmt.Sprintf("Unable to start WireGuard keys rotation: %v", err.Error()))
	}

	return nil
}

// SessionNew creates new session
func (s *Service) SessionNew(accountID string, forceLogin bool, captchaID string, captcha string, confirmation2FA string) (
	apiCode int,
	apiErrorMsg string,
	accountInfo preferences.AccountStatus,
	rawResponse string,
	err error) {

	// Temporary allow API server access (If Firewall is enabled)
	// Otherwise, there will not be any possibility to Login (because all connectivity is blocked)
	fwIsEnabled, _, _, _, fwIsAllowApiServers, _, _ := s.KillSwitchState()
	if fwIsEnabled && !fwIsAllowApiServers {
		s.SetKillSwitchAllowAPIServers(true)
	}
	defer func() {
		if fwIsEnabled && !fwIsAllowApiServers {
			// restore state for 'AllowAPIServers' configuration (previously, was enabled)
			s.SetKillSwitchAllowAPIServers(false)
		}
	}()

	// delete current session (if exists)
	isCanDeleteSessionLocally := true
	if err := s.SessionDelete(isCanDeleteSessionLocally); err != nil {
		log.Error("Creating new session -> Failed to delete active session: ", err)
	}

	// Generate keys for Key Encapsulation Mechanism using post-quantum cryptographic algorithms
	var kemKeys api_types.KemPublicKeys
	kemHelper, err := kem.CreateHelper(platform.KemHelperBinaryPath(), kem.GetDefaultKemAlgorithms())
	if err != nil {
		log.Error("Failed to generate KEM keys: ", err)
	} else {
		kemKeys.KemPublicKey_Kyber1024, err = kemHelper.GetPublicKey(kem.AlgName_Kyber1024)
		if err != nil {
			log.Error(err)
		}
		kemKeys.KemPublicKey_ClassicMcEliece348864, err = kemHelper.GetPublicKey(kem.AlgName_ClassicMcEliece348864)
		if err != nil {
			log.Error(err)
		}
	}

	log.Info("Logging in...")
	defer func() {
		if err != nil {
			log.Info("Logging in - FAILED: ", err)
		} else {
			log.Info("Logging in - SUCCESS")

		}
	}()

	var (
		publicKey  string
		privateKey string

		wgPresharedKey string
		successResp    *api_types.SessionNewResponse
		errorLimitResp *api_types.SessionNewErrorLimitResponse
		apiErr         *api_types.APIErrorResponse
		rawRespStr     string // RAW response
	)

	for {
		// generate new keys for WireGuard
		publicKey, privateKey, err = wireguard.GenerateKeys(platform.WgToolBinaryPath())
		if err != nil {
			log.Warning("Failed to generate wireguard keys for new session: %s", err)
		}

		successResp, errorLimitResp, apiErr, rawRespStr, err = s._api.SessionNew(accountID, publicKey, kemKeys, forceLogin, captchaID, captcha, confirmation2FA)
		rawResponse = rawRespStr

		apiCode = 0
		if apiErr != nil {
			apiCode = apiErr.Status
		}

		if err != nil {
			// if SessionsLimit response
			if errorLimitResp != nil {
				accountInfo = s.createAccountStatus(errorLimitResp.SessionLimitData)
				return apiCode, apiErr.Message, accountInfo, rawResponse, err
			}

			// in case of other API error
			if apiErr != nil {
				return apiCode, apiErr.Message, accountInfo, rawResponse, err
			}

			// not API error
			return apiCode, "", accountInfo, rawResponse, err
		}

		if successResp == nil {
			return apiCode, "", accountInfo, rawResponse, fmt.Errorf("unexpected error when creating a new session")
		}

		if kemHelper != nil {
			if len(successResp.WireGuard.KemCipher_Kyber1024) == 0 && len(successResp.WireGuard.KemCipher_ClassicMcEliece348864) == 0 {
				log.Warning("The server did not respond with KEM ciphers. The WireGuard PresharedKey has not been initialized!")
			} else {
				if err := kemHelper.SetCipher(kem.AlgName_Kyber1024, successResp.WireGuard.KemCipher_Kyber1024); err != nil {
					log.Error(err)
				}
				if err := kemHelper.SetCipher(kem.AlgName_ClassicMcEliece348864, successResp.WireGuard.KemCipher_ClassicMcEliece348864); err != nil {
					log.Error(err)
				}

				wgPresharedKey, err = kemHelper.CalculatePresharedKey()
				if err != nil {
					log.Error(fmt.Sprintf("Failed to decode KEM ciphers! (%s). Retry Log-in without WireGuard PresharedKey...", err))
					kemHelper = nil
					kemKeys = api_types.KemPublicKeys{}
					if err := s.SessionDelete(true); err != nil {
						log.Error("Creating new session (retry 2) -> Failed to delete active session: ", err)
					}
					continue
				}
			}
		}
		break
	}
	// get account status info
	accountInfo = s.createAccountStatus(successResp.ServiceStatus)

	s.setCredentials(accountInfo,
		accountID,
		successResp.Token,
		successResp.VpnUsername,
		successResp.VpnPassword,
		publicKey,
		privateKey,
		successResp.WireGuard.IPAddress, 0, wgPresharedKey)

	log.Info(fmt.Sprintf("(logging in) WG keys updated (%s:%s; psk:%v)", successResp.WireGuard.IPAddress, publicKey, len(wgPresharedKey) > 0))

	return apiCode, "", accountInfo, rawResponse, nil
}

// SessionDelete removes session info
func (s *Service) SessionDelete(isCanDeleteSessionLocally bool) error {
	sessionNeedToDeleteOnBackend := true
	return s.logOut(sessionNeedToDeleteOnBackend, isCanDeleteSessionLocally)
}

// logOut performs log out from current session
// 1) if 'sessionNeedToDeleteOnBackend' == false: the app not trying to make API request
//	  the session info just erasing locally
//    (this is useful for the situations when we already know that session is not available on backend anymore)
// 2) if 'sessionNeedToDeleteOnBackend' == true (and 'isCanDeleteSessionLocally' == false): app is trying to make API request to logout correctly
//	  in case if API request failed the function returns error (session keeps not logged out)
// 3) if 'isCanDeleteSessionLocally' == true (and 'sessionNeedToDeleteOnBackend' == true): app is trying to make API request to logout correctly
//	  in case if API request failed we just erasing session info locally (no errors returned)

func (s *Service) logOut(sessionNeedToDeleteOnBackend bool, isCanDeleteSessionLocally bool) error {
	// Disconnect (if connected)
	s.Disconnect()

	// stop session checker (use goroutine to avoid deadlocks)
	go s.stopSessionChecker()

	// stop WG keys rotation
	s._wgKeysMgr.StopKeysRotation()

	if sessionNeedToDeleteOnBackend {

		// Temporary allow API server access (If Firewall is enabled)
		// Otherwise, there will not be any possibility to Login (because all connectivity is blocked)
		fwIsEnabled, _, _, _, fwIsAllowApiServers, _, _ := s.KillSwitchState()
		if fwIsEnabled && !fwIsAllowApiServers {
			s.SetKillSwitchAllowAPIServers(true)
		}
		defer func() {
			if fwIsEnabled && !fwIsAllowApiServers {
				// restore state for 'AllowAPIServers' configuration (previously, was enabled)
				s.SetKillSwitchAllowAPIServers(false)
			}
		}()

		session := s.Preferences().Session
		if session.IsLoggedIn() {
			log.Info("Logging out")
			err := s._api.SessionDelete(session.Session)
			if err != nil {
				log.Info("Logging out error:", err)
				if !isCanDeleteSessionLocally {
					return err // do not allow to logout if failed to delete session on backend
				}
			} else {
				log.Info("Logging out: done")
			}
		}
	}

	s._preferences.SetSession(preferences.AccountStatus{}, "", "", "", "", "", "", "", "")
	log.Info("Logged out locally")

	// notify clients about session update
	s._evtReceiver.OnServiceSessionChanged()
	return nil
}

func (s *Service) OnSessionNotFound() {
	// Logging out now
	log.Info("Session not found. Logging out.")
	needToDeleteOnBackend := false
	canLogoutOnlyLocally := true
	s.logOut(needToDeleteOnBackend, canLogoutOnlyLocally)
}

func (s *Service) OnAccountStatus(sessionToken string, accountInfo preferences.AccountStatus) {
	// save last known info about account status
	s._preferences.UpdateAccountInfo(accountInfo)
	// notify about account status
	s._evtReceiver.OnAccountStatus(sessionToken, accountInfo)
}

// RequestSessionStatus receives session status
func (s *Service) RequestSessionStatus() (
	apiCode int,
	apiErrorMsg string,
	sessionToken string,
	accountInfo preferences.AccountStatus,
	err error) {

	session := s.Preferences().Session
	if !session.IsLoggedIn() {
		return apiCode, "", "", accountInfo, srverrors.ErrorNotLoggedIn{}
	}

	// if no connectivity - skip request (and activate _isWaitingToUpdateAccInfoChan)
	if err := s.IsConnectivityBlocked(); err != nil {
		s._isNeedToUpdateSessionInfo = true
		return apiCode, "", "", accountInfo, fmt.Errorf("session status request skipped (%w)", err)
	}
	// defer: ensure s._isWaitingToUpdateAccInfoChan is empty
	defer func() {
		s._isNeedToUpdateSessionInfo = false
	}()

	log.Info("Requesting session status...")
	stat, apiErr, err := s._api.SessionStatus(session.Session)
	log.Info("Session status request: done")

	currSession := s.Preferences().Session
	if currSession.Session != session.Session {
		// It could happen that logout\login was performed during the session check
		// Ignoring result if there is already a new session
		log.Info("Ignoring requested session status result. Local session already changed.")
		return apiCode, "", "", accountInfo, srverrors.ErrorNotLoggedIn{}
	}

	apiCode = 0
	if apiErr != nil {
		apiCode = apiErr.Status

		// Session not found - can happens when user forced to logout from another device
		if apiCode == api_types.SessionNotFound {
			s.OnSessionNotFound()
		}

		// save last account info AND notify clients that account not active
		if apiCode == api_types.AccountNotActive {
			accountInfo = preferences.AccountStatus{}
			if stat != nil {
				accountInfo = s.createAccountStatus(*stat)
			}
			accountInfo.Active = false
			// notify about account status
			s.OnAccountStatus(session.Session, accountInfo)
			return apiCode, apiErr.Message, session.Session, accountInfo, err
		}
	}

	if err != nil {
		// in case of other API error
		if apiErr != nil {
			return apiCode, apiErr.Message, "", accountInfo, err
		}

		// not API error
		return apiCode, "", "", accountInfo, err
	}

	if stat == nil {
		return apiCode, "", "", accountInfo, fmt.Errorf("unexpected error when creating requesting session status")
	}

	// get account status info
	accountInfo = s.createAccountStatus(*stat)
	// ave last account info AND notify about account status
	s.OnAccountStatus(session.Session, accountInfo)

	// success
	return apiCode, "", session.Session, accountInfo, nil
}

func (s *Service) createAccountStatus(apiResp api_types.ServiceStatusAPIResp) preferences.AccountStatus {
	return preferences.AccountStatus{
		Active:         apiResp.Active,
		ActiveUntil:    apiResp.ActiveUntil,
		CurrentPlan:    apiResp.CurrentPlan,
		PaymentMethod:  apiResp.PaymentMethod,
		IsRenewable:    apiResp.IsRenewable,
		WillAutoRebill: apiResp.WillAutoRebill,
		IsFreeTrial:    apiResp.IsFreeTrial,
		Capabilities:   apiResp.Capabilities,
		Upgradable:     apiResp.Upgradable,
		UpgradeToPlan:  apiResp.UpgradeToPlan,
		UpgradeToURL:   apiResp.UpgradeToURL,
		Limit:          apiResp.Limit}
}

func (s *Service) startSessionChecker() {
	// ensure that session checker is not running
	s.stopSessionChecker()

	session := s.Preferences().Session
	if !session.IsLoggedIn() {
		return
	}

	s._sessionCheckerStopChn = make(chan struct{})
	go func() {
		log.Info("Session checker started")
		defer log.Info("Session checker stopped")

		stopChn := s._sessionCheckerStopChn
		for {
			// check status
			s.RequestSessionStatus()

			// if not logged-in - no sense to check status anymore
			session := s.Preferences().Session
			if !session.IsLoggedIn() {
				return
			}

			// wait for timeout or stop request
			select {
			case <-stopChn:
				return
			case <-time.After(SessionCheckInterval):
			}
		}
	}()
}

func (s *Service) stopSessionChecker() {
	stopChan := s._sessionCheckerStopChn
	s._sessionCheckerStopChn = nil
	if stopChan != nil {
		stopChan <- struct{}{}
	}
}

//////////////////////////////////////////////////////////
// WireGuard keys
//////////////////////////////////////////////////////////

// WireGuardSaveNewKeys saves WG keys
func (s *Service) WireGuardSaveNewKeys(wgPublicKey string, wgPrivateKey string, wgLocalIP string, wgPresharedKey string) {
	s._preferences.UpdateWgCredentials(wgPublicKey, wgPrivateKey, wgLocalIP, wgPresharedKey)

	// notify clients about session (wg keys) update
	s._evtReceiver.OnServiceSessionChanged()

	go func() {
		// reconnect in separate routine (do not block current thread)
		vpnObj := s._vpn
		if vpnObj == nil {
			return
		}
		if vpnObj.Type() != vpn.WireGuard {
			return
		}
		if !s.Connected() || (s.Connected() && s.IsPaused()) {
			// IMPORTANT! : WireGuard 'pause/resume' state is based on complete VPN disconnection and connection back (on all platforms)
			// If this will be changed (e.g. just changing routing) - it will be necessary to implement reconnection even in 'pause' state
			return
		}
		log.Info("Reconnecting WireGuard connection with new credentials...")
		s.reconnect()
	}()
}

// WireGuardSetKeysRotationInterval change WG key rotation interval
func (s *Service) WireGuardSetKeysRotationInterval(interval int64) {
	s._preferences.Session.WGKeysRegenInerval = time.Second * time.Duration(interval)
	s._preferences.SavePreferences()

	// restart WG keys rotation
	if err := s._wgKeysMgr.StartKeysRotation(); err != nil {
		log.Error(err)
	}

	// notify clients about session (wg keys) update
	s._evtReceiver.OnServiceSessionChanged()
}

// WireGuardGetKeys get WG keys
func (s *Service) WireGuardGetKeys() (session, wgPublicKey, wgPrivateKey, wgLocalIP string, generatedTime time.Time, updateInterval time.Duration) {
	p := s._preferences

	return p.Session.Session,
		p.Session.WGPublicKey,
		p.Session.WGPrivateKey,
		p.Session.WGLocalIP,
		p.Session.WGKeyGenerated,
		p.Session.WGKeysRegenInerval
}

// WireGuardGenerateKeys - generate new wireguard keys
func (s *Service) WireGuardGenerateKeys(updateIfNecessary bool) error {
	if !s._preferences.Session.IsLoggedIn() {
		return srverrors.ErrorNotLoggedIn{}
	}

	// Update WG keys, if necessary
	var err error
	if updateIfNecessary {
		err = s._wgKeysMgr.UpdateKeysIfNecessary()
	} else {
		err = s._wgKeysMgr.GenerateKeys()
	}
	if err != nil {
		return fmt.Errorf("failed to regenerate WireGuard keys: %w", err)
	}

	return nil
}

// ////////////////////////////////////////////////////////
// Diagnostic
// ////////////////////////////////////////////////////////
func (s *Service) GetDiagnosticLogs() (logActive string, logPrevSession string, extraInfo string, err error) {
	log, log0, err := logger.GetLogText(1024 * 64)
	if err != nil {
		return "", "", "", err
	}

	extraInfo, err1 := s.implGetDiagnosticExtraInfo()
	if err1 != nil {
		extraInfo = fmt.Sprintf("<failed to obtain extra info> : %s : %s", err1.Error(), extraInfo)
	}

	return log, log0, extraInfo, nil
}

func (s *Service) diagnosticGetCommandOutput(command string, args ...string) string {
	outText, outErrText, _, isBufferTooSmall, err := shell.ExecAndGetOutput(nil, 1024*30, "", command, args...)
	ret := fmt.Sprintf("[ $ %s %v ]:\n%s", command, args, outText)
	if isBufferTooSmall {
		ret += "... (buffer too small)"
	}
	if len(outErrText) > 0 {
		ret += "\n [ERROR CHANNEL OUTPUT]: " + outErrText
	}
	if err != nil {
		ret += "\n [ERROR]: " + err.Error()
	}
	return ret
}

//////////////////////////////////////////////////////////
// Internal methods
//////////////////////////////////////////////////////////

func (s *Service) setPreferences(p preferences.Preferences) {
	if !reflect.DeepEqual(s._preferences, p) {
		//if s._preferences != p {
		s._preferences = p
		s._preferences.SavePreferences()
	}
}
