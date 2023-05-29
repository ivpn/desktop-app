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

	"github.com/ivpn/desktop-app/cli/flags"
)

type CmdConnectionControl struct {
	flags.CmdInfo
	resume bool
	pause  int
}

func (c *CmdConnectionControl) Init() {
	c.KeepArgsOrderInHelp = true

	c.Initialize("connection", "Managing active connection")
	c.IntVar(&c.pause, "pause", 0, "DURATION", "Temporarily pause the connection for a specified duration\n  (DURATION: [1-1440] minutes)")
	c.BoolVar(&c.resume, "resume", false, "Resume a paused connection")

}

func (c *CmdConnectionControl) Run() error {
	if (c.resume && c.pause > 0) || c.pause < 0 {
		return flags.BadParameter{}
	}
	if c.pause > 60*24 {
		return flags.BadParameter{}
	}

	defer showState()

	if c.pause > 0 {
		fmt.Printf("Pausing connection on %d minutes...\n", c.pause)
		return _proto.Pause(uint32(c.pause) * 60)
	}

	if c.resume {
		fmt.Println("Resuming connection...")
		return _proto.Pause(0)
	}

	return nil
}
