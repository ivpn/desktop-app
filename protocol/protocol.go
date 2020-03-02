package protocol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/service"
	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/vpn"
	"github.com/ivpn/desktop-app-daemon/vpn/openvpn"
	"github.com/ivpn/desktop-app-daemon/vpn/wireguard"

	"github.com/pkg/errors"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("prtcl")
	rand.Seed(time.Now().UnixNano())
}

// CreateProtocol - Create new protocol object
func CreateProtocol() (service.Protocol, error) {
	return &protocol{}, nil
}

// Protocol - TCP interface to communicate with IVPN application
type protocol struct {
	_secret uint64

	// new connections listener + current connection (needed to be able to stop service by closing them)
	_connListener          *net.TCPListener
	_clientConnection      net.Conn
	_clientIsAuthenticated bool

	_service service.Service

	_connectMutex        sync.Mutex
	_disconnectRequested bool

	_connectRequestsMutex   sync.Mutex
	_connectRequests        int
	_connectRequestLastTime time.Time

	// _keepAlone informs daemon\service to do nothing when client disconnects
	// 		false (default) - VPN disconnects when client disconnects from a daemon
	// 		true - do nothing when client disconnects from a daemon (if VPN is connected - do not disconnect)
	_keepAlone bool

	// keep info about last VPN state
	_lastVPNState vpn.StateInfo
}

func (p *protocol) setClientConnection(conn net.Conn) {
	p._clientIsAuthenticated = false
	p._clientConnection = conn
}

func (p *protocol) clientConnection() net.Conn {
	return p._clientConnection
}

func (p *protocol) connectReqCount() (int, time.Time) {
	p._connectRequestsMutex.Lock()
	defer p._connectRequestsMutex.Unlock()

	return p._connectRequests, p._connectRequestLastTime
}
func (p *protocol) connectReqEnter() time.Time {
	p._connectRequestsMutex.Lock()
	defer p._connectRequestsMutex.Unlock()

	p._connectRequestLastTime = time.Now()
	p._connectRequests++
	return p._connectRequestLastTime
}
func (p *protocol) connectReqExit() {
	p._connectRequestsMutex.Lock()
	defer p._connectRequestsMutex.Unlock()

	p._connectRequests--
}

func (p *protocol) Stop() {
	log.Info("Stopping")

	listener := p._connListener
	conn := p._clientConnection
	if listener != nil {
		// do not accept new incoming connections
		listener.Close()
	}
	if conn != nil {
		// notifying client "service is going to stop" (client application (UI) will close)
		sendResponse(conn, types.ServiceExitingResp{}, 0)

		// closing current connection with a client
		conn.Close()
	}
}

// Start - starts TCP interface to communicate with IVPN application (server to listen incoming connections)
func (p *protocol) Start(secret uint64, startedOnPort chan<- int, service service.Service) error {
	if p._service != nil {
		return errors.New("unable to start protocol communication. It is already initialized")
	}
	p._service = service
	p._secret = secret

	defer func() {
		log.Warning("Protocol stopped")

		// Disconnect VPN (if not connected)
		p._service.Disconnect()
	}()

	adrr := fmt.Sprintf("127.0.0.1:0")
	// Initializing listener
	tcpAddr, err := net.ResolveTCPAddr("tcp4", adrr)
	if err != nil {
		return fmt.Errorf("failed to resolve TCP address: %w", err)
	}

	// strt listener
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

	// infinite loop of procesing IVPN client connection
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error("Server: failed to accept incoming connection:", err)
			return fmt.Errorf("(server) failed to accept incoming connection: %w", err)
		}
		p.processClient(conn)
	}
}

