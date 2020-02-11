// +build windows

package iphlpapi

import (
	"encoding/binary"
	"ivpn/daemon/logger"
	"net"
	"syscall"
	"unsafe"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("iphlpW")
}

var (
	_dll                   = syscall.NewLazyDLL("iphlpapi.dll")
	_fNotifyRouteChange    = _dll.NewProc("NotifyRouteChange")
	_fCancelIPChangeNotify = _dll.NewProc("CancelIPChangeNotify")
	_fGetBestRoute         = _dll.NewProc("GetBestRoute")
	_fGetIPForwardTable    = _dll.NewProc("GetIpForwardTable")
)

// APINotifyRouteChange - The GetBestRoute function retrieves the best route to the specified destination IP address.
// https://docs.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-notifyroutechange
func APINotifyRouteChange(handle *syscall.Handle, overlapped *syscall.Overlapped) (err error) {
	defer catchPanic(&err)

	retval, _, err := _fNotifyRouteChange.Call(uintptr(unsafe.Pointer(handle)), uintptr(unsafe.Pointer(overlapped)))
	if err == syscall.ERROR_IO_PENDING {
		return nil
	}

	return checkDefaultAPIResp(retval, err)
}

// CancelIPChangeNotify - The CancelIPChangeNotify function cancels notification of IPv4 address and route changes previously requested with successful calls to the NotifyAddrChange or NotifyRouteChange functions.
// https://docs.microsoft.com/ru-ru/windows/win32/api/iphlpapi/nf-iphlpapi-cancelipchangenotify
func CancelIPChangeNotify(overlapped *syscall.Overlapped) (err error) {
	defer catchPanic(&err)

	retval, _, err := _fCancelIPChangeNotify.Call(uintptr(unsafe.Pointer(overlapped)))
	return checkDefaultAPIResp(retval, err)
}

// APIGetBestRoute - The NotifyRouteChange function causes a notification to be sent to the caller whenever a change occurs in the IPv4 routing table.
// https://docs.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getbestroute
func APIGetBestRoute(dwDestAddr net.IP, dwSourceAddr net.IP, bestRoute *APIMibIPForwardRow) (err error) {
	defer catchPanic(&err)

	dest := binary.BigEndian.Uint32(dwDestAddr.To4())
	sorc := binary.BigEndian.Uint32(dwSourceAddr.To4())

	retval, _, err := _fGetBestRoute.Call(uintptr(dest), uintptr(sorc), uintptr(unsafe.Pointer(bestRoute)))

	return checkDefaultAPIResp(retval, err)
}

// GetIPForwardTable - The GetIpForwardTable function retrieves the IPv4 routing table.
// https://docs.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getipforwardtable
func GetIPForwardTable(pIPForwardTable []byte, pdwSize *uint32, bOrder bool) (r syscall.Errno, err error) {
	defer catchPanic(&err)

	var order int
	if bOrder {
		order = 1
	}

	retval, _, err := _fGetIPForwardTable.Call(
		uintptr(unsafe.Pointer(&pIPForwardTable[0])),
		uintptr(unsafe.Pointer(pdwSize)),
		uintptr(order))

	if err != syscall.Errno(0) {
		return syscall.Errno(retval), err
	}

	return syscall.Errno(retval), nil
}
