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
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ivpn/desktop-app/cli/cliplatform"
	"github.com/ivpn/desktop-app/cli/protocol"
	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/splittun"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

var _proto *protocol.Client

// Initialize initializes commands. Must be called before using any command.
func Initialize(proto *protocol.Client) {
	_proto = proto
}

func printAccountInfo(w *tabwriter.Writer, accountID string) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	if len(accountID) > 0 {
		return w // Do nothing in case of logged in
	}

	fmt.Fprintf(w, "Account\t:\t%v", "Not logged in\n")

	return w
}

func printState(w *tabwriter.Writer, state vpn.State, connected types.ConnectedResp, serverInfo string, exitServerInfo string) *tabwriter.Writer {

	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	fmt.Fprintf(w, "VPN\t:\t%v\n", state)

	if len(serverInfo) > 0 {
		fmt.Fprintf(w, "\t\t%v\n", serverInfo)
		if len(exitServerInfo) > 0 {
			fmt.Fprintf(w, "\t\t%v (Multi-Hop exit server)\n", exitServerInfo)
		}
	}

	if state != vpn.CONNECTED {
		return w
	}
	since := time.Unix(connected.TimeSecFrom1970, 0)
	fmt.Fprintf(w, "    Protocol\t:\t%v\n", connected.VpnType)
	fmt.Fprintf(w, "    Local IP\t:\t%v\n", connected.ClientIP)
	if len(connected.ClientIPv6) > 0 {
		fmt.Fprintf(w, "    Local IPv6\t:\t%v\n", connected.ClientIPv6)
	}

	portInfo := ""
	if connected.ServerPort > 0 {
		if connected.IsTCP {
			portInfo += " (TCP:"
		} else {
			portInfo += " (UDP:"
		}
		portInfo += fmt.Sprintf(":%d)", connected.ServerPort)
	}
	fmt.Fprintf(w, "    Server IP\t:\t%v%v\n", connected.ServerIP, portInfo)

	fmt.Fprintf(w, "    Connected\t:\t%v\n", since)

	return w
}

func printDNSState(w *tabwriter.Writer, dnsCfg dns.DnsSettings, servers *apitypes.ServersInfoResponse) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	if dnsCfg.IsEmpty() {
		fmt.Fprintf(w, "DNS\t:\tDefault (auto)\n")
		return w
	}

	antitrackerText := strings.Builder{}

	if dnsCfg.Encryption == dns.EncryptionNone {
		isAntitracker, isAtHardcore := IsAntiTrackerIP(dnsCfg.DnsHost, servers)
		if isAtHardcore {
			antitrackerText.WriteString("Enabled (Hardcore)")
		} else if isAntitracker {
			antitrackerText.WriteString("Enabled")
		}
	}

	if antitrackerText.Len() > 0 {
		fmt.Fprintf(w, "AntiTracker\t:\t%v\n", antitrackerText.String())
	} else {
		fmt.Fprintf(w, "DNS\t:\t%v\n", dnsCfg.InfoString())
	}

	return w
}

func printFirewallState(w *tabwriter.Writer, isEnabled, isPersistent, isAllowLAN, isAllowMulticast, isAllowApiServers bool, userExceptions string) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	fwState := "Disabled"
	if isEnabled {
		fwState = "Enabled"
	}

	fmt.Fprintf(w, "Firewall\t:\t%v\n", fwState)
	fmt.Fprintf(w, "    Allow LAN\t:\t%v\n", isAllowLAN)
	if isPersistent {
		fmt.Fprintf(w, "    Persistent\t:\t%v\n", isPersistent)
	}
	fmt.Fprintf(w, "    Allow IVPN servers\t:\t%v\n", isAllowApiServers)
	if len(userExceptions) > 0 {
		fmt.Fprintf(w, "    Allow IP masks\t:\t%v\n", userExceptions)
	}

	return w
}

func printSplitTunState(w *tabwriter.Writer, isShortPrint bool, isFullPrint bool, isEnabled bool, apps []string, runningApps []splittun.RunningApp) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	if !cliplatform.IsSplitTunSupported() {
		return w
	}

	state := "Disabled"
	if isEnabled {
		state = "Enabled"
	}

	fmt.Fprintf(w, "Split Tunnel\t:\t%v\n", state)

	if !isShortPrint {
		for i, path := range apps {
			if i == 0 {
				fmt.Fprintf(w, "Split Tunnel apps\t:\t%v\n", path)
			} else {
				fmt.Fprintf(w, "\t\t%v\n", path)
			}
		}

		sort.Slice(runningApps, func(i, j int) bool {
			return runningApps[i].Pid < runningApps[j].Pid
		})

		isFirstLineShown := false
		for _, exec := range runningApps {
			if exec.Pid != exec.ExtIvpnRootPid {
				continue
			}

			cmd := exec.ExtModifiedCmdLine
			if len(cmd) <= 0 {
				cmd = exec.Cmdline
			}
			if !isFirstLineShown {
				isFirstLineShown = true
				fmt.Fprintf(w, "Running commands\t:\t[pid:%d] %s\n", exec.Pid, cmd)
			} else {
				fmt.Fprintf(w, "\t\t[pid:%d] %s\n", exec.Pid, cmd)
			}
		}

		if isFullPrint {
			regexpBinaryArgs := regexp.MustCompile("(\".*\"|\\S*)(.*)")
			funcTruncateCmdStr := func(cmd string, maxLenSoftLimit int) string {
				cols := regexpBinaryArgs.FindStringSubmatch(cmd)
				if len(cols) != 3 {
					return cmd
				}
				ret := cols[1] // bin

				args := cmd[len(ret):]
				if len(ret) < maxLenSoftLimit && len(args) > 0 {
					ret += " " + args
					if len(ret) > maxLenSoftLimit {
						ret = ret[:maxLenSoftLimit] + "..."
					}
				}
				return ret
				//cols := regexpBinaryArgs.FindStringSubmatch(cmd)
				//if len(cols) != 3 {
				//	return cmd
				//}
				//ret := cols[1] // bin
				//args := strings.Split(cols[2], " ")
				//for _, arg := range args {
				//	if len(arg) <= 0 {
				//		continue
				//	}
				//	if len(ret)+len(arg) <= maxLenSoftLimit {
				//		ret += " " + arg
				//	} else {
				//		ret += "..."
				//		break
				//	}
				//}
				//return ret
			}

			if len(runningApps) > 0 {
				fmt.Fprintf(w, "All running processes\t:\t\n")

				for _, exec := range runningApps {
					detachedProcWarning := ""
					if exec.ExtIvpnRootPid <= 0 {
						detachedProcWarning = "*"
					}

					fmt.Fprintf(w, "  [pid:%d ppid:%d exe:%s]%s %s\n", exec.Pid, exec.Ppid, exec.Exe, detachedProcWarning, funcTruncateCmdStr(exec.Cmdline, 60))
				}
			}
		}
	}

	return w
}

func printParamoidModeState(w *tabwriter.Writer, helloResp types.HelloResp) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	pModeStatusText := "Disabled"
	if helloResp.ParanoidMode.IsEnabled {
		pModeStatusText = "Enabled"
	}
	fmt.Fprintf(w, "EAA\t:\t%s\n", pModeStatusText)

	return w
}
