package commands

import (
	"fmt"
	"net"
	"os"

	"github.com/ivpn/desktop-app-cli/flags"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/service"
	"github.com/ivpn/desktop-app-daemon/vpn"
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
	return nil
}

//-----------------------------------------------

type CmdConnect struct {
	flags.CmdInfo
	gateway         string
	any             bool
	obfsproxy       bool
	firewall        bool
	dns             string
	antitracker     bool
	antitrackerHard bool

	filter_proto       string
	filter_location    bool
	filter_city        bool
	filter_country     bool
	filter_countryCode bool
	filter_invert      bool

	fastest bool
}

func (c *CmdConnect) Init() {
	c.Initialize("connect", "Establish new VPN connection. Use server location as an argument.\nLOCATION can be a mask for filtering servers (see 'servers' command)")
	c.DefaultStringVar(&c.gateway, "LOCATION")
	c.BoolVar(&c.any, "any", false, "When LOCATION points to more then one servers - use first found server to connect")

	c.BoolVar(&c.obfsproxy, "o", false, "OpenVPN only: Use obfsproxy (only enable if you have trouble connecting)")
	c.BoolVar(&c.obfsproxy, "obfsproxy", false, "OpenVPN only: Use obfsproxy (only enable if you have trouble connecting)")

	c.BoolVar(&c.firewall, "f", false, "Enable firewall (will be disabled after disconnection)")
	c.BoolVar(&c.firewall, "firewall", false, "Enable firewall (will be disabled after disconnection)")

	c.StringVar(&c.dns, "dns", "", "DNS_IP", "Use custom DNS for this connection\n(if 'antitracker' is enabled - this parameter will be ignored)")

	c.BoolVar(&c.antitracker, "antitracker", false, "Enable antitracker for this connection")
	c.BoolVar(&c.antitrackerHard, "antitracker_hard", false, "Enable 'hardcore' antitracker for this connection")

	c.StringVar(&c.filter_proto, "fp", "", "PROTOCOL", "Protocol type [WireGuard/OpenVPN] (can be used short names 'wg' or 'ovpn')")
	c.BoolVar(&c.filter_location, "fl", false, "Apply LOCATION as a filter to server location (serverID)")
	c.BoolVar(&c.filter_country, "fc", false, "Apply LOCATION as a filter to country name")
	c.BoolVar(&c.filter_countryCode, "fcc", false, "Apply LOCATION as a filter to country code")
	c.BoolVar(&c.filter_city, "fcity", false, "Apply LOCATION as a filter to city name")

	c.BoolVar(&c.filter_invert, "filter_invert", false, "Invert filtering")

	c.BoolVar(&c.fastest, "fastest", false, "Connect to fastest server")
}

func (c *CmdConnect) Run() (retError error) {
	if len(c.gateway) == 0 && c.fastest == false {
		return flags.BadParameter{}
	}
	// connection request
	req := types.Connect{}

	// get servers list from daemon
	serverFound := false
	servers, err := _proto.GetServers()
	if err != nil {
		return err
	}

	helloResp := _proto.GetHelloResponse()
	if len(helloResp.Command) > 0 && (len(helloResp.Session.Session) == 0) {
		// We received 'hello' response but no session info - print tips to login
		fmt.Println("Error: Not logged in")
		fmt.Println("")
		fmt.Println("Tips: ")
		fmt.Printf("  %s account -login  ACCOUNT_ID         Log in with your Account ID\n", os.Args[0])
		fmt.Println("")
		return service.ErrorNotLoggedIn{}
	}

	svrs := serversList(servers)
	svrs = serversFilter(svrs, c.gateway, c.filter_proto, c.filter_location, c.filter_city, c.filter_countryCode, c.filter_country, c.filter_invert)

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
		srvID = svrs[len(svrs)-1].gateway
	}

	// if we not foud required server before (by 'fastest' option)
	if len(srvID) == 0 {
		defer func() {
			if retError != nil {
				fmt.Println("Please specify server more correctly or use flag '-any'")
				fmt.Println("\nTips:")
				fmt.Printf("\t%s servers        Show servers list\n", os.Args[0])
				fmt.Printf("\t%s connect -h     Show usage of 'connect' command\n", os.Args[0])
			}
		}()

		// no servers found
		if len(svrs) == 0 {
			fmt.Println("No servers found by your filter")
			return fmt.Errorf("no servers found by your filter")
		}

		// 'any' option
		if len(svrs) > 1 {
			fmt.Println("More than one server found")
			if c.any == false {
				return fmt.Errorf("more than one server found")
			}
			fmt.Printf("Taking first found server\n")
		}
		srvID = svrs[0].gateway
	}
	c.gateway = srvID

	// FW for current connection
	req.FirewallOnDuringConnection = c.firewall

	// set Manual DNS if defined
	if len(c.dns) > 0 {
		dns := net.ParseIP(c.dns)
		if dns == nil {
			return flags.BadParameter{}
		}
		req.CurrentDNS = dns.String()
	}
	// set antitracker DNS (if defined). It will overwrite 'custom DNS' parameter
	if c.antitracker || c.antitrackerHard {
		if c.antitracker {
			req.CurrentDNS = servers.Config.Antitracker.Default.IP
		}
		if c.antitrackerHard {
			req.CurrentDNS = servers.Config.Antitracker.Hardcore.IP
		}
	}

	// looking for connection server
	// WireGuard
	for _, s := range servers.WireguardServers {
		if s.Gateway == c.gateway {
			fmt.Printf("[WireGuard] Connecting to: %s, %s (%s) %s...\n", s.City, s.CountryCode, s.Country, s.Gateway)

			serverFound = true
			host := s.Hosts[0]
			req.VpnType = vpn.WireGuard
			req.WireGuardParameters.Port.Port = 2049
			req.WireGuardParameters.EntryVpnServer.Hosts = []types.WGHost{types.WGHost{Host: host.Host, PublicKey: host.PublicKey, LocalIP: host.LocalIP}}
			break
		}
	}
	// OpenVPN
	for _, s := range servers.OpenvpnServers {
		if s.Gateway == c.gateway {
			fmt.Printf("[OpenVPN] Connecting to: %s, %s (%s) %s...\n", s.City, s.CountryCode, s.Country, s.Gateway)

			// TODO: obfsproxy configuration for this connection must be sent in 'Connect' request (avoid using daemon preferences)
			if err = _proto.SetPreferences("enable_obfsproxy", fmt.Sprint(c.obfsproxy)); err != nil {
				return err
			}

			serverFound = true
			req.VpnType = vpn.OpenVPN
			req.OpenVpnParameters.Port.Port = 2049
			req.OpenVpnParameters.Port.Protocol = 0 // IS TCP
			req.OpenVpnParameters.EntryVpnServer.IPAddresses = s.IPAddresses
			break
		}
	}

	if serverFound == false {
		return fmt.Errorf("serverID not found in servers list (%s)", c.gateway)
	}

	fmt.Println("Connecting...")
	connected, err := _proto.ConnectVPN(req)

	if err != nil {
		err = fmt.Errorf("failed to connect: %w", err)
		fmt.Printf("Disconnecting...\n")
		if err2 := _proto.DisconnectVPN(); err2 != nil {
			fmt.Printf("Failed to disconnect: %v\n", err2)
		}
		return err
	}

	printState(vpn.CONNECTED, connected)

	return nil
}
