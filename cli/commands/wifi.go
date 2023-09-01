//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
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
	"strings"
	"text/tabwriter"

	"github.com/ivpn/desktop-app/cli/flags"
	"github.com/ivpn/desktop-app/cli/helpers"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
)

type actionType string

const (
	action_trusted_vpn_off       actionType = "trusted_disconnect_vpn"
	action_trusted_firewall_off  actionType = "trusted_disable_firewall"
	action_untrusted_vpn_on      actionType = "untrusted_connect_vpn"
	action_untrusted_firewall_on actionType = "untrusted_enable_firewall"
	action_untrusted_block_lan   actionType = "untrusted_block_lan"
)

type trustType string

const (
	NoTrustState trustType = "none"
	Trusted      trustType = "trusted"
	Untrusted    trustType = "untrusted"
)

type CmdWiFi struct {
	flags.CmdInfo
	status               bool
	connect_on_insecure  string //[on/off]
	trusted_control      string //[on/off]
	default_trust_status string //[none/trusted/untrusted]
	set_trusted_action   string // [action:value] // actions: 'trusted_vpn_off:[true/false]', 'trusted_firewall_off', 'untrusted_vpn_on', 'untrusted_firewall_on', untrusted_block_lan
	set_trusted_network  string // [network:status] (status: none/trusted/untrusted; e.g. 'my_home_wifi':trusted)
	reset_settings       bool
}

