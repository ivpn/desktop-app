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

// Useful commands for testing:
// sudo pfctl -a ivpn_firewall -s rules
// sudo pfctl -a ivpn_firewall/tunnel -s rules
// sudo pfctl -a ivpn_firewall -s Tables
// sudo pfctl -a ivpn_firewall -t ivpn_servers -T show

var (
	// key: is a string representation of allowed IP
	// value: true - if exception rule is persistant (persistant, means will stay available even client is disconnected)
	allowedHosts map[string]bool

	delayedAllowLanAllowed bool = true
	delayedAllowLanStarted bool = false
)

const (
	multicastIP = "224.0.0.0/4"

	// An IPv6 local addresses
	// Apple services is using IPv6 addressing (AirDrop, UniversalClipboard, HandOff, CallForwarding iPhone<->Mac ...)
	// AirDrop is using 'awdl0' interface
	// Rest Apple services are using 'utun0', 'utun1'... 'utunX'...
	localhostIPv6 = "fe80::/64"
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
	return shell.Exec(nil, platform.FirewallScript(), "-disable")
}

func implSetPersistant(persistant bool) error {
	if persistant {
		// The persistence is based on such facts:
		// 	- daemon is starting as 'LaunchDaemons'
		// 	- SetPersistant() called by service object on daemon start
		// This means we just have to ensure that firewall enabled.

		// Just ensure that firewall is enabled
		return implSetEnabled(true)
	}
	return nil
}

// ClientConnected - allow communication for local vpn/client IP address
func implClientConnected(clientLocalIPAddress net.IP, clientPort int, serverIP net.IP, serverPort int, isTCP bool) error {
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
	return removeHostsFromExceptions([]string{serverIP.String()})
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
	localIPs, err := getLanIPs()
	if err != nil {
		return fmt.Errorf("failed to get local IPs: %w", err)
	}

	if !isAllowLAN {
		// LAN NOT ALLOWED
		delayedAllowLanAllowed = false
		// disallow everything (LAN + multicast)
		return removeHostsFromExceptions(append(localIPs, multicastIP))
	}

	// LAN ALLOWED
	if len(localIPs) > 0 {
		delayedAllowLanAllowed = false
	} else {
		// this can happen, for example, on system boot (when no network interfaces initialized)
		log.Info("Local LAN addresses not detected: no data to apply the 'Allow LAN' rule")
		go delayedAllowLAN(isAllowLanMulticast)
		return nil
	}

	// An IPv6 local addresses
	// Apple services is using IPv6 addressing (AirDrop, UniversalClipboard, HandOff, CallForwarding iPhone<->Mac ...)
	// AirDrop is using 'awdl0' interface
	// Rest Apple services are using 'utun0', 'utun1'... 'utunX'...
	localIPs = append(localIPs, localhostIPv6)

	if isAllowLanMulticast {
		// allow LAN + multicast
		return addHostsToExceptions(append(localIPs, multicastIP), true)
	}

	// disallow Multicast
	removeHostsFromExceptions([]string{multicastIP})
	// allow LAN
	return addHostsToExceptions(localIPs, true)
}

