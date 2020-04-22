package protocol

import (
	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/version"
)

// OnServiceSessionChanged - SessionChanged handler
func (p *Protocol) OnServiceSessionChanged() {
	service := p._service
	if service == nil {
		return
	}

	// send back Hello message with account session info
	helloResp := types.HelloResp{
		Version: version.Version(),
		Session: types.CreateSessionResp(service.Preferences().Session)}

	p.sendResponse(&helloResp, 0)
}
