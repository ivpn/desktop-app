package openvpn

import (
	"net"

	"github.com/ivpn/desktop-app-daemon/service/dns"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

type platformSpecificProperties struct {
	manualDNS net.IP
}

func (o *OpenVPN) implOnConnected() error {
	// on Windows it is not possible to change network interface properties (over WMI) until it not enabled
	// apply DNS value when VPN connected (TAP interface enabled)
	if o.psProps.manualDNS != nil {
		return dns.SetManual(o.psProps.manualDNS, o.clientIP)
	}

	// There could be manual-dns value saved from last connection in adapter properties. We must ensure that it erased.
	return dns.DeleteManual(o.clientIP)
}

func (o *OpenVPN) implOnDisconnected() error {
	return o.implOnResetManualDNS()
}

func (o *OpenVPN) implOnPause() error {
	// not in use in Windows implementation
	return nil
}

func (o *OpenVPN) implOnResume() error {
	// not in use in Windows implementation
	return nil
}

func (o *OpenVPN) implOnSetManualDNS(addr net.IP) error {
	o.psProps.manualDNS = addr

	if o.state != vpn.CONNECTED {
		// on Windows it is not possible to change network interface properties (over WMI) until it not enabled
		// apply DNS value when VPN connected (TAP interface enabled)
	} else {
		return dns.SetManual(o.psProps.manualDNS, o.clientIP)
	}
	return nil
}

func (o *OpenVPN) implOnResetManualDNS() error {
	if o.psProps.manualDNS != nil {
		o.psProps.manualDNS = nil
		return dns.DeleteManual(o.clientIP)
	}
	return nil
}
