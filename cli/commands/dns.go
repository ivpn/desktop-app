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
	"strings"
	"text/tabwriter"

	"github.com/ivpn/desktop-app/cli/cliplatform"
	"github.com/ivpn/desktop-app/cli/flags"
	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	service_types "github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

type CmdDns struct {
	flags.CmdInfo
	reset                bool
	dns                  string
	dohTemplate          string
	dotTemplate          string
	linuxManagementStyle string // LinuxDnsMgmt
}

type LinuxDnsMgmt string

const (
	LinuxDnsMgmt_Auto       = "auto"
	LinuxDnsMgmt_Resolvconf = "resolvconf"
)
const (
	ArgName_Off        = "off"
	ArgName_DoH        = "doh"
	ArgName_DoT        = "dot"
	ArgName_Management = "management"
)

func IsParamApplicable_LinuxForceModifyResolvconf() (bool, error) {
	// "force_use_resolvconf" is applicable only for linux AND only if both types of DNS management can be applied
	if runtime.GOOS != "linux" {
		return false, fmt.Errorf(fmt.Sprintf("functionality not applicable for %s", runtime.GOOS))
	}

	if _proto != nil {
		hr := _proto.GetHelloResponse()

		if len(hr.DisabledFunctions.Platform.Linux.DnsMgmtOldResolvconfError) > 0 {
			return false, fmt.Errorf(hr.DisabledFunctions.Platform.Linux.DnsMgmtOldResolvconfError)
		}

		if len(hr.DisabledFunctions.Platform.Linux.DnsMgmtNewResolvectlError) > 0 {
			return false, fmt.Errorf(hr.DisabledFunctions.Platform.Linux.DnsMgmtNewResolvectlError)
		}
	}

	return true, nil
}

func (c *CmdDns) Init() {
	c.Initialize("dns", "DNS management for VPN connection\nDNS_IP - optional parameter used to set custom dns value (ignored when AntiTracker enabled)")
	c.DefaultStringVar(&c.dns, "DNS_IP")
	c.BoolVar(&c.reset, ArgName_Off, false, "Reset DNS server to a default")

	if cliplatform.IsDnsOverHttpsSupported() {
		c.StringVar(&c.dohTemplate, ArgName_DoH, "", "URI", "DNS-over-HTTPS URI template\n  Example: ivpn dns -doh https://cloudflare-dns.com/dns-query 1.1.1.1")
	}
	if cliplatform.IsDnsOverTlsSupported() {
		c.StringVar(&c.dotTemplate, ArgName_DoT, "", "URI", "DNS-over-TLS URI template")
	}

	// "force_use_resolvconf" is applicable only for linux AND only if both types of DNS management can be applied
	if runtime.GOOS == "linux" {
		c.StringVarEx(&c.linuxManagementStyle, ArgName_Management, "", "METHOD",
			fmt.Sprintf(`By default IVPN manages DNS resolvers using the 'systemd-resolved' daemon 
		which is the correct method for systems based on Systemd. 
		This option enables you to override this behavior and allow the IVPN app 
		to directly modify the '/etc/resolv.conf' file. 		
		Note: This option is not applicable if there is only one DNS management method supported by the system.
		Possible values: %s (default); %s
			Example: 
				'ivpn dns -management=%s' 
				'ivpn dns -management=%s'`,
				LinuxDnsMgmt_Auto, LinuxDnsMgmt_Resolvconf, LinuxDnsMgmt_Resolvconf, LinuxDnsMgmt_Auto),
			func() bool {
				ret, _ := IsParamApplicable_LinuxForceModifyResolvconf()
				return ret
			})
	}
}

