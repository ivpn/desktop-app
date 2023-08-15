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
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/ivpn/desktop-app/daemon/logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("frwl_w")
}

const (
	// FwpEProviderNotFound - The provider does not exist.
	FwpEProviderNotFound = 0x80320005
)

var (
	fWfpEngineOpen  *syscall.LazyProc
	fWfpEngineClose *syscall.LazyProc

	fCreateWfpSessionObject *syscall.LazyProc
	fDeleteWfpSessionObject *syscall.LazyProc

	fWfpTransactionBegin  *syscall.LazyProc
	fWfpTransactionCommit *syscall.LazyProc
	fWfpTransactionAbort  *syscall.LazyProc

	fWfpGetProviderFlags         *syscall.LazyProc
	fWfpProviderDelete           *syscall.LazyProc
	fWfpProviderAdd              *syscall.LazyProc
	fFWPMPROVIDER0Create         *syscall.LazyProc
	fFWPMPROVIDER0SetFlags       *syscall.LazyProc
	fFWPMPROVIDER0SetDisplayData *syscall.LazyProc
	fFWPMPROVIDER0Delete         *syscall.LazyProc

	fWfpSubLayerIsInstalled      *syscall.LazyProc
	fWfpSubLayerDelete           *syscall.LazyProc
	fWfpSubLayerAdd              *syscall.LazyProc
	fFWPMSUBLAYER0Create         *syscall.LazyProc
	fFWPMSUBLAYER0SetProviderKey *syscall.LazyProc
	fFWPMSUBLAYER0SetDisplayData *syscall.LazyProc
	fFWPMSUBLAYER0SetWeight      *syscall.LazyProc
	fFWPMSUBLAYER0SetFlags       *syscall.LazyProc
	fFWPMSUBLAYER0Delete         *syscall.LazyProc

	fFWPMFILTERCreate                 *syscall.LazyProc
	fFWPMFILTERDelete                 *syscall.LazyProc
	fFWPMFILTERSetProviderKey         *syscall.LazyProc
	fFWPMFILTERSetDisplayData         *syscall.LazyProc
	fFWPMFILTERAllocateConditions     *syscall.LazyProc
	fFWPMFILTERSetConditionFieldKey   *syscall.LazyProc
	fFWPMFILTERSetConditionMatchType  *syscall.LazyProc
	fFWPMFILTERSetConditionV4AddrMask *syscall.LazyProc
	fFWPMFILTERSetConditionV6AddrMask *syscall.LazyProc
	fFWPMFILTERSetConditionUINT16     *syscall.LazyProc
	fFWPMFILTERSetConditionBlobString *syscall.LazyProc
	fFWPMFILTERSetAction              *syscall.LazyProc
	fFWPMFILTERSetFlags               *syscall.LazyProc
	fWfpFilterAdd                     *syscall.LazyProc
	fWfpFilterDeleteByID              *syscall.LazyProc
	fWfpFiltersDeleteByProviderKey    *syscall.LazyProc
)

