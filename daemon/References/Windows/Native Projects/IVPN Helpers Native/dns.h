#pragma once


#include <windows.h>
#include <objbase.h>
#include <atlbase.h>
#include <wbemidl.h>
#include <comutil.h>

#include <algorithm>
#include <string> 
#include <vector>

#ifdef _DEBUG
# pragma comment(lib, "comsuppwd.lib")
#else
# pragma comment(lib, "comsuppw.lib")
#endif
# pragma comment(lib, "wbemuuid.lib")

enum Operation
{
	Set = 0,
	Add = 1,
	Remove = 2
};

HRESULT wmiSetDNS(const int destInterfaceIndex, const std::string destInterfaceMAC, const std::string destInterfaceLocalAddr, std::string dnsIP, Operation operation);
HRESULT wmiSetDNSServerSearchOrder(const WORD interfaceIdx, std::vector<std::string> dnsSearchOrder, IWbemLocator* pLocator = NULL, IWbemServices* pNamespace = NULL);
int		wmiGetInterfaceInfo(const int interfaceIdx, std::string mac, std::string ipAddr, IWbemLocator* pLocator = NULL, IWbemServices* pNamespace = NULL, std::vector<std::string>* retDNSServerSearchOrder = NULL);

HRESULT initializeCoValues(IWbemLocator** pLocator, IWbemServices** pNamespace);
void	unInitializeCoValues(IWbemLocator** pLocator, IWbemServices** pNamespace);


void toLowerStr(std::string* str) {
	std::transform((*str).begin(), (*str).end(), (*str).begin(), [](unsigned char c) { return std::tolower(c); });
}
