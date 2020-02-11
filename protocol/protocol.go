package protocol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"ivpn/daemon/logger"
	"ivpn/daemon/protocol/types"
	"ivpn/daemon/service"
	"ivpn/daemon/service/platform"
	"ivpn/daemon/vpn"
	"ivpn/daemon/vpn/openvpn"
	"ivpn/daemon/vpn/wireguard"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

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
	_activeConnection *net.Conn

	service service.Service

	_connectMutex        sync.Mutex
	_disconnectRequested bool

	_connectRequestsMutex   sync.Mutex
	_connectRequests        int
	_ConnectRequestLastTime time.Time
}

func (p *protocol) connectReqCount() (int, time.Time) {
	p._connectRequestsMutex.Lock()
	defer p._connectRequestsMutex.Unlock()

	return p._connectRequests, p._ConnectRequestLastTime
}
func (p *protocol) connectReqEnter() time.Time {
	p._connectRequestsMutex.Lock()
	defer p._connectRequestsMutex.Unlock()

	p._ConnectRequestLastTime = time.Now()
	p._connectRequests++
	return p._ConnectRequestLastTime
}
func (p *protocol) connectReqExit() {
	p._connectRequestsMutex.Lock()
	defer p._connectRequestsMutex.Unlock()

	p._connectRequests--
}

func (p *protocol) Stop() {
	log.Info("Stopping")

	listener := p._connListener
	conn := p._activeConnection
	if listener != nil {
		// do not accept new incoming connections
		listener.Close()
	}
	if conn != nil {
		// notifying client "service is going to stop" (client application (UI) will close)
		sendResponse(*conn, types.IVPNServiceExitingResponse())

		// closing current connection with a client
		(*conn).Close()
	}
}

// Start - starts TCP interface to communicate with IVPN application (server to listen incoming connections)
func (p *protocol) Start(startedOnPort chan<- int, service service.Service) error {
	if p.service != nil {
		return errors.New("unable to start protocol communication. It is already initialized")
	}
	p.service = service

	defer func() {
		log.Warning("Protocol stopped")

		// Disconnect VPN (if not connected)
		p.service.Disconnect()
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

		// save connection to a protocol field (to be able to stop it)
		p._activeConnection = &conn

		p.processClient(conn)
	}
}

func (p *protocol) processClient(conn net.Conn) {
	log.Info("Client connected: ", conn.RemoteAddr())
	defer func() {

		if r := recover(); r != nil {
			log.Error("PANIC during client communication!: ", r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
		}
		conn.Close()
		log.Info("Client disconnected: ", conn.RemoteAddr())

		stopService, err := p.service.OnControlConnectionClosed()
		if err != nil {
			log.Error(err)
		}

		// Disconnect VPN (if connected)
		if err := p.service.Disconnect(); err != nil {
			log.Error(err)
		}

		if stopService {
			log.Info("Stopping due to configuration: Stop IVPN Agent when application is not running")
			p.Stop()
		}
	}()

	reader := bufio.NewReader(conn)
	// run loop forever (or until ctrl-c)
	for {
		// will listen for message to process ending in newline (\n)
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Error("Error receiving data from client: ", err)
			break
		}

		// Processing requests from client (in seperate routine)
		go p.processRequest(conn, message)
	}
}

