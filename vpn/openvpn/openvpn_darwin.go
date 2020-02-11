package openvpn

import (
	"ivpn/daemon/service/dns"
	"net"
)

type platformSpecificProperties struct {
	// no specific properties for macOS implementation
}

func (o *OpenVPN) implOnConnected() error {
	// not in use in macOS implementation
	return nil
}

func (o *OpenVPN) implOnDisconnected() error {
	// not in use in macOS implementation
	return nil
}

func (o *OpenVPN) implOnPause() error {
	return dns.Pause()
}

func (o *OpenVPN) implOnResume() error {
	return dns.Resume()
}

func (o *OpenVPN) implOnSetManualDNS(addr net.IP) error {
	return dns.SetManual(addr, nil)
}

func (o *OpenVPN) implOnResetManualDNS() error {
	return dns.DeleteManual(nil)
}
