package protocol

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	apitypes "github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
	"github.com/ivpn/desktop-app-daemon/vpn"
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
		_defaultTimeout: time.Second * 30,
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

	helloReq := types.Hello{Secret: c._secret, KeepDaemonAlone: true, GetStatus: true, Version: "1.0"}

	if err := c.sendRecvTimeOut(&helloReq, &c._helloResponse, time.Second*5); err != nil {
		if _, ok := errors.Unwrap(err).(ResponseTimeout); ok {
			return fmt.Errorf("Failed to send 'Hello' request (does another instance of IVPN Client running?): %w", err)
		}
		return fmt.Errorf("Failed to send 'Hello' request: %w", err)
	}

	return nil
}

// GetHelloResponse returns initialisation response from daemon
func (c *Client) GetHelloResponse() types.HelloResp {
	return c._helloResponse
}

// SessionNew creates new session
func (c *Client) SessionNew(accountID string, forceLogin bool) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.SessionNew{AccountID: accountID, ForceLogin: forceLogin}
	var resp types.SessionNewResp

	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	if len(resp.Session.Session) <= 0 {
		return fmt.Errorf("[%d] %s", resp.APIStatus, resp.APIErrorMessage)
	}

	return nil
}

// SessionDelete remove session
func (c *Client) SessionDelete() error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.SessionDelete{}
	var resp types.EmptyResp

	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	return nil
}

// SessionStatus get session status
func (c *Client) SessionStatus() (ret types.SessionStatusResp, err error) {
	if err := c.ensureConnected(); err != nil {
		return ret, err
	}

	req := types.SessionStatus{}
	var resp types.SessionStatusResp

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

	data, cmdBase, err := c.sendRecvRaw(&req)
	if err != nil {
		return vpn.DISCONNECTED, respConnected, err
	}

	switch cmdBase.Command {
	case types.GetTypeName(respConnected):
		if err := deserialize(data, &respConnected); err != nil {
			return vpn.DISCONNECTED, respConnected, fmt.Errorf("response deserialisation failed: %w", err)
		}
		return vpn.CONNECTED, respConnected, nil

	case types.GetTypeName(respDisconnected):
		return vpn.DISCONNECTED, respConnected, nil

	case types.GetTypeName(respState):
		if err := deserialize(data, &respState); err != nil {
			return vpn.DISCONNECTED, respConnected, fmt.Errorf("response deserialisation failed: %w", err)
		}
		return respState.StateVal, respConnected, nil
	}

	return vpn.DISCONNECTED, respConnected, fmt.Errorf("failed to receive VPN state (not expected return type)")
}

// Disconnect disconnect active VPN connection
func (c *Client) Disconnect() error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.Disconnect{}
	respEmpty := types.EmptyResp{}
	respDisconnected := types.DisconnectedResp{}

	_, cmdBase, err := c.sendRecvRaw(&req)
	if err != nil {
		return err
	}

	if cmdBase.Command != types.GetTypeName(respEmpty) && cmdBase.Command != types.GetTypeName(respDisconnected) {
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

	data, cmdBase, err := c.sendRecvRaw(&req)
	if err != nil {
		return respConnected, err
	}

	switch cmdBase.Command {
	case types.GetTypeName(respConnected):
		if err := deserialize(data, &respConnected); err != nil {
			return respConnected, fmt.Errorf("response deserialisation failed: %w", err)
		}
		return respConnected, nil

	case types.GetTypeName(respDisconnected):
		if err := deserialize(data, &respDisconnected); err != nil {
			return respConnected, fmt.Errorf("response deserialisation failed: %w", err)
		}
		return respConnected, fmt.Errorf("%s", respDisconnected.ReasonDescription)
	}

	return respConnected, fmt.Errorf("connect request failed (not expected return type)")
}
