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
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/ivpn/desktop-app/cli/cliplatform"
	"github.com/ivpn/desktop-app/cli/flags"
	"github.com/ivpn/desktop-app/cli/helpers"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
)

type Exclude struct {
	flags.CmdInfo
	execute              string // this parameter is not in use. We need it just for help info. (using executeSpecParseArgs after special parsing)
	executeSpecParseArgs []string
	eaaPassword          string // (if Enhanced App Authentication is enabled) EAA password (using executeSpecParseArgs after special parsing)
	eaaPasswordHash      string // (if Enhanced App Authentication is enabled) EAA password hash (using executeSpecParseArgs after special parsing)
}

func (c *Exclude) Init() {
	// register special parse function (for -execute)
	c.SetParseSpecialFunc(c.specialParse)

	c.Initialize("exclude", "Run command in Split Tunnel environment\n(exclude it's traffic from the VPN tunnel)\nIt is short version of 'ivpn splittun -appadd <command>'\nExamples:\n    ivpn exclude firefox\n    ivpn exclude ping 1.1.1.1\n    ivpn exclude /usr/bin/google-chrome")
	c.DefaultStringVar(&c.execute, "COMMAND")
	c.StringVar(&c.eaaPassword, "eaa", "", "PASSWORD", "(optional) Enhanced App Authentication password\nPlease, refer to 'eaa' command for details ('ivpn eaa -h')\nExample:\n    ivpn exclude -eaa 'my_password' firefox")

}
func (c *Exclude) Run() error {
	if len(c.executeSpecParseArgs) <= 0 {
		c.Usage(false)
		return fmt.Errorf("no parameters defined")
	}

	if len(c.eaaPasswordHash) > 0 {
		return doAddApp(c.executeSpecParseArgs, c.eaaPasswordHash, true)
	}
	return doAddApp(c.executeSpecParseArgs, c.eaaPassword, false)
}
func (c *Exclude) specialParse(arguments []string) bool {
	if len(arguments) <= 0 {
		return false
	}

	if strings.ToLower(arguments[0]) == "-h" {
		return false
	}

	if strings.ToLower(arguments[0]) == "-eaa_hash" {
		c.eaaPasswordHash = arguments[1] // base64 hash of EAA password
		arguments = arguments[2:]
	} else if strings.ToLower(arguments[0]) == "-eaa" {
		if len(arguments) < 3 {
			return false
		}
		c.eaaPassword = arguments[1]
		if (strings.HasPrefix(c.eaaPassword, "'") && strings.HasSuffix(c.eaaPassword, "'")) ||
			(strings.HasPrefix(c.eaaPassword, "\"") && strings.HasSuffix(c.eaaPassword, "\"")) {
			c.eaaPassword = c.eaaPassword[1 : len(c.eaaPassword)-1]
		}
		arguments = arguments[2:]
	}
	c.executeSpecParseArgs = arguments
	return true
}

