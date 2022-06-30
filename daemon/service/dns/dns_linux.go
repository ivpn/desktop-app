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

//go:build linux
// +build linux

package dns

import (
	"net"

	"github.com/ivpn/desktop-app/daemon/service/dns/dnscryptproxy"
	"github.com/ivpn/desktop-app/daemon/service/platform"
)

// For reference: DNS configuration in Linux
// 	https://github.com/systemd/systemd/blob/main/docs/RESOLVED-VPNS.md
// 	https://blogs.gnome.org/mcatanzaro/2020/12/17/understanding-systemd-resolved-split-dns-and-vpn-configuration/
func isResolveCtlInUse() bool {
	return len(platform.ResolvectlBinPath()) > 0
}

var (
	f_implInitialize   func() error
	f_implPause        func(localInterfaceIP net.IP) error
	f_implResume       func(localInterfaceIP net.IP) error
	f_implSetManual    func(dnsCfg DnsSettings, localInterfaceIP net.IP) (dnsInfoForFirewall DnsSettings, retErr error)
	f_implDeleteManual func(localInterfaceIP net.IP) error
)

var (
	isPaused  bool = false
	manualDNS DnsSettings
)

// implInitialize doing initialization stuff (called on application start)
func implInitialize() error {
	if isResolveCtlInUse() {
		f_implInitialize = rctl_implInitialize
		f_implPause = rctl_implPause
		f_implResume = rctl_implResume
		f_implSetManual = rctl_implSetManual
		f_implDeleteManual = rctl_implDeleteManual
	} else {
		f_implInitialize = rconf_implInitialize
		f_implPause = rconf_implPause
		f_implResume = rconf_implResume
		f_implSetManual = rconf_implSetManual
		f_implDeleteManual = rconf_implDeleteManual
	}

	return f_implInitialize()
}

func implGetDnsEncryptionAbilities() (dnsOverHttps, dnsOverTls bool, err error) {
	return true, false, nil
}
func implGetPredefinedDnsConfigurations() ([]DnsSettings, error) {
	return []DnsSettings{}, nil
}

func implPause(localInterfaceIP net.IP) error {
	dnscryptproxy.Stop()
	isPaused = true
	return f_implPause(localInterfaceIP)
}

func implResume(defaultDNS DnsSettings, localInterfaceIP net.IP) error {
	isPaused = false

	if !manualDNS.IsEmpty() {
		// set manual DNS (if defined)
		_, err := f_implSetManual(manualDNS, localInterfaceIP)
		return err
	}

	if !defaultDNS.IsEmpty() {
		_, err := f_implSetManual(defaultDNS, localInterfaceIP)
		return err
	}

	return f_implResume(localInterfaceIP)
}

// Set manual DNS.
func implSetManual(dnsCfg DnsSettings, localInterfaceIP net.IP) (dnsInfoForFirewall DnsSettings, retErr error) {
	defer func() {
		if retErr != nil {
			dnscryptproxy.Stop()
		}
	}()

	// keep info about current manual DNS configuration (can be used for pause/resume)
	manualDNS = dnsCfg

	dnscryptproxy.Stop()

	if isPaused {
		// in case of PAUSED state -> just save manualDNS config
		// it will be applied on RESUME
		return DnsSettings{}, nil
	}

	// start encrypted DNS configuration (if required)
	if !dnsCfg.IsEmpty() && dnsCfg.Encryption != EncryptionNone {
		if err := dnscryptProxyProcessStart(dnsCfg); err != nil {
			return DnsSettings{}, err
		}
		// the local DNS must be configured to the dnscrypt-proxy (localhost)
		dnsCfg = DnsSettings{DnsHost: "127.0.0.1"}
	}

	return f_implSetManual(dnsCfg, localInterfaceIP)
}

// DeleteManual - reset manual DNS configuration to default
// 'localInterfaceIP' (obligatory only for Windows implementation) - local IP of VPN interface
func implDeleteManual(localInterfaceIP net.IP) error {
	manualDNS = DnsSettings{}
	dnscryptproxy.Stop()

	if isPaused {
		// in case of PAUSED state -> just save manualDNS config
		// it will be applied on RESUME
		return nil
	}

	return f_implDeleteManual(localInterfaceIP)
}
