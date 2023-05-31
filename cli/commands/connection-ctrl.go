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
	"time"

	"github.com/ivpn/desktop-app/cli/flags"
	"github.com/ivpn/desktop-app/daemon/vpn"
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

	var retErr error
	if c.pause > 0 {
		fmt.Printf("Pausing connection on %d minutes...\n", c.pause)
		retErr = _proto.Pause(uint32(c.pause) * 60)
	} else if c.resume {
		fmt.Println("Resuming connection...")
		retErr = _proto.Pause(0)
		if retErr == nil {
			// Wait for connected state.
			// After resume command, the state can be changed to different values  (RECONNECTING, INITIALISING, DISCONNECTED... etc.).
			// It depends of VPN protocol implementation for specific platform.
			// So here we wait some time for connected state.
			waitDeadline := time.Now().Add(time.Second * 10)
			disconnectedResponsesCnt := 0
			for ; time.Now().Before(waitDeadline); time.Sleep(time.Second) {
				state, _, err := _proto.GetVPNState()
				if err != nil || state == vpn.CONNECTED || state == vpn.DISCONNECTED {
					if state == vpn.DISCONNECTED {
						// it could be a temporary state, so we wait for a few seconds
						disconnectedResponsesCnt++
						if disconnectedResponsesCnt <= 3 {
							continue
						}
					}
					break
				}
			} // wait for connected state
		}
	} else {
		return flags.BadParameter{}
	}

	if retErr != nil {
		return retErr
	}

	showState()

	return nil
}
