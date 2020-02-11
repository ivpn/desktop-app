package service

import (
	"fmt"
	"ivpn/daemon/logger"
	"ivpn/daemon/netinfo"
	"ivpn/daemon/service/api"
	"ivpn/daemon/service/firewall"
	"ivpn/daemon/vpn"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sparrc/go-ping"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("servc")
}

// Service - IVPN service
type service struct {
	serversUpdater    ServersUpdater
	netChangeDetector NetChangeDetector
	vpn               vpn.Process
	preferences       Preferences
	connectMutex      sync.Mutex

	// Note: Disconnect() function will wait until VPN fully disconnects
	runningWG sync.WaitGroup
}

// CreateService - service constructor
func CreateService(updater ServersUpdater, netChDetector NetChangeDetector) (Service, error) {
	if updater == nil {
		return nil, errors.New("ServersUpdater is not defined")
	}

	serv := &service{
		serversUpdater:    updater,
		netChangeDetector: netChDetector}

	if err := serv.init(); err != nil {
		return nil, fmt.Errorf("service initialisaton error : %w", err)
	}

	return serv, nil
}

// OnControlConnectionClosed - Perform reqired operations when protocol (controll channel with UI application) was closed
// (for example, we must disable firewall (if it not persistant))
// Must be called by protocol object
// Return parameters:
// - isServiceMustBeClosed: true informing that service have to be closed ("Stop IVPN Agent when application is not running" feature)
// - err: error
func (s *service) OnControlConnectionClosed() (isServiceMustBeClosed bool, err error) {
	isServiceMustBeClosed = s.preferences.IsStopOnClientDisconnect
	// disable firewall if it not persistant
	if !s.preferences.IsFwPersistant {
		log.Info("Control connection was closed. Disabling firewall.")
		err = firewall.SetEnabled(false)
	}
	return isServiceMustBeClosed, err
}

// ServersList - get VPN servers info
func (s *service) ServersList() (*api.ServersInfoResponse, error) {
	return s.serversUpdater.GetServers()
}

