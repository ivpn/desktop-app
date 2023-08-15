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

	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

// ManagementInterface structure
type ManagementInterface struct {
	log *logger.Logger

	secret         string
	isConnVerified chan struct{}

	miConn    *net.TCPConn
	listener  *net.TCPListener
	stateChan chan<- vpn.StateInfo
	username  string
	password  string

	routeAddCmdsMutex sync.Mutex
	routeAddCmds      []string

	isConnected           bool
	isDisconnectRequested bool

	pushReplyCmds []string
	pushReplyDNS  net.IP
}

// StartManagementInterface - starts TCP interface to communicate with IVPN application (server to listen incoming connections)
func StartManagementInterface(miSecret string, username string, password string, stateChan chan<- vpn.StateInfo) (mi *ManagementInterface, err error) {
	ret := &ManagementInterface{
		secret:         miSecret,
		isConnVerified: make(chan struct{}),

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
	if !i.isConnected {
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

// SetConnectionVerified sets the current MI connection as verified: communication allowed
func (i *ManagementInterface) SetConnectionVerified() {
	// do not block if channell already full
	select {
	case i.isConnVerified <- struct{}{}:
	default:
	}
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

	ret := make([]string, len(i.routeAddCmds))
	copy(ret, i.routeAddCmds)

	return ret
}

func (i *ManagementInterface) HasRouteAddCommands() bool {
	i.routeAddCmdsMutex.Lock()
	defer i.routeAddCmdsMutex.Unlock()

	return len(i.routeAddCmds) > 0
}

// start - starts TCP interface to communicate with IVPN application (server to listen incoming connections)
func (i *ManagementInterface) start() error {
	if i.isDisconnectRequested {
		return errors.New("disconnection already requested for this MI object. To perform new connection, please, initialize new object")
	}

	adrr := "127.0.0.1:0"
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
			//erase connection properties
			i.pushReplyDNS = nil
			i.pushReplyCmds = make([]string, 0)

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
	// Example: "/sbin/route" - for macOS, "/sbin/ip route" - for Linux, "C:\\Windows\\System32\\ROUTE.EXE" - for Windows
	routeCommand := platform.RouteCommand()

	mesRegexp := regexp.MustCompile("^>([a-zA-Z0-9-]+):(.*)")
	mesNeedPassRegexp := regexp.MustCompile("Need '(.+)' username/password")

	// 'route add ...' commands detection RegExp
	// Windows (OpenVPN 2.4):	>LOG:1612484260,,C:\WINDOWS\system32\route.exe ADD 123.200.11.22 MASK 255.255.255.255 192.168.1.1
	// macOS   (OpenVPN 2.4):	>LOG:1612517083,,/sbin/route add -net 123.200.11.22 192.168.1.1 255.255.255.255\r
	// Linux   (OpenVPN 2.4):	>LOG:1612516859,,/sbin/ip route add 123.200.11.22/32 via 192.168.1.1\r
	mesLogRouteAddCmdRegexp := regexp.MustCompile(
		"(?i)" + // i modifier: insensitive. Case insensitive match (ignores case of [a-zA-Z])
			"^" + // beginning of the line (it is important for security reason)
			regexp.QuoteMeta(routeCommand) + "[ ]+" + // platform-specific route command
			"ADD[ \t]+" + // 'add' instruction
			"(-net[ \t]+)?" + // '-net' instruction for macOS
			"(([0-9]{1,3}[.]){3,3}[0-9]{1,3})(/[0-9]{1,2})?[ \t]+" + // IPv4 address
			"((MASK|via)[ \t]+)?" + // instructions 'MASK' for Windows or 'via' for Linux
			"([0-9]{1,3}[.]){3,3}[0-9]{1,3}([ \t]+" + // IPv4 address
			"([0-9]{1,3}[.]){3,3}[0-9]{1,3})?") // IPv4 address

	mesLogPushReplyCmdRegexp := regexp.MustCompile(".*PUSH.*'PUSH_REPLY[ ,]*(.*)'")

	mesLogRouteAddCmdRegexpOvpn45 := regexp.MustCompile(".*net_route_v4_add:[ \t]+(([0-9]{1,3}[.]){3,3}[0-9]{1,3}(\\/[0-9]+)?[ \t]+.*[ \t]+([0-9]{1,3}[.]){3,3}[0-9]{1,3}).*")

	if i.miConn == nil {
		i.log.Panic("INTERNAL ERROR: OpenVPN MI connection is null!")
	}

	i.log.Info("OpenVPN MI connected: ", i.miConn.RemoteAddr())
	defer func() {
		i.miConn.Close()
		i.log.Info("OpenVPN MI disconnected: ", i.miConn.RemoteAddr())
	}()

	// sending secret value to be verified by daemon
	i.sendResponse(fmt.Sprintf("echo %s", i.secret))
	// waiting for verification
	// if not verified during 5 seconds - close current MI connection
	select {
	case <-i.isConnVerified:
		i.log.Info("Connection verified")
		break
	case <-time.After(5 * time.Second):
		i.log.Error("Connection NOT verified!")
		return
	}

	// request version info
	i.sendResponse("version")

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
			cols := strings.Split(msgText, ",")
			if len(cols) == 3 {
				if len(routeCommand) > 0 {
					cmdStr := strings.ToLower(cols[2])

					submaches := mesLogRouteAddCmdRegexp.FindStringSubmatch(cmdStr)
					if len(submaches) >= 1 {
						i.addRouteAddCommand(submaches[0])
					} else {
						// OpenVPN >= 4.5:
						// Routing log format was changed since OpenVPN 4.5
						// LOG:1607410951,,net_route_v4_add: 193.203.48.54/32 via 192.168.1.1 dev [NULL] table 0 metric -1
						submaches := mesLogRouteAddCmdRegexpOvpn45.FindStringSubmatch(cmdStr)
						if len(submaches) >= 2 {
							i.addRouteAddCommand(fmt.Sprint(routeCommand, " add ", submaches[1]))
						}
					}
				}
			} else {
				// LOG:1586341059,,PUSH: Received control message: 'PUSH_REPLY,redirect-gateway def1,explicit-exit-notify 3,comp-lzo no,route-gateway 10.34.44.1,topology subnet,ping 10,ping-restart 60,dhcp-option DNS 10.34.44.1,ifconfig 10.34.44.19 255.255.252.0,peer-id 17,cipher AES-256-GCM'
				cols := mesLogPushReplyCmdRegexp.FindStringSubmatch(msgText)
				if len(cols) == 2 {
					i.onPushReplyCommands(strings.Split(cols[1], ","))
				}
			}

		case "INFO":

		case "HOLD":
			i.sendResponse("state on", "log on", "hold off", "hold release")

		case "PASSWORD":
			if strings.HasPrefix(msgText, "Verification Failed: 'Auth'") {
				// Authentication error is handled by state: >STATE:1563526742,EXITING,auth-failure,,,,,
				break
			}

			if !mesNeedPassRegexp.Match([]byte(msgText)) {
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

			state, err := vpn.ParseState(stateStr)
			if err != nil {
				i.log.Error("Unable to parse VPN state:", err.Error())
			} else {
				i.log.Info("State changed:", state)

				var clientIP net.IP
				var serverIP net.IP
				var isAuthError bool
				var additionalInfo string

				// If state is Connected - save local and server IP addresses
				if state == vpn.CONNECTED {
					if len(params) > 3 {
						clientIP = net.ParseIP(strings.TrimSpace(params[3]))
					}
					if len(params) > 4 {
						serverIP = net.ParseIP(strings.TrimSpace(params[4]))
					}

				} else if state == vpn.EXITING {
					//>STATE:1563526742,EXITING,auth-failure,,,,,
					if strings.Contains(msgText, "auth-failure") { //if (params[2] == "auth-failure")
						isAuthError = true
					}
				} else if state == vpn.RECONNECTING {
					if len(params) > 2 && len(params[2]) >= 3 {
						additionalInfo = params[2]
					}
				}

				// erase old routing commands
				if state == vpn.RECONNECTING {
					i.eraseRouteAddCommands()
				}

				// save current state info
				state := vpn.StateInfo{
					State:               state,
					Description:         msgText,
					ClientIP:            clientIP,
					ServerIP:            serverIP,
					IsAuthError:         isAuthError,
					StateAdditionalInfo: additionalInfo}

				select {
				case i.stateChan <- state: // notify: state was changed
				default:
					i.log.Debug("State channel is full. Waiting...")
					i.stateChan <- state
				}
			}
		}

	}
}
func (i *ManagementInterface) onPushReplyCommands(cmds []string) {
	// LOG:1586341059,,PUSH: Received control message: 'PUSH_REPLY,redirect-gateway def1,explicit-exit-notify 3,comp-lzo no,route-gateway 10.34.44.1,topology subnet,ping 10,ping-restart 60,dhcp-option DNS 10.34.44.1,ifconfig 10.34.44.19 255.255.252.0,peer-id 17,cipher AES-256-GCM'
	var dns net.IP = nil
	for idx, cmd := range cmds {
		cmd = strings.ToLower(strings.TrimSpace(cmd))
		cmds[idx] = cmd

		// dhcp-option DNS 10.34.44.1
		if strings.HasPrefix(cmd, "dhcp-option dns ") {
			if cols := strings.Split(cmd, " "); len(cols) == 3 {
				dns = net.ParseIP(cols[2])
				if dns == nil {
					i.log.Warning("Unable to parse pushed DNS: ", cols[2])
				} else {
					i.log.Info("DNS pushed: ", dns)
				}
			}
		}
	}
	i.pushReplyDNS = dns
	i.pushReplyCmds = cmds
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

	// Only one command allowed to send
	// This avoids the MI commands injection possibility (for the situations when we are controlling only command prefix)
	command = strings.Split(command, "\n")[0]

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
	// Example: "/sbin/route" - for macOS, "/sbin/ip route" - for Linux, "C:\\Windows\\System32\\ROUTE.EXE" - for Windows
	routeCommand := platform.RouteCommand()

	i.routeAddCmdsMutex.Lock()
	defer i.routeAddCmdsMutex.Unlock()

	if !strings.HasPrefix(strings.ToLower(command), strings.ToLower(routeCommand)) {
		i.log.Warning("Unexpected 'route-add' command: ", command)
		return
	}
	command = strings.TrimSpace(command) // this is reqid

	i.routeAddCmds = append(i.routeAddCmds, command)
	i.log.Debug("New route-add command (", len(i.routeAddCmds), "): ", command)
}

func (i *ManagementInterface) eraseRouteAddCommands() {
	i.routeAddCmdsMutex.Lock()
	defer i.routeAddCmdsMutex.Unlock()
	if len(i.routeAddCmds) > 0 {
		i.log.Info("Forgetting old routing commands")
	}
	i.routeAddCmds = nil
}
