package service

import "ivpn/daemon/service/api"

func (s *service) implIsGoingToPingServers(servers *api.ServersInfoResponse) error {
	// nothing to do for Windows implementation
	// firewall configured to allow all connectivity for service
	return nil
}
