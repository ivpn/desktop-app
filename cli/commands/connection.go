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
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"

	"github.com/ivpn/desktop-app/cli/flags"
	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/obfsproxy"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/srverrors"
	service_types "github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/v2r"
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
	if p.port != 0 {
		return fmt.Sprintf("%s:%d", protoName, p.port)
	}
	return fmt.Sprintf("%s", protoName)
}

func defaultPort() *port {
	return &port{port: 2049}
}

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

// -----------------------------------------------
const AllowedObfsproxyValues = "'obfs4' (default), 'obfs3', 'obfs4_iat' (or 'obfs4_iat1'), 'obfs4_iat_paranoid' (or 'obfs4_iat2')"

func parseObfsproxyParam(param string) (obfsproxy.Config, error) {
	switch strings.ToLower(param) {
	case "":
		return obfsproxy.Config{}, nil
	case "obfs3":
		return obfsproxy.Config{Version: obfsproxy.OBFS3}, nil
	case "obfs4":
		return obfsproxy.Config{Version: obfsproxy.OBFS4}, nil
	case "obfs4_iat1", "obfs4_iat":
		return obfsproxy.Config{Version: obfsproxy.OBFS4, Obfs4Iat: obfsproxy.Obfs4IatOn}, nil
	case "obfs4_iat2", "obfs4_iat_paranoid":
		return obfsproxy.Config{Version: obfsproxy.OBFS4, Obfs4Iat: obfsproxy.Obfs4IatOnParanoid}, nil
	}

	return obfsproxy.Config{}, fmt.Errorf("unsupported obfsproxy value '%s' (acceptable values: %s)", param, AllowedObfsproxyValues)
}

func parseV2RayParam(param string) (v2r.V2RayTransportType, error) {
	switch strings.ToLower(param) {
	case "":
		return v2r.None, nil
	case "quic":
		return v2r.QUIC, nil
	case "tcp":
		return v2r.TCP, nil
	}

	return v2r.None, fmt.Errorf("unsupported v2ray value '%s' (acceptable values: 'quic' or 'tcp')", param)
}

