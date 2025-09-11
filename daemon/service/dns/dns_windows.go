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
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/service/dns/dnscryptproxy"
	"github.com/ivpn/desktop-app/daemon/service/platform"
)

var (
	_fSetDNSByLocalIP      *syscall.LazyProc // DWORD _cdecl SetDNSByLocalIP(const char* interfaceLocalAddr, const char* dnsIP, byte operation, byte isDoH, const char* dohTemplateUrl, byte isIpv6)
	_fIsCanUseDnsOverHttps *syscall.LazyProc // DWORD _cdecl IsCanUseDnsOverHttps()
)

var (
	dnsMutex sync.Mutex

	// last custom-DNS info which was enabled
	_lastDNS DnsSettings
)

// Operation enumerates possible DNS operations
type Operation uint32

// DNS operations
const (
	OperationSet Operation = 0
	OperationAdd Operation = 1
	OperationDel Operation = 2
)

// implInitialize doing initialization stuff (called on application start)
func implInitialize() error {
	helpersDllPath := platform.WindowsNativeHelpersDllPath()
	if len(helpersDllPath) == 0 {
		return fmt.Errorf("unable to initialize DNS wrapper: helpers dll path not initialized")
	}
	if _, err := os.Stat(helpersDllPath); err != nil {
		return fmt.Errorf("unable to initialize DNS wrapper (helpers dll not found) : '%s'", helpersDllPath)
	}

	dll := syscall.NewLazyDLL(helpersDllPath)
	_fSetDNSByLocalIP = dll.NewProc("SetDNSByLocalIP")           // DWORD _cdecl SetDNSByLocalIP(const char* interfaceLocalAddr, const char* dnsIP, byte operation, byte isDoH, const char* dohTemplateUrl, byte isIpv6)
	_fIsCanUseDnsOverHttps = dll.NewProc("IsCanUseDnsOverHttps") // DWORD _cdecl IsCanUseDnsOverHttps()
	return nil
}

func implApplyUserSettings() error {
	return nil // nothing to do here for current platfom
}

func fSetDNSByLocalIP(interfaceLocalAddr net.IP, dnsCfg DnsServerConfig, op Operation) error {
	isDoH := uint32(0)
	dohTemplateUrl := ""
	switch dnsCfg.Encryption {
	case EncryptionDnsOverTls:
		return fmt.Errorf("DnsOverTls settings not supported by Windows. Please, try to use DnsOverHttps")
	case EncryptionDnsOverHttps:
		isDoH = 1
		dohTemplateUrl = dnsCfg.Template
	default:
		isDoH = 0
	}

	dnsIpString := ""
	isIpv6 := uint32(0)

	if !dnsCfg.IsEmpty() {
		dnsIpString = dnsCfg.Ip().String()
		if dnsCfg.IsIPv6() {
			isIpv6 = 1
		}
	}

	dnsMutex.Lock()
	defer dnsMutex.Unlock()

	var (
		err                 error
		cInterfaceLocalAddr *byte
		cDnsIpString        *byte
		cDohTemplateUrl     *byte
	)
	if cInterfaceLocalAddr, err = syscall.BytePtrFromString(interfaceLocalAddr.String()); err != nil {
		return fmt.Errorf("internal error: failed to convert interfaceLocalAddr to byte-pointer: %w", err)
	}
	if cDnsIpString, err = syscall.BytePtrFromString(dnsIpString); err != nil {
		return fmt.Errorf("internal error: failed to convert dnsIpString to byte-pointer: %w", err)
	}
	if cDohTemplateUrl, err = syscall.BytePtrFromString(dohTemplateUrl); err != nil {
		return fmt.Errorf("internal error: failed to convert dohTemplateUrl to byte-pointer: %w", err)
	}

	retval, _, err := _fSetDNSByLocalIP.Call(
		uintptr(unsafe.Pointer(cInterfaceLocalAddr)),
		uintptr(unsafe.Pointer(cDnsIpString)),
		uintptr(op),
		uintptr(isDoH),
		uintptr(unsafe.Pointer(cDohTemplateUrl)),
		uintptr(isIpv6))

	return checkDefaultAPIResp(retval, err)
}

func fIsCanUseNativeDnsOverHttps() bool {
	retval, _, err := _fIsCanUseDnsOverHttps.Call()
	if retval == 0 || err != syscall.Errno(0) {
		return false
	}
	return true
}

