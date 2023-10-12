//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
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
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/logger"
	protocolTypes "github.com/ivpn/desktop-app/daemon/protocol/types"
)

// API URLs
const (
	_defaultRequestTimeout = time.Second * 10 // full request time (for each request)
	_defaultDialTimeout    = time.Second * 5  // time for the dial to the API server (for each request)
	_apiHost               = "api.ivpn.net"
	_updateHost            = "repo.ivpn.net"
	_serversPath           = "v5/servers.json"
	_apiPathPrefix         = "v4"
	_sessionNewPath        = _apiPathPrefix + "/session/new"
	_sessionStatusPath     = _apiPathPrefix + "/session/status"
	_sessionDeletePath     = _apiPathPrefix + "/session/delete"
	_wgKeySetPath          = _apiPathPrefix + "/session/wg/set"
	_geoLookupPath         = _apiPathPrefix + "/geo-lookup"
)

// Alias - alias description of API request (can be requested by UI client)
type Alias struct {
	host string
	path string
	// If isArcIndependent!=true, the path will be updated: the "_<architecture>" will be added to filename
	// (see 'DoRequestByAlias()' for details)
	// Example:
	//		The "updateInfo_macOS" on arm64 platform will use file "/macos/update_arm64.json" (NOT A "/macos/update.json")
	isArcIndependent bool
}

// APIAliases - aliases of API requests (can be requested by UI client)
// NOTE: the aliases bellow are only for amd64 architecture!!!
// If isArcIndependent!=true: Filename construction for non-amd64 architectures: filename_<architecture>.<extensions>
// (see 'DoRequestByAlias()' for details)
// Example:
//
//	The "updateInfo_macOS" on arm64 platform will use file "/macos/update_arm64.json" (NOT A "/macos/update.json")
const (
	GeoLookupApiAlias string = "geo-lookup"
)

var APIAliases = map[string]Alias{
	GeoLookupApiAlias: {host: _apiHost, path: _geoLookupPath},

	"updateInfo_Linux":   {host: _updateHost, path: "/stable/_update_info/update.json"},
	"updateSign_Linux":   {host: _updateHost, path: "/stable/_update_info/update.json.sign.sha256.base64"},
	"updateInfo_macOS":   {host: _updateHost, path: "/macos/update.json"},
	"updateSign_macOS":   {host: _updateHost, path: "/macos/update.json.sign.sha256.base64"},
	"updateInfo_Windows": {host: _updateHost, path: "/windows/update.json"},
	"updateSign_Windows": {host: _updateHost, path: "/windows/update.json.sign.sha256.base64"},

	"updateInfo_manual_Linux":   {host: _updateHost, path: "/stable/_update_info/update_manual.json"},
	"updateSign_manual_Linux":   {host: _updateHost, path: "/stable/_update_info/update_manual.json.sign.sha256.base64"},
	"updateInfo_manual_macOS":   {host: _updateHost, path: "/macos/update_manual.json"},
	"updateSign_manual_macOS":   {host: _updateHost, path: "/macos/update_manual.json.sign.sha256.base64"},
	"updateInfo_manual_Windows": {host: _updateHost, path: "/windows/update_manual.json"},
	"updateSign_manual_Windows": {host: _updateHost, path: "/windows/update_manual.json.sign.sha256.base64"},

	"updateInfo_beta_Linux":   {host: _updateHost, path: "/stable/_update_info/update_beta.json"},
	"updateSign_beta_Linux":   {host: _updateHost, path: "/stable/_update_info/update_beta.json.sign.sha256.base64"},
	"updateInfo_beta_macOS":   {host: _updateHost, path: "/macos/update_beta.json"},
	"updateSign_beta_macOS":   {host: _updateHost, path: "/macos/update_beta.json.sign.sha256.base64"},
	"updateInfo_beta_Windows": {host: _updateHost, path: "/windows/update_beta.json"},
	"updateSign_beta_Windows": {host: _updateHost, path: "/windows/update_beta.json.sign.sha256.base64"},
}

