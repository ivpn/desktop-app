#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif

#include "implDns.h"
#include <vector>
#include <sstream>

#include <windows.h>
#include <winsock2.h>
#include <ws2ipdef.h>
#include <iphlpapi.h>
#include <WS2tcpip.h>

#pragma comment(lib, "Ws2_32.lib")
#define MALLOC(x) HeapAlloc(GetProcessHeap(), 0, (x))
#define FREE(x) HeapFree(GetProcessHeap(), 0, (x))

#pragma comment(lib, "iphlpapi.lib")

// The Windows versions older than WIN10 (e.g. Win8) does not have methods:
//      GetInterfaceDnsSettings, SetInterfaceDnsSettings, FreeInterfaceDnsSettings
// Removing preprocessor parameter MIN_WIN_VER_WIN10 allows us to build with mechanism of dynamic load of this functions.
// Otherwise (when MIN_WIN_VER_WIN10 defined) it will not be possible to load current library under Windows8 (with error: 'specific procedure could not be found' )
// TODO: if we will decide to not support an old windows version - just remove checks for MIN_WIN_VER_WIN10 and leave only sources related to WIN10
#ifdef MIN_WIN_VER_WIN10
NETIOAPI_API callGetInterfaceDnsSettings(_In_ GUID Interface, _Inout_ DNS_INTERFACE_SETTINGS* Settings)
{
    return GetInterfaceDnsSettings(Interface, Settings);
}
NETIOAPI_API callSetInterfaceDnsSettings(_In_ GUID Interface, _In_ const DNS_INTERFACE_SETTINGS* Settings)
{
    return SetInterfaceDnsSettings(Interface, Settings);
}
VOID NETIOAPI_API_ callFreeInterfaceDnsSettings(_Inout_ DNS_INTERFACE_SETTINGS* Settings)
{
    return FreeInterfaceDnsSettings(Settings);
}
#else

typedef DWORD (__stdcall *fnTypeGetInterfaceDnsSettings) (_In_ GUID Interface, _Inout_ DNS_INTERFACE_SETTINGS* Settings);
typedef DWORD (__stdcall *fnTypeSetInterfaceDnsSettings) (_In_ GUID Interface, _In_ const DNS_INTERFACE_SETTINGS* Settings);
typedef void (__stdcall* fnTypeFreeInterfaceDnsSettings) (_Inout_ DNS_INTERFACE_SETTINGS* Settings);

fnTypeGetInterfaceDnsSettings fnGetInterfaceDnsSettings = NULL;
fnTypeSetInterfaceDnsSettings fnSetInterfaceDnsSettings = NULL;
fnTypeFreeInterfaceDnsSettings fnFreeInterfaceDnsSettings = NULL;

HINSTANCE iphlpapiDll = 0;
bool isDllLoadFailed = false;

DWORD InitializeWin10DnsApi()
{
    if (iphlpapiDll != 0)
        return 0; // already loaded

    if (isDllLoadFailed)
        return -3; // previous an attempt to load DLL was failed; Seems there is no required dll installed. No sense to try again.

    iphlpapiDll = LoadLibrary(TEXT("Iphlpapi.dll"));
    if (iphlpapiDll == NULL)
    {
        isDllLoadFailed = true;
        return -1;
    }
    
    fnGetInterfaceDnsSettings = (fnTypeGetInterfaceDnsSettings)GetProcAddress(iphlpapiDll, "GetInterfaceDnsSettings");
    fnSetInterfaceDnsSettings = (fnTypeSetInterfaceDnsSettings)GetProcAddress(iphlpapiDll, "SetInterfaceDnsSettings");
    fnFreeInterfaceDnsSettings = (fnTypeFreeInterfaceDnsSettings)GetProcAddress(iphlpapiDll, "FreeInterfaceDnsSettings");

    if (fnGetInterfaceDnsSettings == NULL || fnSetInterfaceDnsSettings == NULL || fnFreeInterfaceDnsSettings == NULL)
    {
        fnGetInterfaceDnsSettings = NULL;
        fnSetInterfaceDnsSettings = NULL;
        fnFreeInterfaceDnsSettings = NULL;

        FreeLibrary(iphlpapiDll);
        iphlpapiDll = 0;

        isDllLoadFailed = true;
        return -2;
    }

    return 0;
}

NETIOAPI_API callGetInterfaceDnsSettings(_In_ GUID Interface, _Inout_ DNS_INTERFACE_SETTINGS* Settings)
{
    DWORD dllLoadErr = InitializeWin10DnsApi();
    if (dllLoadErr != 0) return dllLoadErr;

    return fnGetInterfaceDnsSettings(Interface, Settings);
}
NETIOAPI_API callSetInterfaceDnsSettings(_In_ GUID Interface, _In_ const DNS_INTERFACE_SETTINGS* Settings)
{
    DWORD dllLoadErr = InitializeWin10DnsApi();
    if (dllLoadErr != 0) return dllLoadErr;

    return fnSetInterfaceDnsSettings(Interface, Settings);
}
VOID NETIOAPI_API_ callFreeInterfaceDnsSettings(_Inout_ DNS_INTERFACE_SETTINGS* Settings)
{
    if (InitializeWin10DnsApi() != 0) return;

    return fnFreeInterfaceDnsSettings(Settings);
}

#endif

DWORD getInterfaceGUIDByLocalIP(std::string interfaceLocalAddr, GUID& ret);

