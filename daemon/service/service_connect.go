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
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	api_types "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/obfsproxy"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/firewall"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/service/platform/filerights"
	"github.com/ivpn/desktop-app/daemon/service/srverrors"
	"github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/vpn"
	"github.com/ivpn/desktop-app/daemon/vpn/openvpn"
	"github.com/ivpn/desktop-app/daemon/vpn/wireguard"
)

func (s *Service) ValidateConnectionParameters(params types.ConnectionParams, isCanFix bool) (types.ConnectionParams, error) {
	if params.VpnType == vpn.WireGuard {
		// WireGuard connection parameters
		if len(params.WireGuardParameters.EntryVpnServer.Hosts) <= 0 {
			return params, fmt.Errorf("no hosts defined for WireGuard connection")
		}
		if len(params.WireGuardParameters.MultihopExitServer.Hosts) > 0 {
			if mhErr := s.IsCanConnectMultiHop(); mhErr != nil {
				if !isCanFix {
					return params, mhErr
				}
				log.Info("Multi-Hop connection is not allowed. Using Single-Hop.")
				params.WireGuardParameters.MultihopExitServer = types.MultiHopExitServer_WireGuard{}
			}
		}
	} else {
		// OpenVPN connection parameters
		if len(params.OpenVpnParameters.EntryVpnServer.Hosts) <= 0 {
			return params, fmt.Errorf("no hosts defined for OpenVPN connection")
		}
		if len(params.OpenVpnParameters.MultihopExitServer.Hosts) > 0 {
			if mhErr := s.IsCanConnectMultiHop(); mhErr != nil {
				if !isCanFix {
					return params, mhErr
				}
				log.Info("Multi-Hop connection is not allowed. Using Single-Hop.")
				params.OpenVpnParameters.MultihopExitServer = types.MultiHopExitServer_OpenVpn{}
			}
		}
	}
	return params, nil
}

