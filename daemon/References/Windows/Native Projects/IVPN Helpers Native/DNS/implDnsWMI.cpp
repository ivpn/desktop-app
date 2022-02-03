#include "implDnsWMI.h"

#include <vector>

#include <windows.h>
#include <objbase.h>
#include <atlbase.h>
#include <wbemidl.h>
#include <comutil.h>

#ifdef _DEBUG
# pragma comment(lib, "comsuppwd.lib")
#else
# pragma comment(lib, "comsuppw.lib")
#endif
# pragma comment(lib, "wbemuuid.lib")

HRESULT wmiSetDNSServerSearchOrder(const WORD interfaceIdx, std::vector<std::string> dnsSearchOrder, IWbemLocator* pLocator = NULL, IWbemServices* pNamespace = NULL);
int		wmiGetInterfaceInfo(const int interfaceIdx, std::string mac, std::string ipAddr, IWbemLocator* pLocator = NULL, IWbemServices* pNamespace = NULL, std::vector<std::string>* retDNSServerSearchOrder = NULL);

HRESULT initializeCoValues(IWbemLocator** pLocator, IWbemServices** pNamespace);
void	unInitializeCoValues(IWbemLocator** pLocator, IWbemServices** pNamespace);

class CoValuesInitializer
{
private:
	IWbemLocator** _locator;
	IWbemServices** _nmspace;
	bool __initialized;
public:
	~CoValuesInitializer()
	{
		if (__initialized)
			unInitializeCoValues(_locator, _nmspace);
	}

	HRESULT Initialize(IWbemLocator** locator, IWbemServices** nmspace) {
		_locator = locator;
		_nmspace = nmspace;
		__initialized = true;

		return initializeCoValues(_locator, _nmspace);
	}
};

// Change DNS settings for a specific local interface
// Interface can be defined: by index OR by MAC address OR by it's local IP address
HRESULT WmiSetDNS(const int destInterfaceIndex, const std::string destInterfaceMAC, const std::string destInterfaceLocalAddr, std::string dnsIP, Operation operation)
{
	// initialize 
	IWbemLocator* pLocator = NULL;
	IWbemServices* pNamespace = NULL;

	// Initialize CoValues. They will be uninitialize automatically (on exiting this funtion) 
	CoValuesInitializer initializer;
	HRESULT hr = initializer.Initialize(&pLocator, &pNamespace);
	if (hr != WBEM_S_NO_ERROR)
		return hr;

	if (dnsIP.empty() && (operation == Operation::Add || operation == Operation::Remove))
		return (HRESULT)0; // nothing to add or remove

	toLowerStr(&dnsIP);
	std::vector<std::string> curDnsSearchOrder;

	bool needCurDnsState = (operation == Operation::Add || operation == Operation::Remove);
	// if interface index not defined - find it by mac address or local IP
	int index = destInterfaceIndex;
	if (index < 0 || needCurDnsState)
		index = wmiGetInterfaceInfo(destInterfaceIndex, destInterfaceMAC, destInterfaceLocalAddr, pLocator, pNamespace, ((needCurDnsState) ? &curDnsSearchOrder : NULL));

	if (index < 0)
		return (HRESULT)-1;

	std::vector<std::string> newDnsSearchOrder;
	if (!dnsIP.empty() && operation != Operation::Remove)
		newDnsSearchOrder.push_back(dnsIP);

	if (operation == Operation::Add || operation == Operation::Remove)
	{
		for (size_t i = 0; i < curDnsSearchOrder.size(); i++)
		{
			std::string curVal = curDnsSearchOrder[i];
			toLowerStr(&curVal);
			if (curVal != dnsIP)
				newDnsSearchOrder.push_back(curVal);
		}
	}

	HRESULT ret = wmiSetDNSServerSearchOrder(index, newDnsSearchOrder, pLocator, pNamespace);

	return ret;
}

