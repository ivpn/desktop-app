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

package iphlpapi

import (
	"encoding/binary"
	"net"
	"syscall"
	"unsafe"

	"github.com/ivpn/desktop-app/daemon/logger"
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
	_fGetExtendedTcpTable  = _dll.NewProc("GetExtendedTcpTable")
)

// APINotifyRouteChange - The NotifyRouteChange function causes a notification to be sent to the caller whenever a change occurs in the IPv4 routing table.
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

// APIGetBestRoute - The GetBestRoute function retrieves the best route to the specified destination IP address.
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

// GetExtendedTCPTable - function retrieves a table that contains a list of TCP endpoints available to the application.
// https://docs.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getextendedtcptable
func GetExtendedTCPTable(tcpTable []byte, pdwSize *uint32, order bool, afType int, tableClass TCPTableClass) (r syscall.Errno, err error) {
	defer catchPanic(&err)

	var sortOrder int
	if order {
		sortOrder = 1
	}

	retval, _, err := _fGetExtendedTcpTable.Call(
		uintptr(unsafe.Pointer(&tcpTable[0])),
		uintptr(unsafe.Pointer(pdwSize)),
		uintptr(sortOrder),
		uintptr(afType),
		uintptr(tableClass), 0)

	if err != syscall.Errno(0) {
		return syscall.Errno(retval), err
	}

	return syscall.Errno(retval), nil
}