type CmdConnect struct {
	flags.CmdInfo
	last            bool
	gateway         string
	port            string
	portsShow       bool
	any             bool
	obfsproxy       string // 'obfs4' (default), 'obfs3', 'obfs4_iat' (or 'obfs4_iat1'), 'obfs4_iat_paranoid' (or 'obfs4_iat2')
	v2rayProxy      string // `quic` or `tcp`
	firewallOff     bool
	dns             string
	antitracker     bool
	antitrackerHard bool
	isIPv6Tunnel    bool

	mtu int // MTU value (applicable only for WireGuard)

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
	c.KeepArgsOrderInHelp = true
	c.SetPreParseFunc(c.preParse)

	c.Initialize("connect", "Establish new VPN connection\nLOCATION can be a mask for filtering servers or full hostname (see 'servers' command)")
	c.DefaultStringVar(&c.gateway, "LOCATION")

	// Filtering flags
	c.BoolVar(&c.filter_city, "city", false, "Apply LOCATION as a filter to city name")
	c.BoolVar(&c.filter_countryCode, "cc", false, "Apply LOCATION as a filter to country code")
	c.BoolVar(&c.filter_countryCode, "country_code", false, "Apply LOCATION as a filter to country code")
	c.BoolVar(&c.filter_country, "c", false, "Apply LOCATION as a filter to country name")
	c.BoolVar(&c.filter_country, "country", false, "Apply LOCATION as a filter to country name")
	c.BoolVar(&c.filter_location, "l", false, "Apply LOCATION as a filter to server location (Hostname)")
	c.BoolVar(&c.filter_location, "location", false, "Apply LOCATION as a filter to server location (Hostname)")
	c.BoolVar(&c.filter_invert, "filter_invert", false, "Invert filtering")

	// Automatic server selection flags
	c.BoolVar(&c.fastest, "fastest", false, "Connect to fastest server")
	c.BoolVar(&c.last, "last", false, "Connect with the last used connection parameters")
	c.BoolVar(&c.any, "any", false, "Use a random server from the found results to connect")

	// Multi-Hop
	c.StringVar(&c.multihopExitSvr, "exit_svr", "", "LOCATION", "Exit-server for Multi-Hop connection\n  (use full serverID as a parameter, servers filtering not applicable for it)")

	// Protocol flags
	c.StringVar(&c.filter_proto, "protocol", "", "PROTOCOL", "Protocol type (OpenVPN|ovpn|WireGuard|wg)")
	c.StringVar(&c.filter_proto, "p", "", "PROTOCOL", "Protocol type (OpenVPN|ovpn|WireGuard|wg)")
	c.IntVar(&c.mtu, "mtu", 0, "MTU", "Maximum transmission unit (applicable only for WireGuard connections)")
	c.BoolVar(&c.isIPv6Tunnel, "ipv6tunnel", false, "Enable IPv6 in VPN tunnel (WireGuard connections only)\n  (IPv6 addresses are preferred when a host has a dual stack IPv6/IPv4; IPv4-only hosts are unaffected)")

	// Port flags
	c.BoolVar(&c.portsShow, "show_ports", false, "Ports which are applicable for '-port' argument. Show all supported connection ports")
	c.StringVar(&c.port, "port", "", "PROTOCOL:PORT", fmt.Sprintf("Port to connect to (default: '%s')\n  Note: port number ignored for Multi-Hop connections; port type only applicable (UDP/TCP)\n  Tip: use `ivpn connect -show_ports` command to show all supported ports", defaultPort()))

	// Firewall flags
	c.BoolVar(&c.firewallOff, "fw_off", false, "Do not enable firewall for this connection\n  (has effect only if Firewall not enabled before)")

	// DNS flags
	c.StringVar(&c.dns, "dns", "", "DNS_IP", "Use custom DNS for this connection\n  (if 'antitracker' is enabled - this parameter will be ignored)")
	c.BoolVar(&c.antitracker, "antitracker", false, "Enable AntiTracker for this connection")
	c.BoolVar(&c.antitrackerHard, "antitracker_hard", false, "Enable 'Hard Core' AntiTracker for this connection")

	// Obfuscation flags
	obfsproxyUsage := fmt.Sprintf("Use obfsproxy (OpenVPN only)\n  Acceptable values: %s", AllowedObfsproxyValues)
	c.StringVar(&c.obfsproxy, "o", "", "TYPE", obfsproxyUsage)
	c.StringVar(&c.obfsproxy, "obfsproxy", "", "TYPE", obfsproxyUsage)
	c.StringVar(&c.v2rayProxy, "v2ray", "", "TYPE", "Use V2Ray obfuscation (this option takes precedence over the '-obfsproxy' option)\n  Acceptable values: 'quic' or 'tcp'")
}

func (c *CmdConnect) preParse(arguments []string) ([]string, error) {
	// Pre-parse function (called before parsing arguments by 'flag' package)
	//
	// We need to have compatibility with old app versions:
	// 	parameters '-o' and '-obfsproxy' can be used without argument (which is not possible by 'flag' package)
	// 	When '-o'|'-obfsproxy' are without argument - use default obfsproxy configuration

	obfsEmptyParamIdx := -1
	for idx, a := range arguments {
		arg := strings.ToLower(a)
		if arg == "-o" || arg == "-obfsproxy" {
			// Check if '-o' parameter is alone (old app implementation style):
			// - argument for '-o' should not be last in arguments list (because last argument for 'connect' is always filter value)
			// - ... and next argument after '-o' must start with '-' (must be new arg definition)
			nextIdx := idx + 1
			lastIdx := len(arguments) - 1

			if nextIdx == lastIdx { // last argument for 'connect' is always filter value (not a parameter of '-o' )
				obfsEmptyParamIdx = idx // empty '-o' argument is detected
				break
			}
			if nextIdx < lastIdx {
				nextArg := arguments[nextIdx]
				if !strings.HasPrefix(nextArg, "-") { // next argument after '-o' must start with '-'
					break
				}
				obfsEmptyParamIdx = idx // empty '-o' argument is detected
			}
			break
		}
	}

	if obfsEmptyParamIdx >= 0 {
		c.obfsproxy = "obfs4"
		return append(arguments[:obfsEmptyParamIdx], arguments[obfsEmptyParamIdx+1:]...), nil
	}

	return arguments, nil
}

