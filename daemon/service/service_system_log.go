//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2022 Privatus Limited.
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

import "fmt"

type SystemLogMessageType int

const (
	Info    SystemLogMessageType = iota
	Warning SystemLogMessageType = iota
	Error   SystemLogMessageType = iota
)

type SystemLogMessage struct {
	Type    SystemLogMessageType
	Message string
	EventId uint32
}

func (s *Service) systemLog(mes SystemLogMessage) bool {
	switch mes.Type {
	case Info:
		log.Info(fmt.Sprintf("<SYS_LOG> INFO: '%s' (%d)", mes.Message, mes.EventId))
	case Warning:
		log.Info(fmt.Sprintf("<SYS_LOG> WARNING: '%s' (%d)", mes.Message, mes.EventId))
	case Error:
		log.Info(fmt.Sprintf("<SYS_LOG> ERROR: '%s' (%d)", mes.Message, mes.EventId))
	default:
	}

	ch := s._systemLog
	if ch == nil {
		return false
	}
	select {
	case ch <- mes:
		return true
	default:
		return false
	}
}
