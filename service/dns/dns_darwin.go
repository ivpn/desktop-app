package dns

import (
	"fmt"
	"net"

	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/shell"
)

func implPause() error {
	err := shell.Exec(log, platform.DNSScript(), "-pause")
	if err != nil {
		return fmt.Errorf("DNS pause: Failed to change DNS: %w", err)
	}
	return nil
}

func implResume() error {
	err := shell.Exec(log, platform.DNSScript(), "-resume")
	if err != nil {
		return fmt.Errorf("DNS resume: Failed to change DNS: %w", err)
	}

	return nil
}

// Set manual DNS.
// 'addr' parameter - DNS IP value
// 'localInterfaceIP' - not in use for macOS implementation
func implSetManual(addr net.IP, localInterfaceIP net.IP) error {
	err := shell.Exec(log, platform.DNSScript(), "-set_alternate_dns", addr.String())
	if err != nil {
		return fmt.Errorf("set manual DNS: Failed to change DNS: %w", err)
	}

	return nil
}

// DeleteManual - reset manual DNS configuration to default (DHCP)
// 'localInterfaceIP' (obligatory only for Windows implementation) - local IP of VPN interface
func implDeleteManual(localInterfaceIP net.IP) error {
	err := shell.Exec(log, platform.DNSScript(), "-delete_alternate_dns")
	if err != nil {
		return fmt.Errorf("reset manual DNS: Failed to change DNS: %w", err)
	}

	return nil
}
