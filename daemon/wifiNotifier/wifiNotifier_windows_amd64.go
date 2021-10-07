// +build windows

package wifiNotifier

/*
#ifndef UNICODE
#define UNICODE
#endif

#include <windows.h>
#include <wlanapi.h>
#include <windot11.h>           // for DOT11_SSID struct
#include <objbase.h>
#include <wtypes.h>

#include <stdio.h>
#include <stdlib.h>

static HANDLE hClient = NULL;

typedef DWORD (WINAPI *_ftype_WlanOpenHandle)(
    _In_ DWORD dwClientVersion,
    _Reserved_ PVOID pReserved,
    _Out_ PDWORD pdwNegotiatedVersion,
    _Out_ PHANDLE phClientHandle
);

typedef DWORD (WINAPI *_ftype_WlanEnumInterfaces)(
    _In_ HANDLE hClientHandle,
    _Reserved_ PVOID pReserved,
    _Outptr_ PWLAN_INTERFACE_INFO_LIST* ppInterfaceList
);

typedef DWORD (WINAPI* _ftype_WlanQueryInterface)(
    _In_ HANDLE hClientHandle,
    _In_ CONST GUID* pInterfaceGuid,
    _In_ WLAN_INTF_OPCODE OpCode,
    _Reserved_ PVOID pReserved,
    _Out_ PDWORD pdwDataSize,
    _Outptr_result_bytebuffer_(*pdwDataSize) PVOID* ppData,
    _Out_opt_ PWLAN_OPCODE_VALUE_TYPE pWlanOpcodeValueType
);

typedef DWORD (WINAPI* _ftype_WlanGetAvailableNetworkList)(
    _In_ HANDLE hClientHandle,
    _In_ CONST GUID* pInterfaceGuid,
    _In_ DWORD dwFlags,
    _Reserved_ PVOID pReserved,
    _Outptr_ PWLAN_AVAILABLE_NETWORK_LIST* ppAvailableNetworkList
);

typedef DWORD (WINAPI* _ftype_WlanRegisterNotification)(
    _In_ HANDLE hClientHandle,
    _In_ DWORD dwNotifSource,
    _In_ BOOL bIgnoreDuplicate,
    _In_opt_ WLAN_NOTIFICATION_CALLBACK funcCallback,
    _In_opt_ PVOID pCallbackContext,
    _Reserved_ PVOID pReserved,
    _Out_opt_ PDWORD pdwPrevNotifSource
);

typedef DWORD (WINAPI* _ftype_WlanScan)(
    _In_ HANDLE hClientHandle,
    _In_ CONST GUID* pInterfaceGuid,
    _In_opt_ CONST PDOT11_SSID pDot11Ssid,
    _In_opt_ CONST PWLAN_RAW_DATA pIeData,
    _Reserved_ PVOID pReserved
);

typedef DWORD (WINAPI* _ftype_WlanGetNetworkBssList)(
    _In_ HANDLE hClientHandle,
    _In_ CONST GUID* pInterfaceGuid,
    _In_opt_ CONST PDOT11_SSID pDot11Ssid,
    _In_ DOT11_BSS_TYPE dot11BssType,
    _In_ BOOL bSecurityEnabled,
    _Reserved_ PVOID pReserved,
    _Outptr_ PWLAN_BSS_LIST* ppWlanBssList
);

typedef VOID(WINAPI* _ftype_WlanFreeMemory)(
    _In_ PVOID pMemory
);

static _ftype_WlanOpenHandle               _f_WlanOpenHandle               = NULL;
static _ftype_WlanEnumInterfaces           _f_WlanEnumInterfaces           = NULL;
static _ftype_WlanQueryInterface           _f_WlanQueryInterface           = NULL;
static _ftype_WlanGetAvailableNetworkList  _f_WlanGetAvailableNetworkList  = NULL;
static _ftype_WlanRegisterNotification     _f_WlanRegisterNotification     = NULL;
static _ftype_WlanScan                     _f_WlanScan                     = NULL;
static _ftype_WlanGetNetworkBssList        _f_WlanGetNetworkBssList        = NULL;
static _ftype_WlanFreeMemory               _f_WlanFreeMemory               = NULL;

#define false 0
#define true 1

static int isInitialized = false;
static int isInitializationError = false;

static inline int initWlanapiDll()
{
    if (isInitialized) return 0;
    if (isInitializationError) return 1;

    isInitializationError = true;

    _ftype_WlanOpenHandle               _wlanOpenHandle             = NULL;
    _ftype_WlanEnumInterfaces           _wlanEnumInterfaces         = NULL;
    _ftype_WlanQueryInterface           _wlanQueryInterface         = NULL;
    _ftype_WlanGetAvailableNetworkList  _wlanGetAvailableNetworkList= NULL;
    _ftype_WlanRegisterNotification     _wlanRegisterNotification   = NULL;
    _ftype_WlanScan                     _wlanScan                   = NULL;
    _ftype_WlanGetNetworkBssList        _wlanGetNetworkBssList      = NULL;
    _ftype_WlanFreeMemory               _wlanFreeMemory             = NULL;

	HINSTANCE hGetProcIDDLL = LoadLibrary(L"wlanapi.dll");

	if (!hGetProcIDDLL)
	{
		wprintf(L"ERROR: could not load the dynamic library 'wlanapi.dll'\n");
		return 1;
	}

	_wlanOpenHandle             = (_ftype_WlanOpenHandle)GetProcAddress(hGetProcIDDLL, "WlanOpenHandle");
	_wlanEnumInterfaces         = (_ftype_WlanEnumInterfaces)GetProcAddress(hGetProcIDDLL, "WlanEnumInterfaces");
	_wlanQueryInterface         = (_ftype_WlanQueryInterface)GetProcAddress(hGetProcIDDLL, "WlanQueryInterface");
	_wlanGetAvailableNetworkList= (_ftype_WlanGetAvailableNetworkList)GetProcAddress(hGetProcIDDLL, "WlanGetAvailableNetworkList");
	_wlanRegisterNotification   = (_ftype_WlanRegisterNotification)GetProcAddress(hGetProcIDDLL, "WlanRegisterNotification");
	_wlanScan                   = (_ftype_WlanScan)GetProcAddress(hGetProcIDDLL, "WlanScan");
	_wlanGetNetworkBssList      = (_ftype_WlanGetNetworkBssList)GetProcAddress(hGetProcIDDLL, "WlanGetNetworkBssList");
	_wlanFreeMemory             = (_ftype_WlanFreeMemory)GetProcAddress(hGetProcIDDLL, "WlanFreeMemory");

	if (!_wlanOpenHandle
		|| !_wlanEnumInterfaces
		|| !_wlanQueryInterface
		|| !_wlanGetAvailableNetworkList
		|| !_wlanRegisterNotification
		|| !_wlanScan
		|| !_wlanGetNetworkBssList
		|| !_wlanFreeMemory)
        {
            wprintf(L"ERROR: could not locate the requirted functions in dynamic library 'wlanapi.dll'\n");
            FreeLibrary(hGetProcIDDLL);
            return 2;
        }

    _f_WlanOpenHandle               = _wlanOpenHandle;
    _f_WlanEnumInterfaces           = _wlanEnumInterfaces;
    _f_WlanQueryInterface           = _wlanQueryInterface;
    _f_WlanGetAvailableNetworkList  = _wlanGetAvailableNetworkList;
    _f_WlanRegisterNotification     = _wlanRegisterNotification;
    _f_WlanScan                     = _wlanScan;
    _f_WlanGetNetworkBssList        = _wlanGetNetworkBssList;
    _f_WlanFreeMemory               = _wlanFreeMemory;

    isInitialized = true;
    isInitializationError = false;

    return 0;
}

static inline void openHandle()
{
    if (initWlanapiDll()) return;
    if (hClient != NULL) return;

    DWORD dwMaxClient = 2;
    DWORD dwCurVersion = 0;
    DWORD dwResult = 0;

    dwResult = _f_WlanOpenHandle(dwMaxClient, NULL, &dwCurVersion, &hClient);
    if (dwResult != ERROR_SUCCESS) {
        wprintf(L"WlanOpenHandle failed with error: %u\n", dwResult);
    }
}

static inline void wchar2char(const wchar_t* wchar, char* m_char)
{
    int len = WideCharToMultiByte(CP_ACP, 0, wchar, wcslen(wchar), NULL, 0, NULL, NULL);
    WideCharToMultiByte(CP_ACP, 0, wchar, wcslen(wchar), m_char, len, NULL, NULL);
    m_char[len] = '\0';
}

static inline char* concatenate(char* baseString, const char* toAdd, char delimiter) {
	if (toAdd == NULL)
		return baseString;
	size_t addingLen = strlen(toAdd);
	if (addingLen == 0)
		return baseString;

	if (baseString == NULL) {
		baseString = (char*)malloc(addingLen + 1);
		//sprintf_s(baseString, addingLen + 1, "%s%s", baseString);

		memset(baseString, 0, addingLen + 1);
		strcpy_s(baseString, addingLen + 1, toAdd);
		return baseString;
	}

	size_t newSize = strlen(baseString) + ((delimiter != 0) ? 1 : 0) + addingLen + 1;
	char* newString = (char*)malloc(newSize);

	if (delimiter != 0)
		sprintf_s(newString, newSize, "%s%c%s", baseString, delimiter, toAdd);
	else
		sprintf_s(newString, newSize, "%s%s", baseString, toAdd);

	free(baseString);

	return newString;
}

static inline char* getCurrentSSID(void) {
    if (initWlanapiDll()) return NULL;

    openHandle();

    char* ssid = (char*) malloc(256);
    memset(ssid, 0, 256);

    DWORD dwResult = 0;
    unsigned int i;

    PWLAN_INTERFACE_INFO_LIST pIfList = NULL;
    PWLAN_INTERFACE_INFO pIfInfo = NULL;

    PWLAN_CONNECTION_ATTRIBUTES pConnectInfo = NULL;
    DWORD connectInfoSize = sizeof(WLAN_CONNECTION_ATTRIBUTES);
    WLAN_OPCODE_VALUE_TYPE opCode = wlan_opcode_value_type_invalid;

    dwResult = _f_WlanEnumInterfaces(hClient, NULL, &pIfList);
    if (dwResult != ERROR_SUCCESS) {
        wprintf(L"WlanEnumInterfaces failed with error: %u\n", dwResult);
    }
    else {

        for (i = 0; i < (int)pIfList->dwNumberOfItems; i++) {
            pIfInfo = (WLAN_INTERFACE_INFO*)&pIfList->InterfaceInfo[i];

            if (pIfInfo->isState == wlan_interface_state_connected) {
                dwResult = _f_WlanQueryInterface(hClient,
                    &pIfInfo->InterfaceGuid,
                    wlan_intf_opcode_current_connection,
                    NULL,
                    &connectInfoSize,
                    (PVOID*)&pConnectInfo,
                    &opCode);

                if (dwResult != ERROR_SUCCESS) {
                    wprintf(L"WlanQueryInterface failed with error: %u\n", dwResult);
                }
                else {
                    // wprintf(L"  Profile name used:\t %ws\n", pConnectInfo->strProfileName);
                    wchar2char(pConnectInfo->strProfileName, ssid);
                    break;
                }
            }
        }

    }

    if (pConnectInfo != NULL) {
        _f_WlanFreeMemory(pConnectInfo);
        pConnectInfo = NULL;
    }

    if (pIfList != NULL) {
        _f_WlanFreeMemory(pIfList);
        pIfList = NULL;
    }

    return ssid;
}

static inline int getCurrentNetworkSecurity() {
    int retSecurity = 0xFFFFFFFF;
    if (initWlanapiDll()) return retSecurity;

    openHandle();

    char* ssid = (char*) malloc(256);
    memset(ssid, 0, 256);

    DWORD dwResult = 0;
    unsigned int i;

    PWLAN_INTERFACE_INFO_LIST pIfList = NULL;
    PWLAN_INTERFACE_INFO pIfInfo = NULL;

    PWLAN_CONNECTION_ATTRIBUTES pConnectInfo = NULL;
    DWORD connectInfoSize = sizeof(WLAN_CONNECTION_ATTRIBUTES);
    WLAN_OPCODE_VALUE_TYPE opCode = wlan_opcode_value_type_invalid;

    dwResult = _f_WlanEnumInterfaces(hClient, NULL, &pIfList);
    if (dwResult != ERROR_SUCCESS) {
        wprintf(L"WlanEnumInterfaces failed with error: %u\n", dwResult);
    }
    else {

        for (i = 0; i < (int)pIfList->dwNumberOfItems; i++) {
            pIfInfo = (WLAN_INTERFACE_INFO*)&pIfList->InterfaceInfo[i];

            if (pIfInfo->isState == wlan_interface_state_connected) {
                dwResult = _f_WlanQueryInterface(hClient,
                    &pIfInfo->InterfaceGuid,
                    wlan_intf_opcode_current_connection,
                    NULL,
                    &connectInfoSize,
                    (PVOID*)&pConnectInfo,
                    &opCode);

                if (dwResult != ERROR_SUCCESS) {
                    wprintf(L"WlanQueryInterface failed with error: %u\n", dwResult);
                }
                else {
                    // wprintf(L"  Profile name used:\t %ws\n", pConnectInfo->strProfileName);
                    //wchar2char(pConnectInfo->strProfileName,ssid);
                    retSecurity = pConnectInfo->wlanSecurityAttributes.dot11CipherAlgorithm;
                    break;
                }
            }
        }

    }

    if (pConnectInfo != NULL) {
        _f_WlanFreeMemory(pConnectInfo);
        pConnectInfo = NULL;
    }

    if (pIfList != NULL) {
        _f_WlanFreeMemory(pIfList);
        pIfList = NULL;
    }

    return retSecurity;
}

static inline char* getAvailableSSIDs(void) {
    if (initWlanapiDll()) return NULL;

    HANDLE hClient = NULL;
    DWORD dwMaxClient = 2;
    DWORD dwCurVersion = 0;
    DWORD dwResult = 0;
    unsigned int i;

    char* retList = NULL;

    PWLAN_INTERFACE_INFO_LIST pIfList = NULL;
    PWLAN_INTERFACE_INFO pIfInfo = NULL;
    PWLAN_AVAILABLE_NETWORK_LIST pBssList = NULL;
    PWLAN_BSS_LIST ppWlanBssList = NULL;

    dwResult = _f_WlanOpenHandle(dwMaxClient, NULL, &dwCurVersion, &hClient);
    if (dwResult != ERROR_SUCCESS) {
        wprintf(L"WlanOpenHandle failed with error: %u\n", dwResult);
        return NULL;
    }

    dwResult = _f_WlanEnumInterfaces(hClient, NULL, &pIfList);
    if (dwResult != ERROR_SUCCESS) {
        wprintf(L"WlanEnumInterfaces failed with error: %u\n", dwResult);
        return NULL;
    }

    for (i = 0; i < (int)pIfList->dwNumberOfItems; i++) {
        pIfInfo = (WLAN_INTERFACE_INFO*)&pIfList->InterfaceInfo[i];

        dwResult = _f_WlanGetAvailableNetworkList(hClient,
            &pIfInfo->InterfaceGuid,
            0,
            NULL,
            &pBssList);

        if (dwResult != ERROR_SUCCESS)
            wprintf(L"WlanGetAvailableNetworkList failed with error: %u\n", dwResult);
        else
        {
            dwResult = _f_WlanScan(hClient, &pIfInfo->InterfaceGuid, 0, 0, 0);
            if (dwResult != ERROR_SUCCESS)
                wprintf(L"WlanScan failed with error: %u\n", dwResult);
            else
            {
                Sleep(3000);

                dwResult = _f_WlanGetNetworkBssList(
                    hClient,
                    &pIfInfo->InterfaceGuid,
                    NULL,
                    dot11_BSS_type_any,
                    0,
                    NULL,
                    &ppWlanBssList);

                if (ERROR_SUCCESS != dwResult)
                    wprintf(L"WlanGetNetworkBssList failed with error: %u\n", dwResult);
                else
                {
                    for (DWORD i = 0; i < ppWlanBssList->dwNumberOfItems; i++)
                        retList = concatenate(retList, (char*)ppWlanBssList->wlanBssEntries[i].dot11Ssid.ucSSID, '\n');
                }
            }
        }
    }

    if (ppWlanBssList != NULL) {
        _f_WlanFreeMemory(ppWlanBssList);
        ppWlanBssList = NULL;
    }

    if (pBssList != NULL) {
        _f_WlanFreeMemory(pBssList);
        pBssList = NULL;
    }

    if (pIfList != NULL) {
        _f_WlanFreeMemory(pIfList);
        pIfList = NULL;
    }

    return retList;
}

static inline void onWifiChanged(PWLAN_NOTIFICATION_DATA data,PVOID context)
{
	extern void __onWifiChanged(char *);
	__onWifiChanged(getCurrentSSID());
}

static inline void setWifiNotifier()
{
	if (initWlanapiDll()) return;
	openHandle();

	DWORD hResult = ERROR_SUCCESS;
	DWORD pdwPrevNotifSource = 0;
	hResult=_f_WlanRegisterNotification(hClient,
									WLAN_NOTIFICATION_SOURCE_ACM,
									TRUE,
									(WLAN_NOTIFICATION_CALLBACK)onWifiChanged,
									NULL,
									NULL,
									&pdwPrevNotifSource);
	if(hResult!=ERROR_SUCCESS)
		printf("failed WlanRegisterNotification=%d \n",hResult);

	while(TRUE){
		Sleep(10);
	}
}
*/
import "C"
import (
	"strings"
	"unsafe"
)