// connect
func (s *service) Connect(vpnProc vpn.Process, manualDNS net.IP, stateChan chan<- vpn.StateInfo) error {
	var connectRoutinesWaiter sync.WaitGroup

	// stop active connection (if exists)
	if err := s.Disconnect(); err != nil {
		return errors.New("failed to connect. Unable to stop active connection: " + err.Error())
	}

	s.connectMutex.Lock()
	defer s.connectMutex.Unlock()

	s.runningWG.Add(1)
	defer s.runningWG.Done()

	log.Info("Connecting...")
	// save vpn object
	s.vpn = vpnProc

	internalStateChan := make(chan vpn.StateInfo, 1)
	stopChannel := make(chan bool, 1)

	// finalyze everything
	defer func() {
		if r := recover(); r != nil {
			log.Error("On finalyzing VPN stop: ", r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
		}

		// Ensure that routing-change detector is stopped (we do not need it when VPN disconnected)
		s.netChangeDetector.Stop()

		// notify firewall that client is disconnected
		err := firewall.ClientDisconnected()
		if err != nil {
			log.Error("Error on notifying FW about disconnected client:", err)
		}

		// notify routines to stop
		close(stopChannel)

		// resetting manual DNS (if it is necessary)
		err = vpnProc.ResetManualDNS()
		if err != nil {
			log.Error("Error resetting manual DNS:", err)
		}

		connectRoutinesWaiter.Wait()

		// Forget VPN object
		s.vpn = nil

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
				// forward state to 'stateChan'
				stateChan <- state

				log.Info(fmt.Sprintf("State: %v", state))

				// internally process VPN state change
				switch state.State {

				case vpn.RECONNECTING:
					// Disable routing-change detector when reconnecting
					s.netChangeDetector.Stop()

				case vpn.CONNECTED:
					// start routing chnage detection
					if netInterface, err := netinfo.InterfaceByIPAddr(state.ClientIP); err != nil {
						log.Error(fmt.Sprintf("Unable to inialize routing change detection. Failed to get interface '%s'", state.ClientIP.String()))
					} else {

						log.Info("Starting route change detection")
						s.netChangeDetector.Start(routingChangeChan, netInterface)
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
				if s.vpn.IsPaused() {
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
		err = errors.Wrap(err, "Unable to set DNS")
		log.Error(err.Error())
		return err
	}

	log.Info("Starting VPN process")
	// connect: start VPN process and wait untill it finishes
	err = vpnProc.Connect(internalStateChan)
	if err != nil {
		err = errors.Wrap(err, "Connection error")
		log.Error(err.Error())
		return err
	}

	return nil
}

// disconnect
func (s *service) Disconnect() error {
	vpn := s.vpn
	if vpn == nil {
		return nil
	}

	log.Info("Disconnecting...")

	// stop detections for routing changes
	s.netChangeDetector.Stop()

	// stop VPN
	if err := vpn.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect VPN: %w", err)
	}

	s.runningWG.Wait()

	return nil
}

func (s *service) Pause() error {
	vpn := s.vpn
	if vpn == nil {
		return nil
	}

	log.Info("Pausing...")
	return vpn.Pause()
}

func (s *service) Resume() error {
	vpn := s.vpn
	if vpn == nil {
		return nil
	}

	log.Info("Resuming...")
	return vpn.Resume()
}

func (s *service) SetKillSwitchState(isEnabled bool) error {
	return firewall.SetEnabled(isEnabled)
}

func (s *service) KillSwitchState() (bool, error) {
	return firewall.GetEnabled()
}

func (s *service) SetManualDNS(dns net.IP) error {
	vpn := s.vpn
	if vpn == nil {
		return nil
	}

	if err := firewall.SetManualDNS(dns); err != nil {
		return fmt.Errorf("failed to set manual DNS: %w", err)
	}

	return vpn.SetManualDNS(dns)
}

func (s *service) ResetManualDNS() error {
	vpn := s.vpn
	if vpn == nil {
		return nil
	}

	if err := firewall.SetManualDNS(nil); err != nil {
		return fmt.Errorf("failed to reset manual DNS: %w", err)
	}

	return vpn.ResetManualDNS()
}

func (s *service) PingServers(retryCount int, timeoutMs int) (map[string]int, error) {
	vpn := s.vpn
	if vpn != nil {
		log.Info("Servers pinging skipped due to connected state")
		return nil, nil
	}

	if retryCount <= 0 || timeoutMs <= 0 {
		log.Debug("Servers pinging skipped: arguments value is 0")
		return nil, nil
	}

	// get servers info
	servers, err := s.serversUpdater.GetServers()
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
		pinger.Interval = time.Millisecond * 1000 // do not use small interval (<350ms). Possible unexpected behavior: pings never return simetimes
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

func (s *service) SetKillSwitchIsPersistent(isPersistant bool) error {
	prefs := s.preferences
	prefs.IsFwPersistant = isPersistant
	s.setPreferences(prefs)

	return firewall.SetPersistant(isPersistant)
}

func (s *service) SetKillSwitchAllowLAN(isAllowLan bool) error {
	return s.setKillSwitchAllowLAN(isAllowLan, s.preferences.IsFwAllowLANMulticast)
}

func (s *service) SetKillSwitchAllowLANMulticast(isAllowLanMulticast bool) error {
	return s.setKillSwitchAllowLAN(s.preferences.IsFwAllowLAN, isAllowLanMulticast)
}

func (s *service) setKillSwitchAllowLAN(isAllowLan bool, isAllowLanMulticast bool) error {
	prefs := s.preferences
	prefs.IsFwAllowLAN = isAllowLan
	prefs.IsFwAllowLANMulticast = isAllowLanMulticast
	s.setPreferences(prefs)

	return firewall.AllowLAN(prefs.IsFwAllowLAN, prefs.IsFwAllowLANMulticast)
}

func (s *service) SetPreference(key string, val string) error {
	prefs := s.preferences

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
	log.Info("Preferences=", fmt.Sprintf("%+v", s.preferences))

	return nil
}

func (s *service) Preferences() Preferences {
	return s.preferences
}

//////////////////////////////////////////////////////////
// Internal methods
//////////////////////////////////////////////////////////

func (s *service) setPreferences(p Preferences) {
	if s.preferences != p {
		s.preferences = p
		s.preferences.savePreferences()
	}
}

func (s *service) init() error {
	if err := s.preferences.loadPreferences(); err != nil {
		log.Error("Failed to load service preferences: ", err)

		log.Warning("Saving default values for preferences")
		s.preferences.savePreferences()
	}

	// Init logger
	logger.Enable(s.preferences.IsLogging)

	// Init firewall
	if err := firewall.AllowLAN(s.preferences.IsFwAllowLAN, s.preferences.IsFwAllowLANMulticast); err != nil {
		log.Error("Failed to initialize firewall with AllowLAN preference value: ", err)
	}

	if s.preferences.IsFwPersistant {
		log.Info("Enabling firewal (persistant configuration)")
		if err := firewall.SetPersistant(true); err != nil {
			log.Error("Failed to enable firewall: ", err)
		}
	}
	return nil
}
