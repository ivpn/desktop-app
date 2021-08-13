#include "Requests.h"
#include "Requests.tmh"

#include "../ProcessMonitor/ProcessMonitor.h"
#include "../WFP/Firewall.h"
#include "../Utils/Timer.h"
#include "../Config/GlobalConfig.h"

typedef NTSTATUS(*ProcessIoctlFunc)(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request);

NTSTATUS StopAndClean()
{
    NTSTATUS retState = STATUS_SUCCESS;
    NTSTATUS status;

    status = prc::Stop();
    if (!NT_SUCCESS(status))
        retState = status;

    status = wfp::Stop();
    if (!NT_SUCCESS(status))
        retState = status;

    status = cfg::Clean();
    if (!NT_SUCCESS(status))
        retState = status;

    return retState;
}

NTSTATUS Process_IOCTL_STOP_ALL_AND_CFG_CLEAN(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Queue);
    UNREFERENCED_PARAMETER(Request);

    return StopAndClean();
}

NTSTATUS Process_IOCTL_GET_STATE(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Queue);

    PVOID   outBuf = NULL;
    NTSTATUS status = WdfRequestRetrieveOutputBuffer(Request, sizeof(DriverStatus), &outBuf, NULL);
    if (!NT_SUCCESS(status)) {
        return status;
    }

    DriverStatus s = {};
    s.IsConfigOk = cfg::IsConfigOk();
    s.IsEnabledProcessMonitor = prc::IsRunning();
    s.IsEnabledSplitting = wfp::IsRunning();

    RtlCopyMemory(outBuf, &s, sizeof(DriverStatus));

    // Assign the length of the data copied to IoStatus.Information
    // of the request and complete the request.
    WdfRequestSetInformation(Request, sizeof(DriverStatus));

    // When the request is completed the content of the SystemBuffer
    // is copied to the User output buffer and the SystemBuffer is
    // is freed.

    return STATUS_SUCCESS;
}


NTSTATUS Process_IOCTL_PROCMON_START(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Queue);
    UNREFERENCED_PARAMETER(Request);

    if (prc::IsRunning())
        return STATUS_SUCCESS;

    if (cfg::GetIsNoImagesToSplit())
        return STATUS_DEVICE_CONFIGURATION_ERROR;

    return prc::Start();
}


NTSTATUS Process_IOCTL_PROCMON_STOP(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Queue);
    UNREFERENCED_PARAMETER(Request);
    return prc::Stop();
}


NTSTATUS Process_IOCTL_PROCMON_SET_PID_DATA(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Queue);

    if (!prc::IsRunning())
        return STATUS_INVALID_DEVICE_STATE;

    PVOID buffer;
    size_t bufferLength;

    NTSTATUS status = WdfRequestRetrieveInputBuffer(Request, sizeof(DWORD)*2, &buffer, &bufferLength);
    if (!NT_SUCCESS(status))
        return status;

    if (bufferLength%(sizeof(DWORD)*2))
        return STATUS_INVALID_PARAMETER;

    size_t elementsCnt = bufferLength / (sizeof(DWORD) * 2);

    return prc::InitPIDs((DWORD*)buffer, elementsCnt);
}

// ----------------------------------------

NTSTATUS Process_IOCTL_SPLITTING_START(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Request);

    if (!cfg::IsConfigOk())
        return STATUS_DEVICE_CONFIGURATION_ERROR;

    WDFDEVICE wdfDevice = WdfIoQueueGetDevice(Queue);

    NTSTATUS status = prc::Start();
    if (!NT_SUCCESS(status))
        return status;

    return wfp::Start(wdfDevice);
}

NTSTATUS Process_IOCTL_SPLITTING_STOP(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Queue);
    UNREFERENCED_PARAMETER(Request);

    return wfp::Stop();
}

