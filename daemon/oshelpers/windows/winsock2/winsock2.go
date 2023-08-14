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

//go:build windows
// +build windows

package winsock2

import (
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	_dll             = windows.NewLazySystemDLL("ws2_32.dll")
	_fWSACreateEvent = _dll.NewProc("WSACreateEvent")
)

// WSACreateEvent - The WSACreateEvent function creates a new event object.
// https://docs.microsoft.com/en-us/windows/win32/api/winsock2/nf-winsock2-wsacreateevent
func WSACreateEvent() (syscall.Handle, error) {
	retval, _, err := _fWSACreateEvent.Call()
	if err != syscall.Errno(0) {
		return 0, err
	}
	return syscall.Handle(retval), nil
}
