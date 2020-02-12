package service

// TODO: move interfaces definitions to a files which are using this interfaces

import (
	"net"
	"time"

	"github.com/ivpn/desktop-app-daemon/service/api"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

// Protocol - interface of communication protocol with IVPN application
type Protocol interface {
	Start(startedOnPort chan<- int, service Service) error
	Stop()
}

// ServersUpdater - interface for updating server info mechanism
type ServersUpdater interface {
	GetServers() (*api.ServersInfoResponse, error)
}

// NetChangeDetector - object is detecting routing changes on a PC
type NetChangeDetector interface {
	Start(routingChangeChan chan<- struct{}, currentDefaultInterface *net.Interface)
	Stop()
	DelayBeforeNotify() time.Duration
}

// Service - service interface
type Service interface {
	// OnControlConnectionClosed - Perform reqired operations when protocol (controll channel with UI application) was closed
	// (for example, we must disable firewall (if it not persistant))
	// Must be called by protocol object
	// Return parameters:
	// - isServiceMustBeClosed: true informing that service have to be closed ("Stop IVPN Agent when application is not running" feature)
	// - err: error
	OnControlConnectionClosed() (isServiceMustBeClosed bool, err error)

	ServersList() (*api.ServersInfoResponse, error)
	PingServers(retryCount int, timeoutMs int) (map[string]int, error)

	KillSwitchState() (bool, error)
	SetKillSwitchState(bool) error

	SetKillSwitchIsPersistent(isPersistant bool) error
	SetKillSwitchAllowLANMulticast(isAllowLanMulticast bool) error
	SetKillSwitchAllowLAN(isAllowLan bool) error

	Preferences() Preferences
	SetPreference(key string, val string) error

	SetManualDNS(dns net.IP) error
	ResetManualDNS() error

	Connect(vpn vpn.Process, manualDNS net.IP, stateChan chan<- vpn.StateInfo) error
	Disconnect() error

	Pause() error
	Resume() error
}
