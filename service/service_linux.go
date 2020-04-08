package service

import (
	"github.com/ivpn/desktop-app-daemon/api/types"
)

func implIsNeedCheckOvpnVer() bool { return true }

func (s *Service) implIsGoingToPingServers(servers *types.ServersInfoResponse) error {
	// TODO: not implemented
	return nil
}