var log *logger.Logger

func init() {
	log = logger.NewLogger("api")
}

// IConnectivityInfo information about connectivity
type IConnectivityInfo interface {
	// IsConnectivityBlocked - returns nil if connectivity NOT blocked
	IsConnectivityBlocked() (err error)
}

type geolookup struct {
	mutex     sync.Mutex
	isRunning bool
	done      chan struct{}

	location types.GeoLookupResponse
	response []byte
	err      error
}

// API contains data about IVPN API servers
type API struct {
	mutex                 sync.Mutex
	alternateIPsV4        []net.IP
	lastGoodAlternateIPv4 net.IP
	alternateIPsV6        []net.IP
	lastGoodAlternateIPv6 net.IP
	connectivityChecker   IConnectivityInfo

	// last geolookups result
	geolookupV4 geolookup
	geolookupV6 geolookup
}

// CreateAPI creates new API object
func CreateAPI() (*API, error) {
	return &API{}, nil
}

func (a *API) SetConnectivityChecker(connectivityChecker IConnectivityInfo) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.connectivityChecker = connectivityChecker
}

// IsAlternateIPsInitialized - checks if the alternate IP initialized
func (a *API) IsAlternateIPsInitialized(IPv6 bool) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if IPv6 {
		return len(a.alternateIPsV6) > 0
	}
	return len(a.alternateIPsV4) > 0
}

func (a *API) GetLastGoodAlternateIP(IPv6 bool) net.IP {
	if IPv6 {
		if a.lastGoodAlternateIPv6.To4() != nil {
			return nil // something wrong here: lastGoodAlternateIPv6 must be IPv6 address
		}
		return a.lastGoodAlternateIPv6
	}
	return a.lastGoodAlternateIPv4.To4()
}

// SetLastGoodAlternateIP - save last good alternate IP address of API server
// It keeps IPv4 and IPv6 addresses separately
func (a *API) SetLastGoodAlternateIP(ip net.IP) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	isIp6Addr := ip.To4() == nil
	if isIp6Addr {
		a.lastGoodAlternateIPv6 = ip
		return
	}
	a.lastGoodAlternateIPv4 = ip
}

func (a *API) getAlternateIPs(IPv6 bool) []net.IP {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if IPv6 {
		return a.alternateIPsV6
	}
	return a.alternateIPsV4
}

// SetAlternateIPs save info about alternate servers IP addresses
func (a *API) SetAlternateIPs(IPv4List []string, IPv6List []string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.doSetAlternateIPs(false, IPv4List)
	a.doSetAlternateIPs(true, IPv6List)
	return nil
}

func (a *API) doSetAlternateIPs(IPv6 bool, IPs []string) error {
	if len(IPs) == 0 {
		log.Warning("Unable to set alternate API IP list. List is empty")
	}

	lastGoodIP := a.GetLastGoodAlternateIP(IPv6)

	ipList := make([]net.IP, 0, len(IPs))

	isLastIPExists := false
	for _, ipStr := range IPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		ipList = append(ipList, ip)

		if ip.Equal(lastGoodIP) {
			isLastIPExists = true
		}
	}

	if !isLastIPExists {
		if IPv6 {
			a.lastGoodAlternateIPv6 = nil
		} else {
			a.lastGoodAlternateIPv4 = nil
		}
	}

	// set new alternate IP list
	if IPv6 {
		a.alternateIPsV6 = ipList
	} else {
		a.alternateIPsV4 = ipList
	}

	return nil
}

// DownloadServersList - download servers list form API IVPN server
func (a *API) DownloadServersList() (*types.ServersInfoResponse, error) {
	servers := new(types.ServersInfoResponse)
	if err := a.request("", _serversPath, "GET", "", nil, servers); err != nil {
		return nil, err
	}

	// save info about alternate API hosts
	a.SetAlternateIPs(servers.Config.API.IPAddresses, servers.Config.API.IPv6Addresses)
	return servers, nil
}

