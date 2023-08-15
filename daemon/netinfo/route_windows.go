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

package netinfo

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ivpn/desktop-app/daemon/oshelpers/windows/iphlpapi"
)

func getWindowsIPv4Routes() ([]iphlpapi.APIMibIPForwardRow, error) {
	var buf = []byte{0}
	bufLen := uint32(0)

	// determine required buffer size
	ret, err := iphlpapi.GetIPForwardTable(buf, &bufLen, false)
	if err != nil {
		return nil, fmt.Errorf("failed to read IP routes: %w", err)
	}
	if ret != syscall.ERROR_INSUFFICIENT_BUFFER {
		return nil, fmt.Errorf("Failed to get the routing table, return value：%v", ret)
	}

	for i := 0; i < 5; i++ {
		buf = make([]byte, bufLen)

		// get route table
		ret, err = iphlpapi.GetIPForwardTable(buf, &bufLen, false)
		if err != nil {
			return nil, fmt.Errorf("failed to read IP routes: %w", err)
		}
		// If buffer is too small - try again
		if ret == syscall.ERROR_INSUFFICIENT_BUFFER {
			continue
		}
		break
	}

	if ret != syscall.Errno(0) {
		return nil, fmt.Errorf("Failed to get the routing table, return value：%v", ret)
	}

	//Returned structure located in 'buf':
	//	typedef struct _MIB_IPFORWARDTABLE
	//	{
	//		DWORD            dwNumEntries; 		// The number of route entries in the table.
	//		MIB_IPFORWARDROW table[ANY_SIZE]; 	// A pointer to a table of route entries implemented as an array of MIB_IPFORWARDROW structures.
	//  }
	num := *(*uint32)(unsafe.Pointer(&buf[0]))

	routes := make([]iphlpapi.APIMibIPForwardRow, num)
	sr := unsafe.Pointer(uintptr(unsafe.Pointer(&buf[0])) + unsafe.Sizeof(num))
	rowSize := unsafe.Sizeof(iphlpapi.APIMibIPForwardRow{})

	if len(buf) < int((unsafe.Sizeof(num) + rowSize*uintptr(num))) {
		return nil, fmt.Errorf("System error: GetIpForwardTable returns the number is too long, beyond the buffer。")
	}

	for i := uint32(0); i < num; i++ {
		routes[i] = *((*iphlpapi.APIMibIPForwardRow)(unsafe.Pointer(uintptr(sr) + (rowSize * uintptr(i)))))
	}

	return routes, nil
}
