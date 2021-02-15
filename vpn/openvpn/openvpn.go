//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
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

package openvpn

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/netinfo"
	"github.com/ivpn/desktop-app-daemon/obfsproxy"
	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/shell"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("ovpn")
	rand.Seed(time.Now().UnixNano())
}

// GetOpenVPNVersion trying to get openvpn binary version
func GetOpenVPNVersion(ovpnBinary string) []int {
	cmd := exec.Command(ovpnBinary, "--version")
	out, _ := cmd.CombinedOutput()
	if out == nil || len(out) == 0 {
		return nil
	}

	regexp := regexp.MustCompile("(?i)^OpenVPN ([0-9.]*) ")
	columns := regexp.FindStringSubmatch(string(out))
	if len(columns) < 2 {
		return nil
	}
	ver := columns[1]
	if len(ver) == 0 {
		return nil
	}
	verNums := make([]int, 0, 3)
	for _, num := range strings.Split(ver, ".") {
		n, err := strconv.Atoi(num)
		if err != nil {
			return nil
		}
		if len(verNums) == 0 && n == 0 {
			continue
		}
		verNums = append(verNums, n)
	}
	return verNums
}

// OpenVPN structure represents all data of OpenVPN connection
type OpenVPN struct {
	binaryPath      string
	configPath      string
	logFile         string
	isObfsProxy     bool
	extraParameters string // user-defined extra-parameters of OpenVPN configuration
	connectParams   ConnectionParams

	managementInterface *ManagementInterface
	obfsproxy           *obfsproxy.Obfsproxy

	// current VPN state
	state     vpn.State
	clientIP  net.IP // applicable only for 'CONNECTED' state
	localPort int

	// platform-specific properties (for macOS, Windows etc. ...)
	psProps platformSpecificProperties

	// If true - the disconnection requested
	// No connection is possible anymore (to make new connection a new OpenVPN must be initialized).
	// If we are in 'connecting' state - stop
	isDisconnectRequested bool

	// Note: Disconnect() function will wait until VPN fully disconnects
	runningWG sync.WaitGroup

	isPaused bool
}

// NewOpenVpnObject creates new OpenVPN structure
func NewOpenVpnObject(
	binaryPath string,
	configPath string,
	logFile string,
	isObfsProxy bool,
	extraParameters string,
	connectionParams ConnectionParams) (*OpenVPN, error) {

	if len(connectionParams.username) == 0 || len(connectionParams.password) == 0 {
		return nil, fmt.Errorf("OpenVPN user credentials not defined")
	}

	return &OpenVPN{
			state:           vpn.DISCONNECTED,
			binaryPath:      binaryPath,
			configPath:      configPath,
			logFile:         logFile,
			isObfsProxy:     isObfsProxy,
			extraParameters: extraParameters,
			connectParams:   connectionParams},
		nil
}

// DestinationIP -  Get destination IPs (VPN host server or proxy server IP address)
// This information if required, for example, to allow this address in firewall
func (o *OpenVPN) DestinationIP() net.IP {
	if o.connectParams.proxyAddress != nil {
		return o.connectParams.proxyAddress
	}
	return o.connectParams.hostIP
}

// Type just returns VPN type
func (o *OpenVPN) Type() vpn.Type { return vpn.OpenVPN }

// Init performs basic initializations before connection
// It is useful, for example:
//	- for WireGuard(Windows) - to ensure that WG service is fully uninstalled
//	- for OpenVPN(Linux) - to ensure that OpenVPN has correct version
func (o *OpenVPN) Init() error {
	return o.implInit()
}

