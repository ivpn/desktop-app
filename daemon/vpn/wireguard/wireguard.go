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

package wireguard

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("wg")
}

// ConnectionParams contains all information to make new connection
type ConnectionParams struct {
	clientLocalIP        net.IP
	clientPrivateKey     string
	presharedKey         string
	hostPort             int
	hostIP               net.IP
	hostPublicKey        string
	hostLocalIP          net.IP
	ipv6Prefix           string
	multihopExitHostname string // (e.g.: "nl4.wg.ivpn.net") we need it only for informing clients about connection status
	mtu                  int    // Set 0 to use default MTU value
}

func (cp *ConnectionParams) GetIPv6ClientLocalIP() net.IP {
	if len(cp.ipv6Prefix) <= 0 {
		return nil
	}
	return net.ParseIP(cp.ipv6Prefix + cp.clientLocalIP.String())
}
func (cp *ConnectionParams) GetIPv6HostLocalIP() net.IP {
	if len(cp.ipv6Prefix) <= 0 {
		return nil
	}
	return net.ParseIP(cp.ipv6Prefix + cp.hostLocalIP.String())
}

// SetCredentials update WG credentials
func (cp *ConnectionParams) SetCredentials(privateKey string, presharedKey string, localIP net.IP) {
	cp.clientPrivateKey = privateKey
	cp.presharedKey = presharedKey
	cp.clientLocalIP = localIP
}

// CreateConnectionParams initializing connection parameters object
func CreateConnectionParams(
	multihopExitHostName string,
	hostPort int,
	hostIP net.IP,
	hostPublicKey string,
	hostLocalIP net.IP,
	ipv6Prefix string,
	mtu int) ConnectionParams {

	return ConnectionParams{
		multihopExitHostname: multihopExitHostName,
		hostPort:             hostPort,
		hostIP:               hostIP,
		hostPublicKey:        hostPublicKey,
		hostLocalIP:          hostLocalIP,
		ipv6Prefix:           ipv6Prefix,
		mtu:                  mtu,
	}
}

// WireGuard structure represents all data of wireguard connection
type WireGuard struct {
	binaryPath     string
	toolBinaryPath string
	configFilePath string
	connectParams  ConnectionParams
	localPort      int

	isDisconnected        bool
	isDisconnectRequested bool

	// Guards healthChecker: it is (re)started on every CONNECTED notification and stopped on
	// RECONNECTING/DISCONNECTED/Disconnect, so its lifetime always matches an actual CONNECTED period.
	// A fresh *healthChecker is created per start, so a goroutine's self-cleanup (see startHealthCheck)
	// can never clear a later generation's checker.
	healthCheckerMutex sync.Mutex
	healthChecker      *healthChecker

	// Must be implemented (AND USED) in correspond file for concrete platform. Must contain platform-specified properties (or can be empty struct)
	internals internalVariables
}

// NewWireGuardObject creates new wireguard structure
func NewWireGuardObject(wgBinaryPath string, wgToolBinaryPath string, wgConfigFilePath string, connectionParams ConnectionParams) (*WireGuard, error) {
	if connectionParams.clientLocalIP == nil || len(connectionParams.clientPrivateKey) == 0 {
		return nil, fmt.Errorf("WireGuard local credentials not defined")
	}

	return &WireGuard{
		binaryPath:     wgBinaryPath,
		toolBinaryPath: wgToolBinaryPath,
		configFilePath: wgConfigFilePath,
		connectParams:  connectionParams}, nil
}

func (wg *WireGuard) GetTunnelName() string {
	return wg.getTunnelName()
}

// Destination -  Get destination address (VPN host server or proxy server IP address)
// This information if required, for example, to allow this address in firewall
func (wg *WireGuard) Destination() (addr net.IP, port uint16, isTcp bool) {
	return wg.connectParams.hostIP, uint16(wg.connectParams.hostPort), false
}
func (wg *WireGuard) DefaultDNS() net.IP {
	if wg.isDisconnected {
		return nil
	}

	return wg.connectParams.hostLocalIP
}

// Type just returns VPN type
func (wg *WireGuard) Type() vpn.Type { return vpn.WireGuard }

// Init performs basic initializations before connection
// It is useful, for example:
//   - for WireGuard(Windows) - to ensure that WG service is fully uninstalled
//   - for OpenVPN(Linux) - to ensure that OpenVPN has correct version
func (wg *WireGuard) Init() error {
	return wg.init()
}

