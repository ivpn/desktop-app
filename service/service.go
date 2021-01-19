//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
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

package service

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app-daemon/api"
	"github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/helpers"
	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/netinfo"
	"github.com/ivpn/desktop-app-daemon/service/dns"
	"github.com/ivpn/desktop-app-daemon/service/firewall"
	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/service/platform/filerights"
	"github.com/ivpn/desktop-app-daemon/service/preferences"
	"github.com/ivpn/desktop-app-daemon/vpn"
	"github.com/ivpn/desktop-app-daemon/vpn/openvpn"
	"github.com/ivpn/desktop-app-daemon/vpn/wireguard"
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
	SessionCheckInterval time.Duration = time.Hour * 6
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

	// manual DNS value (if not defined - nil)
	_manualDNS net.IP

	// Required VPN state which service is going to reach (disconnect->keep connection->connect)
	// When KeepConnection - reconnects immediately after disconnection
	_requiredVpnState RequiredState

	// Note: Disconnect() function will wait until VPN fully disconnects
	_done chan struct{}

	_isServersPingInProgress bool

	// nil - when session checker stopped
	// to stop -> write to channel (it is synchronous channel)
	_sessionCheckerStopChn chan struct{}
}

// CreateService - service constructor
func CreateService(evtReceiver IServiceEventsReceiver, api *api.API, updater IServersUpdater, netChDetector INetChangeDetector, wgKeysMgr IWgKeysManager) (*Service, error) {
	if updater == nil {
		return &Service{}, fmt.Errorf("ServersUpdater is not defined")
	}

	serv := &Service{
		_evtReceiver:       evtReceiver,
		_api:               api,
		_serversUpdater:    updater,
		_netChangeDetector: netChDetector,
		_wgKeysMgr:         wgKeysMgr}

	if err := serv.init(); err != nil {
		return nil, fmt.Errorf("service initialization error : %w", err)
	}

	return serv, nil
}

func (s *Service) init() error {
	if err := s._preferences.LoadPreferences(); err != nil {
		log.Error("Failed to load service preferences: ", err)

		log.Warning("Saving default values for preferences")
		s._preferences.SavePreferences()
	}

	if err := dns.Initialize(); err != nil {
		log.Error(fmt.Sprintf("failed to initialize DNS : %s", err))
	}

	if err := firewall.Initialize(); err != nil {
		return fmt.Errorf("service initialization error : %w", err)
	}

	// Logging mus be already initialized (by launcher). Do nothing here.
	// Init logger (if not initialized before)
	//logger.Enable(s._preferences.IsLogging)

	// Init firewall
	if err := firewall.AllowLAN(s._preferences.IsFwAllowLAN, s._preferences.IsFwAllowLANMulticast); err != nil {
		log.Error("Failed to initialize firewall with AllowLAN preference value: ", err)
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
		if err := s._wgKeysMgr.StartKeysRotation(); err != nil {
			log.Error("Failed to start WG keys rotation:", err)
		}
	}

	s.initWiFiFunctionality()

	// Check session status (start as go-routine to do not block service initialization)
	go s.RequestSessionStatus()
	// Start session status checker
	s.startSessionChecker()

	return nil
}

// OnControlConnectionClosed - Perform reqired operations when protocol (controll channel with UI application) was closed
// (for example, we must disable firewall (if it not persistant))
// Must be called by protocol object
// Return parameters:
// - isServiceMustBeClosed: true informing that service have to be closed ("Stop IVPN Agent when application is not running" feature)
// - err: error
func (s *Service) OnControlConnectionClosed() (isServiceMustBeClosed bool, err error) {
	isServiceMustBeClosed = s._preferences.IsStopOnClientDisconnect
	// disable firewall if it not persistant
	if !s._preferences.IsFwPersistant {
		log.Info("Control connection was closed. Disabling firewall.")
		err = firewall.SetEnabled(false)
	}
	return isServiceMustBeClosed, err
}

// ServersList - get VPN servers info
func (s *Service) ServersList() (*types.ServersInfoResponse, error) {
	return s._serversUpdater.GetServers()
}