func (p *protocol) processClient(clientConn net.Conn) {
	// save connection
	p.setClientConnection(clientConn)

	log.Info("Client connected: ", clientConn.RemoteAddr())
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
		clientConn.Close()
		log.Info("Client disconnected: ", clientConn.RemoteAddr())

		if p._keepAlone == false {
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
			log.Info("Current state not changing [KeepDaemonAlone=true]")
		}
	}()

	// service changes notifier
	go func() {
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
				p.sendResponse(&types.ServerListResp{VpnServers: *serv}, 0)
			case <-stopChannel:
				return // stop loop
			}
		}
	}()

	reader := bufio.NewReader(clientConn)
	// run loop forever (or until ctrl-c)
	for {
		// will listen for message to process ending in newline (\n)
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Error("Error receiving data from client: ", err)
			break
		}

		// CONNECTION AUTHENTICATION: First request should be 'Hello' with correct authentication secret
		if p._clientIsAuthenticated == false {
			messageData := []byte(message)

			cmd, err := types.GetCommandBase(messageData)
			if err != nil {
				log.Error("Failed to parse initialisation request:", err)
				return
			}
			// ensure if client use correct secret
			if cmd.Command != "Hello" {
				logger.Error("Connection not authenticated. Closing.")
				return
			}
			// parsing 'Hello' request
			var hello types.Hello
			if err := json.Unmarshal(messageData, &hello); err != nil {
				p.sendErrorResponse(cmd, fmt.Errorf("connection authentication error: %w", err))
				return
			}
			if hello.Secret != p._secret {
				p.sendErrorResponse(cmd, fmt.Errorf("secret verification error"))
				return
			}
			p._clientIsAuthenticated = true
		}

		// Processing requests from client (in seperate routine)
		go p.processRequest(message)
	}
}

