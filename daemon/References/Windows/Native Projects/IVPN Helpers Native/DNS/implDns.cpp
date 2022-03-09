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
#include <map>
#include <VersionHelpers.h>

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

// Get interface GUID by it's local IP Address
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
                    {
                        FREE(pAddresses);
                        return -3;
                    }
                    FREE(pAddresses);
                    return 0;
                }

                pUnicast = pUnicast->Next;
            }
        }

        pCurrAddresses = pCurrAddresses->Next;
    }
    FREE(pAddresses);
    return -4;
}

DWORD DoSetDNSByLocalIP(std::string interfaceLocalAddr, std::string dnsIP, Operation operation, bool ipv6)
{
    if (interfaceLocalAddr.empty())
        return -1;
    if (dnsIP.empty() && (operation == Operation::Add || operation == Operation::Remove))
        return (HRESULT)0; // nothing to add or remove

    toLowerStr(&dnsIP);
    std::wstring dnsIPwstr(dnsIP.begin(), dnsIP.end());

    // Get GUID of the interface with localIP==interfaceLocalAddr
    GUID ifcGUID{ 0 };
    DWORD ret = getInterfaceGUIDByLocalIP(interfaceLocalAddr, ifcGUID);
    if (ret != 0)
        return ret;

    // ------------------------------------------
    // Configure new values: NameServer
    // ------------------------------------------
    std::wstring newNameServer;

    if (operation == Operation::Add || operation == Operation::Remove)
    {
        // We have to keep the rest user-defined settings. Therefore doing changes with currect DNS settings     

        // Get current DNS settings
        DNS_INTERFACE_SETTINGS currDnsCfg{ 0 };
        currDnsCfg.Version = DNS_INTERFACE_SETTINGS_VERSION1;
        DWORD err = callGetInterfaceDnsSettings(ifcGUID, &currDnsCfg);
        if (err != NO_ERROR)
            return err;

        // Get and parse current NameServer        
        std::vector<std::wstring> curDnsNameServer;
        if (currDnsCfg.NameServer != NULL)
        {
            std::wstring temp;
            std::wstringstream nameserver;
            nameserver << currDnsCfg.NameServer;
            // loop over all coma-separated elements in current settings of NameServer
            while (std::getline(nameserver, temp, L','))
                curDnsNameServer.push_back(temp);
        }

        // make new NameServer: all configured DNS servers except the current one
        for (ULONG i = 0; i < curDnsNameServer.size(); i++)
        {
            std::wstring curVal = curDnsNameServer[i];
            toLowerWStr(&curVal);
            if (curVal == dnsIPwstr)
                continue;

            if (!newNameServer.empty())
                newNameServer += L",";
            newNameServer += curVal;
        }

        callFreeInterfaceDnsSettings(&currDnsCfg);
    }

    if (operation == Operation::Set || operation == Operation::Add)
    {
        // set new DNS on a first position
        if (newNameServer.empty())
            newNameServer = dnsIPwstr;
        else
            newNameServer = dnsIPwstr + L"," + newNameServer;
    }

    // ------------------------------------------
    // Creating DNS_INTERFACE_SETTINGS structure
    // ------------------------------------------
    DNS_INTERFACE_SETTINGS newDnsSettings{ 0 };
    newDnsSettings.NameServer = const_cast<PWSTR>(newNameServer.c_str());

    newDnsSettings.Version = DNS_INTERFACE_SETTINGS_VERSION1;
    newDnsSettings.Flags = DNS_SETTING_NAMESERVER;
    if (ipv6)
        newDnsSettings.Flags |= DNS_SETTING_IPV6;

    // ------------------------------------------
    // Set new DNS configuration
    // ------------------------------------------
    DWORD applyDnsErr = callSetInterfaceDnsSettings(ifcGUID, &newDnsSettings);
    if (applyDnsErr != NO_ERROR)
        return applyDnsErr;

    return 0;
}

