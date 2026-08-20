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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/cli/helpers"
	apitypes "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/protocol/ivpnclient"
	ipc "github.com/ivpn/desktop-app/daemon/protocol/ivpnclient"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
	"github.com/ivpn/desktop-app/daemon/service/dns"
	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	service_types "github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/version"
	"github.com/ivpn/desktop-app/daemon/vpn"
	"golang.org/x/crypto/pbkdf2"
)

// Client for IVPN daemon
type Client struct {
	_locker        sync.RWMutex
	_client        *ipc.Client
	_helloResponse types.HelloResp
	_printFunc     func(string)
}

type ivpnClientLogger struct{}

func (ivpnClientLogger) Info(v ...interface{})  { logger.Info(v...) }
func (ivpnClientLogger) Error(v ...interface{}) { logger.Error(v...) }

// CreateClient initializing new client for IVPN daemon
func CreateClient(
	paranoidModeSecretRequestFunc ipc.ParanoidModeSecretRequestFunc,
	printFunc func(text string)) (*Client, error) {

	ver := version.Version()
	if ver == "" {
		ver = "unknown"
	}

	ivpnClient, err := ivpnclient.NewClient(
		paranoidModeSecretRequestFunc,
		ivpnClientLogger{},
		time.Second*60*3,
		ipc.ClientInfo{
			Type:    ipc.ClientCli,
			Name:    "CLI",
			Version: ver,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create IVPN client: %w", err)
	}

	return &Client{_client: ivpnClient, _printFunc: printFunc}, nil
}

// Connect is connecting to daemon
func (c *Client) Connect() (err error) {
	logger.Info("Connecting...")
	if err := c._client.Connect(time.Second * 5); err != nil {
		return fmt.Errorf("failed to connect to IVPN daemon (does IVPN daemon/service running?): %w", err)
	}
	logger.Info("Connected")

	// Set handler for 'HelloResp' message to update latest HelloResp
	c._client.SetMessageEventHandler("HelloResp", func(messageName string, messageData string) {
		var hr types.HelloResp
		if err := json.Unmarshal([]byte(messageData), &hr); err == nil {
			c._locker.Lock()
			c._helloResponse = hr
			c._locker.Unlock()
			// If we are running in privileged environment AND if daemon informed us about secret file - read it
			// It gives us possibility to bypass EAA (if enabled)
			if hr.ParanoidMode.IsEnabled && helpers.CheckIsAdmin() {
				if secret, err := os.ReadFile(platform.ParanoidModeSecretFile()); err == nil {
					c._client.SetParanoidModeSecret(string(secret))
				}
			}
		}
	})

	// Subscribe for 'ErrorRespDelayed' messages to be able to print delayed error messages from a daemon (if any)
	if c._printFunc != nil {
		messageName := ipc.GetTypeName(ipc.ErrorRespDelayed{})
		c._client.SetMessageEventHandler(messageName, func(messageName string, messageData string) {
			var errDelayed ipc.ErrorRespDelayed
			if err := json.Unmarshal([]byte(messageData), &errDelayed); err == nil {
				c._printFunc("IVPN daemon notifies of an error that occurred earlier: " + errDelayed.ErrorMessage + "\n")
			}
		})
	}

	// Send initial Hello message (required to start communication)
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

func (c *Client) InitSetParanoidModeSecret(pass string) {
	c._client.SetParanoidModeSecretPlainText(pass)
}
func (c *Client) InitSetParanoidModeSecretHash(secretHash string) {
	c._client.SetParanoidModeSecret(secretHash)
}

// SendHello - send initial message and get current status
func (c *Client) SendHello() (helloResponse types.HelloResp, err error) {
	return c.SendHelloEx(false)
}

func (c *Client) SendHelloEx(isSendResponseToAllClients bool) (helloResponse types.HelloResp, err error) {
	helloReq := c._client.InitHelloRequest()
	helloReq.SendResponseToAllClients = isSendResponseToAllClients

	if err := c._client.SendRecvTimeOut(&helloReq, &helloResponse, time.Second*7); err != nil {
		if _, ok := errors.Unwrap(err).(ipc.ResponseTimeout); ok {
			return helloResponse, fmt.Errorf("Failed to send 'Hello' request: %w", err)
		}
		return helloResponse, fmt.Errorf("Failed to send 'Hello' request: %w", err)
	}

	c._locker.Lock()
	defer c._locker.Unlock()
	c._helloResponse = helloResponse

	return c._helloResponse, nil
}

// GetHelloResponse returns initialization response from daemon
func (c *Client) GetHelloResponse() types.HelloResp {
	c._locker.RLock()
	defer c._locker.RUnlock()
	return c._helloResponse
}

// SessionNew creates new session
func (c *Client) SessionNew(accountID string, forceLogin bool, the2FA string) (resp types.SessionNewResp, err error) {
	req := types.SessionNew{AccountID: accountID, ForceLogin: forceLogin, Confirmation2FA: the2FA}

	if err := c._client.SendRecv(&req, &resp); err != nil {
		return resp, err
	}

	if len(resp.Session.Session) <= 0 {
		return resp, fmt.Errorf("[%d] %s", resp.APIStatus, resp.APIErrorMessage)
	}

	return resp, nil
}

// SessionDelete remove session
func (c *Client) SessionDelete(needToDisableFirewall, resetAppSettingsToDefaults, isCanDeleteSessionLocally bool) error {
	req := types.SessionDelete{
		NeedToDisableFirewall:     needToDisableFirewall,
		NeedToResetSettings:       resetAppSettingsToDefaults,
		IsCanDeleteSessionLocally: isCanDeleteSessionLocally}

	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

// SessionStatus get session status
func (c *Client) SessionStatus() (ret types.SessionStatusResp, err error) {
	req := types.SessionStatus{}
	var resp types.SessionStatusResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return ret, err
	}
	return resp, nil
}

// SetPreferences sends config parameter to daemon
// TODO: avoid using keys as a strings
func (c *Client) SetPreferences(key, value string) error {
	req := types.SetPreference{Key: key, Value: value}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

// FirewallSet change firewall state
func (c *Client) FirewallSet(isOn bool) error {
	// changing killswitch state
	req := types.KillSwitchSetEnabled{IsEnabled: isOn}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
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
	// changing killswitch Persistent state
	req := types.KillSwitchSetIsPersistent{IsPersistent: isOn}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
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
	// changing kill-switch configuration
	req := types.KillSwitchSetAllowLAN{AllowLAN: allow}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

// FirewallAllowLan set configuration 'firewall exceptions' (comma separated list of IP addresses/masks in format: x.x.x.x[/xx])
func (c *Client) FirewallSetUserExceptions(exceptions string) error {
	// changing kill-switch configuration
	req := types.KillSwitchSetUserExceptions{UserExceptions: exceptions, FailOnParsingError: true}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

// FirewallAllowApiServers set configuration 'Allow access to IVPN servers when Firewall is enabled'
func (c *Client) FirewallAllowApiServers(allow bool) error {
	// changing kill-switch configuration
	req := types.KillSwitchSetAllowApiServers{IsAllowApiServers: allow}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

// FirewallAllowLanMulticast set configuration 'allow LAN multicast'
func (c *Client) FirewallAllowLanMulticast(allow bool) error {
	// changing kill-switch configuration
	req := types.KillSwitchSetAllowLANMulticast{AllowLANMulticast: allow}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

// FirewallStatus get firewall state
func (c *Client) FirewallStatus() (state types.KillSwitchStatusResp, err error) {
	// requesting status
	statReq := types.KillSwitchGetStatus{}
	if err := c._client.SendRecv(&statReq, &state); err != nil {
		return state, err
	}
	return state, nil
}

// GetSplitTunnelStatus requests the Split-Tunnelling configuration
func (c *Client) GetSplitTunnelStatus() (cfg types.SplitTunnelStatus, err error) {
	// requesting status
	req := types.SplitTunnelGetStatus{}
	if err := c._client.SendRecv(&req, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// SetSplitTunnelConfig sets the split-tunnelling configuration
// Arguments:
//
//	isEnabled  bool - is ST enabled
//	isInversed bool - when inversed - only apps added to ST will use VPN connection, all other apps will use direct unencrypted connection
//	isAnyDns   bool - (only for Inverse Split Tunnel) When false: Allow only DNS servers specified by the IVPN application
//	isAllowWhenNoVpn bool - (only for Inverse Split Tunnel) Allow connectivity for Split Tunnel apps when VPN is disabled
//	reset      bool - reset ST config and disable ST (if enabled - all the rest paremeters are ignored)
func (c *Client) SetSplitTunnelConfig(isEnable, isInversed, isAnyDns, isAllowWhenNoVpn, reset bool) (err error) {
	req := types.SplitTunnelSetConfig{IsEnabled: isEnable, IsInversed: isInversed, IsAnyDns: isAnyDns, IsAllowWhenNoVpn: isAllowWhenNoVpn, Reset: reset}
	resp := ipc.EmptyResp{}
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) SplitTunnelAddApp(execCmd string) (isRequiredToExecuteCommand bool, retErr error) {
	// Description of Split Tunneling commands sequence to run the application:
	//	[client]					          [daemon]
	//	SplitTunnelAddApp		    ->
	//							            <-	windows:	types.EmptyResp (success)
	//							            <-	linux:		types.SplitTunnelAddAppCmdResp (some operations required on client side)
	//	<windows: done>
	// 	<execute shell command: types.SplitTunnelAddAppCmdResp.CmdToExecute and get PID>
	//  SplitTunnelAddedPidInfo	->
	// 							            <-	types.EmptyResp (success)

	var respEmpty ipc.EmptyResp
	var respAppCmdResp types.SplitTunnelAddAppCmdResp
	if val, ok := os.LookupEnv("IVPN_STARTED_BY_PARENT"); !ok || val != "IVPN_UI" {
		// If the CLI was started by IVPN UI - skip sending 'SplitTunnelAddApp'
		// It is already done by IVPN UI

		req := types.SplitTunnelAddApp{Exec: execCmd}
		err := c._client.SendRecvAnyEx(&req, false, &respEmpty, &respAppCmdResp)
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
	if err := c._client.SendRecv(&reqAddedePid, &respEmpty); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Client) SplitTunnelRemoveApp(cmdOrPid string) error {
	pid := 0
	cmd := ""

	if p, err := strconv.Atoi(cmdOrPid); err == nil {
		pid = p
	} else {
		cmd = cmdOrPid
	}

	req := types.SplitTunnelRemoveApp{Exec: cmd, Pid: pid}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

// GetServers gets servers list
func (c *Client) GetServers() (apitypes.ServersInfoResponse, error) {
	req := types.GetServers{}
	var resp types.ServerListResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return resp.VpnServers, err
	}
	return resp.VpnServers, nil
}

// GetServersForceUpdate gets servers list (skip cache; load data from backend)
func (c *Client) GetServersForceUpdate() (apitypes.ServersInfoResponse, error) {
	req := types.GetServers{
		RequestServersUpdate: true,
	}
	var resp types.ServerListResp

	if err := c._client.SendRecv(&req, &resp); err != nil {
		return resp.VpnServers, err
	}
	return resp.VpnServers, nil
}

// GetVPNState returns current VPN connection state
func (c *Client) GetVPNState() (vpn.State, types.ConnectedResp, error) {
	respConnected := types.ConnectedResp{}
	respDisconnected := ivpnclient.DisconnectedResp{}
	respState := types.VpnStateResp{}

	req := types.GetVPNState{}

	err := c._client.SendRecvAny(&req, &respConnected, &respDisconnected, &respState)
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

func (c *Client) GetTunnelHealthState() (ivpnclient.TunnelHealthResp, error) {
	req := ivpnclient.GetTunnelHealth{}
	var resp ivpnclient.TunnelHealthResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// DisconnectVPN disconnect active VPN connection
func (c *Client) DisconnectVPN() error {
	req := types.Disconnect{}
	respEmpty := ipc.EmptyResp{}
	respDisconnected := ivpnclient.DisconnectedResp{}

	err := c._client.SendRecvAny(&req, &respDisconnected, &respEmpty)
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
	respDisconnected := ivpnclient.DisconnectedResp{}

	err := c._client.SendRecvAny(&req, &respConnected, &respDisconnected)
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
	req := types.WireGuardGenerateNewKeys{}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

// WGKeysRotationInterval changes WG keys rotation interval
func (c *Client) WGKeysRotationInterval(uinxTimeInterval int64) error {
	req := types.WireGuardSetKeysRotationInterval{Interval: uinxTimeInterval}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) Pause(durationSec uint32) error {
	if durationSec > 0 {
		req := types.PauseConnection{Duration: durationSec}
		var resp types.ConnectedResp
		if err := c._client.SendRecv(&req, &resp); err != nil {
			return err
		}
	} else {
		req := types.ResumeConnection{}
		var resp ipc.EmptyResp
		if err := c._client.SendRecv(&req, &resp); err != nil {
			return err
		}
	}
	return nil
}

// PingServers
func (c *Client) PingServers(vpnTypePrioritized *vpn.Type) (pingResults []types.PingResultType, err error) {
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
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return pingResults, err
	}
	return resp.PingResults, nil
}

// SetManualDNS - sets manual DNS for current VPN connection
func (c *Client) SetManualDNS(dnsCfg dns.DnsSettings, antiTracker service_types.AntiTrackerMetadata) error {
	// converting internal types
	dnsArg := ipc.DnsSettings{Servers: make([]ipc.DnsServerConfig, len(dnsCfg.Servers))}
	for i, srv := range dnsCfg.Servers {
		dnsArg.Servers[i] = ipc.DnsServerConfig{
			Address:    srv.Address,
			Encryption: ipc.DnsEncryption(srv.Encryption), // converting to internal type
			Template:   srv.Template,
		}
	}
	antitrackerArg := ipc.AntiTrackerMetadata{
		Enabled:                  antiTracker.Enabled,
		Hardcore:                 antiTracker.Hardcore,
		AntiTrackerBlockListName: antiTracker.AntiTrackerBlockListName,
	}

	// sending request
	req := ipc.SetAlternateDns{Dns: dnsArg, AntiTracker: antitrackerArg}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

// SetParanoidModePassword - set password for ParanoidMode (empty string -> disable ParanoidMode)
func (c *Client) SetParanoidModePassword(secret string) error {
	req := types.ParanoidModeSetPasswordReq{NewSecret: paranoidModeSecretHash(secret)}
	var resp types.HelloResp
	// Waiting for HelloResp (ignoring command index) or for ErrorResp (not ignoring command index)
	if err := c._client.SendRecvAny(&req, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) SetUserPreferences(upref preferences.UserPreferences) error {
	req := types.SetUserPreferences{UserPrefs: upref}
	var resp types.SettingsResp
	if err := c._client.SendRecvAny(&req, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) GetWiFiCurrentNetwork() (types.WiFiCurrentNetworkResp, error) {
	var resp types.WiFiCurrentNetworkResp
	req := types.WiFiCurrentNetwork{}
	if err := c._client.SendRecvAny(&req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func (c *Client) SetWiFiSettings(params preferences.WiFiParams) error {
	req := types.WiFiSettings{Params: params}
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&req, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) SetDefConnectionParams(params types.ConnectSettings) error {
	var resp ipc.EmptyResp
	if err := c._client.SendRecv(&params, &resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) GetDefConnectionParams() (types.ConnectSettings, error) {
	var resp types.ConnectSettings
	err := c._client.SendRecvAny(&types.ConnectSettingsGet{}, &resp)
	return resp, err
}