// Connect - SYNCHRONOUSLY execute openvpn process (wait until it finished)
func (wg *WireGuard) Connect(stateChan chan<- vpn.StateInfo) error {

	disconnectDescription := ""
	wg.isDisconnected = false
	wg.isDisconnectRequested = false
	stateChan <- vpn.NewStateInfo(vpn.CONNECTING, "")
	defer func() {
		wg.stopHealthCheck()
		wg.isDisconnected = true
		stateChan <- vpn.NewStateInfo(vpn.DISCONNECTED, disconnectDescription)
	}()

	err := func() error {
		// Check custom MTU value
		if wg.connectParams.mtu > 0 {
			// According to Windows specification: "... For IPv4 the minimum value is 576 bytes. For IPv6 the minimum is value is 1280 bytes... "
			// Using the same limitations for all platforms
			if wg.connectParams.mtu < 1280 || wg.connectParams.mtu > 65535 {
				return fmt.Errorf("bad MTU value (acceptable interval is: [1280 - 65535])")
			}
		}

		return wg.connect(stateChan)
	}()

	if err != nil {
		disconnectDescription = err.Error()
	}

	return err
}

// Disconnect stops the connection
func (wg *WireGuard) Disconnect() error {
	wg.isDisconnectRequested = true
	wg.stopHealthCheck()
	return wg.disconnect()
}

// IsPaused checking if we are in paused state
func (wg *WireGuard) IsPaused() bool {
	return wg.isPaused()
}

// Pause doing required operation for Pause (temporary restoring default DNS)
func (wg *WireGuard) Pause() error {
	// IMPORTANT! When the WG keys regenerated (see service.WireGuardSaveNewKeys()):
	// WireGuard 'pause/resume' state is based on complete VPN disconnection and restoring connection back (on all platforms)
	// If this will be changed (e.g. just changing routing) - it will be necessary to implement reconnection even in 'pause' state (when keys were regenerated)
	if ret := wg.pause(); ret != nil {
		return ret
	}

	// make this method synchronous: waiting until paused (until WG connection disappear)
	return <-WaitForDisconnectChan(wg.GetTunnelName(), []*bool{&wg.isDisconnectRequested, &wg.isDisconnected})
}

// Resume doing required operation for Resume (restores DNS configuration before Pause)
func (wg *WireGuard) Resume() error {
	if ret := wg.resume(); ret != nil {
		return ret
	}

	// make this method synchronous: waiting until paused (until WG connection disappear)
	return <-WaitForConnectChan(wg.GetTunnelName(), []*bool{&wg.isDisconnectRequested, &wg.isDisconnected})
}

// SetManualDNS changes DNS to manual IP
func (wg *WireGuard) SetManualDNS(dnsCfg dns.DnsSettings) error {
	return wg.setManualDNS(dnsCfg)
}

// ResetManualDNS restores DNS
func (wg *WireGuard) ResetManualDNS() error {
	return wg.resetManualDNS()
}

func (wg *WireGuard) generateAndSaveConfigFile(cfgFilePath string) error {
	cfg, err := wg.generateConfig()
	if err != nil {
		return fmt.Errorf("failed to generate WireGuard configuration: %w", err)
	}

	// write configuration into temporary file
	configText := strings.Join(cfg, "\n")

	err = os.WriteFile(cfgFilePath, []byte(configText), 0600)
	if err != nil {
		return fmt.Errorf("failed to save WireGuard configuration into a file: %w", err)
	}

	configToLog := strings.ReplaceAll(configText, wg.connectParams.clientPrivateKey, "***")
	if len(wg.connectParams.presharedKey) > 0 {
		configToLog = strings.ReplaceAll(configToLog, wg.connectParams.presharedKey, "***")
	}
	log.Info("WireGuard  configuration:",
		"\n=====================\n",
		configToLog,
		"\n=====================\n")

	return nil
}

func (wg *WireGuard) generateConfig() ([]string, error) {
	localPort, err := netinfo.GetFreeUDPPort()
	if err != nil {
		return nil, fmt.Errorf("unable to obtain free local port: %w", err)
	}

	wg.localPort = localPort

	// prevent user-defined data injection: ensure that nothing except the base64 public key will be stored in the configuration
	if !helpers.ValidateBase64(wg.connectParams.hostPublicKey) {
		return nil, fmt.Errorf("WG public key is not base64 string")
	}
	if !helpers.ValidateBase64(wg.connectParams.clientPrivateKey) {
		return nil, fmt.Errorf("WG private key is not base64 string")
	}
	if len(wg.connectParams.presharedKey) > 0 && !helpers.ValidateBase64(wg.connectParams.presharedKey) {
		return nil, fmt.Errorf("WG PresharedKey is not base64 string")
	}

	interfaceCfg := []string{
		"[Interface]",
		"PrivateKey = " + wg.connectParams.clientPrivateKey,
		"ListenPort = " + strconv.Itoa(wg.localPort)}

	peerCfg := []string{
		"[Peer]",
		"PublicKey = " + wg.connectParams.hostPublicKey,
		"Endpoint = " + wg.connectParams.hostIP.String() + ":" + strconv.Itoa(wg.connectParams.hostPort),
		"PersistentKeepalive = 25"}

	if len(wg.connectParams.presharedKey) > 0 {
		peerCfg = append(peerCfg, "PresharedKey = "+wg.connectParams.presharedKey)
	}
	// add some OS-specific configurations (if necessary)
	iCfg, pCgf := wg.getOSSpecificConfigParams()
	interfaceCfg = append(interfaceCfg, iCfg...)
	peerCfg = append(peerCfg, pCgf...)

	return append(interfaceCfg, peerCfg...), nil
}

