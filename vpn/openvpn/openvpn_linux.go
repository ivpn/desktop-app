package openvpn

import (
	"fmt"
	"net"

	"github.com/ivpn/desktop-app-daemon/service/dns"
	"github.com/ivpn/desktop-app-daemon/service/platform"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

type platformSpecificProperties struct {
	// no specific properties for Linux implementation
	isCanUseParamsV24 bool
}

func (o *OpenVPN) implInit() error {
	o.psProps.isCanUseParamsV24 = true

	if err := platform.CheckExecutableRights("OpenVPN binary", o.binaryPath); err != nil {
		return nil
	}

	// Check OpenVPN minimum version
	minVer := []int{2, 3}
	verNums := GetOpenVPNVersion(o.binaryPath)
	for i := range minVer {
		if len(verNums) <= i {
			continue
		}
		if verNums[i] < minVer[i] {
			return fmt.Errorf("OpenVPN version '%v' not supported (minimum required version '%v')", verNums, minVer)
		}
	}
	if len(verNums) >= 2 && verNums[0] == 2 && verNums[1] < 4 {
		o.psProps.isCanUseParamsV24 = false
	}
	return nil
}

func (o *OpenVPN) implIsCanUseParamsV24() bool {
	return o.psProps.isCanUseParamsV24
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
