package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ivpn/desktop-app-cli/flags"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/vpn"

	"golang.org/x/crypto/ssh/terminal"
)

// NotImplemented error
type NotImplemented struct {
	Message string
}

func (e NotImplemented) Error() string {
	if len(e.Message) == 0 {
		return "not implemented"
	}
	return e.Message
}

//-----------------------------------------------
func printAccountInfo(accountID string) {
	status := "Not logged in"
	if len(accountID) > 0 {
		return // Do nothing in case of logged in
	}
	fmt.Printf("Account                 : %v\n", status)
}

func printState(state vpn.State, connected types.ConnectedResp) {
	fmt.Printf("VPN                     : %v\n", state)

	if state != vpn.CONNECTED {
		return
	}
	since := time.Unix(connected.TimeSecFrom1970, 0)
	fmt.Printf("    Protocol            : %v\n", connected.VpnType)
	fmt.Printf("    Local IP            : %v\n", connected.ClientIP)
	fmt.Printf("    Server IP           : %v\n", connected.ServerIP)
	fmt.Printf("    Connected           : %v\n", since)
}

func printFirewallState(isEnabled, isPersistent, isAllowLAN, isAllowMulticast bool) {
	fmt.Println("Firewall:")
	fmt.Printf("    Enabled             : %v\n", isEnabled)
	fmt.Printf("    Persistent          : %v\n", isPersistent)
	fmt.Printf("    Allow LAN           : %v\n", isAllowLAN)
	fmt.Printf("    Allow LAN multicast : %v\n", isAllowMulticast)
}

//-----------------------------------------------
type cmdLogin struct {
	flags.CmdInfo
	loginAccountID string
	forceLogin     bool
}

func (c *cmdLogin) Init() {
	c.Initialize("login", "Login operation (register accountID)")
	c.DefaultStringVar(&c.loginAccountID, "ACCOUNT_ID")
	c.BoolVar(&c.forceLogin, "force", false, "Log out from all other devices")
}

func (c *cmdLogin) Run() error {
	if len(c.loginAccountID) == 0 {
		fmt.Print("Enter your Account ID: ")
		data, err := terminal.ReadPassword(0)
		if err != nil {
			return fmt.Errorf("failed to read accountID: %w", err)
		}
		fmt.Println("")
		c.loginAccountID = string(data)
	}
	return _proto.SessionNew(c.loginAccountID, c.forceLogin)
}

//-----------------------------------------------
type cmdLogout struct {
	flags.CmdInfo
}

func (c *cmdLogout) Init() {
	c.Initialize("logout", "Logout from this device (if logged-in)")
}
func (c *cmdLogout) Run() error {
	return _proto.SessionDelete()
}

//-----------------------------------------------
type cmdFirewall struct {
	flags.CmdInfo
	status bool
	on     bool
	off    bool
}

func (c *cmdFirewall) Init() {
	c.Initialize("firewall", "Firewall management")
	c.BoolVar(&c.status, "status", false, "(default) Show info about current firewall status")
	c.BoolVar(&c.off, "off", false, "Switch-off firewall")
	c.BoolVar(&c.on, "on", false, "Switch-on firewall")
}
func (c *cmdFirewall) Run() error {
	if c.on && c.off {
		return flags.BadParameter{}
	}

	if c.on {
		return _proto.FirewallSet(true)
	} else if c.off {
		return _proto.FirewallSet(false)
	}

	state, err := _proto.FirewallStatus()
	if err != nil {
		return err
	}

	printFirewallState(state.IsEnabled, state.IsPersistent, state.IsAllowLAN, state.IsAllowMulticast)
	return nil
}

//-----------------------------------------------
type cmdServers struct {
	flags.CmdInfo
	protocol string
	filter   string
}