// ServersUpdateNotifierChannel returns channel which is notifying when servers was updated
func (s *Service) ServersUpdateNotifierChannel() chan struct{} {
	return s._serversUpdater.UpdateNotifierChannel()
}

// APIRequest do custom request to API
func (s *Service) APIRequest(apiPath string, method string, contentType string, requestObject interface{}) (responseData []byte, err error) {
	return s._api.DoRequestRaw(apiPath, method, contentType, requestObject)
}

// GetDisabledFunctions returns info about functions which are disabled
// Some functionality can be not accessible
// It can happen, for example, if some external binaries not installed
// (e.g. obfsproxy or WireGuard on Linux)
func (s *Service) GetDisabledFunctions() (wgErr, ovpnErr, obfspErr error) {
	if err := filerights.CheckFileAccessRightsExecutable(platform.OpenVpnBinaryPath()); err != nil {
		ovpnErr = fmt.Errorf("OpenVPN binary: %w", err)
	}

	if err := filerights.CheckFileAccessRightsExecutable(platform.ObfsproxyStartScript()); err != nil {
		obfspErr = fmt.Errorf("obfsproxy binary: %w", err)
	}

	if err := filerights.CheckFileAccessRightsExecutable(platform.WgBinaryPath()); err != nil {
		wgErr = fmt.Errorf("WireGuard binary: %w", err)
	} else {
		if err := filerights.CheckFileAccessRightsExecutable(platform.WgToolBinaryPath()); err != nil {
			wgErr = fmt.Errorf("WireGuard tools binary: %w", err)
		}
	}

	if errors.Is(ovpnErr, os.ErrNotExist) {
		ovpnErr = fmt.Errorf("%w. Please install OpenVPN", ovpnErr)
	}
	if errors.Is(obfspErr, os.ErrNotExist) {
		obfspErr = fmt.Errorf("%w. Please install obfsproxy binary", obfspErr)
	}
	if errors.Is(wgErr, os.ErrNotExist) {
		wgErr = fmt.Errorf("%w. Please install WireGuard", wgErr)
	}

	return wgErr, ovpnErr, obfspErr
}

// ConnectOpenVPN start OpenVPN connection
func (s *Service) ConnectOpenVPN(connectionParams openvpn.ConnectionParams, manualDNS net.IP, firewallDuringConnection bool, stateChan chan<- vpn.StateInfo) error {

	createVpnObjfunc := func() (vpn.Process, error) {
		prefs := s.Preferences()

		// checking if functionality accessible
		_, ovpnErr, obfspErr := s.GetDisabledFunctions()
		if ovpnErr != nil {
			return nil, ovpnErr
		}
		if prefs.IsObfsproxy == true && obfspErr != nil {
			return nil, obfspErr
		}

		connectionParams.SetCredentials(prefs.Session.OpenVPNUser, prefs.Session.OpenVPNPass)

		openVpnExtraParameters := ""
		// read user-defined extra parameters for OpenVPN configuration (if exists)
		extraParamsFile := platform.OpenvpnUserParamsFile()

		if helpers.FileExists(extraParamsFile) {
			if err := filerights.CheckFileAccessRightsConfig(extraParamsFile); err != nil {
				log.Info("NOTE! User-defined OpenVPN parameters are ignored! %w", err)
				os.Remove(extraParamsFile)
			} else {
				// read file line by line
				openVpnExtraParameters = func() string {
					var allParams strings.Builder

					file, err := os.Open(extraParamsFile)
					if err != nil {
						log.Error(err)
						return ""
					}
					defer file.Close()

					scanner := bufio.NewScanner(file)
					for scanner.Scan() {
						line := scanner.Text()
						line = strings.TrimSpace(line)
						if len(line) <= 0 {
							continue
						}
						if strings.HasPrefix(line, "#") {
							continue // comment
						}
						if strings.HasPrefix(line, ";") {
							continue // comment
						}
						allParams.WriteString(line + "\n")
					}

					if err := scanner.Err(); err != nil {
						log.Error("Failed to parse '%s': %s", extraParamsFile, err)
						return ""
					}
					return allParams.String()
				}()

				if len(openVpnExtraParameters) > 0 {
					log.Info(fmt.Sprintf("WARNING! User-defined OpenVPN parameters loaded from file '%s'!", extraParamsFile))
				}
			}
		}

		// creating OpenVPN object
		vpnObj, err := openvpn.NewOpenVpnObject(
			platform.OpenVpnBinaryPath(),
			platform.OpenvpnConfigFile(),
			platform.OpenvpnLogFile(),
			prefs.IsObfsproxy,
			openVpnExtraParameters,
			connectionParams)

		if err != nil {
			return nil, fmt.Errorf("failed to create new openVPN object: %w", err)
		}
		return vpnObj, nil
	}

	return s.keepConnection(createVpnObjfunc, manualDNS, firewallDuringConnection, stateChan)
}

