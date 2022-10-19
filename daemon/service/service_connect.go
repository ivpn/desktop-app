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

package service

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/helpers"
	"github.com/ivpn/desktop-app/daemon/obfsproxy"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/service/platform/filerights"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	"github.com/ivpn/desktop-app/daemon/vpn"
	"github.com/ivpn/desktop-app/daemon/vpn/openvpn"
	"github.com/ivpn/desktop-app/daemon/vpn/wireguard"
)

func (s *Service) GetConnectionParams() (preferences.ConnectionParams, error) {
	return s._preferences.LastConnectionParams, nil
}

func (s *Service) Connect(params preferences.ConnectionParams) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("panic on connect: " + fmt.Sprint(r))
			log.Error(err)
		}
	}()

	// keep last used connection params
	s.SetConnectionParams(params)

	retManualDNS := params.ManualDNS

	if vpn.Type(params.VpnType) == vpn.OpenVPN {
		// PARAMETERS VALIDATION
		// parsing hosts
		var hosts []net.IP
		for _, v := range params.OpenVpnParameters.EntryVpnServer.Hosts {
			hosts = append(hosts, net.ParseIP(v.Host))
		}
		if len(hosts) < 1 {
			return fmt.Errorf("VPN host not defined")
		}
		// in case of multiple hosts - take random host from the list
		host := hosts[0]
		if len(hosts) > 1 {
			if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(hosts)))); err == nil {
				host = hosts[rnd.Int64()]
			}
		}

		// nothing from supported proxy types should be in this parameter
		proxyType := params.OpenVpnParameters.Proxy.Type
		if len(proxyType) > 0 && proxyType != "http" && proxyType != "socks" {
			proxyType = ""
		}

		// Multi-Hop
		var exitHostValue *apitypes.OpenVPNServerHostInfo
		multihopExitHosts := params.OpenVpnParameters.MultihopExitServer.Hosts
		if len(multihopExitHosts) > 0 {
			exitHostValue = &multihopExitHosts[0]
			if len(multihopExitHosts) > 1 {
				if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(multihopExitHosts)))); err == nil {
					exitHostValue = &multihopExitHosts[rnd.Int64()]
				}
			}
		}

		// only one-line parameter is allowed
		proxyUsername := strings.Split(params.OpenVpnParameters.Proxy.Username, "\n")[0]
		proxyPassword := strings.Split(params.OpenVpnParameters.Proxy.Password, "\n")[0]

		// CONNECTION
		// OpenVPN connection parameters
		var connectionParams openvpn.ConnectionParams
		if exitHostValue != nil {
			// Check is it allowed to connect multihop
			if mhErr := s.IsCanConnectMultiHop(); mhErr != nil {
				return mhErr
			}

			// Multi-Hop
			connectionParams = openvpn.CreateConnectionParams(
				exitHostValue.Hostname,
				params.OpenVpnParameters.Port.Protocol > 0, // is TCP
				exitHostValue.MultihopPort,
				host,
				proxyType,
				net.ParseIP(params.OpenVpnParameters.Proxy.Address),
				params.OpenVpnParameters.Proxy.Port,
				proxyUsername,
				proxyPassword)
		} else {
			// Single-Hop
			connectionParams = openvpn.CreateConnectionParams(
				"",
				params.OpenVpnParameters.Port.Protocol > 0, // is TCP
				params.OpenVpnParameters.Port.Port,
				host,
				proxyType,
				net.ParseIP(params.OpenVpnParameters.Proxy.Address),
				params.OpenVpnParameters.Proxy.Port,
				proxyUsername,
				proxyPassword)
		}

		return s.ConnectOpenVPN(connectionParams, retManualDNS, params.FirewallOn, params.FirewallOnDuringConnection)

	} else if vpn.Type(params.VpnType) == vpn.WireGuard {
		hosts := params.WireGuardParameters.EntryVpnServer.Hosts
		multihopExitHosts := params.WireGuardParameters.MultihopExitServer.Hosts

		// filter hosts: use IPv6 hosts
		if params.IPv6 {
			ipv6Hosts := append(hosts[0:0], hosts...)
			n := 0
			for _, h := range ipv6Hosts {
				if h.IPv6.LocalIP != "" {
					ipv6Hosts[n] = h
					n++
				}
			}
			if n == 0 {
				if params.IPv6Only {
					return fmt.Errorf("unable to make IPv6 connection inside tunnel. Server does not support IPv6")
				}
			} else {
				hosts = ipv6Hosts[:n]
			}
		}

		// filter exit servers (Multi-Hop connection):
		// 1) each exit server must have initialized 'multihop_port' field
		// 2) (in case of IPv6Only) IPv6 local address should be defined
		if len(multihopExitHosts) > 0 {
			isHasMHPort := false
			ipv6ExitHosts := append(multihopExitHosts[0:0], multihopExitHosts...)
			n := 0
			for _, h := range ipv6ExitHosts {
				if h.MultihopPort == 0 {
					continue
				}
				isHasMHPort = true
				if params.IPv6 && h.IPv6.LocalIP == "" {
					continue
				}

				ipv6ExitHosts[n] = h
				n++
			}
			if n == 0 {
				if !isHasMHPort {
					return fmt.Errorf("unable to make Multi-Hop connection inside tunnel. Exit server does not support Multi-Hop")
				}
				if params.IPv6Only {
					return fmt.Errorf("unable to make IPv6 Multi-Hop connection inside tunnel. Exit server does not support IPv6")
				}
			} else {
				multihopExitHosts = ipv6ExitHosts[:n]
			}
		}

		hostValue := hosts[0]
		if len(hosts) > 1 {
			if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(hosts)))); err == nil {
				hostValue = hosts[rnd.Int64()]
			}
		}

		var exitHostValue *apitypes.WireGuardServerHostInfo
		if len(multihopExitHosts) > 0 {
			exitHostValue = &multihopExitHosts[0]
			if len(multihopExitHosts) > 1 {
				if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(multihopExitHosts)))); err == nil {
					exitHostValue = &multihopExitHosts[rnd.Int64()]
				}
			}
		}

		// prevent user-defined data injection: ensure that nothing except the base64 public key will be stored in the configuration
		if !helpers.ValidateBase64(hostValue.PublicKey) {
			return fmt.Errorf("WG public key is not base64 string")
		}

		hostLocalIP := net.ParseIP(strings.Split(hostValue.LocalIP, "/")[0])
		ipv6Prefix := ""
		if params.IPv6 {
			ipv6Prefix = strings.Split(hostValue.IPv6.LocalIP, "/")[0]
		}

		var connectionParams wireguard.ConnectionParams
		if exitHostValue != nil {
			// Check is it allowed to connect multihop
			if mhErr := s.IsCanConnectMultiHop(); mhErr != nil {
				return mhErr
			}

			// Multi-Hop
			connectionParams = wireguard.CreateConnectionParams(
				exitHostValue.Hostname,
				exitHostValue.MultihopPort,
				net.ParseIP(hostValue.Host),
				exitHostValue.PublicKey,
				hostLocalIP,
				ipv6Prefix,
				params.WireGuardParameters.Mtu)
		} else {
			// Single-Hop
			connectionParams = wireguard.CreateConnectionParams(
				"",
				params.WireGuardParameters.Port.Port,
				net.ParseIP(hostValue.Host),
				hostValue.PublicKey,
				hostLocalIP,
				ipv6Prefix,
				params.WireGuardParameters.Mtu)
		}

		return s.ConnectWireGuard(connectionParams, retManualDNS, params.FirewallOn, params.FirewallOnDuringConnection)

	}

	return fmt.Errorf("unexpected VPN type to connect (%v)", params.VpnType)
}

