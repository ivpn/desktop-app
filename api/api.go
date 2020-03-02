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
	"sync"

	commontypes "github.com/ivpn/desktop-app-daemon/api/common/types"
	"github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/logger"
)

// API URLs
const (
	_apiHost           = "api.ivpn.net"
	_serversPath       = "v4/servers.json"
	_sessionNewPath    = "v4/session/new"
	_sessionDeletePath = "v4/session/delete"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("api")
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
func (a *API) SessionNew(accountID string, wgPublicKey string, forceLogin bool) (*types.SessionsAuthenticateFullResponse, error) {
	request := &commontypes.SessionAuthenticateRequest{
		Username:   accountID,
		PublicKey:  wgPublicKey,
		ForceLogin: forceLogin}
	resp := new(types.SessionsAuthenticateFullResponse)
	if err := a.request(_sessionNewPath, "POST", "application/json", request, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// SessionDelete - remove session
func (a *API) SessionDelete(session string) error {
	request := &commontypes.SessionTokenRequest{Token: session}
	resp := new(types.APIError)
	if err := a.request(_sessionDeletePath, "POST", "application/json", request, resp); err != nil {
		return err
	}
	if resp.Status != commontypes.CodeSuccess {
		return fmt.Errorf("[%d] %s", resp.Status, resp.Message)
	}
	return nil
}

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
	client := &http.Client{Transport: transCfg}

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

func (a *API) request(urlPath string, method string, contentType string, requestObject interface{}, responseObject interface{}) error {
	resp, err := a.doRequest(urlPath, method, contentType, requestObject)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to get API HTTP response body: %w", err)
	}

	if err := json.Unmarshal(body, responseObject); err != nil {
		return fmt.Errorf("failed to deserialize API response: %w", err)
	}

	return nil
}