// ConnectWireGuard start WireGuard connection
func (s *Service) ConnectWireGuard(connectionParams wireguard.ConnectionParams, manualDNS net.IP, firewallDuringConnection bool, stateChan chan<- vpn.StateInfo) error {
	// stop active connection (if exists)
	if err := s.Disconnect(); err != nil {
		return fmt.Errorf("failed to connect. Unable to stop active connection: %w", err)
	}

	// checking if functionality accessible
	wgErr, _, _ := s.GetDisabledFunctions()
	if wgErr != nil {
		return wgErr
	}

	// Update WG keys, if necessary
	err := s.WireGuardGenerateKeys(true)
	if err != nil {
		log.Warning("Failed to regenerate WireGuard keys: ", err)
		// TODO: notify UI
	}

	createVpnObjfunc := func() (vpn.Process, error) {
		session := s.Preferences().Session

		if session.IsWGCredentialsOk() == false {
			return nil, fmt.Errorf("WireGuard credentials are not defined (please, regenerate WG credentials or re-login)")
		}

		localip := net.ParseIP(session.WGLocalIP)
		if localip == nil {
			return nil, fmt.Errorf("error updating WG connection preferences (failed parsing local IP for WG connection)")
		}
		connectionParams.SetCredentials(session.WGPrivateKey, localip)

		vpnObj, err := wireguard.NewWireGuardObject(
			platform.WgBinaryPath(),
			platform.WgToolBinaryPath(),
			platform.WGConfigFilePath(),
			connectionParams)

		if err != nil {
			return nil, fmt.Errorf("failed to create new WireGuard object: %w", err)
		}
		return vpnObj, nil
	}

	return s.keepConnection(createVpnObjfunc, manualDNS, firewallDuringConnection, stateChan)
}

func (s *Service) keepConnection(createVpnObj func() (vpn.Process, error), manualDNS net.IP, firewallDuringConnection bool, stateChan chan<- vpn.StateInfo) error {
	prefs := s.Preferences()
	if prefs.Session.IsLoggedIn() == false {
		return ErrorNotLoggedIn{}
	}

	s._manualDNS = manualDNS

	// Not necessary to keep connection until we are not connected
	// So just 'Connect' required for now
	s._requiredVpnState = Connect

	// no delay before first reconnection
	delayBeforeReconnect := 0 * time.Second

	stateChan <- vpn.NewStateInfo(vpn.CONNECTING, "Connecting")
	for {
		// create new VPN object
		vpnObj, err := createVpnObj()
		if err != nil {
			return fmt.Errorf("failed to create VPN object: %w", err)
		}

		lastConnectionTryTime := time.Now()

		// start connection
		err = s.connect(vpnObj, s._manualDNS, firewallDuringConnection, stateChan)
		if err != nil {
			log.Error(fmt.Sprintf("Connection error: %s", err))
			if s._requiredVpnState == Connect {
				// throw error only on first try to connect
				// if we were already connected (_requiredVpnState==KeepConnection) - ignore error and try to reconnect
				return err
			}
		}

		// retry, if reconnection requested
		if s._requiredVpnState == KeepConnection {
			// notifying clients about reconnection
			stateChan <- vpn.NewStateInfo(vpn.RECONNECTING, "Reconnecting due to disconnection")

			// no delay before reconnection (if last connection was long time ago)
			if time.Now().After(lastConnectionTryTime.Add(time.Second * 30)) {
				delayBeforeReconnect = 0
			}

			if delayBeforeReconnect > 0 {
				log.Info(fmt.Sprintf("Reconnecting (pause %s)...", delayBeforeReconnect))
				// do delay before next reconnection
				pauseTill := time.Now().Add(delayBeforeReconnect)
				for time.Now().Before(pauseTill) && s._requiredVpnState != Disconnect {
					time.Sleep(time.Millisecond * 10)
				}
			} else {
				log.Info("Reconnecting...")
			}

			if s._requiredVpnState == KeepConnection {
				// consecutive reconnections has delay 5 seconds
				delayBeforeReconnect = time.Second * 5
				continue
			}
		}

		// stop loop
		break
	}

	return nil
}

