#pragma once

#include "../Driver.h"
#include "../trace.h"

EXTERN_C_START

NTSTATUS StopAndClean();

NTSTATUS ProcessEvtIoDeviceControl(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request,
    _In_ size_t OutputBufferLength,
    _In_ size_t InputBufferLength,
    _In_ ULONG IoControlCode
);

EXTERN_C_END