// ConnectOpenVPN start OpenVPN connection
func (s *Service) ConnectOpenVPN(connectionParams openvpn.ConnectionParams, manualDNS dns.DnsSettings, firewallOn bool, firewallDuringConnection bool) error {

	createVpnObjfunc := func() (vpn.Process, error) {
		prefs := s.Preferences()

		// checking if functionality accessible
		disabledFuncs := s.GetDisabledFunctions()
		if len(disabledFuncs.OpenVPNError) > 0 {
			return nil, fmt.Errorf(disabledFuncs.OpenVPNError)
		}
		if prefs.Obfs4proxy.IsObfsproxy() && len(disabledFuncs.ObfsproxyError) > 0 {
			return nil, fmt.Errorf(disabledFuncs.ObfsproxyError)
		}

		connectionParams.SetCredentials(prefs.Session.OpenVPNUser, prefs.Session.OpenVPNPass)

		openVpnExtraParameters := ""
		// read user-defined extra parameters for OpenVPN configuration (if exists)
		extraParamsFile := platform.OpenvpnUserParamsFile()

		if helpers.FileExists(extraParamsFile) {
			if err := filerights.CheckFileAccessRightsConfig(extraParamsFile); err != nil {
				log.Info("NOTE! User-defined OpenVPN parameters are ignored! %w", err)
				os.Remove(extraParamsFile)
			} else {
				// read file line by line
				openVpnExtraParameters = func() string {
					var allParams strings.Builder

					file, err := os.Open(extraParamsFile)
					if err != nil {
						log.Error(err)
						return ""
					}
					defer file.Close()

					scanner := bufio.NewScanner(file)
					for scanner.Scan() {
						line := scanner.Text()
						line = strings.TrimSpace(line)
						if len(line) <= 0 {
							continue
						}
						if strings.HasPrefix(line, "#") {
							continue // comment
						}
						if strings.HasPrefix(line, ";") {
							continue // comment
						}
						allParams.WriteString(line + "\n")
					}

					if err := scanner.Err(); err != nil {
						log.Error("Failed to parse '%s': %s", extraParamsFile, err)
						return ""
					}
					return allParams.String()
				}()

				if len(openVpnExtraParameters) > 0 {
					log.Info(fmt.Sprintf("WARNING! User-defined OpenVPN parameters loaded from file '%s'!", extraParamsFile))
				}
			}
		}

		// initialize obfsproxy parameters
		obfsParams := openvpn.ObfsParams{}
		if prefs.Obfs4proxy.IsObfsproxy() {
			obfsParams.Config = prefs.Obfs4proxy
			svrs, err := s.ServersList()
			if err != nil {
				return nil, fmt.Errorf("failed to initialize obfsproxy configuration: %w", err)
			}

			if connectionParams.IsMultihop() {
				// find host by hostname
				host, err := s.findOpenVpnHost(connectionParams.GetMultihopExitHostName(), nil, svrs.OpenvpnServers)
				if err != nil {
					return nil, fmt.Errorf("failed to initialize obfsproxy configuration: %w", err)
				}

				switch obfsParams.Config.Version {
				case obfsproxy.OBFS3:
					obfsParams.RemotePort = host.Obfs.Obfs3MultihopPort
				case obfsproxy.OBFS4:
					obfsParams.RemotePort = host.Obfs.Obfs4MultihopPort
					obfsParams.Obfs4Key = host.Obfs.Obfs4Key
				default:
					return nil, fmt.Errorf("failed to initialize obfsproxy configuration: unsupported obfs version: %d", obfsParams.Config.Version)
				}
			} else {
				switch obfsParams.Config.Version {
				case obfsproxy.OBFS3:
					obfsParams.RemotePort = svrs.Config.Ports.Obfs3.Port
				case obfsproxy.OBFS4:
					{
						// find host by host ip
						host, err := s.findOpenVpnHost("", connectionParams.GetHostIp(), svrs.OpenvpnServers)
						if err != nil {
							return nil, fmt.Errorf("failed to initialize obfsproxy configuration: %w", err)
						}

						obfsParams.RemotePort = svrs.Config.Ports.Obfs4.Port
						obfsParams.Obfs4Key = host.Obfs.Obfs4Key
					}
				default:
					return nil, fmt.Errorf("failed to initialize obfsproxy configuration: unsupported obfs version: %d", obfsParams.Config.Version)
				}
			}

		}

		// creating OpenVPN object
		vpnObj, err := openvpn.NewOpenVpnObject(
			platform.OpenVpnBinaryPath(),
			platform.OpenvpnConfigFile(),
			"",
			obfsParams,
			openVpnExtraParameters,
			connectionParams)

		if err != nil {
			return nil, fmt.Errorf("failed to create new openVPN object: %w", err)
		}
		return vpnObj, nil
	}

	return s.keepConnection(createVpnObjfunc, manualDNS, firewallOn, firewallDuringConnection)
}

