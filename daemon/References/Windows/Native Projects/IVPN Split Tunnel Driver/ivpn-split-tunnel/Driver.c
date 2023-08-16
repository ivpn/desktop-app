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

    driver.c

Abstract:

    This file contains the driver entry points and callbacks.

Environment:

    Kernel-mode Driver Framework

--*/

#include "driver.h"
#include "driver.tmh"

#include "Config/GlobalConfigC.h"

#ifdef ALLOC_PRAGMA
#pragma alloc_text (INIT, DriverEntry)
#pragma alloc_text (INIT, NonPnpDeviceAdd)
#pragma alloc_text (PAGE, NonPnpEvtDriverContextCleanup)
#pragma alloc_text (PAGE, NonPnpEvtDriverUnload)
#endif

// Routine Description:
//    DriverEntry initializes the driver and is the first routine called by the
//    system after the driver is loaded. DriverEntry specifies the other entry
//    points in the function driver, such as EvtDevice and DriverUnload.
//
// Parameters Description:
//
//    DriverObject - represents the instance of the function driver that is loaded
//    into memory. DriverEntry must initialize members of DriverObject before it
//    returns to the caller. DriverObject is allocated by the system before the
//    driver is loaded, and it is released by the system after the system unloads
//    the function driver from memory.
//
//    RegistryPath - represents the driver specific path in the Registry.
//    The function driver can use the path to store driver related data between
//    reboots. The path does not store hardware instance specific data.
//
// Return Value:
//
//    STATUS_SUCCESS if successful,
//    STATUS_UNSUCCESSFUL otherwise.

NTSTATUS
DriverEntry(
    _In_ PDRIVER_OBJECT  DriverObject,
    _In_ PUNICODE_STRING RegistryPath
    )
{
    //
    //  [Useful link]
    //  An official example of non-PnP driver from Microsoft:
    //  https://github.com/microsoft/Windows-driver-samples/tree/master/general/ioctl/kmdf
    //

    NTSTATUS status;
    WDF_OBJECT_ATTRIBUTES attributes;

    //
    // Initialize WPP Tracing
    // Do not forget to call 'WPP_CLEANUP(DriverObject);' if driver initialisation failed
    // (normally, we are doing it in 'EvtDriverContextCleanup()') 
    //
    WPP_INIT_TRACING(DriverObject, RegistryPath);

    TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "%!FUNC! Entry");
    
    //
    // Driver configuration.
    // If you are writing a driver for a device that does not support Plugand Play(PnP), the driver must :
    // - Set the WdfDriverInitNonPnpDriver flag in the WDF_DRIVER_CONFIG structure's DriverInitFlags member.
    // - Provide an EvtDriverUnload event callback function.
    // - Create framework device objects that only represent control device objects.
    //
    WDF_DRIVER_CONFIG config;

    WDF_DRIVER_CONFIG_INIT(&config,
        WDF_NO_EVENT_CALLBACK // This is a non-pnp driver
    );
    
    // Tell the framework that this is non-pnp driver so that it doesn't set the default AddDevice routine.    
    config.DriverInitFlags |= WdfDriverInitNonPnpDriver;
    // NonPnp driver must explicitly register an unload routine for the driver to be unloaded.
    config.EvtDriverUnload = NonPnpEvtDriverUnload;
    // A driver-defined pool tag that the framework will assign to all of the driver's pool allocations. Debuggers display this tag. 
    config.DriverPoolTag = POOL_TAG;

    //
    // Register a cleanup callback so that we can call WPP_CLEANUP when
    // the framework driver object is deleted during driver unload.
    //
    WDF_OBJECT_ATTRIBUTES_INIT(&attributes);
    attributes.EvtCleanupCallback = NonPnpEvtDriverContextCleanup;

    //
    // Create a framework driver object to represent our driver.
    //
    WDFDRIVER hDriver;
    status = WdfDriverCreate(DriverObject,
                             RegistryPath,
                             &attributes,
                             &config,
                             &hDriver);

    if (!NT_SUCCESS(status)) {
        TraceEvents(TRACE_LEVEL_ERROR, TRACE_DRIVER, "%!FUNC! WdfDriverCreate failed %!STATUS!", status);
        WPP_CLEANUP(DriverObject);
        return status;
    }
    TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "%!FUNC! WdfDriverCreate success");

    //
    // Creating device object...
    //

    // In order to create a control device, we first need to allocate a
    // WDFDEVICE_INIT structure and set all properties.
    PWDFDEVICE_INIT                deviceInit = NULL;
    deviceInit = WdfControlDeviceInitAllocate(
        hDriver,
        &SDDL_DEVOBJ_SYS_ALL_ADM_ALL // SDDL_DEVOBJ_SYS_ALL_ADM_ALL allows the kernel, system, and administrator complete control over the device. No other users may access the device.
    );

    if (deviceInit == NULL) {
        TraceEvents(TRACE_LEVEL_ERROR, TRACE_DRIVER, "%!FUNC! WdfControlDeviceInitAllocate failed %!STATUS!", status);
        status = STATUS_INSUFFICIENT_RESOURCES;
        WPP_CLEANUP(DriverObject);
        return status;
    }

    status = NonPnpDeviceAdd(hDriver, deviceInit);

    if (!NT_SUCCESS(status)) {
        TraceEvents(TRACE_LEVEL_ERROR, TRACE_DRIVER, "%!FUNC! NonPnpDeviceAdd failed %!STATUS!", status);

        // If the device is created successfully, framework would clear the
        // DeviceInit value. Otherwise device create must have failed so we
        // should free the memory ourself.
        if (deviceInit != NULL) {
            WdfDeviceInitFree(deviceInit);
        }

        WPP_CLEANUP(DriverObject);
        return status;
    }

    //
    // Success
    //
    TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "%!FUNC! Exit (Success)");

    return status;
}

