//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
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
	"fmt"
	"net"

	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/platform/filerights"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

type platformSpecificProperties struct {
	// no specific properties for Linux implementation
	isCanUseParamsV24 bool
}

func (o *OpenVPN) implInit() error {
	o.psProps.isCanUseParamsV24 = true

	if err := filerights.CheckFileAccessRightsExecutable(o.binaryPath); err != nil {
		return fmt.Errorf("error checking OpenVPN binary file: %w", err)
	}

	// Check OpenVPN minimum version
	minVer := []int{2, 3}
	verNums := GetOpenVPNVersion(o.binaryPath)
	log.Info("OpenVPN version:", verNums)
	for i := range minVer {
		if len(verNums) <= i {
			continue
		}
		if verNums[i] < minVer[i] {
			return fmt.Errorf("OpenVPN version '%v' not supported (minimum required version '%v')", verNums, minVer)
		}
	}
	if len(verNums) >= 2 && verNums[0] == 2 && verNums[1] < 4 {
		o.psProps.isCanUseParamsV24 = false
	}
	return nil
}

func (o *OpenVPN) implIsCanUseParamsV24() bool {
	return o.psProps.isCanUseParamsV24
}

func (o *OpenVPN) implOnConnected() error {
	// TODO: not implemented
	return nil
}

func (o *OpenVPN) implOnDisconnected() error {
	// TODO: not implemented
	return nil
}

func (o *OpenVPN) implOnPause() error {
	return dns.Pause()
}

func (o *OpenVPN) implOnResume() error {
	return dns.Resume(o.getDefaultDNS())
}

func (o *OpenVPN) implOnSetManualDNS(addr net.IP) error {
	return dns.SetManual(addr, nil)
}

func (o *OpenVPN) implOnResetManualDNS() error {
	if o.IsPaused() == false {
		// restore default DNS pushed by OpenVPN server
		defaultDNS := o.getDefaultDNS()
		if defaultDNS != nil {
			return dns.SetManual(defaultDNS, nil)
		}
	}

	return dns.DeleteManual(nil)
}

// getDefaultDNS returns default DNS pushed by OpenVPN server
func (o *OpenVPN) getDefaultDNS() net.IP {
	mi := o.managementInterface
	if mi != nil && mi.isConnected && o.state != vpn.DISCONNECTED && o.state != vpn.EXITING {
		return mi.pushReplyDNS
	}
	return nil
}
