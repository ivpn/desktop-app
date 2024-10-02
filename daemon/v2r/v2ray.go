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

package v2r

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/shell"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("v2ray")
	implInit() // platform-specific initialisation
}

type V2RayTransportType int

const (
	None V2RayTransportType = iota
	QUIC V2RayTransportType = iota
	TCP  V2RayTransportType = iota
)

func (t V2RayTransportType) ToString() string {
	switch t {
	case None:
		return ""
	case QUIC:
		return "QUIC"
	case TCP:
		return "TCP"
	default:
		return "unknown"
	}
}

type V2RayWrapper struct {
	binary         string
	tempConfigFile string
	config         *V2RayConfig
	command        *exec.Cmd
	mutex          sync.Mutex

	routeStatusMutex sync.Mutex
	// IP address of the default gateway which was used for static route to V2Ray server
	defaultGeteway net.IP
	// IP address of local interface which is in use for communication with V2Ray server
	// (we use use it to detect changes of the route to V2Ray server)
	localInterfaceIp net.IP
}

// CreateV2RayWrapper - creates new V2RayWrapper object
// Please refer to the v2r.V2RayConfig (in v2r/config.go) struct for more information about the V2Ray data flow and configuration
func CreateV2RayWrapper(binary string, tmpConfigFile string, cfg *V2RayConfig) *V2RayWrapper {
	return &V2RayWrapper{
		binary:         binary,
		tempConfigFile: tmpConfigFile,
		config:         cfg,
	}
}

func (v *V2RayWrapper) GetLocalPort() (port int, isTcp bool, err error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if v.config == nil {
		return 0, false, fmt.Errorf("config is empty")
	}

	p, t := v.config.GetLocalPort()
	return p, t, nil
}

func (v *V2RayWrapper) GetRemoteEndpoint() (host net.IP, port int, err error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.getRemoteEndpoint()
}

func (v *V2RayWrapper) getRemoteEndpoint() (host net.IP, port int, err error) {
	if v.config == nil {
		return nil, 0, fmt.Errorf("config is empty")
	}

	host = net.ParseIP(v.config.Outbounds[0].Settings.Vnext[0].Address)
	port = v.config.Outbounds[0].Settings.Vnext[0].Port

	return host, port, nil
}

func (v *V2RayWrapper) Stop() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.stop()
}

func (v *V2RayWrapper) stop() error {
	cmd := v.command
	if cmd == nil {
		return nil
	}
	err := shell.Kill(cmd)
	if err != nil {
		return err
	}

	// wait to stop
	cmd.Wait()

	return nil
}

func (v *V2RayWrapper) Start(defaultGateway net.IP) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.start(defaultGateway)
}

func (v *V2RayWrapper) Restart() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if err := v.stop(); err != nil {
		return err
	}

	gwIP, _ := v.getSavedServerRoute()
	return v.start(gwIP)
}

// UpdateMainRoute - updates the route to V2Ray server.
// This method must be called when the default route was changed (e.g. chnaged WiFi network)
func (v *V2RayWrapper) UpdateMainRoute(defaultGateway net.IP, force bool) error {
	if defaultGateway == nil {
		return fmt.Errorf("default gateway is empty")
	}

	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.updateMainRoute(defaultGateway, force)
}

func (v *V2RayWrapper) updateMainRoute(defaultGateway net.IP, force bool) error {
	if defaultGateway == nil {
		return fmt.Errorf("default gateway is empty")
	}

	if !force {
		ifChanged := false
		curGwIp, curLocalInterfaceIp := v.getSavedServerRoute()
		if curLocalInterfaceIp != nil {
			lInterfaceIp, err := v.getLocalInterfaceIpInUseToAccessServer()
			if err != nil || !curLocalInterfaceIp.Equal(lInterfaceIp) {
				ifChanged = true
			}
		}

		if !ifChanged && curGwIp != nil && curGwIp.Equal(defaultGateway) {
			return nil
		}
	}

	log.Info("Updating route to V2Ray server...")
	if err := v.deleteMainRoute(); err != nil {
		log.Error(err)
	}
	return v.setMainRoute(defaultGateway)
}

func (v *V2RayWrapper) setMainRoute(defaultGateway net.IP) error {
	if defaultGateway == nil {
		return fmt.Errorf("default gateway is empty")
	}

	if err := v.implSetMainRoute(defaultGateway); err == nil {
		// save IP address of local interface which is in use for communication with V2Ray server
		lIfaceIp, err := v.getLocalInterfaceIpInUseToAccessServer()
		v.saveServerRoute(defaultGateway, lIfaceIp)
		if err != nil {
			return fmt.Errorf("unable to obtain local interface info for V2Ray connection: %w", err)
		}
	}
	return nil
}

func (v *V2RayWrapper) deleteMainRoute() error {
	v.saveServerRoute(nil, nil)
	return v.implDeleteMainRoute()
}