var internalOnWifiChangedCb func(string)

//export __onWifiChanged
func __onWifiChanged(ssid *C.char) {
	goSsid := C.GoString(ssid)
	C.free(unsafe.Pointer(ssid))

	if internalOnWifiChangedCb != nil {
		internalOnWifiChangedCb(goSsid)
	}
}

// GetAvailableSSIDs returns the list of the names of available Wi-Fi networks
func GetAvailableSSIDs() []string {
	ssidList := C.getAvailableSSIDs()
	goSsidList := C.GoString(ssidList)
	C.free(unsafe.Pointer(ssidList))
	return strings.Split(goSsidList, "\n")
}

// GetCurrentSSID returns current WiFi SSID
func GetCurrentSSID() string {
	ssid := C.getCurrentSSID()
	goSsid := C.GoString(ssid)
	C.free(unsafe.Pointer(ssid))
	return goSsid
}

// GetCurrentNetworkIsInsecure returns current security mode
func GetCurrentNetworkIsInsecure() bool {
	const (
		DOT11_CIPHER_ALGO_NONE          = 0x00
		DOT11_CIPHER_ALGO_WEP40         = 0x01
		DOT11_CIPHER_ALGO_TKIP          = 0x02
		DOT11_CIPHER_ALGO_CCMP          = 0x04
		DOT11_CIPHER_ALGO_WEP104        = 0x05
		DOT11_CIPHER_ALGO_WPA_USE_GROUP = 0x100
		DOT11_CIPHER_ALGO_RSN_USE_GROUP = 0x100
		DOT11_CIPHER_ALGO_WEP           = 0x101
		DOT11_CIPHER_ALGO_IHV_START     = 0x80000000
		DOT11_CIPHER_ALGO_IHV_END       = 0xffffffff
	)

	security := C.getCurrentNetworkSecurity()
	switch security {
	case DOT11_CIPHER_ALGO_NONE,
		DOT11_CIPHER_ALGO_WEP40,
		DOT11_CIPHER_ALGO_WEP104,
		DOT11_CIPHER_ALGO_WEP:
		return true
	}
	return false
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func SetWifiNotifier(cb func(string)) error {
	internalOnWifiChangedCb = cb
	go C.setWifiNotifier()
	return nil
}
