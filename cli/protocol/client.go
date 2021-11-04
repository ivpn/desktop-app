//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the IVPN command line interface.
//
//  The IVPN command line interface is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The IVPN command line interface is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the IVPN command line interface. If not, see <https://www.gnu.org/licenses/>.
//

package protocol

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/vpn"
)

// Client for IVPN daemon
type Client struct {
	_port   int
	_secret uint64
	_conn   net.Conn

	_requestIdx int

	_defaultTimeout  time.Duration
	_receivers       map[*receiverChannel]struct{}
	_receiversLocker sync.Mutex

	_helloResponse types.HelloResp
}

// ResponseTimeout error
type ResponseTimeout struct {
}

func (e ResponseTimeout) Error() string {
	return "response timeout"
}

// CreateClient initialising new client for IVPN daemon
func CreateClient(port int, secret uint64) *Client {
	return &Client{
		_port:           port,
		_secret:         secret,
		_defaultTimeout: time.Second * 60 * 3,
		_receivers:      make(map[*receiverChannel]struct{})}
}

// Connect is connecting to daemon
func (c *Client) Connect() (err error) {
	if c._conn != nil {
		return fmt.Errorf("already connected")
	}

	logger.Info("Connecting...")

	c._conn, err = net.Dial("tcp", fmt.Sprintf(":%d", c._port))
	if err != nil {
		return fmt.Errorf("failed to connect to IVPN daemon (does IVPN daemon/service running?): %w", err)
	}

	logger.Info("Connected")

	// start receiver
	go c.receiverRoutine()

	if _, err := c.SendHello(); err != nil {
		return err
	}

	return nil
}

// SendHello - send initial message and get current status
func (c *Client) SendHello() (helloResponse types.HelloResp, err error) {
	if err := c.ensureConnected(); err != nil {
		return helloResponse, err
	}

	helloReq := types.Hello{Secret: c._secret, KeepDaemonAlone: true, GetStatus: true, Version: "1.0"}

	if err := c.sendRecvTimeOut(&helloReq, &c._helloResponse, time.Second*7); err != nil {
		if _, ok := errors.Unwrap(err).(ResponseTimeout); ok {
			return helloResponse, fmt.Errorf("Failed to send 'Hello' request: %w", err)
		}
		return helloResponse, fmt.Errorf("Failed to send 'Hello' request: %w", err)
	}
	return c._helloResponse, nil
}

// GetHelloResponse returns initialization response from daemon
func (c *Client) GetHelloResponse() types.HelloResp {
	return c._helloResponse
}

// SessionNew creates new session
func (c *Client) SessionNew(accountID string, forceLogin bool, the2FA string) (apiStatus int, err error) {
	if err := c.ensureConnected(); err != nil {
		return 0, err
	}

	req := types.SessionNew{AccountID: accountID, ForceLogin: forceLogin, Confirmation2FA: the2FA}
	var resp types.SessionNewResp

	if err := c.sendRecv(&req, &resp); err != nil {
		return 0, err
	}

	if len(resp.Session.Session) <= 0 {
		return resp.APIStatus, fmt.Errorf("[%d] %s", resp.APIStatus, resp.APIErrorMessage)
	}

	return resp.APIStatus, nil
}

// SessionDelete remove session
func (c *Client) SessionDelete(needToDisableFirewall, resetAppSettingsToDefaults, isCanDeleteSessionLocally bool) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.SessionDelete{
		NeedToDisableFirewall:     needToDisableFirewall,
		NeedToResetSettings:       resetAppSettingsToDefaults,
		IsCanDeleteSessionLocally: isCanDeleteSessionLocally}

	var resp types.EmptyResp

	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	return nil
}

// SessionStatus get session status
func (c *Client) SessionStatus() (ret types.AccountStatusResp, err error) {
	if err := c.ensureConnected(); err != nil {
		return ret, err
	}

	req := types.AccountStatus{}
	var resp types.AccountStatusResp

	if err := c.sendRecv(&req, &resp); err != nil {
		return ret, err
	}

	return resp, nil
}

// SetPreferences sends config parameter to daemon
// TODO: avoid using keys as a strings
func (c *Client) SetPreferences(key, value string) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.SetPreference{Key: key, Value: value}

	// TODO: daemon have to return confirmation
	if err := c.send(&req, 0); err != nil {
		return err
	}

	return nil
}

// FirewallSet change firewall state
func (c *Client) FirewallSet(isOn bool) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	// changing killswitch state
	req := types.KillSwitchSetEnabled{IsEnabled: isOn}
	var resp types.EmptyResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	// requesting status
	state, err := c.FirewallStatus()
	if err != nil {
		return err
	}

	if state.IsEnabled != isOn {
		return fmt.Errorf("firewall state did not change [isEnabled=%v]", state.IsEnabled)
	}

	return nil
}

// FirewallSet change firewall Persistent state
func (c *Client) FirewallPersistentSet(isOn bool) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	// changing killswitch Persistent state
	req := types.KillSwitchSetIsPersistent{IsPersistent: isOn}
	var resp types.EmptyResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	// requesting status
	state, err := c.FirewallStatus()
	if err != nil {
		return err
	}

	if state.IsPersistent != isOn || (isOn == true && state.IsEnabled != true) {
		return fmt.Errorf("firewall 'persistent' state did not change [isEnabled=%v; IsPersistent=%v]", state.IsEnabled, state.IsPersistent)
	}

	return nil
}

// FirewallAllowLan set configuration 'allow LAN'
func (c *Client) FirewallAllowLan(allow bool) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	// changing killswitch configuration
	req := types.KillSwitchSetAllowLAN{AllowLAN: allow, Synchronously: true}
	var resp types.EmptyResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	return nil
}

