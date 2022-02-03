#pragma once

// This implementation works for all Windows versions
// it uses WMI (Windows Management Instrumentation) 

#include <string>

#include "../dns.h"

// Change DNS settings for a specific local interface
// Interface can be defined: by index OR by MAC address OR by it's local IP address
HRESULT WmiSetDNS(const int destInterfaceIndex, const std::string destInterfaceMAC, const std::string destInterfaceLocalAddr, std::string dnsIP, Operation operation);
