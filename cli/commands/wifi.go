//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the IVPN command line interface.
//
//  The IVPN command line interface is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The IVPN command line interface is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the IVPN command line interface. If not, see <https://www.gnu.org/licenses/>.
//

package commands

import (
	"fmt"
	"os"
	"runtime"
	"text/tabwriter"

	"github.com/ivpn/desktop-app/cli/flags"
)

type CmdWiFi struct {
	flags.CmdInfo
	status                 bool
	background_control     string //[on/off]
	connect_on_insecure    string //[on/off]
	trusted_control        string //[on/off]
	default_trust_status   string //[none/trusted/untrusted]
	set_trusted_action     string // [action:value] // actions: 'trusted_vpn_off:[true/false]', 'trusted_firewall_off', 'untrusted_vpn_on', 'untrusted_firewall_on'
	set_trusted_network    string // [network:status] (status: none/trusted/untrusted; e.g. 'my_home_wifi':trusted)
	reset_trusted_settings bool
}

func (c *CmdWiFi) Init() {
	c.KeepArgsOrderInHelp = true

	c.Initialize("wifi", "WiFi control settings")
	c.BoolVar(&c.status, "status", false, "(default) Show settings")
	c.StringVar(&c.background_control, "background_control", "", "[on/off]",
		`Allow background daemon to Apply WiFi Control settings
		By enabling this feature the IVPN daemon will apply the WiFi control settings
		before the IVPN app has been launched. This enables the WiFi control settings
		to be applied as quickly as possible as the daemon is started early
		in the operating system boot process and before the IVPN app (The GUI).`)
	c.StringVarEx(&c.connect_on_insecure, "connect_on_insecure", "", "[on/off]", "Autoconnect on joining WiFi networks without encryption",
		isInsecureNetworksSuppported)
	c.StringVar(&c.trusted_control, "trusted_control", "", "[on/off]",
		`Trusted/Untrusted WiFi network control
		By enabling this feature you can define a WiFi network as trusted or
		untrusted and what actions to take when joining the WiFi network.`)
	c.StringVar(&c.default_trust_status, "default_trust_status", "", "STATUS",
		`Default trust status for undefined networks
		Acceptable status values:
			* trusted   - apply 'trusted' actions when joining a network
			* untrusted - apply 'untrusted' actions when joining a network
			* none 	    - no actions will be applied when joining a network`)
	c.StringVar(&c.set_trusted_action, "set_trusted_action", "", "CONFIG",
		`Configure action for when joining wifi networks
		CONFIG parameter format: <ACTION>:<VALUE> 
			(VALUE: [on/off])
		Acceptable actions:
			Actions for Untrusted WiFi
				* trusted_vpn_off       - Disconnect from VPN
				* trusted_firewall_off  - Disable firewall
			Actions for Trusted WiFi
				* untrusted_vpn_on      - Connect to VPN
				* untrusted_firewall_on - Enable firewall
		Example: 
			ivpn -set_trusted_action untrusted_vpn_on:off
			ivpn -set_trusted_action untrusted_firewall_on:on`)
	c.StringVar(&c.set_trusted_network, "set_trusted_network", "", "CONFIG",
		`Set trust status for WiFi network
			CONFIG parameter format: '<NETWORK_NAME>':<VALUE> 
				VALUE: [none/trusted/untrusted]
				NETWORK_NAME: if empty - will be used WiFi network name which is currently connected
			Example:
				ivpn -set_trusted_network work:untrusted
				ivpn -set_trusted_network 'my home network':trusted
				Define current WiFi network as 'untrusted':
					ivpn -set_trusted_network untrusted`)

	c.BoolVar(&c.reset_trusted_settings, "reset_trusted_settings", false, "Reset settings of Trusted/Untrusted WiFi network control")
}

func (c *CmdWiFi) Run() error {

	w := c.printStatus(nil)
	w.Flush()
	return nil
}

func isInsecureNetworksSuppported() bool {
	return runtime.GOOS != "linux"
}

func (c *CmdWiFi) printStatus(w *tabwriter.Writer) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	boolToStrEx := func(v *bool, trueVal, falseVal, nullVal string) string {
		if v == nil {
			return nullVal
		}
		if *v {
			return trueVal
		}
		return falseVal
	}
	boolToStr := func(v bool) string {
		return boolToStrEx(&v, "Enabled", "Disabled", "")
	}

	wifiSettings := _proto.GetHelloResponse().DaemonSettings.WiFi
	//fmt.Fprintf(w, "Connected WiFi network\t:\t%v\n", "stenya_house") // TODO:

	fmt.Fprintf(w, "Allow background daemon to Apply WiFi Control settings\t:\t%v\n", boolToStr(wifiSettings.CanApplyInBackground))
	if isInsecureNetworksSuppported() {
		fmt.Fprintf(w, "Autoconnect on joining WiFi networks without encryption\t:\t%v\n", boolToStr(wifiSettings.ConnectVPNOnInsecureNetwork))
	}
	fmt.Fprintf(w, "Trusted/Untrusted WiFi network control\t:\t%v\n", boolToStr(wifiSettings.TrustedNetworksControl))
	fmt.Fprintf(w, "Default trust status for undefined networks\t:\t%v\n", boolToStrEx(wifiSettings.DefaultTrustStatusTrusted, "Trusted", "Untrusted", "No status"))
	fmt.Fprintf(w, "Actions:\t\n")
	fmt.Fprintf(w, "    Actions for Untrusted WiFi:\t\n")
	fmt.Fprintf(w, "        Connect to VPN\t:\t%v\n", boolToStr(wifiSettings.Actions.UnTrustedConnectVpn))
	fmt.Fprintf(w, "        Enable firewall\t:\t%v\n", boolToStr(wifiSettings.Actions.UnTrustedEnableFirewall))
	fmt.Fprintf(w, "    Actions for Trusted WiFi:\t\n")
	fmt.Fprintf(w, "        Disconnect from VPN\t:\t%v\n", boolToStr(wifiSettings.Actions.TrustedDisconnectVpn))
	fmt.Fprintf(w, "        Disable firewall\t:\t%v\n", boolToStr(wifiSettings.Actions.TrustedDisableFirewall))

	if len(wifiSettings.Networks) == 0 {
		fmt.Fprintf(w, "Networks\t:\tnot defined\n")
	} else {
		fmt.Fprintf(w, "Networks:\t\n")
		for _, n := range wifiSettings.Networks {
			fmt.Fprintf(w, "        %s\t:\t%v\n", n.SSID, boolToStrEx(&n.IsTrusted, "Trusted", "Untrusted", "No status"))
		}
	}
	return w
}
