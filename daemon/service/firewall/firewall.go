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
	"unicode"

	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/service/dns"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("frwl")
}

var (
	connectedClientInterfaceIP   net.IP
	connectedClientInterfaceIPv6 net.IP
	connectedClientPort          int
	connectedHostIP              net.IP
	connectedHostPort            int
	connectedIsTCP               bool
	mutex                        sync.Mutex
	isClientPaused               bool
	dnsConfig                    *dns.DnsSettings

	// List of IP masks that are allowed for any communication
	userExceptions []net.IPNet

	stateAllowLan          bool
	stateAllowLanMulticast bool
)

// Initialize is doing initialization stuff
// Must be called on application start
func Initialize() error {
	return implInitialize()
}

// SetEnabled - change firewall state
func SetEnabled(enable bool) error {
	mutex.Lock()
	defer mutex.Unlock()

	if enable {
		log.Info("Enabling...")
	} else {
		log.Info("Disabling...")
	}

	err := implSetEnabled(enable)
	if err != nil {
		log.Error(err)
		return fmt.Errorf("failed to change firewall state : %w", err)
	}

	if enable {
		// To fulfill such flow (example): FWEnable -> Connected -> FWDisable -> FWEnable
		// Here we should notify that client is still connected
		// We must not do it in Paused state!
		clientAddr := connectedClientInterfaceIP
		clientAddrIPv6 := connectedClientInterfaceIPv6
		if clientAddr != nil && !isClientPaused {
			e := implClientConnected(clientAddr, clientAddrIPv6, connectedClientPort, connectedHostIP, connectedHostPort, connectedIsTCP)
			if e != nil {
				log.Error(e)
			}
		}
	}
	return err
}

// SetPersistant - set persistant firewall state and enable it if necessary
func SetPersistant(persistant bool) error {
	mutex.Lock()
	defer mutex.Unlock()

	log.Info(fmt.Sprintf("Persistent:%t", persistant))

	err := implSetPersistant(persistant)
	if err != nil {
		log.Error(err)
	}
	return err
}

// GetEnabled - get firewall status enabled/disabled
func GetEnabled() (bool, error) {
	mutex.Lock()
	defer mutex.Unlock()

	ret, err := implGetEnabled()
	if err != nil {
		log.Error("Status check error: ", err)
	}

	return ret, err
}

func GetState() (isEnabled, isLanAllowed, isMulticatsAllowed bool, err error) {
	mutex.Lock()
	defer mutex.Unlock()

	ret, err := implGetEnabled()
	if err != nil {
		log.Error("Status check error: ", err)
	}

	return ret, stateAllowLan, stateAllowLanMulticast, err
}

// SingleDnsRuleOn - add rule to allow DNS communication with specified IP only
// (usefull for Inverse Split Tunneling feature)
// Returns error if IVPN firewall is enabled.
// As soon as IVPN firewall enables - this rule will be removed
func SingleDnsRuleOn(dnsAddr net.IP) (retErr error) {
	mutex.Lock()
	defer mutex.Unlock()
	return implSingleDnsRuleOn(dnsAddr)
}

// SingleDnsRuleOff - remove rule (if exist) to allow DNS communication with specified IP only defined by SingleDnsRuleOn()
// (usefull for Inverse Split Tunneling feature)
func SingleDnsRuleOff() (retErr error) {
	mutex.Lock()
	defer mutex.Unlock()
	return implSingleDnsRuleOff()
}

// ClientPaused saves info about paused state of vpn
func ClientPaused() {
	isClientPaused = true
}

// ClientResumed saves info about resumed state of vpn
func ClientResumed() {
	isClientPaused = false
}

// ClientConnected - allow communication for local vpn/client IP address
func ClientConnected(clientLocalIPAddress net.IP, clientLocalIPv6Address net.IP, clientPort int, serverIP net.IP, serverPort int, isTCP bool) error {
	mutex.Lock()
	defer mutex.Unlock()
	ClientResumed()

	log.Info("Client connected: ", clientLocalIPAddress)

	connectedClientInterfaceIP = clientLocalIPAddress
	connectedClientInterfaceIPv6 = clientLocalIPv6Address
	connectedClientPort = clientPort
	connectedHostIP = serverIP
	connectedHostPort = serverPort
	connectedIsTCP = isTCP

	err := implClientConnected(clientLocalIPAddress, clientLocalIPv6Address, clientPort, serverIP, serverPort, isTCP)
	if err != nil {
		log.Error(err)
	}
	return err
}

// ClientDisconnected - Remove all hosts exceptions
func ClientDisconnected() error {
	mutex.Lock()
	defer mutex.Unlock()
	ClientResumed()

	// Remove client interface from exceptions
	if connectedClientInterfaceIP != nil {
		connectedClientInterfaceIP = nil
		connectedClientInterfaceIPv6 = nil
		log.Info("Client disconnected")
		err := implClientDisconnected()
		if err != nil {
			log.Error(err)
		}
		return err
	}
	return nil
}

