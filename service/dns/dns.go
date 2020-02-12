package dns

import (
	"net"

	"github.com/ivpn/desktop-app-daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("dns")
}

// Pause pauses DNS (restore original DNS)
func Pause() error {
	return implPause()
}

// Resume resuming DNS (set DNS back which was before Pause)
func Resume() error {
	return implResume()
}

// SetManual - set manual DNS.
// 'addr' parameter - DNS IP value
// 'localInterfaceIP' (obligatory only for Windows implementation) - local IP of VPN interface
func SetManual(addr net.IP, localInterfaceIP net.IP) error {
	ret := implSetManual(addr, localInterfaceIP)
	return ret
}

// DeleteManual - reset manual DNS configuration to default (DHCP)
// 'localInterfaceIP' (obligatory only for Windows implementation) - local IP of VPN interface
func DeleteManual(localInterfaceIP net.IP) error {
	ret := implDeleteManual(localInterfaceIP)
	return ret
}
