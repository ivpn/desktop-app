//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
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
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/obfsproxy"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	service_types "github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/version"
	"github.com/ivpn/desktop-app/daemon/vpn"
	"golang.org/x/crypto/pbkdf2"
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

	_paranoidModeSecret            string
	_paranoidModeSecretRequestFunc func(*Client) (string, error)

	_printFunc func(string)
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

func paranoidModeSecretHash(secret string) string {
	if len(secret) <= 0 {
		return ""
	}
	hash := pbkdf2.Key([]byte(secret), []byte(""), 4096, 64, sha256.New)
	return base64.StdEncoding.EncodeToString(hash)
}

func (c *Client) InitSetParanoidModeSecret(secret string) {
	c._paranoidModeSecret = paranoidModeSecretHash(secret)
}
func (c *Client) InitSetParanoidModeSecretHash(secretHash string) {
	c._paranoidModeSecret = secretHash
}

func (c *Client) SetParanoidModeSecretRequestFunc(f func(*Client) (string, error)) {
	c._paranoidModeSecretRequestFunc = f
}

func (c *Client) SetPrintFunc(f func(string)) {
	c._printFunc = f
}

// SendHello - send initial message and get current status
func (c *Client) SendHello() (helloResponse types.HelloResp, err error) {
	return c.SendHelloEx(false)
}

func (c *Client) SendHelloEx(isSendResponseToAllClients bool) (helloResponse types.HelloResp, err error) {
	if err := c.ensureConnected(); err != nil {
		return helloResponse, err
	}

	ver := version.Version()
	if ver == "" {
		ver = "unknown"
	}
	helloReq := types.Hello{
		Secret:                   c._secret,
		ClientType:               types.ClientCli,
		GetStatus:                true,
		Version:                  ver + ": CLI",
		SendResponseToAllClients: isSendResponseToAllClients,
	}

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

	var resp types.EmptyResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	return nil
}

func (c *Client) SetObfsProxy(cfg obfsproxy.Config) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.SetObfsProxy{ObfsproxyConfig: cfg}
	var resp types.EmptyResp
	if err := c.sendRecv(&req, &resp); err != nil {
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
	req := types.KillSwitchSetAllowLAN{AllowLAN: allow}
	var resp types.EmptyResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	return nil
}

// FirewallAllowLan set configuration 'firewall exceptions' (comma separated list of IP addresses/masks in format: x.x.x.x[/xx])
func (c *Client) FirewallSetUserExceptions(exceptions string) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	// changing killswitch configuration
	req := types.KillSwitchSetUserExceptions{UserExceptions: exceptions, FailOnParsingError: true}
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
	req := types.KillSwitchSetAllowLANMulticast{AllowLANMulticast: allow}
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

