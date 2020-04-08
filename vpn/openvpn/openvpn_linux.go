package openvpn

import (
	"net"

	"github.com/ivpn/desktop-app-daemon/service/dns"
	"github.com/ivpn/desktop-app-daemon/vpn"
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

	mi := o.managementInterface
	if o.state != vpn.DISCONNECTED && o.state != vpn.EXITING && o.IsPaused() == false {
		// restore default dns pushed by OpenVPN server
		defaultDNS := mi.pushReplyDNS
		if defaultDNS != nil {
			return dns.SetManual(defaultDNS, nil)
		}
	}

	return dns.DeleteManual(nil)
}
