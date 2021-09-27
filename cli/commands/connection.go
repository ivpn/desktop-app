//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the IVPN command line interface.
//
//  The IVPN command line interface is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The IVPN command line interface is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the IVPN command line interface. If not, see <https://www.gnu.org/licenses/>.
//

package commands

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/ivpn/desktop-app/cli/commands/config"
	"github.com/ivpn/desktop-app/cli/flags"
	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/srverrors"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

type port struct {
	port int
	tcp  bool
}

func (p *port) IsTCP() int {
	if p.tcp {
		return 1
	}
	return 0
}

func (p *port) String() string {
	protoName := "UDP"
	if p.tcp {
		protoName = "TCP"
	}
	return fmt.Sprintf("%s:%d", protoName, p.port)
}

var (
	portsWireGuard = [...]port{
		port{port: 2049},
		port{port: 2050},
		port{port: 53},
		port{port: 1194},
		port{port: 30587},
		port{port: 41893},
		port{port: 48574},
		port{port: 58237}}

	portsOpenVpn = [...]port{
		port{port: 2049},
		port{port: 2050},
		port{port: 53},
		port{port: 1194},
		port{port: 443, tcp: true},
		port{port: 1443, tcp: true},
		port{port: 80, tcp: true}}
)

type CmdDisconnect struct {
	flags.CmdInfo
}

func (c *CmdDisconnect) Init() {
	c.Initialize("disconnect", "Disconnect active VPN connection (if connected)")
}
func (c *CmdDisconnect) Run() error {
	if err := _proto.DisconnectVPN(); err != nil {
		return err
	}

	showState()

	return nil
}

//-----------------------------------------------

type CmdConnect struct {
	flags.CmdInfo
	last            bool
	gateway         string
	port            string
	any             bool
	obfsproxy       bool
	firewallOff     bool
	dns             string
	antitracker     bool
	antitrackerHard bool
	isIPv6Tunnel    bool

	filter_proto       string
	filter_location    bool
	filter_city        bool
	filter_country     bool
	filter_countryCode bool
	filter_invert      bool

	multihopExitSvr string

	fastest bool
}

func (c *CmdConnect) Init() {
	c.Initialize("connect", "Establish new VPN connection\nLOCATION can be a mask for filtering servers (see 'servers' command)")
	c.DefaultStringVar(&c.gateway, "LOCATION")

	c.StringVar(&c.port, "port", "", "PROTOCOL:PORT", fmt.Sprintf("Port to connect to (default: %s - OpenVPN, %s - WireGuard)\nOpenVPN: %s\nWireGuard: %s",
		portsOpenVpn[0].String(), portsWireGuard[0].String(),
		allPortsString(portsOpenVpn[:]), allPortsString(portsWireGuard[:])))

	c.BoolVar(&c.any, "any", false, "When LOCATION points to more than one server, use first found server to connect")

	c.BoolVar(&c.obfsproxy, "o", false, "OpenVPN only: Use obfsproxy")
	c.BoolVar(&c.obfsproxy, "obfsproxy", false, "OpenVPN only: Use obfsproxy")

	c.StringVar(&c.multihopExitSvr, "exit_svr", "", "LOCATION", "OpenVPN only: Exit-server for Multi-Hop connection\n(use full serverID as a parameter, servers filtering not applicable for it)")

	c.BoolVar(&c.firewallOff, "fw_off", false, "Do not enable firewall for this connection\n(has effect only if Firewall not enabled before)")

	c.StringVar(&c.dns, "dns", "", "DNS_IP", "Use custom DNS for this connection\n(if 'antitracker' is enabled - this parameter will be ignored)")

	c.BoolVar(&c.antitracker, "antitracker", false, "Enable AntiTracker for this connection")
	c.BoolVar(&c.antitrackerHard, "antitracker_hard", false, "Enable 'Hard Core' AntiTracker for this connection")
	c.BoolVar(&c.isIPv6Tunnel, "ipv6tunnel", false, "Enable IPv6 in VPN tunnel (WireGuard connections only)\n(IPv6 addresses are preferred when a host has a dual stack IPv6/IPv4; IPv4-only hosts are unaffected)")

	// filters
	c.StringVar(&c.filter_proto, "p", "", "PROTOCOL", "Protocol type OpenVPN|ovpn|WireGuard|wg")
	c.StringVar(&c.filter_proto, "protocol", "", "PROTOCOL", "Protocol type OpenVPN|ovpn|WireGuard|wg")

	c.BoolVar(&c.filter_location, "l", false, "Apply LOCATION as a filter to server location (Hostname)")
	c.BoolVar(&c.filter_location, "location", false, "Apply LOCATION as a filter to server location (Hostname)")

	c.BoolVar(&c.filter_country, "c", false, "Apply LOCATION as a filter to country name")
	c.BoolVar(&c.filter_country, "country", false, "Apply LOCATION as a filter to country name")

	c.BoolVar(&c.filter_countryCode, "cc", false, "Apply LOCATION as a filter to country code")
	c.BoolVar(&c.filter_countryCode, "country_code", false, "Apply LOCATION as a filter to country code")

	c.BoolVar(&c.filter_city, "city", false, "Apply LOCATION as a filter to city name")

	c.BoolVar(&c.filter_invert, "filter_invert", false, "Invert filtering")

	c.BoolVar(&c.fastest, "fastest", false, "Connect to fastest server")

	c.BoolVar(&c.last, "last", false, "Connect with last successful connection parameters")
}