// GetSplitTunnelStatus requests the Split-Tunnelling configuration
func (c *Client) GetSplitTunnelStatus() (cfg types.SplitTunnelStatus, err error) {
	if err := c.ensureConnected(); err != nil {
		return cfg, err
	}

	// requesting status
	req := types.SplitTunnelGetStatus{}
	if err := c.sendRecv(&req, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// SetSplitTunnelConfig sets the split-tunnelling configuration
func (c *Client) SetSplitTunnelConfig(isEnable, reset bool) (err error) {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.SplitTunnelSetConfig{IsEnabled: isEnable, Reset: reset}
	resp := types.SplitTunnelStatus{}
	if _, _, err := c.sendRecvAny(&req, &resp); err != nil {
		return err
	}

	return nil
}

func (c *Client) SplitTunnelAddApp(execCmd string) (isRequiredToExecuteCommand bool, retErr error) {
	if err := c.ensureConnected(); err != nil {
		return false, err
	}

	// Description of Split Tunneling commands sequence to run the application:
	//	[client]					          [daemon]
	//	SplitTunnelAddApp		    ->
	//							            <-	windows:	types.EmptyResp (success)
	//							            <-	linux:		types.SplitTunnelAddAppCmdResp (some operations required on client side)
	//	<windows: done>
	// 	<execute shell command: types.SplitTunnelAddAppCmdResp.CmdToExecute and get PID>
	//  SplitTunnelAddedPidInfo	->
	// 							            <-	types.EmptyResp (success)

	var respEmpty types.EmptyResp
	var respAppCmdResp types.SplitTunnelAddAppCmdResp
	if val, ok := os.LookupEnv("IVPN_STARTED_BY_PARENT"); !ok || val != "IVPN_UI" {
		// If the CLI was started by IVPN UI - skip sending 'SplitTunnelAddApp'
		// It is already done by IVPN UI

		req := types.SplitTunnelAddApp{Exec: execCmd}
		_, _, err := c.sendRecvAnyEx(&req, false, &respEmpty, &respAppCmdResp)
		if err != nil {
			return false, err
		}

		if len(respEmpty.Command) > 0 {
			// success. No additional operations required
			return false, nil
		}

		if len(respAppCmdResp.Command) <= 0 {
			return false, fmt.Errorf("unexpected response from the daemon")
		}
	}

	if respAppCmdResp.IsAlreadyRunning {
		warningMes := respAppCmdResp.IsAlreadyRunningMessage
		if len(warningMes) <= 0 {
			// Note! Normally, this message will be never used. The text will come from daemon in 'IsAlreadyRunningMessage'
			warningMes = "It appears the application is already running.\nSome applications must be closed before launching them in the Split Tunneling environment or they may not be excluded from the VPN tunnel."
		}
		fmt.Println("WARNING! " + warningMes)

		fmt.Print("Do you really want to launch the command? [y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		yn, _ := reader.ReadString('\n')
		yn = strings.TrimSuffix(yn, "\n")
		yn = strings.TrimSuffix(yn, "\r")
		if yn == "" {
			yn = "yes"
			fmt.Println(yn)
		}
		yn = strings.ToUpper(yn)
		if yn != "Y" && yn != "YES" {
			return false, fmt.Errorf("canceled")
		}
	}

	// register new PID and inform that command must be executed
	reqAddedePid := types.SplitTunnelAddedPidInfo{Pid: os.Getpid(), Exec: execCmd, CmdToExecute: strings.Join(os.Args[:], " ")}
	if err := c.sendRecv(&reqAddedePid, &respEmpty); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Client) SplitTunnelRemoveApp(cmdOrPid string) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	pid := 0
	cmd := ""

	if p, err := strconv.Atoi(cmdOrPid); err == nil {
		pid = p
	} else {
		cmd = cmdOrPid
	}

	req := types.SplitTunnelRemoveApp{Exec: cmd, Pid: pid}
	var resp types.EmptyResp
	if err := c.sendRecv(&req, &resp); err != nil {
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

// GetServersForceUpdate gets servers list (skip cache; load data from backend)
func (c *Client) GetServersForceUpdate() (apitypes.ServersInfoResponse, error) {
	if err := c.ensureConnected(); err != nil {
		return apitypes.ServersInfoResponse{}, err
	}

	req := types.GetServers{
		RequestServersUpdate: true,
	}
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

func (c *Client) Pause(durationSec uint32) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	if durationSec > 0 {
		req := types.PauseConnection{Duration: durationSec}
		var resp types.ConnectedResp
		if err := c.sendRecv(&req, &resp); err != nil {
			return err
		}
	} else {
		req := types.ResumeConnection{}
		var resp types.EmptyResp
		if err := c.sendRecv(&req, &resp); err != nil {
			return err
		}
	}
	return nil
}

// PingServers
func (c *Client) PingServers(vpnTypePrioritized *vpn.Type) (pingResults []types.PingResultType, err error) {
	if err := c.ensureConnected(); err != nil {
		return pingResults, err
	}

	vpnTypePrioritization := false
	var vpnType vpn.Type
	if vpnTypePrioritized != nil {
		vpnType = *vpnTypePrioritized
		vpnTypePrioritization = true
	}
	// hosts for this VPN type will be pinged first (only if VpnTypePrioritization == true)

	req := types.PingServers{
		TimeOutMs:             6000,
		SkipSecondPhase:       true,
		VpnTypePrioritized:    vpnType,
		VpnTypePrioritization: vpnTypePrioritization,
	}
	var resp types.PingServersResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return pingResults, err
	}

	return resp.PingResults, nil
}

// SetManualDNS - sets manual DNS for current VPN connection
func (c *Client) SetManualDNS(dnsCfg dns.DnsSettings, antiTracker service_types.AntiTrackerMetadata) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.SetAlternateDns{Dns: dnsCfg, AntiTracker: antiTracker}
	var resp types.SetAlternateDNSResp
	if err := c.sendRecv(&req, &resp); err != nil {
		return err
	}

	if !resp.IsSuccess {
		if len(resp.ErrorMessage) > 0 {
			return fmt.Errorf("DNS not changed: " + resp.ErrorMessage)
		} else {
			return fmt.Errorf("DNS not changed")
		}
	}

	return nil
}

// SetParanoidModePassword - set password for ParanoidMode (empty string -> disable ParanoidMode)
func (c *Client) SetParanoidModePassword(secret string) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.ParanoidModeSetPasswordReq{NewSecret: paranoidModeSecretHash(secret)}
	var resp types.HelloResp
	// Waiting for HelloResp (ignoring command index) or for ErrorResp (not ignoring command index)
	if _, _, err := c.sendRecvAny(&req, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) SetUserPreferences(upref preferences.UserPreferences) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.SetUserPreferences{UserPrefs: upref}
	var resp types.SettingsResp
	if _, _, err := c.sendRecvAny(&req, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) GetWiFiCurrentNetwork() (types.WiFiCurrentNetworkResp, error) {
	var resp types.WiFiCurrentNetworkResp
	if err := c.ensureConnected(); err != nil {
		return resp, err
	}

	req := types.WiFiCurrentNetwork{}
	if _, _, err := c.sendRecvAny(&req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func (c *Client) SetWiFiSettings(params preferences.WiFiParams) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	req := types.WiFiSettings{Params: params}
	var resp types.EmptyResp
	if _, _, err := c.sendRecvAny(&req, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) SetDefConnectionParams(params types.ConnectSettings) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	var resp types.EmptyResp
	if _, _, err := c.sendRecvAny(&params, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) GetDefConnectionParams() (types.ConnectSettings, error) {
	if err := c.ensureConnected(); err != nil {
		return types.ConnectSettings{}, err
	}

	var resp types.ConnectSettings
	_, _, err := c.sendRecvAny(&types.ConnectSettingsGet{}, &resp)
	return resp, err
}
