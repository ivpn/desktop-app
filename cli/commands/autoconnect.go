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
	"text/tabwriter"

	"github.com/ivpn/desktop-app/cli/flags"
	"github.com/ivpn/desktop-app/cli/helpers"
	service_types "github.com/ivpn/desktop-app/daemon/protocol/types"
)

type CmdAutoConnect struct {
	flags.CmdInfo
	status        bool
	on_launch_val string // on/off
}

func (c *CmdAutoConnect) Init() {
	c.KeepArgsOrderInHelp = true

	c.Initialize("autoconnect", "Manage VPN auto-connection parameters")
	c.BoolVar(&c.status, "status", false, "(default) Show settings")
	c.StringVar(&c.on_launch_val, "on_launch", "", "[on/off]", "Autoconnect on daemon launch\nThis enables the VPN tunnel to startup as quickly as possible\nas the daemon is started early in the operating system boot process\nand before the IVPN app (The GUI)")

}

func (c *CmdAutoConnect) Run() error {
	isChanged := false

	if len(c.on_launch_val) > 0 {
		val, err := helpers.BoolParameterParse(c.on_launch_val)
		if err != nil {
			return err
		}

		if err := c.setAutoconnectOnLaunch(val, true); err != nil {
			return err
		}

		isChanged = true
	}

	// -status

	// request updated daemon settings
	if _, err := _proto.SendHello(); err != nil {
		return err
	}
	w := c.printAutoconnectSettings(nil)
	w.Flush()

	if !isChanged {
		PrintTips([]TipType{TipAutoconnectHelp})
	}

	return nil
}

func (c *CmdAutoConnect) setAutoconnectOnLaunch(enable bool, runInBackground bool) error {

	if enable && runInBackground && _proto.GetHelloResponse().ParanoidMode.IsEnabled {
		return EaaEnabledOptionNotApplicable{}
	}

	enable_strVal := "false"
	if enable {
		enable_strVal = "true"
	}

	runInBackground_strVal := "false"
	if enable {
		runInBackground_strVal = "true"
	}

	if err := _proto.SetPreferences(string(service_types.Prefs_IsAutoconnectOnLaunch), enable_strVal); err != nil {
		return err
	}

	if err := _proto.SetPreferences(string(service_types.Prefs_IsAutoconnectOnLaunch_Daemon), runInBackground_strVal); err != nil {
		return err
	}

	return nil
}

func (c *CmdAutoConnect) printAutoconnectSettings(w *tabwriter.Writer) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	daemonSettings := _proto.GetHelloResponse().DaemonSettings

	aol := "Disabled"
	if daemonSettings.IsAutoconnectOnLaunch && daemonSettings.IsAutoconnectOnLaunchDaemon {
		aol = "Enabled"
	}
	fmt.Fprintf(w, "Autoconnect on daemon launch\t:\t%v\n", aol)

	//inBackground := "Disabled"
	//if daemonSettings.IsAutoconnectOnLaunchDaemon {
	//	inBackground = "Enabled"
	//}
	//fmt.Fprintf(w, "Autoconnect on launch (background daemon)\t:\t%v\n", inBackground)
	return w
}