func (p *protocol) processRequest(message string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC during processing request!: ", r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
			log.Info("Closing connection and recovering state")
			p.clientConnection().Close()
		}
	}()

	messageData := []byte(message)

	reqCmd, err := types.GetCommandBase(messageData)
	if err != nil {
		log.Error("Failed to parse request:", err)
		return
	}

	log.Info("[<--] ", reqCmd.Command)

	switch reqCmd.Command {
	case "Hello":
		var req types.Hello

		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqCmd, err)
		}

		// TODO: remove TEST !
		req.KeepDaemonAlone = true

		log.Info(fmt.Sprintf("Connected client version: '%s' [set KeepDaemonAlone = %t]", req.Version, req.KeepDaemonAlone))
		p._keepAlone = req.KeepDaemonAlone

		p.sendResponse(&types.HelloResp{Version: "1.0"}, req.Idx)

		if req.GetServersList == true {
			serv, _ := p._service.ServersList()
			p.sendResponse(&types.ServerListResp{VpnServers: *serv}, req.Idx)
		}

		if req.GetStatus == true {
			/*
				// send Firewall state
				if isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast, err := p._service.KillSwitchState(); err != nil {
					p.sendErrorResponse(reqType, err)
				} else {
					p.sendResponse(&types.KillSwitchStatusResp{IsEnabled: isEnabled, IsPersistent: isPersistant, IsAllowLAN: isAllowLAN, IsAllowMulticast: isAllowLanMulticast})
				}*/

			// send VPN connection  state
			vpnState := p._lastVPNState
			if vpnState.State == vpn.CONNECTED {
				p.sendResponse(&types.ConnectedResp{TimeSecFrom1970: vpnState.Time, ClientIP: vpnState.ClientIP.String(), ServerIP: vpnState.ServerIP.String(), VpnType: vpnState.VpnType}, req.Idx)
			}
		}
		break

	case "GetVPNState":
		// send VPN connection  state
		vpnState := p._lastVPNState
		if vpnState.State == vpn.CONNECTED {
			p.sendResponse(&types.ConnectedResp{TimeSecFrom1970: vpnState.Time, ClientIP: vpnState.ClientIP.String(), ServerIP: vpnState.ServerIP.String(), VpnType: vpnState.VpnType}, reqCmd.Idx)
		} else if vpnState.State == vpn.DISCONNECTED {
			p.sendResponse(&types.DisconnectedResp{Failure: false, Reason: 0, ReasonDescription: ""}, reqCmd.Idx)
		} else {
			p.sendResponse(&types.VpnStateResp{StateVal: vpnState.State, State: vpnState.State.String()}, reqCmd.Idx)
		}

		break

	case "GetServers":
		serv, _ := p._service.ServersList()
		p.sendResponse(&types.ServerListResp{VpnServers: *serv}, reqCmd.Idx)
		break

	case "PingServers":
		var req types.PingServers
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		retMap, err := p._service.PingServers(req.RetryCount, req.TimeOutMs)
		if err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		var results []types.PingResultType
		for k, v := range retMap {
			results = append(results, types.PingResultType{Host: k, Ping: v})
		}

		p.sendResponse(&types.PingServersResp{PingResults: results}, req.Idx)
		break

	case "KillSwitchGetStatus":
		if isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast, err := p._service.KillSwitchState(); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		} else {
			p.sendResponse(&types.KillSwitchStatusResp{IsEnabled: isEnabled, IsPersistent: isPersistant, IsAllowLAN: isAllowLAN, IsAllowMulticast: isAllowLanMulticast}, reqCmd.Idx)
		}
		break

	case "KillSwitchSetEnabled":
		var req types.KillSwitchSetEnabled
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		if err := p._service.SetKillSwitchState(req.IsEnabled); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		p.sendResponse(&types.EmptyResp{}, req.Idx)
		break

	case "KillSwitchSetAllowLANMulticast":
		var req types.KillSwitchSetAllowLANMulticast
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		p._service.SetKillSwitchAllowLANMulticast(req.AllowLANMulticast)
		break

	case "KillSwitchSetAllowLAN":
		var req types.KillSwitchSetAllowLAN
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		p._service.SetKillSwitchAllowLAN(req.AllowLAN)
		break

	case "KillSwitchSetIsPersistent":
		var req types.KillSwitchSetIsPersistent
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		if err := p._service.SetKillSwitchIsPersistent(req.IsPersistent); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		p.sendResponse(&types.EmptyResp{}, req.Idx)
		break

	// TODO: can be fully replaced by 'KillSwitchGetStatus'
	case "KillSwitchGetIsPestistent":
		isPersistant := p._service.Preferences().IsFwPersistant
		p.sendResponse(&types.KillSwitchGetIsPestistentResp{IsPersistent: isPersistant}, reqCmd.Idx)
		break

	case "SetPreference":
		var req types.SetPreferenceRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		if err := p._service.SetPreference(req.Key, req.Value); err != nil {
			p.sendErrorResponse(reqCmd, err)
		}
		break

	case "GenerateDiagnostics":
		if log, log0, err := logger.GetLogText(1024 * 64); err != nil {
			p.sendErrorResponse(reqCmd, err)
		} else {
			p.sendResponse(&types.DiagnosticsGeneratedResp{ServiceLog: log, ServiceLog0: log0}, reqCmd.Idx)
		}
		break

	case "SetAlternateDns":
		var req types.SetAlternateDNS
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqCmd, err)
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
			p.sendResponse(&types.SetAlternateDNSResp{IsSuccess: false, ChangedDNS: net.IPv4zero.String()}, req.Idx)
		} else {
			p.sendResponse(&types.SetAlternateDNSResp{IsSuccess: true, ChangedDNS: req.DNS}, req.Idx)
		}
		break

	case "PauseConnection":
		if err := p._service.Pause(); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		p.sendResponse(&types.EmptyResp{}, reqCmd.Idx)
		break

	case "ResumeConnection":
		if err := p._service.Resume(); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		p.sendResponse(&types.EmptyResp{}, reqCmd.Idx)
		break

	case "SessionNew":
		var req types.SessionNew
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}

		apiResp, err := p._service.SessionNew(req.AccountID, req.ForceLogin)
		if err != nil {
			// If apiResp is not nil - we must return it to client
			// (response can contain addition information about error which can be processed by a client)
			// If apiResp == nil - it is communication error (we just return an error)
			if apiResp == nil {
				p.sendErrorResponse(reqCmd, err)
				break
			} else {
				log.Error(err)
			}
		}
		p.sendResponse(&types.SessionNewResp{APIResponse: *apiResp}, reqCmd.Idx)
		break

	case "SessionDelete":
		err := p._service.SessionDelete()
		if err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}
		p.sendResponse(&types.EmptyResp{}, reqCmd.Idx)
		break

	case "Disconnect":
		p._disconnectRequested = true

		if err := p._service.Disconnect(); err != nil {
			p.sendErrorResponse(reqCmd, err)
			break
		}
		p.sendResponse(&types.EmptyResp{}, reqCmd.Idx)
		break

	case "Connect":
		p._disconnectRequested = false
		requestTime := p.connectReqEnter()

		stateChan := make(chan vpn.StateInfo, 1)
		isExitChan := make(chan bool, 1)
		disconnectAuthError := false
		disconnectDescription := ""

		// disconnect active connection (if connected)
		if err := p._service.Disconnect(); err != nil {
			log.ErrorTrace(err)
		}

		p._connectMutex.Lock()
		defer p._connectMutex.Unlock()
		defer p.connectReqExit()

		// skip sending 'disconnected' state because we are giong to connect immediately
		if _, lastRequestTime := p.connectReqCount(); requestTime.Equal(lastRequestTime) == false {
			log.Info("Skipping awaited connection request. Newest request received.")
			return
		}

		var waiter sync.WaitGroup

		// do not forget to notify that process was stopped (disconnected)
		defer func() {

			// stop all go-routines related to this connections
			close(isExitChan)

			// Do not send "Disconnected" notification if we are giong to establish new connection immediately
			if cnt, _ := p.connectReqCount(); cnt == 1 || p._disconnectRequested == true {
				p._lastVPNState = vpn.NewStateInfo(vpn.DISCONNECTED, "")

				// Sending "Disconnected" only in one place (after VPN process stopped)
				authErr := 0
				if disconnectAuthError == true {
					authErr = 1
				}
				p.sendResponse(&types.DisconnectedResp{Failure: disconnectAuthError, Reason: authErr, ReasonDescription: disconnectDescription}, reqCmd.Idx)
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
				case state := <-stateChan:
					p._lastVPNState = state

					switch state.State {
					case vpn.CONNECTED:
						// Do not send "Connected" notification if we are giong to establish new connection immediately
						if cnt, _ := p.connectReqCount(); cnt == 1 || p._disconnectRequested == true {
							p.sendResponse(&types.ConnectedResp{TimeSecFrom1970: state.Time, ClientIP: state.ClientIP.String(), ServerIP: state.ServerIP.String(), VpnType: state.VpnType}, reqCmd.Idx)
						} else {
							log.Debug("Skip sending 'Connected' notification. New connection request is awaiting ", cnt)
						}
					case vpn.EXITING:
						disconnectAuthError = state.IsAuthError
					default:
						p.sendResponse(&types.VpnStateResp{StateVal: state.State, State: state.State.String()}, 0)
					}
				case <-isExitChan:
					break state_forward_loop
				}
			}
		}()

		// Send 'connecting' status
		p.sendResponse(&types.VpnStateResp{StateVal: vpn.CONNECTING, State: vpn.CONNECTING.String()}, 0)

		// SYNCHRONOUSLY start VPN connection process (wait until it finished)
		if err := p.processConnectRequest(messageData, stateChan); err != nil {
			disconnectDescription = err.Error()
			log.ErrorTrace(err)
		}

		break

	default:
		log.Warning("!!! Unsupported request type !!! ", reqCmd.Command)
		log.Debug("Unsupported request:", message)
	}
}

