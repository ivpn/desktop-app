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
	"net/url"

	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/service/dns/dnscryptproxy"
	"github.com/ivpn/desktop-app/daemon/service/platform"
)

type FuncDnsChangeFirewallNotify func(dns *DnsSettings) error
type FuncGetUserSettings func() DnsExtraSettings

type DnsExtraSettings struct {
	// If true - use old style DNS management mechanism
	// by direct modifying file '/etc/resolv.conf'
	Linux_IsDnsMgmtOldStyle bool
}

var (
	log                         *logger.Logger
	lastManualDNS               DnsSettings
	funcDnsChangeFirewallNotify FuncDnsChangeFirewallNotify
	funcGetUserSettings         FuncGetUserSettings
)

func init() {
	log = logger.NewLogger("dns")
}

func GetExtraSettings() DnsExtraSettings {
	if funcGetUserSettings != nil {
		return funcGetUserSettings()
	}
	return DnsExtraSettings{}
}

type DnsError struct {
	Err error
}

func (e *DnsError) Error() string {
	if e.Err == nil {
		return "DNS error"
	}
	return "DNS error: " + e.Err.Error()
}
func (e *DnsError) Unwrap() error { return e.Err }

func wrapErrorIfFailed(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*DnsError); ok {
		return err
	}
	return &DnsError{Err: err}
}

// Initialize is doing initialization stuff
// Must be called on application start
func Initialize(fwNotifyDnsChangeFunc FuncDnsChangeFirewallNotify, getUserSettingsFunc FuncGetUserSettings) error {
	funcDnsChangeFirewallNotify = fwNotifyDnsChangeFunc
	if funcDnsChangeFirewallNotify == nil {
		logger.Debug("WARNING! Firewall notification function not defined!")
	}

	funcGetUserSettings = getUserSettingsFunc
	if funcGetUserSettings == nil {
		logger.Debug("WARNING! getUserSettingsFunc() function not defined!")
	}

	return wrapErrorIfFailed(implInitialize())
}

// ApplyUserSettings - reinitialize DNS configuration according to user settings
// It is applicable, for example for Linux: when the user changed DNS management style
func ApplyUserSettings() error {
	return implApplyUserSettings()
}

// Pause pauses DNS (restore original DNS)
func Pause(localInterfaceIP net.IP) error {
	return wrapErrorIfFailed(implPause(localInterfaceIP))
}

// Resume resuming DNS (set DNS back which was before Pause)
func Resume(defaultDNS DnsSettings, localInterfaceIP net.IP) error {
	return wrapErrorIfFailed(implResume(defaultDNS, localInterfaceIP))
}

func EncryptionAbilities() (dnsOverHttps, dnsOverTls bool, err error) {
	dnsOverHttps, dnsOverTls, err = implGetDnsEncryptionAbilities()
	return dnsOverHttps, dnsOverTls, wrapErrorIfFailed(err)
}

// SetDefault set DNS configuration treated as default (non-manual) configuration
// 'dnsCfg' parameter - DNS configuration
// 'localInterfaceIP' - local IP of VPN interface
func SetDefault(dnsCfg DnsSettings, localInterfaceIP net.IP) error {
	ret := SetManual(dnsCfg, localInterfaceIP)
	if ret == nil {
		lastManualDNS = DnsSettings{}
	}

	return wrapErrorIfFailed(ret)
}

func notifyFirewall(dnsCfg DnsSettings) error {
	if funcDnsChangeFirewallNotify == nil {
		return nil
	}
	return funcDnsChangeFirewallNotify(&dnsCfg)
}

// SetManual - set manual DNS.
// 'dnsCfg' parameter - DNS configuration
// 'localInterfaceIP' - local IP of VPN interface
func SetManual(dnsCfg DnsSettings, localInterfaceIP net.IP) error {
	dnsForFirewallRules, err := implSetManual(dnsCfg, localInterfaceIP)
	if err == nil {
		lastManualDNS = dnsCfg
	} else {
		return wrapErrorIfFailed(err)
	}

	// notify firewall about DNS configuration
	return wrapErrorIfFailed(notifyFirewall(dnsForFirewallRules))
}