// Connect - SYNCHRONOUSLY execute openvpn process (wait until it finished)
func (o *OpenVPN) Connect(stateChan chan<- vpn.StateInfo) (retErr error) {

	// Note: Disconnect() function will wait until VPN fully disconnects
	o.runningWG.Add(1)
	// mark openVPN is fully stopped
	defer o.runningWG.Done()

	if o.isDisconnectRequested {
		return errors.New("disconnection already requested for this OpenVPN object. To make a new connection, please, initialize new one")
	}

	// it allows to wait till all routines finished
	var routinesWaiter sync.WaitGroup
	// marker to stop state-forward routine
	stopStateChan := make(chan struct{})
	// channel will be analyzed for state change. States will be forwarded to channel above ( to 'stateChan')
	internalStateChan := make(chan vpn.StateInfo, 1)

	// EXIT: stopping everything: Management interface, Obfsproxy
	defer func() {

		if retErr != nil {
			log.Error("Connection error: ", retErr)
		}

		// stop state-forward routine
		stopStateChan <- struct{}{}

		mi := o.managementInterface
		if mi != nil {
			if err := mi.StopManagementInterface(); err != nil {
				log.Error(err)
			}
		}

		obfspxy := o.obfsproxy
		if obfspxy != nil {
			obfspxy.Stop()
		}

		o.obfsproxy = nil

		if err := o.implOnDisconnected(); err != nil {
			log.Error(err)
		}

		// wait till all routines finished
		routinesWaiter.Wait()
	}()

	// analyse and forward state changes
	routinesWaiter.Add(1)
	go func() {
		defer routinesWaiter.Done()

		var stateInf vpn.StateInfo
		for {
			select {
			case stateInf = <-internalStateChan:
				// save current state
				o.state = stateInf.State

				if o.state == vpn.CONNECTED {
					// save exitServerID (in MultiHop)
					stateInf.ExitServerID = o.connectParams.multihopExitSrvID
					// save source and destination port
					stateInf.ClientPort = o.localPort
					stateInf.ServerPort = o.connectParams.hostPort

					// notify about correct local IP in VPN network
					o.clientIP = stateInf.ClientIP

					if o.isObfsProxy {
						// in case of obfsproxy - 'stateInf.ServerIP' returns local IP (IP of obfsproxy 127.0.0.1)
						// We must notify about real remote ServerIP, therefore we modifying this parameter before notifying about successful connection
						stateInf.ServerIP = o.connectParams.hostIP
					}

					o.implOnConnected() // process "on connected" event (if necessary)
				} else {
					o.clientIP = nil
				}

				// forward state
				// Notifying about 'connected' state only after 'o.implOnConnected()'
				// There could be additional stuff to do: e.g. change DNS (in implementation for Windows)
				stateChan <- stateInf

			case <-stopStateChan: // openvpn process stopped
				return // stop goroutine
			}
		}
	}()

	if o.managementInterface != nil {
		return errors.New("unable to connect OpenVPN. Management interface already initialized")
	}

	var err error
	obfsproxyPort := 0
	// start Obfsproxy (if necessary)
	if o.isObfsProxy {
		o.obfsproxy = obfsproxy.CreateObfsproxy(platform.ObfsproxyStartScript())
		if obfsproxyPort, err = o.obfsproxy.Start(); err != nil {
			return errors.New("unable to initialize OpenVPN (obfsproxy not started): " + err.Error())
		}

		// detect obfsproxy ptocess stop
		routinesWaiter.Add(1)
		go func() {
			defer routinesWaiter.Done()

			opxy := o.obfsproxy
			if opxy == nil {
				return
			}

			// wait for obfsproxy stop
			opxy.Wait()
			if o.isDisconnectRequested == false {
				// If obfsproxy stopped unexpectedly - disconnect VPN
				log.Error("Obfsproxy stopped unexpectedly. Disconnecting VPN...")
				o.doDisconnect()
			}
		}()
	}

	// start new management interface
	mi, err := StartManagementInterface(o.connectParams.username, o.connectParams.password, internalStateChan)
	if err != nil {
		return fmt.Errorf("failed to start MI: %w", err)
	}
	o.managementInterface = mi

	if o.isDisconnectRequested {
		// If the disconnection request received immediately after 'connect' request - stop connection after MI is initialized
		log.Info("Connection process cancelled.")
		return nil
	}

	// local port to be used for connection (source port)
	o.localPort, err = netinfo.GetFreePort(o.connectParams.tcp)
	if err != nil {
		return err
	}

	miIP, miPort, err := mi.ListenAddress()
	if err != nil {
		return fmt.Errorf("failed to start MI listener: %w", err)
	}

	// create config file
	err = o.connectParams.WriteConfigFile(
		o.localPort,
		o.configPath,
		miIP, miPort,
		o.logFile,
		obfsproxyPort,
		o.extraParameters,
		o.implIsCanUseParamsV24())

	if err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	// Saving first lines of OpenVPN console output into buffer
	// (can be useful in case of OpenVPN start error to analyze it in a log)
	const maxBufSize int = 512
	strOut := strings.Builder{}
	strErr := strings.Builder{}
	outProcessFunc := func(text string, isError bool) {
		if len(text) == 0 {
			return
		}
		if isError {
			if strErr.Len() > maxBufSize {
				return
			}
			strErr.WriteString(text)
		} else {
			if strOut.Len() > maxBufSize {
				return
			}
			strOut.WriteString(text)
		}
	}

	// SYNCHRONOUSLY execute openvpn process (wait until it finished)
	if err = shell.ExecAndProcessOutput(log, outProcessFunc, "", o.binaryPath, "--config", o.configPath); err != nil {
		if strOut.Len() > 0 {
			log.Info(fmt.Sprintf("OpenVPN start ERROR. Output: %s...", strOut.String()))
		}
		if strErr.Len() > 0 {
			log.Info(fmt.Sprintf("OpenVPN start ERROR. Errors output : %s...", strErr.String()))
		}

		if len(o.extraParameters) > 0 {
			return fmt.Errorf("failed to start OpenVPN process: %w. Please, ensure that user-defined OpenVPN configuration parameters are correct", err)
		}
		return fmt.Errorf("failed to start OpenVPN process: %w", err)
	}

	return nil
}

