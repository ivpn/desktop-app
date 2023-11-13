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

package protocol

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	api_types "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/oshelpers"
	"github.com/ivpn/desktop-app/daemon/protocol/eaa"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	service_types "github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/vpn"
	"github.com/ivpn/desktop-app/daemon/wifiNotifier"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("prtcl")
}

// Service - service interface
type Service interface {
	UnInitialise() error

	OnAuthenticatedClient(t types.ClientTypeEnum)

	// GetDisabledFunctions returns info about functions which are disabled
	// Some functionality can be not accessible
	// It can happen, for example, if some external binaries not installed
	// (e.g. obfsproxy or WireGuard on Linux)
	GetDisabledFunctions() types.DisabledFunctionality

	// ServersList returns servers info
	// (if there is a cached data available - will be returned data from cache)
	ServersList() (*api_types.ServersInfoResponse, error)
	// ServersListForceUpdate returns servers list info.
	// The daemon will make request to update servers from the backend.
	// The cached data will be ignored in this case.
	ServersListForceUpdate() (*api_types.ServersInfoResponse, error)

	PingServers(timeoutMs int, vpnTypePrioritized vpn.Type, skipSecondPhase bool) (map[string]int, error)

	APIRequest(apiAlias string, ipTypeRequired types.RequiredIPProtocol) (responseData []byte, err error)
	DetectAccessiblePorts(portsToTest []api_types.PortInfo) (retPorts []api_types.PortInfo, err error)

	KillSwitchState() (status service_types.KillSwitchStatus, err error)
	SetKillSwitchState(bool) error
	SetKillSwitchIsPersistent(isPersistant bool) error
	SetKillSwitchAllowLANMulticast(isAllowLanMulticast bool) error
	SetKillSwitchAllowLAN(isAllowLan bool) error
	SetKillSwitchAllowAPIServers(isAllowAPIServers bool) error
	SetKillSwitchUserExceptions(exceptions string, ignoreParsingErrors bool) error

	GetConnectionParams() service_types.ConnectionParams
	SetConnectionParams(params service_types.ConnectionParams) error
	SetWiFiSettings(params preferences.WiFiParams) error

	SplitTunnelling_SetConfig(isEnabled, isInversed, isAnyDns, isAllowWhenNoVpn, reset bool) error
	SplitTunnelling_GetStatus() (types.SplitTunnelStatus, error)
	SplitTunnelling_AddApp(exec string) (cmdToExecute string, isAlreadyRunning bool, err error)
	SplitTunnelling_RemoveApp(pid int, exec string) (err error)
	SplitTunnelling_AddedPidInfo(pid int, exec string, cmdToExecute string) error

	GetInstalledApps(extraArgsJSON string) ([]oshelpers.AppInfo, error)
	GetBinaryIcon(binaryPath string) (string, error)

	Preferences() preferences.Preferences
	SetPreference(key types.ServicePreference, val string) (isChanged bool, err error)
	SetUserPreferences(userPrefs preferences.UserPreferences) (err error)
	ResetPreferences() error

	// SetManualDNS update default DNS parameters AND apply new DNS value for current VPN connection
	// If 'antiTracker' is enabled - the 'dnsCfg' will be ignored
	SetManualDNS(dns dns.DnsSettings, antiTracker service_types.AntiTrackerMetadata) (changedDns dns.DnsSettings, retErr error)
	GetManualDNSStatus() dns.DnsSettings
	GetAntiTrackerStatus() service_types.AntiTrackerMetadata

	IsCanConnectMultiHop() error
	Connect(params service_types.ConnectionParams) error
	Disconnect() error
	Connected() bool

	Pause(durationSeconds uint32) error
	Resume() error
	IsPaused() bool
	PausedTill() time.Time

	SessionNew(accountID string, forceLogin bool, captchaID string, captcha string, confirmation2FA string) (
		apiCode int,
		apiErrorMsg string,
		accountInfo preferences.AccountStatus,
		rawResponse string,
		err error)

	SessionDelete(isCanDeleteSessionLocally bool) error
	RequestSessionStatus() (
		apiCode int,
		apiErrorMsg string,
		sessionToken string,
		accountInfo preferences.AccountStatus,
		err error)

	WireGuardGenerateKeys(updateIfNecessary bool) error
	WireGuardSetKeysRotationInterval(interval int64)

	GetWiFiCurrentState() wifiNotifier.WifiInfo
	GetWiFiAvailableNetworks() []string

	GetDiagnosticLogs() (logActive string, logPrevSession string, extraInfo string, err error)
}