// DeleteManual - reset manual DNS configuration to default (DHCP)
// 'localInterfaceIP' - local IP of VPN interface
func DeleteManual(defaultDns net.IP, localInterfaceIP net.IP) error {
	// reset custom DNS
	ret := implDeleteManual(localInterfaceIP)
	if ret == nil {
		lastManualDNS = DnsSettings{}
	} else {
		return wrapErrorIfFailed(ret)
	}

	// notify firewall about default DNS
	return wrapErrorIfFailed(notifyFirewall(DnsSettingsCreate(defaultDns)))
}

// GetLastManualDNS - returns information about current manual DNS
func GetLastManualDNS() DnsSettings {
	// TODO: get real DNS configuration of the OS
	return lastManualDNS
}

func GetPredefinedDnsConfigurations() ([]DnsSettings, error) {
	settings, err := implGetPredefinedDnsConfigurations()
	return settings, wrapErrorIfFailed(err)
}

// UpdateDnsIfWrongSettings - ensures that current DNS configuration is correct. If not - it re-apply the required configuration.
// Currently, it is in use for macOS - like a DNS change monitor.
func UpdateDnsIfWrongSettings() error {
	return implUpdateDnsIfWrongSettings()
}

func dnscryptProxyProcessStart(dnsCfg DnsSettings) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC (recovered): ", r)
			retErr = fmt.Errorf("%v", r)
			if err, ok := r.(error); ok {
				log.ErrorTrace(err)
			}
		}

		if retErr != nil {
			dnscryptproxy.Stop()
			retErr = fmt.Errorf("failed to start dnscrypt-proxy: %w", retErr)
		}
	}()

	for _, svr := range dnsCfg.Servers {
		if svr.Encryption != EncryptionNone && svr.Encryption != EncryptionDnsOverHttps {
			return fmt.Errorf("dnscryptProxyProcessStart: unsupported DNS encryption type %d", svr.Encryption)
		}
	}

	binPath, configPathTemplate, configPathMutable, logfile := platform.DnsCryptProxyInfo()
	if len(binPath) == 0 || len(configPathTemplate) == 0 || len(configPathMutable) == 0 {
		return fmt.Errorf("configuration not defined")
	}

	// Configure + start dnscrypt-proxy
	stamps := make([]string, 0, len(dnsCfg.Servers))
	for _, svr := range dnsCfg.Servers {
		stamp := dnscryptproxy.ServerStamp{ServerAddrStr: svr.Address, Proto: dnscryptproxy.StampProtoTypePlain}

		if svr.Encryption == EncryptionDnsOverHttps {
			u, err := url.Parse(svr.Template)
			if err != nil {
				return err
			}
			if u.Scheme != "https" {
				return fmt.Errorf("bad template URL scheme: %q", u.Scheme)
			}
			stamp.Proto = dnscryptproxy.StampProtoTypeDoH
			stamp.Path = u.Path
			stamp.ProviderName = u.Hostname()
		}

		//stamp.Props |= dnscryptproxy.ServerInformalPropertyDNSSEC
		//stamp.Props |= dnscryptproxy.ServerInformalPropertyNoLog
		//stamp.Props |= dnscryptproxy.ServerInformalPropertyNoFilter

		stamps = append(stamps, stamp.String())
	}

	// generate dnscrypt-proxy configuration
	if err := dnscryptproxy.SaveConfigFile(stamps, configPathTemplate, configPathMutable, logfile); err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("dnscrypt-proxy config: %q", configPathMutable))

	if err := dnscryptproxy.Init(binPath, configPathMutable, logfile); err != nil {
		return err
	}

	if err := dnscryptproxy.Start(); err != nil {
		if stopErr := dnscryptproxy.Stop(); stopErr != nil {
			log.Warning("failed to stop dnscrypt-proxy: ", stopErr)
		}
		return err
	}

	return nil
}
