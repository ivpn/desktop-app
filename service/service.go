package service

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/ivpn/desktop-app-daemon/api"
	"github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/netinfo"
	"github.com/ivpn/desktop-app-daemon/service/firewall"
	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/service/preferences"
	"github.com/ivpn/desktop-app-daemon/vpn"
	"github.com/ivpn/desktop-app-daemon/vpn/openvpn"
	"github.com/ivpn/desktop-app-daemon/vpn/wireguard"

	"github.com/sparrc/go-ping"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("servc")
}

// Service - IVPN service
type Service struct {
	_evtReceiver           IServiceEventsReceiver
	_api                   *api.API
	_serversUpdater        IServersUpdater
	_netChangeDetector     INetChangeDetector
	_wgKeysMgr             IWgKeysManager
	_vpn                   vpn.Process
	_vpnReconnectRequested bool
	_preferences           preferences.Preferences
	_connectMutex          sync.Mutex

	// Note: Disconnect() function will wait until VPN fully disconnects
	_runningWG sync.WaitGroup

	_isServersPingInProgress bool
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
		return nil, fmt.Errorf("service initialisaton error : %w", err)
	}

	return serv, nil
}

func (s *Service) init() error {
	if err := s._preferences.LoadPreferences(); err != nil {
		log.Error("Failed to load service preferences: ", err)

		log.Warning("Saving default values for preferences")
		s._preferences.SavePreferences()
	}

	// Init logger
	logger.Enable(s._preferences.IsLogging)

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
		log.Error("Failed to intialize WG keys rotation:", err)
	} else {
		if err := s._wgKeysMgr.StartKeysRotation(); err != nil {
			log.Error("Failed to start WG keys rotation:", err)
		}
	}

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

// ServersUpdateNotifierChannel returns channel which is nitifying when servers was updated
func (s *Service) ServersUpdateNotifierChannel() chan struct{} {
	return s._serversUpdater.UpdateNotifierChannel()
}

// ConnectOpenVPN start OpenVPN connection
func (s *Service) ConnectOpenVPN(connectionParams openvpn.ConnectionParams, manualDNS net.IP, stateChan chan<- vpn.StateInfo) error {
	createVpnObjfunc := func() (vpn.Process, error) {
		prefs := s.Preferences()

		connectionParams.SetCredentials(prefs.Session.OpenVPNUser, prefs.Session.OpenVPNPass)

		vpnObj, err := openvpn.NewOpenVpnObject(
			platform.OpenVpnBinaryPath(),
			platform.OpenvpnConfigFile(),
			platform.OpenvpnLogFile(),
			prefs.IsObfsproxy,
			prefs.OpenVpnExtraParameters,
			connectionParams)

		if err != nil {
			return nil, fmt.Errorf("failed to create new openVPN object: %w", err)
		}
		return vpnObj, nil
	}

	return s.keepConnection(createVpnObjfunc, manualDNS, stateChan)
}

