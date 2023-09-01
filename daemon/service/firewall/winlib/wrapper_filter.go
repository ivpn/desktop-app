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

// FwpMatchType - FWP_MATCH_TYPE
type FwpMatchType uint32

// FWP_MATCH_TYPE
const (
	FwpMatchEqual                FwpMatchType = 0
	FwpMatchGreater              FwpMatchType = FwpMatchEqual + 1
	FwpMatchLess                 FwpMatchType = FwpMatchGreater + 1
	FwpMatchGreaterOrEqual       FwpMatchType = FwpMatchLess + 1
	FwpMatchLessOrEqual          FwpMatchType = FwpMatchGreaterOrEqual + 1
	FwpMatchRange                FwpMatchType = FwpMatchLessOrEqual + 1
	FwpMatchFlagsAllSet          FwpMatchType = FwpMatchRange + 1
	FwpMatchFlagsAnySet          FwpMatchType = FwpMatchFlagsAllSet + 1
	FwpMatchFlagsNonESet         FwpMatchType = FwpMatchFlagsAnySet + 1
	FwpMatchEqualCaseInsensitive FwpMatchType = FwpMatchFlagsNonESet + 1
	FwpMatchNotEqual             FwpMatchType = FwpMatchEqualCaseInsensitive + 1
	FwpMatchPrefix               FwpMatchType = FwpMatchNotEqual + 1
	FwpMatchNotPrefix            FwpMatchType = FwpMatchPrefix + 1
	FwpMatchTypeMax              FwpMatchType = FwpMatchNotPrefix + 1
)

// FwpActionType - FWP_ACTION_TYPE
type FwpActionType uint32

// FWP_ACTION_TYPE
const (
	FwpActionBlock              FwpActionType = 0x1001
	FwpActionPermit             FwpActionType = 0x1002
	FwpActionCalloutTerminating FwpActionType = 0x5003
	FwpActionCalloutInspection  FwpActionType = 0x6004
	FwpActionCalloutUnknown     FwpActionType = 0x4005
	FwpActionContinue           FwpActionType = 0x2006
	FwpActionNone               FwpActionType = 0x0007
	FwpActionNoneNoMatch        FwpActionType = 0x0008
)

// FwpmFilterFlags - FWPM_FILTER_FLAGS uint32
type FwpmFilterFlags uint32

// FWPM_FILTER_FLAGS
const (
	FwpmFilterFlagNone                       FwpmFilterFlags = 0x00000000
	FwpmFilterFlagPersistent                 FwpmFilterFlags = 0x00000001
	FwpmFilterFlagBoottime                   FwpmFilterFlags = 0x00000002
	FwpmFilterFlagHasProviderContext         FwpmFilterFlags = 0x00000004
	FwpmFilterFlagClearActionRight           FwpmFilterFlags = 0x00000008
	FwpmFilterFlagPermitIfCalloutUregistered FwpmFilterFlags = 0x00000010
	FwpmFilterFlagDisabled                   FwpmFilterFlags = 0x00000020
	FwpmFilterFlagIndexed                    FwpmFilterFlags = 0x00000040
)

// FWPMFILTERCreate creates WFP filter
func FWPMFILTERCreate(filterGUID syscall.GUID, layerGUID syscall.GUID, subLayerGUID syscall.GUID,
	weight uint8, flags FwpmFilterFlags) (filter syscall.Handle, err error) {

	defer catchPanic(&err)

	filterPtr, _, err := fFWPMFILTERCreate.Call(
		uintptr(unsafe.Pointer(&filterGUID)),
		uintptr(unsafe.Pointer(&layerGUID)),
		uintptr(unsafe.Pointer(&subLayerGUID)),
		uintptr(weight),
		uintptr(flags))

	if filterPtr == 0 {
		if err != syscall.Errno(0) {
			return 0, err
		}

		return 0, errors.New("failed to create WFP provider object")
	}

	return syscall.Handle(filterPtr), nil
}

// FWPMFILTERDelete deletes WFP filter
func FWPMFILTERDelete(filter syscall.Handle) (err error) {
	defer catchPanic(&err)

	_, _, err = fFWPMFILTERDelete.Call(uintptr(filter))
	if err != syscall.Errno(0) {
		return err
	}

	return nil
}

// FWPMFILTERSetProviderKey sets provider to a filter
func FWPMFILTERSetProviderKey(filter syscall.Handle, providerGUID syscall.GUID) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMFILTERSetProviderKey.Call(uintptr(filter), uintptr(unsafe.Pointer(&providerGUID)))
	return checkDefaultAPIResp(retval, err)
}

// FWPMFILTERSetDisplayData sets display data to a filter
func FWPMFILTERSetDisplayData(filter syscall.Handle, name string, description string) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMFILTERSetDisplayData.Call(uintptr(filter),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(description))))

	return checkDefaultAPIResp(retval, err)
}

