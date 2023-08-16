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

package dns

import (
	"fmt"
	"net"

	"github.com/ivpn/desktop-app/daemon/service/dns/dnscryptproxy"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/shell"
)

// implInitialize doing initialization stuff (called on application start)
func implInitialize() error {
	return nil
}

func implApplyUserSettings() error {
	return nil // nothing to do here for current platfom
}

func implPause(localInterfaceIP net.IP) error {
	err := shell.Exec(log, platform.DNSScript(), "-pause")
	if err != nil {
		return fmt.Errorf("DNS pause: Failed to change DNS: %w", err)
	}
	return nil
}

// defaultDNS - not in use for darwin platfrom
func implResume(defaultDNS DnsSettings, localInterfaceIP net.IP) error {
	err := shell.Exec(log, platform.DNSScript(), "-resume")
	if err != nil {
		return fmt.Errorf("DNS resume: Failed to change DNS: %w", err)
	}

	return nil
}

func implGetDnsEncryptionAbilities() (dnsOverHttps, dnsOverTls bool, err error) {
	return true, false, nil
}

// Set manual DNS.
// 'localInterfaceIP' - not in use for macOS implementation
func implSetManual(dnsCfg DnsSettings, localInterfaceIP net.IP) (dnsInfoForFirewall DnsSettings, retErr error) {
	defer func() {
		if retErr != nil {
			dnscryptproxy.Stop()
		}
	}()

	dnscryptproxy.Stop()
	// start encrypted DNS configuration (if required)
	if dnsCfg.Encryption != EncryptionNone {
		if err := dnscryptProxyProcessStart(dnsCfg); err != nil {
			return DnsSettings{}, err
		}
		// the local DNS must be configured to the dnscrypt-proxy (localhost)
		dnsCfg = DnsSettings{DnsHost: "127.0.0.1"}
	}

	err := shell.Exec(log, platform.DNSScript(), "-set_alternate_dns", dnsCfg.Ip().String())
	if err != nil {
		return DnsSettings{}, fmt.Errorf("set manual DNS: Failed to change DNS: %w", err)
	}

	return dnsCfg, nil
}

// DeleteManual - reset manual DNS configuration to default (DHCP)
// 'localInterfaceIP' (obligatory only for Windows implementation) - local IP of VPN interface
func implDeleteManual(localInterfaceIP net.IP) error {
	dnscryptproxy.Stop()

	err := shell.Exec(log, platform.DNSScript(), "-delete_alternate_dns")
	if err != nil {
		return fmt.Errorf("reset manual DNS: Failed to change DNS: %w", err)
	}

	return nil
}

func implGetPredefinedDnsConfigurations() ([]DnsSettings, error) {
	return []DnsSettings{}, nil
}

// IsPrimaryInterfaceFound (macOS specific implementation) returns 'true' when networking is available (primary interface is available)
// When no networking available (WiFi off ?) - returns 'false'
// <this method in use by macOS:WireGuard implementation>
func IsPrimaryInterfaceFound() bool {
	err := shell.Exec(log, platform.DNSScript(), "-is_main_interface_detected")
	return err == nil
}

// UpdateDnsIfWrongSettings - ensures that current DNS configuration is correct. If not - it re-apply the required configuration.
// Currently, it is in use for macOS - like a DNS change monitor.
func implUpdateDnsIfWrongSettings() error {
	log.Info("Validating DNS configuration ...")
	err := shell.Exec(log, platform.DNSScript(), "-update")
	if err != nil {
		return fmt.Errorf("the DNS configuration validation error: %w", err)
	}
	return nil
}