func (c *CmdDns) Run() error {
	if c.reset && len(c.dns) > 0 {
		return flags.BadParameter{}
	}

	if len(c.dohTemplate) > 0 && len(c.dotTemplate) > 0 {
		return flags.BadParameter{}
	}

	hr := _proto.GetHelloResponse()
	uPrefs := hr.DaemonSettings.UserPrefs

	if len(c.linuxManagementStyle) > 0 {
		if ret, err := IsParamApplicable_LinuxForceModifyResolvconf(); !ret {
			return flags.BadParameter{Message: fmt.Sprintf("Option '-%s' is not applicable for current environment: %v", ArgName_Management, err)}
		}

		val := strings.TrimSpace(strings.ToLower(c.linuxManagementStyle))
		if val != LinuxDnsMgmt_Auto && val != LinuxDnsMgmt_Resolvconf {
			return flags.BadParameter{}
		}
		isForceResolvconf := val == LinuxDnsMgmt_Resolvconf
		if uPrefs.Linux.IsDnsMgmtOldStyle != isForceResolvconf {
			if isForceResolvconf {
				fmt.Print("Applying configuration: force the IVPN app to directly modify the '/etc/resolv.conf' file (when VPN connected)...\n\n")
			} else {
				fmt.Print("Applying configuration: use default DNS configuration management style (when VPN connected)...\n\n")
			}
			uPrefs.Linux.IsDnsMgmtOldStyle = isForceResolvconf
			if err := _proto.SetUserPreferences(uPrefs); err != nil {
				return err
			}

			// trigger daemon to send HelloResponse with updated user preferences (will be in use for 'printDNSConfigInfo()')
			if _, err := _proto.SendHello(); err != nil {
				return err
			}
		}
	}

	var servers *apitypes.ServersInfoResponse
	// do we have to change custom DNS configuration ?
	if c.reset || len(c.dns) > 0 {
		// get default connection parameters (dns, anti-tracker, ... etc.)
		defConnCfg, err := _proto.GetDefConnectionParams()
		if err != nil {
			return err
		}
		defManualDns := defConnCfg.Params.ManualDNS

		if c.reset {
			defManualDns = dns.DnsSettings{}
		} else {
			defManualDns.DnsHost = c.dns
			if len(c.dohTemplate) > 0 {
				defManualDns.Encryption = dns.EncryptionDnsOverHttps
				defManualDns.DohTemplate = c.dohTemplate
			}
			if len(c.dotTemplate) > 0 {
				defManualDns.Encryption = dns.EncryptionDnsOverTls
				defManualDns.DohTemplate = c.dotTemplate
			}
		}

		if err := _proto.SetManualDNS(defManualDns, service_types.AntiTrackerMetadata{}); err != nil {
			return err
		}
	}

	// print state
	var w *tabwriter.Writer

	state, connected, err := _proto.GetVPNState()
	if err != nil {
		return err
	}

	if state == vpn.CONNECTED {
		if servers == nil {
			svrs, _ := _proto.GetServers()
			servers = &svrs
		}
		w = printDNSState(w, connected.Dns, servers)
	} else {
		defConnCfg, err := _proto.GetDefConnectionParams()
		if err != nil {
			return err
		}
		w = printDNSConfigInfo(w, defConnCfg.Params.ManualDNS)
	}
	w.Flush()

	return nil
}

//----------------------------------------------------------------------------------------

type CmdAntitracker struct {
	flags.CmdInfo
	on         string
	off        bool
	hardcore   string
	blocklists bool
}

const EmptyBlockListName = "<default>"

func (c *CmdAntitracker) Init() {
	c.SetPreParseFunc(c.preParse)

	cmdName := "antitracker"
	argNameShowBlocklists := "show_blocklists"

	c.Initialize(cmdName, "Default AntiTracker configuration management for VPN connection")

	tipText := fmt.Sprintf("Tip: use 'ivpn %s -%s' command to show all supported DNS block lists", cmdName, argNameShowBlocklists)

	c.StringVar(&c.on, "on", "", "[BLOCK_LIST]", "Enable AntiTracker\n BLOCK_LIST - optional parameter used to set custom DNS block list\n "+tipText)
	c.StringVar(&c.hardcore, "on_hardcore", "", "[BLOCK_LIST]", "Enable AntiTracker 'hardcore' mode\n BLOCK_LIST - optional parameter used to set custom DNS block list\n "+tipText)
	c.BoolVar(&c.off, "off", false, "Disable AntiTracker")
	c.BoolVar(&c.blocklists, argNameShowBlocklists, false, "Show all supported DNS block lists")
}

func (c *CmdAntitracker) preParse(arguments []string) ([]string, error) {
	isArgRemoved := false
	arguments, isArgRemoved = flags.RemoveArgIfNoValue(arguments, "-on")
	if isArgRemoved {
		c.on = EmptyBlockListName
	}
	arguments, isArgRemoved = flags.RemoveArgIfNoValue(arguments, "-on_hardcore")
	if isArgRemoved {
		c.hardcore = EmptyBlockListName
	}
	return arguments, nil
}

