package wgkeys

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ivpn/desktop-app-daemon/api"
	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/vpn/wireguard"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("wgkeys")
}

//HardExpirationIntervalDays = 40;

// IWgKeysChangeReceiver WG key update handler
type IWgKeysChangeReceiver interface {
	WireGuardSaveNewKeys(wgPublicKey string, wgPrivateKey string, wgLocalIP net.IP)
	WireGuardGetKeys() (session, wgPublicKey, wgPrivateKey, wgLocalIP string, generatedTime time.Time, updateInterval time.Duration)
	Connected() bool
}

// CreateKeysManager create WireGuard keys manager
func CreateKeysManager(apiObj *api.API, wgToolBinPath string) *KeysManager {
	return &KeysManager{
		_stopKeysRotation: make(chan struct{}),
		_wgToolBinPath:    wgToolBinPath,
		_apiObj:           apiObj}
}

// KeysManager WireGaurd keys manager
type KeysManager struct {
	_mutex            sync.Mutex
	_service          IWgKeysChangeReceiver
	_apiObj           *api.API
	_wgToolBinPath    string
	_stopKeysRotation chan struct{}
}

// Init - initialize master service
func (m *KeysManager) Init(receiver IWgKeysChangeReceiver) error {
	if receiver == nil || m._service != nil {
		return fmt.Errorf("failed to initialize WG KeysManager")
	}
	m._service = receiver
	return nil
}

// StartKeysRotation start keys rotation
func (m *KeysManager) StartKeysRotation() error {
	if m._service == nil {
		return fmt.Errorf("unable to start WG keys rotation (KeysManager not initialized)")
	}

	m.StopKeysRotation()

	_, activePublicKey, _, _, lastUpdate, interval := m._service.WireGuardGetKeys()
	if interval <= 0 {
		return fmt.Errorf("unable to start WG keys rotation (update interval not defined)")
	}

	if len(activePublicKey) == 0 {
		log.Info("Active public WG key is not defined. WG key rotation disabled.")
		return nil
	}

	go func() {
		log.Info(fmt.Sprintf("Keys rotation started (interval:%v)", interval))
		defer log.Info("Keys rotation stopped")

		needStop := false
		isLastUpdateFailed := false

		for needStop == false {
			_, _, _, _, lastUpdate, interval = m._service.WireGuardGetKeys()
			waitInterval := time.Until(lastUpdate.Add(interval))
			if isLastUpdateFailed {
				waitInterval = time.Hour
			}

			// update immediately, if it is a time
			if lastUpdate.Add(waitInterval).Before(time.Now()) {
				waitInterval = time.Second
			}

			select {
			case <-time.After(waitInterval):
				err := m.UpdateKeysIfNecessary()
				if err != nil {
					isLastUpdateFailed = true
				} else {
					isLastUpdateFailed = false
					lastUpdate = time.Now()
				}

				break

			case <-m._stopKeysRotation:
				needStop = true
				break
			}
		}
	}()

	return nil
}

// StopKeysRotation stop keys rotation
func (m *KeysManager) StopKeysRotation() {
	select {
	case m._stopKeysRotation <- struct{}{}:
	default:
	}
}

// GenerateKeys generate keys
func (m *KeysManager) GenerateKeys() error {
	return m.generateKeys(false)
}

// UpdateKeysIfNecessary generate or update keys
// 1) If no active WG keys defined - new keys will be generated + key rotation will be started
// 2) If active WG key defined - key will be updated only if it is a time to do it
func (m *KeysManager) UpdateKeysIfNecessary() error {
	return m.generateKeys(true)
}

func (m *KeysManager) generateKeys(onlyUpdateIfNecessary bool) (retErr error) {
	defer func() {
		if retErr != nil {
			log.Error("Failed to update WG keys: ", retErr)
		}
	}()

	if m._service == nil {
		return fmt.Errorf("WG KeysManager not initialized")
	}

	m._mutex.Lock()
	defer m._mutex.Unlock()

	session, activePublicKey, _, _, lastUpdate, interval := m._service.WireGuardGetKeys()
	// update interval must be defined
	if onlyUpdateIfNecessary && interval <= 0 {
		return fmt.Errorf("unable to 'GenerateOrUpdateKeys' (update interval is not defined)")
	}

	// If active WG key defined - key will be updated only if it is a time to do it
	if onlyUpdateIfNecessary && len(activePublicKey) > 0 {
		if lastUpdate.Add(interval).After(time.Now()) {
			// it is not a time to regenerate keys
			return nil
		}
	}

	log.Info("Updating WG keys...")

	pub, priv, err := wireguard.GenerateKeys(m._wgToolBinPath)
	if err != nil {
		return err
	}

	activeKeyToUpdate := activePublicKey
	// When VPN is not connected - no sense to use 'update',
	// just set new WG key for this session.
	// This can avoid any potential issues regarding 'WgPublicKeyNotFound' error.
	if m._service.Connected() == false {
		activeKeyToUpdate = ""
	}

	localIP, err := m._apiObj.WireGuardKeySet(session, pub, activeKeyToUpdate)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("WG keys updated (%s:%s) ", localIP.String(), pub))
	m._service.WireGuardSaveNewKeys(pub, priv, localIP)

	// If no active WG keys defined - new keys will be generated + key rotation will be started
	if len(activePublicKey) == 0 {
		m.StartKeysRotation()
	}
	return nil
}
