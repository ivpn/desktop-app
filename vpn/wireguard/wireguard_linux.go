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
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ivpn/desktop-app-daemon/service/dns"
	"github.com/ivpn/desktop-app-daemon/shell"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

// internalVariables of wireguard implementation for Linux
type internalVariables struct {
	manualDNS net.IP
	isRunning bool
}

func (wg *WireGuard) init() error {
	// stop current WG connection (if exists)
	//
	// ifname := filepath.Base(wg.configFilePath)
	// ifname = strings.TrimSuffix(ifname, path.Ext(ifname))
	// err := shell.Exec(log, "ip", "link", "set", "down", ifname) // command: sudo ip link set down wgivpn
	// if err != nil {
	// 	log.Warning(err)
	// }
	// err = shell.Exec(log, "ip", "link", "delete", ifname) // command: sudo ip link delete wgivpn
	// if err != nil {
	// 	log.Warning(err)
	// }

	return nil
}

// connect - SYNCHRONOUSLY execute openvpn process (wait until it finished)
func (wg *WireGuard) connect(stateChan chan<- vpn.StateInfo) error {

	wg.internals.isRunning = true
	defer func() {
		wg.internals.isRunning = false
		// do not forget to remove config file after finishing configuration
		if err := os.Remove(wg.configFilePath); err != nil {
			log.Warning(fmt.Sprintf("failed to remove WG configuration: %s", err))
		}

		// restore DNS configuration
		if err := dns.DeleteManual(nil); err != nil {
			log.Warning(fmt.Sprintf("failed to restore DNS configuration: %s", err))
		}
	}()

	// generate configuration
	err := wg.generateAndSaveConfigFile(wg.configFilePath)
	if err != nil {
		return fmt.Errorf("failed to save WG config file: %w", err)
	}

	// update DNS configuration
	if wg.internals.manualDNS == nil {
		if err := dns.SetManual(wg.connectParams.hostLocalIP, nil); err != nil {
			return fmt.Errorf("failed to set DNS: %w", err)
		}
	}

	// start WG
	log.Info("Shell exec: ", wg.binaryPath, " up ", wg.configFilePath)
	cmd := exec.Command(wg.binaryPath, "up", wg.configFilePath)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		if len(outBytes) > 0 {
			log.Error(fmt.Sprintf("'%s' error. Output: %s", wg.binaryPath, string(outBytes)))
		}
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
	// set DNS called outside
	wg.internals.manualDNS = addr
	return dns.SetManual(addr, nil)
}

func (wg *WireGuard) resetManualDNS() error {
	// reset DNS called outside
	wg.internals.manualDNS = nil
	if wg.internals.isRunning {
		// changing DNS to default value for current WireGuard connection
		return dns.SetManual(wg.connectParams.hostLocalIP, nil)
	}
	return dns.DeleteManual(nil)
}

func (wg *WireGuard) getOSSpecificConfigParams() (interfaceCfg []string, peerCfg []string) {
	interfaceCfg = append(interfaceCfg, "Address = "+wg.connectParams.clientLocalIP.String()+"/32")
	interfaceCfg = append(interfaceCfg, "SaveConfig = true")

	peerCfg = append(peerCfg, "AllowedIPs = 0.0.0.0/0")
	return interfaceCfg, peerCfg
}