// Connect connect vpn
func (s *Service) connect(vpnProc vpn.Process, manualDNS net.IP, firewallDuringConnection bool, stateChan chan<- vpn.StateInfo) error {
	var connectRoutinesWaiter sync.WaitGroup

	// stop active connection (if exists)
	if err := s.disconnect(); err != nil {
		return fmt.Errorf("failed to connect. Unable to stop active connection: %w", err)
	}

	// check session status each disconnection (asynchronously, in separate goroutine)
	defer func() { go s.RequestSessionStatus() }()

	s._connectMutex.Lock()
	defer s._connectMutex.Unlock()

	s._done = make(chan struct{}, 1)
	defer func() {
		// notify: connection stopped
		done := s._done
		s._done = nil
		if done != nil {
			done <- struct{}{}
			// Closing channel
			// Note: reading from empty and closed channel will not lead to deadlock (immediately returns zero value)
			close(done)
		}
	}()

	log.Info("Connecting...")
	// save vpn object
	s._vpn = vpnProc

	internalStateChan := make(chan vpn.StateInfo, 1)
	stopChannel := make(chan bool, 1)

	fwInitState := false
	// finalize everything
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic on VPN connection: ", r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
		}

		// Ensure that routing-change detector is stopped (we do not need it when VPN disconnected)
		s._netChangeDetector.Stop()

		// notify firewall that client is disconnected
		err := firewall.ClientDisconnected()
		if err != nil {
			log.Error("(stopping) error on notifying FW about disconnected client:", err)
		}

		// when we were requested to enable firewall for this connection
		// And initial FW state was disabled - we have to disable it back
		if firewallDuringConnection == true && fwInitState == false {
			if err = s.SetKillSwitchState(false); err != nil {
				log.Error("(stopping) failed to disable firewall:", err)
			}
		}

		// notify routines to stop
		close(stopChannel)

		// resetting manual DNS (if it is necessary)
		err = vpnProc.ResetManualDNS()
		if err != nil {
			log.Error("(stopping) error resetting manual DNS: ", err)
		}

		connectRoutinesWaiter.Wait()

		// Forget VPN object
		s._vpn = nil

		log.Info("VPN process stopped")
	}()

	routingChangeChan := make(chan struct{}, 1)

	// goroutine: process + forward VPN state change
	connectRoutinesWaiter.Add(1)
	go func() {
		log.Info("VPN state forwarder started")
		defer func() {
			log.Info("VPN state forwarder stopped")
			connectRoutinesWaiter.Done()
		}()

		var state vpn.StateInfo
		for isRuning := true; isRuning; {
			select {
			case state = <-internalStateChan:

				// store info about current time
				state.Time = time.Now().Unix()
				// store info about VPN connection type
				state.VpnType = vpnProc.Type()

				// forward state to 'stateChan'
				stateChan <- state

				log.Info(fmt.Sprintf("State: %v", state))

				// internally process VPN state change
				switch state.State {

				case vpn.RECONNECTING:
					// Disable routing-change detector when reconnecting
					s._netChangeDetector.Stop()

				case vpn.CONNECTED:
					// since we are connected - keep connection (reconnect if unexpected disconnection)
					if s._requiredVpnState == Connect {
						s._requiredVpnState = KeepConnection
					}

					// start routing change detection
					if netInterface, err := netinfo.InterfaceByIPAddr(state.ClientIP); err != nil {
						log.Error(fmt.Sprintf("Unable to initialize routing change detection. Failed to get interface '%s'", state.ClientIP.String()))
					} else {

						log.Info("Starting route change detection")
						s._netChangeDetector.Start(routingChangeChan, netInterface)
					}

					// Inform firewall about client local IP
					firewall.ClientConnected(state.ClientIP)
				default:
				}

			case <-stopChannel: // triggered when the stopChannel is closed
				isRuning = false
			}
		}
	}()

	// receiving routing change notifications
	connectRoutinesWaiter.Add(1)
	go func() {
		log.Info("Route change receiver started")
		defer func() {
			log.Info("Route change receiver stopped")
			connectRoutinesWaiter.Done()
		}()

		for isRuning := true; isRuning; {
			select {
			case <-routingChangeChan: // routing changed
				if s._vpn.IsPaused() {
					log.Info("Route change ignored due to Paused state.")
				} else {
					// Disconnect (client will request then reconnection, because of unexpected disconnection)
					// reconnect in separate routine (do not block current thread)
					go func() {
						defer func() {
							if r := recover(); r != nil {
								log.Error("PANIC: ", r)
							}
						}()

						log.Info("Route change detected. Reconnecting...")
						s.reconnect()
					}()

					isRuning = false
				}
			case <-stopChannel: // triggered when the stopChannel is closed
				isRuning = false
			}
		}
	}()

	log.Info("Initializing...")
	if err := vpnProc.Init(); err != nil {
		return fmt.Errorf("failed to initialize VPN object: %w", err)
	}

	log.Info("Initializing firewall")
	if firewallDuringConnection == true {
		// in case to enable FW for this connection parameter:
		// - check initial FW state
		// - if it disabled - enable it (will be disabled on disconnect)
		fw, err := firewall.GetEnabled()
		if err != nil {
			log.Error("Failed to check firewall state:", err.Error())
			return err
		}
		fwInitState = fw
		if fwInitState == false {
			if err := s.SetKillSwitchState(true); err != nil {
				log.Error("Failed to enable firewall:", err.Error())
				return err
			}
		}
	}

	// Add host IP to firewall exceptions
	const onlyForICMP = false
	err := firewall.AddHostsToExceptions(vpnProc.DestinationIPs(), onlyForICMP)
	if err != nil {
		log.Error("Failed to start. Unable to add hosts to firewall exceptions:", err.Error())
		return err
	}

	log.Info("Initializing DNS")
	// set manual DNS
	if manualDNS == nil || manualDNS.Equal(net.IPv4zero) || manualDNS.Equal(net.IPv4bcast) {
		err = s.ResetManualDNS()
	} else {
		err = s.SetManualDNS(manualDNS)
	}
	if err != nil {
		err = fmt.Errorf("failed to set DNS: %w", err)
		log.Error(err.Error())
		return err
	}

	log.Info("Starting VPN process")
	// connect: start VPN process and wait until it finishes
	err = vpnProc.Connect(internalStateChan)
	if err != nil {
		err = fmt.Errorf("connection error: %w", err)
		log.Error(err.Error())
		return err
	}

	return nil
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
	if s._vpn == nil {
		return false
	}
	return true
}