// ConnectWireGuard start WireGuard connection
func (s *Service) ConnectWireGuard(connectionParams wireguard.ConnectionParams, manualDNS dns.DnsSettings, firewallOn bool, firewallDuringConnection bool) error {
	// stop active connection (if exists)
	if err := s.Disconnect(); err != nil {
		return fmt.Errorf("failed to connect. Unable to stop active connection: %w", err)
	}

	// checking if functionality accessible
	disabledFuncs := s.GetDisabledFunctions()
	if len(disabledFuncs.WireGuardError) > 0 {
		return fmt.Errorf(disabledFuncs.WireGuardError)
	}

	// Update WG keys, if necessary
	err := s.WireGuardGenerateKeys(true)
	if err != nil {
		// If new WG keys regeneration failed but we still have active keys - keep connecting
		// (this could happen, for example, when FW is enabled and we even not tried to make API request)
		// Return error only if the keys had to be regenerated more than 3 days ago.
		_, activePublicKey, _, _, lastUpdate, interval := s.WireGuardGetKeys()

		if len(activePublicKey) > 0 && lastUpdate.Add(interval).Add(time.Hour*24*3).After(time.Now()) {
			// continue connection
			log.Warning(fmt.Errorf("WG KEY generation failed (%w). But we keep connecting (will try to regenerate it next 3 days)", err))
		} else {
			return err
		}
	}

	createVpnObjfunc := func() (vpn.Process, error) {
		session := s.Preferences().Session

		if !session.IsWGCredentialsOk() {
			return nil, fmt.Errorf("WireGuard credentials are not defined (please, regenerate WG credentials or re-login)")
		}

		localip := net.ParseIP(session.WGLocalIP)
		if localip == nil {
			return nil, fmt.Errorf("error updating WG connection preferences (failed parsing local IP for WG connection)")
		}
		connectionParams.SetCredentials(session.WGPrivateKey, localip)

		vpnObj, err := wireguard.NewWireGuardObject(
			platform.WgBinaryPath(),
			platform.WgToolBinaryPath(),
			platform.WGConfigFilePath(),
			connectionParams)

		if err != nil {
			return nil, fmt.Errorf("failed to create new WireGuard object: %w", err)
		}
		return vpnObj, nil
	}

	return s.keepConnection(createVpnObjfunc, manualDNS, firewallOn, firewallDuringConnection)
}