func (p *protocol) processRequest(conn net.Conn, message string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC during processing request!: ", r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
			log.Info("Closing connection and recovering state")
			conn.Close()
		}
	}()

	messageData := []byte(message)

	reqType, err := getNetTypeName(messageData)
	if err != nil {
		log.Error("Failed to parse request:", err)
		return
	}

	log.Info("[<--] ", strings.Replace(reqType, "IVPN.", "", 1))

	switch reqType {
	case "IVPN.IVPNHelloRequest":
		sendResponse(conn, types.IVPNHelloResponse())
		//sendResponse(conn, types.IVPNGetPreferencesResponse(make(map[string]string)))

		serv, _ := p.service.ServersList()
		sendResponse(conn, types.IVPNServerListResponse(serv))
		break

	case "IVPN.IVPNPingServers":
		var req types.PingServers
		if err := json.Unmarshal(messageData, &req); err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
		}

		retMap, err := p.service.PingServers(req.RetryCount, req.TimeOutMs)
		if err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
		} else {
			sendResponse(conn, types.IVPNPingServersResponse(retMap))
		}
		break

	case "IVPN.IVPNKillSwitchGetStatusRequest":
		if isEnabled, err := p.service.KillSwitchState(); err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
		} else {
			sendResponse(conn, types.IVPNKillSwitchGetStatusResponse(isEnabled))
		}
		break

	case "IVPN.IVPNKillSwitchSetAllowLANMulticastRequest":
		var req types.KillSwitchSetAllowLANMulticastRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
		} else {
			p.service.SetKillSwitchAllowLANMulticast(req.AllowLANMulticast)
		}
		break

	case "IVPN.IVPNKillSwitchSetAllowLANRequest":
		var req types.KillSwitchSetAllowLANRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
		} else {
			p.service.SetKillSwitchAllowLAN(req.AllowLAN)
		}
		break

	case "IVPN.IVPNKillSwitchGetIsPestistentRequest":
		isPersistant := p.service.Preferences().IsFwPersistant
		sendResponse(conn, types.IVPNKillSwitchGetIsPestistentResponse(isPersistant))
		break

	case "IVPN.IVPNKillSwitchSetIsPersistentRequest":
		var req types.KillSwitchSetIsPersistentRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
			break
		} else {
			if err := p.service.SetKillSwitchIsPersistent(req.IsPersistent); err != nil {
				sendResponse(conn, types.IVPNErrorResponse(err))
			} else {
				sendResponse(conn, types.IVPNEmptyResponse())
			}
		}
		break

	case "IVPN.IVPNSetPreferenceRequest":
		var req types.SetPreferenceRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
		} else {
			if err := p.service.SetPreference(req.Key, req.Value); err != nil {
				sendResponse(conn, types.IVPNErrorResponse(err))
			}
		}
		break

	case "IVPN.IVPNGenerateDiagnosticsRequest":
		if log, log0, err := logger.GetLogText(1024 * 64); err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
		} else {
			sendResponse(conn, types.IVPNDiagnosticsGeneratedResponse(log, log0))
		}
		break

	case "IVPN.IVPNSetAlternateDns":
		var req types.SetAlternateDNS
		if err := json.Unmarshal(messageData, &req); err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
		} else {

			var err error
			if ip := net.ParseIP(req.DNS); ip == nil || ip.Equal(net.IPv4zero) || ip.Equal(net.IPv4bcast) {
				err = p.service.ResetManualDNS()
			} else {
				err = p.service.SetManualDNS(ip)

				if err != nil {
					// DNS set failed. Trying to reset DNS
					errReset := p.service.ResetManualDNS()
					if errReset != nil {
						log.ErrorTrace(errReset)
					}
				}
			}

			if err != nil {
				log.ErrorTrace(err)
				sendResponse(conn, types.IVPNSetAlternateDNSResponse(false, net.IPv4zero.String()))
			} else {
				sendResponse(conn, types.IVPNSetAlternateDNSResponse(true, req.DNS))
			}
		}
		break

	case "IVPN.IVPNKillSwitchSetEnabledRequest":
		var req types.KillSwitchSetEnabledRequest
		if err := json.Unmarshal(messageData, &req); err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
		} else {
			if err := p.service.SetKillSwitchState(req.IsEnabled); err != nil {
				sendResponse(conn, types.IVPNErrorResponse(err))
			} else {
				sendResponse(conn, types.IVPNEmptyResponse())
			}
		}
		break

	case "IVPN.IVPNPauseConnection":
		if err := p.service.Pause(); err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
		} else {
			sendResponse(conn, types.IVPNEmptyResponse())
		}
		break

	case "IVPN.IVPNResumeConnection":
		if err := p.service.Resume(); err != nil {
			sendResponse(conn, types.IVPNErrorResponse(err))
		} else {
			sendResponse(conn, types.IVPNEmptyResponse())
		}
		break

	case "IVPN.IVPNDisconnectRequest":
		p._disconnectRequested = true

		if err := p.service.Disconnect(); err != nil {
			log.Error("Disconnection error: ", err)
		}
		break

	case "IVPN.IVPNConnectRequest":
		p._disconnectRequested = false
		requestTime := p.connectReqEnter()

		stateChan := make(chan vpn.StateInfo, 1)
		isExitChan := make(chan bool, 1)
		disconnectAuthError := false
		disconnectDescription := ""

		// disconnect active connection (if connected)
		if err := p.service.Disconnect(); err != nil {
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
				// Sending "Disconnected" only in one place (after VPN process stopped)
				sendResponse(conn, types.IVPNDisconnectedResponse(disconnectAuthError, disconnectAuthError, disconnectDescription))
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
					switch state.State {
					case vpn.CONNECTED:
						// Do not send "Connected" notification if we are giong to establish new connection immediately
						if cnt, _ := p.connectReqCount(); cnt == 1 || p._disconnectRequested == true {
							sendResponse(conn, types.IVPNConnectedResponse(time.Now().Unix(), state.ClientIP.String(), state.ServerIP.String()))
						} else {
							log.Debug("Skip sending 'Connected' notification. New connection request is awaiting ", cnt)
						}
					case vpn.EXITING:
						disconnectAuthError = state.IsAuthError
					default:
						sendResponse(conn, types.IVPNStateResponse(state.State.String(), ""))
					}
				case <-isExitChan:
					break state_forward_loop
				}
			}
		}()

		// Send 'connecting' status
		sendResponse(conn, types.IVPNStateResponse(vpn.CONNECTING.String(), ""))

		// SYNCHRONOUSLY start VPN connection process (wait until it finished)
		if err := p.processConnectRequest(messageData, stateChan); err != nil {
			disconnectDescription = err.Error()
			log.ErrorTrace(err)
		}

		break

	default:
		log.Warning("!!! Unsupported request type !!! ", reqType)
		log.Debug("Unsupported request:", message)
		//sendResponse(conn, types.IVPNErrorResponse(errors.New("unsupported request:"+reqType)))
	}
}