DWORD DoSetDNSByLocalIP(std::string interfaceLocalAddr, std::string dnsIP, Operation operation)
{
    if (interfaceLocalAddr.empty())
        return -1;
	if (dnsIP.empty() && (operation == Operation::Add || operation == Operation::Remove))
		return (HRESULT)0; // nothing to add or remove

	toLowerStr(&dnsIP);

    GUID ifcGUID{ 0 };
    DWORD ret = getInterfaceGUIDByLocalIP(interfaceLocalAddr, ifcGUID);
    if (ret != 0)
        return ret;

    std::vector<std::wstring> curDnsNameServer;
    if (operation == Operation::Add || operation == Operation::Remove)
    {
        DNS_INTERFACE_SETTINGS dnsV1{ 0 };
        dnsV1.Version = DNS_INTERFACE_SETTINGS_VERSION1;
        DWORD err = callGetInterfaceDnsSettings(ifcGUID, &dnsV1);
        if (err != NO_ERROR)
            return err;
        
        if (dnsV1.NameServer != NULL)
        {
            std::wstring temp;
            std::wstringstream nameserver;
            nameserver << dnsV1.NameServer;

            while (std::getline(nameserver, temp, L','))
                curDnsNameServer.push_back(temp);
        }

        callFreeInterfaceDnsSettings(&dnsV1);
    }

    std::wstring dnsIPwstr(dnsIP.begin(), dnsIP.end());

    std::wstring newNameServer;
    if (!dnsIP.empty() && (operation == Operation::Set || operation == Operation::Add))
        newNameServer = dnsIPwstr;

    if (operation == Operation::Add || operation == Operation::Remove)
    {
        for (size_t i = 0; i < curDnsNameServer.size(); i++)
        {
            std::wstring curVal = curDnsNameServer[i];
            toLowerWStr(&curVal);
            if (curVal != dnsIPwstr) {
                if (!newNameServer.empty())
                    newNameServer += L",";
                newNameServer += curVal;
            }
        }
    }
    else return -1;

    DNS_INTERFACE_SETTINGS newDnsSettingsV1{ 0 };
    newDnsSettingsV1.Version = DNS_INTERFACE_SETTINGS_VERSION1;
    newDnsSettingsV1.Flags = DNS_SETTING_NAMESERVER;
    newDnsSettingsV1.NameServer = const_cast<PWSTR>(newNameServer.c_str());
    DWORD err = callSetInterfaceDnsSettings(ifcGUID, &newDnsSettingsV1);
    if (err != NO_ERROR)
        return err;

	return 0;
}

// Get interface GUID ny it's local IP Address OR by it's MAC address
DWORD getInterfaceGUIDByLocalIP(std::string interfaceLocalAddr, GUID& ret)
{
    if (interfaceLocalAddr.empty())
        return -1;

    toLowerStr(&interfaceLocalAddr);

    DWORD dwRetVal = 0;
    PIP_ADAPTER_ADDRESSES pAddresses = NULL;
    ULONG outBufLen = 15000;

    ULONG Iterations = 0;
    do
    {

        pAddresses = (IP_ADAPTER_ADDRESSES*)MALLOC(outBufLen);
        if (pAddresses == NULL)
            return -2;

        dwRetVal = GetAdaptersAddresses(AF_UNSPEC, GAA_FLAG_INCLUDE_PREFIX, NULL, pAddresses, &outBufLen);
        if (dwRetVal == ERROR_BUFFER_OVERFLOW)
        {
            FREE(pAddresses);
            pAddresses = NULL;
        }
        else
            break;

        Iterations++;

    } while ((dwRetVal == ERROR_BUFFER_OVERFLOW) && (Iterations < 3));

    if (dwRetVal != NO_ERROR)
    {
        FREE(pAddresses);
        return dwRetVal;
    }

    PIP_ADAPTER_ADDRESSES pCurrAddresses = NULL;
    PIP_ADAPTER_UNICAST_ADDRESS pUnicast = NULL;

    const DWORD bufflen = 100;
    char buff[bufflen];

    pCurrAddresses = pAddresses;

    while (pCurrAddresses)
    {
        pUnicast = pCurrAddresses->FirstUnicastAddress;
        if (pUnicast != NULL)
        {
            for (int i = 0; pUnicast != NULL; i++)
            {
                std::string uaddr = "";

                if (pUnicast->Address.lpSockaddr->sa_family == AF_INET)
                {
                    sockaddr_in* sa_in = (sockaddr_in*)pUnicast->Address.lpSockaddr;
                    if (inet_ntop(AF_INET, &(sa_in->sin_addr), buff, bufflen) != NULL)
                        uaddr = buff;
                }
                else if (pUnicast->Address.lpSockaddr->sa_family == AF_INET6)
                {
                    sockaddr_in6* sa_in6 = (sockaddr_in6*)pUnicast->Address.lpSockaddr;
                    if (inet_ntop(AF_INET6, &(sa_in6->sin6_addr), buff, bufflen) != NULL)
                        uaddr = buff;
                }
                else
                    continue;

                toLowerStr(&uaddr);
                if (uaddr == interfaceLocalAddr)
                {
                    if (ConvertInterfaceLuidToGuid(&pCurrAddresses->Luid, &ret) != NO_ERROR)
                        return -3;
                    return 0;
                }

                pUnicast = pUnicast->Next;
            }
        }

        pCurrAddresses = pCurrAddresses->Next;
    }

    return -4;
}