// ConnectWireGuard start WireGuard connection
func (s *Service) ConnectWireGuard(connectionParams wireguard.ConnectionParams, manualDNS net.IP, stateChan chan<- vpn.StateInfo) error {
	// stop active connection (if exists)
	if err := s.Disconnect(); err != nil {
		return fmt.Errorf("failed to connect. Unable to stop active connection: %w", err)
	}

	// Update WG keys, if necessary
	err := s.WireGuardGenerateKeys(true)
	if err != nil {
		log.Warning("Failed to regenerate WireGuard keys: ", err)
		// TODO: notify UI
	}

	createVpnObjfunc := func() (vpn.Process, error) {
		session := s.Preferences().Session

		localip := net.ParseIP(session.WGLocalIP)
		if localip == nil {
			return nil, fmt.Errorf("error updating WG connection preferences (failed parsing local IP)")
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

	return s.keepConnection(createVpnObjfunc, manualDNS, stateChan)
}

func (s *Service) keepConnection(createVpnObj func() (vpn.Process, error), manualDNS net.IP, stateChan chan<- vpn.StateInfo) error {
	defer func() { s._vpnReconnectRequested = false }()

	prefs := s.Preferences()
	if prefs.Session.IsLoggedIn() == false {
		return ErrorNotLoggedIn{}
	}

	for {
		s._vpnReconnectRequested = false
		// create new VPN object
		vpnObj, err := createVpnObj()
		if err != nil {
			return fmt.Errorf("failed to create VPN object: %w", err)
		}

		// start connection
		err = s.connect(vpnObj, manualDNS, stateChan)
		if err != nil {
			return err
		}

		// retry, if reconnection requested
		if s._vpnReconnectRequested {
			log.Info("Reconnecting...")
			continue
		}

		// stop loop
		break
	}

	return nil
}

// Connect connect vpn
func (s *Service) connect(vpnProc vpn.Process, manualDNS net.IP, stateChan chan<- vpn.StateInfo) error {
	var connectRoutinesWaiter sync.WaitGroup

	// stop active connection (if exists)
	if err := s.Disconnect(); err != nil {
		return fmt.Errorf("failed to connect. Unable to stop active connection: %w", err)
	}

	s._connectMutex.Lock()
	defer s._connectMutex.Unlock()

	s._runningWG.Add(1)
	defer s._runningWG.Done()

	log.Info("Connecting...")
	// save vpn object
	s._vpn = vpnProc

	internalStateChan := make(chan vpn.StateInfo, 1)
	stopChannel := make(chan bool, 1)

	// finalyze everything
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
		log.Info("Vpn state forwarder started")
		defer func() {
			log.Info("Vpn state forwarder stopped")
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
					// start routing chnage detection
					if netInterface, err := netinfo.InterfaceByIPAddr(state.ClientIP); err != nil {
						log.Error(fmt.Sprintf("Unable to inialize routing change detection. Failed to get interface '%s'", state.ClientIP.String()))
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
					log.Info("Route change detected. Disconnecting...")
					// Disconnect (client will request then reconnection, because of unexpected disconnection)
					go func() {
						defer func() {
							if r := recover(); r != nil {
								log.Error("PANIC: ", r)
							}
						}()
						s.Disconnect() // use separate routine to disconnect
					}()

					isRuning = false
				}
			case <-stopChannel: // triggered when the stopChannel is closed
				isRuning = false
			}
		}
	}()

	log.Info("Initialising...")
	if err := vpnProc.Init(); err != nil {
		return fmt.Errorf("failed to initialise VPN object: %w", err)
	}

	log.Info("Initializing firewall")
	// Add host IP to firewall exceptions
	err := firewall.AddHostsToExceptions(vpnProc.DestinationIPs())
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
	// connect: start VPN process and wait untill it finishes
	err = vpnProc.Connect(internalStateChan)
	if err != nil {
		err = fmt.Errorf("connection error: %w", err)
		log.Error(err.Error())
		return err
	}

	return nil
}

// Disconnect disconnect vpn
func (s *Service) Disconnect() error {
	vpn := s._vpn
	if vpn == nil {
		return nil
	}

	log.Info("Disconnecting...")

	// stop detections for routing changes
	s._netChangeDetector.Stop()

	// stop VPN
	if err := vpn.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect VPN: %w", err)
	}

	s._runningWG.Wait()

	return nil
}

// Connected returns 'true' if VPN connected
func (s *Service) Connected() bool {
	if s._vpn == nil {
		return false
	}
	return true
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

	if err := firewall.SetManualDNS(dns); err != nil {
		return fmt.Errorf("failed to set manual DNS: %w", err)
	}

	return vpn.SetManualDNS(dns)
}

// ResetManualDNS set dns to default
func (s *Service) ResetManualDNS() error {
	vpn := s._vpn
	if vpn == nil {
		return nil
	}

	if err := firewall.SetManualDNS(nil); err != nil {
		return fmt.Errorf("failed to reset manual DNS: %w", err)
	}

	return vpn.ResetManualDNS()
}

