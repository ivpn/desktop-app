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
	// new connections listener + current connection (needed to be able to stop service by closing them)
	_connListener     *net.TCPListener
	_clientConnection net.Conn

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
		sendResponse(conn, types.IVPNServiceExitingResponse())

		// closing current connection with a client
		conn.Close()
	}
}

// Start - starts TCP interface to communicate with IVPN application (server to listen incoming connections)
func (p *protocol) Start(startedOnPort chan<- int, service service.Service) error {
	if p._service != nil {
		return errors.New("unable to start protocol communication. It is already initialized")
	}
	p._service = service

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

	log.Info("IVPN service started: ", openedPortStr)
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
				p.sendResponse(types.IVPNServerListResponse(serv))
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

	reqType, err := getNetTypeName(messageData, true)
	if err != nil {
		log.Error("Failed to parse request:", err)
		return
	}

	log.Info("[<--] ", reqType)

	switch reqType {
	case "Hello":
		var req types.Hello

		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqType, err)
		}

		// TODO: remove TEST !
		req.KeepDaemonAlone = true

		log.Info(fmt.Sprintf("Connected client version: '%s' [set KeepDaemonAlone = %t]", req.Version, req.KeepDaemonAlone))
		p._keepAlone = req.KeepDaemonAlone

		p.sendResponse(types.IVPNHelloResponse())

		if req.GetServersList == true {
			serv, _ := p._service.ServersList()
			p.sendResponse(types.IVPNServerListResponse(serv))
		}

		if req.GetStatus == true {
			// send Firewall state
			if isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast, err := p._service.KillSwitchState(); err != nil {
				p.sendErrorResponse(reqType, err)
			} else {
				p.sendResponse(types.IVPNKillSwitchStatusResponse(isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast))
			}

			// send VPN connection  state
			vpnState := p._lastVPNState
			if vpnState.State == vpn.CONNECTED {
				p.sendResponse(types.IVPNConnectedResponse(vpnState.Time, vpnState.ClientIP.String(), vpnState.ServerIP.String(), vpnState.VpnType))
			}
		}
		break

	case "PingServers":
		var req types.PingServers
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqType, err)
		}

		retMap, err := p._service.PingServers(req.RetryCount, req.TimeOutMs)
		if err != nil {
			p.sendErrorResponse(reqType, err)
		} else {
			p.sendResponse(types.IVPNPingServersResponse(retMap))
		}
		break

	case "KillSwitchGetStatus":
		if isEnabled, _, _, _, err := p._service.KillSwitchState(); err != nil {
			p.sendErrorResponse(reqType, err)
		} else {
			p.sendResponse(types.IVPNKillSwitchGetStatusResponse(isEnabled))
		}
		break

	case "KillSwitchSetEnabled":
		var req types.KillSwitchSetEnabledRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqType, err)
		} else {
			if err := p._service.SetKillSwitchState(req.IsEnabled); err != nil {
				p.sendErrorResponse(reqType, err)
			} else {
				p.sendResponse(types.IVPNEmptyResponse())
			}
		}
		break

	case "KillSwitchSetAllowLANMulticast":
		var req types.KillSwitchSetAllowLANMulticastRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqType, err)
		} else {
			p._service.SetKillSwitchAllowLANMulticast(req.AllowLANMulticast)
		}
		break

	case "KillSwitchSetAllowLAN":
		var req types.KillSwitchSetAllowLANRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqType, err)
		} else {
			p._service.SetKillSwitchAllowLAN(req.AllowLAN)
		}
		break

	case "KillSwitchGetIsPestistent":
		isPersistant := p._service.Preferences().IsFwPersistant
		p.sendResponse(types.IVPNKillSwitchGetIsPestistentResponse(isPersistant))
		break

	case "KillSwitchSetIsPersistent":
		var req types.KillSwitchSetIsPersistentRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqType, err)
			break
		} else {
			if err := p._service.SetKillSwitchIsPersistent(req.IsPersistent); err != nil {
				p.sendErrorResponse(reqType, err)
			} else {
				p.sendResponse(types.IVPNEmptyResponse())
			}
		}
		break

	case "SetPreference":
		var req types.SetPreferenceRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqType, err)
		} else {
			if err := p._service.SetPreference(req.Key, req.Value); err != nil {
				p.sendErrorResponse(reqType, err)
			}
		}
		break

	case "GenerateDiagnostics":
		if log, log0, err := logger.GetLogText(1024 * 64); err != nil {
			p.sendErrorResponse(reqType, err)
		} else {
			p.sendResponse(types.IVPNDiagnosticsGeneratedResponse(log, log0))
		}
		break

	case "SetAlternateDns":
		var req types.SetAlternateDNS
		if err := json.Unmarshal(messageData, &req); err != nil {
			p.sendErrorResponse(reqType, err)
		} else {

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
				p.sendResponse(types.IVPNSetAlternateDNSResponse(false, net.IPv4zero.String()))
			} else {
				p.sendResponse(types.IVPNSetAlternateDNSResponse(true, req.DNS))
			}
		}
		break

	case "PauseConnection":
		if err := p._service.Pause(); err != nil {
			p.sendErrorResponse(reqType, err)
		} else {
			p.sendResponse(types.IVPNEmptyResponse())
		}
		break

	case "ResumeConnection":
		if err := p._service.Resume(); err != nil {
			p.sendErrorResponse(reqType, err)
		} else {
			p.sendResponse(types.IVPNEmptyResponse())
		}
		break

	case "Disconnect":
		p._disconnectRequested = true

		if err := p._service.Disconnect(); err != nil {
			p.sendErrorResponse(reqType, err)
		}
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
				p.sendResponse(types.IVPNDisconnectedResponse(disconnectAuthError, disconnectAuthError, disconnectDescription))
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
							p.sendResponse(types.IVPNConnectedResponse(state.Time, state.ClientIP.String(), state.ServerIP.String(), state.VpnType))
						} else {
							log.Debug("Skip sending 'Connected' notification. New connection request is awaiting ", cnt)
						}
					case vpn.EXITING:
						disconnectAuthError = state.IsAuthError
					default:
						p.sendResponse(types.IVPNVpnStateResponse(state.State.String(), ""))
					}
				case <-isExitChan:
					break state_forward_loop
				}
			}
		}()

		// Send 'connecting' status
		p.sendResponse(types.IVPNVpnStateResponse(vpn.CONNECTING.String(), ""))

		// SYNCHRONOUSLY start VPN connection process (wait until it finished)
		if err := p.processConnectRequest(messageData, stateChan); err != nil {
			disconnectDescription = err.Error()
			log.ErrorTrace(err)
		}

		break

	default:
		log.Warning("!!! Unsupported request type !!! ", reqType)
		log.Debug("Unsupported request:", message)
	}
}

