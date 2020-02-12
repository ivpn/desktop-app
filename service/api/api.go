package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path"
	"sync"

	"github.com/ivpn/desktop-app-daemon/logger"
)

// API URLs
const (
	apiHost     = "api.ivpn.net"
	serversPath = "v4/servers.json"
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

func (a *API) get(urlPath string) (resp *http.Response, err error) {
	lastIP, ips := a.getAlternateIPs()

	// When trying to access API server by alternate IPs (not by DNS name)
	// we need to configure TLS to use api.ivpn.net hostname
	// (to avoid certificate errors)
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{
			ServerName: apiHost,
		},
	}
	// configure http-client with preconfigured TLS transport
	client := &http.Client{Transport: transCfg}

	if lastIP != nil {
		resp, err = client.Get(getURL(lastIP.String(), urlPath))
		if err == nil {
			return resp, nil
		}
	}

	// try to access API server by host DNS
	firstResp, firstErr := http.Get(getURL(apiHost, urlPath))
	if firstErr == nil {
		// save last good IP
		a.saveLastGoodAlternateIP(nil)
		return firstResp, firstErr
	}
	log.Warning("Failed to access " + apiHost)

	// try to access API server by alternate IP
	for i, ip := range ips {
		log.Info(fmt.Sprintf("Trying to use alternate API IP #%d...", i))
		resp, err = client.Get(getURL(ip.String(), urlPath))
		if err != nil {
			fmt.Println("Failed: ", err.Error())
			continue
		}

		// save last good IP
		a.saveLastGoodAlternateIP(ip)

		log.Info(fmt.Sprintf("Success!"))
		return resp, err

	}

	log.Error("Unable to access IVPN API server")

	return firstResp, firstErr
}

// DownloadServersList - download servers list form API IVPN server
func (a *API) DownloadServersList() (*ServersInfoResponse, error) {
	resp, err := a.get(serversPath)
	if err != nil {
		log.Error("Unable to download servers list:", err)
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("ServersList: Failed to get HTTP response body:", err)
		return nil, err
	}

	servers := new(ServersInfoResponse)
	if err := json.Unmarshal(body, servers); err != nil {
		log.Error("Failed to deserialize servers list:", err)
		return nil, err
	}

	log.Info(fmt.Sprintf("Updated servers info (%d OpenVPN; %d WireGuard)\n", len(servers.OpenvpnServers), len(servers.WireguardServers)))

	a.SetAlternateIPs(servers.Config.API.IPAddresses)

	return servers, nil
}
