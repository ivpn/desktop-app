package dns

import (
	"net"

	"github.com/ivpn/desktop-app-daemon/helpers"
)

func implPause() error {
	// TODO: not implemented
	return helpers.NewErrNotImplemented()
}

func implResume() error {
	// TODO: not implemented
	return helpers.NewErrNotImplemented()
}

// Set manual DNS.
// 'addr' parameter - DNS IP value
// 'localInterfaceIP' - not in use for macOS implementation
func implSetManual(addr net.IP, localInterfaceIP net.IP) error {
	// TODO: not implemented
	return helpers.NewErrNotImplemented()
}

// DeleteManual - reset manual DNS configuration to default (DHCP)
// 'localInterfaceIP' (obligatory only for Windows implementation) - local IP of VPN interface
func implDeleteManual(localInterfaceIP net.IP) error {
	// TODO: not implemented
	return helpers.NewErrNotImplemented()
}
