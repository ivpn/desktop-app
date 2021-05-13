//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
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

package protocol

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	apitypes "github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/helpers/process"
	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/service/dns"
	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/service/platform/filerights"
	"github.com/ivpn/desktop-app-daemon/service/preferences"
	"github.com/ivpn/desktop-app-daemon/vpn"
	"github.com/ivpn/desktop-app-daemon/vpn/openvpn"
	"github.com/ivpn/desktop-app-daemon/vpn/wireguard"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("prtcl")
	rand.Seed(time.Now().UnixNano())
}

// Service - service interface
type Service interface {
	// OnControlConnectionClosed - Perform reqired operations when protocol (controll channel with UI application) was closed
	// (for example, we must disable firewall (if it not persistant))
	// Must be called by protocol object
	// Return parameters:
	// - isServiceMustBeClosed: true informing that service have to be closed ("Stop IVPN Agent when application is not running" feature)
	// - err: error
	OnControlConnectionClosed() (isServiceMustBeClosed bool, err error)

	// GetDisabledFunctions returns info about functions which are disabled
	// Some functionality can be not accessible
	// It can happen, for example, if some external binaries not installed
	// (e.g. obfsproxy or WireGuard on Linux)
	GetDisabledFunctions() (wgErr, ovpnErr, obfspErr error)

	ServersList() (*apitypes.ServersInfoResponse, error)
	PingServers(retryCount int, timeoutMs int) (map[string]int, error)
	ServersUpdateNotifierChannel() chan struct{}

	APIRequest(apiAlias string) (responseData []byte, err error)

	KillSwitchState() (isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast bool, err error)
	SetKillSwitchState(bool) error
	SetKillSwitchIsPersistent(isPersistant bool) error
	SetKillSwitchAllowLANMulticast(isAllowLanMulticast bool) error
	SetKillSwitchAllowLAN(isAllowLan bool) error

	Preferences() preferences.Preferences
	SetPreference(key string, val string) error

	SetManualDNS(dns net.IP) error
	ResetManualDNS() error

	ConnectOpenVPN(connectionParams openvpn.ConnectionParams, manualDNS net.IP, firewallOn bool, firewallDuringConnection bool, stateChan chan<- vpn.StateInfo) error
	ConnectWireGuard(connectionParams wireguard.ConnectionParams, manualDNS net.IP, firewallOn bool, firewallDuringConnection bool, stateChan chan<- vpn.StateInfo) error
	Disconnect() error
	Connected() bool

	Pause() error
	Resume() error
	IsPaused() bool

	SessionNew(accountID string, forceLogin bool, captchaID string, captcha string, confirmation2FA string) (
		apiCode int,
		apiErrorMsg string,
		accountInfo preferences.AccountStatus,
		rawResponse string,
		err error)

	SessionDelete() error
	RequestSessionStatus() (
		apiCode int,
		apiErrorMsg string,
		sessionToken string,
		accountInfo preferences.AccountStatus,
		err error)

	WireGuardGenerateKeys(updateIfNecessary bool) error
	WireGuardSetKeysRotationInterval(interval int64)

	GetWiFiCurrentState() (ssid string, isInsecureNetwork bool)
	GetWiFiAvailableNetworks() []string
}

// CreateProtocol - Create new protocol object
func CreateProtocol() (*Protocol, error) {
	return &Protocol{_connections: make(map[net.Conn]struct{})}, nil
}

// Protocol - TCP interface to communicate with IVPN application
type Protocol struct {
	_secret uint64

	// connections listener
	_connListener *net.TCPListener

	_connectionsMutex sync.RWMutex
	_connections      map[net.Conn]struct{}

	_service Service

	_vpnConnectMutex     sync.Mutex
	_disconnectRequested bool

	_connectRequestsMutex   sync.Mutex
	_connectRequests        int
	_connectRequestLastTime time.Time

	// keep info about last VPN state
	_lastVPNState vpn.StateInfo
}

