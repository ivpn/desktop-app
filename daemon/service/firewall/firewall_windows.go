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

package firewall

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/service/firewall/winlib"
	"github.com/ivpn/desktop-app/daemon/service/platform"
)

var (
	providerKey          = syscall.GUID{Data1: 0xfed0afd4, Data2: 0x98d4, Data3: 0x4233, Data4: [8]byte{0xa4, 0xf3, 0x8b, 0x7c, 0x02, 0x44, 0x50, 0x01}}
	sublayerKey          = syscall.GUID{Data1: 0xfed0afd4, Data2: 0x98d4, Data3: 0x4233, Data4: [8]byte{0xa4, 0xf3, 0x8b, 0x7c, 0x02, 0x44, 0x50, 0x02}}
	providerKeySingleDns = syscall.GUID{Data1: 0xfed0afd4, Data2: 0x98d4, Data3: 0x4233, Data4: [8]byte{0xa4, 0xf3, 0x8b, 0x7c, 0x02, 0x44, 0x50, 0x03}}
	sublayerKeySingleDns = syscall.GUID{Data1: 0xfed0afd4, Data2: 0x98d4, Data3: 0x4233, Data4: [8]byte{0xa4, 0xf3, 0x8b, 0x7c, 0x02, 0x44, 0x50, 0x04}}

	v4Layers = []syscall.GUID{winlib.FwpmLayerAleAuthConnectV4, winlib.FwpmLayerAleAuthRecvAcceptV4}
	v6Layers = []syscall.GUID{winlib.FwpmLayerAleAuthConnectV6, winlib.FwpmLayerAleAuthRecvAcceptV6}

	manager                winlib.Manager
	clientLocalIPFilterIDs []uint64
	customDNS              net.IP

	isPersistant        bool
	isAllowLAN          bool
	isAllowLANMulticast bool
)

const (
	providerDName          = "IVPN Kill Switch Provider"
	sublayerDName          = "IVPN Kill Switch Sub-Layer"
	filterDName            = "IVPN Kill Switch Filter"
	providerDNameSingleDns = "IVPN Kill Switch Provider single DNS"
	sublayerDNameSingleDns = "IVPN Kill Switch Sub-Layer single DNS"
	filterDNameSingleDns   = "IVPN Kill Switch Filter single DNS"
)

// implInitialize doing initialization stuff (called on application start)
func implInitialize() error {
	if err := winlib.Initialize(platform.WindowsWFPDllPath()); err != nil {
		return err
	}

	pInfo, err := manager.GetProviderInfo(providerKey)
	if err != nil {
		return err
	}

	// save initial persistant state into package-variable
	isPersistant = pInfo.IsPersistent

	return nil
}

func implGetEnabled() (bool, error) {
	pInfo, err := manager.GetProviderInfo(providerKey)
	if err != nil {
		return false, fmt.Errorf("failed to get provider info: %w", err)
	}
	return pInfo.IsInstalled, nil
}

func implSetEnabled(isEnabled bool) (retErr error) {
	// start transaction
	if err := manager.TransactionStart(); err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	// do not forget to stop transaction
	defer func() {
		if r := recover(); r == nil {
			manager.TransactionCommit() // commit WFP transaction
		} else {
			manager.TransactionAbort() // abort WFPtransaction

			log.Error("PANIC (recovered): ", r)
			if e, ok := r.(error); ok {
				retErr = e
			} else {
				retErr = errors.New(fmt.Sprint(r))
			}
		}
	}()

	if isEnabled {
		return doEnable()
	}
	return doDisable()
}

func implSetPersistant(persistant bool) (retErr error) {
	// save persistent state
	isPersistant = persistant

	pinfo, err := manager.GetProviderInfo(providerKey)
	if err != nil {
		return fmt.Errorf("failed to get provider info: %w", err)
	}

	if pinfo.IsInstalled {
		if pinfo.IsPersistent == isPersistant {
			log.Info(fmt.Sprintf("Already enabled (persistent=%t).", isPersistant))
			return nil
		}

		log.Info(fmt.Sprintf("Re-enabling with persistent flag = %t", isPersistant))
		return reEnable()
	}

	return doEnable()
}

