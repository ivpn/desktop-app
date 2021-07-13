#pragma once

#include "../public.h"
#include "../QueueIoctlProtocol/Types.h"
#include <minwindef.h>

namespace cfg
{
    NTSTATUS Init();

    NTSTATUS Clean();
    bool IsConfigOk();
    
    void SetIPs(const IPAddrConfig&);
    const IPAddrConfig GetIPs();

    NTSTATUS SetImagesToSplit(const char* buff, DWORD bufSize);
    NTSTATUS GetImagesToSplit(char* buff_inOut, DWORD *bufSize_inOut);
    
    bool GetIsImageToSplit(PCUNICODE_STRING imagePath);
    bool GetIsNoImagesToSplit();
}
