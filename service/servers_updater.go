package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/ivpn/desktop-app-daemon/service/api"
	"github.com/ivpn/desktop-app-daemon/service/platform"
)

type serversUpdater struct {
	servers *api.ServersInfoResponse
	api     *api.API
}

// CreateServersUpdater - constructor for serversUpdater object
func CreateServersUpdater(apiObj *api.API) (ServersUpdater, error) {
	updater := &serversUpdater{api: apiObj}

	servers, err := updater.GetServers()
	if err == nil && servers != nil {
		apiObj.SetAlternateIPs(servers.Config.API.IPAddresses)
	}

	// update servers list in background
	if err := updater.startUpdater(); err != nil {
		log.Error("Failed to start servers-list updater: ", err)
		return nil, err
	}
	return updater, nil
}

// GetServers - get servers list.
// Use cached data (if exists), otherwise - download servers list.
func (s *serversUpdater) GetServers() (*api.ServersInfoResponse, error) {
	if s.servers != nil {
		return s.servers, nil
	}

	servers, err := readServersFromCache()
	if servers != nil && err == nil {
		s.servers = servers
		return servers, nil
	}

	return s.updateServers()
}

func (s *serversUpdater) startUpdater() error {
	go func(s *serversUpdater) {
		for {
			s.updateServers()
			time.Sleep(time.Hour)
		}
	}(s)

	return nil
}

// UpdateServers - download servers list
func (s *serversUpdater) updateServers() (*api.ServersInfoResponse, error) {
	servers, err := s.api.DownloadServersList()
	if err != nil {
		return servers, fmt.Errorf("failed to download servers list: %w", err)
	}

	s.servers = servers
	writeServersToCache(servers)

	return servers, nil
}

func readServersFromCache() (*api.ServersInfoResponse, error) {
	data, err := ioutil.ReadFile(platform.ServersFile())
	if err != nil {
		return nil, fmt.Errorf("failed to read servers cache file: %w", err)
	}

	servers := new(api.ServersInfoResponse)
	if err := json.Unmarshal(data, servers); err != nil {
		return nil, fmt.Errorf("failed to unmsrshal servers cache file: %w", err)
	}

	return servers, nil
}

func writeServersToCache(servers *api.ServersInfoResponse) error {
	if servers == nil {
		return errors.New("nothing to save. Servers is null")
	}

	data, err := json.Marshal(servers)
	if err != nil {
		return fmt.Errorf("failed to marshal servers into a cache: %w", err)
	}

	if data == nil {
		return errors.New("failed to serialize servers")
	}

	return ioutil.WriteFile(platform.ServersFile(), data, 0644)
}