// Run executes command
func (c *CmdConnect) Run() (retError error) {

	if len(c.gateway) == 0 && c.fastest == false && c.any == false && c.last == false && c.portsShow == false {
		return flags.BadParameter{}
	}
	if c.v2rayProxy != "" && c.obfsproxy != "" {
		return flags.BadParameter{Message: "cannot use both '-v2ray' and '-obfsproxy' options"}
	}

	// connection request
	req := types.Connect{}

	// get servers list from daemon
	serverFound := false
	servers, err := _proto.GetServers()
	if err != nil {
		return err
	}

	customHostEntryServer := c.gateway
	customHostExitServer := c.multihopExitSvr

	obfsproxyCfg, err := parseObfsproxyParam(c.obfsproxy)
	if err != nil {
		return flags.BadParameter{Message: err.Error()}
	}

	v2rayCfg, err := parseV2RayParam(c.v2rayProxy)
	if err != nil {
		return flags.BadParameter{Message: err.Error()}
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

	allowedPortsWg := servers.Config.Ports.WireGuard
	allowedPortsOvpn := servers.Config.Ports.OpenVPN
	if c.portsShow {
		printAllowedPorts(allowedPortsWg, allowedPortsOvpn)
		return nil
	}

	if len(allowedPortsWg) <= 0 || len(allowedPortsOvpn) <= 0 {
		fmt.Println("Internal ERROR: daemon does not provide allowed ports info !")
		printAllowedPorts(allowedPortsWg, allowedPortsOvpn)
	}

	// show current state after on finished
	defer func() {
		if retError == nil {
			showState()
		}
	}()

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

	defaultConnSettings, err := _proto.GetDefConnectionParams()
	if err != nil {
		return err
	}

	// do we need to connect with default connection parameters
	if c.last {
		fmt.Println("Enabled '-last' parameter. Using parameters from last used configuration")
		req.Params = defaultConnSettings.Params
		req.Params.FirewallOnDuringConnection = true
	} else {
		// MULTI\SINGLE -HOP
		// Check if the parameters are correct and define correct values for c.gateway and c.multihopExitSvr
		if len(c.multihopExitSvr) > 0 {
			// MULTI-HOP

			if err := helloResp.Account.IsCanConnectMultiHop(); err != nil {
				return err
			}

			if c.fastest {
				return flags.BadParameter{Message: "'fastest' flag is not applicable for Multi-Hop connection [exit_svr]"}
			}

			if c.filter_location || c.filter_city || c.filter_countryCode || c.filter_country || c.filter_invert {
				fmt.Println("WARNING: filtering flags are ignored for Multi-Hop connection [exit_svr]")
			}

			entrySvrs := serversFilter(isWgDisabled, isOpenVPNDisabled, svrs, c.gateway, c.filter_proto, false, false, false, false, false)
			if len(entrySvrs) == 0 || len(entrySvrs) > 1 {
				return flags.BadParameter{Message: "specify correct entry server ID for multi-hop connection"}
			}

			exitSvrs := serversFilter(isWgDisabled, isOpenVPNDisabled, svrs, c.multihopExitSvr, c.filter_proto, false, false, false, false, false)
			if len(exitSvrs) == 0 || len(exitSvrs) > 1 {
				return flags.BadParameter{Message: "specify correct exit server ID for multi-hop connection"}
			}
			entrySvr := entrySvrs[0]
			exitSvr := exitSvrs[0]

			if entrySvr.gateway == exitSvr.gateway {
				return flags.BadParameter{Message: "unable to use same entry- and exit- servers"}
			}

			if entrySvr.countryCode == exitSvr.countryCode {
				fmt.Println("Warning! Entry- and exit- servers located in the same country.")
			}

			c.gateway = entrySvr.gateway
			c.multihopExitSvr = exitSvr.gateway
		} else { //SINGLE-HOP
			svrs = serversFilter(isWgDisabled, isOpenVPNDisabled, svrs, c.gateway, c.filter_proto, c.filter_location, c.filter_city, c.filter_countryCode, c.filter_country, c.filter_invert)

			srvID := ""

			// Fastest server
			if c.fastest && len(svrs) > 1 {
				var vpnType *vpn.Type = nil
				if len(c.filter_proto) > 0 {
					if p, err := getVpnTypeByFlag(c.filter_proto); err == nil {
						vpnType = &p
					}
				}
				if err := serversPing(svrs, true, vpnType); err != nil {
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
					PrintTips([]TipType{TipServers, TipConnectHelp})
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
					fmt.Print("More than one server was found. ")

					if c.any == false {
						fmt.Println("Please specify server more correctly or use flag '-any'")
						showTipsServerFilterError()
						return fmt.Errorf("more than one server found")
					}
					fmt.Printf("Taking one random from found servers ...\n")
				}

				if rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(svrs)))); err == nil {
					srvID = svrs[rnd.Int64()].gateway
				} else {
					srvID = svrs[0].gateway
				}
			}
			c.gateway = srvID
		}

		// Firewall for current connection
		req.Params.FirewallOnDuringConnection = true
		if c.firewallOff {
			// check current FW state
			state, err := _proto.FirewallStatus()
			if err != nil {
				return fmt.Errorf("unable to check Firewall state: %w", err)
			}
			if state.IsEnabled == false {
				req.Params.FirewallOnDuringConnection = false
			} else {
				fmt.Println("WARNING! Firewall option ignored (Firewall already enabled manually)")
			}
		}

		// Looking for connection server

		// WireGuard
		{
			funcApplyCustomHost := func(hosts []apitypes.WireGuardServerHostInfo, hostname string) []apitypes.WireGuardServerHostInfo {
				for _, h := range hosts {
					if h.Hostname == hostname {
						return []apitypes.WireGuardServerHostInfo{h}
					}
				}
				return hosts
			}

			var entrySvrWg *apitypes.WireGuardServerInfo = nil
			var exitSvrWg *apitypes.WireGuardServerInfo = nil
			// exit server
			if len(c.multihopExitSvr) > 0 {
				for i, s := range servers.WireguardServers {
					if s.Gateway == c.multihopExitSvr {
						exitSvrWg = &servers.WireguardServers[i]
						break
					}
				}
			}
			// entry server
			for i, s := range servers.WireguardServers {
				if s.Gateway == c.gateway {
					entrySvrWg = &servers.WireguardServers[i]

					serverFound = true
					req.Params.VpnType = vpn.WireGuard
					req.Params.WireGuardParameters.EntryVpnServer.Hosts = funcApplyCustomHost(s.Hosts, customHostEntryServer)
					req.Params.IPv6 = c.isIPv6Tunnel

					if c.mtu > 0 {
						fmt.Printf("[!] Using custom MTU: %d\n", c.mtu)
						req.Params.WireGuardParameters.Mtu = c.mtu
					}

					if len(c.multihopExitSvr) == 0 {
						// port
						p, err := getPort(c.port, allowedPortsWg)
						if err != nil {
							printAllowedPorts(allowedPortsWg, allowedPortsOvpn)
							return err
						}
						req.Params.WireGuardParameters.Port.Port = p.port

						fmt.Printf("[WireGuard] Connecting to: %s, %s (%s) %s %s...\n", s.City, s.CountryCode, s.Country, s.Gateway, p.String())
					} else {
						if exitSvrWg == nil {
							return fmt.Errorf("serverID not found in servers list (%s)", c.multihopExitSvr)
						}

						// port definition is not required for WireGuard multi-hop (in use: UDP + port-based-multihop)
						if len(c.port) > 0 {
							// if user manually defined port for obfsproxy connection - inform that it is ignored
							fmt.Printf("Note: port definition is ignored for WireGuard Multi-Hop connections\n")
						}

						req.Params.WireGuardParameters.MultihopExitServer.ExitSrvID = strings.Split(exitSvrWg.Gateway, ".")[0]
						req.Params.WireGuardParameters.MultihopExitServer.Hosts = funcApplyCustomHost(exitSvrWg.Hosts, customHostExitServer)

						fmt.Printf("[WireGuard] Connecting Multi-Hop...\n")
						fmt.Printf("\tentry server: %s, %s (%s) %s\n", entrySvrWg.City, entrySvrWg.CountryCode, entrySvrWg.Country, entrySvrWg.Gateway)
						fmt.Printf("\texit server : %s, %s (%s) %s\n", exitSvrWg.City, exitSvrWg.CountryCode, exitSvrWg.Country, exitSvrWg.Gateway)
					}
					break
				}
			}
		}

		// OpenVPN
		if serverFound == false {
			if obfsproxyCfg.IsObfsproxy() && len(helloResp.DisabledFunctions.ObfsproxyError) > 0 {
				return fmt.Errorf(helloResp.DisabledFunctions.ObfsproxyError)
			}

			funcApplyCustomHost := func(hosts []apitypes.OpenVPNServerHostInfo, hostname string) []apitypes.OpenVPNServerHostInfo {
				for _, h := range hosts {
					if h.Hostname == hostname {
						return []apitypes.OpenVPNServerHostInfo{h}
					}
				}
				return hosts
			}

			var entrySvrOvpn *apitypes.OpenvpnServerInfo = nil
			var exitSvrOvpn *apitypes.OpenvpnServerInfo = nil

			// exit server
			if len(c.multihopExitSvr) > 0 {
				for i, s := range servers.OpenvpnServers {
					if s.Gateway == c.multihopExitSvr {
						exitSvrOvpn = &servers.OpenvpnServers[i]
						break
					}
				}
			}

			var destPort port
			// entry server
			for i, s := range servers.OpenvpnServers {
				if s.Gateway == c.gateway {
					entrySvrOvpn = &servers.OpenvpnServers[i]

					// set obfsproxy config
					if obfsproxyCfg.IsObfsproxy() {
						fmt.Println("obfsproxy configuration: " + obfsproxyCfg.ToString())
						req.Params.OpenVpnParameters.Obfs4proxy = obfsproxyCfg
					}

					serverFound = true
					req.Params.VpnType = vpn.OpenVPN
					req.Params.OpenVpnParameters.EntryVpnServer.Hosts = funcApplyCustomHost(s.Hosts, customHostEntryServer)

					isMultihop := exitSvrOvpn != nil && len(c.multihopExitSvr) > 0
					if !isMultihop {
						// port
						destPort, err = getPort(c.port, allowedPortsOvpn)
						if err != nil {
							printAllowedPorts(allowedPortsWg, allowedPortsOvpn)
							return err
						}
					} else {
						// port
						destPort, err = getPort(c.port, nil)
						if err != nil {
							printAllowedPorts(allowedPortsWg, allowedPortsOvpn)
							return err
						}

						// get Multi-Hop ID
						req.Params.OpenVpnParameters.MultihopExitServer.ExitSrvID = strings.Split(c.multihopExitSvr, ".")[0]
						req.Params.OpenVpnParameters.MultihopExitServer.Hosts = funcApplyCustomHost(exitSvrOvpn.Hosts, customHostExitServer)
						destPort.port = 0 // do not use port number (port-based multihop)
					}

					req.Params.OpenVpnParameters.Port.Port = destPort.port
					req.Params.OpenVpnParameters.Port.Protocol = destPort.IsTCP()

					break
				}
			}

			if entrySvrOvpn == nil {
				return fmt.Errorf("serverID not found in servers list (%s)", c.gateway)
			}
			if len(c.multihopExitSvr) > 0 && exitSvrOvpn == nil {
				return fmt.Errorf("serverID not found in servers list (%s)", c.multihopExitSvr)
			}

			portStrInfo := destPort.String()
			if obfsproxyCfg.IsObfsproxy() {
				if len(c.port) > 0 {
					// if user manually defined port for obfsproxy connection - inform that it is ignored
					fmt.Printf("Note: port definition is ignored for the connections when the obfsproxy enabled\n")
				}
				portStrInfo = "TCP"
				destPort.tcp = true
			}

			if len(c.multihopExitSvr) == 0 {
				fmt.Printf("[OpenVPN] Connecting to: %s, %s (%s) %s %s...\n", entrySvrOvpn.City, entrySvrOvpn.CountryCode, entrySvrOvpn.Country, entrySvrOvpn.Gateway, portStrInfo)
			} else {
				portStrInfo = "UDP"
				if destPort.tcp {
					portStrInfo = "TCP"
				}

				fmt.Printf("[OpenVPN] Connecting Multi-Hop...\n")
				fmt.Printf("\tentry server: %s, %s (%s) %s %s\n", entrySvrOvpn.City, entrySvrOvpn.CountryCode, entrySvrOvpn.Country, entrySvrOvpn.Gateway, portStrInfo)
				fmt.Printf("\texit server : %s, %s (%s) %s\n", exitSvrOvpn.City, exitSvrOvpn.CountryCode, exitSvrOvpn.Country, exitSvrOvpn.Gateway)
			}
		}

		if serverFound == false {
			return fmt.Errorf("serverID not found in servers list (%s)", c.gateway)
		}

		// Set V2Ray obfuscation parameters
		if v2rayCfg != v2r.None && len(helloResp.DisabledFunctions.V2RayError) > 0 {
			return fmt.Errorf(helloResp.DisabledFunctions.V2RayError)
		}
		if _proto.GetHelloResponse().DaemonSettings.V2RayConfig != v2rayCfg {
			if err = _proto.SetV2RayProxy(v2rayCfg); err != nil {
				return err
			}
		}

		// SET ANTI-TRACKER parameters. It will overwrite 'custom DNS' parameter
		if c.antitracker || c.antitrackerHard {
			req.Params.Metadata.AntiTracker.Enabled = true
			req.Params.Metadata.AntiTracker.Hardcore = c.antitrackerHard
		} else {
			// AntiTracker parameters not defined for current connection
			// Taking default configuration parameters (if defined)
			req.Params.Metadata.AntiTracker = defaultConnSettings.Params.Metadata.AntiTracker
			if req.Params.Metadata.AntiTracker.IsEnabled() { // print default AntiTracker values
				printAntitrackerConfigInfo(nil, req.Params.Metadata.AntiTracker.Enabled, req.Params.Metadata.AntiTracker.Hardcore).Flush()
			}
		}

		// Set MANUAL DNS
		if len(c.dns) > 0 {
			if req.Params.Metadata.AntiTracker.Enabled {
				fmt.Println("WARNING! Manual DNS configuration ignored due to AntiTracker")
			}
			dnsIp := net.ParseIP(c.dns)
			if dnsIp == nil {
				return flags.BadParameter{}
			}
			req.Params.ManualDNS = dns.DnsSettings{DnsHost: dnsIp.String(), Encryption: dns.EncryptionNone}
		} else {
			// Taking default configuration parameters
			req.Params.ManualDNS = defaultConnSettings.Params.ManualDNS
			if !req.Params.ManualDNS.IsEmpty() {
				printDNSConfigInfo(nil, req.Params.ManualDNS).Flush()
			}
		}

		// metadata
		if c.fastest {
			req.Params.Metadata.ServerSelectionEntry = service_types.Fastest
		} else if c.any {
			req.Params.Metadata.ServerSelectionEntry = service_types.Random
		}
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

	return nil
}

