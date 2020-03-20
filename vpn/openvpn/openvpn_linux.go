package openvpn

import (
	"net"

	"github.com/ivpn/desktop-app-daemon/service/dns"
)

type platformSpecificProperties struct {
	// no specific properties for Linux implementation
}

func (o *OpenVPN) implOnConnected() error {
	// TODO: not implemented
	return nil
}

func (o *OpenVPN) implOnDisconnected() error {
	// TODO: not implemented
	return nil
}

func (o *OpenVPN) implOnPause() error {
	// TODO: not implemented
	return nil
}

func (o *OpenVPN) implOnResume() error {
	// TODO: not implemented
	return nil
}

func (o *OpenVPN) implOnSetManualDNS(addr net.IP) error {
	return dns.SetManual(addr, nil)
}

func (o *OpenVPN) implOnResetManualDNS() error {
	return dns.DeleteManual(nil)
}
