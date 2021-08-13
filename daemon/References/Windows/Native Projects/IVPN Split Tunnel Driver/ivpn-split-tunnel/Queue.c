/*++

Module Name:

    queue.c

Abstract:

    This file contains the queue entry points and callbacks.

Environment:

    Kernel-mode Driver Framework

--*/

#include "driver.h"
#include "queue.tmh"



#ifdef ALLOC_PRAGMA
#pragma alloc_text (PAGE, QueueInitialize)
#endif


// The I/O dispatch callbacks for the frameworks device object
// are configured in this function.
//
// A single default I/O Queue is configured for parallel request
// processing, and a driver context memory allocation is created
// to hold our structure QUEUE_CONTEXT.
// 
// Arguments:
//    Device - Handle to a framework device object.
NTSTATUS QueueInitialize( _In_ WDFDEVICE Device )
{
    WDFQUEUE queue;
    NTSTATUS status;
    WDF_IO_QUEUE_CONFIG queueConfig;

    PAGED_CODE();
    TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_QUEUE, "%!FUNC! Entry");

    //
    // Configure a default queue so that requests that are not
    // configure-fowarded using WdfDeviceConfigureRequestDispatching to goto
    // other queues get dispatched here.
    //
    WDF_IO_QUEUE_CONFIG_INIT_DEFAULT_QUEUE(
         &queueConfig,
        WdfIoQueueDispatchParallel // WdfIoQueueDispatchSequential ?
        );

    queueConfig.EvtIoDeviceControl = EvtIoDeviceControl;
    queueConfig.PowerManaged = WdfFalse;

    WDF_OBJECT_ATTRIBUTES attributes;
    WDF_OBJECT_ATTRIBUTES_INIT(&attributes);
    // Since we are using Zw function set execution level to passive so that
    // framework ensures that our Io callbacks called at only passive-level
    // even if the request came in at DISPATCH_LEVEL from another driver.
    attributes.ExecutionLevel = WdfExecutionLevelPassive;

    status = WdfIoQueueCreate(
                 Device,
                 &queueConfig,
                 &attributes,
                 &queue
                 );

    if(!NT_SUCCESS(status)) {
        TraceEvents(TRACE_LEVEL_ERROR, TRACE_QUEUE, "%!FUNC! WdfIoQueueCreate failed %!STATUS!", status);
        return status;
    }

    TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_QUEUE, "%!FUNC! Exit (Success)");
    return status;
}

//  This event is invoked when the framework receives IRP_MJ_DEVICE_CONTROL request.
//  Arguments:
//    Queue -  Handle to the framework queue object that is associated with the I/O request.
//    Request - Handle to a framework request object.
//    OutputBufferLength - Size of the output buffer in bytes
//    InputBufferLength - Size of the input buffer in bytes
//    IoControlCode - I/O control code.
VOID
EvtIoDeviceControl(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request,
    _In_ size_t OutputBufferLength,
    _In_ size_t InputBufferLength,
    _In_ ULONG IoControlCode
    )
{
    TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_QUEUE, 
                "%!FUNC! IoControlCode=0x%X (outBuffLen=%d inBuffLen=%d)", 
                IoControlCode, (int)OutputBufferLength, (int)InputBufferLength);

    NTSTATUS status = ProcessEvtIoDeviceControl(Queue, Request, OutputBufferLength, InputBufferLength, IoControlCode);
            
    WdfRequestComplete(Request, status);

    return;
}

NTSTATUS StopAndCleanDriver()
{
    return StopAndClean();
}