func (p *protocol) sendResponse(cmd interface{}, idx int) error {
	if p._clientIsAuthenticated == false {
		return fmt.Errorf("client is not authenticated")
	}

	err := sendResponse(p.clientConnection(), cmd, idx)
	if err != nil {
		log.Error(err)
	}
	return err
}

func (p *protocol) sendErrorResponse(request types.CommandBase, err error) {
	log.Error(fmt.Sprintf("Error processing request '%s': %s", request.Command, err))
	sendResponse(p.clientConnection(), &types.ErrorResp{ErrorMessage: err.Error()}, request.Idx)
}

func sendResponse(conn net.Conn, cmd interface{}, idx int) (retErr error) {
	if err := types.Send(conn, cmd, idx); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	// Just for logging
	if reqType := types.GetTypeName(cmd); len(reqType) > 0 {
		log.Info("[-->] ", reqType)
	} else {
		return fmt.Errorf("protocol error: BAD DATA SENT")
	}

	return nil
}

func (p *protocol) processConnectRequest(messageData []byte, stateChan chan<- vpn.StateInfo) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC on connect: ", r)
			// changing return values of main method
			err = errors.New("panic on connect: " + fmt.Sprint(r))
		}
	}()

	vpnObj, manualDNS, err := p.parseReqAndCreateVpnObj(messageData)
	if err != nil {
		return fmt.Errorf("connection request failed: %w", err)
	}

	if p._disconnectRequested == true {
		log.Info("Disconnection was requested. Canceling connection.")
		return vpnObj.Disconnect()
	}

	return p._service.Connect(vpnObj, manualDNS, stateChan)
}