func (c *cmdServers) Init() {
	c.Initialize("servers", "Show servers list")
	c.StringVar(&c.protocol, "p", "", "PROTO", "Show only servers for the given protocol. Possible values: 'wireguard' ('wg'), 'openvpn' ('ovpn')")
	c.StringVar(&c.protocol, "protocol", "", "PROTO", "Show only servers for the given protocol. Possible values: 'wireguard' ('wg'), 'openvpn' ('ovpn')")

	c.StringVar(&c.filter, "f", "", "MASK", "Filter servers: show only servers which contains <filter_string> in description (eg. -f 'US')")
	c.StringVar(&c.filter, "filter", "", "MASK", "Filter servers: show only servers which contains <filter_string> in description (eg. -f 'US')")
}
func (c *cmdServers) Run() error {
	servers, err := _proto.GetServers()
	if err != nil {
		return err
	}

	c.protocol = strings.TrimSpace(strings.ToLower(c.protocol))

	const padding = 1
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', tabwriter.AlignRight|tabwriter.Debug)

	fmt.Fprintln(w, "PROTOCOL\tLOCATION\tCITY\tCOUNTRY\t")

	if c.protocol != "openvpn" && c.protocol != "ovpn" {
		for _, s := range servers.WireguardServers {
			str := fmt.Sprintf("WireGuard\t%s\t%s (%s)\t %s\t", s.Gateway, s.City, s.CountryCode, s.Country)
			if len(c.filter) > 0 && strings.Contains(str, c.filter) == false {
				continue
			}
			fmt.Fprintln(w, str)
		}
	}

	if c.protocol != "wireguard" && c.protocol != "wg" {
		for _, s := range servers.OpenvpnServers {
			str := fmt.Sprintf("OpenVPN\t%s\t%s (%s)\t %s\t", s.Gateway, s.City, s.CountryCode, s.Country)
			if len(c.filter) > 0 && strings.Contains(str, c.filter) == false {
				continue
			}
			fmt.Fprintln(w, str)
		}
	}

	w.Flush()

	return nil
}

//-----------------------------------------------
type cmdState struct {
	flags.CmdInfo
}

func (c *cmdState) Init() {
	c.Initialize("state", "Prints full info about IVPN state")
}
func (c *cmdState) Run() error {
	fwstate, err := _proto.FirewallStatus()
	if err != nil {
		return err
	}

	state, connected, err := _proto.GetVPNState()
	if err != nil {
		return err
	}

	printAccountInfo(_proto.GetHelloResponse().Session.AccountID)
	printState(state, connected)
	printFirewallState(fwstate.IsEnabled, fwstate.IsPersistent, fwstate.IsAllowLAN, fwstate.IsAllowMulticast)

	fmt.Println("\nTips: ")
	if len(_proto.GetHelloResponse().Session.AccountID) == 0 {
		fmt.Println("  ivpn login        Log in with your Account ID")
	}
	fmt.Println("  ivpn -help        Show all commands")

	return nil
}

//-----------------------------------------------
type cmdDisconnect struct {
	flags.CmdInfo
}

func (c *cmdDisconnect) Init() {
	c.Initialize("disconnect", "Disconnect active VPN connection (if connected)")
}
func (c *cmdDisconnect) Run() error {
	if err := _proto.Disconnect(); err != nil {
		return err
	}
	return nil
}

//-----------------------------------------------
type cmdConnect struct {
	flags.CmdInfo
	gateway         string
	obfsproxy       bool
	firewall        bool
	dns             string
	antitracker     bool
	antitrackerHard bool
}

func (c *cmdConnect) Init() {
	c.Initialize("connect", "Establish new VPN connection")
	c.DefaultStringVar(&c.gateway, "SERVER_ID")
	c.BoolVar(&c.obfsproxy, "o", false, "OpenVPN only: Use obfsproxy (only enable if you have trouble connecting)")
	c.BoolVar(&c.obfsproxy, "obfsproxy", false, "OpenVPN only: Use obfsproxy (only enable if you have trouble connecting)")

	c.BoolVar(&c.firewall, "f", false, "Enable firewall (will be disabled after disconnection)")
	c.BoolVar(&c.firewall, "firewall", false, "Enable firewall (will be disabled after disconnection)")

	c.StringVar(&c.dns, "dns", "", "DNS_IP", "Use custom DNS for this connection\n(if 'antitracker' is enabled - this parameter will be ignored)")

	c.BoolVar(&c.antitracker, "antitracker", false, "Enable antitracker for this connection")
	c.BoolVar(&c.antitrackerHard, "antitracker_hard", false, "Enable 'hardcore' antitracker for this connection")
}
func (c *cmdConnect) Run() error {
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
