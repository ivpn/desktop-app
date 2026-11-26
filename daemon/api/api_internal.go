//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
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
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"path"
	"sync/atomic"
	"time"

	"github.com/ivpn/desktop-app/daemon/netinfo"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
)

// finalError is an error after which the request must not be repeated for the other API hosts (see doRequestAPIHost)
type finalError struct{ error }

// apiRequest describes a request to the IVPN API server (or to the update server).
// Zero values mean defaults.
type apiRequest struct {
	ipTypeRequired      types.RequiredIPProtocol // IPvAny (default): IPv4 is used first, IPv6 as a fallback
	host                string                   // "" or _apiHost: API server; _updateHost: update server
	urlPath             string                   // path of the request on the server
	method              string                   // default: "GET"
	contentType         string                   // value of the "Content-type" header; "": header is not added
	body                interface{}              // object to send (serialized to JSON); nil: no body
	timeoutMs           int                      // timeout for the full request; 0: default
	timeoutDialMs       int                      // timeout for the dial; 0: default (API server only)
	noRetryAfterConnect bool                     // see doRequestAPIHost (API server only)
}

func getURL(host string, urlpath string) string {
	return "https://" + path.Join(host, urlpath)
}

func getURL_IPHost(ip net.IP, isIPv6 bool, urlpath string) string {
	if isIPv6 {
		return "https://" + path.Join("["+ip.String()+"]", urlpath)
	}
	return "https://" + path.Join(ip.String(), urlpath)
}

func newRequest(urlPath string, method string, contentType string, body io.Reader) (*http.Request, error) {
	if len(method) == 0 {
		method = "GET"
	}

	req, err := http.NewRequest(method, urlPath, body)
	if err != nil {
		return nil, err
	}

	if len(contentType) > 0 {
		req.Header.Add("Content-type", contentType)
	}

	return req, nil
}

func findPinnedKey(certHashes []string, certBase64hash256 string) bool {
	for _, hash := range certHashes {
		if hash == certBase64hash256 {
			return true
		}
	}
	return false
}

type dialer func(network, addr string) (net.Conn, error)

func makeDialer(certHashes []string, serverName string, dialTimeout time.Duration) dialer {
	if len(certHashes) == 0 {
		log.Warning("No pinned certificates for ", _apiHost)
		return nil
	}

	return func(network, addr string) (net.Conn, error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("PANIC (API request): ", r)
				if err, ok := r.(error); ok {
					log.ErrorTrace(err)
				}
			}
		}()

		tlsConfig := &tls.Config{
			// NOTE: Can't use SSLv3 because of POODLE and BEAST
			// NOTE: Can't use TLSv1.0 because of POODLE and BEAST using CBC cipher
			// NOTE: Can't use TLSv1.1 because of RC4 cipher usage
			MinVersion: tls.VersionTLS12,
			ServerName: serverName,
		}

		c, err := tls.DialWithDialer(&net.Dialer{Timeout: dialTimeout}, network, addr, tlsConfig)

		if err != nil {
			return c, err
		}
		connstate := c.ConnectionState()
		var lastErr error = nil
		for _, peercert := range connstate.PeerCertificates {
			der, err := x509.MarshalPKIXPublicKey(peercert.PublicKey)
			if err != nil {
				lastErr = err
				continue
			}

			hash := sha256.Sum256(der)
			certBase64hash := base64.StdEncoding.EncodeToString(hash[:])

			if err != nil {
				log.Error(err)
			}

			if findPinnedKey(certHashes, certBase64hash) {
				return c, nil // Pinned Key found
			}

		}
		if lastErr != nil {
			return nil, fmt.Errorf("certificate check error: pinned certificate key not found: %w", lastErr)
		}
		return nil, fmt.Errorf("certificate check error: pinned certificate key not found")
	}
}

func (a *API) doRequest(r apiRequest) (resp *http.Response, err error) {
	connectivityChecker := a.connectivityChecker
	if connectivityChecker != nil {
		if err := connectivityChecker.IsConnectivityBlocked(); err != nil {
			return nil, err
		}
	}

	if len(r.host) == 0 || r.host == _apiHost {
		if r.ipTypeRequired != types.IPvAny {
			// The specific IP version required to use
			return a.doRequestAPIHost(r, false)
		} else {
			// No specific IP version required to use
			// Trying first to use IPv4, as fallback - try to use IPv6
			canUseDNS := true
			r.ipTypeRequired = types.IPv4
			resp4, err4 := a.doRequestAPIHost(r, canUseDNS)
			if err4 != nil && !errors.As(err4, new(finalError)) {
				// checking if IPv6 connectivity exists
				_, errIPv6 := netinfo.GetOutboundIP(true)
				if errIPv6 == nil && len(a.getAlternateIPs(true)) > 0 {
					log.Info("Failed to access API server using IPv4. Trying IPv6 ...")
					canUseDNS = false // we already tried to access using DNS. No sense to try it again
					r.ipTypeRequired = types.IPv6
					resp6, err6 := a.doRequestAPIHost(r, canUseDNS)
					if err6 == nil {
						return resp6, err6
					}
				}
			}
			return resp4, err4
		}

	} else if r.host == _updateHost {
		return a.doRequestUpdateHost(r)
	}
	return nil, fmt.Errorf("unknown host type")
}

