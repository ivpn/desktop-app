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

package wireguard

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app-daemon/netinfo"
	"github.com/ivpn/desktop-app-daemon/service/dns"
	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/shell"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

//TODO: BE CAREFUL! Constant string! (can be changed after WireGuard update)
const (
	strTriggerSuccessInit      string = "UAPI listener started"
	strTriggerInterfaceDown    string = "Interface set down"
	strTriggerAddrAlreadyInUse string = "Address already in use"
)

const subnetMask string = "255.0.0.0"

// internalVariables of wireguard implementation for macOS
type internalVariables struct {
	// WG running process (shell command)
	command       *exec.Cmd
	isGoingToStop bool
	defGateway    net.IP

	isPaused      bool
	omResumedChan chan struct{} // channel for 'On Resume' events
}

func (wg *WireGuard) init() error {
	return nil // do nothing for macOS
}

// connect - SYNCHRONOUSLY execute openvpn process (wait until it finished)
func (wg *WireGuard) connect(stateChan chan<- vpn.StateInfo) (err error) {
	wg.internals.omResumedChan = make(chan struct{}, 1)
	defer func() {
		// The 'Pause' functionality is based on fact that connection will be re-connected by a service
		// if we disconnected without any 'disconnect' request.
		// Therefore, in case of 'pause' we just stopping real connection
		// and waiting for 'resume' command to return control to the owner service.
		if wg.internals.isPaused && !wg.internals.isGoingToStop {
			// waiting to 'resume' event
			<-wg.internals.omResumedChan
			err = &vpn.ReconnectionRequiredError{Err: err}
		}
	}()

	return wg.internalConnect(stateChan)
}

// connect - SYNCHRONOUSLY execute openvpn process (wait until it finished)
func (wg *WireGuard) internalConnect(stateChan chan<- vpn.StateInfo) error {

	var routineStopWaiter sync.WaitGroup

	// if we are trying to connect when no connectivity (WiFi off?) -
	// waiting until network appears
	// Retry to check each 5 seconds (sending RECONNECTING event)
	for wg.internals.isGoingToStop == false {
		if dns.IsPrimaryInterfaceFound() == true {
			break
		}
		log.Info(fmt.Sprintf("No connectivity. Waiting 5 sec to retry..."))

		stateChan <- vpn.NewStateInfo(vpn.RECONNECTING, "No connectivity")
		pauseEnd := time.Now().Add(time.Second * 5)
		for time.Now().Before(pauseEnd) && wg.internals.isGoingToStop == false {
			time.Sleep(time.Millisecond * 50)
		}
	}

	// get default Gateway IP
	defaultGwIP, err := netinfo.DefaultGatewayIP()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to detect default getway: %s", err))
		return err
	}
	wg.internals.defGateway = defaultGwIP

	if wg.internals.isGoingToStop {
		return nil
	}

	defer func() {
		wg.removeRoutes()
		wg.removeDNS()

		// wait to stop all routines
		routineStopWaiter.Wait()

		log.Info("Stopped")
	}()

	utunName, err := getFreeTunInterfaceName()
	if err != nil {
		log.Error(err.Error())
		return fmt.Errorf("unable to start WireGuard. Failed to obtain free utun interface: %w", err)
	}

	log.Info("Starting WireGuard in interface ", utunName)
	wg.internals.command = exec.Command(wg.binaryPath, "-f", utunName)

	isStartedChannel := make(chan bool)

	// output reader
	outPipe, err := wg.internals.command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Failed to start WireGuard: %w", err)
	}
	outPipeScanner := bufio.NewScanner(outPipe)
	routineStopWaiter.Add(1)
	go func() {
		defer routineStopWaiter.Done()

		isWaitingToStart := true
		for outPipeScanner.Scan() {
			text := outPipeScanner.Text()
			log.Info("[out] ", text)

			if isWaitingToStart && strings.Contains(text, strTriggerSuccessInit) {
				isWaitingToStart = false
				isStartedChannel <- true
			}

			// todo
			if strings.Contains(text, strTriggerInterfaceDown) {
				// TODO: detecting if an interface was down. It can happen, for example, when another tunnel was initialized (e.g. separate OpenVPN connection)
				// Normally, we no not need it. netchange.Detector checking for routing changes (implemented on service level)
			}
		}
	}()

	// error reader
	errPipe, err := wg.internals.command.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to start WireGuard: %w", err)
	}
	errPipeScanner := bufio.NewScanner(errPipe)
	routineStopWaiter.Add(1)
	go func() {
		defer routineStopWaiter.Done()

		for errPipeScanner.Scan() {
			log.Info("[err] ", errPipeScanner.Text())
		}
	}()

	// start
	if err := wg.internals.command.Start(); err != nil {
		log.Error(err.Error())
		return fmt.Errorf("failed to start WireGuard process: %w", err)
	}

	// waiting to start and initialize
	routineStopWaiter.Add(1)
	go func() {
		defer routineStopWaiter.Done()
		isHaveToBeStopped := false

		select {
		case <-isStartedChannel:
			// Process started. Perform initialization...
			if err := wg.initialize(utunName); err != nil {
				// TODO: REWORK - return initialization error as a result of connect
				log.ErrorTrace(err)
				isHaveToBeStopped = true
			} else {
				log.Info("Started")
				// CONNECTED
				wg.notifyConnectedStat(stateChan)
			}

		case <-time.After(time.Second * 5):
			// stop process if WG not successfully started during 5 sec
			log.Error("Start timeout.")
			isHaveToBeStopped = true
		}

		if isHaveToBeStopped {
			log.Error("Stopping process manually...")
			if err := wg.disconnect(); err != nil {
				log.Error("Failed to stop process: ", err)
			}
		}
	}()

	if wg.internals.isGoingToStop == true {
		wg.disconnect()
	}

	if err := wg.internals.command.Wait(); err != nil {
		// error will be received anyway. We are logging it only if process was stopped unexpectable
		if wg.internals.isGoingToStop == false {
			log.Error(err.Error())
			return fmt.Errorf("WireGuard prosess error: %w", err)
		}
	}
	return nil
}

