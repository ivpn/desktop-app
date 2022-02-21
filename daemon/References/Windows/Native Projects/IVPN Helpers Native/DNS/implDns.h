#pragma once

#include "../dns.h"

DWORD DoSetDNSByLocalIP    (std::string interfaceLocalAddr, std::string dnsIP, Operation operation, bool ipv6);
DWORD DoSetDNSByLocalIPEx	(std::string interfaceLocalAddr, std::string dnsIP, bool isDoH, std::string dohTemplate, Operation operation, bool ipv6);
DWORD IsDnsOverHttpsAccessible();

// The Windows versions older than WIN10 (e.g. Win8) does not have methods:
//      GetInterfaceDnsSettings, SetInterfaceDnsSettings, FreeInterfaceDnsSettings
// Removing preprocessor parameter MIN_WIN_VER_WIN10 allows us to build with mechanism of dynamic load of this functions.
// Otherwise (when MIN_WIN_VER_WIN10 defined) it will not be possible to load current library under Windows8 (with error: 'specific procedure could not be found' )
// TODO: if we will decide to not support an old windows version - just remove checks for MIN_WIN_VER_WIN10 and leave only sources related to WIN10
#ifdef MIN_WIN_VER_WIN10
#else
DWORD InitializeWin10DnsApi();
#endif