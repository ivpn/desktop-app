package commands

import (
	"fmt"
	"os"

	"github.com/ivpn/desktop-app-cli/flags"
	"github.com/ivpn/desktop-app-daemon/service"
)

type CmdState struct {
	flags.CmdInfo
}

func (c *CmdState) Init() {
	c.Initialize("state", "Prints full info about IVPN state")
}
func (c *CmdState) Run() error {
	err := showState()

	fmt.Println("\nTips: ")
	if len(_proto.GetHelloResponse().Session.Session) == 0 {
		fmt.Println(" ", service.ErrorNotLoggedIn{})
		fmt.Printf("  %s account -login  ACCOUNT_ID         Log in with your Account ID\n", os.Args[0])
	}
	fmt.Printf("  %s -help                              Show all commands\n", os.Args[0])

	return err
}

func showState() error {
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

	return nil
}
