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

// Provider flags
const (
	FwpmProviderFlagPersistent uint32 = 0x00000001
)

// WfpGetProviderFlags return provider flags
func WfpGetProviderFlags(engine syscall.Handle, providerGUID syscall.GUID) (isInstalled bool, flags uint32, err error) {
	defer catchPanic(&err)

	retval, _, err := fWfpGetProviderFlags.Call(uintptr(engine), uintptr(unsafe.Pointer(&providerGUID)), uintptr(unsafe.Pointer(&flags)))
	if err != syscall.Errno(0) {
		return false, 0, err
	}
	if retval != 0 {
		if retval == FwpEProviderNotFound {
			return false, 0, nil
		}
		return false, 0, fmt.Errorf("FwpmProviderGetByKey0 returned error code: 0x%x", retval)
	}
	return true, flags, nil
}

// WfpProviderDelete removes provider
func WfpProviderDelete(engine syscall.Handle, providerGUID syscall.GUID) (err error) {
	defer catchPanic(&err)

	retval, _, err := fWfpProviderDelete.Call(uintptr(engine), uintptr(unsafe.Pointer(&providerGUID)))
	return checkDefaultAPIResp(retval, err)
}

// WfpProviderAdd adding provider
func WfpProviderAdd(engine syscall.Handle, provider syscall.Handle) (err error) {
	defer catchPanic(&err)

	retval, _, err := fWfpProviderAdd.Call(uintptr(engine), uintptr(provider))
	return checkDefaultAPIResp(retval, err)
}

// FWPMPROVIDER0Create creating provider
func FWPMPROVIDER0Create(providerGUID syscall.GUID) (povider syscall.Handle, err error) {
	defer catchPanic(&err)

	providerPtr, _, err := fFWPMPROVIDER0Create.Call(uintptr(unsafe.Pointer(&providerGUID)))
	if providerPtr == 0 {
		if err != syscall.Errno(0) {
			return 0, err
		}

		return 0, errors.New("failed to create WFP provider object")
	}

	return syscall.Handle(providerPtr), nil
}

// FWPMPROVIDER0SetFlags sets provider flags
func FWPMPROVIDER0SetFlags(provider syscall.Handle, flags uint32) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMPROVIDER0SetFlags.Call(uintptr(provider), uintptr(flags))
	return checkDefaultAPIResp(retval, err)
}

// FWPMPROVIDER0SetDisplayData sets provider display data
func FWPMPROVIDER0SetDisplayData(provider syscall.Handle, name string, description string) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMPROVIDER0SetDisplayData.Call(uintptr(provider),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(description))))

	return checkDefaultAPIResp(retval, err)
}

// FWPMPROVIDER0Delete removes provider
func FWPMPROVIDER0Delete(provider syscall.Handle) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMPROVIDER0Delete.Call(uintptr(provider))
	return checkDefaultAPIResp(retval, err)
}