// ClientConnected - allow communication for local vpn/client IP address
func implClientConnected(clientLocalIPAddress net.IP, clientLocalIPv6Address net.IP, clientPort int, serverIP net.IP, serverPort int, isTCP bool) (retErr error) {
	// start / commit transaction
	if err := manager.TransactionStart(); err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if retErr == nil {
			manager.TransactionCommit()
		} else {
			// abort transaction if there was an error
			manager.TransactionAbort()
		}
	}()

	err := doRemoveClientIPFilters()
	if err != nil {
		log.Error("Failed to remove previously defined client IP filters: ", err)
	}
	return doAddClientIPFilters(clientLocalIPAddress, clientLocalIPv6Address)
}

// ClientDisconnected - Disable communication for local vpn/client IP address
func implClientDisconnected() (retErr error) {
	// start / commit transaction
	if err := manager.TransactionStart(); err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if retErr == nil {
			manager.TransactionCommit()
		} else {
			// abort transaction if there was an error
			manager.TransactionAbort()
		}
	}()

	return doRemoveClientIPFilters()
}

func implAddHostsToExceptions(IPs []net.IP, onlyForICMP bool, isPersistent bool) error {
	// nothing to do for windows implementation
	return nil
}

func implRemoveHostsFromExceptions(IPs []net.IP, onlyForICMP bool, isPersistent bool) error {
	// nothing to do for windows implementation
	return nil
}

// AllowLAN - allow/forbid LAN communication
func implAllowLAN(allowLan bool, allowLanMulticast bool) error {

	if isAllowLAN == allowLan && isAllowLANMulticast == allowLanMulticast {
		return nil
	}

	isAllowLAN = allowLan
	isAllowLANMulticast = allowLanMulticast

	enabled, err := implGetEnabled()
	if err != nil {
		return fmt.Errorf("failed to get info if firewall is on: %w", err)
	}
	if !enabled {
		return nil
	}

	return reEnable()
}

// OnChangeDNS - must be called on each DNS change (to update firewall rules according to new DNS configuration)
func implOnChangeDNS(addr net.IP) error {
	if addr.Equal(customDNS) {
		return nil
	}

	customDNS = addr

	enabled, err := implGetEnabled()
	if err != nil {
		return fmt.Errorf("failed to get info if firewall is on: %w", err)
	}
	if !enabled {
		return nil
	}

	return reEnable()
}

// implOnUserExceptionsUpdated() called when 'userExceptions' value were updated. Necessary to update firewall rules.
func implOnUserExceptionsUpdated() error {
	enabled, err := implGetEnabled()
	if err != nil {
		return fmt.Errorf("failed to get info if firewall is on: %w", err)
	}
	if !enabled {
		return nil
	}

	return reEnable()
}

func reEnable() (retErr error) {
	// start / commit transaction
	if err := manager.TransactionStart(); err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if retErr == nil {
			manager.TransactionCommit()
		} else {
			// abort transaction if there was an error
			manager.TransactionAbort()
		}
	}()

	err := doDisable()
	if err != nil {
		return fmt.Errorf("failed to disable firewall: %w", err)
	}

	err = doEnable()
	if err != nil {
		return fmt.Errorf("failed to enable firewall: %w", err)
	}

	return doAddClientIPFilters(connectedClientInterfaceIP, connectedClientInterfaceIPv6)
}

