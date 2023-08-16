//
//  Split-Tunnel Driver for Windows
//  https://github.com/ivpn/desktop-app/daemon/References/Windows/Native%20Projects/IVPN%20Split%20Tunnel%20Driver
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
//
//  This file is part of the IVPN Client for Desktop project.
//  https://github.com/ivpn/desktop-app
//
//  The IVPN Client for Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The IVPN Client for Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the IVPN Client for Desktop. If not, see <https://www.gnu.org/licenses/>.
//

/*++

Module Name:

    public.h

Abstract:

    This module contains the common declarations shared by driver
    and user applications.

Environment:

    user and kernel

--*/

//
// Device name to be accessible by the applications (CreateFileW())
//
#define DEVICE_NAME_PUBLIC L"\\\\.\\IVPNSPLITTUNNEL"

//
// Driver registration params
//
#define DRV_DEVICE_NAME_STRING      L"\\Device\\IVPNSPLITTUNNEL"
#define DRV_SYMBOLIC_NAME_STRING    L"\\Global??\\IVPNSPLITTUNNEL"
//
// A driver-defined pool tag that the framework will assign to all of the driver's pool allocations. Debuggers display this tag. 
//
#define POOL_TAG                    'IVPN' 

//
// The IOCTL function codes.
//

// Stop and clean everything:
//		Stop splitting
//		Stop processes monitoring 
//		Clean all configuration/statuses
#define IOCTL_STOP_ALL_AND_CFG_CLEAN            0x100 

// Get state of process monitor and splitting (enabled\disabled?) 
#define IOCTL_GET_STATE                         0x200

// start process monitor
#define IOCTL_PROCMON_START                     0x300 
// stop process monitor (splitting also will be stopped)
#define IOCTL_PROCMON_STOP                      0x301
// Set application PID\PPIDs which have to be splitted.
// It adds new info to internal process tree but not erasing current known PID\PPIDs.
// Operaion fails when 'process monitor' not running
#define IOCTL_PROCMON_SET_PID_DATA              0x302

// Start splitting.
// If "process monitor" not running - it will be started.
//	
// Operation fails when configuration not complete:
//		- no splitting apps defined
//		- no IP configuration (IP-public + IP-tunnel) defined at least for one protocol type (IPv4\IPv6)
// 
// If only IPv4 configuration defined - splitting will work only for IPv4
// If only IPv6 configuration defined - splitting will work only for IPv6
#define IOCTL_SPLITTING_START                   0x400
// stop splitting
#define IOCTL_SPLITTING_STOP                    0x401

// Set/Get configuration: IP addresses 
#define IOCTL_CFG_SET_ADDRESSES                 0x500
#define IOCTL_CFG_GET_ADDRESSES                 0x501

// Update applications (full paths) which have to be monitored (splitted).
// The current configuration will remain unchanged. Will be added only new elements.
#define IOCTL_CFG_SET_IMAGES_TO_SPLIT           0x600
#define IOCTL_CFG_GET_IMAGES_TO_SPLIT           0x601
#define IOCTL_CFG_GET_IMAGES_TO_SPLIT_BUFF_SIZE 0x602