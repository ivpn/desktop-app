// +build windows

package wifiNotifier

/*
#cgo LDFLAGS: -lwlanapi -lole32

#ifndef UNICODE
#define UNICODE
#endif

#include <windows.h>
#include <wlanapi.h>
#include <windot11.h>           // for DOT11_SSID struct
#include <objbase.h>
#include <wtypes.h>

//#include <wchar.h>
#include <stdio.h>
#include <stdlib.h>

static HANDLE hClient = NULL;

static inline void openHandle()
{
	if(hClient != NULL)return;

    DWORD dwMaxClient = 2;
    DWORD dwCurVersion = 0;
    DWORD dwResult = 0;

    dwResult = WlanOpenHandle(dwMaxClient, NULL, &dwCurVersion, &hClient);
    if (dwResult != ERROR_SUCCESS) {
        wprintf(L"WlanOpenHandle failed with error: %u\n", dwResult);
    }

}

static inline void wchar2char(const wchar_t* wchar , char * m_char)
{
    int len= WideCharToMultiByte( CP_ACP ,0,wchar ,wcslen( wchar ), NULL,0, NULL ,NULL );
    WideCharToMultiByte( CP_ACP ,0,wchar ,wcslen( wchar ),m_char,len, NULL ,NULL );
    m_char[len]= '\0';
}

static inline char* concatenate(char* baseString, const char* toAdd, char delimiter) {
	if (toAdd == NULL)
		return baseString;
	int addingLen = strlen(toAdd);
	if (addingLen == 0)
		return baseString;

	if (baseString == NULL) {
		baseString = (char*)malloc(addingLen +1);
		//sprintf_s(baseString, addingLen + 1, "%s%s", baseString);

		memset(baseString, 0, addingLen + 1);
		strcpy_s(baseString, addingLen + 1, toAdd);
		return baseString;
	}

	int newSize = strlen(baseString) + ((delimiter != 0) ? 1 : 0) + addingLen + 1;
	char* newString = (char*)malloc(newSize);

	if (delimiter != 0)
		sprintf_s(newString, newSize, "%s%c%s", baseString, delimiter, toAdd);
	else
		sprintf_s(newString, newSize, "%s%s", baseString, toAdd);

	free(baseString);

	return newString;
}

static inline char * getCurrentSSID(void) {

	openHandle();

	char *ssid = malloc(256);
	memset(ssid,0,256);

	DWORD dwResult = 0;
    unsigned int i;

    PWLAN_INTERFACE_INFO_LIST pIfList = NULL;
    PWLAN_INTERFACE_INFO pIfInfo = NULL;

    PWLAN_CONNECTION_ATTRIBUTES pConnectInfo = NULL;
    DWORD connectInfoSize = sizeof(WLAN_CONNECTION_ATTRIBUTES);
	WLAN_OPCODE_VALUE_TYPE opCode = wlan_opcode_value_type_invalid;

    dwResult = WlanEnumInterfaces(hClient, NULL, &pIfList);
    if (dwResult != ERROR_SUCCESS) {
        wprintf(L"WlanEnumInterfaces failed with error: %u\n", dwResult);
    } else {

        for (i = 0; i < (int) pIfList->dwNumberOfItems; i++) {
			pIfInfo = (WLAN_INTERFACE_INFO *) & pIfList->InterfaceInfo[i];

            if (pIfInfo->isState == wlan_interface_state_connected) {
                dwResult = WlanQueryInterface(hClient,
                                              &pIfInfo->InterfaceGuid,
                                              wlan_intf_opcode_current_connection,
                                              NULL,
                                              &connectInfoSize,
                                              (PVOID *) &pConnectInfo,
                                              &opCode);

                if (dwResult != ERROR_SUCCESS) {
                    wprintf(L"WlanQueryInterface failed with error: %u\n", dwResult);
                } else {
					// wprintf(L"  Profile name used:\t %ws\n", pConnectInfo->strProfileName);
					wchar2char(pConnectInfo->strProfileName,ssid);
					break;
                }
            }
        }

	}

    if (pConnectInfo != NULL) {
        WlanFreeMemory(pConnectInfo);
        pConnectInfo = NULL;
    }

    if (pIfList != NULL) {
        WlanFreeMemory(pIfList);
        pIfList = NULL;
    }

	return ssid;
}

static inline int getCurrentNetworkSecurity() {
	openHandle();

	int retSecurity = 0xFFFFFFFF;

	char *ssid = malloc(256);
	memset(ssid,0,256);

	DWORD dwResult = 0;
    unsigned int i;

    PWLAN_INTERFACE_INFO_LIST pIfList = NULL;
    PWLAN_INTERFACE_INFO pIfInfo = NULL;

    PWLAN_CONNECTION_ATTRIBUTES pConnectInfo = NULL;
    DWORD connectInfoSize = sizeof(WLAN_CONNECTION_ATTRIBUTES);
	WLAN_OPCODE_VALUE_TYPE opCode = wlan_opcode_value_type_invalid;

    dwResult = WlanEnumInterfaces(hClient, NULL, &pIfList);
    if (dwResult != ERROR_SUCCESS) {
        wprintf(L"WlanEnumInterfaces failed with error: %u\n", dwResult);
    } else {

        for (i = 0; i < (int) pIfList->dwNumberOfItems; i++) {
			pIfInfo = (WLAN_INTERFACE_INFO *) & pIfList->InterfaceInfo[i];

            if (pIfInfo->isState == wlan_interface_state_connected) {
                dwResult = WlanQueryInterface(hClient,
                                              &pIfInfo->InterfaceGuid,
                                              wlan_intf_opcode_current_connection,
                                              NULL,
                                              &connectInfoSize,
                                              (PVOID *) &pConnectInfo,
                                              &opCode);

                if (dwResult != ERROR_SUCCESS) {
                    wprintf(L"WlanQueryInterface failed with error: %u\n", dwResult);
                } else {
					// wprintf(L"  Profile name used:\t %ws\n", pConnectInfo->strProfileName);
					//wchar2char(pConnectInfo->strProfileName,ssid);
					retSecurity = pConnectInfo->wlanSecurityAttributes.dot11CipherAlgorithm;
					break;
                }
            }
        }

	}

    if (pConnectInfo != NULL) {
        WlanFreeMemory(pConnectInfo);
        pConnectInfo = NULL;
    }

    if (pIfList != NULL) {
        WlanFreeMemory(pIfList);
        pIfList = NULL;
    }

	return retSecurity;
}

static inline char* getAvailableSSIDs(void) {
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

	dwResult = WlanOpenHandle(dwMaxClient, NULL, &dwCurVersion, &hClient);
	if (dwResult != ERROR_SUCCESS) {
		wprintf(L"WlanOpenHandle failed with error: %u\n", dwResult);
		return NULL;
	}

	dwResult = WlanEnumInterfaces(hClient, NULL, &pIfList);
	if (dwResult != ERROR_SUCCESS) {
		wprintf(L"WlanEnumInterfaces failed with error: %u\n", dwResult);
		return NULL;
	}

	for (i = 0; i < (int)pIfList->dwNumberOfItems; i++) {
		pIfInfo = (WLAN_INTERFACE_INFO*)&pIfList->InterfaceInfo[i];

		dwResult = WlanGetAvailableNetworkList(hClient,
			&pIfInfo->InterfaceGuid,
			0,
			NULL,
			&pBssList);

		if (dwResult != ERROR_SUCCESS)
			wprintf(L"WlanGetAvailableNetworkList failed with error: %u\n", dwResult);
		else
		{
			dwResult = WlanScan(hClient, &pIfInfo->InterfaceGuid, 0, 0, 0);
			if (dwResult != ERROR_SUCCESS)
				wprintf(L"WlanScan failed with error: %u\n", dwResult);
			else
			{
				Sleep(3000);

				dwResult = WlanGetNetworkBssList(
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
		WlanFreeMemory(ppWlanBssList);
		ppWlanBssList = NULL;
	}

	if (pBssList != NULL) {
		WlanFreeMemory(pBssList);
		pBssList = NULL;
	}

	if (pIfList != NULL) {
		WlanFreeMemory(pIfList);
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
	openHandle();

	DWORD hResult = ERROR_SUCCESS;
	DWORD pdwPrevNotifSource = 0;
	hResult=WlanRegisterNotification(hClient,
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

	WlanCloseHandle(hClient,NULL);
	printf("WlanCloseHandle success \n");
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

// GetCurrentNetworkSecurity returns current security mode
func GetCurrentNetworkSecurity() WiFiSecurity {
	return WiFiSecurity(C.getCurrentNetworkSecurity())
}

// SetWifiNotifier initializes a handler method 'OnWifiChanged'
func SetWifiNotifier(cb func(string)) {
	internalOnWifiChangedCb = cb
	go C.setWifiNotifier()
}
