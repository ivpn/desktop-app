//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package api

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/logger"
)

// API URLs
const (
	_defaultRequestTimeout = time.Second * 15
	_apiHost               = "api.ivpn.net"
	_serversPath           = "v4/servers.json"
	_sessionNewPath        = "v4/session/new"
	_sessionStatusPath     = "v4/session/status"
	_sessionDeletePath     = "v4/session/delete"
	_wgKeySetPath          = "v4/session/wg/set"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("api")
}

// IConnectivityInfo information about connectivity
type IConnectivityInfo interface {
	IsConnectivityBlocked() bool
}

// API contains data about IVPN API servers
type API struct {
	mutex               sync.Mutex
	alternateIPs        []net.IP
	lastGoodAlternateIP net.IP
}

// CreateAPI creates new API object
func CreateAPI() (*API, error) {
	return &API{}, nil
}

// IsAlternateIPsInitialized - checks if the alternate IP initialized
func (a *API) IsAlternateIPsInitialized() bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	return len(a.alternateIPs) > 0
}

// SetAlternateIPs save info about alternate servers IP addresses
func (a *API) SetAlternateIPs(IPs []string) error {
	if len(IPs) == 0 {
		log.Warning("Unable to set alternate API IP list. List is empty")
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	ipList := make([]net.IP, 0, len(IPs))

	isLastIPExists := false
	for _, ipStr := range IPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		ipList = append(ipList, ip)

		if ip.Equal(a.lastGoodAlternateIP) {
			isLastIPExists = true
		}
	}

	if isLastIPExists == false {
		a.lastGoodAlternateIP = nil
	}

	// set new alternate IP list
	a.alternateIPs = ipList

	return nil
}

// DownloadServersList - download servers list form API IVPN server
func (a *API) DownloadServersList() (*types.ServersInfoResponse, error) {
	servers := new(types.ServersInfoResponse)
	if err := a.request(_serversPath, "GET", "", nil, servers); err != nil {
		return nil, err
	}

	// save info about alternate API hosts
	a.SetAlternateIPs(servers.Config.API.IPAddresses)
	return servers, nil
}

// SessionNew - try to register new session
func (a *API) SessionNew(accountID string, wgPublicKey string, forceLogin bool) (
	*types.SessionNewResponse,
	*types.SessionNewErrorLimitResponse,
	*types.APIErrorResponse,
	error) {

	var successResp types.SessionNewResponse
	var errorLimitResp types.SessionNewErrorLimitResponse
	var apiErr types.APIErrorResponse

	request := &types.SessionNewRequest{
		AccountID:  accountID,
		PublicKey:  wgPublicKey,
		ForceLogin: forceLogin}

	data, err := a.requestRaw(_sessionNewPath, "POST", "application/json", request)
	if err != nil {
		return nil, nil, nil, err
	}

	// Check is it API error
	if err := json.Unmarshal(data, &apiErr); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to deserialize API response: %w", err)
	}

	// success
	if apiErr.Status == types.CodeSuccess {
		if err := json.Unmarshal(data, &successResp); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to deserialize API response: %w", err)
		}
		return &successResp, nil, &apiErr, nil
	}

	// Session limit check
	if apiErr.Status == types.CodeSessionsLimitReached {
		if err := json.Unmarshal(data, &errorLimitResp); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to deserialize API response: %w", err)
		}
		return nil, &errorLimitResp, &apiErr, fmt.Errorf("API error: [%d] %s", apiErr.Status, apiErr.Message)
	}

	return nil, nil, &apiErr, fmt.Errorf("API error: [%d] %s", apiErr.Status, apiErr.Message)
}

// SessionStatus - get session status
func (a *API) SessionStatus(session string) (
	*types.ServiceStatusAPIResp,
	*types.APIErrorResponse,
	error) {

	var resp types.SessionStatusResponse
	var apiErr types.APIErrorResponse

	request := &types.SessionStatusRequest{Session: session}

	data, err := a.requestRaw(_sessionStatusPath, "POST", "application/json", request)
	if err != nil {
		return nil, nil, err
	}

	// Check is it API error
	if err := json.Unmarshal(data, &apiErr); err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize API response: %w", err)
	}

	// success
	if apiErr.Status == types.CodeSuccess {
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, nil, fmt.Errorf("failed to deserialize API response: %w", err)
		}
		return &resp.ServiceStatus, &apiErr, nil
	}

	return nil, &apiErr, fmt.Errorf("API error: [%d] %s", apiErr.Status, apiErr.Message)
}

// SessionDelete - remove session
func (a *API) SessionDelete(session string) error {
	request := &types.SessionDeleteRequest{Session: session}
	resp := &types.APIErrorResponse{}
	if err := a.request(_sessionDeletePath, "POST", "application/json", request, resp); err != nil {
		return err
	}
	if resp.Status != types.CodeSuccess {
		return fmt.Errorf("API error: [%d] %s", resp.Status, resp.Message)
	}
	return nil
}

// WireGuardKeySet - update WG key
func (a *API) WireGuardKeySet(session string, newPublicWgKey string, activePublicWgKey string) (localIP net.IP, err error) {

	request := &types.SessionWireGuardKeySetRequest{
		Session:            session,
		PublicKey:          newPublicWgKey,
		ConnectedPublicKey: activePublicWgKey}

	resp := &types.SessionsWireGuardResponse{}

	if err := a.request(_wgKeySetPath, "POST", "application/json", request, resp); err != nil {
		return nil, err
	}

	if resp.Status != types.CodeSuccess {
		return nil, fmt.Errorf("API error: [%d] %s", resp.Status, resp.Message)
	}

	localIP = net.ParseIP(resp.IPAddress)
	if localIP == nil {
		return nil, fmt.Errorf("failed to set WG key (failed to parse local IP in API response)")
	}

	return localIP, nil
}
