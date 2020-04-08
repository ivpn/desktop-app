package service

import "github.com/ivpn/desktop-app-daemon/api/types"

func implIsNeedCheckOvpnVer() bool { return false }

func (s *Service) implIsGoingToPingServers(servers *types.ServersInfoResponse) error {
	// nothing to do for Windows implementation
	// firewall configured to allow all connectivity for service
	return nil
}