func (v *V2RayWrapper) start(defaultGateway net.IP) (retError error) {
	defer func() {
		if retError != nil {
			log.Error(retError)
			// ensure process is stopped
			if v.command != nil {
				if err := shell.Kill(v.command); err != nil {
					log.Error(fmt.Errorf("error stopping V2Ray process: %w", err))
				}
			}
		}
	}()

	// check if object correctly initialized
	if v.binary == "" {
		return fmt.Errorf("path to binary is empty")
	}
	if v.tempConfigFile == "" {
		return fmt.Errorf("temp config file is empty")
	}
	if v.config == nil {
		return fmt.Errorf("config object is empty")
	}

	// check if config is valid
	if err := v.config.isValid(); err != nil {
		return fmt.Errorf("config is invalid: %w", err)
	}

	// create temp config file
	cfgStr, err := json.Marshal(v.config)
	if err != nil {
		return fmt.Errorf("error marshalling config: %w", err)
	}
	if err := os.WriteFile(v.tempConfigFile, cfgStr, 0600); err != nil {
		return fmt.Errorf("error writing v2Ray config file: %w", err)
	}
	// delete temp config file on exit
	defer os.Remove(v.tempConfigFile)

	// Beatify json data from cgfStr and print it to log
	var prettyJSON bytes.Buffer
	if e := json.Indent(&prettyJSON, cfgStr, "", "\t"); e == nil {
		log.Debug("V2Ray configuration: ", prettyJSON.String())
	}

	// Apply route to remote endpoint
	if err := v.setMainRoute(defaultGateway); err != nil {
		return fmt.Errorf("error applying route to remote V2Ray endpoint: %w", err)
	}
	defer func() {
		// in case of error starting V2Ray process - ensure route is deleted
		if retError != nil {
			v.deleteMainRoute()
		}
	}()

	localPort := 0
	initialised := make(chan struct{}, 1)

	// regexp to parse output and get local port number from it (if any)
	// Example: "... [Info] transport/internet/udp: listening UDP on 0.0.0.0:58683"
	portRegExp := regexp.MustCompile(`^.+\s+\[Info\]\s+transport/internet/((udp)|(tcp)):\s+listening\s+((UDP)|(TCP))\s+on\s+0\.0\.0\.0:([0-9]+)\s*$`)
	outputParseFunc := func(text string, isError bool) {
		if isError {
			log.Info("[ERR] ", text)

			if localPort == 0 {
				// if port not found yet (error occurred before port number was found)
				// send signal to channel to unblock Start() method (v2ray failed to start)
				select {
				case initialised <- struct{}{}:
				default:
				}
			}
		} else {
			log.Info("[OUT] ", text)

			// check for port number
			if localPort == 0 {
				columns := portRegExp.FindStringSubmatch(text)
				if len(columns) > 7 {
					localPort, _ = strconv.Atoi(columns[7])
					if localPort > 0 {
						// port found - send signal to channel to unblock Start() method (v2ray started successfully)
						select {
						case initialised <- struct{}{}:
						default:
						}
					}
				}
			}
		}
	}

	log.Info("Starting V2Ray client")
	v.command = exec.Command(v.binary, "run", "-config", v.tempConfigFile)
	// start reading output
	if err := shell.StartConsoleReaders(v.command, outputParseFunc); err != nil {
		return fmt.Errorf("failed to init process console reader: %w", err)
	}
	// start process
	if err := v.command.Start(); err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}

	configuredPort, configuredPortIsTCP := v.config.GetLocalPort()
	configuredPortStr := fmt.Sprintf("%d:UDP", configuredPort)
	if configuredPortIsTCP {
		configuredPortStr = fmt.Sprintf("%d:TCP", configuredPort)
	}

	// wait for v2ray to start (or timeout)
	select {
	case <-initialised:
		if localPort == 0 {
			return fmt.Errorf("V2Ray start failed (local port %s)", configuredPortStr)
		} else if configuredPort != localPort {
			return fmt.Errorf("V2Ray client started on unexpected local port: %s", configuredPortStr)
		}
	case <-time.After(10 * time.Second):
		return fmt.Errorf("V2Ray start timeout (port %s)", configuredPortStr)
	}

	log.Info(fmt.Sprintf("V2Ray client started (port %s)", configuredPortStr))
	go func() {
		v.command.Wait()
		v.deleteMainRoute() // ensure route is deleted
		log.Info(fmt.Sprintf("V2Ray client stopped (port %s)", configuredPortStr))
	}()
	return nil
}

func (v *V2RayWrapper) isMainRouteLocalInfAddressChanged() bool {
	_, curLocalInterfaceIp := v.getSavedServerRoute()

	if curLocalInterfaceIp == nil {
		return false
	}

	// check: do we need to update route to V2Ray server? (due to it was changed or removed)
	if lInterfaceIp, err := v.getLocalInterfaceIpInUseToAccessServer(); err == nil {
		if !curLocalInterfaceIp.Equal(lInterfaceIp) {
			return true
		}
	}
	return false
}

// Get IP address of local interface which is in use for communication with V2Ray server
// (it depends of routing table)
func (v *V2RayWrapper) getLocalInterfaceIpInUseToAccessServer() (net.IP, error) {
	remoteHost, _, err := v.getRemoteEndpoint()
	if err != nil {
		return nil, err
	}
	return netinfo.GetOutboundIPEx(remoteHost)
}

func (v *V2RayWrapper) getSavedServerRoute() (gateway net.IP, localIp net.IP) {
	v.routeStatusMutex.Lock()
	defer v.routeStatusMutex.Unlock()
	return v.defaultGeteway, v.localInterfaceIp
}
func (v *V2RayWrapper) saveServerRoute(gateway net.IP, localIp net.IP) {
	v.routeStatusMutex.Lock()
	defer v.routeStatusMutex.Unlock()
	v.defaultGeteway = gateway
	v.localInterfaceIp = localIp
}