func getNetTypeName(messageData []byte) (string, error) {
	var dat map[string]interface{}

	if err := json.Unmarshal(messageData, &dat); err != nil {
		return "", err
	}

	reqType, ok := dat["$type"].(string)
	if ok == false {
		return "", errors.New("bad JSON request, no '$type' field")
	}

	return strings.Split(reqType, ",")[0], nil
}

func sendResponse(conn net.Conn, bytesToSend []byte) {
	if bytesToSend == nil {
		log.Error("Unable to send response. Response is nil")
	}

	conn.Write(bytesToSend)
	conn.Write([]byte("\n"))

	// Just for logging
	if reqType, err := getNetTypeName(bytesToSend); err == nil {
		log.Info("[-->] ", reqType)
	} else {
		log.Error("Protocol error: BAD DATA WAS SENT. ", err)
	}
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

	return p.service.Connect(vpnObj, manualDNS, stateChan)
}

// parseReqAndCreateVpnObj - Parse 'connect' request and create VPN object
func (p *protocol) parseReqAndCreateVpnObj(messageData []byte) (retVpnObj vpn.Process, manualDNS net.IP, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC when parsing 'Connect' request: ", r)
			// changing return values of main method
			retVpnObj = nil
			err = errors.New("panic when parsing 'Connect' request: " + fmt.Sprint(r))
		}
	}()

	var dat map[string]interface{}

	if err := json.Unmarshal(messageData, &dat); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal json request: %w", err)
	}

	vpnType := (vpn.Type)(dat["VpnType"].(float64))

	manualDNS = net.ParseIP(dat["CurrentDns"].(string))

	if vpnType == vpn.OpenVPN {
		var username string
		var password string
		var proxyType string
		var proxyAddress string
		var proxyUsername string
		var proxyPassword string
		var proxyPort int
		var protocol int
		var hosts []net.IP

		params := dat["OpenVpnParameters"].(map[string]interface{})
		username = params["Username"].(string)
		password = params["Password"].(string)

		if val := params["ProxyType"]; val != nil {
			proxyType = val.(string)
		}
		if val := params["ProxyAddress"]; val != nil {
			proxyAddress = val.(string)
		}
		if val := params["ProxyUsername"]; val != nil {
			proxyUsername = val.(string)
		}
		if val := params["ProxyPassword"]; val != nil {
			proxyPassword = val.(string)
		}
		if val := params["ProxyPort"]; val != nil {
			proxyPort = int(val.(float64))
		}

		portObj := params["Port"].(map[string]interface{})
		hostPort := (int)(portObj["Port"].(float64))
		protocol = (int)(portObj["Protocol"].(float64))

		svr := params["EntryVpnServer"].(map[string]interface{})
		addreses := svr["IpAddresses"].(map[string]interface{})
		addresesValues := addreses["$values"].([]interface{})
		for _, v := range addresesValues {
			hostValue := v.(string)
			hosts = append(hosts, net.ParseIP(hostValue))
		}

		isTCP := protocol > 0

		connectionParams := openvpn.CreateConnectionParams(username, password, isTCP, hostPort, hosts, proxyType, net.ParseIP(proxyAddress), proxyPort, proxyUsername, proxyPassword)

		prefs := p.service.Preferences()

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
	} else if vpnType == vpn.WireGuard {
		wgParams := dat["WireGuardParameters"].(map[string]interface{})

		portObj := wgParams["Port"].(map[string]interface{})
		hostPort := (int)(portObj["Port"].(float64))

		clientLocalIP := wgParams["InternalClientIp"].(string)
		clientPrivateKey := wgParams["LocalPrivateKey"].(string)

		svr := wgParams["EntryVpnServer"].(map[string]interface{})
		hosts := svr["Hosts"].(map[string]interface{})
		hostsValues := hosts["$values"].([]interface{})

		hostsCnt := len(hostsValues)
		if hostsCnt == 0 {
			return nil, nil, errors.New("WireGuardParameters parsing error: no hosts defined")
		}

		hostValue := hostsValues[rand.Intn(hostsCnt)].(map[string]interface{})

		hostIP := hostValue["Host"].(string)
		hostPublicKey := hostValue["PublicKey"].(string)
		hostLocalIP := hostValue["LocalIp"].(string)
		// 10.0.0.1/24 => 10.0.0.1
		hostLocalIP = strings.Split(hostLocalIP, "/")[0]

		connectionParams := wireguard.CreateConnectionParams(net.ParseIP(clientLocalIP), clientPrivateKey, hostPort, net.ParseIP(hostIP), hostPublicKey, net.ParseIP(hostLocalIP))

		retVpnObj, err = wireguard.NewWireGuardObject(
			platform.WgBinaryPath(),
			platform.WgToolBinaryPath(),
			platform.WGConfigFilePath(),
			connectionParams)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create new WireGuard object: %w", err)
		}
	} else {
		log.Error("Unexpected VPN type to connect: ", vpnType)
		return nil, nil, errors.New("unexpected VPN type to connect")
	}

	return retVpnObj, manualDNS, nil
}
