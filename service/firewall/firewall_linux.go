package firewall

import (
	"net"
)

func implGetEnabled() (bool, error) {
	// TODO: not implemented
	return false, nil
}

func implSetEnabled(isEnabled bool) error {
	// TODO: not implemented
	return nil
}

func implSetPersistant(persistant bool) error {
	// TODO: not implemented
	return nil
}

// ClientConnected - allow communication for local vpn/client IP address
func implClientConnected(clientLocalIPAddress net.IP) error {
	// TODO: not implemented
	return nil
}

// ClientDisconnected - Disable communication for local vpn/client IP address
func implClientDisconnected() error {
	// TODO: not implemented
	return nil
}

func implAllowLAN(isAllowLAN bool, isAllowLanMulticast bool) error {
	// TODO: not implemented
	return nil
}

// AddHostsToExceptions - allow comminication with this hosts
// Note!: all added hosts will be removed from exceptions after client disconnection (after call 'ClientDisconnected()')
func implAddHostsToExceptions(IPs []net.IP) error {
	// TODO: not implemented
	return nil
}

// SetManualDNS - configure firewall to allow DNS which is out of VPN tunnel
// Applicable to Windows implementation (to allow custom DNS from local network)
func implSetManualDNS(addr net.IP) error {
	// TODO: not implemented
	return nil
}