// Run executes command
func (c *CmdConnect) Run() (retError error) {
	if len(c.gateway) == 0 && c.fastest == false && c.last == false {
		return flags.BadParameter{}
	}

	// show current state after on finished
	defer func() {
		if retError == nil {
			showState()
		}
	}()

	// connection request
	req := types.Connect{}

	// get servers list from daemon
	serverFound := false
	servers, err := _proto.GetServers()
	if err != nil {
		return err
	}

	// check is logged-in
	helloResp := _proto.GetHelloResponse()
	if len(helloResp.Command) > 0 && (len(helloResp.Session.Session) == 0) {
		// We received 'hello' response but no session info - print tips to login
		fmt.Println("Error: Not logged in")

		fmt.Println()
		PrintTips([]TipType{TipLogin})
		fmt.Println()

		return srverrors.ErrorNotLoggedIn{}
	}

	// requesting servers list
	svrs := serversList(servers)

	// check which VPN protocols can be used
	isWgDisabled := len(helloResp.DisabledFunctions.WireGuardError) > 0
	isOpenVPNDisabled := len(helloResp.DisabledFunctions.OpenVPNError) > 0
	funcWarnDisabledProtocols := func() {
		if isOpenVPNDisabled {
			fmt.Println("WARNING: OpenVPN functionality disabled:\n\t", helloResp.DisabledFunctions.OpenVPNError)
		}
		if isWgDisabled {
			fmt.Println("WARNING: WireGuard functionality disabled:\n\t", helloResp.DisabledFunctions.WireGuardError)
		}
	}

	// do we need to connect with last successful connection parameters
	if c.last {
		fmt.Println("Enabled '-last' parameter. Using parameters from last successful connection")
		ci := config.RestoreLastConnectionInfo()
		if ci == nil {
			return fmt.Errorf("no information about last connection")
		}

		// reset filters
		c.filter_proto = ""
		c.filter_location = false
		c.filter_city = false
		c.filter_countryCode = false
		c.filter_country = false
		c.filter_invert = false

		// load last connection parameters
		c.gateway = ci.Gateway
		c.port = ci.Port
		c.obfsproxy = ci.Obfsproxy
		c.firewallOff = ci.FirewallOff
		c.dns = ci.DNS
		c.antitracker = ci.Antitracker
		c.antitrackerHard = ci.AntitrackerHard
		c.multihopExitSvr = ci.MultiopExitSvr
		c.isIPv6Tunnel = ci.IPv6Tunnel
	}

	if c.obfsproxy && len(helloResp.DisabledFunctions.ObfsproxyError) > 0 {
		return fmt.Errorf(helloResp.DisabledFunctions.ObfsproxyError)
	}

	// MULTI\SINGLE -HOP
	if len(c.multihopExitSvr) > 0 {
		if isOpenVPNDisabled {
			return fmt.Errorf(helloResp.DisabledFunctions.OpenVPNError)
		}
		// MULTI-HOP
		if c.fastest {
			return flags.BadParameter{Message: "'fastest' flag is not applicable for Multi-Hop connection [exit_svr]"}
		}

		if len(c.filter_proto) > 0 {
			pType, err := getVpnTypeByFlag(c.filter_proto)
			if err != nil || pType != vpn.OpenVPN {
				return flags.BadParameter{Message: "protocol flag [fp] is not applicable for Multi-Hop connection [exit_svr], only OpenVPN connection allowed"}
			}
		}

		if c.filter_location || c.filter_city || c.filter_countryCode || c.filter_country || c.filter_invert {
			fmt.Println("WARNING: filtering flags are ignored for Multi-Hop connection [exit_svr]")
		}

		entrySvrs := serversFilter(isWgDisabled, isOpenVPNDisabled, svrs, c.gateway, ProtoName_OpenVPN, false, false, false, false, false)
		if len(entrySvrs) == 0 || len(entrySvrs) > 1 {
			return flags.BadParameter{Message: "specify correct entry server ID for multi-hop connection"}
		}

		exitSvrs := serversFilter(isWgDisabled, isOpenVPNDisabled, svrs, c.multihopExitSvr, ProtoName_OpenVPN, false, false, false, false, false)
		if len(exitSvrs) == 0 || len(exitSvrs) > 1 {
			return flags.BadParameter{Message: "specify correct exit server ID for multi-hop connection"}
		}
		entrySvr := entrySvrs[0]
		exitSvr := exitSvrs[0]

		if entrySvr.gateway == exitSvr.gateway || entrySvr.countryCode == exitSvr.countryCode {
			return flags.BadParameter{Message: "unable to use entry- and exit- servers from the same country for multi-hop connection"}
		}

		c.gateway = entrySvr.gateway
		c.multihopExitSvr = exitSvr.gateway
	} else {
		//SINGLE-HOP
		svrs = serversFilter(isWgDisabled, isOpenVPNDisabled, svrs, c.gateway, c.filter_proto, c.filter_location, c.filter_city, c.filter_countryCode, c.filter_country, c.filter_invert)

		srvID := ""

		// Fastest server
		if c.fastest && len(svrs) > 1 {
			if err := serversPing(svrs, true); err != nil && c.any == false {
				if c.any {
					fmt.Printf("Error: Failed to ping servers to determine fastest: %s\n", err)
				} else {
					return err
				}
			}
			fastestSrv := svrs[len(svrs)-1]
			if fastestSrv.pingMs == 0 {
				fmt.Println("WARNING! Servers pinging problem.")
			}
			srvID = fastestSrv.gateway
		}

		// if we not found required server before (by 'fastest' option)
		if len(srvID) == 0 {
			showTipsServerFilterError := func() {
				fmt.Println()

				tips := []TipType{TipServers, TipConnectHelp}
				if config.LastConnectionExist() {
					tips = append(tips, TipLastConnection)
				}
				PrintTips(tips)
			}

			// no servers found
			if len(svrs) == 0 {
				fmt.Println("No servers found by your filter")
				fmt.Println("Please specify server more correctly")

				funcWarnDisabledProtocols() // print info about disabled functionality
				showTipsServerFilterError()
				return fmt.Errorf("no servers found by your filter")
			}

			// 'any' option
			if len(svrs) > 1 {
				fmt.Println("More than one server found")

				if c.any == false {
					fmt.Println("Please specify server more correctly or use flag '-any'")
					showTipsServerFilterError()
					return fmt.Errorf("more than one server found")
				}
				fmt.Printf("Taking first found server\n")
			}
			srvID = svrs[0].gateway
		}
		c.gateway = srvID
	}

	// Firewall for current connection
	req.FirewallOnDuringConnection = true
	if c.firewallOff {
		// check current FW state
		state, err := _proto.FirewallStatus()
		if err != nil {
			return fmt.Errorf("unable to check Firewall state: %w", err)
		}
		if state.IsEnabled == false {
			req.FirewallOnDuringConnection = false
		} else {
			fmt.Println("WARNING! Firewall option ignored (Firewall already enabled manually)")
		}
	}
	// get configuration
	cfg, _ := config.GetConfig()

	// set antitracker DNS (if defined). It will overwrite 'custom DNS' parameter
	if c.antitracker == false && c.antitrackerHard == false {
		// AntiTracker parameters not defined for current connection
		// Taking default configuration parameters (if defined)
		if cfg.Antitracker || cfg.AntitrackerHardcore {
			// print info
			printAntitrackerConfigInfo(nil, cfg.Antitracker, cfg.AntitrackerHardcore).Flush()
			// set values
			c.antitracker = cfg.Antitracker
			c.antitrackerHard = cfg.AntitrackerHardcore
		}
	}
	if c.antitracker || c.antitrackerHard {
		atDNS, err := GetAntitrackerIP(c.antitrackerHard, len(c.multihopExitSvr) > 0, &servers)
		if err != nil {
			return err
		}
		req.CurrentDNS = atDNS

		if len(c.dns) > 0 {
			fmt.Println("WARNING! Manual DNS configuration ignored due to AntiTracker")
		}
	}

	// set Manual DNS if defined (only in case if AntiTracker not defined)
	if len(req.CurrentDNS) == 0 {
		if len(c.dns) > 0 {
			dns := net.ParseIP(c.dns)
			if dns == nil {
				return flags.BadParameter{}
			}
			req.CurrentDNS = dns.String()
		} else if len(cfg.CustomDNS) > 0 {
			// using default DNS configuration
			printDNSConfigInfo(nil, cfg.CustomDNS).Flush()
			req.CurrentDNS = cfg.CustomDNS
		}
	}

	// looking for connection server
	// WireGuard
	for _, s := range servers.WireguardServers {
		if s.Gateway == c.gateway {

			serverFound = true
			host := s.Hosts[0]
			req.VpnType = vpn.WireGuard
			req.WireGuardParameters.EntryVpnServer.Hosts = []apitypes.WireGuardServerHostInfo{host}
			req.IPv6 = c.isIPv6Tunnel

			// port
			p, err := getPort(vpn.WireGuard, c.port)
			if err != nil {
				return err
			}
			req.WireGuardParameters.Port.Port = p.port

			fmt.Printf("[WireGuard] Connecting to: %s, %s (%s) %s %s...\n", s.City, s.CountryCode, s.Country, s.Gateway, p.String())

			break
		}
	}
	// OpenVPN
	if serverFound == false {
		var entrySvr *apitypes.OpenvpnServerInfo = nil
		var exitSvr *apitypes.OpenvpnServerInfo = nil

		// exit server
		if len(c.multihopExitSvr) > 0 {
			for _, s := range servers.OpenvpnServers {
				if s.Gateway == c.multihopExitSvr {
					exitSvr = &s
					break
				}
			}
		}

		var destPort port
		// entry server
		for _, s := range servers.OpenvpnServers {
			if s.Gateway == c.gateway {
				entrySvr = &s
				// TODO: obfsproxy configuration for this connection must be sent in 'Connect' request (avoid using daemon preferences)
				if err = _proto.SetPreferences("enable_obfsproxy", fmt.Sprint(c.obfsproxy)); err != nil {
					return err
				}

				serverFound = true
				req.VpnType = vpn.OpenVPN
				req.OpenVpnParameters.EntryVpnServer.IPAddresses = s.IPAddresses

				// port
				destPort, err = getPort(vpn.OpenVPN, c.port)
				if err != nil {
					return err
				}
				req.OpenVpnParameters.Port.Port = destPort.port
				req.OpenVpnParameters.Port.Protocol = destPort.IsTCP()

				if len(c.multihopExitSvr) > 0 {
					// get Multi-Hop ID
					req.OpenVpnParameters.MultihopExitSrvID = strings.Split(c.multihopExitSvr, ".")[0]
				}
				break
			}
			if len(c.multihopExitSvr) == 0 {
				if entrySvr != nil {
					break
				}
				if entrySvr != nil && exitSvr != nil {
					break
				}
			}
		}

		if entrySvr == nil {
			return fmt.Errorf("serverID not found in servers list (%s)", c.gateway)
		}
		if len(c.multihopExitSvr) > 0 && exitSvr == nil {
			return fmt.Errorf("serverID not found in servers list (%s)", c.multihopExitSvr)
		}

		if len(c.multihopExitSvr) == 0 {
			fmt.Printf("[OpenVPN] Connecting to: %s, %s (%s) %s %s...\n", entrySvr.City, entrySvr.CountryCode, entrySvr.Country, entrySvr.Gateway, destPort.String())
		} else {
			fmt.Printf("[OpenVPN] Connecting Multi-Hop...\n")
			fmt.Printf("\tentry server: %s, %s (%s) %s %s\n", entrySvr.City, entrySvr.CountryCode, entrySvr.Country, entrySvr.Gateway, destPort.String())
			fmt.Printf("\texit server : %s, %s (%s) %s\n", exitSvr.City, exitSvr.CountryCode, exitSvr.Country, exitSvr.Gateway)
		}
	}

	if serverFound == false {
		return fmt.Errorf("serverID not found in servers list (%s)", c.gateway)
	}

	fmt.Println("Connecting...")
	_, err = _proto.ConnectVPN(req)
	if err != nil {
		err = fmt.Errorf("failed to connect: %w", err)
		fmt.Printf("Disconnecting...\n")
		if err2 := _proto.DisconnectVPN(); err2 != nil {
			fmt.Printf("Failed to disconnect: %v\n", err2)
		}
		return err
	}

	// save last connection parameters
	config.SaveLastConnectionInfo(config.LastConnectionInfo{
		Gateway:         c.gateway,
		Port:            c.port,
		Obfsproxy:       c.obfsproxy,
		FirewallOff:     c.firewallOff,
		DNS:             c.dns,
		Antitracker:     c.antitracker,
		AntitrackerHard: c.antitrackerHard,
		IPv6Tunnel:      c.isIPv6Tunnel,
		MultiopExitSvr:  c.multihopExitSvr})

	return nil
}