// FirewallEnabled returns firewall state (enabled\disabled)
// (in use, for example, by WireGuard keys manager, to know is it have sense to make API requests.)
func (s *Service) FirewallEnabled() (bool, error) {
	return firewall.GetEnabled()
}

// Pause pause vpn connection
func (s *Service) Pause() error {
	vpn := s._vpn
	if vpn == nil {
		return nil
	}

	log.Info("Pausing...")
	firewall.ClientPaused()
	return vpn.Pause()
}

// Resume resume vpn connection
func (s *Service) Resume() error {
	vpn := s._vpn
	if vpn == nil {
		return nil
	}

	log.Info("Resuming...")
	firewall.ClientResumed()
	return vpn.Resume()
}

// SetManualDNS set dns
func (s *Service) SetManualDNS(dns net.IP) error {
	vpn := s._vpn
	if vpn == nil {
		return nil
	}

	s._manualDNS = dns
	if err := firewall.SetManualDNS(dns); err != nil {
		return fmt.Errorf("failed to set manual DNS: %w", err)
	}

	err := vpn.SetManualDNS(dns)
	if err == nil {
		s._evtReceiver.OnDNSChanged(dns)
	}
	return err
}

// ResetManualDNS set dns to default
func (s *Service) ResetManualDNS() error {
	vpn := s._vpn
	if vpn == nil {
		return nil
	}

	s._manualDNS = nil
	if err := firewall.SetManualDNS(nil); err != nil {
		return fmt.Errorf("failed to reset manual DNS: %w", err)
	}

	err := vpn.ResetManualDNS()
	if err == nil {
		s._evtReceiver.OnDNSChanged(nil)
	}
	return err
}

