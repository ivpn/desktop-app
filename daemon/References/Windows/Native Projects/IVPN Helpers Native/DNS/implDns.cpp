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

// Helper structure to hold DNS server information
struct DnsServerInfo
{
    std::wstring address;
    bool isDoH;
    std::wstring dohTemplate;
    
    DnsServerInfo(const std::wstring& addr, bool doh = false, const std::wstring& tmpl = L"")
        : address(addr), isDoH(doh), dohTemplate(tmpl) {}
};

// Helper function to read current DNS settings and parse server information
static std::vector<DnsServerInfo> ReadCurrentDnsServers(const GUID& ifcGUID)
{
    std::vector<DnsServerInfo> servers;
    
    DNS_INTERFACE_SETTINGS3 currDnsCfg{ 0 };
    if (IsDnsOverHttpsAccessible())
        currDnsCfg.Version = DNS_INTERFACE_SETTINGS_VERSION3;
    else
        currDnsCfg.Version = DNS_INTERFACE_SETTINGS_VERSION1;

    DWORD err = callGetInterfaceDnsSettings(ifcGUID, (DNS_INTERFACE_SETTINGS*)&currDnsCfg);
    if (err != NO_ERROR)
        return servers; // Return empty vector on error

    // Parse current NameServer
    std::vector<std::wstring> nameServers;
    if (currDnsCfg.NameServer != NULL)
    {
        std::wstring temp;
        std::wstringstream nameserver;
        nameserver << currDnsCfg.NameServer;
        while (std::getline(nameserver, temp, L','))
            nameServers.push_back(temp);
    }

    // Create server info with DoH status
    for (size_t i = 0; i < nameServers.size(); ++i)
    {
        bool isDoH = false;
        std::wstring dohTemplate;
        
        if (currDnsCfg.ServerProperties != NULL)
        {
            for (ULONG j = 0; j < currDnsCfg.cServerProperties; ++j)
            {
                if (currDnsCfg.ServerProperties[j].ServerIndex == i &&
                    currDnsCfg.ServerProperties[j].Type == DNS_SERVER_PROPERTY_TYPE::DnsServerDohProperty)
                {
                    isDoH = true;
                    if (currDnsCfg.ServerProperties[j].Property.DohSettings != NULL &&
                        currDnsCfg.ServerProperties[j].Property.DohSettings->Template != NULL)
                    {
                        dohTemplate = currDnsCfg.ServerProperties[j].Property.DohSettings->Template;
                    }
                    break;
                }
            }
        }
        
        servers.emplace_back(nameServers[i], isDoH, dohTemplate);
    }

    callFreeInterfaceDnsSettings((DNS_INTERFACE_SETTINGS*)&currDnsCfg);
    return servers;
}

// Helper function to build final server list based on operation
static std::vector<DnsServerInfo> BuildFinalServerList(
    const std::vector<DnsServerInfo>& currentServers,
    const std::wstring& dnsIPwstr,
    bool isDoH,
    const std::wstring& dohTemplateWstr,
    Operation operation)
{
    std::vector<DnsServerInfo> finalServers;
    
    if (operation == Operation::Set)
    {
        finalServers.emplace_back(dnsIPwstr, isDoH, dohTemplateWstr);
        return finalServers;
    }
    
    if (operation == Operation::Add)
    {
        // Add new DNS at first position
        finalServers.emplace_back(dnsIPwstr, isDoH, dohTemplateWstr);
    }
    
    // Add existing DNS servers (except the one being removed)
    for (const auto& server : currentServers)
    {
        std::wstring curVal = server.address;
        toLowerWStr(&curVal);
        std::wstring targetVal = dnsIPwstr;
        toLowerWStr(&targetVal);
        
        if (curVal == targetVal)
            continue; // Skip if removing or if duplicate when adding
            
        finalServers.push_back(server);
    }
    
    return finalServers;
}

// Helper function to create DoH properties from server list
static std::vector<DNS_SERVER_PROPERTY> CreateDohProperties(const std::vector<DnsServerInfo>& servers)
{
    std::vector<DNS_SERVER_PROPERTY> properties;
    
    for (size_t i = 0; i < servers.size(); ++i)
    {
        if (!servers[i].isDoH)
            continue;
            
        DNS_SERVER_PROPERTY prop = { 0 };
        prop.Version = DNS_SERVER_PROPERTY_VERSION1;
        prop.Type = DNS_SERVER_PROPERTY_TYPE::DnsServerDohProperty;
        prop.ServerIndex = (ULONG)i;
        
        prop.Property.DohSettings = new DNS_DOH_SERVER_SETTINGS{ 0 };
        if (servers[i].dohTemplate.empty())
        {
            prop.Property.DohSettings->Flags = DNS_DOH_SERVER_SETTINGS_ENABLE_AUTO;
            prop.Property.DohSettings->Template = NULL;
        }
        else
        {
            prop.Property.DohSettings->Flags = DNS_DOH_SERVER_SETTINGS_ENABLE;
            size_t templateLen = servers[i].dohTemplate.length();
            prop.Property.DohSettings->Template = new WCHAR[templateLen + 1]{ 0 };
            wcscpy_s(prop.Property.DohSettings->Template, templateLen + 1, servers[i].dohTemplate.c_str());
        }
        
        properties.push_back(prop);
    }
    
    return properties;
}

