package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ivpn/desktop-app-daemon/api"
	"github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/service/platform"
)

type serversUpdater struct {
	servers           *types.ServersInfoResponse
	api               *api.API
	updatedNotifyChan chan struct{}
}

// CreateServersUpdater - constructor for serversUpdater object
func CreateServersUpdater(apiObj *api.API) (IServersUpdater, error) {
	updater := &serversUpdater{api: apiObj}

	updater.updatedNotifyChan = make(chan struct{}, 1)

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
func (s *serversUpdater) GetServers() (*types.ServersInfoResponse, error) {
	if s.servers != nil {
		return s.servers, nil
	}

	servers, err := readServersFromCache()
	if servers != nil && err == nil {
		s.servers = servers
		return servers, nil
	} else if err != nil {
		log.Warning(err)
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
func (s *serversUpdater) updateServers() (*types.ServersInfoResponse, error) {
	servers, err := s.api.DownloadServersList()
	if err != nil {
		return servers, fmt.Errorf("failed to download servers list: %w", err)
	}
	log.Info(fmt.Sprintf("Updated servers info (%d OpenVPN; %d WireGuard)\n", len(servers.OpenvpnServers), len(servers.WireguardServers)))

	s.servers = servers
	if err := writeServersToCache(servers); err != nil {
		log.Error("failed to save servers cache file: ", err)
	}

	select {
	case s.updatedNotifyChan <- struct{}{}:
		// notified
	default:
		// channel is full
	}

	return servers, nil
}

// UpdateNotifierChannel returns channel which is nitifying when servers was updated
func (s *serversUpdater) UpdateNotifierChannel() chan struct{} {
	return s.updatedNotifyChan
}

func readServersFromCache() (*types.ServersInfoResponse, error) {
	stat, err := os.Stat(platform.ServersFile())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read servers cache file: %w", err)
		}
		return nil, fmt.Errorf("failed to info about servers cache file: %w", err)
	}

	mode := stat.Mode()
	if mode != platform.DefaultFilePermissionForConfig {
		os.Remove(platform.ServersFile())
		return nil, fmt.Errorf(fmt.Sprintf("skip reading servers cache file (wrong permissions: %o but expected %o)", mode, platform.DefaultFilePermissionForConfig))
	}

	data, err := ioutil.ReadFile(platform.ServersFile())
	if err != nil {
		return nil, fmt.Errorf("failed to read servers cache file: %w", err)
	}

	servers := new(types.ServersInfoResponse)
	if err := json.Unmarshal(data, servers); err != nil {
		return nil, fmt.Errorf("failed to unmsrshal servers cache file: %w", err)
	}

	return servers, nil
}

func writeServersToCache(servers *types.ServersInfoResponse) error {
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

	return ioutil.WriteFile(platform.ServersFile(), data, platform.DefaultFilePermissionForConfig) // only owner (root) can read/write file
}
