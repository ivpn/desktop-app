package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

// BadParameter error
type BadParameter struct {
	Message string
}

func (e BadParameter) Error() string {
	if len(e.Message) == 0 {
		return "bad parameter"
	}
	return e.Message
}

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
func printState(state vpn.State, connected types.ConnectedResp) {
	fmt.Printf("VPN state             : %v\n", state)

	if state != vpn.CONNECTED {
		return
	}
	since := time.Unix(connected.TimeSecFrom1970, 0)
	fmt.Printf("\tVPN type            : %v\n", connected.VpnType)
	fmt.Printf("\tLocal IP            : %v\n", connected.ClientIP)
	fmt.Printf("\tServer IP           : %v\n", connected.ServerIP)
	fmt.Printf("\tConnected           : %v\n", since)
}

func printFirewallState(isEnabled, isPersistent, isAllowLAN, isAllowMulticast bool) {
	fmt.Println("Firewall state:")
	fmt.Printf("\tEnabled             : %v\n", isEnabled)
	fmt.Printf("\tPersistent          : %v\n", isPersistent)
	fmt.Printf("\tAllow LAN           : %v\n", isAllowLAN)
	fmt.Printf("\tAllow LAN multicast : %v\n", isAllowMulticast)
}

//-----------------------------------------------
type commandBase struct {
	fs *flag.FlagSet
}

//-----------------------------------------------
type cmdLogin struct {
	commandBase
	loginAccountID string
	forceLogin     bool
}

func (c *cmdLogin) Description() (string, bool) {
	return "Login operation (register accountID)", true
}
func (c *cmdLogin) Init() {
	c.fs = flag.NewFlagSet("login", flag.ExitOnError)

	c.fs.StringVar(&c.loginAccountID, "a", "", "Account ID")
	c.fs.StringVar(&c.loginAccountID, "account", "", "Account ID")
	c.fs.BoolVar(&c.forceLogin, "force", false, "Log out from all other devices")
}
func (c *cmdLogin) FlagSet() *flag.FlagSet {
	return c.fs
}
func (c *cmdLogin) Run() error {
	if len(c.loginAccountID) == 0 {
		return BadParameter{}
	}

	return _proto.SessionNew(c.loginAccountID, c.forceLogin)
}

//-----------------------------------------------
type cmdLogout struct {
	commandBase
}

func (c *cmdLogout) Description() (string, bool) {
	return "Logout from this device (if logged-in)", false
}

func (c *cmdLogout) Init() {
	c.fs = flag.NewFlagSet("logout", flag.ExitOnError)
}

func (c *cmdLogout) FlagSet() *flag.FlagSet {
	return c.fs
}

func (c *cmdLogout) Run() error {
	return _proto.SessionDelete()
}

//-----------------------------------------------
type cmdFirewall struct {
	commandBase
	status bool
	on     bool
	off    bool
}

func (c *cmdFirewall) Description() (string, bool) {
	return "Firewall management", true
}

func (c *cmdFirewall) Init() {
	c.fs = flag.NewFlagSet("firewall", flag.ExitOnError)

	c.fs.BoolVar(&c.status, "status", false, "Show info about current firewall status")
	c.fs.BoolVar(&c.off, "off", false, "Switch-off firewall")
	c.fs.BoolVar(&c.on, "on", false, "Switch-on firewall")
}

func (c *cmdFirewall) FlagSet() *flag.FlagSet {
	return c.fs
}

func (c *cmdFirewall) Run() error {
	if c.on && c.off {
		return BadParameter{}
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
	commandBase
	protocol string
	filter   string
}

func (c *cmdServers) Description() (string, bool) {
	return "Show servers list", true
}

func (c *cmdServers) Init() {
	c.fs = flag.NewFlagSet("servers", flag.ExitOnError)
	c.fs.StringVar(&c.protocol, "p", "", "Show only servers for the given protocol. Possible values: 'wireguard' ('wg'), 'openvpn' ('ovpn')")
	c.fs.StringVar(&c.protocol, "protocol", "", "Show only servers for the given protocol. Possible values: 'wireguard' ('wg'), 'openvpn' ('ovpn')")

	c.fs.StringVar(&c.filter, "f", "", "Filter servers: show only servers which contains <filter_string> in description (eg. -f 'US')")
	c.fs.StringVar(&c.filter, "filter", "", "Filter servers: show only servers which contains <filter_string> in description (eg. -f 'US')")
}

func (c *cmdServers) FlagSet() *flag.FlagSet {
	return c.fs
}

func (c *cmdServers) Run() error {
	servers, err := _proto.GetServers()
	if err != nil {
		return err
	}

	c.protocol = strings.TrimSpace(strings.ToLower(c.protocol))

	const padding = 1
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', tabwriter.AlignRight|tabwriter.Debug)

	fmt.Fprintln(w, "PROTOCOL\tGATEWAY\tCITY\tCOUNTRY\t")

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
	commandBase
}

func (c *cmdState) Description() (string, bool) {
	return "Prints full info about IVPN state", false
}

func (c *cmdState) Init() {
	c.fs = flag.NewFlagSet("state", flag.ExitOnError)
}

func (c *cmdState) FlagSet() *flag.FlagSet {
	return c.fs
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

	printState(state, connected)
	printFirewallState(fwstate.IsEnabled, fwstate.IsPersistent, fwstate.IsAllowLAN, fwstate.IsAllowMulticast)

	return nil
}

//-----------------------------------------------
type cmdDisconnect struct {
	commandBase
}

func (c *cmdDisconnect) Description() (string, bool) {
	return "Disconnect active VPN connection (if connected)", false
}

func (c *cmdDisconnect) Init() {
	c.fs = flag.NewFlagSet("disconnect", flag.ExitOnError)
}

func (c *cmdDisconnect) FlagSet() *flag.FlagSet {
	return c.fs
}

func (c *cmdDisconnect) Run() error {
	if err := _proto.Disconnect(); err != nil {
		return err
	}
	return nil
}

//-----------------------------------------------
type cmdConnect struct {
	commandBase
	gateway string
	//port    int
}

func (c *cmdConnect) Description() (string, bool) {
	return "Establish new VPN connection", true
}
func (c *cmdConnect) Init() {
	c.fs = flag.NewFlagSet("connect", flag.ExitOnError)

	c.fs.StringVar(&c.gateway, "s", "", "Server ID (gateway)")
	c.fs.StringVar(&c.gateway, "server", "", "Server ID (gateway)")

	//c.fs.StringVar(&c.gateway, "port", "", "Connection port")

}
func (c *cmdConnect) FlagSet() *flag.FlagSet {
	return c.fs
}
func (c *cmdConnect) Run() error {
	if len(c.gateway) == 0 {
		return BadParameter{}
	}

	req := types.Connect{}
	serverFound := false

	servers, err := _proto.GetServers()
	if err != nil {
		return err
	}

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

	for _, s := range servers.OpenvpnServers {
		if s.Gateway == c.gateway {
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

	connected, err := _proto.ConnectVPN(req)

	if err != nil {
		err = fmt.Errorf("failed to connect: %w", err)
		//fmt.Printf("%v\n", err)
		fmt.Printf("Disconnecting...\n")
		if err2 := _proto.Disconnect(); err2 != nil {
			fmt.Printf("Failed to disconnect: %v\n", err2)
		}
		return err
	}

	printState(vpn.CONNECTED, connected)

	return nil
}
