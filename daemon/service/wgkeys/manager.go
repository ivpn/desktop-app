//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
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

package wgkeys

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/api"
	"github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/vpn"
	"github.com/ivpn/desktop-app/daemon/vpn/wireguard"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("wgkeys")
}

//HardExpirationIntervalDays = 40;

// IWgKeysChangeReceiver WG key update handler
type IWgKeysChangeReceiver interface {
	WireGuardSaveNewKeys(wgPublicKey string, wgPrivateKey string, wgLocalIP string)
	WireGuardGetKeys() (session, wgPublicKey, wgPrivateKey, wgLocalIP string, generatedTime time.Time, updateInterval time.Duration)
	FirewallEnabled() (bool, error)
	Connected() bool
	ConnectedType() (isConnected bool, connectedVpnType vpn.Type)
	IsConnectivityBlocked() (err error) // IsConnectivityBlocked - returns nil if connectivity NOT blocked
	OnSessionNotFound()
}

// CreateKeysManager create WireGuard keys manager
func CreateKeysManager(apiObj *api.API, wgToolBinPath string) *KeysManager {
	return &KeysManager{
		wgToolBinPath: wgToolBinPath,
		api:           apiObj}
}

// KeysManager WireGuard keys manager
type KeysManager struct {
	mutex         sync.Mutex
	service       IWgKeysChangeReceiver
	api           *api.API
	wgToolBinPath string

	activeRotationInterval time.Duration
	stop                   context.CancelFunc
	activeRotationWg       sync.WaitGroup
}

// Init - initialize master service
func (m *KeysManager) Init(receiver IWgKeysChangeReceiver) error {
	if receiver == nil || m.service != nil {
		return fmt.Errorf("failed to initialize WG KeysManager")
	}
	m.service = receiver
	return nil
}

