package wireguard

import (
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ivpn/desktop-app-daemon/shell"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

// internalVariables of wireguard implementation for Linux
type internalVariables struct {
}

func (wg *WireGuard) init() error {
	return nil
}

// connect - SYNCHRONOUSLY execute openvpn process (wait untill it finished)
func (wg *WireGuard) connect(stateChan chan<- vpn.StateInfo) error {
	// do not forget to remove config file after finishing configuration
	defer os.Remove(wg.configFilePath)
	// generate configuration
	err := wg.generateAndSaveConfigFile(wg.configFilePath)
	if err != nil {
		return fmt.Errorf("failed to save WG config file: %w", err)
	}

	err = shell.Exec(log, wg.binaryPath, "up", wg.configFilePath)
	if err != nil {
		return fmt.Errorf("failed to start WireGuard: %w", err)
	}

	// notify connected
	stateChan <- vpn.NewStateInfoConnected(wg.connectParams.clientLocalIP, wg.connectParams.hostIP)

	wgInterfaceName := filepath.Base(wg.configFilePath)
	wgInterfaceName = strings.TrimSuffix(wgInterfaceName, path.Ext(wgInterfaceName))
	// wait until wireguard interface is available
	for {
		time.Sleep(time.Microsecond * 100)
		i, err := net.InterfaceByName(wgInterfaceName)
		if err != nil {
			fmt.Println(err)
			break
		}
		if i == nil {
			break
		}
	}

	return nil
}

func (wg *WireGuard) disconnect() error {
	err := shell.Exec(log, wg.binaryPath, "down", wg.configFilePath)
	if err != nil {
		return fmt.Errorf("failed to stop WireGuard: %w", err)
	}
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

func (wg *WireGuard) getOSSpecificConfigParams() (interfaceCfg []string, peerCfg []string) {
	interfaceCfg = append(interfaceCfg, "Address = "+wg.connectParams.clientLocalIP.String()+"/32")

	peerCfg = append(peerCfg, "AllowedIPs = 0.0.0.0/0")
	return interfaceCfg, peerCfg
}
