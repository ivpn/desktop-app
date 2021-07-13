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

// stop process monitor and splitting
#define IOCTL_STOP_ALL_AND_CFG_CLEAN            0x100 

// Get state of process monitor and splitting (enabled\disabled?) 
#define IOCTL_GET_STATE                         0x200

// start process monitor
#define IOCTL_PROCMON_START                     0x300 
// stop process monitor (splitting also will be stopped)
#define IOCTL_PROCMON_STOP                      0x301
// initialize process monitor about PID+PPID+IMAGE info of running processes
#define IOCTL_PROCMON_SET_PID_DATA              0x302

// Start splitting 
// If the process monitor is not running - it will be also started
// The configuration have to be already initialised
#define IOCTL_SPLITTING_START                   0x400
// stop splitting
#define IOCTL_SPLITTING_STOP                    0x401

// Set/Get configuration: IP addresses 
#define IOCTL_CFG_SET_ADDRESSES                 0x500
#define IOCTL_CFG_GET_ADDRESSES                 0x501

// Set/Get configuration: applications path to split
#define IOCTL_CFG_SET_IMAGES_TO_SPLIT           0x600
#define IOCTL_CFG_GET_IMAGES_TO_SPLIT           0x601
#define IOCTL_CFG_GET_IMAGES_TO_SPLIT_BUFF_SIZE 0x602