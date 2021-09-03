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
	"strings"

	"github.com/ivpn/desktop-app/cli/flags"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
)

type SplitTun struct {
	flags.CmdInfo
	status    bool
	on        bool
	off       bool
	reset     bool
	appadd    string
	appremove string
}

func (c *SplitTun) Init() {
	c.Initialize("splittun", "Split Tunnel management\nBy enabling this feature you can exclude traffic for a specific applications from the VPN tunnel")
	c.BoolVar(&c.status, "status", false, "(default) Show Split Tunnel status and configuration")
	c.BoolVar(&c.on, "on", false, "Enable")
	c.BoolVar(&c.off, "off", false, "Disable")
	c.BoolVar(&c.reset, "clean", false, "Erase configuration (delete all applications from configuration and disable)")
	c.StringVar(&c.appadd, "appadd", "", "PATH", "Add application to configuration (use full path to binary)")
	c.StringVar(&c.appremove, "appremove", "", "PATH", "Delete application from configuration (use full path to binary)")
}

func (c *SplitTun) Run() error {
	if c.on && c.off {
		return flags.BadParameter{}
	}
	if len(c.appadd) > 0 && len(c.appremove) > 0 {
		return flags.BadParameter{}
	}

	cfg, err := _proto.GetSplitTunnelConfig()
	if err != nil {
		return err
	}

	if c.reset {
		cfg.IsEnabled = false
		cfg.SplitTunnelApps = make([]string, 0)

		if err = _proto.SetSplitTunnelConfig(cfg); err != nil {
			return err
		}
		cfg, err = _proto.GetSplitTunnelConfig()
		if err != nil {
			return err
		}
		return c.doShowStatus(cfg)
	}

	if c.on || c.off {
		cfg.IsEnabled = false
		if c.on {
			cfg.IsEnabled = true
		}
		if err = _proto.SetSplitTunnelConfig(cfg); err != nil {
			return err
		}
		cfg, err = _proto.GetSplitTunnelConfig()
		if err != nil {
			return err
		}
		return c.doShowStatusShort(cfg)
	}

	if len(c.appadd) > 0 || len(c.appremove) > 0 {
		if len(c.appadd) > 0 {
			cfg.SplitTunnelApps = append(cfg.SplitTunnelApps, c.appadd)
		} else if len(c.appremove) > 0 {
			isRemoved := false
			newAppsList := make([]string, 0, len(cfg.SplitTunnelApps))
			for _, path := range cfg.SplitTunnelApps {
				if strings.EqualFold(c.appremove, path) {
					isRemoved = true
					continue
				}
				newAppsList = append(newAppsList, path)
			}
			if !isRemoved {
				return fmt.Errorf("nothing to remove (defined application path is not in Split Tunnel configuration)")
			}
			cfg.SplitTunnelApps = newAppsList
		}

		if err = _proto.SetSplitTunnelConfig(cfg); err != nil {
			return err
		}
		cfg, err = _proto.GetSplitTunnelConfig()
		if err != nil {
			return err
		}
	}

	return c.doShowStatus(cfg)
}

func (c *SplitTun) doShowStatus(cfg types.SplitTunnelConfig) error {
	w := printSplitTunState(nil, false, cfg.IsEnabled, cfg.SplitTunnelApps)
	w.Flush()
	return nil
}

func (c *SplitTun) doShowStatusShort(cfg types.SplitTunnelConfig) error {
	w := printSplitTunState(nil, true, cfg.IsEnabled, cfg.SplitTunnelApps)
	w.Flush()
	return nil
}
