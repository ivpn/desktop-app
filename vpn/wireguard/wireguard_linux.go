package wireguard

import (
	"net"

	"github.com/ivpn/desktop-app-daemon/vpn"
)

// internalVariables of wireguard implementation for Linux
type internalVariables struct {
}

func (wg *WireGuard) init() error {
	// TODO: not implemented
	return nil
}

// connect - SYNCHRONOUSLY execute openvpn process (wait untill it finished)
func (wg *WireGuard) connect(stateChan chan<- vpn.StateInfo) error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) disconnect() error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) isPaused() bool {
	// TODO: not implemented
	return false
}

func (wg *WireGuard) pause() error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) resume() error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) setManualDNS(addr net.IP) error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) resetManualDNS() error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) initialize(utunName string) error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) initializeConfiguration(utunName string) error {
	// TODO: not implemented
	return nil
}

// Configure WireGuard interface
// example command: ipconfig set utun7 MANUAL 10.0.0.121 255.255.255.0
func (wg *WireGuard) initializeUnunInterface(utunName string) error {
	// TODO: not implemented
	return nil
}

// WireGuard configuration
func (wg *WireGuard) setWgConfiguration(utunName string) error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) setRoutes() error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) removeRoutes() error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) setDNS() error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) removeDNS() error {
	// TODO: not implemented
	return nil
}

func (wg *WireGuard) getOSSpecificConfigParams() (interfaceCfg []string, peerCfg []string) {
	// TODO: not implemented
	return interfaceCfg, peerCfg
}
