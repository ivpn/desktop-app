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
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/shell"
)

var (
	// key: is a string representation of allowed IP
	// value: true - if exception rule is persistant (persistant, means will stay available even client is disconnected)
	allowedHosts   map[string]bool
	allowedForICMP map[string]struct{} // IP addresses allowed for ICMP

	curAllowedLanIPs          []string // IP addresses allowed for LAN
	curStateAllowLAN          bool     // Allow LAN is enabled
	curStateAllowLanMulticast bool     // Allow Multicast is enabled
	curStateEnabled           bool     // Firewall is enabled
	isPersistant              bool     // Firewall is persistant
	mutexInternal             sync.Mutex
)

func init() {
	allowedHosts = make(map[string]bool)
}

func implInitialize() error {
	return nil
}

func implGetEnabled() (bool, error) {
	err := shell.Exec(nil, platform.FirewallScript(), "-status")

	if err != nil {
		exitCode, err := shell.GetCmdExitCode(err)
		if err != nil {
			return false, fmt.Errorf("failed to get Cmd exit code: %w", err)
		}
		if exitCode == 0 {
			return true, nil
		}
		return false, nil
	}
	return true, nil
}

func implSetEnabled(isEnabled bool) error {
	curStateEnabled = isEnabled

	if isEnabled {
		err := shell.Exec(nil, platform.FirewallScript(), "-enable")
		if err != nil {
			return fmt.Errorf("failed to execute shell command: %w", err)
		}

		// To fulfill such flow (example): Connected -> FWDisable -> FWEnable
		// Here we should restore all exceptions (all hosts which are allowed)
		return reApplyExceptions()
	}

	// disable FW ...
	curAllowedLanIPs = nil // forget allowed LAN IP addresses
	isPersistant = false
	allowedForICMP = nil
	return shell.Exec(nil, platform.FirewallScript(), "-disable")
}

func implSetPersistant(persistant bool) error {
	isPersistant = persistant
	if persistant {
		// The persistence is based on such facts:
		// 	- daemon is starting as on system boot
		// 	- SetPersistant() called by service object on daemon start
		// This means we just have to ensure that firewall enabled.

		// Just ensure that firewall is enabled
		ret := implSetEnabled(true)

		// Some Linux distributions erasing IVPN rules during system boot
		// During some period of time (60 seconds should be enough)
		// check if FW rules still exist (if not - re-apply them)
		go ensurePersistant(60)

		return ret
	}
	return nil
}

// Some Linux distributions erasing IVPN rules during system boot
// During some period of time (60 seconds should be enough)
// check if FW rules still exist (if not - re-apply them)
func ensurePersistant(secondsToWait int) {
	const delaySec = 5
	log.Info("[ensurePersistant] started")
	for i := 0; i <= secondsToWait/delaySec; i++ {
		time.Sleep(time.Second * delaySec)
		if !isPersistant {
			break
		}
		enabled, err := implGetEnabled()
		if err != nil {
			log.Error("[ensurePersistant] ", err)
			continue
		}
		if isPersistant && !enabled {
			log.Warning("[ensurePersistant] Persistant FW rules not available. Retry to apply...")
			implSetEnabled(true)
		}
	}
	log.Info("[ensurePersistant] stopped.")
}

// ClientConnected - allow communication for local vpn/client IP address
func implClientConnected(clientLocalIPAddress net.IP, clientLocalIPv6Address net.IP, clientPort int, serverIP net.IP, serverPort int, isTCP bool) error {
	inf, err := netinfo.InterfaceByIPAddr(clientLocalIPAddress)
	if err != nil {
		return fmt.Errorf("failed to get local interface by IP: %w", err)
	}

	protocol := "udp"
	if isTCP {
		protocol = "tcp"
	}
	scriptArgs := fmt.Sprintf("-connected %s %s %d %s %d %s",
		inf.Name,
		clientLocalIPAddress,
		clientPort,
		serverIP,
		serverPort,
		protocol)
	err = shell.Exec(nil, platform.FirewallScript(), scriptArgs)
	if err != nil {
		return fmt.Errorf("failed to add rule for current connection directions: %w", err)
	}

	// Connection already established. The rule for VPN interface is defined.
	// Removing host IP from exceptions
	return removeHostsFromExceptions([]string{serverIP.String()}, false, false)
}

