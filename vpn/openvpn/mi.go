package openvpn

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

// ManagementInterface structure
type ManagementInterface struct {
	log *logger.Logger

	miConn    *net.TCPConn
	listener  *net.TCPListener
	stateChan chan<- vpn.StateInfo
	username  string
	password  string

	routeAddCmdsMutex sync.Mutex
	routeAddCmds      []string

	isConnected           bool
	isDisconnectRequested bool
}

// StartManagementInterface - starts TCP interface to communicate with IVPN application (server to listen incoming connections)
func StartManagementInterface(username string, password string, stateChan chan<- vpn.StateInfo) (mi *ManagementInterface, err error) {
	ret := &ManagementInterface{
		log:       logger.NewLogger("ovpnmi"),
		stateChan: stateChan,
		username:  username,
		password:  password}

	if err = ret.start(); err != nil {
		return nil, fmt.Errorf("failed to start MI: %w", err)
	}

	return ret, nil
}

// StopManagementInterface - Stop management interface manually
func (i *ManagementInterface) StopManagementInterface() error {
	if i.isConnected == false {
		return nil
	}

	i.log.Info("OpenVPN MI Stopping manually...")

	var ret error
	err := i.listener.Close()
	if err != nil {
		ret = err
		i.log.Error("OpenVPN MI Stopping: Failed to close listener:", err)
	}

	if i.miConn != nil {
		err = i.miConn.Close()
		if err != nil {
			ret = err
			i.log.Error("OpenVPN MI Stopping: Failed to close connection:", err)
		}
	}
	return ret
}

// ListenAddress returns ip:port of listener
func (i *ManagementInterface) ListenAddress() (ip string, port int, e error) {

	listener := i.listener
	if listener == nil {
		return "", 0, errors.New("listener not defined")
	}

	// return port opened for connection
	splittedAddr := strings.Split(listener.Addr().String(), ":")

	addr := splittedAddr[0]
	port, err := strconv.Atoi(splittedAddr[1])
	if err != nil {
		return "", 0, err
	}

	return addr, port, nil
}

// SendDisconnect - Send disconnect command to openvpn
func (i *ManagementInterface) SendDisconnect() error {
	i.isDisconnectRequested = true
	return i.sendResponse("signal SIGTERM")
}

// GetRouteAddCommands - return all detected route-add command
func (i *ManagementInterface) GetRouteAddCommands() []string {
	i.routeAddCmdsMutex.Lock()
	defer i.routeAddCmdsMutex.Unlock()

	return i.routeAddCmds
}

// start - starts TCP interface to communicate with IVPN application (server to listen incoming connections)
func (i *ManagementInterface) start() error {
	if i.isDisconnectRequested {
		return errors.New("disconnection already requested for this MI object. To perform new connection, please, initialize new object")
	}

	adrr := fmt.Sprintf("127.0.0.1:0")
	// Initializing listener
	tcpAddr, err := net.ResolveTCPAddr("tcp4", adrr)
	if err != nil {
		return fmt.Errorf("failed to resolve local ip addr: %w", err)
	}

	// start listener
	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("failed to start MI listener: %w", err)
	}
	i.listener = l

	// asynchronously accept connection from openvpn and start communication
	go func() {
		i.isConnected = true

		defer func() {
			i.listener.Close()
			i.log.Info("OpenVPN MI stopped")
			i.isConnected = false // mark: connection is closed
		}()

		i.log.Info("OpenVPN MI started")

		conn, err := i.listener.AcceptTCP()
		if err != nil {
			i.log.Error("Failed to accept incoming connection from OpenVPN MI: ", err)
			return
		}
		i.miConn = conn

		if i.isDisconnectRequested {
			i.log.Info("Disconnection requested")
			i.SendDisconnect()
			return
		}

		i.miCommunication()
	}()

	return nil
}