func getPort(vpnType vpn.Type, portInfo string) (port, error) {
	var err error
	var portPtr *int
	var isTCPPtr *bool
	if len(portInfo) > 0 {
		portPtr, isTCPPtr, err = parsePort(portInfo)
		if err != nil {
			return port{}, err
		}
	}

	var retPort port
	if vpnType == vpn.WireGuard {
		if isTCPPtr != nil && *isTCPPtr {
			return port{}, flags.BadParameter{Message: "port"}
		}
		retPort = portsWireGuard[0] // default port
	} else {
		retPort = portsOpenVpn[0] // default port
	}

	if portPtr != nil {
		retPort.port = *portPtr
	}
	if isTCPPtr != nil {
		retPort.tcp = *isTCPPtr
	}

	// ckeck is port allowed
	if vpnType == vpn.WireGuard {
		if isPortAllowed(portsWireGuard[:], retPort) == false {
			fmt.Printf("WARNING: using non-standard port '%s' (allowed ports: %s)\n", retPort.String(), allPortsString(portsWireGuard[:]))
		}
	} else {
		if isPortAllowed(portsOpenVpn[:], retPort) == false {
			fmt.Printf("WARNING: using non-standard port '%s' (allowed ports: %s)\n", retPort.String(), allPortsString(portsOpenVpn[:]))
		}
	}

	return retPort, nil
}