// CreateProtocol - Create new protocol object
func CreateProtocol() (*Protocol, error) {
	return &Protocol{
		_connections:     make(map[net.Conn]connectionInfo),
		_eaa:             eaa.Init(platform.ParanoidModeSecretFile()),
		_connRequestChan: make(chan service_types.ConnectionParams, 1),
	}, nil
}

type connectionInfo struct {
	Type            types.ClientTypeEnum // UI or CLI
	IsAuthenticated bool                 // true when connection fully authenticated (secret is OK and EAA check is passed)
}

// Protocol - TCP interface to communicate with IVPN application
type Protocol struct {
	_secret uint64

	// connections listener
	_connListener *net.TCPListener

	_connectionsMutex sync.RWMutex
	_connections      map[net.Conn]connectionInfo

	// Only last connect request will be processed (if there are more then one received in short period of time)
	_connRequestMutex sync.Mutex
	_connRequestChan  chan service_types.ConnectionParams
	_connRequestReady sync.WaitGroup

	_disconnectRequested bool

	_service Service

	// keep info about last VPN state
	_lastVPNState vpn.StateInfo

	_eaa *eaa.Eaa

	_isRunning bool // 'false' when not running OR after Stop() command call

	// Send this error info to a first connected client
	// (in use if no clients connected when the error happened)
	_lastConnectionErrorToNotifyClient string
}

// Stop - stop communication
func (p *Protocol) Stop() {
	log.Info("Stopping ...")

	// Notifying clients "service is going to stop" (client application (UI) will close)
	// Closing and erasing all clients connections
	// (do it only if stopping was requested by Stop() )
	p.notifyClientsDaemonExiting()

	listener := p._connListener
	if listener != nil {
		// keep info that stop command requested
		p._isRunning = false
		// do not accept new incoming connections
		listener.Close()

		// Do not use any send\receive communications with connected clients after listener stopped
	}
}

// Start - starts TCP interface to communicate with IVPN application (server to listen incoming connections)
func (p *Protocol) Start(secret uint64, startedOnPort chan<- int, service Service) error {
	if p._service != nil {
		return errors.New("unable to start protocol communication. It is already initialized")
	}
	p._service = service
	p._secret = secret

	p._isRunning = true
	defer func() {
		p._isRunning = false
		log.Info("Protocol stopped")

		// Disconnect VPN (if connected)
		p._service.UnInitialise()
	}()

	addr := "127.0.0.1:0"
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
		log.Info("Listener closed")
	}()

	// Start processing of new connection requests
	// (connection requests collecting in to chain and processing in order they were received.
	// See also "RegisterConnectionRequest()" for details)
	go p.processConnectionRequests()

	// infinite loop of processing IVPN client connection
	for {
		conn, err := listener.Accept()
		if err != nil {
			if !p._isRunning {
				return nil // it is expected to get error here (we are requested protocol to stop): "use of closed network connection"
			}
			log.Error("Server: failed to accept incoming connection:", err)
			return fmt.Errorf("(server) failed to accept incoming connection: %w", err)
		}
		go p.processClient(conn)
	}
}

