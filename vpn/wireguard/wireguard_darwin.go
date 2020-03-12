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

	"github.com/ivpn/desktop-app-daemon/service/dns"
	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/shell"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

//TODO: BE CAREFUL! Constant string! (can be changed after WireGuard update)
const (
	strTriggerSuccessInit     string = "UAPI listener started"
	strTriggerInterfaceDown   string = "Interface set down"
	strTriggerAddrAlredyInUse string = "Address already in use"
)

const subnetMask string = "255.0.0.0"

// internalVariables of wireguard implementation for macOS
type internalVariables struct {
	// WG running process (shell command)
	command       *exec.Cmd
	isGoingToStop bool
	isPaused      bool
	defGateway    net.IP
}

func (wg *WireGuard) init() error {
	defaultGwIP, err := netinfo.DefaultGatewayIP()
	if err != nil {
		return nil, fmt.Errorf("unable to determine default gateway IP: %w", err)
	}
	wg.internals.defGateway = defaultGwIP
	return nil // do nothing for macOS
}

// connect - SYNCHRONOUSLY execute openvpn process (wait untill it finished)
func (wg *WireGuard) connect(stateChan chan<- vpn.StateInfo) error {
	var routineStopWaiter sync.WaitGroup

	defer func() {
		wg.removeRoutes()
		wg.removeDNS()

		// wait to stop all routines
		routineStopWaiter.Wait()

		log.Info("Stopped")
	}()

	if wg.internals.isGoingToStop {
		return fmt.Errorf("disconnection already requested for this object. To make a new connection, please, initialize new one")
	}

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
				// Normally, we no not need it. netchange.Detector checking for routing chnages (implemented on service level)
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
			// Process started. Perform initialisation...
			if err := wg.initialize(utunName); err != nil {
				// TODO: REWORK - return initialization error as a result of connect
				log.ErrorTrace(err)
				isHaveToBeStopped = true
			} else {
				log.Info("Started")
				// CONNECTED
				stateChan <- vpn.NewStateInfoConnected(wg.connectParams.clientLocalIP, wg.connectParams.hostIP)
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

	if err := wg.removeRoutes(); err != nil {
		return fmt.Errorf("failed to remove routes: %w", err)
	}

	if err := dns.Pause(); err != nil {
		return fmt.Errorf("failed to restore DNS: %w", err)
	}
	return nil
}

func (wg *WireGuard) resume() error {
	defer func() {
		wg.internals.isPaused = false
	}()

	if err := wg.setRoutes(); err != nil {
		return fmt.Errorf("failed to set routes: %w", err)
	}

	if err := dns.Resume(); err != nil {
		return fmt.Errorf("failed to set DNS: %w", err)
	}
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
	// example command: ipconfig set utun7 MANUAL 10.0.0.121 255.255.255.0
	if err := wg.initializeUnunInterface(utunName); err != nil {
		return fmt.Errorf("failed to initialize interface: %w", err)
	}

	// WireGuard configuration
	return wg.setWgConfiguration(utunName)
}

// Configure WireGuard interface
// example command: ipconfig set utun7 MANUAL 10.0.0.121 255.255.255.0
func (wg *WireGuard) initializeUnunInterface(utunName string) error {
	return shell.Exec(log, "ipconfig", "set", utunName, "MANUAL", wg.connectParams.clientLocalIP.String(), subnetMask)
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
			if strings.Contains(text, strTriggerAddrAlredyInUse) {
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

	// Update main route
	// example command: sudo route -n add -net 0/1 10.0.0.1
	if err := shell.Exec(log, "route", "-n", "add", "-net", "0/1", wg.connectParams.hostLocalIP.String()); err != nil {
		return fmt.Errorf("adding route shell comand error : %w", err)
	}

	// Update routing to remote server (remote_server default_router 255.255.255)
	// example command: sudo route -n add -net 145.239.239.55 192.168.1.1 255.255.255.255
	if err := shell.Exec(log, "route", "-n", "add", "-net", wg.connectParams.hostIP.String(), wg.internals.defGateway.String(), "255.255.255.255"); err != nil {
		return fmt.Errorf("adding route shell comand error : %w", err)
	}

	// Update routing table
	// example command: sudo route -n add -net 128.0.0.0 10.0.0.1 128.0.0.0
	if err := shell.Exec(log, "route", "-n", "add", "-net", "128.0.0.0", wg.connectParams.hostLocalIP.String(), "128.0.0.0"); err != nil {
		return fmt.Errorf("adding route shell comand error : %w", err)
	}

	return nil
}

func (wg *WireGuard) removeRoutes() error {
	log.Info("Restoring routing table...")

	shell.Exec(log, "route", "-n", "delete", "0/1", wg.connectParams.hostIP.String())
	shell.Exec(log, "route", "-n", "delete", wg.connectParams.hostIP.String())
	shell.Exec(log, "route", "-n", "delete", "-net", "128.0.0.0", wg.connectParams.hostLocalIP.String(), "128.0.0.0")

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
	// We need to disable WireGurd-s firewall because we have our own implementation of firewall.
	//  For details, refer to WireGuard-windows sources: tunnel\ifaceconfig.go (enableFirewall(...) method)
	peerCfg = append(peerCfg, "AllowedIPs = 128.0.0.0/1, 0.0.0.0/1")

	return interfaceCfg, peerCfg
}
