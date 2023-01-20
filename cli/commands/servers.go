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
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/vpn"

	"github.com/ivpn/desktop-app/cli/flags"
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
	hosts        bool
	load         bool
	filterInvert bool
}

func (c *CmdServers) Init() {
	c.Initialize("servers", "Show servers list\n(FILTER - optional parameter: show only servers which contains FILTER in server description)")
	c.DefaultStringVar(&c.filter, "FILTER")

	c.StringVar(&c.proto, "p", "", "PROTOCOL", "Protocol type OpenVPN|ovpn|WireGuard|wg")
	c.StringVar(&c.proto, "protocol", "", "PROTOCOL", "Protocol type OpenVPN|ovpn|WireGuard|wg")

	c.BoolVar(&c.location, "l", false, "Apply FILTER to server location (Hostname)")
	c.BoolVar(&c.location, "location", false, "Apply FILTER to server location (Hostname)")

	c.BoolVar(&c.country, "c", false, "Apply FILTER to country name")
	c.BoolVar(&c.country, "country", false, "Apply FILTER to country name")

	c.BoolVar(&c.countryCode, "cc", false, "Apply FILTER to country code")
	c.BoolVar(&c.countryCode, "country_code", false, "Apply FILTER to country code")

	c.BoolVar(&c.city, "city", false, "Apply FILTER to city name")

	c.BoolVar(&c.ping, "ping", false, "Ping servers and view ping result")

	c.BoolVar(&c.hosts, "hosts", false, "Show location hosts")
	c.BoolVar(&c.load, "load", false, "Show load info for each host")

	c.BoolVar(&c.filterInvert, "filter_invert", false, "Invert filtering result")
}
func (c *CmdServers) Run() error {
	var servers apitypes.ServersInfoResponse
	var err error

	isServersLoaded := false
	if c.load {
		fmt.Println("Updating servers load info...")
		c.hosts = true                                // show also host info
		servers, err = _proto.GetServersForceUpdate() // force update servers info (we need latest host load statuses)
		if err != nil {
			fmt.Println("Failed to update servers load info. Using cached data!")
		} else {
			isServersLoaded = true
		}
	}

	if !isServersLoaded {
		servers, err = _proto.GetServers()
		if err != nil {
			return err
		}
	}

	slist := serversList(servers)

	if c.ping {
		var vpnType *vpn.Type = nil
		if len(c.proto) > 0 {
			if p, err := getVpnTypeByFlag(c.proto); err == nil {
				vpnType = &p
			}
		}
		if err := serversPing(slist, true, c.hosts, vpnType); err != nil {
			return err
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight|tabwriter.Debug)

	pingHeader := ""
	hostsHeader := ""
	hostsLoadHeader := ""
	if c.ping {
		pingHeader = "PING\t"
	}
	if c.hosts {
		hostsHeader = "HOSTS\t"
		if c.load {
			hostsLoadHeader = "LOAD\t"
		}
	}

	fmt.Fprintln(w, "PROTOCOL\tLOCATION\tCITY\tCOUNTRY\tISP\tIPv? tunnel\t"+pingHeader+hostsHeader+hostsLoadHeader)

	helloResp := _proto.GetHelloResponse()
	isWgDisabled := len(helloResp.DisabledFunctions.WireGuardError) > 0
	isOpenVPNDisabled := len(helloResp.DisabledFunctions.OpenVPNError) > 0

	svrs := serversFilter(isWgDisabled, isOpenVPNDisabled,
		slist, c.filter, c.proto, c.location, c.city, c.countryCode, c.country, c.filterInvert)
	for _, s := range svrs {
		str := ""
		IPvInfo := "IPv4"
		if s.isIPv6Tunnel {
			IPvInfo = "IPv4/IPv6"
		}

		pingStr := ""
		if c.ping {
			pingStr = " ?  \t"
			if s.pingMs > 0 {
				pingStr = fmt.Sprintf("%dms\t", s.pingMs)
			}
		}

		firstHostStr := ""
		firstHostLoadStr := ""
		if c.hosts {
			firstHostStr = "\t"
			if len(s.hosts) > 0 {
				firstHostStr = s.hosts[0].hostname + "\t"
				if c.load {

					firstHostLoadStr = fmt.Sprintf("%d", int(s.hosts[0].load+0.5)) + "%\t"
				}
			}
		}

		str = fmt.Sprintf("%s\t%s\t%s (%s)\t %s\t%s\t%s\t%s%s%s", s.protocol, s.gateway, s.city, s.countryCode, s.country, s.isp, IPvInfo, pingStr, firstHostStr, firstHostLoadStr)
		fmt.Fprintln(w, str)

		if c.hosts && len(s.hosts) > 1 {
			for _, h := range s.hosts[1:] {
				if c.ping {
					pingStr = " ?  \t"
					if h.pingMs > 0 {
						pingStr = fmt.Sprintf("%dms\t", h.pingMs)
					}
				}

				loadStr := ""
				if c.load {
					loadStr = fmt.Sprintf("%d", int(h.load+0.5)) + "%\t"
				}
				str = fmt.Sprintf("%s\t%s\t%s %s\t %s\t%s\t%s\t%s%s%s", "", "", "", "", "", "", "", pingStr, h.hostname+"\t", loadStr)
				fmt.Fprintln(w, str)
			}
		}

	}

	w.Flush()

	if isOpenVPNDisabled {
		fmt.Println("WARNING: OpenVPN servers were not shown because OpenVPN functionality disabled:\n\t", helloResp.DisabledFunctions.OpenVPNError)
	}
	if isWgDisabled {
		fmt.Println("WARNING: WireGuard servers were not shown because WireGuard functionality disabled:\n\t", helloResp.DisabledFunctions.WireGuardError)
	}

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
	svrs := serversListByVpnType(servers, vpn.WireGuard)
	svrs = append(svrs, serversListByVpnType(servers, vpn.OpenVPN)...)
	return svrs
}

func serversListByVpnType(servers apitypes.ServersInfoResponse, t vpn.Type) []serverDesc {

	var ret []serverDesc
	if t == vpn.WireGuard {
		ret = make([]serverDesc, 0, len(servers.WireguardServers))

		for _, s := range servers.WireguardServers {
			hosts := make([]hostDesc, 0, len(s.Hosts))

			isIPv6Tunnel := false
			for _, h := range s.Hosts {
				if len(h.IPv6.LocalIP) > 0 {
					isIPv6Tunnel = true
				}
				hosts = append(hosts, hostDesc{host: strings.TrimSpace(h.Host), hostname: strings.TrimSpace(h.Hostname), load: h.Load})
			}
			ret = append(ret, serverDesc{protocol: ProtoName_WireGuard, gateway: s.Gateway, city: s.City, countryCode: s.CountryCode, country: s.Country, isp: s.ISP, hosts: hosts, isIPv6Tunnel: isIPv6Tunnel})
		}
	} else {
		ret = make([]serverDesc, 0, len(servers.OpenvpnServers))

		for _, s := range servers.OpenvpnServers {
			hosts := make([]hostDesc, 0, len(s.Hosts))

			for _, h := range s.Hosts {
				hosts = append(hosts, hostDesc{host: strings.TrimSpace(h.Host), hostname: strings.TrimSpace(h.Hostname), load: h.Load})
			}
			ret = append(ret, serverDesc{protocol: ProtoName_OpenVPN, gateway: s.Gateway, city: s.City, countryCode: s.CountryCode, country: s.Country, isp: s.ISP, hosts: hosts})
		}
	}
	return ret
}

func serversFilter(isWgDisabled bool, isOvpnDisabled bool, servers []serverDesc, mask string, proto string, useGw, useCity, useCCode, useCountry, invertFilter bool) (svrs []serverDesc) {
	if isWgDisabled || isOvpnDisabled {
		oldSvrs := servers
		servers = make([]serverDesc, 0, len(oldSvrs))
		for _, s := range oldSvrs {
			if isWgDisabled && s.protocol == ProtoName_WireGuard {
				continue
			}
			if isOvpnDisabled && s.protocol == ProtoName_OpenVPN {
				continue
			}
			servers = append(servers, s)
		}
	}

	if len(mask) == 0 && len(proto) == 0 {
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

		if len(mask) == 0 {
			isOK = true
		}

		if (checkAll || useGw) && strings.ToLower(s.gateway) == mask {
			isOK = true
		}
		if (checkAll || useCity) && strings.Contains(strings.ToLower(s.city), mask) {
			isOK = true
		}
		if (checkAll || useCCode) && strings.ToLower(s.countryCode) == mask {
			isOK = true
		}
		if (checkAll || useCountry) && strings.Contains(strings.ToLower(s.country), mask) {
			isOK = true
		}

		for _, h := range s.hosts {
			if h.hostname == mask {
				isOK = true
				break
			}
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

func serversPing(servers []serverDesc, needSort bool, pingAllHostsOnFirstPhase bool, vpnTypePrioritized *vpn.Type) error {
	fmt.Println("Pinging servers ...")
	pingRes, err := _proto.PingServers(pingAllHostsOnFirstPhase, vpnTypePrioritized)
	if err != nil {
		return err
	}
	if len(pingRes) == 0 {
		return fmt.Errorf("failed to ping servers")
	}

	for _, pr := range pingRes {
		for i, s := range servers {
			serverPing := 0
			// set ping result for each host
			for j, h := range s.hosts {
				if h.host == pr.Host {
					h.pingMs = pr.Ping
					s.hosts[j] = h
					if serverPing <= 0 || serverPing > pr.Ping {
						serverPing = pr.Ping
					}
				}
			}
			// set min ping result for server
			if serverPing > 0 {
				s.pingMs = serverPing
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

type hostDesc struct {
	hostname string
	host     string // ip
	pingMs   int
	load     float32
}

type serverDesc struct {
	protocol     string
	gateway      string
	city         string
	countryCode  string
	country      string
	isp          string
	hosts        []hostDesc
	pingMs       int
	isIPv6Tunnel bool
}

func (s *serverDesc) String() string {
	return fmt.Sprintf("%s, %s (%s), %s", s.gateway, s.city, s.countryCode, s.country)
}
