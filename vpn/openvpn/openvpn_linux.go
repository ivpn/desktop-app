package openvpn

import (
	"net"
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
	// TODO: not implemented
	return nil
}

func (o *OpenVPN) implOnResetManualDNS() error {
	// TODO: not implemented
	return nil
}
