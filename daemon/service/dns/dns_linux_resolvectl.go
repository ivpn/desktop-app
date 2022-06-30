//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2022 Privatus Limited.
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

//go:build linux
// +build linux

package dns

import (
	"fmt"
	"net"

	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/shell"
)

// For reference: DNS configuration in Linux
// 	https://github.com/systemd/systemd/blob/main/docs/RESOLVED-VPNS.md
// 	https://blogs.gnome.org/mcatanzaro/2020/12/17/understanding-systemd-resolved-split-dns-and-vpn-configuration/

func rctl_implInitialize() error {
	return nil
}

func rctl_implPause(localInterfaceIP net.IP) error {
	inf, err := netinfo.InterfaceByIPAddr(localInterfaceIP)
	if err != nil {
		return nil // seems the interface not created. Nothing to resume
	}
	localInterfaceName := inf.Name

	binPath := platform.ResolvectlBinPath()
	err = shell.Exec(log, binPath, "domain", localInterfaceName, "")
	if err != nil {
		return rctl_error(err)
	}
	err = shell.Exec(log, binPath, "default-route", localInterfaceName, "false")
	if err != nil {
		return rctl_error(err)
	}

	return nil
}

func rctl_implResume(localInterfaceIP net.IP) error {
	inf, err := netinfo.InterfaceByIPAddr(localInterfaceIP)
	if err != nil {
		return rctl_error(err)
	}
	localInterfaceName := inf.Name

	binPath := platform.ResolvectlBinPath()
	err = shell.Exec(log, binPath, "domain", localInterfaceName, "~.")
	if err != nil {
		return rctl_error(err)
	}
	err = shell.Exec(log, binPath, "default-route", localInterfaceName, "true")
	if err != nil {
		return rctl_error(err)
	}

	return nil
}

// Set manual DNS.
func rctl_implSetManual(dnsCfg DnsSettings, localInterfaceIP net.IP) (dnsInfoForFirewall DnsSettings, retErr error) {
	//resolvectl domain privacy0 '~.'
	//resolvectl default-route privacy0 true
	//resolvectl dns privacy0 8.8.8.8

	if localInterfaceIP == nil {
		log.Info("'Set DNS' call ignored due to no local address initialized")
		return dnsCfg, nil
	}
	inf, err := netinfo.InterfaceByIPAddr(localInterfaceIP)
	if err != nil {
		return DnsSettings{}, rctl_error(err)
	}
	localInterfaceName := inf.Name

	binPath := platform.ResolvectlBinPath()
	err = shell.Exec(log, binPath, "domain", localInterfaceName, "~.")
	if err != nil {
		return DnsSettings{}, rctl_error(err)
	}
	err = shell.Exec(log, binPath, "default-route", localInterfaceName, "true")
	if err != nil {
		return DnsSettings{}, rctl_error(err)
	}
	err = shell.Exec(log, binPath, "dns", localInterfaceName, dnsCfg.DnsHost)
	if err != nil {
		return DnsSettings{}, rctl_error(err)
	}

	return dnsCfg, nil
}

// DeleteManual - reset manual DNS configuration to default
func rctl_implDeleteManual(localInterfaceIP net.IP) error {
	return rctl_implPause(localInterfaceIP)
}

func rctl_error(err error) error {
	return fmt.Errorf("failed to change DNS configuration: %w", err)
}
