//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2022 Privatus Limited.
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

	"github.com/ivpn/desktop-app/cli/flags"
	"github.com/ivpn/desktop-app/cli/helpers"
	"golang.org/x/crypto/ssh/terminal"
)

type CmdParanoidMode struct {
	flags.CmdInfo
	status  bool
	disable bool
	enable  bool
}

func (c *CmdParanoidMode) Init() {
	c.Initialize("pmode", "Paranoid Mode management\nWhen Paranoid Mode is enabled - the password will be requested to execute each command")
	c.BoolVar(&c.status, "status", false, "(default) Show current Paramoid Mode status")
	c.BoolVar(&c.disable, "off", false, "Disable Paranoid Mode")
	c.BoolVar(&c.enable, "on", false, "Enable Paranoid Mode and set password")
}

func (c *CmdParanoidMode) Run() error {
	if c.disable && c.enable {
		return flags.BadParameter{}
	}

	if c.disable {
		if _proto.GetHelloResponse().ParanoidModeIsEnabled {
			if helpers.CheckIsAdmin() {
				// We are running in privilaged environment
				// Trying to remove ParanoidMode file manually
				// (we are in privilaged mode - so old PM password is not required)

				// 1 - get path of PM file
				resp, err := _proto.SendHelloEx(true)
				if err != nil {
					return err
				}
				if len(resp.ParanoidModeFilePath) <= 0 {
					return fmt.Errorf("failed to disable Paranoid Mode in privilaged user environment (file path not defined)")
				}

				// 2 - remove file
				if err := os.Remove(resp.ParanoidModeFilePath); err != nil {
					return err
				}

				// request new PM state (to print actual state for user)
				if _, err := _proto.SendHello(); err != nil {
					return err
				}
			} else {
				fmt.Println("Disabling Paranoid Mode")
				fmt.Print("\tEnter old password for Paranoid Mode : ")
				data, err := terminal.ReadPassword(0)
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

	if c.enable {
		fmt.Println("Enabling Paranoid Mode")

		if _proto.GetHelloResponse().ParanoidModeIsEnabled {
			fmt.Print("\tEnter old password for Paranoid Mode : ")
			data, err := terminal.ReadPassword(0)
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			oldSecret := strings.TrimSpace(string(data))
			_proto.InitSetParanoidModeSecret(oldSecret)
			fmt.Println("")
		}

		fmt.Print("\tEnter new password for Paranoid Mode : ")
		data, err := terminal.ReadPassword(0)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		newSecret1 := strings.TrimSpace(string(data))
		fmt.Println("")

		fmt.Print("\tRepeat new password for Paranoid Mode: ")
		data, err = terminal.ReadPassword(0)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		newSecret2 := strings.TrimSpace(string(data))
		fmt.Println("")

		if newSecret1 != newSecret2 {
			return fmt.Errorf("password confirmation error")
		}

		if err := _proto.SetParanoidModePassword(newSecret1); err != nil {
			return err
		}
	}

	// print state
	var w *tabwriter.Writer
	w = printParamoidModeState(w, _proto.GetHelloResponse())
	w.Flush()

	return nil
}