// Helper function to build NameServer string from server list
static std::wstring BuildNameServerString(const std::vector<DnsServerInfo>& servers)
{
    std::wstring result;
    for (size_t i = 0; i < servers.size(); ++i)
    {
        if (i > 0)
            result += L",";
        result += servers[i].address;
    }
    return result;
}

// Helper function to cleanup DoH properties
static void CleanupDohProperties(DNS_SERVER_PROPERTY* properties, ULONG count)
{
    if (properties == nullptr || count == 0)
        return;
        
    for (ULONG i = 0; i < count; ++i)
    {
        if (properties[i].Property.DohSettings != NULL)
        {
            if (properties[i].Property.DohSettings->Template != NULL)
            {
                delete[] properties[i].Property.DohSettings->Template;
                properties[i].Property.DohSettings->Template = NULL;
            }
            delete properties[i].Property.DohSettings;
            properties[i].Property.DohSettings = NULL;
        }
    }
    delete[] properties;
}

DWORD DoSetDNSByLocalIPEx(std::string interfaceLocalAddr, std::string dnsIP, bool isDoH, std::string dohTemplate, Operation operation, bool ipv6)
{
    // Input validation
    if (isDoH && !IsDnsOverHttpsAccessible())
        return ERROR_INVALID_PARAMETER;
    if (interfaceLocalAddr.empty())
        return -1;
    if (dnsIP.empty() && (operation == Operation::Add || operation == Operation::Remove))
        return (HRESULT)0; // nothing to add or remove

    // Prepare strings
    toLowerStr(&dnsIP);
    std::wstring dnsIPwstr(dnsIP.begin(), dnsIP.end());
    std::wstring dohTemplateWstr(dohTemplate.begin(), dohTemplate.end());

    // Get interface GUID
    GUID ifcGUID{ 0 };
    DWORD ret = getInterfaceGUIDByLocalIP(interfaceLocalAddr, ifcGUID);
    if (ret != 0)
        return ret;

    // Build final server list
    std::vector<DnsServerInfo> finalServers;
    if (operation == Operation::Set)
    {
        finalServers.emplace_back(dnsIPwstr, isDoH, dohTemplateWstr);
    }
    else // Add or Remove
    {
        std::vector<DnsServerInfo> currentServers = ReadCurrentDnsServers(ifcGUID);
        finalServers = BuildFinalServerList(currentServers, dnsIPwstr, isDoH, dohTemplateWstr, operation);
    }

    // Build NameServer string and DoH properties
    std::wstring newNameServer = BuildNameServerString(finalServers);
    std::vector<DNS_SERVER_PROPERTY> dohPropertiesVec = CreateDohProperties(finalServers);
    
    // Convert to array for Windows API
    ULONG newCServerProperties = (ULONG)dohPropertiesVec.size();
    DNS_SERVER_PROPERTY* newServerProperties = nullptr;
    if (newCServerProperties > 0)
    {
        newServerProperties = new DNS_SERVER_PROPERTY[newCServerProperties];
        for (ULONG i = 0; i < newCServerProperties; ++i)
            newServerProperties[i] = dohPropertiesVec[i];
    }

    // Create DNS settings structure
    DNS_INTERFACE_SETTINGS3 newDnsSettingsV3{ 0 };
    newDnsSettingsV3.ServerProperties = newServerProperties;
    newDnsSettingsV3.cServerProperties = newCServerProperties;
    newDnsSettingsV3.NameServer = const_cast<PWSTR>(newNameServer.c_str());
    
    if (IsDnsOverHttpsAccessible())
    {
        newDnsSettingsV3.Version = DNS_INTERFACE_SETTINGS_VERSION3;
        newDnsSettingsV3.Flags = DNS_SETTING_NAMESERVER | DNS_SETTING_DOH;
    }
    else
    {
        newDnsSettingsV3.Version = DNS_INTERFACE_SETTINGS_VERSION1;
        newDnsSettingsV3.Flags = DNS_SETTING_NAMESERVER;
    }
    
    if (ipv6)
        newDnsSettingsV3.Flags |= DNS_SETTING_IPV6;

    // Apply DNS configuration
    DWORD applyDnsErr = callSetInterfaceDnsSettings(ifcGUID, (const DNS_INTERFACE_SETTINGS*)&newDnsSettingsV3);

    // Cleanup
    CleanupDohProperties(newServerProperties, newCServerProperties);

    return applyDnsErr;
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