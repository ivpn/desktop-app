//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
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

// +build windows

package winlib

import (
	"fmt"
	"syscall"
	"unsafe"
)

func WfpRegisterSplitTunFilters(
	engine syscall.Handle,
	providerGUID syscall.GUID,
	subLayerGUID syscall.GUID,
	isPersistant bool) (err error) {

	var isPersistantDW uint32
	if isPersistant {
		isPersistantDW = 1
	}

	defer catchPanic(&err)

	retval, _, err := fWfpRegisterSplitTunFilters.Call(
		uintptr(engine),
		uintptr(unsafe.Pointer(&providerGUID)),
		uintptr(unsafe.Pointer(&subLayerGUID)),
		uintptr(isPersistantDW))

	if err != syscall.Errno(0) {
		return err
	}
	if retval != 0 {
		return fmt.Errorf("error: 0x%X", retval)
	}
	return nil
}