// FirewallAllowApiServers set configuration 'Allow access to IVPN servers when Firewall is enabled'
func (c *Client) FirewallAllowApiServers(allow bool) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	// changing killswitch configuration
	req := types.KillSwitchSetAllowApiServers{IsAllowApiServers: allow}
	var resp types.EmptyResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	return nil
}

// FirewallAllowLanMulticast set configuration 'allow LAN multicast'
func (c *Client) FirewallAllowLanMulticast(allow bool) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	// changing killswitch configuration
	req := types.KillSwitchSetAllowLANMulticast{AllowLANMulticast: allow, Synchronously: true}
	var resp types.EmptyResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	return nil
}

// FirewallStatus get firewall state
func (c *Client) FirewallStatus() (state types.KillSwitchStatusResp, err error) {
	if err := c.ensureConnected(); err != nil {
		return state, err
	}

	// requesting status
	statReq := types.KillSwitchGetStatus{}
	if err := c.sendRecv(&statReq, &state); err != nil {
		return state, err
	}

	return state, nil
}

// GetSplitTunnelConfig requests the Split-Tunnelling configuration
func (c *Client) GetSplitTunnelConfig() (cfg types.SplitTunnelConfig, err error) {
	if err := c.ensureConnected(); err != nil {
		return cfg, err
	}

	// requesting status
	req := types.SplitTunnelGetConfig{}
	if err := c.sendRecv(&req, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// SetSplitTunnelConfig sets the split-tunnelling configuration
func (c *Client) SetSplitTunnelConfig(cfg types.SplitTunnelConfig) (err error) {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.SplitTunnelSetConfig{IsEnabled: cfg.IsEnabled, SplitTunnelApps: cfg.SplitTunnelApps}
	resp := types.SplitTunnelConfig{}
	if _, _, err := c.sendRecvAny(&req, &resp); err != nil {
		return err
	}

	return nil
}

// GetServers gets servers list
func (c *Client) GetServers() (apitypes.ServersInfoResponse, error) {
	if err := c.ensureConnected(); err != nil {
		return apitypes.ServersInfoResponse{}, err
	}

	req := types.GetServers{}
	var resp types.ServerListResp

	if err := c.sendRecv(&req, &resp); err != nil {
		return resp.VpnServers, err
	}

	return resp.VpnServers, nil
}

// GetVPNState returns current VPN connection state
func (c *Client) GetVPNState() (vpn.State, types.ConnectedResp, error) {
	respConnected := types.ConnectedResp{}
	respDisconnected := types.DisconnectedResp{}
	respState := types.VpnStateResp{}

	if err := c.ensureConnected(); err != nil {
		return vpn.DISCONNECTED, respConnected, err
	}

	req := types.GetVPNState{}

	_, _, err := c.sendRecvAny(&req, &respConnected, &respDisconnected, &respState)
	if err != nil {
		return vpn.DISCONNECTED, respConnected, err
	}

	if len(respConnected.Command) > 0 {
		return vpn.CONNECTED, respConnected, nil
	}

	if len(respDisconnected.Command) > 0 {
		return vpn.DISCONNECTED, respConnected, nil
	}

	if len(respState.Command) > 0 {
		return respState.StateVal, respConnected, nil
	}

	return vpn.DISCONNECTED, respConnected, fmt.Errorf("failed to receive VPN state (not expected return type)")
}

// DisconnectVPN disconnect active VPN connection
func (c *Client) DisconnectVPN() error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.Disconnect{}
	respEmpty := types.EmptyResp{}
	respDisconnected := types.DisconnectedResp{}

	_, _, err := c.sendRecvAny(&req, &respDisconnected, &respEmpty)
	if err != nil {
		return err
	}

	if len(respDisconnected.Command) == 0 && len(respEmpty.Command) == 0 {
		return fmt.Errorf("disconnect request failed (not expected return type)")
	}

	return nil
}

// ConnectVPN - establish new VPN connection
func (c *Client) ConnectVPN(req types.Connect) (types.ConnectedResp, error) {
	respConnected := types.ConnectedResp{}
	respDisconnected := types.DisconnectedResp{}

	if err := c.ensureConnected(); err != nil {
		return respConnected, err
	}

	_, _, err := c.sendRecvAny(&req, &respConnected, &respDisconnected)
	if err != nil {
		return respConnected, err
	}

	if len(respConnected.Command) > 0 {
		return respConnected, nil
	}

	if len(respDisconnected.Command) > 0 {
		return respConnected, fmt.Errorf("%s", respDisconnected.ReasonDescription)
	}

	return respConnected, fmt.Errorf("connect request failed (not expected return type)")
}

// WGKeysGenerate regenerate WG keys
func (c *Client) WGKeysGenerate() error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.WireGuardGenerateNewKeys{}
	var resp types.EmptyResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	return nil
}

// WGKeysRotationInterval changes WG keys rotation interval
func (c *Client) WGKeysRotationInterval(uinxTimeInterval int64) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.WireGuardSetKeysRotationInterval{Interval: uinxTimeInterval}
	var resp types.EmptyResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	return nil
}

// PingServers changes WG keys rotation interval
func (c *Client) PingServers() (pingResults []types.PingResultType, err error) {
	if err := c.ensureConnected(); err != nil {
		return pingResults, err
	}

	req := types.PingServers{RetryCount: 4, TimeOutMs: 6000}
	var resp types.PingServersResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return pingResults, err
	}

	return resp.PingResults, nil
}

// SetManualDNS - sets manual DNS for current VPN connection
func (c *Client) SetManualDNS(dns string) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.SetAlternateDns{DNS: dns}
	var resp types.SetAlternateDNSResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	if resp.IsSuccess == false {
		return fmt.Errorf("DNS not changed")
	}

	return nil
}