//Routine Description :
//      NonPnpDeviceAdd is manually called from the DriverEntry (because of non-PnP driver).
// Arguments :
//      Driver - Handle to a framework driver object created in DriverEntry
//      DeviceInit - Pointer to a framework - allocated WDFDEVICE_INIT structure.
// Return Value :
//      NTSTATUS
NTSTATUS
NonPnpDeviceAdd(
    IN WDFDRIVER Driver,
    IN PWDFDEVICE_INIT DeviceInit
)
{
    NTSTATUS status;
    UNREFERENCED_PARAMETER(Driver);

    DECLARE_CONST_UNICODE_STRING(DrvDeviceName, DRV_DEVICE_NAME_STRING);
    DECLARE_CONST_UNICODE_STRING(DrvSymbolicLinkName, DRV_SYMBOLIC_NAME_STRING);

    TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "%!FUNC! Entry");

    //
    // Set exclusive to TRUE so that no more than one app can talk to the
    // control device at any time.
    //
    WdfDeviceInitSetExclusive(DeviceInit, TRUE);

    // Skipped initialisation functions (they are not required):
    // WdfDeviceInitSetIoType()                     - because EvtIoRead\EvtIoWrite events not in use
    // WdfControlDeviceInitSetShutdownNotification()- because the queues is not PowerManaged
    // WdfDeviceInitSetFileObjectConfig()           - we do not care  Create, Close and Cleanup requests that gets generated when an application or another
    //                                                kernel component opens an handle to the device
    // WdfDeviceInitSetIoInCallerContextCallback()  - This callback is only required if you are handling method - neither IOCTLs,
    //                                                or want to process requests in the context of the calling process.

    status = WdfDeviceInitAssignName(DeviceInit, &DrvDeviceName);
    if (!NT_SUCCESS(status)) {
        TraceEvents(TRACE_LEVEL_ERROR, TRACE_DRIVER, "%!FUNC! WdfDeviceInitAssignName failed %!STATUS!", status);
        return status;
    }

    //
    // Initialize device context and create device
    //
    WDF_OBJECT_ATTRIBUTES deviceAttributes;
    WDF_OBJECT_ATTRIBUTES_INIT_CONTEXT_TYPE(&deviceAttributes, DEVICE_CONTEXT);
    WDFDEVICE device;
    status = WdfDeviceCreate(&DeviceInit, &deviceAttributes, &device);
    if (!NT_SUCCESS(status)) 
    {
        TraceEvents(TRACE_LEVEL_ERROR, TRACE_DRIVER, "%!FUNC! WdfDeviceCreate failed %!STATUS!", status);
        return status;
    }

    //
    // Create a symbolic link for the control object so that usermode can open the device.
    //
    status = WdfDeviceCreateSymbolicLink(device, &DrvSymbolicLinkName);

    if (!NT_SUCCESS(status)) {
        TraceEvents(TRACE_LEVEL_ERROR, TRACE_DRIVER, "%!FUNC! WdfDeviceCreateSymbolicLink failed %!STATUS!", status);
        return status;
    }

    // 
    // Initialize Queue
    //
    status = QueueInitialize(device);
    if (!NT_SUCCESS(status)) {
        TraceEvents(TRACE_LEVEL_ERROR, TRACE_DRIVER, "%!FUNC! QueueInitialize failed %!STATUS!", status);
        return status;
    }

    // Initialize internal configuration
    status = ConfigurationInit();
    if (!NT_SUCCESS(status)) {
        TraceEvents(TRACE_LEVEL_ERROR, TRACE_DRIVER, "%!FUNC! ConfigurationInit failed %!STATUS!", status);
        return status;
    }
    //
    // Control devices must notify WDF when they are done initializing.   I/O is
    // rejected until this call is made.
    //
    WdfControlFinishInitializing(device);

    TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "%!FUNC! Exit  (Success)");

    return status;
}

// Routine Description :
//      Free all the resources allocated in DriverEntry.
// Arguments:
//  DriverObject - handle to a WDF Driver object.
VOID NonPnpEvtDriverContextCleanup(_In_ WDFOBJECT DriverObject)
{
    PAGED_CODE();
    TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "%!FUNC! Entry");

    // Stop WPP Tracing
    WPP_CLEANUP(WdfDriverWdmGetDriverObject((WDFDRIVER)DriverObject));
}

// Routine Description :
//      Called by the I / O subsystem just before unloading the driver.
//      You can free the resources created in the DriverEntry either
//      in this routine or in the EvtDriverContextCleanup callback.
// Arguments :
//      Driver - Handle to a framework driver object created in DriverEntry
// Return Value :
//      NTSTATUS
void NonPnpEvtDriverUnload(WDFDRIVER Driver)
{
    UNREFERENCED_PARAMETER(Driver);
    PAGED_CODE();
    TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "%!FUNC! Entry");
    StopAndCleanDriver();
}