func (s *Service) Connect(params types.ConnectionParams) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("panic on connect: " + fmt.Sprint(r))
			log.Error(err)
		}
	}()

	// keep last used connection params
	s.setConnectionParams(params)

	retManualDNS := params.ManualDNS

	if vpn.Type(params.VpnType) == vpn.OpenVPN {
		// PARAMETERS VALIDATION
		// parsing hosts
		var hosts []net.IP
		for _, v := range params.OpenVpnParameters.EntryVpnServer.Hosts {
			hosts = append(hosts, net.ParseIP(v.Host))
		}
		if len(hosts) < 1 {
			return fmt.Errorf("VPN host not defined")
		}
		// in case of multiple hosts - take random host from the list
		host := hosts[0]
		if len(hosts) > 1 {
			if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(hosts)))); err == nil {
				host = hosts[rnd.Int64()]
			}
		}

		// nothing from supported proxy types should be in this parameter
		proxyType := params.OpenVpnParameters.Proxy.Type
		if len(proxyType) > 0 && proxyType != "http" && proxyType != "socks" {
			proxyType = ""
		}

		// Multi-Hop
		var exitHostValue *api_types.OpenVPNServerHostInfo
		multihopExitHosts := params.OpenVpnParameters.MultihopExitServer.Hosts
		if len(multihopExitHosts) > 0 {
			exitHostValue = &multihopExitHosts[0]
			if len(multihopExitHosts) > 1 {
				if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(multihopExitHosts)))); err == nil {
					exitHostValue = &multihopExitHosts[rnd.Int64()]
				}
			}
		}

		// only one-line parameter is allowed
		proxyUsername := strings.Split(params.OpenVpnParameters.Proxy.Username, "\n")[0]
		proxyPassword := strings.Split(params.OpenVpnParameters.Proxy.Password, "\n")[0]

		// CONNECTION
		// OpenVPN connection parameters
		var connectionParams openvpn.ConnectionParams
		if exitHostValue != nil {
			// Check is it allowed to connect multihop
			if mhErr := s.IsCanConnectMultiHop(); mhErr != nil {
				return mhErr
			}

			// Multi-Hop
			connectionParams = openvpn.CreateConnectionParams(
				exitHostValue.Hostname,
				params.OpenVpnParameters.Port.Protocol > 0, // is TCP
				exitHostValue.MultihopPort,
				host,
				proxyType,
				net.ParseIP(params.OpenVpnParameters.Proxy.Address),
				params.OpenVpnParameters.Proxy.Port,
				proxyUsername,
				proxyPassword)
		} else {
			// Single-Hop
			connectionParams = openvpn.CreateConnectionParams(
				"",
				params.OpenVpnParameters.Port.Protocol > 0, // is TCP
				params.OpenVpnParameters.Port.Port,
				host,
				proxyType,
				net.ParseIP(params.OpenVpnParameters.Proxy.Address),
				params.OpenVpnParameters.Proxy.Port,
				proxyUsername,
				proxyPassword)
		}

		return s.ConnectOpenVPN(connectionParams, retManualDNS, params.FirewallOn, params.FirewallOnDuringConnection)

	} else if vpn.Type(params.VpnType) == vpn.WireGuard {
		hosts := params.WireGuardParameters.EntryVpnServer.Hosts
		multihopExitHosts := params.WireGuardParameters.MultihopExitServer.Hosts

		// filter hosts: use IPv6 hosts
		if params.IPv6 {
			ipv6Hosts := append(hosts[0:0], hosts...)
			n := 0
			for _, h := range ipv6Hosts {
				if h.IPv6.LocalIP != "" {
					ipv6Hosts[n] = h
					n++
				}
			}
			if n == 0 {
				if params.IPv6Only {
					return fmt.Errorf("unable to make IPv6 connection inside tunnel. Server does not support IPv6")
				}
			} else {
				hosts = ipv6Hosts[:n]
			}
		}

		// filter exit servers (Multi-Hop connection):
		// 1) each exit server must have initialized 'multihop_port' field
		// 2) (in case of IPv6Only) IPv6 local address should be defined
		if len(multihopExitHosts) > 0 {
			isHasMHPort := false
			ipv6ExitHosts := append(multihopExitHosts[0:0], multihopExitHosts...)
			n := 0
			for _, h := range ipv6ExitHosts {
				if h.MultihopPort == 0 {
					continue
				}
				isHasMHPort = true
				if params.IPv6 && h.IPv6.LocalIP == "" {
					continue
				}

				ipv6ExitHosts[n] = h
				n++
			}
			if n == 0 {
				if !isHasMHPort {
					return fmt.Errorf("unable to make Multi-Hop connection inside tunnel. Exit server does not support Multi-Hop")
				}
				if params.IPv6Only {
					return fmt.Errorf("unable to make IPv6 Multi-Hop connection inside tunnel. Exit server does not support IPv6")
				}
			} else {
				multihopExitHosts = ipv6ExitHosts[:n]
			}
		}

		hostValue := hosts[0]
		if len(hosts) > 1 {
			if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(hosts)))); err == nil {
				hostValue = hosts[rnd.Int64()]
			}
		}

		var exitHostValue *api_types.WireGuardServerHostInfo
		if len(multihopExitHosts) > 0 {
			exitHostValue = &multihopExitHosts[0]
			if len(multihopExitHosts) > 1 {
				if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(multihopExitHosts)))); err == nil {
					exitHostValue = &multihopExitHosts[rnd.Int64()]
				}
			}
		}

		// prevent user-defined data injection: ensure that nothing except the base64 public key will be stored in the configuration
		if !helpers.ValidateBase64(hostValue.PublicKey) {
			return fmt.Errorf("WG public key is not base64 string")
		}

		hostLocalIP := net.ParseIP(strings.Split(hostValue.LocalIP, "/")[0])
		ipv6Prefix := ""
		if params.IPv6 {
			ipv6Prefix = strings.Split(hostValue.IPv6.LocalIP, "/")[0]
		}

		var connectionParams wireguard.ConnectionParams
		if exitHostValue != nil {
			// Check is it allowed to connect multihop
			if mhErr := s.IsCanConnectMultiHop(); mhErr != nil {
				return mhErr
			}

			// Multi-Hop
			connectionParams = wireguard.CreateConnectionParams(
				exitHostValue.Hostname,
				exitHostValue.MultihopPort,
				net.ParseIP(hostValue.Host),
				exitHostValue.PublicKey,
				hostLocalIP,
				ipv6Prefix,
				params.WireGuardParameters.Mtu)
		} else {
			// Single-Hop
			connectionParams = wireguard.CreateConnectionParams(
				"",
				params.WireGuardParameters.Port.Port,
				net.ParseIP(hostValue.Host),
				hostValue.PublicKey,
				hostLocalIP,
				ipv6Prefix,
				params.WireGuardParameters.Mtu)
		}

		return s.ConnectWireGuard(connectionParams, retManualDNS, params.FirewallOn, params.FirewallOnDuringConnection)

	}

	return fmt.Errorf("unexpected VPN type to connect (%v)", params.VpnType)
}

