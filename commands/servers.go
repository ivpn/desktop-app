package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	apitypes "github.com/ivpn/desktop-app-daemon/api/types"

	"github.com/ivpn/desktop-app-cli/flags"
)

type CmdServers struct {
	flags.CmdInfo
	proto       bool
	location    bool
	city        bool
	country     bool
	countryCode bool
	filter      string
}

func (c *CmdServers) Init() {
	c.Initialize("servers", "Show servers list\n(FILTER - optional parameter: show only servers which contains FILTER in server description)")
	c.DefaultStringVar(&c.filter, "FILTER")

	c.BoolVar(&c.proto, "p", false, "Apply FILTER to protocol type (can be used short names 'wg' or 'ovpn')")
	c.BoolVar(&c.location, "l", false, "Apply FILTER to server location (serverID)")

	c.BoolVar(&c.country, "c", false, "Apply FILTER to country name")
	c.BoolVar(&c.country, "country", false, "Apply FILTER to country name")

	c.BoolVar(&c.countryCode, "cc", false, "Apply FILTER to country code")
	c.BoolVar(&c.countryCode, "countrycode", false, "Apply FILTER to country code")

	c.BoolVar(&c.city, "city", false, "Apply FILTER to city name")
}
func (c *CmdServers) Run() error {
	servers, err := _proto.GetServers()
	if err != nil {
		return err
	}

	const padding = 1
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', tabwriter.AlignRight|tabwriter.Debug)

	fmt.Fprintln(w, "PROTOCOL\tLOCATION\tCITY\tCOUNTRY\t")

	svrs := serversFilter(serversList(servers), c.filter, c.proto, c.location, c.city, c.countryCode, c.country)
	for _, s := range svrs {
		str := fmt.Sprintf("%s\t%s\t%s (%s)\t %s\t", s.protocol, s.gateway, s.city, s.countryCode, s.country)
		fmt.Fprintln(w, str)
	}

	w.Flush()

	return nil
}

// ---------------------
func serversList(servers apitypes.ServersInfoResponse) []serverDesc {
	ret := make([]serverDesc, 0, len(servers.OpenvpnServers)+len(servers.WireguardServers))
	for _, s := range servers.WireguardServers {
		ret = append(ret, serverDesc{protocol: "WireGuard", gateway: s.Gateway, city: s.City, countryCode: s.CountryCode, country: s.Country})
	}
	for _, s := range servers.OpenvpnServers {
		ret = append(ret, serverDesc{protocol: "OpenVPN", gateway: s.Gateway, city: s.City, countryCode: s.CountryCode, country: s.Country})
	}
	return ret
}

func serversFilter(servers []serverDesc, mask string, useProto, useGw, useCity, useCCode, useCountry bool) []serverDesc {
	if len(mask) == 0 {
		return servers
	}

	mask = strings.ToLower(mask)
	checkAll := !(useProto || useGw || useCity || useCCode || useCountry)

	ret := make([]serverDesc, 0, len(servers))
	for _, s := range servers {
		isOK := false
		if checkAll || useProto {
			if strings.ToLower(s.protocol) == mask || (mask == "wg" && s.protocol == "WireGuard") || (mask == "ovpn" && s.protocol == "OpenVPN") {
				isOK = true
			}
		}
		if (checkAll || useGw) && strings.Contains(strings.ToLower(s.gateway), mask) {
			isOK = true
		}
		if (checkAll || useCity) && strings.Contains(strings.ToLower(s.city), mask) {
			isOK = true
		}
		if (checkAll || useCCode) && strings.Contains(strings.ToLower(s.countryCode), mask) {
			isOK = true
		}
		if (checkAll || useCountry) && strings.Contains(strings.ToLower(s.country), mask) {
			isOK = true
		}

		if isOK {
			ret = append(ret, s)
		}
	}
	return ret
}

type serverDesc struct {
	protocol    string
	gateway     string
	city        string
	countryCode string
	country     string
}

func (s *serverDesc) String() string {
	return fmt.Sprintf("%s\t%s\t%s (%s)\t %s\t", s.protocol, s.gateway, s.city, s.countryCode, s.country)
}