DWORD DoSetDNSByLocalIPEx(std::string interfaceLocalAddr, std::string dnsIP, bool isDoH, std::string dohTemplate, Operation operation, bool ipv6)
{
    if (isDoH && !IsDnsOverHttpsAccessible())
        return ERROR_INVALID_PARAMETER;

    if (interfaceLocalAddr.empty())
        return -1;
    if (dnsIP.empty() && (operation == Operation::Add || operation == Operation::Remove))
        return (HRESULT)0; // nothing to add or remove

    toLowerStr(&dnsIP);
    std::wstring dnsIPwstr(dnsIP.begin(), dnsIP.end());
    std::wstring dohTemplateWstr = std::wstring(dohTemplate.begin(), dohTemplate.end());

    // Get GUID of the interface with localIP==interfaceLocalAddr
    GUID ifcGUID{ 0 };
    DWORD ret = getInterfaceGUIDByLocalIP(interfaceLocalAddr, ifcGUID);
    if (ret != 0)
        return ret;

    // ------------------------------------------
    // Configure new values: NameServer/cServerProperties/ServerProperties
    // ------------------------------------------

    std::wstring newNameServer;
    ULONG newCServerProperties = 0;
    DNS_SERVER_PROPERTY* newServerProperties = NULL;
    ULONG svrPropertiesCntToDestroy = 0;

    if (isDoH && operation == Operation::Set)
    {
        // First element in newServerProperties is reserved for current (new) DNS settings
        svrPropertiesCntToDestroy = newCServerProperties = 1;
        newServerProperties = new DNS_SERVER_PROPERTY[1]{ 0 };
    }
    else if (operation == Operation::Add || operation == Operation::Remove)
    {
        // We have to keep the rest user-defined settings. Therefore doing changes with currect DNS settings     

        // Get current DNS settings
        DNS_INTERFACE_SETTINGS3 currDnsCfg{ 0 };
        if (IsDnsOverHttpsAccessible())
            currDnsCfg.Version = DNS_INTERFACE_SETTINGS_VERSION3;
        else
            currDnsCfg.Version = DNS_INTERFACE_SETTINGS_VERSION1;

        DWORD err = callGetInterfaceDnsSettings(ifcGUID, (DNS_INTERFACE_SETTINGS*)&currDnsCfg);
        if (err != NO_ERROR)
            return err;

        // Get and parse current NameServer        
        std::vector<std::wstring> curDnsNameServer;
        if (currDnsCfg.NameServer != NULL)
        {
            std::wstring temp;
            std::wstringstream nameserver;
            nameserver << currDnsCfg.NameServer;
            // loop over all coma-separated elements in current settings of NameServer
            while (std::getline(nameserver, temp, L','))
                curDnsNameServer.push_back(temp);
        }

        // Due to we can remove dnsIPwstr from the the current configuration, indexes in DoH config can be changed.
        // Therefore, we are saving new indexes.
        std::map <ULONG/*old index*/, ULONG/*new index*/> newDohIndexes;

        // make new NameServer: all configured DNS servers except the current one
        for (ULONG i = 0; i < curDnsNameServer.size(); i++)
        {
            std::wstring curVal = curDnsNameServer[i];
            toLowerWStr(&curVal);
            if (curVal == dnsIPwstr)
                continue;

            if (!newNameServer.empty())
                newNameServer += L",";
            newNameServer += curVal;
            newDohIndexes[i] = (ULONG)newDohIndexes.size() + ((operation == Operation::Add) ? 1 : 0);
        }

        // Get (copy) current ServerProperties (we have to keep the rest user-defined settings)        
        if (isDoH && operation == Operation::Add && (currDnsCfg.cServerProperties <= 0 || currDnsCfg.ServerProperties == NULL))
        {
            // First element in newServerProperties is reserved for current (new) DNS settings       
            svrPropertiesCntToDestroy = newCServerProperties = 1;
            newServerProperties = new DNS_SERVER_PROPERTY[1]{ 0 };
        }
        else
        {
            // in case of 'Add' - first element in newServerProperties is reserved for current (new) DNS settings            
            newCServerProperties = (isDoH && operation == Operation::Add) ? 1 : 0;
            svrPropertiesCntToDestroy = newCServerProperties + currDnsCfg.cServerProperties;
            newServerProperties = new DNS_SERVER_PROPERTY[svrPropertiesCntToDestroy]{ 0 };

            for (ULONG i = 0; i < currDnsCfg.cServerProperties; i++)
            {
                if (newDohIndexes.find(currDnsCfg.ServerProperties[i].ServerIndex) == newDohIndexes.end())
                    continue;

                newServerProperties[newCServerProperties] = currDnsCfg.ServerProperties[i];
                newServerProperties[newCServerProperties].ServerIndex = newDohIndexes[currDnsCfg.ServerProperties[i].ServerIndex];

                if (currDnsCfg.ServerProperties[i].Property.DohSettings != NULL)
                {
                    size_t templateLen = wcslen(currDnsCfg.ServerProperties[i].Property.DohSettings->Template);

                    newServerProperties[newCServerProperties].Property.DohSettings = new DNS_DOH_SERVER_SETTINGS{ 0 };
                    newServerProperties[newCServerProperties].Property.DohSettings->Flags = currDnsCfg.ServerProperties[i].Property.DohSettings->Flags;
                    newServerProperties[newCServerProperties].Property.DohSettings->Template = new WCHAR[templateLen + 1]{ 0 };
                    wcscpy_s(
                        newServerProperties[newCServerProperties].Property.DohSettings->Template,
                        templateLen + 1,
                        currDnsCfg.ServerProperties[i].Property.DohSettings->Template);
                }
                newCServerProperties++;
            }
        }

        callFreeInterfaceDnsSettings((DNS_INTERFACE_SETTINGS*)&currDnsCfg);
    }

    if (operation == Operation::Set || operation == Operation::Add)
    {
        // set new DNS on a first position
        if (newNameServer.empty())
            newNameServer = dnsIPwstr;
        else
            newNameServer = dnsIPwstr + L"," + newNameServer;

        // Set new DNS serverProperties (dnsIP+dohTemplate) as main config
        // The newServerProperties[0] already created for it
        if (isDoH && newServerProperties != NULL && newCServerProperties > 0)
        {
            // An array of DNS_SERVER_PROPERTY structures, containing cServerProperties elements. 
            // Only DNS - over - HTTPS properties are supported, with the additional restriction of at most 1 property for each server specified in the NameServer member.
            newServerProperties[0].Version = DNS_SERVER_PROPERTY_VERSION1;
            newServerProperties[0].Type = DNS_SERVER_PROPERTY_TYPE::DnsServerDohProperty;
            newServerProperties[0].ServerIndex = 0; // The ServerIndex member of the DNS_SERVER_PROPERTY must be set to the index of the corresponding DNS server from the NameServer member.

            newServerProperties[0].Property.DohSettings = new DNS_DOH_SERVER_SETTINGS{ 0 };
            if (dohTemplateWstr.empty())
            {
                // load URI template from the system DNS-over-HTTPS system list
                newServerProperties[0].Property.DohSettings->Flags = DNS_DOH_SERVER_SETTINGS_ENABLE_AUTO;
                newServerProperties[0].Property.DohSettings->Template = NULL;
            }
            else
            {
                newServerProperties[0].Property.DohSettings->Flags = DNS_DOH_SERVER_SETTINGS_ENABLE;
                size_t templateLen = dohTemplateWstr.length();
                newServerProperties[0].Property.DohSettings->Template = new WCHAR[templateLen + 1]{ 0 };
                wcscpy_s(
                    newServerProperties[0].Property.DohSettings->Template,
                    templateLen + 1,
                    const_cast<PWSTR>(dohTemplateWstr.c_str()));
            }
        }
    }

    // ------------------------------------------
    // Creating DNS_INTERFACE_SETTINGS3 structure
    // ------------------------------------------
    DNS_INTERFACE_SETTINGS3 newDnsSettingsV3{ 0 };
    newDnsSettingsV3.ServerProperties = newServerProperties;
    newDnsSettingsV3.cServerProperties = newCServerProperties;
    newDnsSettingsV3.NameServer = const_cast<PWSTR>(newNameServer.c_str());
    if (IsDnsOverHttpsAccessible())
    {
        newDnsSettingsV3.Version = DNS_INTERFACE_SETTINGS_VERSION3;
        newDnsSettingsV3.Flags = DNS_SETTING_NAMESERVER | DNS_SETTING_DOH; // [NameServer , cServerProperties , ServerProperties ]
    } 
    else 
    {
        newDnsSettingsV3.Version = DNS_INTERFACE_SETTINGS_VERSION1;
        newDnsSettingsV3.Flags = DNS_SETTING_NAMESERVER; // [NameServer]
    }
    if (ipv6)
        newDnsSettingsV3.Flags |= DNS_SETTING_IPV6;

    // ------------------------------------------
    // Set new DNS configuration
    // ------------------------------------------
    DWORD applyDnsErr = callSetInterfaceDnsSettings(ifcGUID, (const DNS_INTERFACE_SETTINGS*)&newDnsSettingsV3);

    // Erase currServerProperties
    if (svrPropertiesCntToDestroy > 0 && newServerProperties != NULL)
    {
        for (ULONG i = 0; i < svrPropertiesCntToDestroy; i++)
        {
            if (newServerProperties[i].Property.DohSettings != NULL)
            {
                if (newServerProperties[i].Property.DohSettings->Template != NULL)
                {
                    delete[] newServerProperties[i].Property.DohSettings->Template;
                    newServerProperties[i].Property.DohSettings->Template = NULL;
                }
                delete newServerProperties[i].Property.DohSettings;
                newServerProperties[i].Property.DohSettings = NULL;
            }
        }
        delete[] newServerProperties;
        newServerProperties = NULL;
    }

    if (applyDnsErr != NO_ERROR)
        return applyDnsErr;

    return 0;
}

