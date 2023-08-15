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
	"path/filepath"
	"runtime"
	"text/tabwriter"
)

type TipType uint

const (
	TipHelp                      TipType = iota
	TipHelpFull                  TipType = iota
	TipHelpCommand               TipType = iota
	TipLogout                    TipType = iota
	TipLogin                     TipType = iota
	TipForceLogin                TipType = iota
	TipServers                   TipType = iota
	TipConnectHelp               TipType = iota
	TipDisconnect                TipType = iota
	TipFirewallDisable           TipType = iota
	TipFirewallEnable            TipType = iota
	TipFirewallDisablePersistent TipType = iota
	TipSplittunEnable            TipType = iota
	TipEaaDisable                TipType = iota
	TipWiFiStatus                TipType = iota
	TipWiFiHelp                  TipType = iota
	TipAutoconnectHelp           TipType = iota
)

func PrintTips(tips []TipType) {
	if len(tips) == 0 {
		return
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	fmt.Println("")
	fmt.Fprintln(writer, "Tips:")
	for _, t := range tips {
		PrintTip(writer, t)
	}

	writer.Flush()
	fmt.Println("")
}

func PrintTip(w *tabwriter.Writer, tip TipType) {

	str := ""
	switch tip {
	case TipHelp:
		str = newTip("-h", "Show all commands")
	case TipHelpFull:
		str = newTip("-h -full", "Show detailed description about all commands")
	case TipHelpCommand:
		str = newTip("COMMAND -h", "Show detailed description of command")
	case TipLogout:
		str = newTip("logout", "Logout from this device")
	case TipLogin:
		str = newTip("login ACCOUNT_ID", "Log in with your Account ID")
	case TipForceLogin:
		str = newTip("login -force ACCOUNT_ID", "Log in with your Account ID and logout from all other devices")
	case TipServers:
		str = newTip("servers", "Show servers list")
	case TipConnectHelp:
		str = newTip("connect -h", "Show usage of 'connect' command")
	case TipDisconnect:
		str = newTip("disconnect", "Stop current VPN connection")
	case TipFirewallDisable:
		str = newTip("firewall -off", "Disable firewall (to allow connectivity outside VPN)")
	case TipFirewallEnable:
		str = newTip("firewall -on", "Enable firewall (to block all connectivity outside VPN)")
	case TipFirewallDisablePersistent:
		str = newTip("firewall -persistent_off", "Disable firewall persistency (Always-on firewall)")
	case TipSplittunEnable:
		str = newTip("splittun -on", "Enable Split Tunnel functionality")
	case TipEaaDisable:
		description := "Disable Enhanced App Authentication (use 'sudo ...' if you forgot your current EAA password)"
		if runtime.GOOS == "windows" {
			description = "Disable Enhanced App Authentication (start command with 'Run as Administrator' if you forgot your current EAA password)"
		}
		str = newTip("eaa -off", description)
	case TipWiFiStatus:
		str = newTip("wifi -status", "Show WiFi settings")
	case TipWiFiHelp:
		str = newTip("wifi -h", "Show usage of 'wifi' command")
	case TipAutoconnectHelp:
		str = newTip("autoconnect -h", "Show usage of 'autoconnect' command")
	}

	if len(str) > 0 {
		fmt.Fprintln(w, str)
	}
}

func newTip(command string, description string) string {
	return fmt.Sprintf("\t%s %s\t        %s", filepath.Base(os.Args[0]), command, description)
}