func (p *Protocol) processClient(conn net.Conn) {
	// The first request from a client should be 'Hello' request with correct secret
	// In case of wrong secret - the daemon drops connection
	isAuthenticated := false

	clientRemoteAddr := conn.RemoteAddr()
	log.Info("Client connected: ", clientRemoteAddr)

	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC during client communication!: ", r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
		}

		disconnectedClientInfo := p.clientDisconnected(conn)
		log.Info("Client disconnected: ", conn.RemoteAddr())

		// if VPN Paused:
		//	- if only UI client was connected and now it disconnected - disconnect VPN
		//	- if only CLI client was connected - do nothing (keep VPN paused)
		if p._service.IsPaused() &&
			p.clientsConnectedCount() == 0 &&
			disconnectedClientInfo != nil &&
			disconnectedClientInfo.Type == types.ClientUi &&
			disconnectedClientInfo.IsAuthenticated {
			log.Info("Connection is in paused state and no active clients available. Disconnecting ...")
			if err := p._service.Disconnect(); err != nil {
				log.Error(err)
			}
		} else {
			log.Info("Current state not changing")
		}

	}()

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
		if !isAuthenticated {
			messageData := []byte(message)

			cmd, err := types.GetRequestBase(messageData)
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
				log.Warning(fmt.Errorf("refusing connection: secret verification error"))
				p.sendErrorResponse(conn, cmd, fmt.Errorf("secret verification error"))
				return
			}

			// AUTHENTICATED
			isAuthenticated = true
			p.clientConnected(conn, hello.ClientType)
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

	reqCmd, err := types.GetRequestBase(messageData)
	if err != nil {
		log.Error(fmt.Sprintf("%sFailed to parse request:", p.connLogID(conn)), err)
		return
	}

	// Log the request info
	cmdExtraInfo := ""
	if reqCmd.Command == "APIRequest" {
		var req types.APIRequest
		if err := json.Unmarshal(messageData, &req); err == nil {
			cmdExtraInfo = " " + req.APIPath
			if req.IPProtocolRequired == types.IPv4 {
				cmdExtraInfo += " (IPv4)"
			} else if req.IPProtocolRequired == types.IPv6 {
				cmdExtraInfo += " (IPv6)"
			}
		}
	}
	log.Info("[<--] ", p.connLogID(conn), reqCmd.Command, fmt.Sprintf(" [%d]%s", reqCmd.Idx, cmdExtraInfo))

	isDoSkipParanoidMode := func(commandName string) bool {

		switch commandName {
		case "Hello",
			"GetVPNState",
			"GetServers",
			"PingServers",
			"APIRequest",
			"WiFiAvailableNetworks",
			"KillSwitchGetStatus",
			"SplitTunnelGetStatus",
			"GetDnsPredefinedConfigs",
			"AccountStatus":
			return true
		}

		return false
	}

	sendState := func(reqIdx int, isOnlyIfConnected bool) {
		vpnState := p._lastVPNState
		if vpnState.State == vpn.CONNECTED {
			p.sendResponse(conn, p.createConnectedResponse(vpnState), reqIdx)
		} else if !isOnlyIfConnected {
			if vpnState.State == vpn.DISCONNECTED {
				p.sendResponse(conn, &types.DisconnectedResp{IsStateInfo: true, Failure: false, Reason: 0, ReasonDescription: ""}, reqIdx)
			} else {
				p.sendResponse(conn, &types.VpnStateResp{StateVal: vpnState.State, State: vpnState.State.String()}, reqIdx)
			}
		}
	}

	if !p._eaa.IsEnabled() {
		// EAA is disabled. So, mark connection as authenticated
		p.clientSetAuthenticated(conn)
	} else {
		if !isDoSkipParanoidMode(reqCmd.Command) {
			isOK, err := p._eaa.CheckSecret(reqCmd.ProtocolSecret)
			if !isOK {
				// ParanoidMode: wrong password
				errorResp := types.ErrorResp{
					ErrorType:    types.ErrorParanoidModePasswordError,
					ErrorTitle:   "Enhanced App Authentication",
					ErrorMessage: "The password is incorrect. Please try again."}

				if err != nil && len(errorResp.Error()) > 0 {
					errorResp.ErrorMessage = err.Error()
				}

				p.sendResponse(conn,
					&errorResp,
					reqCmd.Idx)

				log.Info(fmt.Sprintf("      [%d] %sRequest error '%s': %s", reqCmd.Idx, p.connLogID(conn), reqCmd.Command, errorResp))

				// send current connection state
				if reqCmd.Command == "Connect" || reqCmd.Command == "Disconnect" {
					sendState(reqCmd.Idx, false)
				}

				return
			}

			// We are here. So we are authenticated.
			// mark connection as authenticated
			p.clientSetAuthenticated(conn)
		}
	}

	switch reqCmd.Command {
	case "EmptyReq":
		// test request (e.g. checking PM password)
		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)

	case "Hello":
		var req types.Hello

		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		}

		log.Info(fmt.Sprintf("%sConnected client version: '%s'", p.connLogID(conn), req.Version))

		// send back Hello message with account session info
		helloResponse := p.createHelloResponse()
		p.sendResponse(conn, helloResponse, req.Idx)
		if req.SendResponseToAllClients {
			p.notifyClients(helloResponse)
		}

		if req.GetServersList {
			serv, _ := p._service.ServersList()
			if serv != nil {
				p.sendResponse(conn, &types.ServerListResp{VpnServers: *serv}, req.Idx)
			}
		}

		if req.GetStatus {
			// send VPN connection  state
			sendState(req.Idx, false)

			// send Firewall state
			if status, err := p._service.KillSwitchState(); err == nil {
				p.sendResponse(conn,
					&types.KillSwitchStatusResp{KillSwitchStatus: status}, reqCmd.Idx)
			}
		}

		if req.GetSplitTunnelStatus {
			// sending split-tunnelling configuration
			p.OnSplitTunnelStatusChanged()
		}

		if req.GetWiFiCurrentState {
			// sending WIFI info
			p.OnWiFiChanged(p._service.GetWiFiCurrentState())
		}

	case "ParanoidModeSetPasswordReq":
		var req types.ParanoidModeSetPasswordReq
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		if err := p._eaa.SetSecret(req.ProtocolSecret, req.NewSecret); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		} else {
			// send 'success' response to the requestor
			p.notifyClients(p.createHelloResponse())
			p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		}

	case "GetVPNState":
		// send VPN connection  state
		sendState(reqCmd.Idx, false)

	case "GetServers":
		var req types.GetServers
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		sendResponseFunc := func(retServ *api_types.ServersInfoResponse, retErr error) {
			if retErr != nil {
				p.sendErrorResponse(conn, reqCmd, retErr)
				return
			}
			if retServ == nil {
				p.sendErrorResponse(conn, reqCmd, fmt.Errorf("failed to get servers info"))
				return
			}
			p.sendResponse(conn, &types.ServerListResp{VpnServers: *retServ}, reqCmd.Idx)
		}

		if req.RequestServersUpdate {
			// Force to update servers from the backend (RequestServersUpdate ==  true)
			// Send response only after request to backend finished (cached data is ignored)
			sendResponseFunc(p._service.ServersListForceUpdate())
			break
		}

		// return servers info (cashed data can be used, if exists)
		sendResponseFunc(p._service.ServersList())

	case "PingServers":
		var req types.PingServers
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		vpnType := vpn.Type(-1)
		if req.VpnTypePrioritization {
			vpnType = req.VpnTypePrioritized
		}
		retMap, err := p._service.PingServers(req.TimeOutMs, vpnType, req.SkipSecondPhase)
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		var results []types.PingResultType
		for k, v := range retMap {
			results = append(results, types.PingResultType{Host: k, Ping: v})
		}

		p.sendResponse(conn, &types.PingServersResp{PingResults: results}, req.Idx)

	case "APIRequest":
		var req types.APIRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		data, err := p._service.APIRequest(req.APIPath, req.IPProtocolRequired)
		if err != nil {
			p.sendResponse(conn, &types.APIResponse{APIPath: req.APIPath, Error: err.Error()}, req.Idx)
			break
		}
		p.sendResponse(conn, &types.APIResponse{APIPath: req.APIPath, ResponseData: string(data)}, req.Idx)

	case "CheckAccessiblePorts":
		var req types.CheckAccessiblePorts
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		accessiblePorts, err := p._service.DetectAccessiblePorts(req.PortsToTest)
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		p.sendResponse(conn, &types.CheckAccessiblePortsResponse{Ports: accessiblePorts}, req.Idx)

	case "WiFiAvailableNetworks":
		networks := p._service.GetWiFiAvailableNetworks()
		nets := make([]types.WiFiNetworkInfo, 0, len(networks))
		for _, ssid := range networks {
			nets = append(nets, types.WiFiNetworkInfo{SSID: ssid})
		}

		p.notifyClients(&types.WiFiAvailableNetworksResp{Networks: nets})
		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)

	case "KillSwitchGetStatus":
		if status, err := p._service.KillSwitchState(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		} else {
			p.sendResponse(conn,
				&types.KillSwitchStatusResp{KillSwitchStatus: status}, reqCmd.Idx)
		}

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
		// all clients will be notified in case of successful change by OnKillSwitchStateChanged() handler

	case "KillSwitchSetAllowLANMulticast":
		var req types.KillSwitchSetAllowLANMulticast
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p._service.SetKillSwitchAllowLANMulticast(req.AllowLANMulticast)
		p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		// all clients will be notified in case of successful change by OnKillSwitchStateChanged() handler

	case "KillSwitchSetAllowLAN":
		var req types.KillSwitchSetAllowLAN
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p._service.SetKillSwitchAllowLAN(req.AllowLAN)
		p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		// all clients will be notified in case of successful change by OnKillSwitchStateChanged() handler

	case "KillSwitchSetUserExceptions":
		var req types.KillSwitchSetUserExceptions
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		err := p._service.SetKillSwitchUserExceptions(strings.TrimSpace(req.UserExceptions), !req.FailOnParsingError)
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		} else {
			p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		}
		// all clients will be notified in case of successful change by OnKillSwitchStateChanged() handler

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
		// all clients will be notified in case of successful change by OnKillSwitchStateChanged() handler

	case "KillSwitchSetAllowApiServers":
		var req types.KillSwitchSetAllowApiServers
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		if err := p._service.SetKillSwitchAllowAPIServers(req.IsAllowApiServers); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		// send the response to the requestor
		p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		// all clients will be notified in case of successful change by OnKillSwitchStateChanged() handler

	// TODO: avoid using raw key as a string
	// NOTE: please, use 'SetUserPreferences' for future extensions
	case "SetPreference":
		var req types.SetPreference
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		if isChanged, err := p._service.SetPreference(types.ServicePreference(req.Key), req.Value); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		} else {
			//  notify all connected clients about changed preferences
			if isChanged {
				p.notifyClients(p.createSettingsResponse())
			}

			// notify 'success'
			p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		}

	case "SetUserPreferences":
		func() {
			defer func() {
				//  notify all connected clients about changed (or not changed!) preferences
				p.notifyClients(p.createSettingsResponse())
			}()

			var req types.SetUserPreferences
			if err := json.Unmarshal(messageData, &req); err != nil {
				p.sendErrorResponse(conn, reqCmd, err)
				return
			}

			if err := p._service.SetUserPreferences(req.UserPrefs); err != nil {
				p.sendErrorResponse(conn, reqCmd, err)
				return
			}

			// notify 'success'
			p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		}()

	case "SplitTunnelGetStatus":
		status, err := p._service.SplitTunnelling_GetStatus()
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		p.sendResponse(conn, &status, reqCmd.Idx)

	case "SplitTunnelSetConfig":
		var req types.SplitTunnelSetConfig
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		if err := p._service.SplitTunnelling_SetConfig(req.IsEnabled, req.IsInversed, req.IsAnyDns, req.IsAllowWhenNoVpn, req.Reset); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		// notify 'success'
		p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		// all clients will be notified about configuration change by service in OnSplitTunnelStatusChanged() handler

	case "SplitTunnelAddApp":
		var req types.SplitTunnelAddApp
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		// Description of Split Tunneling commands sequence to run the application:
		//	[client]					[daemon]
		//	SplitTunnelAddApp		->
		//							<-	windows:	types.EmptyResp (success)
		//							<-	linux:		types.SplitTunnelAddAppCmdResp (some operations required on client side)
		//	<windows: done>
		// 	<execute shell command: types.SplitTunnelAddAppCmdResp.CmdToExecute and get PID>
		//  SplitTunnelAddedPidInfo	->
		// 							<-	types.EmptyResp (success)
		cmdToExecute, isAlreadyRunning, err := p._service.SplitTunnelling_AddApp(req.Exec)
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		if len(cmdToExecute) <= 0 {
			p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)
			return
		}

		isRunningWarningMes := ""
		if isAlreadyRunning {
			isRunningWarningMes = "It appears the application is already running.\nSome applications must be closed before launching them in the Split Tunneling environment or they may not be excluded from the VPN tunnel."
		}
		p.sendResponse(conn,
			&types.SplitTunnelAddAppCmdResp{
				Exec:                    req.Exec,
				CmdToExecute:            cmdToExecute,
				IsAlreadyRunning:        isAlreadyRunning,
				IsAlreadyRunningMessage: isRunningWarningMes},
			reqCmd.Idx)
		// all clients will be notified about configuration change by service in OnSplitTunnelStatusChanged() handler

	case "SplitTunnelRemoveApp":
		var req types.SplitTunnelRemoveApp
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		if err := p._service.SplitTunnelling_RemoveApp(req.Pid, req.Exec); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)
		// all clients will be notified about configuration change by service in OnSplitTunnelStatusChanged() handler

	case "SplitTunnelAddedPidInfo":
		var req types.SplitTunnelAddedPidInfo
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		if err := p._service.SplitTunnelling_AddedPidInfo(req.Pid, req.Exec, req.CmdToExecute); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)

	case "GenerateDiagnostics":
		if log, log0, extraInfo, err := p._service.GetDiagnosticLogs(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		} else {
			p.sendResponse(conn, &types.DiagnosticsGeneratedResp{Log1_Active: log, Log0_Old: log0, ExtraInfo: extraInfo}, reqCmd.Idx)
		}

	case "SetAlternateDns":
		{
			var req types.SetAlternateDns
			if err := json.Unmarshal(messageData, &req); err != nil {
				p.sendErrorResponse(conn, reqCmd, err)
				break
			}

			// wrapper around strings.Fields(). Returns only first field or empty string.
			getSingleField := func(s string) string {
				fields := strings.Fields(s)
				if len(fields) <= 0 {
					return ""
				}
				return fields[0]
			}
			req.Dns.DnsHost = getSingleField(req.Dns.DnsHost)
			req.Dns.DohTemplate = getSingleField(req.Dns.DohTemplate)

			_, err := p._service.SetManualDNS(req.Dns, req.AntiTracker)
			if err != nil {
				log.ErrorTrace(err)
				p.sendErrorResponse(conn, reqCmd, err)
			} else {
				p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx) // notify: request processed
			}
			// notify current DNS status
			p.notifyClients(&types.SetAlternateDNSResp{Dns: types.DnsStatus{Dns: p._service.GetManualDNSStatus(), AntiTrackerStatus: p._service.GetAntiTrackerStatus()}})
		}
	case "GetDnsPredefinedConfigs":
		cfgs, err := dns.GetPredefinedDnsConfigurations()
		if err != nil {
			log.ErrorTrace(err)
			p.sendErrorResponse(conn, reqCmd, err)
		} else {
			p.sendResponse(conn, &types.DnsPredefinedConfigsResp{DnsConfigs: cfgs}, reqCmd.Idx)
		}

	case "PauseConnection":
		var req types.PauseConnection
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		if err := p._service.Pause(req.Duration); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p.sendResponse(conn, p.createConnectedResponse(p._lastVPNState), reqCmd.Idx)
		// all clients will be notified by service in OnVpnPauseChanged() handler

	case "ResumeConnection":
		if err := p._service.Resume(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)
		// all clients will be notified by service in OnVpnPauseChanged() handler

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

	case "SessionDelete":
		var req types.SessionDelete
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		err := p._service.SessionDelete(req.IsCanDeleteSessionLocally)
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		// It is important to ensure FW is disabled after SessionDelete() call.
		// (because the SessionDelete->Disconnect->Resume restores original FW state which were before pause)
		if req.NeedToDisableFirewall {
			p._service.SetKillSwitchIsPersistent(false)
			p._service.SetKillSwitchState(false)
		}

		if req.NeedToResetSettings {
			// disable paranoid mode
			p._eaa.ForceDisable()

			oldPrefs := p._service.Preferences()

			// Reset settings only after SessionDelete() to correctly logout on the backed
			p._service.ResetPreferences()
			prefs := p._service.Preferences()

			// restore active persistant Firewall state
			if oldPrefs.IsFwPersistant != prefs.IsFwPersistant {
				p._service.SetKillSwitchIsPersistent(oldPrefs.IsFwPersistant)
			}

			// set AllowLan and exceptions according to default values
			p._service.SetKillSwitchAllowLAN(prefs.IsFwAllowLAN)
			p._service.SetKillSwitchAllowLANMulticast(prefs.IsFwAllowLANMulticast)
			p._service.SetKillSwitchUserExceptions(prefs.FwUserExceptions, true)
		}

		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)

		// notify all clients about changed session status
		p.notifyClients(p.createHelloResponse())

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

	case "WireGuardSetKeysRotationInterval":
		var req types.WireGuardSetKeysRotationInterval
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p._service.WireGuardSetKeysRotationInterval(req.Interval)
		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)

	case "GetAppIcon":
		var req types.GetAppIcon
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		base64Png, err := p._service.GetBinaryIcon(req.AppBinaryPath)
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		p.sendResponse(conn, &types.AppIconResp{AppBinaryPath: req.AppBinaryPath, AppIcon: base64Png}, reqCmd.Idx)

	case "GetInstalledApps":
		var req types.GetInstalledApps
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		apps, err := p._service.GetInstalledApps(req.ExtraArgsJSON)
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		p.sendResponse(conn, &types.InstalledAppsResp{Apps: apps}, reqCmd.Idx)

	case "WiFiCurrentNetwork":
		// sending WIFI info
		wifi := p._service.GetWiFiCurrentState()
		p.sendResponse(conn, &types.WiFiCurrentNetworkResp{
			SSID:              wifi.SSID,
			IsInsecureNetwork: wifi.IsInsecure}, reqCmd.Idx)

	case "WiFiSettings":
		var r types.WiFiSettings
		if err := json.Unmarshal(messageData, &r); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			return
		}
		if err := p._service.SetWiFiSettings(r.Params); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			return
		}
		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)

		// notify all clients about changed wifi settings
		p.notifyClients(p.createHelloResponse())

	case "Disconnect":
		p._disconnectRequested = true
		p._lastConnectionErrorToNotifyClient = ""

		if !p._service.Connected() {
			p.sendResponse(conn, &types.DisconnectedResp{Reason: types.DisconnectRequested}, reqCmd.Idx)
			// INFO: _service.Connected() is based on a simple check (s._vpn != nil). So there is still a chance
			// that the connection-retry loop is running and we just caught a moment while s._vpn is temporarily nil.
			// Therefore, we continue to ensure that Disconnect() is called.
		}

		if err := p._service.Disconnect(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		}

	case "ConnectSettingsGet":
		p.sendResponse(conn, &types.ConnectSettings{Params: p._service.GetConnectionParams()}, reqCmd.Idx)

	case "ConnectSettings":
		// Similar data to 'Connect' request but this command not start the connection.
		// UI client have to notify daemon about changes in connection settings.
		// It is required for automatic connection on daemon's side (e.g. 'Auto-connect on Launch' or 'Trusted WiFi' functionality)

		// parse request
		var connectionSettings types.ConnectSettings
		if err := json.Unmarshal(messageData, &connectionSettings); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			return
		}

		if err := p._service.SetConnectionParams(connectionSettings.Params); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			return
		}

		// send request confirmation to client
		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)

	case "Connect":
		// parse request
		var connectRequest types.Connect
		if err := json.Unmarshal(messageData, &connectRequest); err != nil {
			p.sendErrorResponse(conn, reqCmd, fmt.Errorf("failed to unmarshal json 'Connect' request: %w", err))
			return
		}

		// Save last received connection request. It will be processed in separate routine 'processConnectionRequests()' which is already running
		p.RegisterConnectionRequest(connectRequest.Params)

		// send request confirmation to client
		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)

	default:
		log.Warning("!!! Unsupported request type !!! ", reqCmd.Command)
		log.Debug("Unsupported request:", message)
		p.sendErrorResponse(conn, reqCmd, fmt.Errorf("unsupported request: '%s'", reqCmd.Command))
	}
}

