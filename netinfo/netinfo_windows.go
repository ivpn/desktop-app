package netinfo

import (
	"bytes"
	"fmt"
	"net"
)

// doDefaultGatewayIP - returns: default gateway IP
func doDefaultGatewayIP() (defGatewayIP net.IP, err error) {
	routes, err := getWindowsIPv4Routes()
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	for _, route := range routes {
		// Eample:
		// Network 		Destination  	Netmask   		Gateway    		Interface  Metric
		// 0.0.0.0   	0.0.0.0      	192.168.1.1 	192.168.1.248	35
		// 0.0.0.0 		128.0.0.0      	10.59.44.1   	10.59.44.2  	15 <- route to virtual VPN interface !!!
		zeroBytes := []byte{0, 0, 0, 0}
		if bytes.Equal(route.DwForwardDest[:], zeroBytes) && bytes.Equal(route.DwForwardMask[:], zeroBytes) { // Network == 0.0.0.0 && Netmask == 0.0.0.0
			return net.IPv4(route.DwForwardNextHop[0],
					route.DwForwardNextHop[1],
					route.DwForwardNextHop[2],
					route.DwForwardNextHop[3]),
				nil
		}
	}

	return nil, fmt.Errorf("failed to determine default route")
}
