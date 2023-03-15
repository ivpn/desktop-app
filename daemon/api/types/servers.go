//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package types

import (
	"fmt"
	"strings"
)

// -----------------------------------------------------------
type ServerGeneric interface {
	GetServerInfoBase() ServerInfoBase
	GetHostsInfoBase() []HostInfoBase
}

type HostInfoBase struct {
	Hostname     string  `json:"hostname"`
	Host         string  `json:"host"`
	DnsName      string  `json:"dns_name"`
	MultihopPort int     `json:"multihop_port"`
	Load         float32 `json:"load"`
}

func (h HostInfoBase) GetHostInfoBase() HostInfoBase {
	return h
}

type ServerInfoBase struct {
	Gateway     string `json:"gateway"`
	CountryCode string `json:"country_code"`
	Country     string `json:"country"`
	City        string `json:"city"`

	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`

	ISP string `json:"isp"`
}

func (s ServerInfoBase) GetServerInfoBase() ServerInfoBase {
	return s
}

// -----------------------------------------------------------

type WireGuardServerHostInfoIPv6 struct {
	Host    string `json:"host"`
	LocalIP string `json:"local_ip"`
}

// WireGuardServerHostInfo contains info about WG server host
type WireGuardServerHostInfo struct {
	HostInfoBase
	PublicKey string                      `json:"public_key"`
	LocalIP   string                      `json:"local_ip"`
	IPv6      WireGuardServerHostInfoIPv6 `json:"ipv6"`
}

// WireGuardServerInfo contains all info about WG server
type WireGuardServerInfo struct {
	ServerInfoBase
	Hosts []WireGuardServerHostInfo `json:"hosts"`
}

func (s WireGuardServerInfo) GetHostsInfoBase() []HostInfoBase {
	ret := []HostInfoBase{}
	for _, host := range s.Hosts {
		ret = append(ret, host.HostInfoBase)
	}
	return ret
}

// -----------------------------------------------------------

type ObfsParams struct {
	Obfs3MultihopPort int    `json:"obfs3_multihop_port"`
	Obfs4MultihopPort int    `json:"obfs4_multihop_port"`
	Obfs4Key          string `json:"obfs4_key"`
}

// OpenVPNServerHostInfo contains info about OpenVPN server host
type OpenVPNServerHostInfo struct {
	HostInfoBase
	Obfs ObfsParams `json:"obfs"`
}

// OpenvpnServerInfo contains all info about OpenVPN server
type OpenvpnServerInfo struct {
	ServerInfoBase
	Hosts []OpenVPNServerHostInfo `json:"hosts"`
}

func (s OpenvpnServerInfo) GetHostsInfoBase() []HostInfoBase {
	ret := []HostInfoBase{}
	for _, host := range s.Hosts {
		ret = append(ret, host.HostInfoBase)
	}
	return ret
}

// -----------------------------------------------------------

// DNSInfo contains info about DNS server
type DNSInfo struct {
	IP string `json:"ip"`
}

// AntitrackerInfo all info about antitracker DNSs
type AntitrackerInfo struct {
	Default  DNSInfo `json:"default"`
	Hardcore DNSInfo `json:"hardcore"`
}

// InfoAPI contains API IP adresses
type InfoAPI struct {
	IPAddresses   []string `json:"ips"`
	IPv6Addresses []string `json:"ipv6s"`
}

type PortRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type PortInfo struct {
	Type  string    `json:"type"` // "TCP" or "UDP"
	Port  int       `json:"port"`
	Range PortRange `json:"range"`
}

func (pi PortInfo) String() string {
	if pi.Port > 0 {
		return fmt.Sprintf("%s:%d", pi.Type, pi.Port)
	}
	if pi.Range.Min > 0 && pi.Range.Min < pi.Range.Max {
		return fmt.Sprintf("%s:[%d-%d]", pi.Type, pi.Range.Min, pi.Range.Max)
	}
	return ""
}

func (pi PortInfo) IsTCP() bool {
	return strings.TrimSpace(strings.ToLower(pi.Type)) == "tcp"
}

func (pi PortInfo) IsUDP() bool {
	return strings.TrimSpace(strings.ToLower(pi.Type)) == "udp"
}

func (pi PortInfo) Equal(x PortInfo) bool {
	return pi.Port == x.Port &&
		strings.TrimSpace(strings.ToLower(pi.Type)) == strings.TrimSpace(strings.ToLower(x.Type)) &&
		pi.Range.Max == x.Range.Max && pi.Range.Min == x.Range.Min
}

type ObfsPortInfo struct {
	Port int `json:"port"`
}

type PortsInfo struct {
	OpenVPN   []PortInfo   `json:"openvpn"`
	WireGuard []PortInfo   `json:"wireguard"`
	Obfs3     ObfsPortInfo `json:"obfs3"`
	Obfs4     ObfsPortInfo `json:"obfs4"`
}

// ConfigInfo contains different configuration info (Antitracker, API ...)
type ConfigInfo struct {
	Antitracker AntitrackerInfo `json:"antitracker"`
	API         InfoAPI         `json:"api"`
	Ports       PortsInfo       `json:"ports"`
}

// ServersInfoResponse all info from servers.json
type ServersInfoResponse struct {
	WireguardServers []WireGuardServerInfo `json:"wireguard"`
	OpenvpnServers   []OpenvpnServerInfo   `json:"openvpn"`
	Config           ConfigInfo            `json:"config"`
}

func (si ServersInfoResponse) ServersGenericWireguard() (ret []ServerGeneric) {
	for _, s := range si.WireguardServers {
		ret = append(ret, s)
	}
	return
}

func (si ServersInfoResponse) ServersGenericOpenvpn() (ret []ServerGeneric) {
	for _, s := range si.OpenvpnServers {
		ret = append(ret, s)
	}
	return
}