// ClientDisconnected - Disable communication for local vpn/client IP address
func implClientDisconnected() error {
	// remove all exceptions related to current connection (all non-persistant exceptions)
	err := removeAllHostsFromExceptions()
	if err != nil {
		log.Error(err)
	}

	return shell.Exec(nil, platform.FirewallScript(), "-disconnected")
}

func implAllowLAN(isAllowLAN bool, isAllowLanMulticast bool) error {
	return doAllowLAN(isAllowLAN, isAllowLanMulticast)
}

func doAllowLAN(isAllowLAN, isAllowLanMulticast bool) error {
	mutexInternal.Lock()
	defer mutexInternal.Unlock()

	// save expected state of AllowLAN
	curStateAllowLAN = isAllowLAN
	curStateAllowLanMulticast = isAllowLanMulticast

	if isAllowLAN && !curStateEnabled {
		return nil // do nothing if firewall disabled
	}

	// constants
	const persistant = true
	const notOnlyForICMP = false

	// disallow everything (LAN + multicast)
	if len(curAllowedLanIPs) > 0 {
		if err := removeHostsFromExceptions(curAllowedLanIPs, persistant, notOnlyForICMP); err != nil {
			log.Warning("failed to erase 'Allow LAN' rules")
		}
	}
	curAllowedLanIPs = nil

	if !isAllowLAN {
		return nil // LAN NOT ALLOWED
	}

	// LAN ALLOWED

	// TODO: implement LAN access also for IPv6 addresses
	const ipV4 = false
	localRanges := ipNetListToStrings(filterIPNetList(netinfo.GetNonRoutableLocalAddrRanges(), ipV4))
	multicastRanges := ipNetListToStrings(filterIPNetList(netinfo.GetMulticastAddresses(), ipV4))

	curAllowedLanIPs = localRanges
	if isAllowLanMulticast {
		// allow LAN + multicast
		curAllowedLanIPs = append(curAllowedLanIPs, multicastRanges...)
	}

	// allow LAN
	return addHostsToExceptions(curAllowedLanIPs, persistant, notOnlyForICMP)
}

// implAddHostsToExceptions - allow communication with this hosts
// Note: if isPersistent == false -> all added hosts will be removed from exceptions after client disconnection (after call 'ClientDisconnected()')
// Arguments:
//   - IPs			-	list of IP addresses to ba allowed
//   - onlyForICMP	-	try add rule to allow only ICMP protocol for this IP
//   - isPersistent	-	keep rule enabled even if VPN disconnected
//
// NOTE! if (isPersistent==false and onlyForICMP==false) - this exceptions have highest priority (e.g. they will not be blocked by DNS restrictions of the FW)
func implAddHostsToExceptions(IPs []net.IP, onlyForICMP bool, isPersistent bool) error {
	IPsStr := make([]string, 0, len(IPs))
	for _, ip := range IPs {
		if ip.Equal(net.IPv4(127, 0, 0, 1)) {
			continue // we do not need localhost in exceptions
		}
		IPsStr = append(IPsStr, ip.String())
	}

	return addHostsToExceptions(IPsStr, isPersistent, onlyForICMP)
}

func implRemoveHostsFromExceptions(IPs []net.IP, onlyForICMP bool, isPersistent bool) error {
	IPsStr := make([]string, 0, len(IPs))
	for _, ip := range IPs {
		IPsStr = append(IPsStr, ip.String())
	}

	return removeHostsFromExceptions(IPsStr, isPersistent, onlyForICMP)
}

// OnChangeDNS - must be called on each DNS change (to update firewall rules according to new DNS configuration)
func implOnChangeDNS(addr net.IP) error {
	addrStr := ""
	if addr != nil {
		if addr.To4() == nil {
			return fmt.Errorf("DNS is not IPv4 address")
		}
		addrStr = addr.String()
	}

	log.Info("-set_dns", " ", addrStr)
	return shell.Exec(nil, platform.FirewallScript(), "-set_dns", addrStr)
}

