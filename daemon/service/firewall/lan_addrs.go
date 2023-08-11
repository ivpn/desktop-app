package firewall

import (
	"fmt"
	"net"

	"github.com/ivpn/desktop-app/daemon/netinfo"
)

// getLocalIPAddresses - returns list of local IPs
func getLocalIPAddresses(isV6 bool) ([]net.IPNet, error) {
	var (
		ipnetList []net.IPNet
		err       error
	)

	if isV6 {
		ipnetList, err = netinfo.GetAllLocalV6Addresses()
	} else {
		ipnetList, err = netinfo.GetAllLocalV4Addresses()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	retIps := make([]net.IPNet, 0, 4)
	for _, ifs := range ipnetList {

		if ifs.IP.IsLoopback() ||
			(connectedClientInterfaceIP != nil && ifs.IP.Equal(connectedClientInterfaceIP)) ||
			(connectedClientInterfaceIPv6 != nil && ifs.IP.Equal(connectedClientInterfaceIPv6)) {
			continue
		}

		retIps = append(retIps, ifs)
	}

	return retIps, nil
}

// getLanIPs - returns list of local IPv4 IPs as strings
func getLanIPs() ([]string, error) {

	ipnetList, err := getLocalIPAddresses(false) // IPv4 addresses
	if err != nil {
		return nil, err
	}

	retIps := make([]string, 0, 4)
	for _, ipnet := range ipnetList {
		retIps = append(retIps, ipnet.String())
	}

	return retIps, nil
}
