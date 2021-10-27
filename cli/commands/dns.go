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
	"strings"
	"text/tabwriter"

	"github.com/ivpn/desktop-app/cli/commands/config"
	"github.com/ivpn/desktop-app/cli/flags"
	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

type CmdDns struct {
	flags.CmdInfo
	reset bool
	dns   string
}

func (c *CmdDns) Init() {
	c.Initialize("dns", "Default 'custom DNS' management for VPN connection\nDNS_IP - optional parameter used to set custom dns value (ignored when AntiTracker enabled)")
	c.DefaultStringVar(&c.dns, "DNS_IP")
	c.BoolVar(&c.reset, "off", false, "Reset DNS server to a default")
}

func (c *CmdDns) Run() error {
	if c.reset && len(c.dns) > 0 {
		return flags.BadParameter{}
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	var servers *apitypes.ServersInfoResponse
	// do we have to change custom DNS configuration ?
	if c.reset || len(c.dns) > 0 {
		cfg.CustomDNS = ""
		if len(c.dns) > 0 {
			cfg.CustomDNS = c.dns
		}

		err = config.SaveConfig(cfg)
		if err != nil {
			return err
		}

		// update DNS if VPN is connected
		state, connectedInfo, err := _proto.GetVPNState()
		if err != nil {
			return err
		}
		if state == vpn.CONNECTED {
			svrs, _ := _proto.GetServers()
			servers = &svrs
			isAntitracker, isAtHardcore := IsAntiTrackerIP(connectedInfo.ManualDNS, servers)
			if c.reset && (isAntitracker || isAtHardcore) {
				fmt.Println("Nothing to disable")
			} else {
				if err := _proto.SetManualDNS(cfg.CustomDNS); err != nil {
					return err
				}
				fmt.Println("Custom DNS successfully changed for current VPN connection")
			}
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
		w = printDNSState(w, connected.ManualDNS, servers)
	}

	w = printDNSConfigInfo(w, cfg.CustomDNS)
	w.Flush()

	return nil
}

//----------------------------------------------------------------------------------------

type CmdAntitracker struct {
	flags.CmdInfo
	def      bool
	off      bool
	hardcore bool
}

func (c *CmdAntitracker) Init() {
	c.Initialize("antitracker", "Default AntiTracker configuration management for VPN connection")
	c.BoolVar(&c.def, "on", false, "Enable AntiTracker")
	c.BoolVar(&c.hardcore, "on_hardcore", false, "Enable AntiTracker 'hardcore' mode")
	c.BoolVar(&c.off, "off", false, "Disable AntiTracker")
}

func (c *CmdAntitracker) Run() error {
	if c.NFlag() > 1 {
		return flags.BadParameter{Message: "Not allowed to use more than one argument for this command"}
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	var servers apitypes.ServersInfoResponse
	var dns string

	servers, err = _proto.GetServers()
	if err != nil {
		return err
	}

	// do we have to change antitracker configuration ?
	if c.off || c.def || c.hardcore {
		cfg.Antitracker = false
		cfg.AntitrackerHardcore = false

		if c.hardcore {
			cfg.AntitrackerHardcore = true
		} else if c.def {
			cfg.Antitracker = true
		}

		err = config.SaveConfig(cfg)
		if err != nil {
			return err
		}

		// update DNS if VPN is connected
		state, connectInfo, err := _proto.GetVPNState()
		if err != nil {
			return err
		}

		if state == vpn.CONNECTED {
			isAntitracker, isAtHardcore := IsAntiTrackerIP(connectInfo.ManualDNS, &servers)
			if c.off && !(isAntitracker || isAtHardcore) {
				fmt.Println("AntiTracker already disabled")
			} else {
				if cfg.Antitracker || cfg.AntitrackerHardcore {
					dns, err = GetAntitrackerIP(connectInfo.VpnType, cfg.AntitrackerHardcore, len(connectInfo.ExitServerID) > 0, &servers)
				}
				if err := _proto.SetManualDNS(dns); err != nil {
					return err
				}
				fmt.Println("AntiTracker successfully updated for current VPN connection")
			}
		}
	}

	// print state
	var w *tabwriter.Writer

	state, connected, err := _proto.GetVPNState()
	if err != nil {
		return err
	}

	if state == vpn.CONNECTED {
		servers, _ := _proto.GetServers()
		w = printDNSState(w, connected.ManualDNS, &servers)
	}

	w = printAntitrackerConfigInfo(w, cfg.Antitracker, cfg.AntitrackerHardcore)
	w.Flush()

	return nil
}

//----------------------------------------------------------------------------------------

func printDNSConfigInfo(w *tabwriter.Writer, customDNS string) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	if len(customDNS) > 0 {
		fmt.Fprintln(w, fmt.Sprintf("Default config\t:\tCustom DNS %v", customDNS))
	} else {
		fmt.Fprintln(w, fmt.Sprintf("Default config\t:\tCustom DNS not defined"))
	}

	return w
}

func printAntitrackerConfigInfo(w *tabwriter.Writer, antitracker, antitrackerHardcore bool) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	if antitrackerHardcore {
		fmt.Fprintln(w, fmt.Sprintf("Default config\t:\tAntiTracker Enabled (Hardcore)"))
	} else if antitracker {
		fmt.Fprintln(w, fmt.Sprintf("Default config\t:\tAntiTracker Enabled"))
	} else {
		fmt.Fprintln(w, fmt.Sprintf("Default config\t:\tAntiTracker Disabled"))
	}

	return w
}

//----------------------------------------------------------------------------------------

// GetAntitrackerIP - returns IP of antitracker DNS
func GetAntitrackerIP(vpntype vpn.Type, isHardcore, isMultihop bool, servers *apitypes.ServersInfoResponse) (string, error) {

	if isHardcore {
		if vpntype == vpn.OpenVPN && isMultihop {
			return servers.Config.Antitracker.Hardcore.MultihopIP, nil
		}
		return servers.Config.Antitracker.Hardcore.IP, nil
	}

	if vpntype == vpn.OpenVPN && isMultihop {
		return servers.Config.Antitracker.Default.MultihopIP, nil
	}
	return servers.Config.Antitracker.Default.IP, nil
}

// IsAntiTrackerIP returns info 'is this IP equals to antitracker IP'
func IsAntiTrackerIP(dns string, servers *apitypes.ServersInfoResponse) (antitracker, antitrackerHardcore bool) {
	if servers == nil {
		return false, false
	}

	dns = strings.TrimSpace(dns)
	if len(dns) == 0 {
		return false, false
	}

	if dns == servers.Config.Antitracker.Default.IP || dns == servers.Config.Antitracker.Default.MultihopIP {
		return true, false
	} else if dns == servers.Config.Antitracker.Hardcore.IP || dns == servers.Config.Antitracker.Hardcore.MultihopIP {
		return true, true
	}

	return false, false
}
