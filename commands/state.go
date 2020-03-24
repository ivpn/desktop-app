package commands

import (
	"fmt"

	"github.com/ivpn/desktop-app-cli/flags"
)

type CmdState struct {
	flags.CmdInfo
}

func (c *CmdState) Init() {
	c.Initialize("state", "Prints full info about IVPN state")
}
func (c *CmdState) Run() error {
	fwstate, err := _proto.FirewallStatus()
	if err != nil {
		return err
	}

	state, connected, err := _proto.GetVPNState()
	if err != nil {
		return err
	}

	printAccountInfo(_proto.GetHelloResponse().Session.AccountID)
	printState(state, connected)
	printFirewallState(fwstate.IsEnabled, fwstate.IsPersistent, fwstate.IsAllowLAN, fwstate.IsAllowMulticast)

	fmt.Println("\nTips: ")
	if len(_proto.GetHelloResponse().Session.AccountID) == 0 {
		fmt.Println("  ivpn login        Log in with your Account ID")
	}
	fmt.Println("  ivpn -help        Show all commands")

	return nil
}
