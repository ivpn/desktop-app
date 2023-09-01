//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package openvpn

import (
	"fmt"

	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/service/platform/filerights"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

type platformSpecificProperties struct {
	isCanUseParamsV24 bool
	manualDNS         dns.DnsSettings
}

func (o *OpenVPN) implInit() error {
	o.psProps.isCanUseParamsV24 = true

	if err := filerights.CheckFileAccessRightsExecutable(o.binaryPath); err != nil {
		return fmt.Errorf("error checking OpenVPN binary file: %w", err)
	}

	// Check OpenVPN minimum version
	minVer := []int{2, 3}
	verNums := GetOpenVPNVersion(o.binaryPath)
	log.Info("OpenVPN version:", verNums)
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
	// It is not possible to change network interface properties until it not enabled
	// apply DNS value when VPN connected (interface enabled)
	if !o.psProps.manualDNS.IsEmpty() {
		return dns.SetManual(o.psProps.manualDNS, o.clientIP)
	}

	// TODO: to think: do we really need OpenVPN config `up client.up`?

	// Normally, the DNS configuration is performed by OpenVPN by calling a script (OpenVPN config: "up client.up").
	// However, we also have to start DNS-change monitoring mechanisms. So we are applying the DNS config again and starting/initializing all necessary internal DNS components.
	//
	// We also need to do this manually to ensure that DNS was updated correctly: the client.up script does not fail in case of an error (this is intentional, so as not to break the OpenVPN connection).
	// In the SNAP environment (if there is no access to '/etc/resolv.conf'), OpenVPN connects, but DNS settings are not updated due to lack of access.
	// So, here we are trying to change the DNS again and analyzing any errors (if there are any).
	defDns := dns.DnsSettingsCreate(o.DefaultDNS())
	return dns.SetDefault(defDns, o.clientIP)
}

func (o *OpenVPN) implOnDisconnected() error {
	// TODO: not implemented
	return nil
}

func (o *OpenVPN) implOnPause() error {
	return dns.Pause(o.clientIP)
}

func (o *OpenVPN) implOnResume() error {
	defDns := dns.DnsSettingsCreate(o.DefaultDNS())
	return dns.Resume(defDns, o.clientIP)
}

func (o *OpenVPN) implOnSetManualDNS(dnsCfg dns.DnsSettings) error {
	o.psProps.manualDNS = dnsCfg

	if o.state != vpn.CONNECTED {
		// Is not possible to change network interface properties until it not enabled
		// apply DNS value when VPN connected (interface enabled)
	} else {
		return dns.SetManual(o.psProps.manualDNS, o.clientIP)
	}

	return nil
}

func (o *OpenVPN) implOnResetManualDNS() error {
	defaultDns := o.DefaultDNS()
	o.psProps.manualDNS = dns.DnsSettings{}
	if !o.IsPaused() {
		// restore default DNS pushed by OpenVPN server
		if defaultDns != nil {
			return dns.SetDefault(dns.DnsSettingsCreate(defaultDns), o.clientIP)
		}
	}

	return dns.DeleteManual(defaultDns, o.clientIP)
}

func (o *OpenVPN) implGetUpDownScriptArgs() string {
	resolvectlBinPath := platform.ResolvectlBinPath()
	if len(resolvectlBinPath) > 0 {
		extraDnsParams := dns.GetExtraSettings()
		if !extraDnsParams.Linux_IsDnsMgmtOldStyle {
			return "-use-resolvconf " + resolvectlBinPath
		}
	}
	return ""
}