// Stop - stop communication
func (p *Protocol) Stop() {
	log.Info("Stopping")

	listener := p._connListener
	if listener != nil {
		// do not accept new incoming connections
		listener.Close()
	}

	// Notifying clients "service is going to stop" (client application (UI) will close)
	// Closing and erasing all clients connections
	// (do it only if stopping was requested by Stop() )
	p.notifyClientsDaemonExiting()
}

// Start - starts TCP interface to communicate with IVPN application (server to listen incoming connections)
func (p *Protocol) Start(secret uint64, startedOnPort chan<- int, service Service) error {
	if p._service != nil {
		return errors.New("unable to start protocol communication. It is already initialized")
	}
	p._service = service
	p._secret = secret

	defer func() {
		log.Info("Protocol stopped")

		// Disconnect VPN (if connected)
		p._service.Disconnect()
	}()

	addr := fmt.Sprintf("127.0.0.1:0")
	// Initializing listener
	tcpAddr, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve TCP address: %w", err)
	}

	// start listener
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("failed to start TCP listener: %w", err)
	}

	// save listener to a protocol field (to be able to stop it)
	p._connListener = listener

	// get port opened by listener
	openedPortStr := strings.Split(listener.Addr().String(), ":")[1]
	openedPort, err := strconv.Atoi(openedPortStr)
	if err != nil {
		return fmt.Errorf("failed to convert port string to int: %w", err)
	}
	startedOnPort <- openedPort

	log.Info(fmt.Sprintf("IVPN service started: %d [...%s]", openedPort, fmt.Sprintf("%016x", secret)[12:]))
	defer func() {
		listener.Close()
		log.Info("IVPN service stopped")
	}()

	// infinite loop of processing IVPN client connection
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error("Server: failed to accept incoming connection:", err)
			return fmt.Errorf("(server) failed to accept incoming connection: %w", err)
		}
		go p.processClient(conn)
	}
}

// isNewConnectionAllowed is checking who is owner (PID & binary path) of this port
// If connecting binary is not in the list of allowed clients
// OR connected binary has wrong properties (owner, access rights, location) - return error
func (p *Protocol) isNewConnectionAllowed(clientRemoteAddrTCP *net.TCPAddr) error {
	pid, err := process.GetPortOwnerPID(clientRemoteAddrTCP.Port)
	if err != nil {
		return fmt.Errorf("unable to check connected TCP port owner: %w", err)
	}
	binPath, err := process.GetBinaryPathByPID(pid)
	if err != nil {
		return fmt.Errorf("unable to check connected TCP port owner binary %w", err)
	}
	if len(binPath) == 0 {
		return fmt.Errorf("unable to check connected TCP port owner binary")
	}
	log.Info(fmt.Sprintf("Connected binary (%v): '%s'", clientRemoteAddrTCP, binPath))

	// check is connected binary is allowed to connect
	isClientAllowed := false
	allowedClients := platform.AllowedClients()
	if len(allowedClients) == 0 {
		isClientAllowed = true // if 'allowedClients' not defined - skip checking connected client path
	} else {
		for _, allowedBinPath := range allowedClients {
			if strings.ToLower(allowedBinPath) == strings.ToLower(binPath) {
				isClientAllowed = true
				break
			}
		}
	}
	if isClientAllowed != true {
		return fmt.Errorf("not known client binary (%s)", binPath)
	}
	// check is connected binary has correct properties (owner, access rights, location)
	if err := filerights.CheckFileAccessRightsExecutable(binPath); err != nil {
		return fmt.Errorf("unaccepted client binary properties: %w", err)
	}
	return nil
}

