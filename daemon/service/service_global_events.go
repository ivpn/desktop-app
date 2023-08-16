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

type ServiceEventType uint32

const (
	On_Power_WakeUp  ServiceEventType = 0x10
	On_Session_Logon ServiceEventType = 0x20
)

func (s *Service) startProcessingPowerEvents() bool {
	eventsChan := s._globalEvents
	if eventsChan == nil {
		return false
	}
	go func() {
		log.Info("Power events receiver started")
		defer log.Info("Power events receiver stopped")
		for {
			evt := <-eventsChan
			if evt == On_Session_Logon {
				log.Info("Event: On_Session_Logon")
				s.autoConnectIfRequired(OnSessionLogon, nil)
			}
		}
	}()
	return true
}