// RegisterConnectionRequest - Register new connection request.
// If there is more than one connection request available - all requests will be ignored except the last one
// Call can be also initiated outside by service (e.g. "trusted-wifi" or "auto-connect on launch" functionality)
func (p *Protocol) RegisterConnectionRequest(r service_types.ConnectionParams) error {
	p._disconnectRequested = false

	// New connection request would not start processing until p._connRequestReady.Done()
	p._connRequestReady.Add(1)
	// At the end: allow processing connection request which was added
	defer p._connRequestReady.Done()

	// synchronized block: only one connection request allowed. Remove previous request (if exists)
	func() {
		p._connRequestMutex.Lock()
		defer p._connRequestMutex.Unlock()
		// remove previous unprocessed requests (if they are)
		select {
		case <-p._connRequestChan:
			log.Info("Skipping previous connection request. Newest request received!")
		default:
		}

		// Add request to chain (it will be processed in 'processConnectionRequests()' routine)
		// Note: new connection request would not start processing until p._connRequestReady.Done()
		p._connRequestChan <- r
	}()

	// Disconnect active connection (if connected).
	// "Disconnected" notification will not be sent to the clients in this case (because new connection request is pending).
	// It is important to call it after new connection request registered
	// Note: new connection will no start untill exit this function (see 'p._connRequestReady.Done()')
	if p._service != nil {
		if err := p._service.Disconnect(); err != nil {
			log.ErrorTrace(err)
		}
	}
	return nil
}

