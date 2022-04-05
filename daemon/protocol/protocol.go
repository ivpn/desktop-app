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
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/oshelpers"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	"github.com/ivpn/desktop-app/daemon/vpn"
	"github.com/ivpn/desktop-app/daemon/vpn/openvpn"
	"github.com/ivpn/desktop-app/daemon/vpn/wireguard"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("prtcl")
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
	GetDisabledFunctions() (wgErr, ovpnErr, obfspErr, splitTunErr error)

	ServersList() (*apitypes.ServersInfoResponse, error)
	PingServers(retryCount int, timeoutMs int) (map[string]int, error)

	APIRequest(apiAlias string, ipTypeRequired types.RequiredIPProtocol) (responseData []byte, err error)

	KillSwitchState() (isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast, isAllowApiServers bool, err error)
	SetKillSwitchState(bool) error
	SetKillSwitchIsPersistent(isPersistant bool) error
	SetKillSwitchAllowLANMulticast(isAllowLanMulticast bool) error
	SetKillSwitchAllowLAN(isAllowLan bool) error
	SetKillSwitchAllowAPIServers(isAllowAPIServers bool) error

	SplitTunnelling_SetConfig(isEnabled bool, reset bool) error
	SplitTunnelling_GetStatus() (types.SplitTunnelStatus, error)
	SplitTunnelling_AddApp(exec string) (cmdToExecute string, isAlreadyRunning bool, err error)
	SplitTunnelling_RemoveApp(pid int, exec string) (err error)
	SplitTunnelling_AddedPidInfo(pid int, exec string, cmdToExecute string) error

	GetInstalledApps(extraArgsJSON string) ([]oshelpers.AppInfo, error)
	GetBinaryIcon(binaryPath string) (string, error)

	Preferences() preferences.Preferences
	SetPreference(key string, val string) error
	ResetPreferences() error

	SetManualDNS(dns dns.DnsSettings) error
	ResetManualDNS() error

	ConnectOpenVPN(connectionParams openvpn.ConnectionParams, manualDNS dns.DnsSettings, firewallOn bool, firewallDuringConnection bool, stateChan chan<- vpn.StateInfo) error
	ConnectWireGuard(connectionParams wireguard.ConnectionParams, manualDNS dns.DnsSettings, firewallOn bool, firewallDuringConnection bool, stateChan chan<- vpn.StateInfo) error
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

	SessionDelete(isCanDeleteSessionLocally bool) error
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
	_secret             uint64
	_paranoidModeSecret string

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

	// Notifying clients "service is going to stop" (client application (UI) will close)
	// Closing and erasing all clients connections
	// (do it only if stopping was requested by Stop() )
	p.notifyClientsDaemonExiting()

	listener := p._connListener
	if listener != nil {
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

	defer func() {
		log.Info("Protocol stopped")

		// Disconnect VPN (if connected)
		p._service.Disconnect()
	}()

	if err := p.paranoidModeInitFromFile(); err != nil {
		log.Error(fmt.Errorf("failed to initialize Paranoid Mode: %f", err))
	}

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

	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC during client communication!: ", r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
		}

		p.clientDisconnected(conn)
		log.Info("Client disconnected: ", conn.RemoteAddr())

		if isAuthenticated && !keepAlone {
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
			keepAlone = hello.KeepDaemonAlone
			isAuthenticated = true
			p.clientConnected(conn)
		}

		// Processing requests from client (in separate routine)
		go p.processRequest(conn, message)
	}
}

