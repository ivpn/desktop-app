package netinfo

import (
	"errors"
	"fmt"
	"net"

	"github.com/ivpn/desktop-app-daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("netinf")
}

// DefaultGatewayIP - returns: default gatewat IP
func DefaultGatewayIP() (defGatewayIP net.IP, err error) {
	// method should be implemented in platform-specific file
	return doDefaultGatewayIP()
}

// InterfaceByIPAddr - Get network interface object by it's local IP address
func InterfaceByIPAddr(localAddr net.IP) (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, ifs := range ifaces {

		addrs, _ := ifs.Addrs()
		if addrs == nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil {
				if ip.Equal(localAddr) {
					return &ifs, nil
				}
			}
		}
	}
	return nil, errors.New("not found network interface with address:" + localAddr.String())
}

// GetFreePort - get unused TCP local port
// Note there is no guarantee that port will not be in use right after finding it
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to resolve TCP address: %w", err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("failed to start listener: %w", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// GetInterfaceByIndex - get interface info by its index
func GetInterfaceByIndex(index int) (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, ifs := range ifaces {
		if ifs.Index == index {
			return &ifs, nil
		}
	}
	return nil, nil
}

// GetAllLocalV4Addresses - returns IPv4 addresses of all local interfaces
func GetAllLocalV4Addresses() ([]net.IPNet, error) {
	return getAllLocalAddresses(nil, false)
}

// GetAllLocalV6Addresses - returns IPv6 addresses of all local interfaces
func GetAllLocalV6Addresses() ([]net.IPNet, error) {
	return getAllLocalAddresses(nil, true)
}

/*
// GetInterfaceV4Addresses - returns IPv4 addresses of the local interface
func GetInterfaceV4Addresses(inf net.Interface) ([]net.IPNet, error) {
	return getAllLocalAddresses([]net.Interface{inf}, false)
}

// GetInterfaceV6Addresses - returns IPv6 addresses of the local interfaces
func GetInterfaceV6Addresses(inf net.Interface) ([]net.IPNet, error) {
	return getAllLocalAddresses([]net.Interface{inf}, true)
}*/

func getAllLocalAddresses(ifaces []net.Interface, isIPv6 bool) ([]net.IPNet, error) {
	ret := make([]net.IPNet, 0, 8)

	var err error
	if len(ifaces) == 0 {
		ifaces, err = net.Interfaces()
		if err != nil {
			return ret, fmt.Errorf("failed to get network interfaces: %w", err)
		}
	}

	for _, ifs := range ifaces {
		if ifs.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, _ := ifs.Addrs()
		if addrs == nil {
			continue
		}

		for _, addr := range addrs {
			var ip *net.IPNet

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v
			}

			if ip == nil {
				continue
			}

			isIPv6Addr := ip.IP.To4() == nil // check address is IPv6
			if isIPv6Addr != isIPv6 {
				continue // ignore unexpected IP address family (v4 or v6)
			}

			ret = append(ret, *ip)
		}
	}
	return ret, nil
}