// implOnUserExceptionsUpdated() called when 'userExceptions' value were updated. Necessary to update firewall rules.
func implOnUserExceptionsUpdated() error {

	applyFunc := func(isIpv4 bool) error {
		userExceptions := getUserExceptions(isIpv4, !isIpv4)

		var expMasks []string
		for _, mask := range userExceptions {
			expMasks = append(expMasks, mask.String())
		}

		scriptCommand := "-set_user_exceptions_static"
		if !isIpv4 {
			scriptCommand = "-set_user_exceptions_static_ipv6"
		}

		ipList := strings.Join(expMasks, ",")

		if len(ipList) > 250 {
			log.Info(scriptCommand, " <...multiple addresses...>")
		} else {
			log.Info(scriptCommand, " ", ipList)
		}

		return shell.Exec(nil, platform.FirewallScript(), scriptCommand, ipList)
	}

	err := applyFunc(false)
	errIpv6 := applyFunc(true)
	if err == nil && errIpv6 != nil {
		return errIpv6
	}
	return err
}

func implSingleDnsRuleOff() (retErr error) {
	return shell.Exec(log, platform.FirewallScript(), "-only_dns_off")
}

func implSingleDnsRuleOn(dnsAddr net.IP) (retErr error) {
	exceptions := ""
	if prioritized, _ := getAllowedIpExceptions(); len(prioritized) > 0 {
		exceptions = strings.Join(prioritized, ",")
	}

	return shell.Exec(log, platform.FirewallScript(), "-only_dns", dnsAddr.String(), exceptions)
}

//---------------------------------------------------------------------

func applyAddHostsToExceptions(hostsIPs []string, isPersistant bool, onlyForICMP bool) error {
	ipList := strings.Join(hostsIPs, ",")

	if len(ipList) > 0 {
		scriptCommand := "-add_exceptions"

		if onlyForICMP {
			scriptCommand = "-add_exceptions_icmp"
		} else if isPersistant {
			scriptCommand = "-add_exceptions_static"
		}

		if len(ipList) > 250 {
			log.Info(scriptCommand, " <...multiple addresses...>")
		} else {
			log.Info(scriptCommand, " ", ipList)
		}

		return shell.Exec(nil, platform.FirewallScript(), scriptCommand, ipList)
	}
	return nil
}

func applyRemoveHostsFromExceptions(hostsIPs []string, isPersistant bool, onlyForICMP bool) error {
	ipList := strings.Join(hostsIPs, ",")

	if len(ipList) > 0 {
		scriptCommand := "-remove_exceptions"

		if onlyForICMP {
			scriptCommand = "-remove_exceptions_icmp"
		} else if isPersistant {
			scriptCommand = "-remove_exceptions_static"
		}

		if len(ipList) > 250 {
			log.Info(scriptCommand, " <...multiple addresses...>")
		} else {
			log.Info(scriptCommand, " ", ipList)
		}

		return shell.Exec(nil, platform.FirewallScript(), scriptCommand, ipList)
	}
	return nil
}

func reApplyExceptions() error {
	// Allow LAN communication (if necessary)
	// Restore all exceptions (all hosts which are allowed)

	allowedIPs, allowedIPsPersistant := getAllowedIpExceptions()
	allowedIPsICMP := make([]string, 0, len(allowedForICMP))
	if len(allowedForICMP) > 0 {
		for ipStr := range allowedForICMP {
			allowedIPsICMP = append(allowedIPsICMP, ipStr)
		}
	}

	const persistantTRUE = true
	const persistantFALSE = false
	const onlyIcmpTRUE = true
	const onlyIcmpFALSE = false

	// define DNS rules
	err := implOnChangeDNS(getDnsIP())
	if err != nil {
		log.Error(err)
	}

	// Apply all allowed hosts
	err = applyAddHostsToExceptions(allowedIPsICMP, persistantFALSE, onlyIcmpTRUE)
	if err != nil {
		log.Error(err)
	}
	err = applyAddHostsToExceptions(allowedIPs, persistantFALSE, onlyIcmpFALSE)
	if err != nil {
		log.Error(err)
		return err
	}
	err = applyAddHostsToExceptions(allowedIPsPersistant, persistantTRUE, onlyIcmpFALSE)
	if err != nil {
		log.Error(err)
	}

	err = implAllowLAN(curStateAllowLAN, curStateAllowLanMulticast)
	if err != nil {
		log.Error(err)
	}

	err = implOnUserExceptionsUpdated()
	if err != nil {
		log.Error(err)
	}

	return err
}

