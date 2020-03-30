package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	apitypes "github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/vpn"

	"github.com/ivpn/desktop-app-cli/flags"
)

const (
	ProtoName_OpenVPN   = "OpenVPN"
	ProtoName_WireGuard = "WireGuard"
)

type CmdServers struct {
	flags.CmdInfo
	proto        string
	location     bool
	city         bool
	country      bool
	countryCode  bool
	filter       string
	ping         bool
	filterInvert bool
}

func (c *CmdServers) Init() {
	c.Initialize("servers", "Show servers list\n(FILTER - optional parameter: show only servers which contains FILTER in server description)")
	c.DefaultStringVar(&c.filter, "FILTER")

	c.StringVar(&c.proto, "p", "", "PROTOCOL", "Protocol type [WireGuard/OpenVPN] (can be used short names 'wg' or 'ovpn')")

	c.BoolVar(&c.location, "l", false, "Apply FILTER to server location (serverID)")

	c.BoolVar(&c.country, "c", false, "Apply FILTER to country name")
	c.BoolVar(&c.country, "country", false, "Apply FILTER to country name")

	c.BoolVar(&c.countryCode, "cc", false, "Apply FILTER to country code")
	c.BoolVar(&c.countryCode, "countrycode", false, "Apply FILTER to country code")

	c.BoolVar(&c.city, "city", false, "Apply FILTER to city name")

	c.BoolVar(&c.ping, "ping", false, "Ping servers and view ping result")
	c.BoolVar(&c.filterInvert, "filter_invert", false, "Invert filtering result")
}
func (c *CmdServers) Run() error {
	servers, err := _proto.GetServers()
	if err != nil {
		return err
	}

	slist := serversList(servers)

	if c.ping {
		if err := serversPing(slist, true); err != nil {
			return err
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight|tabwriter.Debug)

	if c.ping {
		fmt.Fprintln(w, "PROTOCOL\tLOCATION\tCITY\tCOUNTRY\tPING\t")
	} else {
		fmt.Fprintln(w, "PROTOCOL\tLOCATION\tCITY\tCOUNTRY\t")
	}

	svrs := serversFilter(slist, c.filter, c.proto, c.location, c.city, c.countryCode, c.country, c.filterInvert)
	for _, s := range svrs {
		str := ""
		if c.ping {
			str = fmt.Sprintf("%s\t%s\t%s (%s)\t %s\t%dms\t", s.protocol, s.gateway, s.city, s.countryCode, s.country, s.pingMs)
		} else {
			str = fmt.Sprintf("%s\t%s\t%s (%s)\t %s\t", s.protocol, s.gateway, s.city, s.countryCode, s.country)
		}
		fmt.Fprintln(w, str)
	}

	w.Flush()

	return nil
}

// ---------------------

func getVpnTypeByFlag(proto string) (t vpn.Type, err error) {
	proto = strings.ToLower(proto)

	if len(proto) == 0 {
		return t, fmt.Errorf("parameter is empty")
	}

	if proto == "wg" || proto == strings.ToLower(ProtoName_WireGuard) {
		return vpn.WireGuard, nil
	}

	if proto == "ovpn" || proto == strings.ToLower(ProtoName_OpenVPN) {
		return vpn.OpenVPN, nil
	}

	return t, flags.BadParameter{Message: "protocol definition not correct"}
}

func serversList(servers apitypes.ServersInfoResponse) []serverDesc {
	ret := make([]serverDesc, 0, len(servers.OpenvpnServers)+len(servers.WireguardServers))
	for _, s := range servers.WireguardServers {
		hosts := make(map[string]struct{}, len(s.Hosts))
		for _, h := range s.Hosts {
			hosts[strings.ToLower(strings.TrimSpace(h.Host))] = struct{}{}
		}
		ret = append(ret, serverDesc{protocol: ProtoName_WireGuard, gateway: s.Gateway, city: s.City, countryCode: s.CountryCode, country: s.Country, hosts: hosts})
	}
	for _, s := range servers.OpenvpnServers {
		hosts := make(map[string]struct{}, len(s.IPAddresses))
		for _, h := range s.IPAddresses {
			hosts[strings.ToLower(strings.TrimSpace(h))] = struct{}{}
		}
		ret = append(ret, serverDesc{protocol: ProtoName_OpenVPN, gateway: s.Gateway, city: s.City, countryCode: s.CountryCode, country: s.Country, hosts: hosts})
	}
	return ret
}

func serversFilter(servers []serverDesc, mask string, proto string, useGw, useCity, useCCode, useCountry, invertFilter bool) []serverDesc {
	if len(mask) == 0 {
		return servers
	}
	mask = strings.ToLower(mask)
	checkAll := !(useGw || useCity || useCCode || useCountry)

	ret := make([]serverDesc, 0, len(servers))
	for _, s := range servers {
		isOK := false

		if len(proto) > 0 {
			sProto, err1 := getVpnTypeByFlag(s.protocol)
			fProto, err2 := getVpnTypeByFlag(proto)
			if sProto != fProto || err1 != nil || err2 != nil {
				continue
			}
		}

		if (checkAll || useGw) && strings.ToLower(s.gateway) == mask {
			isOK = true
		}
		if (checkAll || useCity) && strings.ToLower(s.city) == mask {
			isOK = true
		}
		if (checkAll || useCCode) && strings.ToLower(s.countryCode) == mask {
			isOK = true
		}
		if (checkAll || useCountry) && strings.ToLower(s.country) == mask {
			isOK = true
		}

		if invertFilter {
			isOK = !isOK
		}
		if isOK {
			ret = append(ret, s)
		}
	}
	return ret
}

func serversPing(servers []serverDesc, needSort bool) error {
	fmt.Println("Pinging servers ...")
	pingRes, err := _proto.PingServers()
	if err != nil {
		return err
	}
	if len(pingRes) == 0 {
		return fmt.Errorf("failed to ping servers")
	}

	for _, pr := range pingRes {
		for i, s := range servers {
			if _, ok := s.hosts[pr.Host]; ok {
				s.pingMs = pr.Ping
				servers[i] = s
			}
		}
	}

	if needSort {
		sort.Slice(servers, func(i, j int) bool {
			if servers[i].pingMs == 0 && servers[j].pingMs == 0 {
				return strings.Compare(servers[i].city, servers[j].city) < 0
			} else if servers[i].pingMs <= 0 {
				return true
			} else if servers[j].pingMs <= 0 {
				return false
			}

			return servers[i].pingMs > servers[j].pingMs
		})
	}

	return nil
}

type serverDesc struct {
	protocol    string
	gateway     string
	city        string
	countryCode string
	country     string
	hosts       map[string]struct{}
	pingMs      int
}

func (s *serverDesc) String() string {
	return fmt.Sprintf("%s\t%s\t%s (%s)\t %s\t", s.protocol, s.gateway, s.city, s.countryCode, s.country)
}
