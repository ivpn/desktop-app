//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
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

// +build windows

package process

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ivpn/desktop-app-daemon/oshelpers/windows/iphlpapi"
	"golang.org/x/sys/windows"
)

/*
#cgo LDFLAGS: -lpsapi

#include <windows.h> // Kernel32.lib
#include <psapi.h> // For access to GetModuleFileNameEx

char* GetProcFullPath(int pid)
{
  HANDLE processHandle = NULL;
  char filename[MAX_PATH] = {0};

  processHandle = OpenProcess(PROCESS_QUERY_INFORMATION | PROCESS_VM_READ, FALSE, pid);
  if (processHandle != NULL) {
    GetModuleFileNameEx(processHandle, NULL, filename, MAX_PATH);
    CloseHandle(processHandle);
  }
  return strdup (filename);
}
*/
import "C"

// doGetPortOwnerPID returns PID of a process which is an owning of local TCP port
func doGetPortOwnerPID(localTCPPort int) (int, error) {

	var buf = []byte{0}
	bufLen := uint32(0)

	for i := 0; ; i++ { // 5 retries (determine required buffer size on first retry)
		ret, err := iphlpapi.GetExtendedTCPTable(buf, &bufLen, true, windows.AF_INET, iphlpapi.TCPTableOwnerPidConnections)
		if err != nil {
			return 0, fmt.Errorf("GetExtendedTCPTable error: %w", err)
		}
		if ret == syscall.ERROR_INSUFFICIENT_BUFFER {
			if i > 5 {
				return 0, fmt.Errorf("GetExtendedTCPTable ERROR_INSUFFICIENT_BUFFER retries limit")
			}
			buf = make([]byte, bufLen)
			continue
		}
		if ret == windows.DS_S_SUCCESS {
			break
		}
		return 0, fmt.Errorf("GetExtendedTCPTable return value：%v", ret)
	}

	//type MibTCPTableOwnerPid struct {
	//	dwNumEntries uint32
	//	table        []MibTCPRowOwnerPid
	//}
	dwNumEntries := *(*uint32)(unsafe.Pointer(&buf[0]))
	rowsStartPtr := uintptr(unsafe.Pointer(&buf[0])) + unsafe.Sizeof(dwNumEntries)
	rowSize := unsafe.Sizeof(iphlpapi.MibTCPRowOwnerPid{})

	if len(buf) < int((unsafe.Sizeof(dwNumEntries) + rowSize*uintptr(dwNumEntries))) {
		return 0, fmt.Errorf("System error: GetExtendedTCPTable returns the number is too long, beyond the buffer。")
	}
	var row iphlpapi.MibTCPRowOwnerPid
	for i := uint32(0); i < dwNumEntries; i++ {
		row = *((*iphlpapi.MibTCPRowOwnerPid)(unsafe.Pointer(rowsStartPtr + (rowSize * uintptr(i)))))

		localPort := uint16(uint16(row.DwLocalPort[0])<<8 | uint16(row.DwLocalPort[1]))

		if localPort == uint16(localTCPPort) {
			return int(row.DwOwningPid), nil
		}
	}

	return 0, fmt.Errorf("owner PID for local port %d not found", localTCPPort)
}

// doGetBinaryPathByPID returns absolute path of process binary
func doGetBinaryPathByPID(pid int) (string, error) {
	fpathC := C.GetProcFullPath(C.int(pid))
	fpath := C.GoString(fpathC)
	C.free(unsafe.Pointer(fpathC))
	return fpath, nil
}