// ConnectOpenVPN start OpenVPN connection
func (s *Service) ConnectOpenVPN(connectionParams openvpn.ConnectionParams, manualDNS dns.DnsSettings, firewallOn bool, firewallDuringConnection bool) error {

	createVpnObjfunc := func() (vpn.Process, error) {
		prefs := s.Preferences()

		// checking if functionality accessible
		disabledFuncs := s.GetDisabledFunctions()
		if len(disabledFuncs.OpenVPNError) > 0 {
			return nil, fmt.Errorf(disabledFuncs.OpenVPNError)
		}
		if prefs.Obfs4proxy.IsObfsproxy() && len(disabledFuncs.ObfsproxyError) > 0 {
			return nil, fmt.Errorf(disabledFuncs.ObfsproxyError)
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

		// initialize obfsproxy parameters
		obfsParams := openvpn.ObfsParams{}
		if prefs.Obfs4proxy.IsObfsproxy() {
			obfsParams.Config = prefs.Obfs4proxy
			svrs, err := s.ServersList()
			if err != nil {
				return nil, fmt.Errorf("failed to initialize obfsproxy configuration: %w", err)
			}

			if connectionParams.IsMultihop() {
				// find host by hostname
				host, err := s.findOpenVpnHost(connectionParams.GetMultihopExitHostName(), nil, svrs.OpenvpnServers)
				if err != nil {
					return nil, fmt.Errorf("failed to initialize obfsproxy configuration: %w", err)
				}

				switch obfsParams.Config.Version {
				case obfsproxy.OBFS3:
					obfsParams.RemotePort = host.Obfs.Obfs3MultihopPort
				case obfsproxy.OBFS4:
					obfsParams.RemotePort = host.Obfs.Obfs4MultihopPort
					obfsParams.Obfs4Key = host.Obfs.Obfs4Key
				default:
					return nil, fmt.Errorf("failed to initialize obfsproxy configuration: unsupported obfs version: %d", obfsParams.Config.Version)
				}
			} else {
				switch obfsParams.Config.Version {
				case obfsproxy.OBFS3:
					obfsParams.RemotePort = svrs.Config.Ports.Obfs3.Port
				case obfsproxy.OBFS4:
					{
						// find host by host ip
						host, err := s.findOpenVpnHost("", connectionParams.GetHostIp(), svrs.OpenvpnServers)
						if err != nil {
							return nil, fmt.Errorf("failed to initialize obfsproxy configuration: %w", err)
						}

						obfsParams.RemotePort = svrs.Config.Ports.Obfs4.Port
						obfsParams.Obfs4Key = host.Obfs.Obfs4Key
					}
				default:
					return nil, fmt.Errorf("failed to initialize obfsproxy configuration: unsupported obfs version: %d", obfsParams.Config.Version)
				}
			}

		}

		// creating OpenVPN object
		vpnObj, err := openvpn.NewOpenVpnObject(
			platform.OpenVpnBinaryPath(),
			platform.OpenvpnConfigFile(),
			"",
			obfsParams,
			openVpnExtraParameters,
			connectionParams)

		if err != nil {
			return nil, fmt.Errorf("failed to create new openVPN object: %w", err)
		}
		return vpnObj, nil
	}

	return s.keepConnection(createVpnObjfunc, manualDNS, firewallOn, firewallDuringConnection)
}

// ConnectWireGuard start WireGuard connection
func (s *Service) ConnectWireGuard(connectionParams wireguard.ConnectionParams, manualDNS dns.DnsSettings, firewallOn bool, firewallDuringConnection bool) error {
	// stop active connection (if exists)
	if err := s.Disconnect(); err != nil {
		return fmt.Errorf("failed to connect. Unable to stop active connection: %w", err)
	}

	// checking if functionality accessible
	disabledFuncs := s.GetDisabledFunctions()
	if len(disabledFuncs.WireGuardError) > 0 {
		return fmt.Errorf(disabledFuncs.WireGuardError)
	}

	// Update WG keys, if necessary
	err := s.WireGuardGenerateKeys(true)
	if err != nil {
		// If new WG keys regeneration failed but we still have active keys - keep connecting
		// (this could happen, for example, when FW is enabled and we even not tried to make API request)
		// Return error only if the keys had to be regenerated more than 3 days ago.
		_, activePublicKey, _, _, lastUpdate, interval := s.WireGuardGetKeys()

		if len(activePublicKey) > 0 && lastUpdate.Add(interval).Add(time.Hour*24*3).After(time.Now()) {
			// continue connection
			log.Warning(fmt.Errorf("WG KEY generation failed (%w). But we keep connecting (will try to regenerate it next 3 days)", err))
		} else {
			return err
		}
	}

	createVpnObjfunc := func() (vpn.Process, error) {
		session := s.Preferences().Session

		if !session.IsWGCredentialsOk() {
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

	return s.keepConnection(createVpnObjfunc, manualDNS, firewallOn, firewallDuringConnection)
}

func (s *Service) keepConnection(createVpnObj func() (vpn.Process, error), manualDNS dns.DnsSettings, firewallOn bool, firewallDuringConnection bool) (retError error) {
	prefs := s.Preferences()
	if !prefs.Session.IsLoggedIn() {
		return srverrors.ErrorNotLoggedIn{}
	}

	defer func() {
		// If no any clients connected - disconnection notification will not be passed to user
		// In this case we are trying to save message into system log
		if !s._evtReceiver.IsClientConnected(false) {
			if retError != nil {
				s.systemLog(Error, "Failed to connect VPN: "+retError.Error())
			} else {
				s.systemLog(Info, "VPN disconnected")
			}
		}
	}()

	s._manualDNS = manualDNS

	// Not necessary to keep connection until we are not connected
	// So just 'Connect' required for now
	s._requiredVpnState = Connect

	// no delay before first reconnection
	delayBeforeReconnect := 0 * time.Second

	s._evtReceiver.OnVpnStateChanged(vpn.NewStateInfo(vpn.CONNECTING, "Connecting"))
	for {
		// create new VPN object
		vpnObj, err := createVpnObj()
		if err != nil {
			return fmt.Errorf("failed to create VPN object: %w", err)
		}

		lastConnectionTryTime := time.Now()

		// start connection
		connErr := s.connect(vpnObj, s._manualDNS, firewallOn, firewallDuringConnection)
		if connErr != nil {
			log.Error(fmt.Sprintf("Connection error: %s", connErr))
			if s._requiredVpnState == Connect {
				// throw error only on first try to connect
				// if we were already connected (_requiredVpnState==KeepConnection) - ignore error and try to reconnect
				return connErr
			}
		}

		// retry, if reconnection requested
		if s._requiredVpnState == KeepConnection {
			// notifying clients about reconnection
			s._evtReceiver.OnVpnStateChanged(vpn.NewStateInfo(vpn.RECONNECTING, "Reconnecting due to disconnection"))

			// no delay before reconnection (if last connection was long time ago)
			if time.Now().After(lastConnectionTryTime.Add(time.Second * 30)) {
				delayBeforeReconnect = 0
			}
			// no delay before reconnection if reconnection was requested by VPN object
			if connErr != nil {
				var reconnectReqErr *vpn.ReconnectionRequiredError
				if errors.As(connErr, &reconnectReqErr) {
					log.Info("VPN object requested re-connection")
					delayBeforeReconnect = 0
				}
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
				// consecutive re-connections has delay 5 seconds
				delayBeforeReconnect = time.Second * 5
				continue
			}
		}

		// stop loop
		break
	}

	return nil
}

// Connect connect vpn.
// Param 'firewallOn' - enable firewall before connection (if true - the parameter 'firewallDuringConnection' will be ignored).
// Param 'firewallDuringConnection' - enable firewall before connection and disable after disconnection (has effect only if Firewall not enabled before)
func (s *Service) connect(vpnProc vpn.Process, manualDNS dns.DnsSettings, firewallOn bool, firewallDuringConnection bool) error {
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
			// Note: reading from empty or closed channel will not lead to deadlock (immediately returns zero value)
			close(done)
		}
	}()

	var err error

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

		// ensure firewall removed rules for DNS
		firewall.OnChangeDNS(nil)

		// notify firewall that client is disconnected
		err := firewall.ClientDisconnected()
		if err != nil {
			log.Error("(stopping) error on notifying FW about disconnected client:", err)
		}

		// when we were requested to enable firewall for this connection
		// And initial FW state was disabled - we have to disable it back
		if firewallDuringConnection && !fwInitState {
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

		// Notify Split-Tunneling module about disconnected VPN status
		s.splitTunnelling_ApplyConfig()

		log.Info("VPN process stopped")
	}()

	// Signaling when the default routing is NOT over the 'interfaceToProtect' anymore
	routingChangeChan := make(chan struct{}, 1)
	// Signaling when there were some routing changes but 'interfaceToProtect' is still is the default route
	routingUpdateChan := make(chan struct{}, 1)

	destinationHostIP := vpnProc.DestinationIP()

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
				s._evtReceiver.OnVpnStateChanged(state)

				log.Info(fmt.Sprintf("State: %v", state))

				// internally process VPN state change
				switch state.State {

				case vpn.RECONNECTING:
					// Disable routing-change detector when reconnecting
					s._netChangeDetector.Stop()

					// Add host IP to firewall exceptions
					// Some OS-specific implementations (e.g. macOS) can remove server host from firewall rules after connection established
					// We have to allow it's IP to be able to reconnect
					const onlyForICMP = false
					const isPersistent = false
					err := firewall.AddHostsToExceptions([]net.IP{destinationHostIP}, onlyForICMP, isPersistent)
					if err != nil {
						log.Error("Unable to add host to firewall exceptions:", err.Error())
					}

				case vpn.CONNECTED:
					// since we are connected - keep connection (reconnect if unexpected disconnection)
					if s._requiredVpnState == Connect {
						s._requiredVpnState = KeepConnection
					}

					// If no any clients connected - connection notification will not be passed to user
					// In this case we are trying to save info message into system log
					if !s._evtReceiver.IsClientConnected(false) {
						s.systemLog(Info, "VPN connected")
					}

					// start routing change detection
					if netInterface, err := netinfo.InterfaceByIPAddr(state.ClientIP); err != nil {
						log.Error(fmt.Sprintf("Unable to initialize routing change detection. Failed to get interface '%s'", state.ClientIP.String()))
					} else {

						log.Info("Starting route change detection")
						s._netChangeDetector.Start(routingChangeChan, routingUpdateChan, netInterface)
					}

					// Inform firewall about client local IP
					firewall.ClientConnected(
						state.ClientIP, state.ClientIPv6,
						state.ClientPort,
						state.ServerIP, state.ServerPort,
						state.IsTCP)

					// Ensure firewall is configured to allow DNS communication
					// At this moment, firewall must be already configured for custom DNS
					// but if it still has no rule - apply DNS rules for default DNS
					if _, isInitialized := firewall.GetDnsInfo(); !isInitialized {
						d := dns.DnsSettingsCreate(vpnProc.DefaultDNS())
						firewall.OnChangeDNS(&d)
					}

					// save ClientIP/ClientIPv6 into vpn-session-info
					sInfo := s.GetVpnSessionInfo()
					sInfo.VpnLocalIPv4 = state.ClientIP
					sInfo.VpnLocalIPv6 = state.ClientIPv6
					s.SetVpnSessionInfo(sInfo)

					// Notify Split-Tunneling module about connected VPN status
					s.splitTunnelling_ApplyConfig()
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
			case <-routingChangeChan: // routing changed (the default routing is NOT over the 'interfaceToProtect' anymore)
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
			case <-routingUpdateChan: // there were some routing changes but 'interfaceToProtect' is still is the default route
				s._vpn.OnRoutingChanged()
				go func() {
					// Ensure that current DNS configuration is correct. If not - it re-apply the required configuration.
					// Currently, it is in use for macOS - like a DNS change monitor.
					err := dns.UpdateDnsIfWrongSettings()
					if err != nil {
						log.Error(fmt.Errorf("failed to update DNS settings: %w", err))
					}
				}()
			case <-stopChannel: // triggered when the stopChannel is closed
				isRuning = false
			}
		}
	}()

	// Initialize VPN: ensure everything is prepared for a new connection
	// (e.g. correct OpenVPN version or a previously started WireGuard service is stopped)
	log.Info("Initializing connection...")
	if err := vpnProc.Init(); err != nil {
		return fmt.Errorf("failed to initialize VPN object: %w", err)
	}

	// Split-Tunnelling: Checking default outbound IPs
	// (note: it is important to call this code after 'vpnProc.Init()')
	var sInfo VpnSessionInfo
	sInfo.OutboundIPv4, err = netinfo.GetOutboundIP(false)
	if err != nil {
		log.Warning(fmt.Errorf("failed to detect outbound IPv4 address: %w", err))
	}
	sInfo.OutboundIPv6, err = netinfo.GetOutboundIP(true)
	if err != nil {
		log.Warning(fmt.Errorf("failed to detect outbound IPv6 address: %w", err))
	}
	s.SetVpnSessionInfo(sInfo)

	log.Info("Initializing firewall")
	// ensure firewall has no rules for DNS
	firewall.OnChangeDNS(nil)
	// firewallOn - enable firewall before connection (if true - the parameter 'firewallDuringConnection' will be ignored)
	// firewallDuringConnection - enable firewall before connection and disable after disconnection (has effect only if Firewall not enabled before)
	if firewallOn {
		fw, err := firewall.GetEnabled()
		if err != nil {
			log.Error("Failed to check firewall state:", err.Error())
			return err
		}
		if !fw {
			if err := s.SetKillSwitchState(true); err != nil {
				log.Error("Failed to enable firewall:", err.Error())
				return err
			}
		}
	} else if firewallDuringConnection {
		// in case to enable FW for this connection parameter:
		// - check initial FW state
		// - if it disabled - enable it (will be disabled on disconnect)
		fw, err := firewall.GetEnabled()
		if err != nil {
			log.Error("Failed to check firewall state:", err.Error())
			return err
		}
		fwInitState = fw
		if !fwInitState {
			if err := s.SetKillSwitchState(true); err != nil {
				log.Error("Failed to enable firewall:", err.Error())
				return err
			}
		}
	}

	// Add host IP to firewall exceptions
	const onlyForICMP = false
	const isPersistent = false
	err = firewall.AddHostsToExceptions([]net.IP{destinationHostIP}, onlyForICMP, isPersistent)
	if err != nil {
		log.Error("Failed to start. Unable to add hosts to firewall exceptions:", err.Error())
		return err
	}

	log.Info("Initializing DNS")

	// Re-initialize DNS configuration according to user settings
	// It is applicable, for example for Linux: when the user changed DNS management style
	if err := dns.ApplyUserSettings(); err != nil {
		return err
	}

	// set manual DNS
	if manualDNS.IsEmpty() {
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