// PingServers ping vpn servers
func (s *Service) PingServers(retryCount int, timeoutMs int) (map[string]int, error) {

	// do not allow multimple ping request simultaneously
	if s._isServersPingInProgress {
		log.Info("Servers pinging skipped. Ping already in progress")
		return nil, nil
	}
	defer func() { s._isServersPingInProgress = false }()
	s._isServersPingInProgress = true

	vpn := s._vpn
	if vpn != nil {
		log.Info("Servers pinging skipped due to connected state")
		return nil, nil
	}

	if retryCount <= 0 || timeoutMs <= 0 {
		log.Debug("Servers pinging skipped: arguments value is 0")
		return nil, nil
	}

	// get servers info
	servers, err := s._serversUpdater.GetServers()
	if err != nil {
		log.Info("Servers ping failed (unable to get servers list): " + err.Error())
		return nil, err
	}

	// OS-specific preparations (e.g. we need to add servers IPs to firewall exceptions list)
	if err := s.implIsGoingToPingServers(servers); err != nil {
		log.Info("Servers ping failed : " + err.Error())
		return nil, err
	}

	// initialize waiter (will wait to finish all go-routines)
	var waiter sync.WaitGroup

	type pair struct {
		host string
		ping int
	}

	resultChan := make(chan pair, 1)
	// define generic ping function
	pingFunc := func(ip string) {
		// notify waiter: goroutine is finished
		defer waiter.Done()

		pinger, err := ping.NewPinger(ip)
		if err != nil {
			log.Error("Pinger creation error: " + err.Error())
			return
		}

		pinger.SetPrivileged(true)
		pinger.Count = retryCount
		pinger.Interval = time.Millisecond * 1000 // do not use small interval (<350ms). Possible unexpected behavior: pings never return sometimes
		pinger.Timeout = time.Millisecond * time.Duration(timeoutMs)

		pinger.Run()

		stat := pinger.Statistics()

		// Pings filtering ...
		// there is a chance that one ping responce is much higher than the rest recived responses
		// This, for example, observed on some virtual machines. The first ping result is catastrophically higher than the rest
		// Hera we are ignoring such situations (ignoring higest pings when necessary)
		var avgPing time.Duration = 0
		maxAllowedTTL := float32(stat.AvgRtt) * 1.3
		if stat.PacketLoss < 0 || float32(stat.MaxRtt) < maxAllowedTTL {
			avgPing = stat.AvgRtt
			//log.Debug(int(stat.AvgRtt/time.Millisecond), " == ", int(avgPing/time.Millisecond), "\t", stat)
		} else {
			cntResults := 0
			for _, p := range stat.Rtts {
				if float32(p) >= maxAllowedTTL {
					continue
				}
				avgPing += p
				cntResults++
			}
			if cntResults > 0 {
				avgPing = avgPing / time.Duration(cntResults)
			} else {
				avgPing = stat.AvgRtt
			}
			//log.Debug(int(stat.AvgRtt/time.Millisecond), " -> ", int(avgPing/time.Millisecond), "\t", stat)
		}

		resultChan <- pair{host: ip, ping: int(avgPing / time.Millisecond)}

		// ... pings filtering

		// Original pings data:
		//resultChan <- pair{host: ip, ping: int(stat.AvgRtt / time.Millisecond)}
	}

	log.Info("Pingging servers...")

	// ping each OpenVPN server
	for _, s := range servers.OpenvpnServers {
		if len(s.IPAddresses) <= 0 {
			continue
		}
		waiter.Add(1) // +1 goroutine to wait
		go pingFunc(s.IPAddresses[0])
	}

	// ping each WireGuard server
	for _, s := range servers.WireguardServers {
		if len(s.Hosts) <= 0 {
			continue
		}
		waiter.Add(1) // +1 goroutine to wait
		go pingFunc(s.Hosts[0].Host)
	}

	successfullyPinged := 0
	retMap := make(map[string]int)
	done := make(chan bool)
	go func() {
		for {
			select {
			case r := <-resultChan:
				retMap[r.host] = r.ping
				if r.ping > 0 {
					successfullyPinged = successfullyPinged + 1
				}
			case <-done:
				return
			}
		}
	}()

	waiter.Wait()
	done <- true

	log.Info(fmt.Sprintf("Pinged %d of %d servers (%d successfully)", len(retMap), len(servers.OpenvpnServers)+len(servers.WireguardServers), successfullyPinged))

	return retMap, nil
}

