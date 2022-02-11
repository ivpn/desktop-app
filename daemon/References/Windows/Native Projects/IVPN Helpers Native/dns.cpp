#include "dns.h"

#include "DNS/implDns.h"
#ifndef MIN_WIN_VER_WIN10
#include "DNS/implDnsWMI.h"
#endif

#include "versionhelpers.h"

#define EXPORT __declspec(dllexport)

extern "C" 
{
	EXPORT DWORD _cdecl IsCanUseDnsOverHttps()
	{
		return IsDnsOverHttpsAccessible();
	}

	EXPORT DWORD _cdecl SetDNSByLocalIP(const char* interfaceLocalAddr, const char* dnsIP, byte operation, byte isDoH, const char* dohTemplateUrl, byte isIpv6)
	{
// The Windows versions older than WIN10 (e.g. Win8) does not have methods:
//      GetInterfaceDnsSettings, SetInterfaceDnsSettings, FreeInterfaceDnsSettings
// Removing preprocessor parameter MIN_WIN_VER_WIN10 allows us to build with mechanism of dynamic load of this functions.
// Otherwise (when MIN_WIN_VER_WIN10 defined) it will not be possible to load current library under Windows8 (with error: 'specific procedure could not be found' )
// TODO: if we will decide to not support an old windows version - just remove checks for MIN_WIN_VER_WIN10 and leave only sources related to WIN10
#ifdef MIN_WIN_VER_WIN10
		return DoSetDNSByLocalIPEx(interfaceLocalAddr, dnsIP, isDoH, dohTemplateUrl, (Operation)operation, isIpv6);		
#else
		if (InitializeWin10DnsApi() != 0)
		{
			// Modern DNS API initialisation failed.
			// Falling back to the old implementation
			return WmiSetDNS(
				-1,
				"",
				(interfaceLocalAddr == NULL) ? "" : interfaceLocalAddr,
				(dnsIP == NULL) ? "" : dnsIP,
				(Operation)operation);
		}
		
		return DoSetDNSByLocalIPEx(interfaceLocalAddr, dnsIP, isDoH, dohTemplateUrl, (Operation)operation, isIpv6);
#endif
		
	}
}