func (c *CmdWiFi) Init() {
	c.KeepArgsOrderInHelp = true

	c.Initialize("wifi", "WiFi control settings")
	c.BoolVar(&c.status, "status", false, "(default) Show settings")
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
			VALUE: [on/off]
			ACTION:
				Actions for Untrusted WiFi
					* untrusted_connect_vpn     - Connect to VPN
					* untrusted_enable_firewall - Enable firewall
					* untrusted_block_lan       - Block LAN traffic
				Actions for Trusted WiFi
					* trusted_disconnect_vpn    - Disconnect from VPN
					* trusted_disable_firewall  - Disable firewall
		Example: 
					ivpn wifi -set_trusted_action untrusted_connect_vpn:off
					ivpn wifi -set_trusted_action untrusted_enable_firewall:on`)
	c.StringVar(&c.set_trusted_network, "set_trusted_network", "", "CONFIG",
		`Set trust status for WiFi network
			CONFIG parameter format: '<NETWORK_NAME>':<VALUE> 
				NETWORK_NAME: if empty - will be used WiFi network name which is currently connected
				VALUE: [none/trusted/untrusted]
					(Set the value to "none" to remove the network from the list)
			Example:
					ivpn wifi -set_trusted_network work:untrusted
					ivpn wifi -set_trusted_network 'my home network':trusted
					Define current WiFi network as 'untrusted':
						ivpn wifi -set_trusted_network untrusted`)

	c.BoolVar(&c.reset_settings, "reset_settings", false, "Reset WiFi settings to defaults")
}

func (c *CmdWiFi) Run() error {
	helloResp := _proto.GetHelloResponse()
	wifiSettings := helloResp.DaemonSettings.WiFi

	isSettingsChanged := false

	if len(c.connect_on_insecure) > 0 {
		val, err := helpers.BoolParameterParse(c.connect_on_insecure) // [on/off]
		if err != nil {
			return err
		}
		if val && helloResp.ParanoidMode.IsEnabled {
			return EaaEnabledOptionNotApplicable{}
		}
		wifiSettings.ConnectVPNOnInsecureNetwork = val
		isSettingsChanged = true
	}

	if len(c.trusted_control) > 0 {
		val, err := helpers.BoolParameterParse(c.trusted_control) // [on/off]
		if err != nil {
			return err
		}
		if val && helloResp.ParanoidMode.IsEnabled {
			return EaaEnabledOptionNotApplicable{}
		}
		wifiSettings.TrustedNetworksControl = val
		isSettingsChanged = true
	}

	// change "Allow background daemon to Apply WiFi Control settings" (based on other wifi parameters)
	wifiSettings.CanApplyInBackground = wifiSettings.ConnectVPNOnInsecureNetwork || wifiSettings.TrustedNetworksControl

	if len(c.default_trust_status) > 0 {
		//[none/trusted/untrusted]

		val, isNull, err := helpers.BoolParameterParseEx(c.default_trust_status, []string{"trusted"}, []string{"untrusted"}, []string{"none"})
		if err != nil {
			return err
		}
		if isNull {
			wifiSettings.DefaultTrustStatusTrusted = nil
		} else {
			wifiSettings.DefaultTrustStatusTrusted = &val
		}
		isSettingsChanged = true
	}

	if len(c.set_trusted_action) > 0 {
		// [action:value(on/off)]; (Example: 'trusted_disconnect_vpn:on')

		c.set_trusted_action = strings.ToLower(c.set_trusted_action)

		actionParams := strings.Split(c.set_trusted_action, ":")
		if len(actionParams) != 2 {
			return flags.BadParameter{Message: "action"}
		}
		actionTypeStr := actionParams[0]
		actionValStr := actionParams[1]

		val, err := helpers.BoolParameterParse(actionValStr) // [on/off]
		if err != nil {
			return err
		}

		switch actionType(actionTypeStr) {
		case action_trusted_vpn_off:
			wifiSettings.Actions.TrustedDisconnectVpn = val
		case action_trusted_firewall_off:
			wifiSettings.Actions.TrustedDisableFirewall = val
		case action_untrusted_vpn_on:
			wifiSettings.Actions.UnTrustedConnectVpn = val
		case action_untrusted_firewall_on:
			wifiSettings.Actions.UnTrustedEnableFirewall = val
			if !val {
				wifiSettings.Actions.UnTrustedBlockLan = false
			}
		case action_untrusted_block_lan:
			wifiSettings.Actions.UnTrustedBlockLan = val
			if val {
				wifiSettings.Actions.UnTrustedEnableFirewall = true
			}
		default:
			return flags.BadParameter{
				Message: fmt.Sprintf("not supported action name '%s' (acceptable actions: %s, %s, %s, %s)", actionTypeStr,
					string(action_trusted_vpn_off),
					string(action_trusted_firewall_off),
					string(action_untrusted_vpn_on),
					string(action_untrusted_firewall_on))}
		}
		isSettingsChanged = true
	}

	if len(c.set_trusted_network) > 0 {
		//c.set_trusted_network = helpers.TrimSpacesAndRemoveQuotes(c.set_trusted_network)
		dividerIdx := strings.LastIndex(c.set_trusted_network, ":")

		netName := ""
		isTrusted := false
		isUndefined := false

		valueStr := ""
		if dividerIdx >= 0 {
			netName = c.set_trusted_network[:dividerIdx]
		}
		valueStr = c.set_trusted_network[dividerIdx+1:]

		netName = helpers.TrimSpacesAndRemoveQuotes(netName)
		valueStr = helpers.TrimSpacesAndRemoveQuotes(valueStr)

		switch strings.ToLower(valueStr) {
		case string(NoTrustState):
			isUndefined = true
		case string(Trusted):
			isTrusted = true
		case string(Untrusted):
			isTrusted = false
		default:
			return flags.BadParameter{
				Message: fmt.Sprintf("not supported trust state '%s' (acceptable values: %s, %s, %s)", valueStr,
					string(NoTrustState),
					string(Trusted),
					string(Untrusted))}
		}

		if len(netName) == 0 {
			curNet, err := _proto.GetWiFiCurrentNetwork()
			if err != nil {
				return fmt.Errorf("failed to obtain info about the currently connected WiFi network: %w", err)
			}
			netName = curNet.SSID
			if len(netName) == 0 {
				return fmt.Errorf("Unable to obtain info about currently connected WiFi network. Please, specify network name")
			}
			fmt.Printf("WiFi network not defined. Using current network: '%s'\n", netName)
		}

		// check if network already exists
		for i, n := range wifiSettings.Networks {
			if n.SSID == netName {
				if isUndefined {
					wifiSettings.Networks = append(wifiSettings.Networks[:i], wifiSettings.Networks[i+1:]...)
				} else {
					wifiSettings.Networks[i].IsTrusted = isTrusted
				}
				isSettingsChanged = true
				break
			}
		}
		if !isSettingsChanged {
			wifiSettings.Networks = append(wifiSettings.Networks, preferences.WiFiNetwork{SSID: netName, IsTrusted: isTrusted})
		}
		isSettingsChanged = true
	}

	// reset all settings
	if c.reset_settings {
		fmt.Println("Resetting settings...")
		wifiSettings = preferences.WiFiParamsCreate()
		isSettingsChanged = true
	}

	// send updated settings
	if isSettingsChanged {
		fmt.Print("Applying changes... ")
		if err := _proto.SetWiFiSettings(wifiSettings); err != nil {
			fmt.Println()
			return err
		}
		fmt.Println("Done")
	}

	// Status
	if c.status || !isSettingsChanged {
		w := c.printStatus(nil)
		w.Flush()
		PrintTips([]TipType{TipWiFiHelp})
	} else {
		PrintTips([]TipType{TipWiFiStatus})
	}
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

	curNetworkName := ""
	curNetworkInfo := ""
	curNet, err := _proto.GetWiFiCurrentNetwork()
	if err != nil {
		fmt.Println(err)
	} else {
		curNetworkName = fmt.Sprintf("%s", curNet.SSID)
		if curNet.IsInsecureNetwork {
			curNetworkInfo = fmt.Sprintf(" (no encryption)")
		}
	}

	wifiSettings := _proto.GetHelloResponse().DaemonSettings.WiFi
	fmt.Fprintf(w, "Connected WiFi network%s\t:\t%v\n", curNetworkInfo, curNetworkName)

	//fmt.Fprintf(w, "Allow background daemon to Apply WiFi Control settings\t:\t%v\n", boolToStr(wifiSettings.CanApplyInBackground))
	if isInsecureNetworksSuppported() {
		fmt.Fprintf(w, "Autoconnect on joining WiFi networks without encryption\t:\t%v\n", boolToStr(wifiSettings.CanApplyInBackground && wifiSettings.ConnectVPNOnInsecureNetwork))
	}
	fmt.Fprintf(w, "Trusted/Untrusted WiFi network control\t:\t%v\n", boolToStr(wifiSettings.CanApplyInBackground && wifiSettings.TrustedNetworksControl))
	fmt.Fprintf(w, "Default trust status for undefined networks\t:\t%v\n", boolToStrEx(wifiSettings.DefaultTrustStatusTrusted, "Trusted", "Untrusted", "No status"))
	fmt.Fprintf(w, "Actions:\t\n")
	fmt.Fprintf(w, "    Actions for Untrusted WiFi:\t\n")
	fmt.Fprintf(w, "        Connect to VPN\t:\t%v\n", boolToStr(wifiSettings.Actions.UnTrustedConnectVpn))
	fmt.Fprintf(w, "        Enable firewall\t:\t%v\n", boolToStr(wifiSettings.Actions.UnTrustedEnableFirewall))
	fmt.Fprintf(w, "        Block LAN traffic\t:\t%v\n", boolToStr(wifiSettings.Actions.UnTrustedBlockLan))
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