func doEnable() (retErr error) {
	implSingleDnsRuleOff()

	enabled, err := implGetEnabled()
	if err != nil {
		return fmt.Errorf("failed to get info if firewall is on: %w", err)
	}
	if enabled {
		return nil
	}

	localAddressesV6 := filterIPNetList(netinfo.GetNonRoutableLocalAddrRanges(), true)
	localAddressesV4 := filterIPNetList(netinfo.GetNonRoutableLocalAddrRanges(), false)
	multicastAddressesV6 := filterIPNetList(netinfo.GetMulticastAddresses(), true)
	multicastAddressesV4 := filterIPNetList(netinfo.GetMulticastAddresses(), false)

	provider := winlib.CreateProvider(providerKey, providerDName, "", isPersistant)
	sublayer := winlib.CreateSubLayer(sublayerKey, providerKey,
		sublayerDName, "",
		0xFFF0, // The weight of current layer should be smaller than 0xFFFF (The layer of split-tunneling driver using weight 0xFFFF)
		isPersistant)

	// add provider
	pinfo, err := manager.GetProviderInfo(providerKey)
	if err != nil {
		return fmt.Errorf("failed to get provider info: %w", err)
	}
	if !pinfo.IsInstalled {
		if err = manager.AddProvider(provider); err != nil {
			return fmt.Errorf("failed to add provider : %w", err)
		}
	}

	// add sublayer
	installed, err := manager.IsSubLayerInstalled(sublayerKey)
	if err != nil {
		return fmt.Errorf("failed to check sublayer is installed: %w", err)
	}
	if !installed {
		if err = manager.AddSubLayer(sublayer); err != nil {
			return fmt.Errorf("failed to add sublayer: %w", err)
		}
	}

	// IPv6 filters
	for _, layer := range v6Layers {
		// block all
		_, err := manager.AddFilter(winlib.NewFilterBlockAll(providerKey, layer, sublayerKey, filterDName, "Block all", true, isPersistant, false))
		if err != nil {
			return fmt.Errorf("failed to add filter 'block all IPv6': %w", err)
		}
		if isPersistant {
			// For 'persistant' state we have to add boot-time blocking rule
			bootTime := true
			_, err = manager.AddFilter(winlib.NewFilterBlockAll(providerKey, layer, sublayerKey, filterDName, "Block all (boot time)", true, false, bootTime))
			if err != nil {
				return fmt.Errorf("failed to add boot-time filter 'block all IPv6': %w", err)
			}
		}

		// block DNS
		_, err = manager.AddFilter(winlib.NewFilterBlockDNS(providerKey, layer, sublayerKey, sublayerDName, "Block DNS", nil, isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'block dns': %w", err)
		}

		ipv6loopback := net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}     // LOOPBACK 		::1/128
		ipv6llocal := net.IP{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} // LINKLOCAL		fe80::/10 // TODO: "fe80::/10" is already part of localAddressesV6. To think: do we need it here?

		_, err = manager.AddFilter(winlib.NewFilterAllowRemoteIPV6(providerKey, layer, sublayerKey, filterDName, "", ipv6loopback, 128, isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow remote IP' for ipv6loopback: %w", err)
		}
		_, err = manager.AddFilter(winlib.NewFilterAllowRemoteIPV6(providerKey, layer, sublayerKey, filterDName, "", ipv6llocal, 10, isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow remote IP' for ipv6llocal: %w", err)
		}

		// LAN
		if isAllowLAN {
			for _, ip := range localAddressesV6 {
				prefixLen, _ := ip.Mask.Size()
				_, err = manager.AddFilter(winlib.NewFilterAllowRemoteIPV6(providerKey, layer, sublayerKey, filterDName, "", ip.IP, byte(prefixLen), isPersistant))
				if err != nil {
					return fmt.Errorf("failed to add filter 'allow lan IPv6': %w", err)
				}
			}

			// Multicast
			if isAllowLANMulticast {
				for _, ip := range multicastAddressesV6 {
					prefixLen, _ := ip.Mask.Size()
					_, err = manager.AddFilter(winlib.NewFilterAllowRemoteIPV6(providerKey, layer, sublayerKey, filterDName, "", ip.IP, byte(prefixLen), isPersistant))
					if err != nil {
						return fmt.Errorf("failed to add filter 'allow LAN multicast IPv6': %w", err)
					}
				}
			}
		}

		// user exceptions
		userExpsNets := getUserExceptions(false, true)
		for _, n := range userExpsNets {
			prefixLen, _ := n.Mask.Size()
			_, err = manager.AddFilter(winlib.NewFilterAllowRemoteIPV6(providerKey, layer, sublayerKey, filterDName, "", n.IP, byte(prefixLen), isPersistant))
			if err != nil {
				return fmt.Errorf("failed to add filter 'user exception': %w", err)
			}
		}
	}

	// IPv4 filters
	for _, layer := range v4Layers {
		// block all
		_, err := manager.AddFilter(winlib.NewFilterBlockAll(providerKey, layer, sublayerKey, filterDName, "Block all", false, isPersistant, false))
		if err != nil {
			return fmt.Errorf("failed to add filter 'block all': %w", err)
		}
		if isPersistant {
			// For 'persistant' state we have to add boot-time blocking rule
			bootTime := true
			_, err = manager.AddFilter(winlib.NewFilterBlockAll(providerKey, layer, sublayerKey, filterDName, "Block all (boot time)", false, false, bootTime))
			if err != nil {
				return fmt.Errorf("failed to add boot-time filter 'block all': %w", err)
			}
		}

		// block DNS
		_, err = manager.AddFilter(winlib.NewFilterBlockDNS(providerKey, layer, sublayerKey, sublayerDName, "Block DNS", customDNS, isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'block dns': %w", err)
		}
		// allow DNS requests to 127.0.0.1:53
		_, err = manager.AddFilter(winlib.AllowRemoteLocalhostDNS(providerKey, layer, sublayerKey, sublayerDName, "", isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow localhost dns': %w", err)
		}

		// allow DHCP port
		_, err = manager.AddFilter(winlib.NewFilterAllowLocalPort(providerKey, layer, sublayerKey, sublayerDName, "", 68, isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow dhcp': %w", err)
		}

		// allow current executable
		binaryPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to obtain executable info: %w", err)
		}
		_, err = manager.AddFilter(winlib.NewFilterAllowApplication(providerKey, layer, sublayerKey, sublayerDName, "", binaryPath, isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow application': %w", err)
		}

		// allow OpenVPN executable
		_, err = manager.AddFilter(winlib.NewFilterAllowApplication(providerKey, layer, sublayerKey, sublayerDName, "", platform.OpenVpnBinaryPath(), isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow application - openvpn': %w", err)
		}
		// allow WireGuard executable
		_, err = manager.AddFilter(winlib.NewFilterAllowApplication(providerKey, layer, sublayerKey, sublayerDName, "", platform.WgBinaryPath(), isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow application - wireguard': %w", err)
		}
		// allow obfsproxy
		_, err = manager.AddFilter(winlib.NewFilterAllowApplication(providerKey, layer, sublayerKey, sublayerDName, "", platform.ObfsproxyStartScript(), isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow application - obfsproxy': %w", err)
		}
		// allow V2Ray
		_, err = manager.AddFilter(winlib.NewFilterAllowApplication(providerKey, layer, sublayerKey, sublayerDName, "", platform.V2RayBinaryPath(), isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow application - V2Ray': %w", err)
		}
		// allow dnscrypt-proxy
		dnscryptProxyBin, _, _, _ := platform.DnsCryptProxyInfo()
		_, err = manager.AddFilter(winlib.NewFilterAllowApplication(providerKey, layer, sublayerKey, sublayerDName, "", dnscryptProxyBin, isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow application - dnscrypt-proxy': %w", err)
		}

		_, err = manager.AddFilter(winlib.NewFilterAllowRemoteIP(providerKey, layer, sublayerKey, filterDName, "", net.ParseIP("127.0.0.1"), net.IPv4(255, 255, 255, 255), isPersistant))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow remote IP': %w", err)
		}

		// LAN
		if isAllowLAN {
			for _, ip := range localAddressesV4 {
				_, err = manager.AddFilter(winlib.NewFilterAllowRemoteIP(providerKey, layer, sublayerKey, filterDName, "", ip.IP, net.IP(ip.Mask), isPersistant))
				if err != nil {
					return fmt.Errorf("failed to add filter 'allow LAN': %w", err)
				}
			}

			// Multicast
			if isAllowLANMulticast {
				for _, ip := range multicastAddressesV4 {
					_, err = manager.AddFilter(winlib.NewFilterAllowRemoteIP(providerKey, layer, sublayerKey, filterDName, "", ip.IP, net.IP(ip.Mask), isPersistant))
					if err != nil {
						return fmt.Errorf("failed to add filter 'allow LAN': %w", err)
					}
				}
			}
		}

		// user exceptions
		userExpsNets := getUserExceptions(true, false)
		for _, n := range userExpsNets {
			_, err = manager.AddFilter(winlib.NewFilterAllowRemoteIP(providerKey, layer, sublayerKey, filterDName, "", n.IP, net.IP(n.Mask), isPersistant))
			if err != nil {
				return fmt.Errorf("failed to add filter 'allow LAN': %w", err)
			}
		}
	}

	return nil
}

func doDisable() error {
	implSingleDnsRuleOff()

	enabled, err := implGetEnabled()
	if err != nil {
		return fmt.Errorf("failed to get info if firewall is on: %w", err)
	}
	if !enabled {
		return nil
	}

	// delete filters
	for _, l := range v6Layers {
		// delete filters and callouts registered for the provider+layer
		if err := manager.DeleteFilterByProviderKey(providerKey, l); err != nil {
			return fmt.Errorf("failed to delete filter : %w", err)
		}
	}

	for _, l := range v4Layers {
		// delete filters and callouts registered for the provider+layer
		if err := manager.DeleteFilterByProviderKey(providerKey, l); err != nil {
			return fmt.Errorf("failed to delete filter : %w", err)
		}
	}

	// delete sublayer
	installed, err := manager.IsSubLayerInstalled(sublayerKey)
	if err != nil {
		return fmt.Errorf("failed to check is sublayer installed : %w", err)
	}
	if installed {
		if err := manager.DeleteSubLayer(sublayerKey); err != nil {
			return fmt.Errorf("failed to delete sublayer : %w", err)
		}
	}

	// delete provider
	pinfo, err := manager.GetProviderInfo(providerKey)
	if err != nil {
		return fmt.Errorf("failed to get provider info : %w", err)
	}
	if pinfo.IsInstalled {
		if err := manager.DeleteProvider(providerKey); err != nil {
			return fmt.Errorf("failed to delete provider : %w", err)
		}
	}

	clientLocalIPFilterIDs = nil

	return nil
}

func doAddClientIPFilters(clientLocalIP net.IP, clientLocalIPv6 net.IP) (retErr error) {
	if clientLocalIP == nil {
		return nil
	}

	enabled, err := implGetEnabled()
	if err != nil {
		return fmt.Errorf("failed to get info if firewall is on: %w", err)
	}
	if !enabled {
		return nil
	}

	filters := make([]uint64, 0, len(v4Layers))
	for _, layer := range v4Layers {
		f := winlib.NewFilterAllowLocalIP(providerKey, layer, sublayerKey, filterDName, "", clientLocalIP, net.IPv4(255, 255, 255, 255), false)
		id, err := manager.AddFilter(f)
		if err != nil {
			return fmt.Errorf("failed to add filter : %w", err)
		}
		filters = append(filters, id)
	}

	// IPv6: allow IPv6 communication inside tunnel
	if clientLocalIPv6 != nil {
		for _, layer := range v6Layers {
			f := winlib.NewFilterAllowLocalIPV6(providerKey, layer, sublayerKey, filterDName, "", clientLocalIPv6, byte(128), false)
			id, err := manager.AddFilter(f)
			if err != nil {
				return fmt.Errorf("failed to add IPv6 filter : %w", err)
			}
			filters = append(filters, id)
		}
	}

	clientLocalIPFilterIDs = filters

	return nil
}

func doRemoveClientIPFilters() (retErr error) {
	defer func() {
		clientLocalIPFilterIDs = nil
	}()

	enabled, err := implGetEnabled()
	if err != nil {
		return fmt.Errorf("failed to get info if firewall is on: %w", err)
	}
	if !enabled {
		return nil
	}

	for _, filterID := range clientLocalIPFilterIDs {
		err := manager.DeleteFilterByID(filterID)
		if err != nil {
			return fmt.Errorf("failed to delete filter : %w", err)
		}
	}

	return nil
}

func getUserExceptions(ipv4, ipv6 bool) []net.IPNet {
	ret := []net.IPNet{}
	for _, e := range userExceptions {
		isIPv6 := e.IP.To4() == nil
		isIPv4 := !isIPv6

		if !(isIPv4 && ipv4) && !(isIPv6 && ipv6) {
			continue
		}

		ret = append(ret, e)
	}
	return ret
}

func implSingleDnsRuleOff() (retErr error) {
	pInfo, err := manager.GetProviderInfo(providerKeySingleDns)
	if err != nil {
		return fmt.Errorf("failed to get provider info: %w", err)
	}
	if !pInfo.IsInstalled {
		return nil
	}

	// delete filters
	for _, l := range v6Layers {
		// delete filters and callouts registered for the provider+layer
		if err := manager.DeleteFilterByProviderKey(providerKeySingleDns, l); err != nil {
			return fmt.Errorf("failed to delete filter : %w", err)
		}
	}

	for _, l := range v4Layers {
		// delete filters and callouts registered for the provider+layer
		if err := manager.DeleteFilterByProviderKey(providerKeySingleDns, l); err != nil {
			return fmt.Errorf("failed to delete filter : %w", err)
		}
	}

	// delete sublayer
	installed, err := manager.IsSubLayerInstalled(sublayerKeySingleDns)
	if err != nil {
		return fmt.Errorf("failed to check is sublayer installed : %w", err)
	}
	if installed {
		if err := manager.DeleteSubLayer(sublayerKeySingleDns); err != nil {
			return fmt.Errorf("failed to delete sublayer : %w", err)
		}
	}

	// delete provider
	if err := manager.DeleteProvider(providerKeySingleDns); err != nil {
		return fmt.Errorf("failed to delete provider : %w", err)
	}
	return nil
}

func implSingleDnsRuleOn(dnsAddr net.IP) (retErr error) {
	if enabled, err := implGetEnabled(); err != err {
		return err
	} else if enabled {
		return fmt.Errorf("failed to apply specific DNS rule: Firewall alredy enabled")
	}

	if dnsAddr == nil {
		return fmt.Errorf("DNS address not defined")
	}

	if err := manager.TransactionStart(); err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	// do not forget to stop transaction
	defer func() {
		if r := recover(); r == nil {
			manager.TransactionCommit() // commit WFP transaction
		} else {
			manager.TransactionAbort() // abort WFPtransaction

			log.Error("PANIC (recovered): ", r)
			if e, ok := r.(error); ok {
				retErr = e
			} else {
				retErr = errors.New(fmt.Sprint(r))
			}
		}
	}()

	provider := winlib.CreateProvider(providerKeySingleDns, providerDNameSingleDns, "", false)
	sublayer := winlib.CreateSubLayer(sublayerKeySingleDns, providerKeySingleDns,
		sublayerDNameSingleDns, "",
		0xFFF0, // The weight of current layer should be smaller than 0xFFFF (The layer of split-tunneling driver using weight 0xFFFF)
		false)

	// add provider
	pinfo, err := manager.GetProviderInfo(providerKeySingleDns)
	if err != nil {
		return fmt.Errorf("failed to get provider info: %w", err)
	}
	if !pinfo.IsInstalled {
		if err = manager.AddProvider(provider); err != nil {
			return fmt.Errorf("failed to add provider : %w", err)
		}
	}

	// add sublayer
	installed, err := manager.IsSubLayerInstalled(sublayerKeySingleDns)
	if err != nil {
		return fmt.Errorf("failed to check sublayer is installed: %w", err)
	}
	if !installed {
		if err = manager.AddSubLayer(sublayer); err != nil {
			return fmt.Errorf("failed to add sublayer: %w", err)
		}
	}

	var ipv6DnsIpException net.IP = nil
	var ipv4DnsIpException net.IP = nil
	if dnsAddr.To4() == nil {
		ipv6DnsIpException = dnsAddr
	} else {
		ipv4DnsIpException = dnsAddr
	}

	// IPv6 filters
	for _, layer := range v6Layers {
		// block DNS
		_, err = manager.AddFilter(winlib.NewFilterBlockDNS(providerKeySingleDns, layer, sublayerKeySingleDns, filterDNameSingleDns, "Block DNS", ipv6DnsIpException, false))
		if err != nil {
			return fmt.Errorf("failed to add filter 'block dns': %w", err)
		}
	}

	// IPv4 filters
	for _, layer := range v4Layers {
		// block DNS
		_, err = manager.AddFilter(winlib.NewFilterBlockDNS(providerKeySingleDns, layer, sublayerKeySingleDns, filterDNameSingleDns, "Block DNS", ipv4DnsIpException, false))
		if err != nil {
			return fmt.Errorf("failed to add filter 'block dns': %w", err)
		}
		// allow DNS requests to 127.0.0.1:53
		_, err = manager.AddFilter(winlib.AllowRemoteLocalhostDNS(providerKeySingleDns, layer, sublayerKeySingleDns, filterDNameSingleDns, "", false))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow localhost dns': %w", err)
		}
		// allow V2Ray: to avoid blocking connections to V2Ray port 53
		_, err = manager.AddFilter(winlib.NewFilterAllowApplication(providerKeySingleDns, layer, sublayerKeySingleDns, filterDNameSingleDns, "", platform.V2RayBinaryPath(), false))
		if err != nil {
			return fmt.Errorf("failed to add filter 'allow application - V2Ray': %w", err)
		}
	}
	return nil
}