//---------------------------------------------------------------------

// allow communication with specified hosts
// if isPersistant == false - exception will be removed when client disconnects
func addHostsToExceptions(IPs []string, isPersistant bool, onlyForICMP bool) error {
	if len(IPs) == 0 {
		return nil
	}

	newIPs := make([]string, 0, len(IPs))
	if !onlyForICMP {
		for _, ip := range IPs {
			// do not add new IP if it already in exceptions
			if _, exists := allowedHosts[ip]; !exists {
				allowedHosts[ip] = isPersistant // add to map
				newIPs = append(newIPs, ip)
			}
		}
	} else {
		if allowedForICMP == nil {
			allowedForICMP = make(map[string]struct{})
		}

		for _, ip := range IPs {
			// do not add new IP if it already in exceptions
			if _, exists := allowedForICMP[ip]; !exists {
				allowedForICMP[ip] = struct{}{} // add to map
				newIPs = append(newIPs, ip)
			}
		}
	}

	if len(newIPs) == 0 {
		return nil
	}

	err := applyAddHostsToExceptions(newIPs, isPersistant, onlyForICMP)
	if err != nil {
		log.Error(err)
	}
	return err
}

// Deprecate communication with this hosts
func removeHostsFromExceptions(IPs []string, isPersistant bool, onlyForICMP bool) error {
	if len(IPs) == 0 {
		return nil
	}

	toRemoveIPs := make([]string, 0, len(IPs))
	if !onlyForICMP {
		for _, ip := range IPs {
			if persVal, exists := allowedHosts[ip]; exists {
				if persVal != isPersistant {
					continue
				}
				delete(allowedHosts, ip) // remove from map
				toRemoveIPs = append(toRemoveIPs, ip)
			}
		}
	} else if allowedForICMP != nil {
		for _, ip := range IPs {
			if _, exists := allowedForICMP[ip]; exists {
				delete(allowedForICMP, ip) // remove from map
				toRemoveIPs = append(toRemoveIPs, ip)
			}
		}
	}

	if len(toRemoveIPs) == 0 {
		return nil
	}

	err := applyRemoveHostsFromExceptions(toRemoveIPs, isPersistant, onlyForICMP)
	if err != nil {
		log.Error(err)
	}
	return err
}

// removeAllHostsFromExceptions - Remove hosts (which are related to a current connection) from exceptions
// Note: some exceptions should stay without changes, they are marked as 'persistant'
//
//	(has 'true' value in allowedHosts; eg.: LAN and Multicast connectivity)
func removeAllHostsFromExceptions() error {
	toRemoveIPs := make([]string, 0, len(allowedHosts))
	for ipStr := range allowedHosts {
		toRemoveIPs = append(toRemoveIPs, ipStr)
	}
	isPersistant := false
	return removeHostsFromExceptions(toRemoveIPs, isPersistant, false)
}

//---------------------------------------------------------------------

func getAllowedIpExceptions() (prioritized, persistant []string) {
	prioritized = make([]string, 0, len(allowedHosts))
	persistant = make([]string, 0, len(allowedHosts))
	for ipStr, isPersistant := range allowedHosts {
		if isPersistant {
			persistant = append(persistant, ipStr)
		} else {
			prioritized = append(prioritized, ipStr)
		}
	}
	return prioritized, persistant
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
