//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2022 IVPN Limited.
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
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/ivpn/desktop-app/cli/flags"
	"github.com/ivpn/desktop-app/cli/helpers"
	"github.com/ivpn/desktop-app/daemon/protocol/eaa"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"golang.org/x/term"
)

type CmdParanoidMode struct {
	flags.CmdInfo
	status  bool
	disable bool
	enable  bool
}

func (c *CmdParanoidMode) Init() {
	c.Initialize("eaa", "Enhanced App Authentication\nEAA implements an additional authentication factor between the IVPN app (UI)\nand the daemon that manages the VPN tunnel. This prevents a malicious app\nfrom being able to manipulate the VPN tunnel without the users permission.\nWhen EAA is active the EAA password will be required to execute a command.")
	c.BoolVar(&c.status, "status", false, "(default) Show current EAA status")
	c.BoolVar(&c.disable, "off", false, "Disable EAA")
	c.BoolVar(&c.enable, "on", false, "Enable EAA and configure password")
}

func (c *CmdParanoidMode) Run() error {
	if c.disable && c.enable {
		return flags.BadParameter{}
	}

	if c.disable {
		if _proto.GetHelloResponse().ParanoidMode.IsEnabled {
			fmt.Println("Disabling Enhanced App Authentication")

			if helpers.CheckIsAdmin() {
				// We are running in privilaged environment
				// Trying to remove ParanoidMode file manually
				// (we are in privilaged mode - so old PM password is not required)

				// 1 - get path of PM file
				fpath := platform.ParanoidModeSecretFile()
				if len(fpath) <= 0 {
					return fmt.Errorf("failed to disable Enhanced App Authentication in privilaged user environment (file path not defined)")
				}

				// 2 - remove file
				eaa := eaa.Init(fpath)
				if err := eaa.ForceDisable(); err != nil {
					return err
				}

				// 3 - request new PM state (to print actual state for user)
				// and notify all connected clients about EAA change
				isSendResponseToAllClients := true
				if _, err := _proto.SendHelloEx(isSendResponseToAllClients); err != nil {
					return err
				}
			} else {

				fmt.Print("\tEnter current EAA password: ")
				data, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return fmt.Errorf("failed to read password: %w", err)
				}
				oldSecret := strings.TrimSpace(string(data))
				_proto.InitSetParanoidModeSecret(oldSecret)
				fmt.Println("")

				if err := _proto.SetParanoidModePassword(""); err != nil {
					return err
				}
			}
		}
	}

	if c.enable && !_proto.GetHelloResponse().ParanoidMode.IsEnabled {
		fmt.Print("Enabling Enhanced App Authentication\n\n")

		daemonSettings := _proto.GetHelloResponse().DaemonSettings
		if daemonSettings.IsAutoconnectOnLaunch && _proto.GetHelloResponse().DaemonSettings.IsAutoconnectOnLaunchDaemon {
			fmt.Print("Warning! 'Autoconnect on daemon launch' will not be applied\n\n")
		}
		if daemonSettings.WiFi.CanApplyInBackground {
			if daemonSettings.WiFi.TrustedNetworksControl {
				fmt.Print("Warning! 'Trusted WiFi' will not be applied\n         (until the EAA password is entered in Graphical User Interface application)\n\n")
			}
			if daemonSettings.WiFi.ConnectVPNOnInsecureNetwork {
				fmt.Print("Warning! 'Autoconnect on joining WiFi networks without encryption' will not be applied\n         (until the EAA password is entered in Graphical User Interface application)\n\n")
			}
		}

		fmt.Print("\tEnter new password: ")
		data, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		newSecret1 := strings.TrimSpace(string(data))
		fmt.Println("")

		fmt.Print("\tConfirm password: ")
		data, err = term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		newSecret2 := strings.TrimSpace(string(data))
		fmt.Println("")

		if newSecret1 != newSecret2 {
			return fmt.Errorf("passwords do not match")
		}

		if len(newSecret1) == 0 {
			return fmt.Errorf("password not defined")
		}

		if err := _proto.SetParanoidModePassword(newSecret1); err != nil {
			return err
		}
	}

	// print state
	var w *tabwriter.Writer
	w = printParanoidModeState(w, _proto.GetHelloResponse())
	w.Flush()

	if _proto.GetHelloResponse().ParanoidMode.IsEnabled {
		PrintTips([]TipType{TipEaaDisable})
	}

	return nil
}
