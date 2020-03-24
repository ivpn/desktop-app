package commands

import (
	"fmt"
	"net"

	"github.com/ivpn/desktop-app-cli/flags"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
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
	obfsproxy       bool
	firewall        bool
	dns             string
	antitracker     bool
	antitrackerHard bool
}

func (c *CmdConnect) Init() {
	c.Initialize("connect", "Establish new VPN connection. Use serverID (Location) as an argument (see 'servers' command)")
	c.DefaultStringVar(&c.gateway, "SERVER_ID")
	c.BoolVar(&c.obfsproxy, "o", false, "OpenVPN only: Use obfsproxy (only enable if you have trouble connecting)")
	c.BoolVar(&c.obfsproxy, "obfsproxy", false, "OpenVPN only: Use obfsproxy (only enable if you have trouble connecting)")

	c.BoolVar(&c.firewall, "f", false, "Enable firewall (will be disabled after disconnection)")
	c.BoolVar(&c.firewall, "firewall", false, "Enable firewall (will be disabled after disconnection)")

	c.StringVar(&c.dns, "dns", "", "DNS_IP", "Use custom DNS for this connection\n(if 'antitracker' is enabled - this parameter will be ignored)")

	c.BoolVar(&c.antitracker, "antitracker", false, "Enable antitracker for this connection")
	c.BoolVar(&c.antitrackerHard, "antitracker_hard", false, "Enable 'hardcore' antitracker for this connection")
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
