package commands

import (
	"fmt"

	"github.com/ivpn/desktop-app-cli/commands/config"
	"github.com/ivpn/desktop-app-cli/flags"
	apitypes "github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

type CmdDns struct {
	flags.CmdInfo
	reset bool
	dns   string
}

func (c *CmdDns) Init() {
	c.Initialize("dns", "Specifying default configuration of 'custom DNS' for VPN connection\nDNS_IP - optional parameter used to set custom dns value (ignored when AntiTracker enabled)")
	c.DefaultStringVar(&c.dns, "DNS_IP")
	c.BoolVar(&c.reset, "reset", false, "Reset DNS server to a default")
}

func (c *CmdDns) Run() error {
	if c.reset && len(c.dns) > 0 {
		return flags.BadParameter{}
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	// do we have to change custom DNS configuration ?
	if c.reset || len(c.dns) > 0 {
		cfg.CustomDNS = ""
		if len(c.dns) > 0 {
			cfg.CustomDNS = c.dns
		}

		err = config.SaveConfig(cfg)
		if err != nil {
			return err
		}

		// update DNS if VPN is connected
		state, _, err := _proto.GetVPNState()
		if err != nil {
			return err
		}
		if state == vpn.CONNECTED {
			if err := _proto.SetManualDNS(cfg.CustomDNS); err != nil {
				return err
			}
			fmt.Println("Custom DNS successfully changed for current VPN connection")
		}
	}

	// print status
	PrintDnsConfigInfo(cfg.CustomDNS)
	return nil
}

//----------------------------------------------------------------------------------------

type CmdAntitracker struct {
	flags.CmdInfo
	def      bool
	off      bool
	hardcore bool
}

func (c *CmdAntitracker) Init() {
	c.Initialize("antitracker", "Specifying default AntiTracker configuration for VPN connection")
	c.BoolVar(&c.def, "default", false, "Enable AntiTracker")
	c.BoolVar(&c.hardcore, "hardcore", false, "Enable AntiTracker 'hardcore' mode")
	c.BoolVar(&c.off, "off", false, "Disable AntiTracker")
}

func (c *CmdAntitracker) Run() error {
	if c.NFlag() > 1 {
		return flags.BadParameter{Message: "Not allowed to use more than one argument for this command"}
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	// do we have to change antitracker configuration ?
	if c.off || c.def || c.hardcore {
		cfg.Antitracker = false
		cfg.AntitrackerHardcore = false

		if c.hardcore {
			cfg.AntitrackerHardcore = true
		} else if c.def {
			cfg.Antitracker = true
		}

		err = config.SaveConfig(cfg)
		if err != nil {
			return err
		}

		// update DNS if VPN is connected
		state, connectInfo, err := _proto.GetVPNState()
		if err != nil {
			return err
		}
		if state == vpn.CONNECTED {
			dns := ""
			if cfg.Antitracker || cfg.AntitrackerHardcore {
				dns, err = GetAntitrackerIP(cfg.AntitrackerHardcore, len(connectInfo.ExitServerID) > 0, nil)
			}
			if err := _proto.SetManualDNS(dns); err != nil {
				return err
			}
			fmt.Println("AntiTracker successfully updated for current VPN connection")
		}
	}

	// print status
	PrintAntitrackerConfigInfo(cfg.Antitracker, cfg.AntitrackerHardcore)
	return nil
}

//----------------------------------------------------------------------------------------

func PrintDnsConfigInfo(customDNS string) {
	if len(customDNS) > 0 {
		fmt.Println("[default configuration] Custom DNS :", customDNS)
	} else {
		fmt.Println("[default configuration] Custom DNS : not defined")
	}
}

func PrintAntitrackerConfigInfo(antitracker, antitrackerHardcore bool) {
	if antitrackerHardcore {
		fmt.Println("[default configuration] AntiTracker: Enabled (hardcore)")
	} else if antitracker {
		fmt.Println("[default configuration] AntiTracker: Enabled")
	} else {
		fmt.Println("[default configuration] AntiTracker: Disabled")
	}
}

//----------------------------------------------------------------------------------------

// GetAntitrackerIP - returns IP of antitracker DNS
func GetAntitrackerIP(isHardcore, isMultihop bool, servers *apitypes.ServersInfoResponse) (string, error) {
	if servers == nil {
		srvs, err := _proto.GetServers()
		if err != nil {
			return "", err
		}
		servers = &srvs
	}

	if isHardcore {
		if isMultihop {
			return servers.Config.Antitracker.Hardcore.MultihopIP, nil
		}
		return servers.Config.Antitracker.Hardcore.IP, nil
	}

	if isMultihop {
		return servers.Config.Antitracker.Default.MultihopIP, nil
	}
	return servers.Config.Antitracker.Default.IP, nil
}