func (wg *WireGuard) waitHandshakeAndNotifyConnected(stateChan chan<- vpn.StateInfo) error {
	log.Info("Initialised")

	// Notify: interface initialised
	wg.notifyInitialisedStat(stateChan)

	// Check connectivity: wait for first handshake
	// function returns only when handshake received or wg.isDisconnectRequested == true
	err := <-WaitForWireguardFirstHanshakeChan(wg.GetTunnelName(), []*bool{&wg.isDisconnectRequested, &wg.isDisconnected}, func(mes string) { log.Info(mes) })
	if err != nil {
		return err
	}

	if !wg.isDisconnectRequested && !wg.isDisconnected {
		log.Info("Connected")
		wg.notifyConnectedStat(stateChan)
	}

	return nil
}

func (wg *WireGuard) newStateInfoConnected() vpn.StateInfo {
	const isTCP = false

	si := vpn.NewStateInfoConnected(
		isTCP,
		wg.connectParams.clientLocalIP,
		wg.connectParams.GetIPv6ClientLocalIP(),
		wg.localPort,
		wg.connectParams.hostIP,
		wg.connectParams.hostPort,
		wg.connectParams.mtu)

	si.ExitHostname = wg.connectParams.multihopExitHostname
	return si
}

func (wg *WireGuard) newStateInfoConnectedUnhealthy() vpn.StateInfo {
	si := wg.newStateInfoConnected()
	si.IsUnhealthy = true
	return si
}

func (wg *WireGuard) notifyConnectedStat(stateChan chan<- vpn.StateInfo) {
	wg.startHealthCheck(stateChan)
	stateChan <- wg.newStateInfoConnected()
}

func (wg *WireGuard) notifyReconnectingStat(stateChan chan<- vpn.StateInfo, description string) {
	wg.stopHealthCheck()
	stateChan <- vpn.NewStateInfo(vpn.RECONNECTING, description)
}

// startHealthCheck (re)starts the connection status checker. Its lifetime is scoped to the
// current CONNECTED period: it is stopped via stopHealthCheck() on every RECONNECTING/DISCONNECTED
// transition, so it never reports health for a tunnel that isn't actually up, and each restart
// begins with fresh RX/TX counters instead of carrying stale values across a reconnect.
func (wg *WireGuard) startHealthCheck(stateChan chan<- vpn.StateInfo) {
	wg.healthCheckerMutex.Lock()
	defer wg.healthCheckerMutex.Unlock()

	if wg.healthChecker != nil {
		return // already running
	}

	c := newHealthChecker(wg.GetTunnelName(), wg.connectParams.hostLocalIP, wg.connectParams.clientLocalIP)
	wg.healthChecker = c

	go func() {
		err := c.run(func(ctx context.Context, isHealthy bool) {
			var state vpn.StateInfo
			if isHealthy {
				log.Info("Connection health restored to healthy")
				state = wg.newStateInfoConnected()
			} else {
				log.Warning("Connection health: unhealthy - tunnel appears dead")
				state = wg.newStateInfoConnectedUnhealthy()
			}
			select {
			case stateChan <- state:
			case <-ctx.Done():
			}
		})

		// Self-cleanup so a future startHealthCheck() call can actually restart monitoring.
		// Only clear wg.healthChecker if it still points to this goroutine's own generation -
		// a newer start/stop cycle may have already replaced it.
		wg.healthCheckerMutex.Lock()
		if wg.healthChecker == c {
			wg.healthChecker = nil
		}
		wg.healthCheckerMutex.Unlock()

		if err != nil {
			log.Error("Connection health checker stopped unexpectedly: ", err)
		}
	}()
}

// stopHealthCheck stops the connection status checker started by startHealthCheck, if running.
func (wg *WireGuard) stopHealthCheck() {
	wg.healthCheckerMutex.Lock()
	defer wg.healthCheckerMutex.Unlock()

	if wg.healthChecker != nil {
		wg.healthChecker.Stop()
		wg.healthChecker = nil
	}
}

func (wg *WireGuard) notifyInitialisedStat(stateChan chan<- vpn.StateInfo) {
	si := wg.newStateInfoConnected()
	si.State = vpn.INITIALISED
	stateChan <- si
}

func (wg *WireGuard) OnRoutingChanged() error {
	return wg.onRoutingChanged()
}

func (wg *WireGuard) IsIPv6InTunnel() bool {
	return len(wg.connectParams.GetIPv6ClientLocalIP()) > 0
}

func (wg *WireGuard) IsReconnectRequiredOnRoutingChange() bool {
	return wg.isReconnectRequiredOnRoutingChange()
}

// If VPN changes "default" route, this function returns gateway IP address of the modified "default route",
// otherwise - nil (system default route keeps unchanged for this VPN connection)
func (wg *WireGuard) DefaultRouteGatewayIP() net.IP {
	return wg.defaultRouteGatewayIP()
}