func (wg *WireGuard) disconnect() error {
	wg.internals.isGoingToStop = true
	log.Info("Stopping")
	wg.resume()
	return wg.internalDisconnect()
}

func (wg *WireGuard) internalDisconnect() error {
	cmd := wg.internals.command

	// ProcessState contains information about an exited process,
	// available after a call to Wait or Run.
	// NOT nil = process finished
	if cmd == nil || cmd.Process == nil || cmd.ProcessState != nil {
		return nil // nothing to stop
	}

	log.Info("Stopping")
	return cmd.Process.Kill()
}

func (wg *WireGuard) isPaused() bool {
	return wg.internals.isPaused
}

func (wg *WireGuard) pause() error {
	wg.internals.isPaused = true
	return wg.internalDisconnect()
}

func (wg *WireGuard) resume() error {
	// send 'resumed' event
	resumeCh := wg.internals.omResumedChan
	if resumeCh != nil {
		select {
		case resumeCh <- struct{}{}:
		default:
		}
	}

	wg.internals.isPaused = false
	return nil
}

func (wg *WireGuard) setManualDNS(addr net.IP) error {
	return dns.SetManual(addr, nil)
}

func (wg *WireGuard) resetManualDNS() error {
	return dns.DeleteManual(nil)
}

func (wg *WireGuard) initialize(utunName string) error {
	if err := wg.initializeConfiguration(utunName); err != nil {
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	if err := wg.setRoutes(); err != nil {
		return fmt.Errorf("failed to set routes: %w", err)
	}

	err := wg.setDNS()
	if err != nil {
		return fmt.Errorf("failed to set DNS: %w", err)
	}
	return nil
}

func (wg *WireGuard) initializeConfiguration(utunName string) error {
	log.Info("Configuring ", utunName, " interface...")

	// Configure WireGuard interface
	// example command:	ipconfig set utun7 MANUAL 10.0.0.121 255.255.255.0
	//				 	ifconfig utun2 inet 172.26.22.146/8 172.26.22.146 alias
	if err := wg.initializeUnunInterface(utunName); err != nil {
		return fmt.Errorf("failed to initialize interface: %w", err)
	}

	// WireGuard configuration
	return wg.setWgConfiguration(utunName)
}

// Configure WireGuard interface
// example command: ipconfig set utun7 MANUAL 10.0.0.121 255.255.255.0
func (wg *WireGuard) initializeUnunInterface(utunName string) error {
	return shell.Exec(log, "/usr/sbin/ipconfig", "set", utunName, "MANUAL", wg.connectParams.clientLocalIP.String(), subnetMask)
}

// WireGuard configuration
func (wg *WireGuard) setWgConfiguration(utunName string) error {
	// do not forget to remove config file after finishing configuration
	defer os.Remove(wg.configFilePath)

	for retries := 0; ; retries++ {
		// few retries if local port is already in use
		if retries >= 5 {
			// not more than 5 retries
			return fmt.Errorf("failed to set wireguard configuration")
		}

		// generate configuration
		err := wg.generateAndSaveConfigFile(wg.configFilePath)
		if err != nil {
			return fmt.Errorf("failed to save WG config file: %w", err)
		}

		// define output processing function
		isPortInUse := false
		errParse := func(text string, isError bool) {
			if isError {
				log.Debug("[wgconf error] ", text)
			} else {
				log.Debug("[wgconf out] ", text)
			}
			if strings.Contains(text, strTriggerAddrAlreadyInUse) {
				isPortInUse = true
			}
		}

		// Configure WireGuard
		// example command: wg setconf utun7 wireguard.conf
		err = shell.ExecAndProcessOutput(log, errParse, "", wg.toolBinaryPath,
			"setconf", utunName, wg.configFilePath)

		if isPortInUse == false {
			return err
		}
	}
}

func (wg *WireGuard) setRoutes() error {
	log.Info("Modifying routing table...")

	if net.IPv4(127, 0, 0, 1).Equal(wg.connectParams.hostIP) {
		return fmt.Errorf("WG server IP error (unable to use '127.0.0.1' as WG server IP)")
	}

	// Update main route
	// example command:	route	-n	add	-net	0/1			10.0.0.1
	// 					route	-n	add	-inet	0.0.0.0/1	-interface utun2
	if err := shell.Exec(log, "/sbin/route", "-n", "add", "-net", "0/1", wg.connectParams.hostLocalIP.String()); err != nil {
		return fmt.Errorf("adding route shell comand error : %w", err)
	}

	// Update routing to remote server (remote_server default_router 255.255.255)
	// example command:	route	-n	add	-net	145.239.239.55	192.168.1.1	255.255.255.255
	//					route	-n	add	-inet	51.77.91.106	-gateway	192.168.1.1
	if err := shell.Exec(log, "/sbin/route", "-n", "add", "-net", wg.connectParams.hostIP.String(), wg.internals.defGateway.String(), "255.255.255.255"); err != nil {
		return fmt.Errorf("adding route shell comand error : %w", err)
	}

	// Update routing table
	// example command:	route	-n	add	-net	128.0.0.0	10.0.0.1	128.0.0.0
	// 					route	-n	add	-inet	128.0.0.0/1	-interface	utun2
	if err := shell.Exec(log, "/sbin/route", "-n", "add", "-net", "128.0.0.0", wg.connectParams.hostLocalIP.String(), "128.0.0.0"); err != nil {
		return fmt.Errorf("adding route shell comand error : %w", err)
	}

	return nil
}

func (wg *WireGuard) removeRoutes() error {
	log.Info("Restoring routing table...")

	shell.Exec(log, "/sbin/route", "-n", "delete", "0/1", wg.connectParams.hostLocalIP.String())
	shell.Exec(log, "/sbin/route", "-n", "delete", wg.connectParams.hostIP.String())
	shell.Exec(log, "/sbin/route", "-n", "delete", "-net", "128.0.0.0", wg.connectParams.hostLocalIP.String())

	return nil
}

func (wg *WireGuard) onRoutingChanged() error {
	defGatewayIP, err := netinfo.DefaultGatewayIP()
	if err != nil {
		log.Warning(fmt.Sprintf("onRoutingChanged: %v", err))
		return err
	}

	if defGatewayIP.String() != wg.internals.defGateway.String() {
		log.Info(fmt.Sprintf("Default gateway chaged: %s -> %s. Updating routes...", wg.internals.defGateway.String(), defGatewayIP.String()))
		wg.internals.defGateway = defGatewayIP
		wg.removeRoutes()
		wg.setRoutes()
	}

	return nil
}

func (wg *WireGuard) setDNS() error {
	log.Info("Updating DNS server to " + wg.connectParams.hostLocalIP.String() + "...")
	err := shell.Exec(log, platform.DNSScript(), "-up_set_dns", wg.connectParams.hostLocalIP.String())
	if err != nil {
		return fmt.Errorf("failed to change DNS: %w", err)
	}

	return nil
}

func (wg *WireGuard) removeDNS() error {
	log.Info("Restoring DNS server.")
	err := shell.Exec(log, platform.DNSScript(), "-down", wg.connectParams.hostLocalIP.String())
	if err != nil {
		return fmt.Errorf("failed to restore DNS: %w", err)
	}

	return nil
}

func getFreeTunInterfaceName() (string, error) {
	utunNameRegExp := regexp.MustCompile("^utun([0-9])+")

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	maxUtunNo := 0
	for _, ifs := range ifaces {
		strs := utunNameRegExp.FindStringSubmatch(ifs.Name)
		if len(strs) == 2 {
			if utunNo, _ := strconv.Atoi(strs[1]); utunNo > maxUtunNo {
				maxUtunNo = utunNo
			}
		}
	}

	return fmt.Sprintf("utun%d", maxUtunNo+1), nil
}

func (wg *WireGuard) getOSSpecificConfigParams() (interfaceCfg []string, peerCfg []string) {

	// TODO: check if we need it for this platform
	// Same as "0.0.0.0/0" but such type of configuration is disabling internal WireGuard-s Firewall
	// It blocks everything except WireGuard traffic.
	// We need to disable WireGuard-s firewall because we have our own implementation of firewall.
	//  For details, refer to WireGuard-windows sources: tunnel\ifaceconfig.go (enableFirewall(...) method)
	peerCfg = append(peerCfg, "AllowedIPs = 128.0.0.0/1, 0.0.0.0/1")

	return interfaceCfg, peerCfg
}
