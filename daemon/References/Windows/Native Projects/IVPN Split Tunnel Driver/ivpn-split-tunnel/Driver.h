#pragma once
/*++

Module Name:

    driver.h

Abstract:

    This file contains the driver definitions.

Environment:

    Kernel-mode Driver Framework

--*/

#include <ntddk.h>
#include <wdf.h>
#include <initguid.h>

#include "devicecontext.h"
#include "queue.h"
#include "trace.h"

EXTERN_C_START

//
// WDFDRIVER Events
//

DRIVER_INITIALIZE DriverEntry;
EVT_WDF_OBJECT_CONTEXT_CLEANUP NonPnpEvtDriverContextCleanup;
EVT_WDF_DRIVER_UNLOAD NonPnpEvtDriverUnload;

// Don't use EVT_WDF_DRIVER_DEVICE_ADD for NonPnpDeviceAdd even though 
// the signature is same because this is not an event called by the 
// framework.
NTSTATUS NonPnpDeviceAdd( IN WDFDRIVER Driver, IN PWDFDEVICE_INIT DeviceInit );

EXTERN_C_END
