//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
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

package service

import (
	"net"
	"time"

	"github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/service/preferences"
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
	OnAccountStatus(sessionToken string, account preferences.AccountStatus)
	OnDNSChanged(dns net.IP)
	OnKillSwitchStateChanged()
}