// Disconnect stops the connection
func (o *OpenVPN) Disconnect() error {

	if err := o.doDisconnect(); err != nil {
		return fmt.Errorf("disconnection error : %w", err)
	}

	// waiting until process is running
	// (ensure all disconnection operations performed (e.g. obfsproxy is stopped, etc. ...))
	o.runningWG.Wait()

	return nil
}

func (o *OpenVPN) doDisconnect() error {

	// there is a chance we are in 'connecting' state, but managementInterface is not defined yet
	// Therefore, we are saving our intention to disconnect
	o.isDisconnectRequested = true

	mi := o.managementInterface
	if mi == nil {
		log.Error("OpenVPN MI is nil")
		return nil // nothing to disconnect
	}

	return mi.SendDisconnect()
}

// Pause doing required operation for Pause (temporary restoring default DNS)
func (o *OpenVPN) Pause() error {
	o.isPaused = true

	mi := o.managementInterface
	if mi == nil {
		return errors.New("OpenVPN MI is nil")
	}

	routeAddCommands := mi.GetRouteAddCommands()
	if len(routeAddCommands) == 0 {
		return errors.New("OpenVPN: no route-add commands detected")
	}

	var retErr error
	for _, cmd := range routeAddCommands {
		delCmd := strings.Replace(cmd, "add", "delete", -1)

		cmdCols := strings.SplitN(delCmd, " ", 2)
		if len(cmdCols) != 2 {
			retErr = errors.New("failed to parse route-change command: " + delCmd)
			log.Error(retErr.Error())
			continue
		}

		arguments := strings.Split(cmdCols[1], " ")
		if err := shell.Exec(log, cmdCols[0], arguments...); err != nil {
			retErr = err
			log.Error(err)
		}
	}

	// OS-specific operation (if required)
	retErr = o.implOnPause()
	if retErr != nil {
		log.ErrorTrace(retErr)
	}

	return retErr
}

// Resume doing required operation for Resume (restores DNS configuration before Pause)
func (o *OpenVPN) Resume() error {
	defer func() {
		o.isPaused = false
	}()

	mi := o.managementInterface
	if mi == nil {
		return errors.New("OpenVPN MI is nil")
	}

	routeAddCommands := mi.GetRouteAddCommands()
	if len(routeAddCommands) == 0 {
		return errors.New("OpenVPN: no route-add commands detected")
	}

	var retErr error
	for _, cmd := range routeAddCommands {
		cmdCols := strings.SplitN(cmd, " ", 2)
		if len(cmdCols) != 2 {
			retErr = errors.New("failed to parse route-change command: " + cmd)
			log.Error(retErr.Error())
			continue
		}

		arguments := strings.Split(cmdCols[1], " ")
		if err := shell.Exec(log, cmdCols[0], arguments...); err != nil {
			retErr = err
			log.Error(err)
		}
	}

	// OS-specific operation (if required)
	retErr = o.implOnResume()
	if retErr != nil {
		log.ErrorTrace(retErr)
	}

	return retErr
}

// IsPaused checking if we are in paused state
func (o *OpenVPN) IsPaused() bool {
	return o.isPaused
}

// SetManualDNS changes DNS to manual IP
func (o *OpenVPN) SetManualDNS(addr net.IP) error {
	return o.implOnSetManualDNS(addr)
}

// ResetManualDNS restores DNS
func (o *OpenVPN) ResetManualDNS() error {
	return o.implOnResetManualDNS()
}
