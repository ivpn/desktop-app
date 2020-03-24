package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ivpn/desktop-app-cli/flags"
)

type CmdServers struct {
	flags.CmdInfo
	protocol string
	filter   string
}

func (c *CmdServers) Init() {
	c.Initialize("servers", "Show servers list")
	c.StringVar(&c.protocol, "p", "", "PROTO", "Show only servers for the given protocol. Possible values: 'wireguard' ('wg'), 'openvpn' ('ovpn')")
	c.StringVar(&c.protocol, "protocol", "", "PROTO", "Show only servers for the given protocol. Possible values: 'wireguard' ('wg'), 'openvpn' ('ovpn')")

	c.StringVar(&c.filter, "f", "", "MASK", "Filter servers: show only servers which contains <filter_string> in description (eg. -f 'US')")
	c.StringVar(&c.filter, "filter", "", "MASK", "Filter servers: show only servers which contains <filter_string> in description (eg. -f 'US')")
}
func (c *CmdServers) Run() error {
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