// DoRequestByAlias do API request (by API endpoint alias). Returns raw data of response
func (a *API) DoRequestByAlias(apiAlias string, ipTypeRequired protocolTypes.RequiredIPProtocol) (responseData []byte, err error) {
	// For geolookup requests we have specific function
	if apiAlias == GeoLookupApiAlias {
		if ipTypeRequired != protocolTypes.IPv4 && ipTypeRequired != protocolTypes.IPv6 {
			return nil, fmt.Errorf("geolookup request failed: IP version not defined")
		}
		_, responseData, err = a.GeoLookup(0, ipTypeRequired)
		return responseData, err
	}

	// get connection info by API alias
	alias, ok := APIAliases[apiAlias]
	if !ok {
		return nil, fmt.Errorf("unexpected request alias")
	}

	if !alias.isArcIndependent {
		// If isArcIndependent!=true, the path will be updated: the "_<architecture>" will be added to filename
		// Example:
		//		The "updateInfo_macOS" on arm64 platform will use file "/macos/update_arm64.json" (NOT A "/macos/update.json"!)
		if runtime.GOARCH != "amd64" {
			extIdx := strings.Index(alias.path, ".")
			if extIdx > 0 {
				newPath := alias.path[:extIdx] + "_" + runtime.GOARCH + alias.path[extIdx:]
				alias.path = newPath
			}
		}
	}

	return a.requestRaw(ipTypeRequired, alias.host, alias.path, "", "", nil, 0, 0)
}

// SessionNew - try to register new session
func (a *API) SessionNew(accountID string, wgPublicKey string, kemKeys types.KemPublicKeys, forceLogin bool, captchaID string, captcha string, confirmation2FA string) (
	*types.SessionNewResponse,
	*types.SessionNewErrorLimitResponse,
	*types.APIErrorResponse,
	string, // RAW response
	error) {

	var successResp types.SessionNewResponse
	var errorLimitResp types.SessionNewErrorLimitResponse
	var apiErr types.APIErrorResponse

	rawResponse := ""

	request := &types.SessionNewRequest{
		AccountID:       accountID,
		PublicKey:       wgPublicKey,
		KemPublicKeys:   kemKeys,
		ForceLogin:      forceLogin,
		CaptchaID:       captchaID,
		Captcha:         captcha,
		Confirmation2FA: confirmation2FA}

	data, err := a.requestRaw(protocolTypes.IPvAny, "", _sessionNewPath, "POST", "application/json", request, 0, 0)
	if err != nil {
		return nil, nil, nil, rawResponse, err
	}

	rawResponse = string(data)

	// Check is it API error
	if err := json.Unmarshal(data, &apiErr); err != nil {
		return nil, nil, nil, rawResponse, fmt.Errorf("failed to deserialize API response: %w", err)
	}

	// success
	if apiErr.Status == types.CodeSuccess {
		if err := json.Unmarshal(data, &successResp); err != nil {
			return nil, nil, nil, rawResponse, fmt.Errorf("failed to deserialize API response: %w", err)
		}

		return &successResp, nil, &apiErr, rawResponse, nil
	}

	// Session limit check
	if apiErr.Status == types.CodeSessionsLimitReached {
		if err := json.Unmarshal(data, &errorLimitResp); err != nil {
			return nil, nil, nil, rawResponse, fmt.Errorf("failed to deserialize API response: %w", err)
		}
		return nil, &errorLimitResp, &apiErr, rawResponse, types.CreateAPIError(apiErr.Status, apiErr.Message)
	}

	return nil, nil, &apiErr, rawResponse, types.CreateAPIError(apiErr.Status, apiErr.Message)
}

