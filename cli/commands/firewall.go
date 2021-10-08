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

	"github.com/ivpn/desktop-app/cli/flags"
)

type CmdFirewall struct {
	flags.CmdInfo
	status             bool
	on                 bool
	off                bool
	allowLan           bool
	blockLan           bool
	ivpnSvrAccessAllow bool
	ivpnSvrAccessBlock bool
	persistentOn       bool
	persistentOff      bool
	//allowLanMulticast bool
	//blockLanMulticast bool
}

func (c *CmdFirewall) Init() {
	c.Initialize("firewall", "Firewall management")
	c.BoolVar(&c.status, "status", false, "(default) Show info about current firewall status")
	c.BoolVar(&c.off, "off", false, "Switch-off firewall")
	c.BoolVar(&c.on, "on", false, "Switch-on firewall")
	c.BoolVar(&c.allowLan, "lan_allow", false, "Set configuration: allow LAN communication (take effect when firewall enabled)")
	c.BoolVar(&c.blockLan, "lan_block", false, "Set configuration: block LAN communication (take effect when firewall enabled)")
	c.BoolVar(&c.ivpnSvrAccessAllow, "ivpn_access_allow", false, "Allow access to IVPN servers when Firewall is enabled")
	c.BoolVar(&c.ivpnSvrAccessBlock, "ivpn_access_block", false, "Block access to IVPN servers when Firewall is enabled")
	c.BoolVar(&c.persistentOff, "persistent_off", false, "Persistent firewall (Always-on firewall): disable")
	c.BoolVar(&c.persistentOn, "persistent_on", false, "Persistent firewall (Always-on firewall): enable. When the option is enabled the IVPN Firewall is started during system boot")

	//c.BoolVar(&c.allowLanMulticast, "lan_multicast_allow", false, "Same as 'lan_allow' + allow multicast communication ")
	//c.BoolVar(&c.blockLanMulticast, "lan_multicast_block", false, "Same as 'lan_block' + block multicast communication")
}
func (c *CmdFirewall) Run() error {
	if c.on && c.off {
		return flags.BadParameter{}
	}

	if c.allowLan && c.blockLan {
		return flags.BadParameter{}
	}

	if c.persistentOn && c.persistentOff {
		return flags.BadParameter{}
	}

	if c.persistentOn && c.off {
		return flags.BadParameter{}
	}

	//if c.allowLanMulticast && c.blockLanMulticast {
	//	return flags.BadParameter{}
	//}

	if c.ivpnSvrAccessAllow {
		if err := _proto.FirewallAllowApiServers(true); err != nil {
			return err
		}
	} else if c.ivpnSvrAccessBlock {
		if err := _proto.FirewallAllowApiServers(false); err != nil {
			return err
		}
	}

	if c.allowLan {
		if err := _proto.FirewallAllowLan(true); err != nil {
			return err
		}
	} else if c.blockLan {
		if err := _proto.FirewallAllowLan(false); err != nil {
			return err
		}
	}

	//if c.allowLanMulticast {
	//	if err := _proto.FirewallAllowLanMulticast(true); err != nil {
	//		return err
	//	}
	//} else if c.blockLanMulticast {
	//	if err := _proto.FirewallAllowLanMulticast(false); err != nil {
	//		return err
	//	}
	//}

	if c.persistentOn {
		if err := _proto.FirewallPersistentSet(true); err != nil {
			return err
		}
	} else if c.persistentOff {
		if err := _proto.FirewallPersistentSet(false); err != nil {
			return err
		}
	}

	if c.on {
		if err := _proto.FirewallSet(true); err != nil {
			return err
		}
	} else if c.off {

		state, err := _proto.FirewallStatus()
		if err == nil && state.IsPersistent {
			PrintTips([]TipType{TipFirewallDisablePersistent})
			return fmt.Errorf("Not possible to disable Firewall in 'Always-on' state")
		}

		if err := _proto.FirewallSet(false); err != nil {
			return err
		}
	}

	state, err := _proto.FirewallStatus()
	if err != nil {
		return err
	}

	w := printFirewallState(nil, state.IsEnabled, state.IsPersistent, state.IsAllowLAN, state.IsAllowMulticast, state.IsAllowApiServers)
	w.Flush()

	// TIPS
	tips := make([]TipType, 0, 2)
	if state.IsEnabled == false {
		tips = append(tips, TipFirewallEnable)
	} else {
		tips = append(tips, TipFirewallDisable)
	}
	PrintTips(tips)
	return nil
}
