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

package winlib

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

// WfpEngineOpen opening WFP engine
func WfpEngineOpen(session syscall.Handle) (engine syscall.Handle, err error) {
	defer catchPanic(&err)

	var enginePtr uintptr = 0

	retval, _, err := fWfpEngineOpen.Call(uintptr(session), uintptr(unsafe.Pointer(&enginePtr)))
	if err != syscall.Errno(0) {
		return syscall.Handle(0), err
	}
	if retval != 0 {
		return syscall.Handle(0), fmt.Errorf("WFP error: 0x%X", retval)
	}
	return syscall.Handle(enginePtr), nil
}

// WfpEngineClose closing WFP engine
func WfpEngineClose(engine syscall.Handle) (err error) {
	defer catchPanic(&err)

	_, _, err = fWfpEngineClose.Call(uintptr(engine))
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

// CreateWfpSessionObject creates WFP session
func CreateWfpSessionObject(isDynamic bool) (session syscall.Handle, err error) {
	defer catchPanic(&err)

	var isDynArg byte = 0
	if isDynamic {
		isDynArg = 1
	}

	sessionPtr, _, err := fCreateWfpSessionObject.Call(uintptr(isDynArg))
	if sessionPtr == 0 {
		if err != syscall.Errno(0) {
			return 0, err
		}

		return 0, errors.New("failed to create WFP session object")
	}

	return syscall.Handle(sessionPtr), nil
}

// DeleteWfpSessionObject deletes WFP session
func DeleteWfpSessionObject(session syscall.Handle) (err error) {
	defer catchPanic(&err)
	_, _, err = fDeleteWfpSessionObject.Call(uintptr(session))
	return err
}

// WfpTransactionBegin starts transaction
func WfpTransactionBegin(engine syscall.Handle) (err error) {
	defer catchPanic(&err)
	_, _, err = fWfpTransactionBegin.Call(uintptr(engine))
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

// WfpTransactionCommit commits transaction
func WfpTransactionCommit(engine syscall.Handle) (err error) {
	defer catchPanic(&err)
	_, _, err = fWfpTransactionCommit.Call(uintptr(engine))
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}

// WfpTransactionAbort aborting transaction
func WfpTransactionAbort(engine syscall.Handle) (err error) {
	defer catchPanic(&err)
	_, _, err = fWfpTransactionAbort.Call(uintptr(engine))
	if err != syscall.Errno(0) {
		return err
	}
	return nil
}
