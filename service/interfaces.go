package service

import (
	"net"
	"time"

	"github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/service/wgkeys"
)

// IServersUpdater - interface for updating server info mechanism
type IServersUpdater interface {
	GetServers() (*types.ServersInfoResponse, error)
	// UpdateNotifierChannel returns channel which is nitifying when servers was updated
	UpdateNotifierChannel() chan struct{}
}

// INetChangeDetector - object is detecting routing changes on a PC
type INetChangeDetector interface {
	Start(routingChangeChan chan<- struct{}, currentDefaultInterface *net.Interface)
	Stop()
	DelayBeforeNotify() time.Duration
}

// IWgKeysManager - WireGuard keys manager
type IWgKeysManager interface {
	Init(receiver wgkeys.IWgKeysChangeReceiver) error
	StartKeysRotation() error
	StopKeysRotation()
	GenerateKeys() error
	UpdateKeysIfNecessary() error
}

// IServiceEventsReceiver is the receiver for service events (normally, it is protocol object)
type IServiceEventsReceiver interface {
	OnServiceSessionChanged()
}
