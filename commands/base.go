package commands

import (
	"fmt"
	"time"

	"github.com/ivpn/desktop-app-cli/protocol"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

var _proto *protocol.Client

// Initialize initializes commands. Must be called before using any command.
func Initialize(proto *protocol.Client) {
	_proto = proto
}

func printAccountInfo(accountID string) {
	status := "Not logged in"
	if len(accountID) > 0 {
		return // Do nothing in case of logged in
	}
	fmt.Printf("Account                 : %v\n", status)
}

func printState(state vpn.State, connected types.ConnectedResp) {
	fmt.Printf("VPN                     : %v\n", state)

	if state != vpn.CONNECTED {
		return
	}
	since := time.Unix(connected.TimeSecFrom1970, 0)
	fmt.Printf("    Protocol            : %v\n", connected.VpnType)
	fmt.Printf("    Local IP            : %v\n", connected.ClientIP)
	fmt.Printf("    Server IP           : %v\n", connected.ServerIP)
	fmt.Printf("    Connected           : %v\n", since)
}

func printFirewallState(isEnabled, isPersistent, isAllowLAN, isAllowMulticast bool) {
	fmt.Println("Firewall:")
	fmt.Printf("    Enabled             : %v\n", isEnabled)
	fmt.Printf("    Persistent          : %v\n", isPersistent)
	fmt.Printf("    Allow LAN           : %v\n", isAllowLAN)
	fmt.Printf("    Allow LAN multicast : %v\n", isAllowMulticast)
}
