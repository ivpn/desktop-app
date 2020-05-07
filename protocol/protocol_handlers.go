package protocol

import (
	"net"

	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/service/preferences"
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

	p.notifyClients(&helloResp)
}

// OnAccountStatus - handler of account status info. Notifying clients.
func (p *Protocol) OnAccountStatus(sessionToken string, accountInfo preferences.AccountStatus) {
	if len(sessionToken) == 0 {
		return
	}

	p.notifyClients(&types.AccountStatusResp{
		SessionToken: sessionToken,
		Account:      accountInfo})
}

// OnDNSChanged - DNS changed handler
func (p *Protocol) OnDNSChanged(dns net.IP) {
	// notify all clients
	if dns == nil {
		p.notifyClients(&types.SetAlternateDNSResp{IsSuccess: true, ChangedDNS: ""})
	} else {
		p.notifyClients(&types.SetAlternateDNSResp{IsSuccess: true, ChangedDNS: dns.String()})
	}
}

// OnKillSwitchStateChanged - Firewall change handler
func (p *Protocol) OnKillSwitchStateChanged() {
	// notify all clients about KillSwitch status
	if isEnabled, isPersistant, isAllowLAN, isAllowLanMulticast, err := p._service.KillSwitchState(); err != nil {
		log.Error(err)
	} else {
		p.notifyClients(&types.KillSwitchStatusResp{IsEnabled: isEnabled, IsPersistent: isPersistant, IsAllowLAN: isAllowLAN, IsAllowMulticast: isAllowLanMulticast})
	}
}
