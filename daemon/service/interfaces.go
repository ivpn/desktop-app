//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
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

package service

import (
	"net"

	api_types "github.com/ivpn/desktop-app/daemon/api/types"
	"github.com/ivpn/desktop-app/daemon/service/preferences"
	service_types "github.com/ivpn/desktop-app/daemon/service/types"
	"github.com/ivpn/desktop-app/daemon/service/wgkeys"
	"github.com/ivpn/desktop-app/daemon/vpn"
	"github.com/ivpn/desktop-app/daemon/wifiNotifier"
)

// IServersUpdater - interface for updating server info mechanism
type IServersUpdater interface {
	// Start periodically updating (downloading) servers in background
	StartUpdater() error
	// GetServers - get servers list.
	// Use cached data (if exists), otherwise - download servers list.
	GetServers() (*api_types.ServersInfoResponse, error)
	// GetServersForceUpdate returns servers list info (locations, hosts and host load).
	// The daemon will make request to update servers from the backend.
	// The cached data will be ignored in this case.
	GetServersForceUpdate() (*api_types.ServersInfoResponse, error)
	// UpdateNotifierChannel returns channel which is notifying when servers was updated
	UpdateNotifierChannel() chan struct{}
}

// INetChangeDetector - object is detecting routing changes on a PC
type INetChangeDetector interface {
	// Init - Initialise route change detector
	//    'routingChangeChan' is the channel for notifying when the default routing is NOT over the 'interfaceToProtect' anymore
	//    'routingUpdateChan' is the channel for notifying when there were some routing changes but 'interfaceToProtect' is still is the default route
	Init(routingChangeChan chan<- struct{}, routingUpdateChan chan<- struct{}, currentDefaultInterface *net.Interface) error
	UnInit() error
	Start() error // Start - Starts route change detector (asynchronous)
	Stop() error
}

// IWgKeysManager - WireGuard keys manager
type IWgKeysManager interface {
	Init(receiver wgkeys.IWgKeysChangeReceiver) error
	StartKeysRotation() error
	StopKeysRotation()
	GenerateKeys() error
	UpdateKeysIfNecessary() (retErr error)
}

// IServiceEventsReceiver is the receiver for service events (normally, it is protocol object)
type IServiceEventsReceiver interface {
	OnServiceSessionChanged()
	OnAccountStatus(sessionToken string, account preferences.AccountStatus)
	OnKillSwitchStateChanged()
	OnWiFiChanged(wifiNotifier.WifiInfo)
	OnPingStatus(retMap map[string]int)
	OnServersUpdated(*api_types.ServersInfoResponse)
	OnSplitTunnelStatusChanged()
	OnVpnStateChanged(state vpn.StateInfo)
	OnVpnPauseChanged()

	// called by a service when new connection is required (e.g. requested by 'trusted-wifi' functionality or 'auto-connect' on launch)
	RegisterConnectionRequest(params service_types.ConnectionParams) error
	// IsClientConnected checks is any authenticated connection available of specific client type
	IsClientConnected(checkOnlyUiClients bool) bool
	// IsCanDoBackgroundAction returns 'false' when no background action allowed (e.g. EAA enabled but no authenticated clients connected)
	IsCanDoBackgroundAction() bool
}