// StartKeysRotation start keys rotation
func (m *KeysManager) StartKeysRotation() error {
	if m.service == nil {
		return fmt.Errorf("unable to start WG keys rotation (KeysManager not initialized)")
	}

	m.StopKeysRotation()

	_, activePublicKey, _, _, lastUpdate, interval := m.service.WireGuardGetKeys()

	if interval <= 0 {
		return fmt.Errorf("unable to start WG keys rotation (update interval not defined)")
	}

	if len(activePublicKey) == 0 {
		log.Info("Active public WG key is not defined. WG key rotation disabled.")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.mutex.Lock()
	m.stop = cancel
	m.activeRotationInterval = interval
	m.mutex.Unlock()

	m.activeRotationWg.Add(1)
	go func(ctx context.Context) {
		log.Info(fmt.Sprintf("Keys rotation started (interval:%v)", interval))
		defer func() {
			log.Info("Keys rotation stopped")
			m.activeRotationWg.Done() // notify routine stopped
		}()

		const maxCheckInterval = time.Minute * 5

		isLastUpdateFailed := false
		isLastUpdateFailedCnt := 0

		for {
			waitInterval := maxCheckInterval

			if isLastUpdateFailed {
				// If the last update failed - do next try after some delay
				// (delay is incrteasing every retry: 5, 10, 15 ... 60 min)
				waitInterval = maxCheckInterval * time.Duration(isLastUpdateFailedCnt)
				if waitInterval > time.Hour {
					waitInterval = time.Hour
				}
				lastUpdate = time.Now()
			} else {
				_, _, _, _, lastUpdate, interval = m.service.WireGuardGetKeys()
				waitInterval = time.Until(lastUpdate.Add(interval))
			}

			// update immediately, if it is a time
			if lastUpdate.Add(interval).Before(time.Now()) {
				waitInterval = time.Second
			}

			if waitInterval > maxCheckInterval && !isLastUpdateFailed {
				// We can not trust "time.After()" that it will be triggered in exact time.
				// If the computer fall to sleep on a long time, after wake up the "time.After()"
				// will trigger after [sleep time]+[time].
				// Therefore we defining maximum allowed interval to check necessity on keys generation
				waitInterval = maxCheckInterval
			}

			select {
			case <-ctx.Done(): // stop signal sent
				return
			case <-time.After(waitInterval):
				if m.UpdateKeysIfNecessary() != nil {
					isLastUpdateFailed = true
					isLastUpdateFailedCnt += 1
				} else {
					isLastUpdateFailed = false
					isLastUpdateFailedCnt = 0
				}
			}
		}
	}(ctx)

	return nil
}

// StopKeysRotation stop keys rotation
func (m *KeysManager) StopKeysRotation() {
	// send stop signal (if already running)
	m.mutex.Lock()
	if m.stop != nil {
		m.stop()
		m.stop = nil
		m.activeRotationInterval = 0
	}
	m.mutex.Unlock()
	// wait untill keys rotution goroutine stops (if running)
	m.activeRotationWg.Wait()
}

// GenerateKeys generate keys
func (m *KeysManager) GenerateKeys() error {
	return m.generateKeys(false)
}

// UpdateKeysIfNecessary generate or update keys
// 1) If no active WG keys defined - new keys will be generated + key rotation will be started
// 2) If active WG key defined - key will be updated only if it is a time to do it
func (m *KeysManager) UpdateKeysIfNecessary() (retErr error) {
	return m.generateKeys(true)
}

func (m *KeysManager) generateKeys(onlyUpdateIfNecessary bool) (retErr error) {
	defer func() {
		if retErr != nil {
			log.Error("Failed to update WG keys: ", retErr)
		}
	}()

	if m.service == nil {
		return fmt.Errorf("WG KeysManager not initialized")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check update configuration second time (locked by mutex)
	session, activePublicKey, _, _, lastUpdate, interval := m.service.WireGuardGetKeys()

	// check if update required
	if onlyUpdateIfNecessary && len(activePublicKey) > 0 {
		if interval <= 0 { // update interval must be defined
			return fmt.Errorf("unable to 'GenerateOrUpdateKeys' (update interval is not defined)")
		}
		// If active WG key defined - key will be updated only if it is a time to do it
		if lastUpdate.Add(interval).Unix() >= time.Now().Unix() {
			return nil // it is not a time to regenerate keys: do nothing and return NO error
		}
	}

	log.Info("Updating WG keys...")

	if err := m.service.IsConnectivityBlocked(); err != nil {
		// Connectivity with API servers is blocked. No sense to make API requests
		return err
	}

	pub, priv, err := wireguard.GenerateKeys(m.wgToolBinPath)
	if err != nil {
		return err
	}

	isVPNConnected, connectedVpnType := m.service.ConnectedType()

	if !isVPNConnected || connectedVpnType != vpn.WireGuard {
		// use 'activePublicKey' ONLY if WireGuard is connected
		activePublicKey = ""
	}

	// trying to update WG keys with notifying API about current active public key (if it exists)
	localIP, err := m.api.WireGuardKeySet(session, pub, activePublicKey)
	if err != nil {
		if len(activePublicKey) == 0 {
			// IMPORTANT! As soon as server receive request with empty 'activePublicKey' - it clears all keys
			// Therefore, we have to ensure that local keys are not using anymore (we have to clear them independently from we received response or not)
			m.service.WireGuardSaveNewKeys("", "", "")
		}
		log.Info("WG keys not updated: ", err)

		var e types.APIError
		if errors.As(err, &e) {
			if e.ErrorCode == types.SessionNotFound {
				m.service.OnSessionNotFound()
				return fmt.Errorf("WG keys not updated (session not found)")
			}
		}
		return fmt.Errorf("WG keys not updated. Please check your internet connection")
	}

	log.Info(fmt.Sprintf("WG keys updated (%s:%s) ", localIP.String(), pub))

	// notify service about new keys
	m.service.WireGuardSaveNewKeys(pub, priv, localIP.String())

	// Keys updated. Start keys rotation only if it not started yet or keys rotation interval changed
	if m.activeRotationInterval != interval || m.stop == nil {
		go m.StartKeysRotation() // run in routine to avoid deadlock
	}

	return nil
}