// if 'currentLocation' defined - the output hosts list will be sorted by distance to current location
func (s *Service) getHostsToPing(currentLocation *types.GeoLookupResponse) ([]net.IP, error) {
	// get servers info
	servers, err := s._serversUpdater.GetServers()
	if err != nil {
		return nil, fmt.Errorf("unable to get servers list: %w", err)
	}

	type hostInfo struct {
		Latitude  float32
		Longitude float32
		host      net.IP
	}

	hosts := make([]hostInfo, 0, len(servers.OpenvpnServers)+len(servers.WireguardServers))

	// OpenVPN servers
	for _, s := range servers.OpenvpnServers {
		if len(s.IPAddresses) <= 0 {
			continue
		}
		ip := net.ParseIP(s.IPAddresses[0])
		if ip != nil {
			hosts = append(hosts, hostInfo{Latitude: s.Latitude, Longitude: s.Longitude, host: ip})
		}
	}

	// ping each WireGuard server
	for _, s := range servers.WireguardServers {
		if len(s.Hosts) <= 0 {
			continue
		}

		ip := net.ParseIP(s.Hosts[0].Host)
		if ip != nil {
			hosts = append(hosts, hostInfo{Latitude: s.Latitude, Longitude: s.Longitude, host: ip})
		}
	}

	if currentLocation != nil {
		cLat := float64(currentLocation.Latitude)
		cLot := float64(currentLocation.Longitude)
		sort.Slice(hosts, func(i, j int) bool {
			di := helpers.GetDistanceFromLatLonInKm(cLat, cLot, float64(hosts[i].Latitude), float64(hosts[i].Longitude))
			dj := helpers.GetDistanceFromLatLonInKm(cLat, cLot, float64(hosts[j].Latitude), float64(hosts[j].Longitude))
			return di < dj
		})
	}
	ret := make([]net.IP, 0, len(hosts))
	for _, h := range hosts {
		ret = append(ret, h.host)
	}
	return ret, nil
}

// SetKillSwitchState enable\disable killswitch
func (s *Service) SetKillSwitchState(isEnabled bool) error {
	err := firewall.SetEnabled(isEnabled)
	if err == nil {
		s._evtReceiver.OnKillSwitchStateChanged()
	}
	return err
}

// KillSwitchState returns killswitch state
func (s *Service) KillSwitchState() (isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast bool, err error) {
	prefs := s._preferences
	enabled, err := firewall.GetEnabled()
	return enabled, prefs.IsFwPersistant, prefs.IsFwAllowLAN, prefs.IsFwAllowLANMulticast, err
}

// SetKillSwitchIsPersistent change kill-switch value
func (s *Service) SetKillSwitchIsPersistent(isPersistant bool) error {
	prefs := s._preferences
	prefs.IsFwPersistant = isPersistant
	s.setPreferences(prefs)

	err := firewall.SetPersistant(isPersistant)
	if err == nil {
		s._evtReceiver.OnKillSwitchStateChanged()
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
		s._evtReceiver.OnKillSwitchStateChanged()
	}
	return err
}