// SessionStatus - get session status
func (a *API) SessionStatus(session string) (
	*types.ServiceStatusAPIResp,
	*types.APIErrorResponse,
	error) {

	var resp types.SessionStatusResponse
	var apiErr types.APIErrorResponse

	request := &types.SessionStatusRequest{Session: session}

	data, err := a.requestRaw(protocolTypes.IPvAny, "", _sessionStatusPath, "POST", "application/json", request, 0, 0)
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

	return nil, &apiErr, types.CreateAPIError(apiErr.Status, apiErr.Message)
}

// SessionDelete - remove session
func (a *API) SessionDelete(session string) error {
	request := &types.SessionDeleteRequest{Session: session}
	resp := &types.APIErrorResponse{}
	if err := a.request("", _sessionDeletePath, "POST", "application/json", request, resp); err != nil {
		return err
	}
	if resp.Status != types.CodeSuccess {
		return types.CreateAPIError(resp.Status, resp.Message)
	}
	return nil
}

// WireGuardKeySet - update WG key
func (a *API) WireGuardKeySet(session string, newPublicWgKey string, activePublicWgKey string, kemKeys types.KemPublicKeys) (responseObj types.SessionsWireGuardResponse, err error) {
	request := &types.SessionWireGuardKeySetRequest{
		Session:            session,
		PublicKey:          newPublicWgKey,
		ConnectedPublicKey: activePublicWgKey,
		KemPublicKeys:      kemKeys,
	}

	resp := types.SessionsWireGuardResponse{}

	if err := a.request("", _wgKeySetPath, "POST", "application/json", request, &resp); err != nil {
		return resp, err
	}

	if resp.Status != types.CodeSuccess {
		return resp, types.CreateAPIError(resp.Status, resp.Message)
	}

	return resp, nil
}

// GeoLookup gets geolocation
func (a *API) GeoLookup(timeoutMs int, ipTypeRequired protocolTypes.RequiredIPProtocol) (location *types.GeoLookupResponse, rawData []byte, retErr error) {
	// There could be multiple Geolookup requests at the same time.
	// It doesn't make sense to make multiple requests to the API.
	// The internal function below reduces the number of similar API calls.
	singletonFunc := func(ipType protocolTypes.RequiredIPProtocol) (*types.GeoLookupResponse, []byte, error) {
		// Each IP protocol has separate request
		var gl *geolookup
		if ipType == protocolTypes.IPv4 {
			gl = &a.geolookupV4
		} else if ipType == protocolTypes.IPv6 {
			gl = &a.geolookupV6
		} else {
			return nil, nil, fmt.Errorf("geolookup request failed: IP version not defined")
		}
		// Try to make API request (if not started yet). Only one API request allowed in the same time.
		func() {
			gl.mutex.Lock()
			defer gl.mutex.Unlock()
			// if API call is already running - do nosing, just wait for results
			if gl.isRunning {
				return
			}
			// mark: call is already running
			gl.isRunning = true
			gl.done = make(chan struct{})
			// do API call in routine
			go func() {
				defer func() {
					// API call finished
					gl.isRunning = false
					close(gl.done)
				}()
				gl.response, gl.err = a.requestRaw(ipType, "", _geoLookupPath, "GET", "", nil, timeoutMs, 0)
				if err := json.Unmarshal(gl.response, &gl.location); err != nil {
					gl.err = fmt.Errorf("failed to deserialize API response: %w", err)
				}
			}()
		}()
		// wait for API call result (for routine stop)
		<-gl.done
		return &gl.location, gl.response, gl.err
	}

	// request Geolocation info
	if ipTypeRequired != protocolTypes.IPvAny {
		location, rawData, retErr = singletonFunc(ipTypeRequired)
	} else {
		location, rawData, retErr = singletonFunc(protocolTypes.IPv4)
		if retErr != nil {
			location, rawData, retErr = singletonFunc(protocolTypes.IPv6)
		}
	}

	if retErr != nil {
		return nil, nil, retErr
	}
	return location, rawData, nil
}
