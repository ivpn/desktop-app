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

	"github.com/ivpn/desktop-app/daemon/interoperability"
	"github.com/ivpn/desktop-app/daemon/protocol/ivpnclient"
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
func (p *Protocol) IsClientConnected() bool {
	p._connectionsMutex.RLock()
	defer p._connectionsMutex.RUnlock()

	for _, val := range p._connections {
		if val.IsAuthenticated {
			return true
		}
	}
	return false
}

// IsMainClientConnected checks is any authenticated connection available of main client type (UI)
// All the rest client types are considered as additional (e.g. CLI, Portmaster, etc.).
func (p *Protocol) IsMainClientConnected() bool {
	return p.isClientConnectedByType(ivpnclient.ClientUi)
}

func (p *Protocol) isClientConnectedByType(clientType ivpnclient.ClientTypeEnum) bool {
	p._connectionsMutex.RLock()
	defer p._connectionsMutex.RUnlock()

	for _, val := range p._connections {
		if val.IsAuthenticated {
			if val.Type == clientType {
				return true
			}
		}
	}
	return false
}

// IsCanDoBackgroundAction returns 'false' when no background action allowed (e.g. EAA enabled but no authenticated clients connected)
func (p *Protocol) IsCanDoBackgroundAction() bool {
	if p._eaa.IsEnabled() {
		return p.IsMainClientConnected()
	}
	return true
}

func (p *Protocol) clientConnected(c net.Conn, cType ivpnclient.ClientTypeEnum) {
	func() {
		p._connectionsMutex.Lock()
		defer p._connectionsMutex.Unlock()
		p._connections[c] = &connectionInfo{Type: cType}
	}()

	interoperability.ClientConnected(cType)

	if cType == ivpnclient.ClientPortmaster {
		p._service.SplitTunnelling_SetDisabledReason("Split Tunnel functionality is currently disabled for compatibility with Portmaster, which has been detected as running on this system.")
	}
}

func (p *Protocol) clientDisconnected(c net.Conn) *connectionInfo {
	isPortmasterConnected := false

	ret := func() *connectionInfo {
		p._connectionsMutex.Lock()
		defer p._connectionsMutex.Unlock()

		var ret *connectionInfo
		if ci, ok := p._connections[c]; ok {
			ret = ci
		}

		delete(p._connections, c)
		c.Close()

		for _, val := range p._connections {
			if val.Type == ivpnclient.ClientPortmaster {
				isPortmasterConnected = true
				break
			}
		}

		return ret
	}()

	if ret != nil {
		interoperability.ClientDisconnected(ret.Type)
	}

	if !isPortmasterConnected {
		p._service.SplitTunnelling_SetDisabledReason("")
	}

	return ret
}

func (p *Protocol) clientsConnectedCount() int {
	p._connectionsMutex.RLock()
	defer p._connectionsMutex.RUnlock()
	return len(p._connections)
}

func (p *Protocol) getConnectionInfo(c net.Conn) (cInfo *connectionInfo) {
	p._connectionsMutex.RLock()
	defer p._connectionsMutex.RUnlock()
	cInfo, _ = p._connections[c]
	return
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
	p._connections = make(map[net.Conn]*connectionInfo)
}

func (p *Protocol) clientSetAuthenticated(c net.Conn) {
	// separate anonymous function for correct mutex unlock
	func() {
		p._connectionsMutex.Lock()
		defer p._connectionsMutex.Unlock()

		if cInfo, ok := p._connections[c]; ok {
			if !cInfo.IsAuthenticated {
				// connected client (first authentication)
				cInfo.IsAuthenticated = true

				go func() {
					// notifying service about authenticated client (autoconnect if needed)
					p._service.OnAuthenticatedClient(cInfo.Type)
				}()
			}
		}
	}()

	if len(p._lastConnectionErrorToNotifyClient) > 0 {
		log.Info("Sending delayed error to client: ", p._lastConnectionErrorToNotifyClient)
		delayedErr := ivpnclient.ErrorRespDelayed{}
		delayedErr.ErrorMessage = p._lastConnectionErrorToNotifyClient
		p.sendResponse(c, &delayedErr, 0)
	}
	p._lastConnectionErrorToNotifyClient = ""
}

// ensureCanModifyDnsSettings checks if the client connection is allowed to modify DNS settings
func (p *Protocol) ensureCanModifyDnsSettings(conn net.Conn) error {
	connInfo := p.getConnectionInfo(conn)
	if connInfo == nil {
		return fmt.Errorf("internal error: connection info not found")
	}
	// If temporary prioritized DNS settings are already defined by another client - do not allow to modify DNS settings
	curVal := p._service.GetSettingsTempPrioritizedDNS()
	if !curVal.IsEmpty() && !connInfo.IsPrioritizedDnsDefined {
		return fmt.Errorf("%s", curVal.Description)
	}
	return nil
}
