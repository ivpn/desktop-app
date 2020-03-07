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

func (a *API) doRequest(urlPath string, method string, contentType string, request interface{}) (resp *http.Response, err error) {
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
	client := &http.Client{Transport: transCfg, Timeout: _defaultRequestTimeout}

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

func (a *API) requestRaw(urlPath string, method string, contentType string, requestObject interface{}) (responseData []byte, err error) {
	resp, err := a.doRequest(urlPath, method, contentType, requestObject)
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
	body, err := a.requestRaw(urlPath, method, contentType, requestObject)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, responseObject); err != nil {
		return fmt.Errorf("failed to deserialize API response: %w", err)
	}

	return nil
}