// Initialize doing initialization stuff (called on application start)
func Initialize(wfpDllPath string) error {
	if len(wfpDllPath) == 0 {
		return fmt.Errorf("unable to initialize firewall wrapper: firewall dll path not initialized")
	}
	if _, err := os.Stat(wfpDllPath); err != nil {
		return fmt.Errorf("unable to initialize firewall wrapper (firewall dll not found) : '%s'", wfpDllPath)
	}

	dll := syscall.NewLazyDLL(wfpDllPath)

	fWfpEngineOpen = dll.NewProc("WfpEngineOpen")
	fWfpEngineClose = dll.NewProc("WfpEngineClose")

	fCreateWfpSessionObject = dll.NewProc("CreateWfpSessionObject")
	fDeleteWfpSessionObject = dll.NewProc("DeleteWfpSessionObject")

	fWfpTransactionBegin = dll.NewProc("WfpTransactionBegin")
	fWfpTransactionCommit = dll.NewProc("WfpTransactionCommit")
	fWfpTransactionAbort = dll.NewProc("WfpTransactionAbort")

	fWfpGetProviderFlags = dll.NewProc("WfpGetProviderFlagsPtr")
	fWfpProviderDelete = dll.NewProc("WfpProviderDeletePtr")
	fWfpProviderAdd = dll.NewProc("WfpProviderAdd")
	fFWPMPROVIDER0Create = dll.NewProc("FWPM_PROVIDER0_CreatePtr")
	fFWPMPROVIDER0SetFlags = dll.NewProc("FWPM_PROVIDER0_SetFlags")
	fFWPMPROVIDER0SetDisplayData = dll.NewProc("FWPM_PROVIDER0_SetDisplayData")
	fFWPMPROVIDER0Delete = dll.NewProc("FWPM_PROVIDER0_Delete")

	fWfpSubLayerIsInstalled = dll.NewProc("WfpSubLayerIsInstalledPtr")
	fWfpSubLayerDelete = dll.NewProc("WfpSubLayerDeletePtr")
	fWfpSubLayerAdd = dll.NewProc("WfpSubLayerAdd")
	fFWPMSUBLAYER0Create = dll.NewProc("FWPM_SUBLAYER0_CreatePtr")
	fFWPMSUBLAYER0SetProviderKey = dll.NewProc("FWPM_SUBLAYER0_SetProviderKeyPtr")
	fFWPMSUBLAYER0SetDisplayData = dll.NewProc("FWPM_SUBLAYER0_SetDisplayData")
	fFWPMSUBLAYER0SetWeight = dll.NewProc("FWPM_SUBLAYER0_SetWeight")
	fFWPMSUBLAYER0SetFlags = dll.NewProc("FWPM_SUBLAYER0_SetFlags")
	fFWPMSUBLAYER0Delete = dll.NewProc("FWPM_SUBLAYER0_Delete")

	fFWPMFILTERCreate = dll.NewProc("FWPM_FILTER_CreatePtr")
	fFWPMFILTERDelete = dll.NewProc("FWPM_FILTER_Delete")
	fFWPMFILTERSetProviderKey = dll.NewProc("FWPM_FILTER_SetProviderKeyPtr")
	fFWPMFILTERSetDisplayData = dll.NewProc("FWPM_FILTER_SetDisplayData")
	fFWPMFILTERAllocateConditions = dll.NewProc("FWPM_FILTER_AllocateConditions")
	fFWPMFILTERSetConditionFieldKey = dll.NewProc("FWPM_FILTER_SetConditionFieldKeyPtr")
	fFWPMFILTERSetConditionMatchType = dll.NewProc("FWPM_FILTER_SetConditionMatchType")
	fFWPMFILTERSetConditionV4AddrMask = dll.NewProc("FWPM_FILTER_SetConditionV4AddrMask")
	fFWPMFILTERSetConditionV6AddrMask = dll.NewProc("FWPM_FILTER_SetConditionV6AddrMask")
	fFWPMFILTERSetConditionUINT16 = dll.NewProc("FWPM_FILTER_SetConditionUINT16")
	fFWPMFILTERSetConditionBlobString = dll.NewProc("FWPM_FILTER_SetConditionBlobString")
	fFWPMFILTERSetAction = dll.NewProc("FWPM_FILTER_SetAction")
	fFWPMFILTERSetFlags = dll.NewProc("FWPM_FILTER_SetFlags")
	fWfpFilterAdd = dll.NewProc("WfpFilterAdd")
	fWfpFilterDeleteByID = dll.NewProc("WfpFilterDeleteById")
	fWfpFiltersDeleteByProviderKey = dll.NewProc("WfpFiltersDeleteByProviderKeyPtr")

	return nil
}

func checkDefaultAPIResp(retval uintptr, err error) error {

	if err != syscall.Errno(0) {
		return err
	}
	if retval != 0 {
		return fmt.Errorf("WFP error: 0x%X", retval)
	}
	return nil
}

func catchPanic(err *error) {
	if r := recover(); r != nil {
		log.Error("PANIC (recovered): ", r)
		if e, ok := r.(error); ok {
			*err = e
		} else {
			*err = errors.New(fmt.Sprint(r))
		}
	}
}

// NewGUID - ininialize new random GUID
func NewGUID() syscall.GUID {
	var guid syscall.GUID

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Error("failed to initialize new GUID: ", err)
		return guid
	}

	b[8] = (b[8] | 0x80) & 0xBF
	b[6] = (b[6] | 0x40) & 0x4F

	guid = syscall.GUID{
		Data1: uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]),
		Data2: uint16(b[4])<<8 | uint16(b[5]),
		Data3: uint16(b[6])<<8 | uint16(b[7])}

	copy(guid.Data4[:], b[8:])

	return guid
}