NTSTATUS Process_IOCTL_CFG_SET_ADDRESSES(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Queue);

    PVOID buffer;
    size_t bufferLength;
    
    NTSTATUS status = WdfRequestRetrieveInputBuffer(Request, sizeof(IPAddrConfig), &buffer, &bufferLength);
    if (!NT_SUCCESS(status))
        return status;

    if (bufferLength != sizeof(IPAddrConfig))
        return STATUS_INVALID_PARAMETER;

    IPAddrConfig ipCfg;
    RtlCopyMemory(&ipCfg, buffer, bufferLength);
        
    TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) IPv4: pub(%d.%d.%d.%d) tun(%d.%d.%d.%d) IPv6: pub(%x:%x:%x:%x:%x:%x:%x:%x) tun(%x:%x:%x:%x:%x:%x:%x:%x)",
        ipCfg.IPv4Public.S_un.S_un_b.s_b1, ipCfg.IPv4Public.S_un.S_un_b.s_b2, ipCfg.IPv4Public.S_un.S_un_b.s_b3, ipCfg.IPv4Public.S_un.S_un_b.s_b4,
        ipCfg.IPv4Tunnel.S_un.S_un_b.s_b1, ipCfg.IPv4Tunnel.S_un.S_un_b.s_b2, ipCfg.IPv4Tunnel.S_un.S_un_b.s_b3, ipCfg.IPv4Tunnel.S_un.S_un_b.s_b4,
        ipCfg.IPv6Public.u.Word[0], ipCfg.IPv6Public.u.Word[1], ipCfg.IPv6Public.u.Word[2], ipCfg.IPv6Public.u.Word[3],
        ipCfg.IPv6Public.u.Word[4], ipCfg.IPv6Public.u.Word[5], ipCfg.IPv6Public.u.Word[6], ipCfg.IPv6Public.u.Word[7],
        ipCfg.IPv6Tunnel.u.Word[0], ipCfg.IPv6Tunnel.u.Word[1], ipCfg.IPv6Tunnel.u.Word[2], ipCfg.IPv6Tunnel.u.Word[3],
        ipCfg.IPv6Tunnel.u.Word[4], ipCfg.IPv6Tunnel.u.Word[5], ipCfg.IPv6Tunnel.u.Word[6], ipCfg.IPv6Tunnel.u.Word[7]
    );

    cfg::SetIPs(ipCfg);

    return STATUS_SUCCESS;
}

NTSTATUS Process_IOCTL_CFG_GET_ADDRESSES(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Queue);

    PVOID   outBuf = NULL;
    NTSTATUS status = WdfRequestRetrieveOutputBuffer(Request, sizeof(IPAddrConfig), &outBuf, NULL);
    if (!NT_SUCCESS(status)) {
        return status;
    }

    IPAddrConfig ips = cfg::GetIPs();

    RtlCopyMemory(outBuf, &ips, sizeof(IPAddrConfig));

    // Assign the length of the data copied to IoStatus.Information
    // of the request and complete the request.
    WdfRequestSetInformation(Request, sizeof(IPAddrConfig));

    // When the request is completed the content of the SystemBuffer
    // is copied to the User output buffer and the SystemBuffer is
    // is freed.

    return STATUS_SUCCESS;
}

NTSTATUS Process_IOCTL_CFG_SET_IMAGES_TO_SPLIT(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Queue);

    PVOID buffer;
    size_t bufferLength;

    NTSTATUS status = WdfRequestRetrieveInputBuffer(Request, 0, &buffer, &bufferLength);
    if (!NT_SUCCESS(status))
        return status;

    if (bufferLength > 0xffffffff) // DWORD max size
        return STATUS_INVALID_PARAMETER;

    return cfg::SetImagesToSplit((char*)buffer, (DWORD) bufferLength);
}

NTSTATUS Process_IOCTL_CFG_GET_IMAGES_TO_SPLIT(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Queue);

    PVOID   outBuf = NULL;
    size_t  len = 0;
    NTSTATUS status = WdfRequestRetrieveOutputBuffer(Request, 0, &outBuf, &len);
    if (!NT_SUCCESS(status)) {
        return status;
    }
    DWORD bufSize = (DWORD)len;

    status = cfg::GetImagesToSplit((char*)outBuf, &bufSize);
    if (!NT_SUCCESS(status)) {
        return status;
    }

    // Assign the length of the data copied to IoStatus.Information
    // of the request and complete the request.
    WdfRequestSetInformation(Request, bufSize);

    // When the request is completed the content of the SystemBuffer
    // is copied to the User output buffer and the SystemBuffer is
    // is freed.
    return STATUS_SUCCESS;
}