// normalizeField - wrapper around strings.Fields(). Returns only first field or empty string.
func normalizeField(s string) string {
	fields := strings.Fields(s)
	if len(fields) <= 0 {
		return ""
	}
	return fields[0]
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

	log.Info("[<--] ", p.connLogID(conn), reqCmd.Command)

	if p.paranoidModeIsEnabled() && !p.paranoidModeCheckSecret(reqCmd.ProtocolSecret) {
		p.sendErrorResponse(conn, reqCmd, fmt.Errorf("'Paranoid Mode' active: password is wrong"))
		return
	}

	sendState := func(reqIdx int, isOnlyIfConnected bool) {
		vpnState := p._lastVPNState
		if vpnState.State == vpn.CONNECTED {
			p.sendResponse(conn, p.createConnectedResponse(vpnState), reqIdx)
		} else if !isOnlyIfConnected {
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
		helloResponse := p.createHelloResponse()
		if req.GetParanoidModeFilePath {
			helloResponse.ParanoidModeFilePath = platform.ParanoidModeSecretFile()
		}
		p.sendResponse(conn, helloResponse, req.Idx)

		if req.GetServersList {
			serv, _ := p._service.ServersList()
			if serv != nil {
				p.sendResponse(conn, &types.ServerListResp{VpnServers: *serv}, req.Idx)
			}
		}

		if req.GetStatus {
			// send VPN connection  state
			sendState(req.Idx, true)

			// send Firewall state
			if isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast, isAllowApiServers, err := p._service.KillSwitchState(); err == nil {
				p.sendResponse(conn, &types.KillSwitchStatusResp{IsEnabled: isEnabled, IsPersistent: isPersistant, IsAllowLAN: isAllowLAN, IsAllowMulticast: isAllowLanMulticast, IsAllowApiServers: isAllowApiServers}, reqCmd.Idx)
			}
		}

		if req.GetConfigParams {
			p.sendResponse(conn,
				&types.ConfigParamsResp{UserDefinedOvpnFile: platform.OpenvpnUserParamsFile()},
				reqCmd.Idx)
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
		if err := p.paranoidModeSetSecret(req.OldSecret, req.NewSecret); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		} else {
			// send 'success' response to the requestor
			p.sendResponse(conn, &types.EmptyResp{}, req.Idx)
		}

	case "GetVPNState":
		// send VPN connection  state
		sendState(reqCmd.Idx, false)

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

	case "WiFiAvailableNetworks":
		networks := p._service.GetWiFiAvailableNetworks()
		nets := make([]types.WiFiNetworkInfo, 0, len(networks))
		for _, ssid := range networks {
			nets = append(nets, types.WiFiNetworkInfo{SSID: ssid})
		}

		p.notifyClients(&types.WiFiAvailableNetworksResp{
			Networks: nets})

	case "KillSwitchGetStatus":
		if isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast, isAllowApiServers, err := p._service.KillSwitchState(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		} else {
			p.sendResponse(conn, &types.KillSwitchStatusResp{IsEnabled: isEnabled, IsPersistent: isPersistant, IsAllowLAN: isAllowLAN, IsAllowMulticast: isAllowLanMulticast, IsAllowApiServers: isAllowApiServers}, reqCmd.Idx)
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
		// all clients will be notified in case of successfull change by OnKillSwitchStateChanged() handler

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
		// all clients will be notified in case of successfull change by OnKillSwitchStateChanged() handler

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
		if err := p._service.SplitTunnelling_SetConfig(req.IsEnabled, req.Reset); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
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
		if log, log0, err := logger.GetLogText(1024 * 64); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		} else {
			p.sendResponse(conn, &types.DiagnosticsGeneratedResp{ServiceLog: log, ServiceLog0: log0}, reqCmd.Idx)
		}

	case "SetAlternateDns":
		var req types.SetAlternateDns
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}
		req.Dns.DnsHost = normalizeField(req.Dns.DnsHost)
		req.Dns.DohTemplate = normalizeField(req.Dns.DohTemplate)

		var err error
		if req.Dns.IsEmpty() {
			err = p._service.ResetManualDNS()
		} else {
			err = p._service.SetManualDNS(req.Dns)

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
			p.sendResponse(conn, &types.SetAlternateDNSResp{IsSuccess: false, ErrorMessage: err.Error()}, req.Idx)
		} else {
			// notify all connected clients
			p.notifyClients(&types.SetAlternateDNSResp{IsSuccess: true, ChangedDNS: req.Dns})
			// send the response to the requestor
			p.sendResponse(conn, &types.SetAlternateDNSResp{IsSuccess: true, ChangedDNS: req.Dns}, req.Idx)
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
		if err := p._service.Pause(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)

	case "ResumeConnection":
		if err := p._service.Resume(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		p.sendResponse(conn, &types.EmptyResp{}, reqCmd.Idx)

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

		if req.NeedToDisableFirewall {
			p._service.SetKillSwitchIsPersistent(false)
			p._service.SetKillSwitchState(false)
		}

		err := p._service.SessionDelete(req.IsCanDeleteSessionLocally)
		if err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
			break
		}

		if req.NeedToResetSettings {
			oldPrefs := p._service.Preferences()

			// Reset settings only after SessionDelete() to correctly logout on the backed
			p._service.ResetPreferences()
			prefs := p._service.Preferences()

			// restore active persistant Firewall state
			if oldPrefs.IsFwPersistant != prefs.IsFwPersistant {
				p._service.SetKillSwitchIsPersistent(oldPrefs.IsFwPersistant)
			}

			// set AllowLan according to default values
			p._service.SetKillSwitchAllowLAN(prefs.IsFwAllowLAN)
			p._service.SetKillSwitchAllowLANMulticast(prefs.IsFwAllowLANMulticast)
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

	case "Disconnect":
		p._disconnectRequested = true

		if !p._service.Connected() {
			p.sendResponse(conn, &types.DisconnectedResp{Reason: types.DisconnectRequested}, reqCmd.Idx)
			break
		}

		if err := p._service.Disconnect(); err != nil {
			p.sendErrorResponse(conn, reqCmd, err)
		}

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
		if _, lastRequestTime := p.vpnConnectReqCounter(); !requestTime.Equal(lastRequestTime) {
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
				if disconnectAuthError {
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
							p.notifyClients(p.createConnectedResponse(state))
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

	default:
		log.Warning("!!! Unsupported request type !!! ", reqCmd.Command)
		log.Debug("Unsupported request:", message)
		p.sendErrorResponse(conn, reqCmd, fmt.Errorf("unsupported request: '%s'", reqCmd.Command))
	}
}