func getPort(portInfo string, allowedPorts []apitypes.PortInfo) (port, error) {
	var err error
	var portPtr *int
	var isTCPPtr *bool
	if len(portInfo) > 0 {
		portPtr, isTCPPtr, err = parsePort(portInfo)
		if err != nil {
			return port{}, err
		}
	}

	retPort := *defaultPort() // default port

	if portPtr != nil {
		retPort.port = *portPtr
	}
	if isTCPPtr != nil {
		retPort.tcp = *isTCPPtr
	}

	if len(allowedPorts) > 0 {
		if isPortAllowed(allowedPorts[:], retPort) == false {
			return port{}, fmt.Errorf(fmt.Sprintf("not allowed port '%s'", retPort.String()))
		}
	}

	return retPort, nil
}

func printAllowedPorts(allowedPortsWg, allowedOvpnPorts []apitypes.PortInfo) {
	fmt.Printf("Allowed ports:\n")
	if allowedPortsWg != nil {
		fmt.Printf("  WireGuard: %s\n", allPortsString(allowedPortsWg[:]))
	}
	if allowedOvpnPorts != nil {
		fmt.Printf("  OpenVPN: %s\n", allPortsString(allowedOvpnPorts[:]))
	}
}

func isPortAllowed(ports []apitypes.PortInfo, thePort port) bool {
	for _, p := range ports {
		if p.Port != 0 && p.Port == thePort.port && p.IsTCP() == thePort.tcp {
			return true
		}
		if p.Range.Min > 0 && thePort.port >= p.Range.Min && thePort.port <= p.Range.Max {
			return true
		}
	}
	return false
}

func allPortsString(ports []apitypes.PortInfo) string {
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

	pInfoOrig := portInfo
	portInfo = strings.ToLower(portInfo)

	fields := strings.Split(portInfo, ":")
	if len(fields) > 2 {
		return nil, nil, fmt.Errorf("failed to parse the port value '%s' (bad format)", pInfoOrig)
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
			return nil, nil, fmt.Errorf("failed to parse the port value '%s' (bad format)", pInfoOrig)
		}
	}

	if len(portStr) > 0 {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse the port value '%s' (bad format)", pInfoOrig)
		}
		port = p
	}

	return &port, &isTCP, nil
}
