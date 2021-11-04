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

package firewall

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/shell"
)

var (
	// key: is a string representation of allowed IP
	// value: true - if exception rule is persistant (persistant, means will stay available even client is disconnected)
	allowedHosts map[string]bool
	// IP addresses of local interfaces (using for 'allow LAN' functionality)
	allowedLanIPs       []string
	allowedForICMP      map[string]struct{}
	connectedVpnLocalIP string

	delayedAllowLanAllowed bool = true
	delayedAllowLanStarted bool = false

	isPersistant bool = false
)

const (
	multicastIP = "224.0.0.0/4"
)

func init() {
	allowedHosts = make(map[string]bool)
}

func implInitialize() error { return nil }

func implGetEnabled() (bool, error) {
	err := shell.Exec(nil, platform.FirewallScript(), "-status")

	if err != nil {
		exitCode, err := shell.GetCmdExitCode(err)
		if err != nil {
			return false, fmt.Errorf("failed to get Cmd exit code: %w", err)
		}
		if exitCode == 1 {
			return false, nil
		}
		return false, nil
	}
	return true, nil
}

func implSetEnabled(isEnabled bool) error {
	if isEnabled {
		err := shell.Exec(nil, platform.FirewallScript(), "-enable")
		if err != nil {
			return fmt.Errorf("failed to execute shell command: %w", err)
		}
		// To fulfill such flow (example): Connected -> FWDisable -> FWEnable
		// Here we should restore all exceptions (all hosts which are allowed)
		return reApplyExceptions()
	}

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
		if isPersistant != true {
			break
		}
		enabled, err := implGetEnabled()
		if err != nil {
			log.Error("[ensurePersistant] ", err)
			continue
		}
		if isPersistant == true && enabled != true {
			log.Warning("[ensurePersistant] Persistant FW rules not available. Retry to apply...")
			implSetEnabled(true)
		}
	}
	log.Info("[ensurePersistant] stopped.")
}

// ClientConnected - allow communication for local vpn/client IP address
func implClientConnected(clientLocalIPAddress net.IP, clientLocalIPv6Address net.IP, clientPort int, serverIP net.IP, serverPort int, isTCP bool) error {
	connectedVpnLocalIP = clientLocalIPAddress.String()
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
	connectedVpnLocalIP = ""
	// remove all exceptions related to current connection (all non-persistant exceptions)
	err := removeAllHostsFromExceptions()
	if err != nil {
		log.Error(err)
	}

	return shell.Exec(nil, platform.FirewallScript(), "-disconnected")
}

func implAllowLAN(isAllowLAN bool, isAllowLanMulticast bool) error {
	const persistant = true
	const notOnlyForICMP = false

	if !isAllowLAN {
		// LAN NOT ALLOWED
		delayedAllowLanAllowed = false
		if len(allowedLanIPs) <= 0 {
			return nil
		}
		// disallow everything (LAN + multicast)
		toRemove := allowedLanIPs
		allowedLanIPs = nil
		return removeHostsFromExceptions(toRemove, persistant, false)
	}

	// LAN ALLOWED
	localIPs, err := getLanIPs()
	if err != nil {
		return fmt.Errorf("failed to get local IPs: %w", err)
	}

	if len(localIPs) > 0 {
		delayedAllowLanAllowed = false
	} else {
		// this can happen, for example, on system boot (when no network interfaces initialized)
		log.Info("Local LAN addresses not detected: no data to apply the 'Allow LAN' rule")
		go delayedAllowLAN(isAllowLanMulticast)
		return nil
	}

	if len(allowedLanIPs) > 0 {
		removeHostsFromExceptions(allowedLanIPs, persistant, false)
	}

	allowedLanIPs = localIPs
	if isAllowLanMulticast {
		// allow LAN + multicast
		allowedLanIPs = append(allowedLanIPs, multicastIP)
		return addHostsToExceptions(allowedLanIPs, persistant, notOnlyForICMP)
	}

	// disallow Multicast
	removeHostsFromExceptions([]string{multicastIP}, persistant, false)
	// allow LAN
	return addHostsToExceptions(allowedLanIPs, persistant, notOnlyForICMP)
}