func (p *Protocol) processConnectionRequests() {
	log.Info("Connection requests processor started")
	defer log.Info("Connection requests processor stopped")

	for {
		if !p._isRunning {
			break
		}

		connectRequest := <-p._connRequestChan
		p._connRequestReady.Wait() // wait processing connection request until everything is ready

		// processing each connection request is wrapped into function in order to call 'defer' sections properly
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error(fmt.Errorf("PANIC during processing connection request: %v", r))
					if err, ok := r.(error); ok {
						log.ErrorTrace(err)
					}
				}
			}()

			saveLastError := func(e error) {
				// If no any clients connected - error notification will not be passed to user
				// Trerefore we keep this error an pass it to the first connected client
				if e != nil && !p.IsClientConnected(false) {
					p._lastConnectionErrorToNotifyClient = fmt.Sprintf("[%v] Failed to connect VPN: %s", time.Now().Format(time.Stamp), e.Error())
				}
			}

			var connectionError error

			// do not forget to notify that process was stopped (disconnected)
			defer func() {
				// Do not send "Disconnected" notification if we are going to establish new connection immediately
				if len(p._connRequestChan) == 0 || p._disconnectRequested {
					lastState := p._lastVPNState
					p._lastVPNState = vpn.NewStateInfo(vpn.DISCONNECTED, "")

					// Sending "Disconnected" only in one place (after VPN process stopped)
					disconnectionReason := types.Unknown
					if lastState.State == vpn.EXITING && lastState.IsAuthError {
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
					saveLastError(connectionError)
					p.notifyClients(&types.DisconnectedResp{Failure: connectionError != nil, Reason: disconnectionReason, ReasonDescription: errMsg})
				}
			}()

			p._lastConnectionErrorToNotifyClient = ""
			// SYNCHRONOUSLY start VPN connection process (wait until it finished)
			if connectionError = p.processConnectRequest(connectRequest); connectionError != nil {
				log.ErrorTrace(connectionError)
				saveLastError(connectionError)
			}
		}()
	}

}

