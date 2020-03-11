package netinfo

import (
	"net"

	"github.com/ivpn/desktop-app-daemon/helpers"
)

// doDefaultGatewayIP - returns: default gateway
func doDefaultGatewayIP() (defGatewayIP net.IP, err error) {
	// TODO: ip route get 1.1.1.1 ???
	return nil, helpers.NewErrNotImplemented()
}
