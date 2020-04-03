package commands

import (
	"strings"

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

	serverInfo := ""
	exitServerInfo := ""

	if state == vpn.CONNECTED {
		servers, err := _proto.GetServers()
		if err == nil {
			slist := serversList(servers)
			serverInfo = getServerInfoByIP(slist, connected.ServerIP)
			exitServerInfo = getServerInfoByID(slist, connected.ExitServerID)
		}
	}

	w := printAccountInfo(nil, _proto.GetHelloResponse().Session.AccountID)
	printState(w, state, connected, serverInfo, exitServerInfo)
	printFirewallState(w, fwstate.IsEnabled, fwstate.IsPersistent, fwstate.IsAllowLAN, fwstate.IsAllowMulticast)
	w.Flush()

	// TIPS
	tips := make([]TipType, 0, 3)
	if len(_proto.GetHelloResponse().Session.Session) == 0 {
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

	if len(_proto.GetHelloResponse().Session.Session) == 0 {
		return service.ErrorNotLoggedIn{}
	}
	return nil
}

func getServerInfoByIP(servers []serverDesc, ip string) string {
	ip = strings.TrimSpace(ip)
	for _, s := range servers {
		for h := range s.hosts {
			if ip == strings.TrimSpace(h) {
				return s.String()
			}
		}
	}
	return ""
}

func getServerInfoByID(servers []serverDesc, id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	if len(id) == 0 {
		return ""
	}

	for _, s := range servers {
		sID := strings.ToLower(strings.TrimSpace(s.gateway))
		if strings.HasPrefix(sID, id) {
			return s.String()
		}
	}
	return ""
}
