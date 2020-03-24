package commands

import "github.com/ivpn/desktop-app-cli/flags"

type CmdFirewall struct {
	flags.CmdInfo
	status bool
	on     bool
	off    bool
}

func (c *CmdFirewall) Init() {
	c.Initialize("firewall", "Firewall management")
	c.BoolVar(&c.status, "status", false, "(default) Show info about current firewall status")
	c.BoolVar(&c.off, "off", false, "Switch-off firewall")
	c.BoolVar(&c.on, "on", false, "Switch-on firewall")
}
func (c *CmdFirewall) Run() error {
	if c.on && c.off {
		return flags.BadParameter{}
	}

	if c.on {
		return _proto.FirewallSet(true)
	} else if c.off {
		return _proto.FirewallSet(false)
	}

	state, err := _proto.FirewallStatus()
	if err != nil {
		return err
	}

	printFirewallState(state.IsEnabled, state.IsPersistent, state.IsAllowLAN, state.IsAllowMulticast)
	return nil
}
