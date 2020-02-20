// +build windows

package winlib

import (
	"crypto/rand"
	"errors"
	"fmt"
	"syscall"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/service/platform"
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
	dll = syscall.NewLazyDLL(platform.WindowsWFPDllPath())

	fWfpEngineOpen  = dll.NewProc("WfpEngineOpen")
	fWfpEngineClose = dll.NewProc("WfpEngineClose")

	fCreateWfpSessionObject = dll.NewProc("CreateWfpSessionObject")
	fDeleteWfpSessionObject = dll.NewProc("DeleteWfpSessionObject")

	fWfpTransactionBegin  = dll.NewProc("WfpTransactionBegin")
	fWfpTransactionCommit = dll.NewProc("WfpTransactionCommit")
	fWfpTransactionAbort  = dll.NewProc("WfpTransactionAbort")

	fWfpGetProviderFlags         = dll.NewProc("WfpGetProviderFlagsPtr")
	fWfpProviderDelete           = dll.NewProc("WfpProviderDeletePtr")
	fWfpProviderAdd              = dll.NewProc("WfpProviderAdd")
	fFWPMPROVIDER0Create         = dll.NewProc("FWPM_PROVIDER0_CreatePtr")
	fFWPMPROVIDER0SetFlags       = dll.NewProc("FWPM_PROVIDER0_SetFlags")
	fFWPMPROVIDER0SetDisplayData = dll.NewProc("FWPM_PROVIDER0_SetDisplayData")
	fFWPMPROVIDER0Delete         = dll.NewProc("FWPM_PROVIDER0_Delete")

	fWfpSubLayerIsInstalled      = dll.NewProc("WfpSubLayerIsInstalledPtr")
	fWfpSubLayerDelete           = dll.NewProc("WfpSubLayerDeletePtr")
	fWfpSubLayerAdd              = dll.NewProc("WfpSubLayerAdd")
	fFWPMSUBLAYER0Create         = dll.NewProc("FWPM_SUBLAYER0_CreatePtr")
	fFWPMSUBLAYER0SetProviderKey = dll.NewProc("FWPM_SUBLAYER0_SetProviderKeyPtr")
	fFWPMSUBLAYER0SetDisplayData = dll.NewProc("FWPM_SUBLAYER0_SetDisplayData")
	fFWPMSUBLAYER0SetWeight      = dll.NewProc("FWPM_SUBLAYER0_SetWeight")
	fFWPMSUBLAYER0SetFlags       = dll.NewProc("FWPM_SUBLAYER0_SetFlags")
	fFWPMSUBLAYER0Delete         = dll.NewProc("FWPM_SUBLAYER0_Delete")

	fFWPMFILTERCreate                 = dll.NewProc("FWPM_FILTER_CreatePtr")
	fFWPMFILTERDelete                 = dll.NewProc("FWPM_FILTER_Delete")
	fFWPMFILTERSetProviderKey         = dll.NewProc("FWPM_FILTER_SetProviderKeyPtr")
	fFWPMFILTERSetDisplayData         = dll.NewProc("FWPM_FILTER_SetDisplayData")
	fFWPMFILTERAllocateConditions     = dll.NewProc("FWPM_FILTER_AllocateConditions")
	fFWPMFILTERSetConditionFieldKey   = dll.NewProc("FWPM_FILTER_SetConditionFieldKeyPtr")
	fFWPMFILTERSetConditionMatchType  = dll.NewProc("FWPM_FILTER_SetConditionMatchType")
	fFWPMFILTERSetConditionV4AddrMask = dll.NewProc("FWPM_FILTER_SetConditionV4AddrMask")
	fFWPMFILTERSetConditionV6AddrMask = dll.NewProc("FWPM_FILTER_SetConditionV6AddrMask")
	fFWPMFILTERSetConditionUINT16     = dll.NewProc("FWPM_FILTER_SetConditionUINT16")
	fFWPMFILTERSetConditionBlobString = dll.NewProc("FWPM_FILTER_SetConditionBlobString")
	fFWPMFILTERSetAction              = dll.NewProc("FWPM_FILTER_SetAction")
	fFWPMFILTERSetFlags               = dll.NewProc("FWPM_FILTER_SetFlags")
	fWfpFilterAdd                     = dll.NewProc("WfpFilterAdd")
	fWfpFilterDeleteByID              = dll.NewProc("WfpFilterDeleteById")
	fWfpFiltersDeleteByProviderKey    = dll.NewProc("WfpFiltersDeleteByProviderKeyPtr")
)

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