func isPortAllowed(ports []port, thePort port) bool {
	for _, p := range ports {
		if p.port == thePort.port && p.tcp == thePort.tcp {
			return true
		}
	}
	return false
}

func allPortsString(ports []port) string {
	s := make([]string, 0, len(ports))
	for _, p := range ports {
		s = append(s, p.String())
	}
	return strings.Join(s, ", ")
}

// parsing port info from string in format "PROTOCOL:PORT"
func parsePort(portInfo string) (pPort *int, pIsTCP *bool, err error) {

	var port int
	var isTCP bool

	if len(portInfo) == 0 {
		return nil, nil, nil
	}

	portInfo = strings.ToLower(portInfo)

	fields := strings.Split(portInfo, ":")
	if len(fields) > 2 {
		return nil, nil, flags.BadParameter{Message: "port"}
	}

	protoStr := ""
	portStr := ""
	if len(fields) == 2 {
		protoStr = fields[0]
		portStr = fields[1]
	} else {
		if _, err := strconv.Atoi(fields[0]); err != nil {
			protoStr = fields[0]
		} else {
			portStr = fields[0]
		}
	}

	if len(protoStr) > 0 {
		if protoStr == "tcp" {
			isTCP = true
		} else if protoStr == "udp" {
			isTCP = false
		} else {
			return nil, nil, flags.BadParameter{Message: "port"}
		}
	}

	if len(portStr) > 0 {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, nil, flags.BadParameter{Message: "port"}
		}
		port = p
	}

	return &port, &isTCP, nil
}
