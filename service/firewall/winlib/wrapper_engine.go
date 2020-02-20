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