// SetPreference set preference value
func (s *Service) SetPreference(key string, val string) error {
	prefs := s._preferences

	switch key {
	case "enable_logging":
		if val, err := strconv.ParseBool(val); err == nil {
			prefs.IsLogging = val
			logger.Enable(val)
		}
		break
	case "is_stop_server_on_client_disconnect":
		if val, err := strconv.ParseBool(val); err == nil {
			prefs.IsStopOnClientDisconnect = val
		}
		break
	case "enable_obfsproxy":
		if val, err := strconv.ParseBool(val); err == nil {
			prefs.IsObfsproxy = val
		}
		break
	case "firewall_is_persistent":
		log.Debug("Skipping 'firewall_is_persistent' value. IVPNKillSwitchSetIsPersistentRequest should be used")
		break
	default:
		log.Warning(fmt.Sprintf("Preference key '%s' not supported", key))
	}

	s.setPreferences(prefs)
	log.Info(fmt.Sprintf("preferences %s='%s'", key, val))

	return nil
}

// Preferences returns preferences
func (s *Service) Preferences() preferences.Preferences {
	return s._preferences
}

//////////////////////////////////////////////////////////
// SESSIONS
//////////////////////////////////////////////////////////

func (s *Service) setCredentials(accountID, session, vpnUser, vpnPass, wgPublicKey, wgPrivateKey, wgLocalIP string, wgKeyGenerated int64) error {
	// save session info
	s._preferences.SetSession(accountID,
		session,
		vpnUser,
		vpnPass,
		wgPublicKey,
		wgPrivateKey,
		wgLocalIP)

	// manually set info about WG keys timestamp
	if wgKeyGenerated > 0 {
		s._preferences.Session.WGKeyGenerated = time.Unix(wgKeyGenerated, 0)
		s._preferences.SavePreferences()
	}

	// notify clients about session update
	s._evtReceiver.OnServiceSessionChanged()

	// success
	s.startSessionChecker()
	return nil
}

// SessionNew creates new session
func (s *Service) SessionNew(accountID string, forceLogin bool) (
	apiCode int,
	apiErrorMsg string,
	accountInfo preferences.AccountStatus,
	err error) {

	// delete current session (if exists)
	if err := s.SessionDelete(); err != nil {
		log.Error("Creating new session -> Failed to delete active session: ", err)
	}

	// generate new keys for WireGuard
	publicKey, privateKey, err := wireguard.GenerateKeys(platform.WgToolBinaryPath())
	if err != nil {
		log.Warning("Failed to generate wireguard keys for new session: %s", err)
	}

	log.Info("Logging in...")
	defer func() {
		if err != nil {
			log.Info("Logging in - FAILED: ", err)
		} else {
			log.Info("Logging in - SUCCESS")

		}
	}()
	successResp, errorLimitResp, apiErr, err := s._api.SessionNew(accountID, publicKey, forceLogin)

	apiCode = 0
	if apiErr != nil {
		apiCode = apiErr.Status
	}

	if err != nil {
		// if SessionsLimit response
		if errorLimitResp != nil {
			accountInfo = s.createAccountStatus(errorLimitResp.SessionLimitData)
			return apiCode, apiErr.Message, accountInfo, err
		}

		// in case of other API error
		if apiErr != nil {
			return apiCode, apiErr.Message, accountInfo, err
		}

		// not API error
		return apiCode, "", accountInfo, err
	}

	if successResp == nil {
		return apiCode, "", accountInfo, fmt.Errorf("unexpected error when creating a new session")
	}

	// get account status info
	accountInfo = s.createAccountStatus(successResp.ServiceStatus)

	s.setCredentials(accountID,
		successResp.Token,
		successResp.VpnUsername,
		successResp.VpnPassword,
		publicKey,
		privateKey,
		successResp.WireGuard.IPAddress, 0)

	return apiCode, "", accountInfo, nil
}

// SessionDelete removes session info
func (s *Service) SessionDelete() error {
	return s.logOut(true)
}

func (s *Service) logOut(needToDeleteOnBackend bool) error {

	// stop session checker (use goroutine to avoid deadlocks)
	go s.stopSessionChecker()

	// stop WG keys rotation
	s._wgKeysMgr.StopKeysRotation()

	if needToDeleteOnBackend {
		session := s.Preferences().Session
		if session.IsLoggedIn() {
			log.Info("Logging out")
			err := s._api.SessionDelete(session.Session)
			if err != nil {
				return err
			}
		}
	}

	s._preferences.SetSession("", "", "", "", "", "", "")

	// notify clients about session update
	s._evtReceiver.OnServiceSessionChanged()

	return nil
}

