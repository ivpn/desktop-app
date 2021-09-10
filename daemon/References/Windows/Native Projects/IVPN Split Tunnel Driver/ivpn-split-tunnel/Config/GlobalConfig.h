#pragma once

#include "../public.h"
#include "../QueueIoctlProtocol/Types.h"
#include <minwindef.h>

namespace cfg
{
    NTSTATUS Init();

    NTSTATUS Clean();
    bool IsConfigOk();

    bool IsConfigIPv4AddrOk();
    bool IsConfigIPv6AddrOk();
    bool IsConfigIPv4PublicAddrOk();
    bool IsConfigIPv6PublicAddrOk();

    bool IsConfigIPv4AddrOk(const IPAddrConfig& cfgIPs);
    bool IsConfigIPv6AddrOk(const IPAddrConfig& cfgIPs);
        
    void SetIPs(const IPAddrConfig&);
    const IPAddrConfig GetIPs();

    NTSTATUS SetImagesToSplit(const char* buff, DWORD bufSize);
    NTSTATUS GetImagesToSplit(char* buff_inOut, DWORD *bufSize_inOut);
    
    bool GetIsImageToSplit(PCUNICODE_STRING imagePath);
    bool GetIsNoImagesToSplit();
}
