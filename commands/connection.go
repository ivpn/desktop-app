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
	if err := _proto.Disconnect(); err != nil {
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

	filter_proto       bool
	filter_location    bool
	filter_city        bool
	filter_country     bool
	filter_countryCode bool
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

	c.BoolVar(&c.filter_proto, "fp", false, "Apply LOCATION as a filter to protocol type (can be used short names 'wg' or 'ovpn')")
	c.BoolVar(&c.filter_location, "fl", false, "Apply LOCATION as a filter to server location (serverID)")
	c.BoolVar(&c.filter_country, "fc", false, "Apply LOCATION as a filter to country name")
	c.BoolVar(&c.filter_countryCode, "fcc", false, "Apply LOCATION as a filter to country code")
	c.BoolVar(&c.filter_city, "fcity", false, "Apply LOCATION as a filter to city name")
}

func (c *CmdConnect) Run() error {
	if len(c.gateway) == 0 {
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

	svrs := serversFilter(serversList(servers), c.gateway, c.filter_proto, c.filter_location, c.filter_city, c.filter_countryCode, c.filter_country)
	if len(svrs) > 1 {
		if c.any == false {
			fmt.Printf("More then one server found (filtering by '%s')\n", c.gateway)
			fmt.Println("Please specify server more correctly or use flag '-any'")
			fmt.Println("\nTips:")
			fmt.Printf("\t%s servers        Show servers list\n", os.Args[0])
			fmt.Printf("\t%s connect -h     Show usage of 'connect' command\n", os.Args[0])
			return nil
		}
		fmt.Printf("More then one server found (filtering by '%s')\n", c.gateway)
		fmt.Printf("Taking first found server...\n")
	}
	c.gateway = svrs[0].gateway

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
		if err2 := _proto.Disconnect(); err2 != nil {
			fmt.Printf("Failed to disconnect: %v\n", err2)
		}
		return err
	}

	printState(vpn.CONNECTED, connected)

	return nil
}