HRESULT wmiSetDNSServerSearchOrder(const WORD interfaceIdx, std::vector<std::string> dnsSearchOrder, IWbemLocator* pLocator, IWbemServices* pNamespace)
{
	HRESULT hr;

	CoValuesInitializer initializer;
	if (pLocator == NULL || pNamespace == NULL)
	{	// initialization required
		// Initialize CoValues. They will be uninitialize automatically (on exiting this funtion) 
		hr = initializer.Initialize(&pLocator, &pNamespace);
		if (hr != WBEM_S_NO_ERROR)
			return hr;
	}

	// Get class object of Win32_NetworkAdapterConfiguration
	IWbemClassObject* pClass = NULL;
	BSTR classPath = SysAllocString(L"Win32_NetworkAdapterConfiguration");
	hr = pNamespace->GetObject(classPath, 0, NULL, &pClass, NULL);
	SysFreeString(classPath);

	if (WBEM_S_NO_ERROR == hr)
	{
		// Get pointer to the input parameter class of the method we are going to call
		BSTR methodName = SysAllocString(L"SetDNSServerSearchOrder");
		IWbemClassObject* method = NULL;
		hr = pClass->GetMethod(methodName, 0, &method, NULL);

		if (WBEM_S_NO_ERROR == hr)
		{
			// Spawn instance of the input parameter class, so that we can stuff our parameters in
			IWbemClassObject* argsObj = NULL;
			hr = method->SpawnInstance(0, &argsObj);

			if (WBEM_S_NO_ERROR == hr)
			{
				// Pack desired parameters into the input class instances
				SAFEARRAY* dnsList = SafeArrayCreateVector(VT_BSTR, 0, (ULONG)dnsSearchOrder.size());
				long idx[] = { 0 }; // 0 value - is an index
				for (long i = 0; i < (long)dnsSearchOrder.size(); i++)
				{
					idx[0] = i;
					BSTR ip = _com_util::ConvertStringToBSTR(dnsSearchOrder[i].c_str());
					hr = SafeArrayPutElement(dnsList, idx, ip);
					SysFreeString(ip);
				}

				if (WBEM_S_NO_ERROR == hr)
				{
					// Wrap safe array in a VARIANT so that it can be passed to COM function

					VARIANT arg1DnsList;
					VariantInit(&arg1DnsList);
					arg1DnsList.vt = VT_ARRAY | VT_BSTR;
					if (!dnsSearchOrder.empty()) // if dnsIP is empty - 'DNSServerSearchOrder' is default (obtained by DHCP)
						arg1DnsList.parray = dnsList;

					hr = argsObj->Put(L"DNSServerSearchOrder", 0, &arg1DnsList, 0);
					if (WBEM_S_NO_ERROR == hr)
					{
						std::string filter = "Win32_NetworkAdapterConfiguration.Index='" + std::to_string(interfaceIdx) + "'";

						BSTR InstancePath = _com_util::ConvertStringToBSTR(filter.c_str());

						// call the method
						IWbemClassObject* pOutInst = NULL;
						hr = pNamespace->ExecMethod(InstancePath, methodName, 0, NULL, argsObj, &pOutInst, NULL);
						if (pOutInst != NULL)
							pOutInst->Release();

						SysFreeString(InstancePath);
					}
				}

				// Destroy safe arrays, which destroys the objects stored inside them
				SafeArrayDestroy(dnsList);

				//SysFreeString(ip);
			}

			// Free up the instances that we spawned
			if (argsObj)
				argsObj->Release();
		}

		// Free up methods input parameters class pointers
		if (method)
			method->Release();

		SysFreeString(methodName);
	}

	// Variable cleanup
	if (pClass)
		pClass->Release();

	return hr;
}