// AddHostsToExceptions - allow comminication with this hosts
// Note: if isPersistent == false -> all added hosts will be removed from exceptions after client disconnection (after call 'ClientDisconnected()')
// Arguments:
//   - IPs			-	list of IP addresses to ba allowed
//   - onlyForICMP	-	(applicable only for Linux) try add rule to allow only ICMP protocol for this IP
//   - isPersistent	-	keep rule enabled even if VPN disconnected
func AddHostsToExceptions(IPs []net.IP, onlyForICMP bool, isPersistent bool) error {
	mutex.Lock()
	defer mutex.Unlock()

	err := implAddHostsToExceptions(IPs, onlyForICMP, isPersistent)
	if err != nil {
		log.Error("Failed to add hosts to exceptions:", err)
	}

	return err
}

func RemoveHostsFromExceptions(IPs []net.IP, onlyForICMP bool, isPersistent bool) error {
	mutex.Lock()
	defer mutex.Unlock()

	err := implRemoveHostsFromExceptions(IPs, onlyForICMP, isPersistent)
	if err != nil {
		log.Error("Failed to remove hosts from exceptions:", err)
	}

	return err
}

// AllowLAN - allow/forbid LAN communication
func AllowLAN(allowLan bool, allowLanMulticast bool) error {
	mutex.Lock()
	defer mutex.Unlock()

	stateAllowLan = allowLan
	stateAllowLanMulticast = allowLanMulticast

	log.Info(fmt.Sprintf("allowLan:%t allowMulticast:%t", allowLan, allowLanMulticast))

	err := implAllowLAN(allowLan, allowLanMulticast)
	if err != nil {
		log.Error(err)
	}
	return err
}

func GetDnsInfo() (dns.DnsSettings, bool) {
	mutex.Lock()
	defer mutex.Unlock()

	if dnsConfig == nil {
		return dns.DnsSettings{}, false
	}
	return *dnsConfig, true
}

func getDnsIP() net.IP {
	cfg := dnsConfig

	var dnsIP net.IP
	if cfg != nil && cfg.Encryption == dns.EncryptionNone {
		dnsIP = cfg.Ip()
	}
	return dnsIP
}

// OnChangeDNS - must be called on each DNS change (to update firewall rules according to new DNS configuration)
func OnChangeDNS(newDnsCfg *dns.DnsSettings) error {
	mutex.Lock()
	defer mutex.Unlock()

	if newDnsCfg != nil && newDnsCfg.IsEmpty() {
		newDnsCfg = nil
	}

	if (dnsConfig == nil && newDnsCfg == nil) ||
		(dnsConfig != nil && newDnsCfg != nil && dnsConfig.Equal(*newDnsCfg)) {
		// DNS rule already applied. Do nothing.
		return nil
	}

	var addr net.IP = nil
	if newDnsCfg != nil && newDnsCfg.Encryption == dns.EncryptionNone {
		// for DoH/DoT - no sense to allow DNS port (53)
		addr = net.ParseIP(newDnsCfg.DnsHost)
	}

	err := implOnChangeDNS(addr)
	if err != nil {
		log.Error(err)
	} else {
		// remember DNS IP
		dnsConfig = newDnsCfg
	}
	return err
}

// SetUserExceptions set ip/mask to be excluded from FW block
// Parameters:
//   - exceptions - comma separated list of IP addresses in format: x.x.x.x[/xx]
func SetUserExceptions(exceptions string, ignoreParseErrors bool) error {
	userExceptions = []net.IPNet{}

	splitFunc := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != rune('/') && c != rune('.') && c != rune(':')
	}
	exceptionsArr := strings.FieldsFunc(exceptions, splitFunc)
	for _, exp := range exceptionsArr {
		exp = strings.TrimSpace(exp)

		var err error
		var n *net.IPNet

		if strings.Contains(exp, "/") {
			_, n, err = net.ParseCIDR(exp)
		} else {
			addr := net.ParseIP(exp)
			if addr == nil {
				err = fmt.Errorf("%s not a IP address", exp)
			} else {
				if addr.To4() == nil {
					// IPv6 single address
					_, n, err = net.ParseCIDR(addr.String() + "/128")
				} else {
					// IPv4 single address
					_, n, err = net.ParseCIDR(addr.String() + "/32")
				}
			}
		}
		if err != nil {
			if !ignoreParseErrors {
				return fmt.Errorf("unable to parse firewall exceptions ('%s'): %w", exceptions, err)
			}
			continue
		}
		userExceptions = append(userExceptions, *n)
	}

	return implOnUserExceptionsUpdated()
}
