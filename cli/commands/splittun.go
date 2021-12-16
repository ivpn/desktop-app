//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2021 Privatus Limited.
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
	"os/exec"
	"strings"
	"syscall"

	"github.com/ivpn/desktop-app/cli/cliplatform"
	"github.com/ivpn/desktop-app/cli/flags"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
)

type Exclude struct {
	flags.CmdInfo
	execute              string // this parameter is not in use. We need it just for help info. (using executeSpecParseArgs after special parsing)
	executeSpecParseArgs []string
}

func (c *Exclude) Init() {
	// register special parse function (for -execute)
	c.SetParseSpecialFunc(c.specialParse)

	c.Initialize("exclude", "Run command in Split Tunnel environment\n(exclude it's traffic from the VPN tunnel)\nIt is short version of 'ivpn splittun -appadd <command>'\nExamples:\n    ivpn exclude firefox\n    ivpn exclude ping 1.1.1.1\n    ivpn exclude /usr/bin/google-chrome")
	c.DefaultStringVar(&c.execute, "COMMAND")
}
func (c *Exclude) Run() error {
	return doAddApp(c.executeSpecParseArgs)
}
func (c *Exclude) specialParse(arguments []string) bool {
	if strings.ToLower(arguments[0]) == "-h" {
		return false
	}
	c.executeSpecParseArgs = arguments
	return true
}

// ================================================================
func doAddApp(args []string) error {
	// Description of Split Tunneling commands sequence to run the application:
	//	[client]					          [daemon]
	//	SplitTunnelAddApp		    ->
	//							            <-	windows:	types.EmptyResp (success)
	//							            <-	linux:		types.SplitTunnelAddAppCmdResp (some operations required on client side)
	//	<windows: done>
	// 	<execute shell command: types.SplitTunnelAddAppCmdResp.CmdToExecute and get PID>
	//  SplitTunnelAddedPidInfo	->
	// 							            <-	types.EmptyResp (success)

	binary := args[0]

	binary, err := exec.LookPath(binary)
	if err != nil {
		return err
	}

	isRequiredToExecuteCommand, err := _proto.SplitTunnelAddApp(strings.Join(args[:], " "))
	if err != nil {
		return err
	}

	if !isRequiredToExecuteCommand {
		// (Windows) Success. No other operations required
		return nil
	}

	// Linux: the command have to be executed
	cfg, err := _proto.GetSplitTunnelStatus()
	if err != nil {
		return err
	}

	if !cfg.IsEnabled {
		fmt.Println("Split Tunneling not enabled")
		PrintTips([]TipType{TipSplittunEnable})
		return fmt.Errorf("unable to start command: Split Tunneling not enabled")
	}
	fmt.Printf("Running command in Split Tunneling environment (pid:%d): %v\n", os.Getpid(), strings.Trim(fmt.Sprint(args), "[]"))
	return syscall.Exec(binary, args, os.Environ())
}

// ================================================================
type SplitTun struct {
	flags.CmdInfo
	status bool
	on     bool
	off    bool
	reset  bool

	appremove  string
	appadd     string // this parameter is not in use. We need it just for help info (using 'appaddArgs' parsed with specific logic)
	appaddArgs []string
}

func (c *SplitTun) Init() {
	// register special parse function for '-appadd' (parsing appaddArgs)
	c.SetParseSpecialFunc(c.specialParse)

	c.Initialize("splittun", "Split Tunnel management\nBy enabling this feature you can exclude traffic for a specific applications from the VPN tunnel")
	c.BoolVar(&c.status, "status", false, "(default) Show Split Tunnel status and configuration")

	if !cliplatform.IsSplitTunRunsApp() {
		// Windows
		c.BoolVar(&c.reset, "clean", false, "Erase configuration (delete all applications from configuration and disable)")
		c.StringVar(&c.appadd, "appadd", "", "PATH", "Add application to configuration (use full path to binary)")
		c.StringVar(&c.appremove, "appremove", "", "PATH", "Delete application from configuration (use full path to binary)")
	} else {
		// Linux
		c.BoolVar(&c.reset, "clean", false, "Erase configuration (delete all applications from configuration and disable)")
		c.StringVar(&c.appadd, "appadd", "", "COMMAND", "Execute command (binary) in Split Tunnel environment (exclude it's traffic from the VPN tunnel)\nInfo: short version of this command is 'ivpn exclude <command>'\nExamples:\n    ivpn splittun -appadd firefox\n    ivpn splittun -appadd ping 1.1.1.1\n    ivpn splittun -appadd /usr/bin/google-chrome")
		c.StringVar(&c.appremove, "appremove", "", "PID", "Remove application from Split Tunnel environment\n(argument: Process ID)")
	}

	c.BoolVar(&c.on, "on", false, "Enable")
	c.BoolVar(&c.off, "off", false, "Disable")
}

func (c *SplitTun) Run() error {
	if c.on && c.off {
		return flags.BadParameter{}
	}
	if len(c.appadd) > 0 && len(c.appremove) > 0 {
		return flags.BadParameter{}
	}

	cfg, err := _proto.GetSplitTunnelStatus()
	if err != nil {
		return err
	}

	if c.reset {
		cfg.IsEnabled = false
		cfg.SplitTunnelApps = make([]string, 0)

		if err = _proto.SetSplitTunnelConfig(false, true); err != nil {
			return err
		}
		cfg, err = _proto.GetSplitTunnelStatus()
		if err != nil {
			return err
		}
		return c.doShowStatus(cfg)
	}

	if c.on || c.off {
		isEnabled := false
		if c.on {
			isEnabled = true
		}
		if err = _proto.SetSplitTunnelConfig(isEnabled, false); err != nil {
			return err
		}
		cfg, err = _proto.GetSplitTunnelStatus()
		if err != nil {
			return err
		}
		return c.doShowStatusShort(cfg)
	}

	if len(c.appaddArgs) > 0 || len(c.appremove) > 0 {
		if len(c.appaddArgs) > 0 {
			if err = doAddApp(c.appaddArgs); err != nil {
				return err
			}
		} else if len(c.appremove) > 0 {
			if err = _proto.SplitTunnelRemoveApp(c.appremove); err != nil {
				return err
			}
		}

		cfg, err = _proto.GetSplitTunnelStatus()
		if err != nil {
			return err
		}
	}

	return c.doShowStatus(cfg)
}

func (c *SplitTun) doShowStatus(cfg types.SplitTunnelStatus) error {
	w := printSplitTunState(nil, false, cfg.IsEnabled, cfg.SplitTunnelApps, cfg.RunningApps)
	w.Flush()
	return nil
}

func (c *SplitTun) doShowStatusShort(status types.SplitTunnelStatus) error {
	w := printSplitTunState(nil, true, status.IsEnabled, status.SplitTunnelApps, status.RunningApps)
	w.Flush()
	return nil
}

func (c *SplitTun) specialParse(arguments []string) bool {
	if len(arguments) > 1 && strings.ToLower(arguments[0]) == "-appadd" {
		c.appaddArgs = arguments[1:]
		return true
	}
	return false
}