func delayedAllowLAN(allowLanMulticast bool) {
	if delayedAllowLanStarted || delayedAllowLanAllowed == false {
		return
	}
	log.Info("Delayed 'Allow LAN': Will try to apply this rule few seconds later...")
	delayedAllowLanStarted = true

	defer func() { delayedAllowLanAllowed = false }()
	for i := 0; i < 25 && delayedAllowLanAllowed; i++ {
		time.Sleep(time.Second)
		ipList, err := getLanIPs()
		if err != nil {
			log.Warning(fmt.Errorf("Delayed 'Allow LAN': failed to get local IPs: %w", err))
			return
		}
		if len(ipList) > 0 {
			time.Sleep(time.Second) // just to ensure that everything initialized
			if delayedAllowLanAllowed {
				log.Info("Delayed 'Allow LAN': apply ...")
				err := implAllowLAN(true, allowLanMulticast)
				if err != nil {
					log.Warning(fmt.Errorf("Delayed 'Allow LAN' error: %w", err))
				}
			}
			return
		}
	}
	log.Info("Delayed 'Allow LAN': no LAN interfaces detected")
}

// implAddHostsToExceptions - allow comminication with this hosts
// Note: if isPersistent == false -> all added hosts will be removed from exceptions after client disconnection (after call 'ClientDisconnected()')
// Arguments:
//	* IPs			-	list of IP addresses to ba allowed
//	* onlyForICMP	-	try add rule to allow only ICMP protocol for this IP
//	* isPersistent	-	keep rule enabled even if VPN disconnected
func implAddHostsToExceptions(IPs []net.IP, onlyForICMP bool, isPersistent bool) error {
	IPsStr := make([]string, 0, len(IPs))
	for _, ip := range IPs {
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

// SetManualDNS - configure firewall to allow DNS which is out of VPN tunnel
// Applicable to Windows implementation (to allow custom DNS from local network)
func implSetManualDNS(addr net.IP) error {
	// not in use for Linux
	return nil
}

//---------------------------------------------------------------------

func applyAddHostsToExceptions(hostsIPs []string, isPersistant bool, onlyForICMP bool) error {
	var ipList string
	ipList = strings.Join(hostsIPs, ",")

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
	allowedIPs := make([]string, 0, len(allowedHosts))
	allowedIPsPersistant := make([]string, 0, len(allowedHosts))
	if len(allowedHosts) > 0 {
		for ipStr, isPersistant := range allowedHosts {
			if isPersistant {
				allowedIPsPersistant = append(allowedIPsPersistant, ipStr)
			} else {
				allowedIPs = append(allowedIPs, ipStr)
			}
		}
	}

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
	// Apply all allowed hosts
	err := applyAddHostsToExceptions(allowedIPsICMP, persistantFALSE, onlyIcmpTRUE)
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

	return err
}

//---------------------------------------------------------------------

// allow communication with specified hosts
// if isPersistant == false - exception will be removed when client disctonnects
func addHostsToExceptions(IPs []string, isPersistant bool, onlyForICMP bool) error {
	if len(IPs) == 0 {
		return nil
	}

	newIPs := make([]string, 0, len(IPs))
	if !onlyForICMP {
		for _, ip := range IPs {
			// do not add new IP if it already in exceptions
			if _, exists := allowedHosts[ip]; exists == false {
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
			if _, exists := allowedForICMP[ip]; exists == false {
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
	} else if (allowedForICMP != nil){
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
//		(has 'true' value in allowedHosts; eg.: LAN and Multicast connectivity)
func removeAllHostsFromExceptions() error {
	toRemoveIPs := make([]string, 0, len(allowedHosts))
	for ipStr := range allowedHosts {
		toRemoveIPs = append(toRemoveIPs, ipStr)
	}
	isPersistant := false
	return removeHostsFromExceptions(toRemoveIPs, isPersistant, false)
}

//---------------------------------------------------------------------

// getLanIPs - returns list of local IPs
func getLanIPs() ([]string, error) {

	ipnetList, err := netinfo.GetAllLocalV4Addresses()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	retIps := make([]string, 0, 4)
	for _, ifs := range ipnetList {
		// Skip localhost interface - we have separate rules for local iface
		if ifs.IP.String() == "127.0.0.1" {
			continue
		}
		if len(connectedVpnLocalIP) > 0 && ifs.IP.String() == connectedVpnLocalIP {
			continue
		}
		retIps = append(retIps, ifs.String())
	}

	return retIps, nil
}