int wmiGetInterfaceInfo(const int interfaceIdx, std::string mac, std::string ipAddr, IWbemLocator* pLocator, IWbemServices* pNamespace, std::vector<std::string>* retDNSServerSearchOrder)
{
	if (mac.empty() && ipAddr.empty() && interfaceIdx < 0)
		return -1;

	int result = -1;

	toLowerStr(&mac);
	toLowerStr(&ipAddr);

	HRESULT hr;
	CoValuesInitializer initializer;
	if (pLocator == NULL || pNamespace == NULL)
	{	// initialization required
		// Initialize CoValues. They will be uninitialize automatically (on exiting this funtion) 
		hr = initializer.Initialize(&pLocator, &pNamespace);
		if (hr != WBEM_S_NO_ERROR)
			return hr;
	}

	// we're going to use CComPtr<>s, whose lifetime must end BEFORE CoUnitialize is called
	{
		// execute a query
		CComPtr< IEnumWbemClassObject > enumerator;
		hr = pNamespace->ExecQuery(_bstr_t("WQL"), _bstr_t("SELECT IPAddress, MacAddress, Index, DNSServerSearchOrder FROM Win32_NetworkAdapterConfiguration where IpEnabled = True"), WBEM_FLAG_FORWARD_ONLY, NULL, &enumerator);

		if (SUCCEEDED(hr))
		{
			for (;;)
			{
				CComPtr< IWbemClassObject > wmiItem = NULL;
				ULONG retcnt;
				hr = enumerator->Next(WBEM_INFINITE, 1L, &wmiItem, &retcnt);
				if (hr != WBEM_S_NO_ERROR || retcnt <= 0)
					break;

				// read interface index
				_variant_t idxVal;
				hr = wmiItem->Get(L"Index", 0, &idxVal, NULL, NULL);
				if (FAILED(hr))
					continue;
				if (idxVal.vt != VT_I4)
					continue;
				int index = idxVal.intVal;

				if (!mac.empty()) // if MAC is not empty - search by MAC 
				{
					_variant_t mac_val;
					hr = wmiItem->Get(L"MacAddress", 0, &mac_val, NULL, NULL);
					if (SUCCEEDED(hr))
					{
						std::string vmac = "";
						if (mac_val.vt == VT_BSTR)
						{
							vmac = std::string(_bstr_t(mac_val.bstrVal));
							toLowerStr(&vmac);
							if (vmac == mac)
								result = index; // found
						}
					}
				}
				else if (!ipAddr.empty()) // search by IP address
				{
					_variant_t addrArrayVal;
					hr = wmiItem->Get(L"IPAddress", 0, &addrArrayVal, NULL, NULL);
					if (SUCCEEDED(hr))
					{
						if (addrArrayVal.vt == (VT_ARRAY | VT_BSTR))
						{
							BSTR* arr = (BSTR*)addrArrayVal.parray->pvData;
							ULONG len = addrArrayVal.parray->rgsabound->cElements;

							for (ULONG idx = 0; idx < len; idx++)
							{
								BSTR item = arr[idx];
								std::string vaddr = std::string(_bstr_t(arr[idx]));
								toLowerStr(&vaddr);
								if (vaddr == ipAddr)
									result = index; // found
							}
						}
					}
				}
				else if (interfaceIdx >= 0)
				{
					if (index == interfaceIdx)
						result = index; // found
				}

				if (result >= 0)
				{
					if (retDNSServerSearchOrder != NULL)
					{	// we have ro return 'DNSServerSearchOrder' info
						_variant_t dnsServerSearchOrderVal;
						hr = wmiItem->Get(L"DNSServerSearchOrder", 0, &dnsServerSearchOrderVal, NULL, NULL);
						if (SUCCEEDED(hr))
						{
							if (dnsServerSearchOrderVal.vt == (VT_ARRAY | VT_BSTR))
							{
								BSTR* arr = (BSTR*)dnsServerSearchOrderVal.parray->pvData;
								ULONG len = dnsServerSearchOrderVal.parray->rgsabound->cElements;

								for (ULONG idx = 0; idx < len; idx++)
								{
									BSTR item = arr[idx];
									std::string vaddr = std::string(_bstr_t(arr[idx]));

									(*retDNSServerSearchOrder).push_back(vaddr);
								}
							}
						}
					}

					break;
				}

			}
		}
	}

	return result;
}

HRESULT initializeCoValues(IWbemLocator** locator, IWbemServices** nmspace)
{
	HRESULT hr;
	// Initialize COM and connect to WMI.
	hr = CoInitialize(0);
	if (FAILED(hr))
		return hr;

	hr = CoInitializeSecurity(NULL, -1, NULL, NULL, RPC_C_AUTHN_LEVEL_DEFAULT, RPC_C_IMP_LEVEL_IMPERSONATE, NULL, EOAC_NONE, NULL);
	if (FAILED(hr))
	{
		CoUninitialize();
		return hr;
	}

	IWbemLocator* pLocator;
	hr = CoCreateInstance(CLSID_WbemLocator, 0, CLSCTX_INPROC_SERVER, IID_IWbemLocator, (LPVOID*)&pLocator);
	if (FAILED(hr))
	{
		CoUninitialize();
		return hr;
	}

	BSTR svrPath = SysAllocString(L"ROOT\\CIMV2");
	IWbemServices* pNamespace;
	hr = pLocator->ConnectServer(svrPath, NULL, NULL, NULL, 0, NULL, NULL, &pNamespace);
	if (FAILED(hr))
	{
		pLocator->Release();
		CoUninitialize();
		return hr;
	}
	SysFreeString(svrPath);

	*locator = pLocator;
	*nmspace = pNamespace;

	return hr;
}

void unInitializeCoValues(IWbemLocator** pLocator, IWbemServices** pNamespace)
{
	if (*pNamespace)
		(*pNamespace)->Release();
	*pNamespace = NULL;
	
	if (*pLocator)
		(*pLocator)->Release();
	*pLocator = NULL;
	
	CoUninitialize();
}
