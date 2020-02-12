package service

import (
	"net"

	"github.com/ivpn/desktop-app-daemon/service/api"
	"github.com/ivpn/desktop-app-daemon/service/firewall"
)

func (s *service) implIsGoingToPingServers(servers *api.ServersInfoResponse) error {

	hosts := make([]net.IP, 0, len(servers.OpenvpnServers)+len(servers.WireguardServers))

	// OpenVPN servers
	for _, s := range servers.OpenvpnServers {
		if len(s.IPAddresses) <= 0 {
			continue
		}
		ip := net.ParseIP(s.IPAddresses[0])
		if ip != nil {
			hosts = append(hosts, ip)
		}
	}

	// ping each WireGuard server
	for _, s := range servers.WireguardServers {
		if len(s.Hosts) <= 0 {
			continue
		}

		ip := net.ParseIP(s.Hosts[0].Host)
		if ip != nil {
			hosts = append(hosts, ip)
		}
	}

	return firewall.AddHostsToExceptions(hosts)
}
