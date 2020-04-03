package commands

import (
	"fmt"

	"github.com/ivpn/desktop-app-cli/flags"
	"github.com/ivpn/desktop-app-daemon/service"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

type CmdState struct {
	flags.CmdInfo
}

func (c *CmdState) Init() {
	c.Initialize("state", "Prints full info about IVPN state")
}
func (c *CmdState) Run() error {
	return showState()
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

	// TIPS
	tips := make([]TipType, 0, 3)
	if len(_proto.GetHelloResponse().Session.Session) == 0 {
		fmt.Println(" ", service.ErrorNotLoggedIn{})
		tips = append(tips, TipLogin)
	}
	if state == vpn.CONNECTED {
		tips = append(tips, TipDisconnect)
		if fwstate.IsEnabled == false {
			tips = append(tips, TipFirewallEnable)
		}
	} else if fwstate.IsEnabled {
		tips = append(tips, TipFirewallDisable)
	}
	tips = append(tips, TipHelp)
	tips = append(tips, TipHelpFull)
	PrintTips(tips)

	return nil
}