// RequestSessionStatus receives session status
func (s *Service) RequestSessionStatus() (
	apiCode int,
	apiErrorMsg string,
	sessionToken string,
	accountInfo preferences.AccountStatus,
	err error) {

	session := s.Preferences().Session
	if session.IsLoggedIn() == false {
		return apiCode, "", "", accountInfo, ErrorNotLoggedIn{}
	}

	log.Info("Requesting session status...")
	stat, apiErr, err := s._api.SessionStatus(session.Session)
	log.Info("Session status request: done")

	currSession := s.Preferences().Session
	if currSession.Session != session.Session {
		// It could happen that logout\login was performed during the session check
		// Ignoring result if there is already a new session
		log.Info("Ignoring requested session status result. Local session already changed.")
		return apiCode, "", "", accountInfo, ErrorNotLoggedIn{}
	}

	apiCode = 0
	if apiErr != nil {
		apiCode = apiErr.Status

		// Session not found - can happens when user forced to logout from another device
		if apiCode == types.SessionNotFound {
			// Logging out now
			log.Info("Session not found. Logging out.")
			s.logOut(false)
		}

		// notify clients that account not active
		if apiCode == types.AccountNotActive {
			// notify about account status
			s._evtReceiver.OnAccountStatus(session.Session, accountInfo)
			return apiCode, apiErr.Message, session.Session, preferences.AccountStatus{Active: false}, err
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
	// notify about account status
	s._evtReceiver.OnAccountStatus(session.Session, accountInfo)

	// success
	return apiCode, "", session.Session, accountInfo, nil
}

func (s *Service) createAccountStatus(apiResp types.ServiceStatusAPIResp) preferences.AccountStatus {
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
	if session.IsLoggedIn() == false {
		return
	}

	s._sessionCheckerStopChn = make(chan struct{})
	go func() {
		log.Info("Session checker started")
		defer log.Info("Session checker stopped")

		stopChn := s._sessionCheckerStopChn
		for {
			// wait for timeout or stop request
			select {
			case <-stopChn:
				return
			case <-time.After(SessionCheckInterval):
				break
			}

			// check status
			s.RequestSessionStatus()

			// if not logged-in - no sense to check status anymore
			session := s.Preferences().Session
			if session.IsLoggedIn() == false {
				return
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
func (s *Service) WireGuardSaveNewKeys(wgPublicKey string, wgPrivateKey string, wgLocalIP string) {
	s._preferences.UpdateWgCredentials(wgPublicKey, wgPrivateKey, wgLocalIP)

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
		if s.Connected() == false {
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

	interval := p.Session.WGKeysRegenInerval

	//----------------------------------------------------------------
	// ONLY FOR TESTS!
	// Interval change 1 day => 1 minute
	// interval = time.Minute * (interval / (time.Hour * 24))
	// log.Debug(fmt.Sprintf("(TESTING) Changed WG keys rotation interval %v => %v", p.Session.WGKeysRegenInerval, interval))
	//----------------------------------------------------------------

	return p.Session.Session,
		p.Session.WGPublicKey,
		p.Session.WGPrivateKey,
		p.Session.WGLocalIP,
		p.Session.WGKeyGenerated,
		interval //p.Session.WGKeysRegenInerval
}

// WireGuardGenerateKeys - generate new wireguard keys
func (s *Service) WireGuardGenerateKeys(updateIfNecessary bool) error {
	if s._preferences.Session.IsLoggedIn() == false {
		return ErrorNotLoggedIn{}
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

//////////////////////////////////////////////////////////
// Internal methods
//////////////////////////////////////////////////////////

func (s *Service) setPreferences(p preferences.Preferences) {
	//if reflect.DeepEqual(s._preferences, p) == false {
	if s._preferences != p {
		s._preferences = p
		s._preferences.SavePreferences()
	}
}