func (p *Protocol) processClient(conn net.Conn) {
	// keepAlone informs daemon\service to do nothing when client disconnects
	// 		false (default) - VPN disconnects when client disconnects from a daemon
	// 		true - do nothing when client disconnects from a daemon (if VPN is connected - do not disconnect)
	keepAlone := false
	// The first request from a client should be 'Hello' request with correct secret
	// In case of wrong secret - the daemon drops connection
	isAuthenticated := false

	clientRemoteAddr := conn.RemoteAddr()
	log.Info("Client connected: ", clientRemoteAddr)

	stopChannel := make(chan struct{}, 1)
	defer func() {
		// notify routines to stop
		close(stopChannel)

		if r := recover(); r != nil {
			log.Error("PANIC during client communication!: ", r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
		}

		p.clientDisconnected(conn)
		log.Info("Client disconnected: ", conn.RemoteAddr())

		if isAuthenticated && keepAlone == false {
			stopService, err := p._service.OnControlConnectionClosed()
			if err != nil {
				log.Error(err)
			}

			// Disconnect VPN (if connected)
			if err := p._service.Disconnect(); err != nil {
				log.Error(err)
			}

			if stopService {
				log.Info("Stopping due to configuration: Stop IVPN Agent when application is not running")
				p.Stop()
			}
		} else {
			if p._service.IsPaused() && p.clientsConnectedCount() == 0 {
				log.Info("Connection is in paused state and no active clients available. Disconnecting ...")
				if err := p._service.Disconnect(); err != nil {
					log.Error(err)
				}
			} else {
				log.Info("Current state not changing [KeepDaemonAlone=true]")
			}
		}
	}()

	// check is connection allowed fir this client
	clientRemoteAddrTCP, err := net.ResolveTCPAddr("tcp4", clientRemoteAddr.String())
	if err != nil {
		p.sendResponse(conn, &types.ErrorResp{ErrorMessage: fmt.Sprintf("Refusing connection: Unable to determine remote TCP4 address of connected client: %s", err.Error())}, 0)
		return
	}
	if err := p.isNewConnectionAllowed(clientRemoteAddrTCP); err != nil {
		p.sendError(conn, fmt.Sprintf("Refusing connection: %s", err.Error()), 0)
		return
	}

	// service changes notifier
	startChangesNotifier := func() {
		if r := recover(); r != nil {
			log.Error("PANIC in client notifier!: ", r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
		}

		for {
			select {
			case <-p._service.ServersUpdateNotifierChannel():
				// servers update notifier
				serv, _ := p._service.ServersList()
				p.sendResponse(conn, &types.ServerListResp{VpnServers: *serv}, 0)
			case <-stopChannel:
				return // stop loop
			}
		}
	}

	reader := bufio.NewReader(conn)
	// run loop forever (or until ctrl-c)
	for {
		// will listen for message to process ending in newline (\n)
		message, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Error("Error receiving data from client: ", err)
			}
			break
		}

		// CONNECTION AUTHENTICATION: First request should be 'Hello' with correct authentication secret
		if isAuthenticated == false {
			messageData := []byte(message)

			cmd, err := types.GetCommandBase(messageData)
			if err != nil {
				log.Error(fmt.Sprintf("%sFailed to parse initialization request:", p.connLogID(conn)), err)
				return
			}
			// ensure if client use correct secret
			if cmd.Command != "Hello" {
				logger.Error(fmt.Sprintf("%sConnection not authenticated. Closing.", p.connLogID(conn)))
				return
			}
			// parsing 'Hello' request
			var hello types.Hello
			if err := json.Unmarshal(messageData, &hello); err != nil {
				p.sendErrorResponse(conn, cmd, fmt.Errorf("connection authentication error: %w", err))
				return
			}
			if hello.Secret != p._secret {
				log.Warning(fmt.Errorf("Refusing connection: secret verification error"))
				p.sendErrorResponse(conn, cmd, fmt.Errorf("secret verification error"))
				return
			}

			// AUTHENTICATED
			keepAlone = hello.KeepDaemonAlone
			isAuthenticated = true
			p.clientConnected(conn)
			go startChangesNotifier()
		}

		// Processing requests from client (in separate routine)
		go p.processRequest(conn, message)
	}
}

