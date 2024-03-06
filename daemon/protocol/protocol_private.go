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

package protocol

import (
	"fmt"
	"net"
	"strings"

	"github.com/ivpn/desktop-app/daemon/protocol/types"
)

func getConnectionName(c net.Conn) string {
	return strings.TrimSpace(strings.Replace(c.RemoteAddr().String(), "127.0.0.1:", "", 1))
}

func (p *Protocol) connLogID(c net.Conn) string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s ", getConnectionName(c))
}

// -------------- clients connections ---------------
// IsClientConnected checks is any authenticated connection available of specific client type
func (p *Protocol) IsClientConnected(checkOnlyUiClients bool) bool {
	p._connectionsMutex.RLock()
	defer p._connectionsMutex.RUnlock()

	for _, val := range p._connections {
		if val.IsAuthenticated {
			if checkOnlyUiClients {
				if val.Type == types.ClientUi {
					return true
				}
			} else {
				return true
			}
		}
	}
	return false
}

// IsCanDoBackgroundAction returns 'false' when no background action allowed (e.g. EAA enabled but no authenticated clients connected)
func (p *Protocol) IsCanDoBackgroundAction() bool {
	if p._eaa.IsEnabled() {
		const checkOnlyUiClients = true
		return p.IsClientConnected(checkOnlyUiClients)
	}
	return true
}

func (p *Protocol) clientConnected(c net.Conn, cType types.ClientTypeEnum) {
	p._connectionsMutex.Lock()
	defer p._connectionsMutex.Unlock()
	p._connections[c] = connectionInfo{Type: cType}
}

func (p *Protocol) clientDisconnected(c net.Conn) (disconnectedClientInfo *connectionInfo) {
	p._connectionsMutex.Lock()
	defer p._connectionsMutex.Unlock()

	if ci, ok := p._connections[c]; ok {
		disconnectedClientInfo = &ci
	}

	delete(p._connections, c)
	c.Close()

	return disconnectedClientInfo
}

func (p *Protocol) clientsConnectedCount() int {
	p._connectionsMutex.Lock()
	defer p._connectionsMutex.Unlock()
	return len(p._connections)
}

// Notifying clients "service is going to stop" (client application (UI) will close)
// Closing and erasing all clients connections
func (p *Protocol) notifyClientsDaemonExiting() {
	func() {
		p._connectionsMutex.RLock()
		defer p._connectionsMutex.RUnlock()

		if len(p._connections) > 0 {
			log.Info("Notifying clients: 'daemon is stopping'...")
		}

		for conn := range p._connections {
			// notifying client "service is going to stop" (client application (UI) will close)
			p.sendResponse(conn, &types.ServiceExitingResp{}, 0)
			// closing current connection with a client
			conn.Close()
		}
	}()

	// erasing clients connections
	p._connectionsMutex.Lock()
	defer p._connectionsMutex.Unlock()
	p._connections = make(map[net.Conn]connectionInfo)
}

func (p *Protocol) clientSetAuthenticated(c net.Conn) {
	// contains information about just connected client (first authentication) or nil
	var justConnectedClientInfo *connectionInfo

	// separate anonymous function for correct mutex unlock
	func() {
		p._connectionsMutex.Lock()
		defer p._connectionsMutex.Unlock()

		if cInfo, ok := p._connections[c]; ok {
			if !cInfo.IsAuthenticated {
				cInfo.IsAuthenticated = true
				p._connections[c] = cInfo

				justConnectedClientInfo = &cInfo
			}
		}
	}()

	if justConnectedClientInfo != nil {
		go func() {
			p._service.OnAuthenticatedClient(justConnectedClientInfo.Type)
		}()
	}

	if len(p._lastConnectionErrorToNotifyClient) > 0 {
		log.Info("Sending delayed error to client: ", p._lastConnectionErrorToNotifyClient)
		delayedErr := types.ErrorRespDelayed{}
		delayedErr.ErrorMessage = p._lastConnectionErrorToNotifyClient
		p.sendResponse(c, &delayedErr, 0)
	}
	p._lastConnectionErrorToNotifyClient = ""
}
