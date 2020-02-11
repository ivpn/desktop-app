// +build windows

package kernel32

import (
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	_dll       = windows.NewLazySystemDLL("kernel32.dll")
	_fSetEvent = _dll.NewProc("SetEvent")
)

// SetEvent - Sets the specified event object to the signaled state.
// https://docs.microsoft.com/en-us/windows/win32/api/synchapi/nf-synchapi-setevent
func SetEvent(evt syscall.Handle) (bool, error) {
	retval, _, err := _fSetEvent.Call(uintptr(evt))
	if err != syscall.Errno(0) {
		return false, err
	}
	return retval != 0, nil
}