func (p *Protocol) processRequest(conn net.Conn, message string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error(fmt.Sprintf("%sPANIC during processing request!: ", p.connLogID(conn)), r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
			log.Info(fmt.Sprintf("%sClosing connection and recovering state", p.connLogID(conn)))
			conn.Close()
		}
	}()

	messageData := []byte(message)

	reqCmd, err := types.GetCommandBase(messageData)
	if err != nil {
		log.Error(fmt.Sprintf("%sFailed to parse request:", p.connLogID(conn)), err)
		return
	}

	log.Info("[<--] ", p.connLogID(conn), reqCmd.Command)

	sendState := func(reqIdx int, isOnlyIfConnected bool) {
		vpnState := p._lastVPNState
		if vpnState.State == vpn.CONNECTED {
			p.sendResponse(conn, &types.ConnectedResp{
				TimeSecFrom1970: vpnState.Time,
				ClientIP:        vpnState.ClientIP.String(),
				ServerIP:        vpnState.ServerIP.String(),
				VpnType:         vpnState.VpnType,
				ExitServerID:    vpnState.ExitServerID,
				ManualDNS:       dns.GetLastManualDNS(),
				IsCanPause:      vpnState.IsCanPause},
				reqIdx)
		} else if isOnlyIfConnected == false {
			if vpnState.State == vpn.DISCONNECTED {
				p.sendResponse(conn, &types.DisconnectedResp{Failure: false, Reason: 0, ReasonDescription: ""}, reqIdx)
			} else {
				p.sendResponse(conn, &types.VpnStateResp{StateVal: vpnState.State, State: vpnState.State.String()}, reqIdx)
			}
		}
	}

	switch reqCmd.Command {
	case "Hello":
		var req types.Hello

		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		}

		log.Info(fmt.Sprintf("%sConnected client version: '%s' [set KeepDaemonAlone = %t]", p.connLogID(conn), req.Version, req.KeepDaemonAlone))

		// send back Hello message with account session info
		p.sendResponse(conn, p.createHelloResponse(), req.Idx)

		if req.GetServersList == true {
			serv, _ := p._service.ServersList()
			if serv != nil {
				p.sendResponse(conn, &types.ServerListResp{VpnServers: *serv}, req.Idx)
			}
		}

		if req.GetStatus == true {
			// send VPN connection  state
			sendState(req.Idx, true)

			// send Firewall state
			if isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast, err := p._service.KillSwitchState(); err == nil {
				p.sendResponse(conn, &types.KillSwitchStatusResp{IsEnabled: isEnabled, IsPersistent: isPersistant, IsAllowLAN: isAllowLAN, IsAllowMulticast: isAllowLanMulticast}, reqCmd.Idx)
			}
		}

		if req.GetConfigParams {
			p.sendResponse(conn,
				&types.ConfigParamsResp{UserDefinedOvpnFile: platform.OpenvpnUserParamsFile()},
				reqCmd.Idx)
		}

		// sending WIFI info
		p.OnWiFiChanged(p._service.GetWiFiCurrentState())
		break

	case "GetVPNState":
		// send VPN connection  state
		sendState(reqCmd.Idx, false)
		break

	case "GetServers":
		serv, err := p._service.ServersList()
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		if serv == nil {
			p.sendErrorResponse(conn, reqCmd, fmt.Errorf("failed to get servers info"))
			break
		}
		p.sendResponse(conn, &types.ServerListResp{VpnServers: *serv}, reqCmd.Idx)
		break

	case "PingServers":
		var req types.PingServers
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		retMap, err := p._service.PingServers(req.RetryCount, req.TimeOutMs)
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		var results []types.PingResultType
		for k, v := range retMap {
			results = append(results, types.PingResultType{Host: k, Ping: v})
		}

		p.sendResponse(conn, &types.PingServersResp{PingResults: results}, req.Idx)
		break

	case "APIRequest":
		var req types.APIRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		data, err := p._service.APIRequest(req.APIPath)
		if err != nil {
			p.sendResponse(conn, &types.APIResponse{APIPath: req.APIPath, Error: err.Error()}, req.Idx)
			break
		}
		p.sendResponse(conn, &types.APIResponse{APIPath: req.APIPath, ResponseData: string(data)}, req.Idx)
		break

	case "WiFiAvailableNetworks":
		networks := p._service.GetWiFiAvailableNetworks()
		nets := make([]types.WiFiNetworkInfo, 0, len(networks))
		for _, ssid := range networks {
			nets = append(nets, types.WiFiNetworkInfo{SSID: ssid})
		}

		p.notifyClients(&types.WiFiAvailableNetworksResp{
			Networks: nets})
		break

	case "KillSwitchGetStatus":
		if isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast, err := p._service.KillSwitchState(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		} else {
			p.sendResponse(conn, &types.KillSwitchStatusResp{IsEnabled: isEnabled, IsPersistent: isPersistant, IsAllowLAN: isAllowLAN, IsAllowMulticast: isAllowLanMulticast}, reqCmd.Idx)
		}
		break

	case "KillSwitchSetEnabled":
		var req types.KillSwitchSetEnabled
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		if err := p._service.SetKillSwitchState(req.IsEnabled); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		// send the response to the requestor
		p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		// all clients will be notified in case of successfull change by OnKillSwitchStateChanged() handler

		break

	case "KillSwitchSetAllowLANMulticast":
		var req types.KillSwitchSetAllowLANMulticast
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p._service.SetKillSwitchAllowLANMulticast(req.AllowLANMulticast)
		if req.Synchronously {
			p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		}
		// all clients will be notified in case of successfull change by OnKillSwitchStateChanged() handler
		break

	case "KillSwitchSetAllowLAN":
		var req types.KillSwitchSetAllowLAN
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p._service.SetKillSwitchAllowLAN(req.AllowLAN)
		if req.Synchronously {
			p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		}
		// all clients will be notified in case of successfull change by OnKillSwitchStateChanged() handler
		break

	case "KillSwitchSetIsPersistent":
		var req types.KillSwitchSetIsPersistent
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		if err := p._service.SetKillSwitchIsPersistent(req.IsPersistent); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		// send the response to the requestor
		p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		// all clients will be notified in case of successfull change by OnKillSwitchStateChanged() handler
		break

	// TODO: can be fully replaced by 'KillSwitchGetStatus'
	case "KillSwitchGetIsPestistent":
		isPersistant := p._service.Preferences().IsFwPersistant
		p.sendResponse(conn, &types.KillSwitchGetIsPestistentResp{IsPersistent: isPersistant}, reqCmd.Idx)
		break

	// TODO: must return response
	// TODO: avoid using raw key as a string
	case "SetPreference":
		var req types.SetPreference
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		if err := p._service.SetPreference(req.Key, req.Value); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		}
		break

	case "GenerateDiagnostics":
		if log, log0, err := logger.GetLogText(1024 * 64); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		} else {
			p.sendResponse(conn, &types.DiagnosticsGeneratedResp{ServiceLog: log, ServiceLog0: log0}, reqCmd.Idx)
		}
		break

	case "SetAlternateDns":
		var req types.SetAlternateDns
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		var err error
		if ip := net.ParseIP(req.DNS); ip == nil || ip.Equal(net.IPv4zero) || ip.Equal(net.IPv4bcast) {
			err = p._service.ResetManualDNS()
		} else {
			err = p._service.SetManualDNS(ip)

			if err != nil {
				// DNS set failed. Trying to reset DNS
				errReset := p._service.ResetManualDNS()
				if errReset != nil {
					log.ErrorTrace(errReset)
				}
			}
		}

		if err != nil {
			log.ErrorTrace(err)
			// send the response to the requestor
			p.sendResponse(conn, &types.SetAlternateDNSResp{IsSuccess: false, ChangedDNS: net.IPv4zero.String()}, req.Idx)
		} else {
			// send the response to the requestor
			p.sendResponse(conn, &types.SetAlternateDNSResp{IsSuccess: true, ChangedDNS: req.DNS}, req.Idx)
			// all clients will be notified in case of successfull change by OnDNSChanged() handler
		}
		break

	case "PauseConnection":
		if err := p._service.Pause(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)
		break

	case "ResumeConnection":
		if err := p._service.Resume(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)
		break

	case "SessionNew":
		var req types.SessionNew
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		// validate AccountID value
		matched, err := regexp.MatchString("^(i-....-....-....)|(ivpn[a-zA-Z0-9]{7,8})$", req.AccountID)
		if err != nil {
			p.sendError(conn, fmt.Sprintf("[daemon] Account ID validation failed: %s", err), reqCmd.Idx)
			break
		}
		if !matched {
			p.sendError(conn, "[daemon] Your account ID has to be in 'i-XXXX-XXXX-XXXX' or 'ivpnXXXXXXXX' format.", reqCmd.Idx)
			break
		}

		var resp types.SessionNewResp
		apiCode, apiErrMsg, accountInfo, rawResponse, err := p._service.SessionNew(req.AccountID, req.ForceLogin, req.CaptchaID, req.Captcha, req.Confirmation2FA)
		if err != nil {
			if apiCode == 0 {
				// if apiCode == 0 - it is not API error. Sending error response
				p.sendErrorResponse(conn, reqCmd, err)
				break
			}
			// sending API error info
			resp = types.SessionNewResp{
				APIStatus:       apiCode,
				APIErrorMessage: apiErrMsg,
				Session:         types.SessionResp{}, // empty session info
				Account:         accountInfo,
				RawResponse:     rawResponse}
		} else {
			// Success. Sending session info
			resp = types.SessionNewResp{
				APIStatus:       apiCode,
				APIErrorMessage: apiErrMsg,
				Session:         types.CreateSessionResp(p._service.Preferences().Session),
				Account:         accountInfo,
				RawResponse:     rawResponse}
		}

		// send response
		p.sendResponse(conn, &resp, reqCmd.Idx)

		// notify all clients about changed session status
		p.notifyClients(p.createHelloResponse())
		break

	case "SessionDelete":
		err := p._service.SessionDelete()
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)

		// notify all clients about changed session status
		p.notifyClients(p.createHelloResponse())
		break

	case "AccountStatus":
		var resp types.AccountStatusResp
		apiCode, apiErrMsg, sessionToken, accountInfo, err := p._service.RequestSessionStatus()
		if err != nil && apiCode == 0 {
			// if apiCode == 0 - it is not API error. Sending error response
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		// Sending session info
		resp = types.AccountStatusResp{
			APIStatus:       apiCode,
			APIErrorMessage: apiErrMsg,
			SessionToken:    sessionToken,
			Account:         accountInfo}

		// send response
		p.sendResponse(conn, &resp, reqCmd.Idx)
		break

	case "WireGuardGenerateNewKeys":
		var req types.WireGuardGenerateNewKeys
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		if err := p._service.WireGuardGenerateKeys(req.OnlyUpdateIfNecessary); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)
		break

	case "WireGuardSetKeysRotationInterval":
		var req types.WireGuardSetKeysRotationInterval
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p._service.WireGuardSetKeysRotationInterval(req.Interval)
		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)
		break

	case "Disconnect":
		p._disconnectRequested = true

		if p._service.Connected() == false {
			p.sendResponse(conn, &types.DisconnectedResp{Reason: types.DisconnectRequested}, reqCmd.Idx)
			break
		}

		if err := p._service.Disconnect(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		}
		break

	case "Connect":
		p._disconnectRequested = false
		requestTime := p.vpnConnectReqCounterIncrease()

		stateChan := make(chan vpn.StateInfo, 1)
		isExitChan := make(chan bool, 1)
		disconnectAuthError := false
		var connectionError error

		// disconnect active connection (if connected)
		if err := p._service.Disconnect(); err != nil {
			log.ErrorTrace(err)
		}

		p._vpnConnectMutex.Lock()
		defer p._vpnConnectMutex.Unlock()

		defer p.vpnConnectReqCounterDecrease()

		// skip this request if new connection request available
		if _, lastRequestTime := p.vpnConnectReqCounter(); requestTime.Equal(lastRequestTime) == false {
			log.Info("Skipping connection request. Newest request received.")
			return
		}

		var waiter sync.WaitGroup

		// do not forget to notify that process was stopped (disconnected)
		defer func() {

			// stop all go-routines related to this connections
			close(isExitChan)

			// Do not send "Disconnected" notification if we are going to establish new connection immediately
			if cnt, _ := p.vpnConnectReqCounter(); cnt == 1 || p._disconnectRequested {
				p._lastVPNState = vpn.NewStateInfo(vpn.DISCONNECTED, "")

				// Sending "Disconnected" only in one place (after VPN process stopped)
				disconnectionReason := types.Unknown
				if disconnectAuthError == true {
					disconnectionReason = types.AuthenticationError
					if connectionError == nil {
						connectionError = fmt.Errorf("authentication failure")
					}
				}
				if p._disconnectRequested {
					// notify clients that disconnection was manually requested by one of connected clients
					// (prevent UI clients trying to reconnect)
					disconnectionReason = types.DisconnectRequested
				}

				errMsg := ""
				if connectionError != nil {
					errMsg = connectionError.Error()
				}
				p.notifyClients(&types.DisconnectedResp{Failure: connectionError != nil, Reason: disconnectionReason, ReasonDescription: errMsg})
			}

			// wait all routines to stop
			waiter.Wait()
		}()

		// forwarding VPN state in separate routine
		waiter.Add(1)
		go func() {
			log.Info("Enter VPN status checker")
			defer func() {
				if r := recover(); r != nil {
					log.Error("VPN status checker panic!")
					if err, ok := r.(error); ok {
						log.ErrorTrace(err)
					}
				}
				log.Info("Exit VPN status checker")
				waiter.Done()
			}()

		state_forward_loop:
			for {
				select {
				case <-isExitChan:
					break state_forward_loop

				case state := <-stateChan:

					select {
					case <-isExitChan:
						// channel closed in defer function (vpn disconnected)
						break state_forward_loop
					default:
					}

					p._lastVPNState = state

					switch state.State {
					case vpn.CONNECTED:
						// Do not send "Connected" notification if we are going to establish new connection immediately
						if cnt, _ := p.vpnConnectReqCounter(); cnt == 1 || p._disconnectRequested {
							p.notifyClients(&types.ConnectedResp{
								TimeSecFrom1970: state.Time,
								ClientIP:        state.ClientIP.String(),
								ServerIP:        state.ServerIP.String(),
								VpnType:         state.VpnType,
								ExitServerID:    state.ExitServerID,
								ManualDNS:       dns.GetLastManualDNS(),
								IsCanPause:      state.IsCanPause})

						} else {
							log.Debug("Skip sending 'Connected' notification. New connection request is awaiting ", cnt)
						}
					case vpn.EXITING:
						disconnectAuthError = state.IsAuthError
					default:
						p.notifyClients(&types.VpnStateResp{StateVal: state.State, State: state.State.String(), StateAdditionalInfo: state.StateAdditionalInfo})
					}
				}
			}
		}()

		// SYNCHRONOUSLY start VPN connection process (wait until it finished)
		if connectionError = p.processConnectRequest(messageData, stateChan); connectionError != nil {
			log.ErrorTrace(connectionError)
		}

		break

	default:
		log.Warning("!!! Unsupported request type !!! ", reqCmd.Command)
		log.Debug("Unsupported request:", message)
		p.sendErrorResponse(conn, reqCmd, fmt.Errorf("unsupported request: '%s'", reqCmd.Command))
	}
}
