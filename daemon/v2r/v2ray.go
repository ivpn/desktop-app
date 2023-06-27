//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 Privatus Limited.
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
	"github.com/ivpn/desktop-app/daemon/shell"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("v2ray")
}

type V2RayWrapper struct {
	binary         string
	tempConfigFile string
	config         *V2RayConfig
	command        *exec.Cmd
	mutex          sync.Mutex
}

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

	cmd := v.command
	if cmd == nil {
		return nil
	}
	return shell.Kill(cmd)
}

func (v *V2RayWrapper) Start() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if v.command != nil {
		return fmt.Errorf("v2ray already started")
	}

	return v.start()
}

func (v *V2RayWrapper) start() (retError error) {
	// check if object correctly initialized
	if v.binary == "" {
		return fmt.Errorf("binary is empty")
	}
	if v.tempConfigFile == "" {
		return fmt.Errorf("temp config file is empty")
	}
	if v.config == nil {
		return fmt.Errorf("config is empty")
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

	// TODO: remove debug lines
	// Beatify json data from cgfStr and print it to log
	var prettyJSON bytes.Buffer
	if e := json.Indent(&prettyJSON, cfgStr, "", "\t"); e == nil {
		log.Debug("V2Ray configuration: ", string(prettyJSON.Bytes()))
	}

	// Apply route to remote endpoint
	if err := v.implSetMainRoute(); err != nil {
		return fmt.Errorf("error applying route to remote V2Ray endpoint: %w", err)
	}
	defer func() {
		if retError != nil {
			// in case of error - ensure route is deleted
			v.implDeleteMainRoute()
		}
	}()

	localPort := 0
	initialised := make(chan struct{}, 1)

	// regexp to parse output and get local port number from it (if any)
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
	defer func() {
		if err != nil {
			// in case of error - ensure process is stopped
			shell.Kill(v.command)
		}
	}()

	// start reading output
	if err := shell.StartConsoleReaders(v.command, outputParseFunc); err != nil {
		log.Error("Failed to init command: ", err.Error())
		return err
	}
	// start process
	if err := v.command.Start(); err != nil {
		log.Error("Failed to start client: ", err.Error())
		return err
	}

	configuredPort, configuredPortIsTCP := v.config.GetLocalPort()
	configuredPortStr := fmt.Sprintf("%d:UDP", configuredPort)
	if configuredPortIsTCP {
		configuredPortStr = fmt.Sprintf("%d:TCP", configuredPort)
	}

	// wait for v2ray to start (or timeout)
	var startError error
	select {
	case <-initialised:
		if localPort == 0 {
			startError = fmt.Errorf("V2Ray start failed (port %s)", configuredPortStr)
		} else if configuredPort != localPort {
			startError = fmt.Errorf("V2Ray client started on unexpected port: %s", configuredPortStr)
		}
	case <-time.After(10 * time.Second):
		startError = fmt.Errorf("V2Ray start timeout (port %s)", configuredPortStr)
	}

	if startError != nil {
		v.command.Process.Kill()
		log.Error(startError)
		return startError
	}

	// log when process finished
	go func() {
		v.command.Wait()
		// ensure route is deleted
		v.implDeleteMainRoute()
		log.Info(fmt.Sprintf("V2Ray client stopped (port %s)", configuredPortStr))
	}()
	log.Info(fmt.Sprintf("V2Ray client started (port %s)", configuredPortStr))
	return nil
}
