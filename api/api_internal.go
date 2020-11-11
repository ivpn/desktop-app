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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"path"
	"time"
)

func (a *API) getAlternateIPs() (lastGoodIP net.IP, ipList []net.IP) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	return a.lastGoodAlternateIP, a.alternateIPs
}

func (a *API) saveLastGoodAlternateIP(lastGoodIP net.IP) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.lastGoodAlternateIP = lastGoodIP
}

func getURL(host string, urlpath string) string {
	return "https://" + path.Join(host, urlpath)
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

func (a *API) doRequest(urlPath string, method string, contentType string, request interface{}, timeoutMs int) (resp *http.Response, err error) {
	lastIP, ips := a.getAlternateIPs()

	// When trying to access API server by alternate IPs (not by DNS name)
	// we need to configure TLS to use api.ivpn.net hostname
	// (to avoid certificate errors)
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{
			ServerName: _apiHost,
		},
	}
	// configure http-client with preconfigured TLS transport
	timeout := _defaultRequestTimeout
	if timeoutMs > 0 {
		timeout = time.Millisecond * time.Duration(timeoutMs)
	}
	client := &http.Client{Transport: transCfg, Timeout: timeout}

	data := []byte{}
	if request != nil {
		data, err = json.Marshal(request)
		if err != nil {
			return nil, err
		}
	}

	bodyBuffer := bytes.NewBuffer(data)

	// access API by last good IP (if defined)
	if lastIP != nil {
		req, err := newRequest(getURL(lastIP.String(), urlPath), method, contentType, bodyBuffer)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}
	}

	// try to access API server by host DNS
	req, err := newRequest(getURL(_apiHost, urlPath), method, contentType, bodyBuffer)
	if err != nil {
		return nil, err
	}
	firstResp, firstErr := client.Do(req)
	if firstErr == nil {
		// save last good IP
		a.saveLastGoodAlternateIP(nil)
		return firstResp, firstErr
	}
	log.Warning("Failed to access " + _apiHost)

	// try to access API server by alternate IP
	for i, ip := range ips {
		log.Info(fmt.Sprintf("Trying to use alternate API IP #%d...", i))

		req, err := newRequest(getURL(ip.String(), urlPath), method, contentType, bodyBuffer)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)

		if err != nil {
			fmt.Println("Failed: ", err.Error())
			continue
		}

		// save last good IP
		a.saveLastGoodAlternateIP(ip)

		log.Info(fmt.Sprintf("Success!"))
		return resp, err
	}

	return firstResp, fmt.Errorf("Unable to access IVPN API server: %w", firstErr)
}

func (a *API) requestRaw(urlPath string, method string, contentType string, requestObject interface{}, timeoutMs int) (responseData []byte, err error) {
	resp, err := a.doRequest(urlPath, method, contentType, requestObject, timeoutMs)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to get API HTTP response body: %w", err)
	}

	return body, nil
}

func (a *API) request(urlPath string, method string, contentType string, requestObject interface{}, responseObject interface{}) error {
	return a.requestEx(urlPath, method, contentType, requestObject, responseObject, 0)
}

func (a *API) requestEx(urlPath string, method string, contentType string, requestObject interface{}, responseObject interface{}, timeoutMs int) error {
	body, err := a.requestRaw(urlPath, method, contentType, requestObject, timeoutMs)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, responseObject); err != nil {
		return fmt.Errorf("failed to deserialize API response: %w", err)
	}

	return nil
}