func delayedAllowLAN(isAllowLanMulticast bool) {
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
				err := implAllowLAN(true, isAllowLanMulticast)
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
//	* onlyForICMP	-	(not in use for macOS) try add rule to allow only ICMP protocol for this IP
//	* isPersistent	-	keep rule enabled even if VPN disconnected
func implAddHostsToExceptions(IPs []net.IP, onlyForICMP bool, isPersistent bool) error {
	IPsStr := make([]string, 0, len(IPs))
	for _, ip := range IPs {
		IPsStr = append(IPsStr, ip.String())
	}

	return addHostsToExceptions(IPsStr, isPersistent)
}

// SetManualDNS - configure firewall to allow DNS which is out of VPN tunnel
// Applicable to Windows implementation (to allow custom DNS from local network)
func implSetManualDNS(addr net.IP) error {
	// not in use for macOS
	return nil
}

//---------------------------------------------------------------------

func applyAddHostsToExceptions(hostsIPs []string) error { //
	var ipList string
	ipList = strings.Join(hostsIPs, " ")

	if len(ipList) > 0 {
		log.Info("-add_exceptions ", ipList)
		return shell.Exec(nil, platform.FirewallScript(), "-add_exceptions", ipList)
	}
	return nil
}

func applyRemoveHostsFromExceptions(hostsIPs []string) error {
	var ipList string
	ipList = strings.Join(hostsIPs, " ")

	if len(ipList) > 0 {
		log.Info("-remove_exceptions ", ipList)
		return shell.Exec(nil, platform.FirewallScript(), "-remove_exceptions", ipList)
	}
	return nil
}

func reApplyExceptions() error {

	// Allow LAN communication (if necessary)
	// Restore all exceptions (all hosts which are allowed)
	allowedIPs := make([]string, 0, len(allowedHosts)+2)
	if len(allowedHosts) > 0 {
		for ipStr := range allowedHosts {
			allowedIPs = append(allowedIPs, ipStr)
		}
	}

	// Apply all allowed hosts
	err := applyAddHostsToExceptions(allowedIPs)
	if err != nil {
		log.Error(err)
	}
	return err
}

//---------------------------------------------------------------------

// allow communication with specified hosts
// if isPersistant == false - exception will be removed when client disctonnects
func addHostsToExceptions(IPs []string, isPersistant bool) error {
	if len(IPs) == 0 {
		return nil
	}

	newIPs := make([]string, 0, len(IPs))
	for _, ip := range IPs {
		// do not add new IP if it already in exceptions
		if _, exists := allowedHosts[ip]; exists == false {
			allowedHosts[ip] = isPersistant // add to map
			newIPs = append(newIPs, ip)
		}
	}

	if len(newIPs) == 0 {
		return nil
	}

	err := applyAddHostsToExceptions(newIPs)
	if err != nil {
		log.Error(err)
	}
	return err
}

// Deprecate comminication with this hosts
func removeHostsFromExceptions(IPs []string) error {
	if len(IPs) == 0 {
		return nil
	}

	toRemoveIPs := make([]string, 0, len(IPs))
	for _, ip := range IPs {
		if _, exists := allowedHosts[ip]; exists {
			delete(allowedHosts, ip) // remove from map
			toRemoveIPs = append(toRemoveIPs, ip)
		}
	}

	if len(toRemoveIPs) == 0 {
		return nil
	}

	err := applyRemoveHostsFromExceptions(toRemoveIPs)
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
	for ipStr, isPersistant := range allowedHosts {
		if isPersistant {
			continue
		}
		toRemoveIPs = append(toRemoveIPs, ipStr)
		delete(allowedHosts, ipStr) // erase map
	}

	return removeHostsFromExceptions(toRemoveIPs)
}

//---------------------------------------------------------------------

// getLanIPs - returns list of local IPs
func getLanIPs() ([]string, error) {

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	retIps := make([]string, 0, 4)

	for _, ifs := range ifaces {
		addrs, _ := ifs.Addrs()
		if addrs == nil {
			continue
		}

		// Skip local tunnel addresses as they are not a real LAN addresses
		if strings.Contains(ifs.Name, "tun") {
			continue
		}

		for _, addr := range addrs {
			addrStr := addr.String()
			if addrStr == "" {
				continue
			}

			// Skip local interface - we have separate rules for local iface
			if strings.HasPrefix(addrStr, "127.") {
				continue
			}

			// ensure IP is IPv4
			ipaddrStr := strings.Split(addrStr, "/")[0] // 192.168.1.106/24 => 192.168.1.106
			ip := net.ParseIP(ipaddrStr)
			if ip.To4() == nil {
				// skip non IPv4
				continue
			}

			retIps = append(retIps, addrStr)
		}
	}

	return retIps, nil
}