// miCommunication - communication with openVPN process (OpenVPN Management Interface).
// Processing requests from client and sending response
func (i *ManagementInterface) miCommunication() {

	mesRegexp := regexp.MustCompile("^>([a-zA-Z0-9-]+):(.*)")
	mesNeedPassRegexp := regexp.MustCompile("Need '(.+)' username/password")
	mesLogRouteAddCmdRegexp := regexp.MustCompile(".*route(.exe)?[ \t]+add[ \t]+")

	if i.miConn == nil {
		i.log.Panic("INTERNAL ERROR: OpenVPN MI connection is null!")
	}

	i.log.Info("OpenVPN MI connected: ", i.miConn.RemoteAddr())
	defer func() {
		i.miConn.Close()
		i.log.Info("OpenVPN MI disconnected: ", i.miConn.RemoteAddr())
	}()

	reader := bufio.NewReader(i.miConn)
	for {
		// will listen for message to process ending in newline (\n)
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				i.log.Info("Connection closed (EOF)")
			} else {
				i.log.Error("Error receiveing data from client: ", err)
			}
			break
		}
		if len(message) <= 0 {
			continue
		}

		i.log.Info("[<-]: ", message)

		columns := mesRegexp.FindStringSubmatch(message)
		if len(columns) <= 2 {
			continue
		}

		msgSource := columns[1]
		msgText := columns[2]

		switch msgSource {
		case "LOG":
			// detect for routing change commands

			// LOG:1564229538,,/sbin/route add -net 128.0.0.0 10.57.40.1 128.0.0.0
			cols := strings.Split(msgText, ",")
			if len(cols) == 3 {
				cmdStr := strings.ToLower(cols[2])
				// /sbin/route add -net 128.0.0.0 10.57.40.1 128.0.0.0
				if mesLogRouteAddCmdRegexp.MatchString(cmdStr) {
					i.addRouteAddCommand(cmdStr)
				}
			}
			break

		case "INFO":
			break

		case "HOLD":
			i.sendResponse("state on", "log on", "hold off", "hold release")
			break

		case "PASSWORD":
			if strings.HasPrefix(msgText, "Verification Failed: 'Auth'") {
				// Authentication error is handled by state: >STATE:1563526742,EXITING,auth-failure,,,,,
				break
			}

			if mesNeedPassRegexp.Match([]byte(msgText)) == false {
				continue
			}

			i.sendResp(false, fmt.Sprintf("username \"Auth\" %s", i.username))

			// Some passwords for tests in case of implementation change:
			//
			// #;0$%:k'j?~?:f3%2,O4x<
			// #;0///$%\\\:k\\'j?\~?://f3%2,/O4x<
			// ;0///$%\\\:k\\'j?\~?://f3%2,/O4x<#456!@#$%^&*()_+}{P||:?><~~
			// ";0///$%\\\:k\\'j?\~?://""f3%2""",/O4x<#456!@#$%^&*()_+}{P||:?><~~
			// lkhgd#;0$%:k'j?~?:f3%2,"O4x<
			escapedPass := strings.ReplaceAll(i.password, "\\", "\\\\")
			escapedPass = strings.ReplaceAll(escapedPass, "\"", "\\\"")
			i.sendResp(false, fmt.Sprintf("password \"Auth\" %s", escapedPass))
			break

		case "STATE":
			// The output format consists of 4 comma-separated parameters:
			//  (a) the integer unix date/time,
			//  (b) the state name,
			//  (c) optional descriptive string (used mostly on RECONNECTING
			//      and EXITING to show the reason for the disconnect),
			//  (d) optional TUN/TAP local IP address (shown for ASSIGN_IP
			//      and CONNECTED), and
			//  (e) optional address of remote server (OpenVPN 2.1 or higher).
			params := strings.Split(msgText, ",")
			if len(params) < 2 {
				i.log.Error("STATE format error.")
				continue
			}
			stateStr := params[1]

			state, err := parseState(stateStr)
			if err != nil {
				i.log.Error("Unable to parse VPN state:", err.Error())
			} else {
				i.log.Info("State changed:", state)

				var clientIP net.IP
				var serverIP net.IP
				var isAuthError bool
				// If state is Connected - save local and server IP addresses
				if state == vpn.CONNECTED {
					if len(params) > 3 {
						clientIP = net.ParseIP(params[3])
					}
					if len(params) > 4 {
						serverIP = net.ParseIP(params[4])
					}
				} else if state == vpn.EXITING {
					//>STATE:1563526742,EXITING,auth-failure,,,,,
					if strings.Contains(msgText, "auth-failure") { //if (params[2] == "auth-failure")
						isAuthError = true
					}
				}

				// save current state info
				state := vpn.StateInfo{
					State:       state,
					Description: msgText,
					ClientIP:    clientIP,
					ServerIP:    serverIP,
					IsAuthError: isAuthError}

				select {
				case i.stateChan <- state: // notify: state was changed
				default:
					i.log.Debug("State channel is full. Waiting...")
					i.stateChan <- state
				}
			}

			break
		}

	}
}

func (i *ManagementInterface) sendResponse(commands ...string) error {
	for _, cmd := range commands {

		if err := i.sendResp(true, cmd); err != nil {
			return fmt.Errorf("failed to send response from MI: %w", err)
		}
	}
	return nil
}

func (i *ManagementInterface) sendResp(canLog bool, command string) error {
	if canLog {
		i.log.Info("[->]: ", command)
	}
	conn := i.miConn
	if conn == nil {
		i.log.Info("Unable to send command to MI. Connection not initialized.")
		return nil
	}

	// FIXME: sometimes, we are not receiving all data from MI connection without this delay (on Windows)
	time.Sleep(time.Millisecond)

	if _, err := i.miConn.Write([]byte(command + "\n")); err != nil {
		return fmt.Errorf("failed to write data to connection: %w", err)
	}

	return nil
}

func (i *ManagementInterface) addRouteAddCommand(command string) {
	i.routeAddCmdsMutex.Lock()
	defer i.routeAddCmdsMutex.Unlock()

	command = strings.TrimSpace(command) // this is reqid

	i.routeAddCmds = append(i.routeAddCmds, command)
	i.log.Debug("New route-add command (", len(i.routeAddCmds), "): ", command)
}

// ParseState - Converts string representation of OpenVPN state to vpn.State
func parseState(stateStr string) (vpn.State, error) {
	stateStr = strings.Trim(stateStr, " \t;,.")
	switch stateStr {
	case "CONNECTING":
		return vpn.CONNECTING, nil
	case "WAIT":
		return vpn.WAIT, nil
	case "AUTH":
		return vpn.AUTH, nil
	case "GET_CONFIG":
		return vpn.GETCONFIG, nil
	case "ASSIGN_IP":
		return vpn.ASSIGNIP, nil
	case "ADD_ROUTES":
		return vpn.ADDROUTES, nil
	case "CONNECTED":
		return vpn.CONNECTED, nil
	case "RECONNECTING":
		return vpn.RECONNECTING, nil
	case "EXITING":
		return vpn.EXITING, nil
	default:
		return vpn.DISCONNECTED, errors.New("unexpected state:" + stateStr)
	}
}