// DoH support implemented since Windows 11. But this library can be used from Windows 10 too.
// This variable keep info if we can use DoH
bool isDnsOverHttpsAccessible = true;
bool isDnsOverHttpsAccessibleTested = false;
DWORD IsDnsOverHttpsAccessible()
{
    if (isDnsOverHttpsAccessibleTested)
        return isDnsOverHttpsAccessible;
    isDnsOverHttpsAccessibleTested = true;

    if  (!IsWindows10OrGreater()) 
    {
        isDnsOverHttpsAccessible = false;
        return isDnsOverHttpsAccessible;
    }

    // Test if DoH functionality supported by current version of the OS:
    // just trying to load DNS settings using type DNS_INTERFACE_SETTINGS_VERSION3.
    // Using empty GUID, we do not care about it. 
    // In case of success - function must return ERROR_FILE_NOT_FOUND (0x2) (because of empty GUID)
    // In case if DoH not supported - function will return ERROR_INVALID_PARAMETER (0x57)
    DNS_INTERFACE_SETTINGS3 currDnsCfg{ 0 };
    currDnsCfg.Version = DNS_INTERFACE_SETTINGS_VERSION3;
    DWORD err = callGetInterfaceDnsSettings(GUID{ 0 }, (DNS_INTERFACE_SETTINGS*)&currDnsCfg);
    if (err == ERROR_INVALID_PARAMETER)
        isDnsOverHttpsAccessible = false;
    if (err == NO_ERROR) // normally, this should not happen
        callFreeInterfaceDnsSettings((DNS_INTERFACE_SETTINGS*)&currDnsCfg);

    return isDnsOverHttpsAccessible;
}