func (p *protocol) sendResponse(bytesToSend []byte) error {
	return sendResponse(p.clientConnection(), bytesToSend)
}

func (p *protocol) sendErrorResponse(requestCommand string, err error) {
	log.Error(fmt.Sprintf("Error processing request '%s': %s", requestCommand, err))
	sendResponse(p.clientConnection(), types.IVPNErrorResponse(err))
}

func sendResponse(conn net.Conn, bytesToSend []byte) (retErr error) {
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("failed to send response to client: %w", retErr)
			log.Error(retErr)
		}
	}()

	if bytesToSend == nil {
		return fmt.Errorf("response is nil")
	}

	if _, err := conn.Write(bytesToSend); err != nil {
		return err
	}

	if _, err := conn.Write([]byte("\n")); err != nil {
		return err
	}

	// Just for logging
	if reqType, err := getNetTypeName(bytesToSend, false); err == nil {
		log.Info("[-->] ", reqType)
	} else {
		return fmt.Errorf("protocol error: BAD DATA SENT (%w)", err)
	}

	return nil
}

func getNetTypeName(messageData []byte, isRequest bool) (string, error) {
	type RequestObject struct {
		Command string
	}

	type ResponseObject struct {
		Type string
	}

	if isRequest {
		var cmdInfo RequestObject
		if err := json.Unmarshal(messageData, &cmdInfo); err != nil {
			log.Error("Failed to parse request:", err)
			return "", fmt.Errorf("failed to parse request (unable to determine Command): %w", err)
		}
		return cmdInfo.Command, nil
	}

	var typInfo ResponseObject
	if err := json.Unmarshal(messageData, &typInfo); err != nil {
		log.Error("Failed to parse response:", err)
		return "", fmt.Errorf("failed to parse response (unable to determine Type): %w", err)
	}

	return typInfo.Type, nil
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
		return fmt.Errorf("failed to parse VPN connection request: %w", err)
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

		connectionParams := openvpn.CreateConnectionParams(
			r.OpenVpnParameters.Username,
			r.OpenVpnParameters.Password,
			r.OpenVpnParameters.Port.Protocol > 0, // is TCP
			r.OpenVpnParameters.Port.Port,
			hosts,
			r.OpenVpnParameters.ProxyType,
			net.ParseIP(r.OpenVpnParameters.ProxyAddress),
			r.OpenVpnParameters.ProxyPort,
			r.OpenVpnParameters.ProxyUsername,
			r.OpenVpnParameters.ProxyPassword)

		prefs := p._service.Preferences()

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

		connectionParams := wireguard.CreateConnectionParams(
			net.ParseIP(r.WireGuardParameters.InternalClientIP),
			r.WireGuardParameters.LocalPrivateKey,
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