// doRequestUpdateHost sends the request to the update server.
// (r.host, r.ipTypeRequired, r.timeoutDialMs and r.noRetryAfterConnect are not used)
func (a *API) doRequestUpdateHost(r apiRequest) (resp *http.Response, err error) {
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12, // seems, it is redundant (since we use custom DialTLS)
			ServerName: _updateHost,      // despite, we using custom DialTLS, we have to define ServerName (this avoids certificate verification problems, for example, when the request is going through a proxy server)
		},

		// using certificate key pinning
		DialTLS: makeDialer(UpdateIvpnHashes, _updateHost, 0),

		// the transport is created for one request: do not keep its connection open after the response
		DisableKeepAlives: true,
	}

	// configure http-client with preconfigured TLS transport
	timeout := _defaultRequestTimeout
	if r.timeoutMs > 0 {
		timeout = time.Millisecond * time.Duration(r.timeoutMs)
	}
	client := &http.Client{Transport: transCfg, Timeout: timeout}

	data := []byte{}
	if r.body != nil {
		data, err = json.Marshal(r.body)
		if err != nil {
			return nil, err
		}
	}

	bodyBuffer := bytes.NewBuffer(data)

	// try to access API server by host DNS
	req, err := newRequest(getURL(_updateHost, r.urlPath), r.method, r.contentType, bodyBuffer)
	if err != nil {
		return nil, err
	}

	resp, e := client.Do(req)
	if e != nil {
		log.Warning("Failed to access " + _updateHost)
		return resp, fmt.Errorf("unable to access IVPN repo server: %w", e)
	}

	return resp, nil
}