// SetKillSwitchState enable\disable killswitch
func (s *Service) SetKillSwitchState(isEnabled bool) error {
	return firewall.SetEnabled(isEnabled)
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

	return firewall.SetPersistant(isPersistant)
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

	return firewall.AllowLAN(prefs.IsFwAllowLAN, prefs.IsFwAllowLANMulticast)
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
	case "open_vpn_extra_parameters":
		prefs.OpenVpnExtraParameters = val
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
	sucessResp, errorLimitResp, apiErr, err := s._api.SessionNew(accountID, publicKey, forceLogin)

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

	if sucessResp == nil {
		return apiCode, "", accountInfo, fmt.Errorf("unexpected error when creating a new session")
	}

	// get account status info
	accountInfo = s.createAccountStatus(sucessResp.ServiceStatus)

	// save session info
	s._preferences.SetSession(accountID,
		sucessResp.Token,
		sucessResp.VpnUsername,
		sucessResp.VpnPassword,
		publicKey,
		privateKey,
		sucessResp.WireGuard.IPAddress)

	// notify clients about sesssion update
	s._evtReceiver.OnServiceSessionChanged()

	// success
	return apiCode, "", accountInfo, nil
}

// SessionDelete removes session info
func (s *Service) SessionDelete() error {
	// stop WG keys rotation
	s._wgKeysMgr.StopKeysRotation()

	session := s.Preferences().Session
	if session.IsLoggedIn() {
		log.Info("Logging out")
		err := s._api.SessionDelete(session.Session)
		if err != nil {
			return err
		}
	}

	s._preferences.SetSession("", "", "", "", "", "", "")

	// notify clients about sesssion update
	s._evtReceiver.OnServiceSessionChanged()

	return nil
}

// SessionStatus receives session status
func (s *Service) SessionStatus() (
	apiCode int,
	apiErrorMsg string,
	accountInfo preferences.AccountStatus,
	err error) {

	session := s.Preferences().Session
	stat, apiErr, err := s._api.SessionStatus(session.Session)

	apiCode = 0
	if apiErr != nil {
		apiCode = apiErr.Status
	}

	if err != nil {
		// in case of other API error
		if apiErr != nil {
			return apiCode, apiErr.Message, accountInfo, err
		}

		// not API error
		return apiCode, "", accountInfo, err
	}

	if stat == nil {
		return apiCode, "", accountInfo, fmt.Errorf("unexpected error when creating requesting session status")
	}

	// get account status info
	accountInfo = s.createAccountStatus(*stat)

	// success
	return apiCode, "", accountInfo, nil
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

//////////////////////////////////////////////////////////
// WireGuard keys
//////////////////////////////////////////////////////////

// WireGuardSaveNewKeys saves WG keys
func (s *Service) WireGuardSaveNewKeys(wgPublicKey string, wgPrivateKey string, wgLocalIP net.IP) {
	s._preferences.UpdateWgCredentials(wgPublicKey, wgPrivateKey, wgLocalIP.String())

	// notify clients about sesssion (wg keys) update
	s._evtReceiver.OnServiceSessionChanged()

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
	s._vpnReconnectRequested = true
	s.Disconnect()
}

// WireGuardSetKeysRotationInterval change WG key rotation interval
func (s *Service) WireGuardSetKeysRotationInterval(interval int64) {
	s._preferences.Session.WGKeysRegenInerval = time.Second * time.Duration(interval)
	s._preferences.SavePreferences()

	// restart WG keys rotation
	if err := s._wgKeysMgr.StartKeysRotation(); err != nil {
		log.Error(err)
	}

	// notify clients about sesssion (wg keys) update
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
	if s._preferences.Session.IsLoggedIn() == false {
		return ErrorNotLoggedIn{}
	}

	// Update WG keys, if necessary
	// No sense to try to update if firewall is enabled (VPN is in disconnected state now)
	enabled, _, _, _, err := s.KillSwitchState()

	if err == nil && enabled == false {
		var err error
		if updateIfNecessary {
			err = s._wgKeysMgr.UpdateKeysIfNecessary()
		} else {
			err = s._wgKeysMgr.GenerateKeys()
		}
		if err != nil {
			return fmt.Errorf("failed to regenerate WireGuard keys: %w", err)
		}
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
