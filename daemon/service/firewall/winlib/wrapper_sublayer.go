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
	"syscall"
	"unsafe"
)

// Sublayer flags
const (
	FwpmSublayerFlagPersistent uint32 = 0x00000001
)

// WfpSubLayerIsInstalled returns true if sublayer is installed
func WfpSubLayerIsInstalled(engine syscall.Handle, sublayerGUID syscall.GUID) (isInstalled bool, err error) {
	defer catchPanic(&err)

	retval, _, err := fWfpSubLayerIsInstalled.Call(uintptr(engine), uintptr(unsafe.Pointer(&sublayerGUID)))
	if err != syscall.Errno(0) {
		return false, err
	}

	return byte(retval) != 0, nil
}

// WfpSubLayerDelete removes sublayer
func WfpSubLayerDelete(engine syscall.Handle, sublayerGUID syscall.GUID) (err error) {
	defer catchPanic(&err)

	retval, _, err := fWfpSubLayerDelete.Call(uintptr(engine), uintptr(unsafe.Pointer(&sublayerGUID)))
	return checkDefaultAPIResp(retval, err)
}

// WfpSubLayerAdd adds sublayer
func WfpSubLayerAdd(engine syscall.Handle, sublayer syscall.Handle) (err error) {
	defer catchPanic(&err)

	retval, _, err := fWfpSubLayerAdd.Call(uintptr(engine), uintptr(sublayer))
	return checkDefaultAPIResp(retval, err)
}

// FWPMSUBLAYER0Create creates syblayer
func FWPMSUBLAYER0Create(sublayerGUID syscall.GUID, weight uint16) (sublayer syscall.Handle, err error) {
	defer catchPanic(&err)

	sublayerPtr, _, err := fFWPMSUBLAYER0Create.Call(uintptr(unsafe.Pointer(&sublayerGUID)), uintptr(weight))
	if sublayerPtr == 0 {
		if err != syscall.Errno(0) {
			return 0, err
		}

		return 0, errors.New("failed to create WFP sublayer object")
	}

	return syscall.Handle(sublayerPtr), nil
}

// FWPMSUBLAYER0SetProviderKey sets provider to a layer
func FWPMSUBLAYER0SetProviderKey(sublayer syscall.Handle, providerGUID syscall.GUID) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMSUBLAYER0SetProviderKey.Call(uintptr(sublayer), uintptr(unsafe.Pointer(&providerGUID)))
	return checkDefaultAPIResp(retval, err)
}

// FWPMSUBLAYER0SetDisplayData sets display data to a sublayer
func FWPMSUBLAYER0SetDisplayData(sublayer syscall.Handle, name string, description string) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMSUBLAYER0SetDisplayData.Call(uintptr(sublayer),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(description))))

	return checkDefaultAPIResp(retval, err)
}

// FWPMSUBLAYER0SetWeight sets sublayer weight
func FWPMSUBLAYER0SetWeight(sublayer syscall.Handle, weight int16) (err error) {
	defer catchPanic(&err)

	_, _, err = fFWPMSUBLAYER0SetWeight.Call(uintptr(sublayer), uintptr(weight))
	if err != syscall.Errno(0) {
		return err
	}

	return nil
}

// FWPMSUBLAYER0SetFlags sets sublayer flags
func FWPMSUBLAYER0SetFlags(sublayer syscall.Handle, flags uint32) (err error) {
	defer catchPanic(&err)

	_, _, err = fFWPMSUBLAYER0SetFlags.Call(uintptr(sublayer), uintptr(flags))
	if err != syscall.Errno(0) {
		return err
	}

	return nil
}

// FWPMSUBLAYER0Delete removas sublayer
func FWPMSUBLAYER0Delete(sublayer syscall.Handle) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMSUBLAYER0Delete.Call(uintptr(sublayer))
	return checkDefaultAPIResp(retval, err)
}