// doRequestAPIHost sends the request to the API server. A failed request is repeated for the next host
// (last good IP -> DNS name -> alternate IPs).
//
// r.ipTypeRequired - IP version to use: IPv4 or IPv6 (IPvAny is handled by doRequest; here it is treated as IPv4).
// r.host is not used.
//
// r.noRetryAfterConnect - do not repeat the request once a connection to a host was established: if the request fails
// after that, finalError is returned. It is intended for the requests with a big body and a long timeout:
//   - such a failure most likely means a slow connection, so the other hosts will not help;
//   - the server may have already received the request, so it must not be sent again.
func (a *API) doRequestAPIHost(r apiRequest, isCanUseDNS bool) (resp *http.Response, err error) {
	isIPv6 := r.ipTypeRequired == types.IPv6

	// timeout time for full request
	timeout := _defaultRequestTimeout
	if r.timeoutMs > 0 {
		timeout = time.Millisecond * time.Duration(r.timeoutMs)
	}
	// timeout for the dial
	timeoutDial := _defaultDialTimeout
	if r.timeoutDialMs > 0 {
		timeoutDial = time.Millisecond * time.Duration(r.timeoutDialMs)
	}
	if timeoutDial > timeout {
		timeoutDial = 0
	}

	// When trying to access API server by alternate IPs (not by DNS name)
	// we need to configure TLS to use api.ivpn.net hostname
	// (to avoid certificate errors)
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12, // seems, it is redundant (since we use custom DialTLS)
			ServerName: _apiHost,         // despite, we using custom DialTLS, we have to define ServerName (this avoids certificate verification problems, for example, when the request is going through a proxy server)
		},

		// using certificate key pinning
		DialTLS: makeDialer(APIIvpnHashes, _apiHost, timeoutDial),

		// the transport is created for one request: do not keep its connection open after the response
		DisableKeepAlives: true,
	}

	// configure http-client with preconfigured TLS transport
	client := &http.Client{Transport: transCfg, Timeout: timeout}

	data := []byte{}
	if r.body != nil {
		data, err = json.Marshal(r.body)
		if err != nil {
			return nil, err
		}
	}

	// Every attempt needs its own reader: a reader is consumed by the attempt which used it.
	// (A shared reader makes all attempts after the first one send a truncated or empty body.)
	newBody := func() io.Reader { return bytes.NewReader(data) }

	// doAttempt performs one attempt of the request.
	// isConnected - a connection to the host was established (tracked only when noRetryAfterConnect is set)
	var isConnected atomic.Bool
	doAttempt := client.Do
	if r.noRetryAfterConnect {
		trace := &httptrace.ClientTrace{GotConn: func(httptrace.GotConnInfo) { isConnected.Store(true) }}
		doAttempt = func(req *http.Request) (*http.Response, error) {
			return client.Do(req.WithContext(httptrace.WithClientTrace(req.Context(), trace)))
		}
	}

	// access API by last good IP (if defined)
	lastGoodIP := a.GetLastGoodAlternateIP(isIPv6)
	if lastGoodIP != nil {
		req, err := newRequest(getURL_IPHost(lastGoodIP, isIPv6, r.urlPath), r.method, r.contentType, newBody())
		if err != nil {
			return nil, err
		}

		resp, err := doAttempt(req)
		if err == nil {
			return resp, nil
		}
		log.Warning(fmt.Sprintf("Failed to access API server by last good IP: %v", err))
		if isConnected.Load() {
			return nil, finalError{toUserFacingAccessError(err)}
		}
	}

	// try to access API server by host DNS
	var firstResp *http.Response
	var firstErr error
	if isCanUseDNS {
		req, err := newRequest(getURL(_apiHost, r.urlPath), r.method, r.contentType, newBody())
		if err != nil {
			return nil, err
		}
		firstResp, firstErr = doAttempt(req)
		if firstErr == nil {
			return firstResp, firstErr
		}
		log.Warning("Failed to access " + _apiHost)
		if isConnected.Load() {
			return nil, finalError{toUserFacingAccessError(firstErr)}
		}
	}

	isLogNotificationPrinted := false

	// try to access API server by alternate IP
	ips := a.getAlternateIPs(isIPv6)
	for _, ip := range ips {
		if ip.Equal(lastGoodIP) {
			continue
		}
		if firstErr != nil && !isLogNotificationPrinted {
			isLogNotificationPrinted = true

			ipVerStr := ""
			if r.ipTypeRequired == types.IPv6 {
				ipVerStr = "(IPv6)"
			}
			log.Info(fmt.Sprintf("Trying to use alternate API IPs %s...", ipVerStr))
		}

		req, err := newRequest(getURL_IPHost(ip, isIPv6, r.urlPath), r.method, r.contentType, newBody())
		if err != nil {
			return nil, err
		}
		resp, err := doAttempt(req)

		if err != nil {
			if isConnected.Load() {
				return nil, finalError{toUserFacingAccessError(err)}
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		// save last good IP
		a.SetLastGoodAlternateIP(ip)

		log.Info("Success!")
		return resp, err
	}

	return nil, toUserFacingAccessError(firstErr)
}

// toUserFacingAccessError converts the failure of an API request (err) into the error to show to the user.
func toUserFacingAccessError(err error) error {
	// Timeout or failed connection: return an error which is understandable for the user (it is shown in the UI as is).
	// The technical details are saved in the log.
	// All other errors (e.g. certificate check errors) are returned unchanged.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		log.Warning(fmt.Sprintf("Unable to access IVPN API server (timeout): %v", err))
		return errors.New("unable to access IVPN API server (the request timed out). Possible reasons: a slow or unstable internet connection, or access to the server is blocked on your network")
	}
	// (the "dial" operation fails when the host name cannot be resolved or the connection cannot be established)
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		log.Warning(fmt.Sprintf("Unable to access IVPN API server (connection failed): %v", err))
		return errors.New("unable to access IVPN API server (connection failed). Please check your internet connection. Access to the server may also be blocked by a firewall or by your network")
	}

	return fmt.Errorf("unable to access IVPN API server: %w", err)
}

func (a *API) requestRaw(r apiRequest) (responseData []byte, httpResp *http.Response, err error) {
	resp, err := a.doRequest(r)
	if err != nil {
		return nil, nil, fmt.Errorf("API request failed: %w", err)
	}

	if resp.StatusCode != 200 {
		log.Debug(fmt.Sprintf("API response: (HTTP status code %d); status=%s", resp.StatusCode, resp.Status))
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read API HTTP response body: %w", err)
	}

	return body, resp, nil
}

func (a *API) request(r apiRequest, responseObject interface{}) error {
	body, _, err := a.requestRaw(r)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, responseObject); err != nil {
		return fmt.Errorf("failed to deserialize API response: %w", err)
	}

	return nil
}