func checkDefaultAPIResp(retval uintptr, err error) error {
	if err != syscall.Errno(0) {
		return err
	}
	if retval != 0 {
		return fmt.Errorf("DNS change error: 0x%X", retval)
	}
	return nil
}

func catchPanic(err *error) {
	if r := recover(); r != nil {
		log.Error("PANIC (recovered): ", r)
		if e, ok := r.(error); ok {
			*err = e
		} else {
			*err = errors.New(fmt.Sprint(r))
		}
	}
}

// Pause - (on vpn paused) temporary restore OS default DNS parameters
func implPause(localInterfaceIP net.IP) error {
	// Not in use for Windows implementation
	// In paused state we are simply switching to the main network interface (to default routes)

	// TODO: in case of custom DNS from local network - necessary to remove custom-DNS configuration from main (non-ivpn) network interface ???

	return nil
}

// Resume - (on vpn resumed) set VPN-defined DNS parameters
func implResume(defaultDNS DnsSettings, localInterfaceIP net.IP) error {
	// Not in use for Windows implementation
	// In paused state we are simply switching to the main network interface (to default routes)

	// TODO: in case of custom DNS from local network - necessary to add (restore) custom-DNS configuration to main (non-ivpn) network interface ???

	return nil
}

func implGetDnsEncryptionAbilities() (dnsOverHttps, dnsOverTls bool, err error) {
	defer catchPanic(&err)

	return true, false, err
}

func implSetManual(dnsCfg DnsSettings, vpnInterfaceIP net.IP) (dnsInfoForFirewall DnsSettings, retErr error) {
	defer catchPanic(&retErr)
	defer func() {
		if retErr != nil {
			if err := dnscryptproxy.Stop(); err != nil {
				log.Error("failed to stop dnscrypt-proxy: ", err)
			}
		}
	}()

	if err := dnscryptproxy.Stop(); err != nil {
		log.Error("failed to stop dnscrypt-proxy: ", err)
	}

	// If there was defined DNS - remove it from non-VPN interfaces (if necessary)
	// (skipping VPN interface, because its data will be overwritten)
	if !_lastDNS.IsEmpty() {
		if err := implDeleteManual(nil); err != nil {
			return DnsSettings{}, fmt.Errorf("failed to set DNS: %w", err)
		}
	}

	if dnsCfg.IsEmpty() {
		return DnsSettings{}, fmt.Errorf("unable to change DNS (configuration is not defined)")
	}

	useOnlyVpnInterface := false

	// If system does not support encrypted DNS natively - start dnscrypt-proxy for encrypted DNS
	if dnsCfg.UseEncryption() && !fIsCanUseNativeDnsOverHttps() {
		if err := dnscryptProxyProcessStart(dnsCfg); err != nil {
			return DnsSettings{}, err
		}
		// the local DNS must be configured to the dnscrypt-proxy (localhost)
		dnsCfg = DnsSettings{Servers: []DnsServerConfig{{Address: "127.0.0.1"}}}
		useOnlyVpnInterface = true
	}

	// Logging
	start := time.Now()
	log.Info(fmt.Sprintf("Changing DNS to %s ...", dnsCfg.InfoString()))
	defer func() {
		if retErr != nil {
			log.Error(fmt.Sprintf("Changing DNS to %s done (%dms) with error: %s", dnsCfg.InfoString(), time.Since(start).Milliseconds(), retErr.Error()))
		} else {
			log.Info(fmt.Sprintf("Changing DNS to %s: done (%dms)", dnsCfg.InfoString(), time.Since(start).Milliseconds()))
		}
	}()

	// Init: ERASE DNS configuration for the VPN interface
	if vpnInterfaceIP != nil {
		if err := fSetDNSByLocalIP(vpnInterfaceIP, DnsServerConfig{}, OperationSet); err != nil {
			return DnsSettings{}, fmt.Errorf("failed to set DNS for local interface: %w", err)
		}
	}

	// keep list of DNS servers which were applied already
	appliedServers := make(map[DnsServerConfig]struct{})

	// For each DNS server - find local interface which can reach it and apply DNS server to this interface
	//
	// Note: Do not apply DNS to the VPN interface if the DNS server is in the local network.
	// Otherwise, it can cause delays in DNS resolution.
	if !useOnlyVpnInterface {
		// All local networks
		lNetworks, err := getLocalNetworks(nil)
		if err != nil {
			return DnsSettings{}, fmt.Errorf("error receiving local addresses: %w", err)
		}

		for _, network := range lNetworks {
			for _, svr := range dnsCfg.Servers {
				if !network.Contains(svr.Ip()) { // 'svr.Ip()' is not in 'network'
					continue
				}
				if err := fSetDNSByLocalIP(network.IP, svr, OperationAdd); err != nil {
					return DnsSettings{}, fmt.Errorf("failed to add DNS config %q for interface: %w", svr.InfoString(), err)
				}
				appliedServers[svr] = struct{}{}
			}
		}
	}

	// Apply DNS servers which were not applied yet to the VPN interface
	for _, svr := range dnsCfg.Servers {
		if _, ok := appliedServers[svr]; ok {
			continue
		}
		if err := fSetDNSByLocalIP(vpnInterfaceIP, svr, OperationAdd); err != nil {
			return DnsSettings{}, fmt.Errorf("failed to add DNS config %q for interface: %w", svr.InfoString(), err)
		}
	}

	// save last changed DNS address
	_lastDNS = dnsCfg

	return _lastDNS, retErr
}