func (c *CmdAntitracker) Run() error {
	if c.NFlag() > 1 {
		return flags.BadParameter{Message: "Not allowed to use more than one argument for this command"}
	}

	if c.blocklists {
		svrs, err := _proto.GetServers()
		if err != nil {
			return err
		}

		return printBlockLists(svrs.Config.AntiTrackerPlus.DnsServers)
	}

	// do we have to change anti-tracker configuration ?
	if c.off || c.on != "" || c.hardcore != "" {
		// get default connection parameters (dns, anti-tracker, ... etc.)
		defConnCfg := service_types.ConnectionParams{}
		if ret, err := _proto.GetDefConnectionParams(); err == nil {
			defConnCfg = ret.Params
		}
		newAtMetadata := defConnCfg.Metadata.AntiTracker

		newAtMetadata.Enabled = false
		newAtMetadata.Hardcore = false

		if c.hardcore != "" {
			newAtMetadata.Hardcore = true
			newAtMetadata.Enabled = true
			if c.hardcore != EmptyBlockListName {
				newAtMetadata.AntiTrackerBlockListName = c.hardcore
			}
		} else if c.on != "" {
			newAtMetadata.Enabled = true
			if c.on != EmptyBlockListName {
				newAtMetadata.AntiTrackerBlockListName = c.on
			}
		}

		if err := _proto.SetManualDNS(defConnCfg.ManualDNS, newAtMetadata); err != nil {
			return err
		}
	}

	// print state
	var w *tabwriter.Writer

	state, connected, err := _proto.GetVPNState()
	if err != nil {
		return err
	}
	defConnCfg, err := _proto.GetDefConnectionParams()
	if err != nil {
		return err
	}

	if state == vpn.CONNECTED {
		servers, _ := _proto.GetServers()
		w = printDNSState(w, connected.Dns, &servers)
	} else {
		w = printAntitrackerConfigInfo(w, defConnCfg.Params.Metadata.AntiTracker)
	}
	w.Flush()

	return nil
}

//----------------------------------------------------------------------------------------

func printDNSConfigInfo(w *tabwriter.Writer, customDNS dns.DnsSettings) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	if !customDNS.IsEmpty() {
		fmt.Fprintf(w, "Default config\t:\tCustom DNS %v\n", customDNS.InfoString())
	} else {
		fmt.Fprintf(w, "Default config\t:\tCustom DNS not defined\n")
	}

	if ret, _ := IsParamApplicable_LinuxForceModifyResolvconf(); ret && _proto != nil {
		hr := _proto.GetHelloResponse()
		if hr.DaemonSettings.UserPrefs.Linux.IsDnsMgmtOldStyle {
			fmt.Fprintf(w, "Management method\t:\tForce to modify the '/etc/resolv.conf' file\n")
		}
	}

	return w
}

func printAntitrackerConfigInfo(w *tabwriter.Writer, antitracker service_types.AntiTrackerMetadata) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}
	fmt.Fprintf(w, "Default config\t:\tAntiTracker %s\n", GetAntiTrackerStatusText(antitracker))
	return w
}

// ----------------------------------------------------------------------------------------
func GetAntiTrackerStatusText(antitracker service_types.AntiTrackerMetadata) string {
	if !antitracker.Enabled {
		return "Disabled"
	}

	infoText := ""
	if antitracker.Hardcore {
		infoText = "Hardcore"
	}
	if antitracker.AntiTrackerBlockListName != "" {
		if len(infoText) > 0 {
			infoText += "; "
		}
		infoText += fmt.Sprintf("block list: '%s'", antitracker.AntiTrackerBlockListName)
	}
	if infoText != "" {
		infoText = fmt.Sprintf(" (%s)", infoText)
	}

	return fmt.Sprintf("Enabled%s", infoText)
}

// GetAntitrackerIP - returns IP of antitracker DNS
func GetAntitrackerIP(vpntype vpn.Type, isHardcore, isMultihop bool, servers *apitypes.ServersInfoResponse) (string, error) {
	if isHardcore {
		return servers.Config.Antitracker.Hardcore.IP, nil
	}

	return servers.Config.Antitracker.Default.IP, nil
}

/*
func isAcceptableDnsBlockListName(atDnsServers []apitypes.AntiTrackerPlusServer) error {
	if len(atDnsServers) == 0 {
		fmt.Println("No DNS block lists available")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "|\tDNS BLOCK LIST NAME\t|\tDETAILS\t|\n")
	fmt.Fprintf(w, "|\t\t|\t\t|\n")
	for _, bl := range atDnsServers {
		fmt.Fprintf(w, "|\t%s\t|\t%s\t|\n", bl.Name, bl.Description)
	}

	w.Flush()

	return nil
}*/

func printBlockLists(atDnsServers []apitypes.AntiTrackerPlusServer) error {
	if len(atDnsServers) == 0 {
		fmt.Println("No DNS block lists available")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Fprintf(w, "|\tDNS BLOCK LIST NAME\t|\tDETAILS\t|\n")
	fmt.Fprintf(w, "|\t\t|\t\t|\n")
	for _, bl := range atDnsServers {
		fmt.Fprintf(w, "|\t%s\t|\t%s\t|\n", bl.Name, bl.Description)
	}

	w.Flush()

	return nil
}