func (p *Protocol) processConnectRequest(r service_types.ConnectionParams) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("panic on connect: " + fmt.Sprint(r))
			log.Error(err)
		}
	}()

	if p._disconnectRequested {
		log.Info("Disconnection was requested. Canceling connection.")
		return p._service.Disconnect()
	}

	return p._service.Connect(r)
}

func (p *Protocol) notifyVpnStateChanged(stateObj *vpn.StateInfo) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic when notifying VPN status to clients! (recovered)")
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
		}
	}()

	state := p._lastVPNState
	if stateObj != nil {
		p._lastVPNState = *stateObj
		state = *stateObj
	}

	switch state.State {
	case vpn.CONNECTED:
		p.notifyClients(p.createConnectedResponse(state))
	case vpn.DISCONNECTED:
		// suppress DISCONNECTED event. It will be sent to the client only after finishing the synchronous function processConnectRequest().
	default:
		p.notifyClients(&types.VpnStateResp{StateVal: state.State, State: state.State.String(), StateAdditionalInfo: state.StateAdditionalInfo})
	}
}

func (p *Protocol) OnVpnStateChanged(state vpn.StateInfo) {
	p.notifyVpnStateChanged(&state)
}

func (p *Protocol) OnVpnPauseChanged() {
	p.notifyVpnStateChanged(nil)
}
