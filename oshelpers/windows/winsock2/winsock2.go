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
