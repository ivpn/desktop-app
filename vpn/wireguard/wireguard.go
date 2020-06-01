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
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/netinfo"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("wg")
}

// ConnectionParams contains all information to make new connection
type ConnectionParams struct {
	clientLocalIP    net.IP
	clientPrivateKey string
	hostPort         int
	hostIP           net.IP
	hostPublicKey    string
	hostLocalIP      net.IP
}

// SetCredentials update WG credentials
func (cp *ConnectionParams) SetCredentials(privateKey string, localIP net.IP) {
	cp.clientPrivateKey = privateKey
	cp.clientLocalIP = localIP
}

// CreateConnectionParams initializing connection parameters object
func CreateConnectionParams(
	hostPort int,
	hostIP net.IP,
	hostPublicKey string,
	hostLocalIP net.IP) ConnectionParams {

	return ConnectionParams{
		hostPort:      hostPort,
		hostIP:        hostIP,
		hostPublicKey: hostPublicKey,
		hostLocalIP:   hostLocalIP}
}

// WireGuard structure represents all data of wireguard connection
type WireGuard struct {
	binaryPath     string
	toolBinaryPath string
	configFilePath string
	connectParams  ConnectionParams

	// Must be implemeted (AND USED) in correspond file for concrete platform. Must contain platform-specified properties (or can be empty struct)
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

// DestinationIPs -  Get destination IPs (VPN host server or proxy server IP address)
// This information if required, for example, to allow this address in firewall
func (wg *WireGuard) DestinationIPs() []net.IP {
	return []net.IP{wg.connectParams.hostIP}
}

// Type just returns VPN type
func (wg *WireGuard) Type() vpn.Type { return vpn.WireGuard }

// Init performs basic initializations before connection
// It is useful, for example:
// 		for WireGuard(Windows) 	- to ensure that WG service is fully uninstalled
//		for WireGuard(macOS) 	- to initialize default gateway IP
//		for OpenVPN(Linux) 		- to ensure that OpenVPN has correct version
func (wg *WireGuard) Init() error {
	return wg.init()
}

// Connect - SYNCHRONOUSLY execute openvpn process (wait untill it finished)
func (wg *WireGuard) Connect(stateChan chan<- vpn.StateInfo) error {

	disconnectDescription := ""

	stateChan <- vpn.NewStateInfo(vpn.CONNECTING, "")
	defer func() {
		stateChan <- vpn.NewStateInfo(vpn.DISCONNECTED, disconnectDescription)
	}()

	err := wg.connect(stateChan)

	if err != nil {
		disconnectDescription = err.Error()
	}

	return err
}

// Disconnect stops the connection
func (wg *WireGuard) Disconnect() error {
	return wg.disconnect()
}

// IsPaused checking if we are in paused state
func (wg *WireGuard) IsPaused() bool {
	return wg.isPaused()
}

// Pause doing required operation for Pause (remporary restoring default DNS)
func (wg *WireGuard) Pause() error {
	return wg.pause()
}

// Resume doing required operation for Resume (restores DNS configuration before Pause)
func (wg *WireGuard) Resume() error {
	return wg.resume()
}

// SetManualDNS changes DNS to manual IP
func (wg *WireGuard) SetManualDNS(addr net.IP) error {
	return wg.setManualDNS(addr)
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

	err = ioutil.WriteFile(cfgFilePath, []byte(configText), 0600)
	if err != nil {
		return fmt.Errorf("failed to save WireGuard configuration into a file: %w", err)
	}

	log.Info("WireGuard  configuration:",
		"\n=====================\n",
		strings.ReplaceAll(configText, wg.connectParams.clientPrivateKey, "***"),
		"\n=====================\n")

	return nil
}

func (wg *WireGuard) generateConfig() ([]string, error) {
	listenPort, err := netinfo.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("unable to obtain free local port: %w", err)
	}

	interfaceCfg := []string{
		"[Interface]",
		"PrivateKey = " + wg.connectParams.clientPrivateKey,
		"ListenPort = " + strconv.Itoa(listenPort)}

	peerCfg := []string{
		"[Peer]",
		"PublicKey = " + wg.connectParams.hostPublicKey,
		"Endpoint = " + wg.connectParams.hostIP.String() + ":" + strconv.Itoa(wg.connectParams.hostPort),
		"PersistentKeepalive = 25"}

	// add some OS-specific configurations (if necessary)
	iCfg, pCgf := wg.getOSSpecificConfigParams()
	interfaceCfg = append(interfaceCfg, iCfg...)
	peerCfg = append(peerCfg, pCgf...)

	return append(interfaceCfg, peerCfg...), nil
}
