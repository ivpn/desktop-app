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
	"text/tabwriter"
	"time"

	"github.com/ivpn/desktop-app/cli/flags"
	"github.com/ivpn/desktop-app/daemon/service/srverrors"
)

type CmdWireGuard struct {
	flags.CmdInfo
	state            bool
	regenerate       bool
	rotationInterval int
}

func (c *CmdWireGuard) Init() {
	c.Initialize("wgkeys", "WireGuard keys management")
	c.BoolVar(&c.state, "status", false, "(default) Show WireGuard configuration")
	c.IntVar(&c.rotationInterval, "rotation_interval", 0, "DAYS", "Set WireGuard keys rotation interval. [1-30] days")
	c.BoolVar(&c.regenerate, "regenerate", false, "Regenerate WireGuard keys")
}
func (c *CmdWireGuard) Run() error {
	if c.rotationInterval < 0 || c.rotationInterval > 30 {
		fmt.Println("Error: keys rotation interval should be in diapasone [1-30] days")
		return flags.BadParameter{}
	}

	defer func() {
		helloResp := _proto.GetHelloResponse()
		if len(helloResp.Session.Session) == 0 {
			fmt.Println(srverrors.ErrorNotLoggedIn{})

			PrintTips([]TipType{TipLogin})
		}
	}()

	resp, err := _proto.SendHello()
	if err != nil {
		return err
	}
	if len(resp.DisabledFunctions.WireGuardError) > 0 {
		return fmt.Errorf("WireGuard functionality disabled:\n\t" + resp.DisabledFunctions.WireGuardError)
	}

	if c.regenerate {
		fmt.Println("Regenerating WG keys...")
		if err := c.generate(); err != nil {
			return err
		}
	}

	if c.rotationInterval > 0 {
		interval := time.Duration(time.Hour * 24 * time.Duration(c.rotationInterval))
		fmt.Printf("Changing WG keys rotation interval to %v ...\n", interval)
		if err := c.setRotateInterval(int64(interval / time.Second)); err != nil {
			return err
		}
	}

	if err := c.getState(); err != nil {
		return err
	}

	return nil
}

func (c *CmdWireGuard) generate() error {
	return _proto.WGKeysGenerate()
}

func (c *CmdWireGuard) setRotateInterval(interval int64) error {
	return _proto.WGKeysRotationInterval(interval)
}

func (c *CmdWireGuard) getState() error {
	resp, err := _proto.SendHello()
	if err != nil {
		return err
	}

	if len(resp.Session.Session) == 0 {
		return nil
	}

	quantumResistanceStatus := "Disabled"
	if resp.Session.WgUsePresharedKey {
		quantumResistanceStatus = "Enabled"
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "Local IP:\t%v\n", resp.Session.WgLocalIP)
	fmt.Fprintf(w, "Public KEY:\t%v\n", resp.Session.WgPublicKey)
	fmt.Fprintf(w, "Quantum Resistance:\t%v\n", quantumResistanceStatus)
	fmt.Fprintf(w, "Generated:\t%v\n", time.Unix(resp.Session.WgKeyGenerated, 0))
	fmt.Fprintf(w, "Rotation interval:\t%v\n", time.Duration(time.Second*time.Duration(resp.Session.WgKeysRegenInerval)))
	w.Flush()

	return nil
}