NTSTATUS Process_IOCTL_CFG_GET_IMAGES_TO_SPLIT_BUFF_SIZE(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request)
{
    UNREFERENCED_PARAMETER(Queue);

    PVOID   outBuf = NULL;
    size_t  len = 0;
    NTSTATUS status = WdfRequestRetrieveOutputBuffer(Request, sizeof(DWORD), &outBuf, &len);
    if (!NT_SUCCESS(status)) {
        return status;
    }

    DWORD minBufSize = 0;
    cfg::GetImagesToSplit(NULL, &minBufSize);

    *(DWORD*)outBuf = (DWORD)minBufSize;

    // Assign the length of the data copied to IoStatus.Information
    // of the request and complete the request.
    WdfRequestSetInformation(Request, sizeof(DWORD));

    // When the request is completed the content of the SystemBuffer
    // is copied to the User output buffer and the SystemBuffer is
    // is freed.
    return STATUS_SUCCESS;
}
// =======================================================================

NTSTATUS ProcessEvtIoDeviceControl(
    _In_ WDFQUEUE Queue,
    _In_ WDFREQUEST Request,
    _In_ size_t OutputBufferLength,
    _In_ size_t InputBufferLength,
    _In_ ULONG IoControlCode)
{
    UNREFERENCED_PARAMETER(OutputBufferLength);
    UNREFERENCED_PARAMETER(InputBufferLength);

    DEBUG_PrintElapsedTimeEx(5);

    ProcessIoctlFunc funcToProcess = nullptr;

    switch (IoControlCode) 
    {

    case IOCTL_STOP_ALL_AND_CFG_CLEAN:
        funcToProcess = Process_IOCTL_STOP_ALL_AND_CFG_CLEAN;
        break;

    case IOCTL_GET_STATE:
        funcToProcess = Process_IOCTL_GET_STATE;
        break;

    case IOCTL_PROCMON_START:
        funcToProcess = Process_IOCTL_PROCMON_START;
        break;
    case IOCTL_PROCMON_STOP:
        funcToProcess = Process_IOCTL_PROCMON_STOP;
        break;
    case IOCTL_PROCMON_SET_PID_DATA:
        funcToProcess = Process_IOCTL_PROCMON_SET_PID_DATA;
        break;

    case IOCTL_SPLITTING_START:        
        funcToProcess = Process_IOCTL_SPLITTING_START;
        break;
    case IOCTL_SPLITTING_STOP:
        funcToProcess = Process_IOCTL_SPLITTING_STOP;
        break;

    case IOCTL_CFG_SET_ADDRESSES:
        funcToProcess = Process_IOCTL_CFG_SET_ADDRESSES;
        break;
    case IOCTL_CFG_GET_ADDRESSES:
        funcToProcess = Process_IOCTL_CFG_GET_ADDRESSES;
        break;

    case IOCTL_CFG_SET_IMAGES_TO_SPLIT:
        funcToProcess = Process_IOCTL_CFG_SET_IMAGES_TO_SPLIT;
        break;
    case IOCTL_CFG_GET_IMAGES_TO_SPLIT:
        funcToProcess = Process_IOCTL_CFG_GET_IMAGES_TO_SPLIT;
        break;
    case IOCTL_CFG_GET_IMAGES_TO_SPLIT_BUFF_SIZE:
        funcToProcess = Process_IOCTL_CFG_GET_IMAGES_TO_SPLIT_BUFF_SIZE;
        break;

    default:
        break;
    }

    if (funcToProcess == nullptr) 
    {
        TraceEvents(TRACE_LEVEL_WARNING, TRACE_QUEUE, "%!FUNC! INVALID_PARAMETER (IoControlCode=%d [0x%X])", IoControlCode, IoControlCode);
        return STATUS_INVALID_PARAMETER;
    }

    return funcToProcess(Queue, Request);
}