// ================================================================
func doAddApp(args []string, eaaPass string, isHashedPass bool) error {
	if len(args) <= 0 {
		return fmt.Errorf("no arguments defined")
	}
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

	if len(eaaPass) > 0 {
		if isHashedPass {
			_proto.InitSetParanoidModeSecretHash(eaaPass)
		} else {
			_proto.InitSetParanoidModeSecret(eaaPass)
		}
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

	if cfg.IsFunctionalityNotAvailable {
		return fmt.Errorf("the Split Tunneling functionality not available")
	}

	if !cfg.IsEnabled {
		fmt.Println("Split Tunneling not enabled")
		PrintTips([]TipType{TipSplittunEnable})
		return fmt.Errorf("unable to start command: Split Tunneling not enabled")
	}

	pid := os.Getpid()
	// Set unique environment var for the process.
	// All child processes will use the same var. It will help us to distinguish processes which belongs to specific command
	if err := os.Setenv("IVPN_STARTED_ST_ID", strconv.Itoa(pid)); err != nil {
		return fmt.Errorf("failed to start command (unable to set environment variable): %w", err)
	}
	fmt.Printf("Running command in Split Tunneling environment (pid:%d): %v\n", os.Getpid(), strings.Trim(fmt.Sprint(args), "[]"))
	return syscall.Exec(binary, args, os.Environ())
}

// ================================================================
type SplitTun struct {
	flags.CmdInfo
	status      bool
	statusFull  bool
	on          bool
	onInverse   bool
	dnsFirewall string // [on/off]
	off         bool
	reset       bool

	appremove  string
	appadd     string // this parameter is not in use. We need it just for help info (using 'appaddArgs' parsed with specific logic)
	appaddArgs []string
}

const (
	cmd_name_on_inverse   = "on_inverse"
	cmd_name_dns_firewall = "block_dns"
)

func (c *SplitTun) Init() {
	c.KeepArgsOrderInHelp = true
	// register special parse function for '-appadd' (parsing appaddArgs)
	c.SetParseSpecialFunc(c.specialParse)

	c.Initialize("splittun", "Split Tunnel management\nBy enabling this feature you can exclude traffic from specific applications from the VPN tunnel")
	c.BoolVar(&c.status, "status", false, "(default) Show Split Tunnel status and configuration")

	if !cliplatform.IsSplitTunRunsApp() {
		// Windows
		c.BoolVar(&c.reset, "clean", false, "Erase configuration (delete all applications from configuration and disable)")
		c.StringVar(&c.appadd, "appadd", "", "PATH", "Add application to configuration (use full path to binary)")
		c.StringVar(&c.appremove, "appremove", "", "PATH", "Delete application from configuration (use full path to binary)")
	} else {
		// Linux
		c.BoolVar(&c.statusFull, "status_full", false, "(extended status info) Show detailed Split Tunnel status")
		c.BoolVar(&c.reset, "clean", false, "Erase configuration (delete all applications from configuration and disable)")
		c.StringVar(&c.appadd, "appadd", "", "COMMAND", "Execute command (binary) in Split Tunnel environment (exclude it's traffic from the VPN tunnel)\nInfo: short version of this command is 'ivpn exclude <command>'\nExamples:\n    ivpn splittun -appadd firefox\n    ivpn splittun -appadd ping 1.1.1.1\n    ivpn splittun -appadd /usr/bin/google-chrome")
		c.StringVar(&c.appremove, "appremove", "", "PID", "Remove application from Split Tunnel environment\n(argument: Process ID)")
	}

	c.StringVar(&c.dnsFirewall, cmd_name_dns_firewall, "", "[on/off]",
		`When this option is enabled, only DNS requests directed to IVPN DNS servers
		or user-defined custom DNS servers within the IVPN appsettings will be allowed.
		All other DNS requests on port 53 will be blocked.
		Note: The IVPN AntiTracker and custom DNS are not functional when this feature is disabled.
		Note: This feature is only applicable for Inverse Split Tunnel mode.`)

	c.BoolVar(&c.on, "on", false, "Enable")
	c.BoolVar(&c.onInverse, cmd_name_on_inverse, false, "Enable in inverse mode. Only specified applications utilize the VPN connection,\nwhile all other traffic circumvents the VPN, using the default connection")

	c.BoolVar(&c.off, "off", false, "Disable")

}

func (c *SplitTun) Run() error {
	if (c.on || c.onInverse) && c.off {
		return flags.ConflictingParameters{}
	}

	if len(c.appadd) > 0 && len(c.appremove) > 0 {
		return flags.ConflictingParameters{}
	}

	cfg, err := _proto.GetSplitTunnelStatus()
	if err != nil {
		return err
	}

	dnsFirewall := !cfg.IsAnyDns
	if len(c.dnsFirewall) > 0 {
		if !c.onInverse {
			return flags.BadParameter{Message: fmt.Sprintf("the '-%s' option is only applicable with '-%s' (Inverse Split Tunnel mode)", cmd_name_dns_firewall, cmd_name_on_inverse)}
		}
		dnsFirewall, err = helpers.BoolParameterParse(c.dnsFirewall) // [on/off]
		if err != nil {
			return err
		}
	}

	if cfg.IsFunctionalityNotAvailable {
		return fmt.Errorf("the Split Tunneling functionality not available")
	}

	if c.reset {
		cfg.IsEnabled = false
		cfg.SplitTunnelApps = make([]string, 0)

		if err = _proto.SetSplitTunnelConfig(false, false, false, true); err != nil {
			return err
		}
		cfg, err = _proto.GetSplitTunnelStatus()
		if err != nil {
			return err
		}
		return c.doShowStatus(cfg, c.statusFull)
	}

	if c.on || c.onInverse || c.off || len(c.dnsFirewall) > 0 {
		isEnabled := c.on || c.onInverse || cfg.IsEnabled
		isInverse := c.onInverse
		isAnyDns := !dnsFirewall

		if c.off {
			isEnabled = false
		}

		if err = _proto.SetSplitTunnelConfig(isEnabled, isInverse, isAnyDns, false); err != nil {
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
			if err = doAddApp(c.appaddArgs, "", false); err != nil {
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

	return c.doShowStatus(cfg, c.statusFull)
}

func (c *SplitTun) doShowStatus(cfg types.SplitTunnelStatus, isFull bool) error {
	w := printSplitTunState(nil, false, isFull, cfg.IsEnabled, cfg.IsInversed, cfg.IsAnyDns, cfg.SplitTunnelApps, cfg.RunningApps)
	w.Flush()
	return nil
}

func (c *SplitTun) doShowStatusShort(cfg types.SplitTunnelStatus) error {
	w := printSplitTunState(nil, true, false, cfg.IsEnabled, cfg.IsInversed, cfg.IsAnyDns, cfg.SplitTunnelApps, cfg.RunningApps)
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
