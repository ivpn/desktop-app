package service

import "github.com/ivpn/desktop-app-daemon/api/types"

func (s *Service) implIsGoingToPingServers(servers *types.ServersInfoResponse) error {
	// nothing to do for Windows implementation
	// firewall configured to allow all connectivity for service
	return nil
}
