package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ivpn/desktop-app-cli/protocol"
	apitypes "github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

var _proto *protocol.Client

// Initialize initializes commands. Must be called before using any command.
func Initialize(proto *protocol.Client) {
	_proto = proto
}

func printAccountInfo(w *tabwriter.Writer, accountID string) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	if len(accountID) > 0 {
		return w // Do nothing in case of logged in
	}

	fmt.Fprintln(w, fmt.Sprintf("Account\t:\t%v", "Not logged in"))

	return w
}

func printState(w *tabwriter.Writer, state vpn.State, connected types.ConnectedResp, serverInfo string, exitServerInfo string) *tabwriter.Writer {

	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	fmt.Fprintln(w, fmt.Sprintf("VPN\t:\t%v", state))

	if len(serverInfo) > 0 {
		fmt.Fprintln(w, fmt.Sprintf("\t\t%v", serverInfo))
		if len(exitServerInfo) > 0 {
			fmt.Fprintln(w, fmt.Sprintf("\t\t%v (Multi-Hop exit server)", exitServerInfo))
		}
	}

	if state != vpn.CONNECTED {
		return w
	}
	since := time.Unix(connected.TimeSecFrom1970, 0)
	fmt.Fprintln(w, fmt.Sprintf("    Protocol\t:\t%v", connected.VpnType))
	fmt.Fprintln(w, fmt.Sprintf("    Local IP\t:\t%v", connected.ClientIP))
	fmt.Fprintln(w, fmt.Sprintf("    Server IP\t:\t%v", connected.ServerIP))
	fmt.Fprintln(w, fmt.Sprintf("    Connected\t:\t%v", since))

	return w
}

func printDNSState(w *tabwriter.Writer, dns string, servers *apitypes.ServersInfoResponse) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	dns = strings.TrimSpace(dns)
	if len(dns) == 0 {
		fmt.Fprintln(w, fmt.Sprintf("DNS\t:\tDefault (auto)"))
		return w
	}

	antitrackerText := strings.Builder{}

	isAntitracker, isAtHardcore := IsAntiTrackerIP(dns, servers)
	if isAtHardcore {
		antitrackerText.WriteString("Enabled (Hardcore)")
	} else if isAntitracker {
		antitrackerText.WriteString("Enabled")
	}

	if antitrackerText.Len() > 0 {
		fmt.Fprintln(w, fmt.Sprintf("AntiTracker\t:\t%v", antitrackerText.String()))
	} else {
		fmt.Fprintln(w, fmt.Sprintf("DNS\t:\t%v", dns))
	}

	return w
}

func printFirewallState(w *tabwriter.Writer, isEnabled, isPersistent, isAllowLAN, isAllowMulticast bool) *tabwriter.Writer {
	if w == nil {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	}

	fwState := "Disabled"
	if isEnabled {
		fwState = "Enabled"
	}

	fmt.Fprintln(w, fmt.Sprintf("Firewall\t:\t%v", fwState))
	fmt.Fprintln(w, fmt.Sprintf("    Allow LAN\t:\t%v", isAllowLAN))
	if isPersistent {
		fmt.Fprintln(w, fmt.Sprintf("    Persistent\t:%v", isPersistent))
	}

	return w
}