// FWPMFILTERAllocateConditions allocates conditions
func FWPMFILTERAllocateConditions(filter syscall.Handle, numFilterConditions uint32) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMFILTERAllocateConditions.Call(uintptr(filter), uintptr(numFilterConditions))
	return checkDefaultAPIResp(retval, err)
}

// FWPMFILTERSetConditionFieldKey sets conditions parameters
func FWPMFILTERSetConditionFieldKey(filter syscall.Handle, conditionIndex uint32, fieldGUID syscall.GUID) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMFILTERSetConditionFieldKey.Call(uintptr(filter),
		uintptr(conditionIndex),
		uintptr(unsafe.Pointer(&fieldGUID)))
	return checkDefaultAPIResp(retval, err)
}

// FWPMFILTERSetConditionMatchType sets conditions parameters
func FWPMFILTERSetConditionMatchType(filter syscall.Handle, conditionIndex uint32, matchType FwpMatchType) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMFILTERSetConditionMatchType.Call(uintptr(filter),
		uintptr(conditionIndex),
		uintptr(matchType))
	return checkDefaultAPIResp(retval, err)
}

// FWPMFILTERSetConditionV4AddrMask sets conditions parameters
func FWPMFILTERSetConditionV4AddrMask(filter syscall.Handle, conditionIndex uint32, address uint32, mask uint32) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMFILTERSetConditionV4AddrMask.Call(uintptr(filter),
		uintptr(conditionIndex),
		uintptr(address),
		uintptr(mask))
	return checkDefaultAPIResp(retval, err)
}

// FWPMFILTERSetConditionV6AddrMask sets conditions parameters
func FWPMFILTERSetConditionV6AddrMask(filter syscall.Handle, conditionIndex uint32,
	address [16]byte, prefixLen byte) (err error) {

	defer catchPanic(&err)

	retval, _, err := fFWPMFILTERSetConditionV6AddrMask.Call(uintptr(filter),
		uintptr(conditionIndex),
		uintptr(unsafe.Pointer(&address[0])),
		uintptr(prefixLen))

	return checkDefaultAPIResp(retval, err)
}

// FWPMFILTERSetConditionUINT16 sets conditions parameters
func FWPMFILTERSetConditionUINT16(filter syscall.Handle, conditionIndex uint32, val uint16) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMFILTERSetConditionUINT16.Call(uintptr(filter),
		uintptr(conditionIndex),
		uintptr(val))
	return checkDefaultAPIResp(retval, err)
}

// FWPMFILTERSetConditionBlobString sets conditions parameters
func FWPMFILTERSetConditionBlobString(filter syscall.Handle, conditionIndex uint32, val string) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMFILTERSetConditionBlobString.Call(uintptr(filter),
		uintptr(conditionIndex),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(val))))
	return checkDefaultAPIResp(retval, err)
}

// FWPMFILTERSetAction sets actions for filter
func FWPMFILTERSetAction(filter syscall.Handle, action FwpActionType) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMFILTERSetAction.Call(uintptr(filter), uintptr(action))
	return checkDefaultAPIResp(retval, err)
}

// FWPMFILTERSetFlags sets filter
func FWPMFILTERSetFlags(filter syscall.Handle, flags uint32) (err error) {
	defer catchPanic(&err)

	retval, _, err := fFWPMFILTERSetFlags.Call(uintptr(filter), uintptr(flags))
	return checkDefaultAPIResp(retval, err)
}

// WfpFilterAdd adds filter
func WfpFilterAdd(engine syscall.Handle, filter syscall.Handle) (id uint64, err error) {
	defer catchPanic(&err)

	id = 0

	retval, _, err := fWfpFilterAdd.Call(uintptr(engine),
		uintptr(filter),
		uintptr(unsafe.Pointer(&id)))

	if err != syscall.Errno(0) {
		return 0, err
	}
	if retval != 0 {
		return 0, fmt.Errorf("WFP error: 0x%X", retval)
	}

	return id, nil
}

// WfpFilterDeleteByID remove filter by id
func WfpFilterDeleteByID(engine syscall.Handle, id uint64) (err error) {
	defer catchPanic(&err)

	retval, _, err := fWfpFilterDeleteByID.Call(uintptr(engine), uintptr(id))
	return checkDefaultAPIResp(retval, err)
}

// WfpFiltersDeleteByProviderKey remove filter by provider
func WfpFiltersDeleteByProviderKey(engine syscall.Handle, providerGUID syscall.GUID, layerGUID syscall.GUID) (err error) {
	defer catchPanic(&err)

	retval, _, err := fWfpFiltersDeleteByProviderKey.Call(uintptr(engine),
		uintptr(unsafe.Pointer(&providerGUID)),
		uintptr(unsafe.Pointer(&layerGUID)))

	return checkDefaultAPIResp(retval, err)
}