// parseReqAndCreateVpnObj - Parse 'connect' request and create VPN object
func (p *protocol) parseReqAndCreateVpnObj(messageData []byte) (retVpnObj vpn.Process, retManualDNS net.IP, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC when parsing 'Connect' request: ", r)
			// changing return values
			retVpnObj = nil
			err = errors.New("panic when parsing 'Connect' request: " + fmt.Sprint(r))
		}
	}()

	var r types.Connect
	if err := json.Unmarshal(messageData, &r); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal json request: %w", err)
	}

	fmt.Println(string(messageData))

	retManualDNS = net.ParseIP(r.CurrentDNS)

	if vpn.Type(r.VpnType) == vpn.OpenVPN {
		var hosts []net.IP
		for _, v := range r.OpenVpnParameters.EntryVpnServer.IPAddresses {
			hosts = append(hosts, net.ParseIP(v))
		}

		prefs := p._service.Preferences()

		username := r.OpenVpnParameters.Username
		password := r.OpenVpnParameters.Password
		if len(username) == 0 || len(password) == 0 {
			username = prefs.VPNUser
			password = prefs.VPNPass
		}

		if len(username) == 0 || len(password) == 0 {
			return nil, retManualDNS, fmt.Errorf("not logged into an account")
		}

		connectionParams := openvpn.CreateConnectionParams(
			username,
			password,
			r.OpenVpnParameters.Port.Protocol > 0, // is TCP
			r.OpenVpnParameters.Port.Port,
			hosts,
			r.OpenVpnParameters.ProxyType,
			net.ParseIP(r.OpenVpnParameters.ProxyAddress),
			r.OpenVpnParameters.ProxyPort,
			r.OpenVpnParameters.ProxyUsername,
			r.OpenVpnParameters.ProxyPassword)

		retVpnObj, err = openvpn.NewOpenVpnObject(
			platform.OpenVpnBinaryPath(),
			platform.OpenvpnConfigFile(),
			platform.OpenvpnLogFile(),
			prefs.IsObfsproxy,
			prefs.OpenVpnExtraParameters,
			connectionParams)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to create new openVPN object: %w", err)
		}
	} else if vpn.Type(r.VpnType) == vpn.WireGuard {
		hostValue := r.WireGuardParameters.EntryVpnServer.Hosts[rand.Intn(len(r.WireGuardParameters.EntryVpnServer.Hosts))]

		internalIP := r.WireGuardParameters.InternalClientIP
		privateKey := r.WireGuardParameters.LocalPrivateKey

		if len(internalIP) == 0 || len(privateKey) == 0 {
			prefs := p._service.Preferences()
			internalIP = prefs.WGLocalIP
			privateKey = prefs.WGPrivateKey
		}

		if len(internalIP) == 0 || len(privateKey) == 0 {
			return nil, retManualDNS, fmt.Errorf("not logged into an account")
		}

		connectionParams := wireguard.CreateConnectionParams(
			net.ParseIP(internalIP),
			privateKey,
			r.WireGuardParameters.Port.Port,
			net.ParseIP(hostValue.Host),
			hostValue.PublicKey,
			net.ParseIP(strings.Split(hostValue.LocalIP, "/")[0]))

		retVpnObj, err = wireguard.NewWireGuardObject(
			platform.WgBinaryPath(),
			platform.WgToolBinaryPath(),
			platform.WGConfigFilePath(),
			connectionParams)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to create new WireGuard object: %w", err)
		}
	} else {
		log.Error("Unexpected VPN type to connect: ", r.VpnType)
		return nil, nil, errors.New("unexpected VPN type to connect")
	}

	return retVpnObj, retManualDNS, nil
}