func implDeleteManual(vpnInterfaceIP net.IP) (retErr error) {
	defer catchPanic(&retErr)

	if err := dnscryptproxy.Stop(); err != nil {
		log.Error("failed to stop dnscrypt-proxy: ", err)
	}

	if _lastDNS.IsEmpty() {
		return nil
	}

	// Logging
	start := time.Now()
	log.Info("Restoring default DNS...")
	defer func() {
		if retErr != nil {
			log.Info(fmt.Sprintf("Restoring default DNS done (%dms) with error: %s", time.Since(start).Milliseconds(), retErr.Error()))
		} else {
			log.Info(fmt.Sprintf("Restoring default DNS: done (%dms)", time.Since(start).Milliseconds()))
		}
	}()

	// RESET DNS for VPN interface
	if vpnInterfaceIP != nil {
		if err := fSetDNSByLocalIP(vpnInterfaceIP, DnsServerConfig{}, OperationSet); err != nil {
			retErr = fmt.Errorf("failed to reset DNS for VPN interface: %w", err)
		}
	}

	// Remove DNS configuration from non-VPN interfaces (if necessary)
	nonVpnNetworks, err := getLocalNetworks(vpnInterfaceIP)
	if err != nil {
		retErr = fmt.Errorf("error receiving local addresses: %w", err)
	}
	for _, network := range nonVpnNetworks {
		for _, svr := range _lastDNS.Servers {
			if !network.Contains(svr.Ip()) { // 'svr.Ip()' is not in 'network'
				continue
			}
			if err := fSetDNSByLocalIP(network.IP, svr, OperationDel); err != nil {
				log.Error(fmt.Errorf("failed to remove previously applied DNS configuration %q for non-VPN interface: %w", svr.InfoString(), err))
			}
		}
	}

	_lastDNS = DnsSettings{}

	return retErr
}

func implGetPredefinedDnsConfigurations() ([]DnsSettings, error) {
	return []DnsSettings{}, nil
}

// UpdateDnsIfWrongSettings - ensures that current DNS configuration is correct. If not - it re-apply the required configuration.
func implUpdateDnsIfWrongSettings() error {
	// Not in use for Windows implementation
	// We are using platform-specific implementation of DNS change monitor for Windows
	return nil
}

// getLocalNetworks returns all local networks, excluding the one associated with the specified interface IP.
// Parameters:
//   - ifaceToSkip - local IP of the interface to exclude from the results (typically the VPN interface)
func getLocalNetworks(ifaceToSkip net.IP) ([]net.IPNet, error) {
	networks, err := netinfo.GetAllLocalAddresses()
	if err != nil {
		return nil, fmt.Errorf("error receiving local addresses: %w", err)
	}

	for _, network := range networks {
		if network.IP.Equal(ifaceToSkip) || network.IP.IsLoopback() {
			continue
		}
	}

	return networks, nil
}

// getNetworkForIP returns the network from the provided list that contains the specified IP address.
// If no such network exists, it returns nil.
func getNetworkForIP(addr net.IP, netWorks []net.IPNet) *net.IPNet {
	if addr == nil {
		return nil
	}

	for _, network := range netWorks {
		if network.Contains(addr) {
			return &network
		}
	}

	return nil
}
