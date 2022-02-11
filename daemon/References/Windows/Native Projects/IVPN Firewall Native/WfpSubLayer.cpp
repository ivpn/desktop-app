#include "stdafx.h"

extern "C" {

	EXPORT bool _cdecl WfpSubLayerIsInstalled(HANDLE engineHandle, GUID subLayerKey)
	{
		FWPM_SUBLAYER0 *subLayer;
		DWORD result = FwpmSubLayerGetByKey0(engineHandle, &subLayerKey, &subLayer);
		FwpmFreeMemory0((void **)&subLayer);

		if (result == 0)
			return true;

		return false;
	}
	EXPORT bool _cdecl WfpSubLayerIsInstalledPtr(HANDLE engineHandle, GUID* subLayerKey) { return WfpSubLayerIsInstalled(engineHandle, *subLayerKey);	}

	EXPORT DWORD _cdecl WfpSubLayerDelete(HANDLE engineHandle, GUID subLayerKey)
	{
		return FwpmSubLayerDeleteByKey0(engineHandle, &subLayerKey);
	}
	EXPORT DWORD _cdecl WfpSubLayerDeletePtr(HANDLE engineHandle, GUID* subLayerKey) { return WfpSubLayerDelete(engineHandle, *subLayerKey); }

	EXPORT DWORD _cdecl WfpSubLayerAdd(HANDLE engineHandle, FWPM_SUBLAYER0 *subLayerStruct)
	{
		return FwpmSubLayerAdd0(engineHandle, subLayerStruct, NULL);
	}

	EXPORT FWPM_SUBLAYER0 * _cdecl FWPM_SUBLAYER0_Create(GUID subLayerKey, UINT32 weight)
	{
		FWPM_SUBLAYER0* subLayer = new FWPM_SUBLAYER0{0};
		subLayer->subLayerKey = subLayerKey;
		if (weight > 0 && weight<=0xFFFF)
			subLayer->weight = (UINT16) weight;

		return subLayer;
	}
	EXPORT FWPM_SUBLAYER0* _cdecl FWPM_SUBLAYER0_CreatePtr(GUID *subLayerKey, UINT32 weight) { return FWPM_SUBLAYER0_Create(*subLayerKey, weight); }

	EXPORT DWORD _cdecl FWPM_SUBLAYER0_SetProviderKey(FWPM_SUBLAYER0 *subLayer, GUID providerKey)
	{
		if (subLayer == NULL)
			return -1;

		subLayer->providerKey = new GUID();
		*(subLayer->providerKey) = providerKey;

		return 0;
	}
	EXPORT DWORD _cdecl FWPM_SUBLAYER0_SetProviderKeyPtr(FWPM_SUBLAYER0* subLayer, GUID *providerKey) { return FWPM_SUBLAYER0_SetProviderKey(subLayer, *providerKey); }

	EXPORT DWORD _cdecl FWPM_SUBLAYER0_SetDisplayData(FWPM_SUBLAYER0 *subLayerStruct,
		wchar_t *name, wchar_t *description)
	{
		size_t nameLen = wcslen(name);
		if (nameLen > 256)
			return -1;

		size_t descriptionLen = wcslen(description);
		if (descriptionLen > 256)
			return -1;

		subLayerStruct->displayData.name = new wchar_t[nameLen + 1];
		subLayerStruct->displayData.description = new wchar_t[descriptionLen + 1];

		wcscpy_s(subLayerStruct->displayData.name, nameLen + 1, name);
		wcscpy_s(subLayerStruct->displayData.description, descriptionLen + 1, description);

		return 0;
	}

	EXPORT void _cdecl FWPM_SUBLAYER0_SetWeight(FWPM_SUBLAYER0 *subLayerStruct, INT16 weight)
	{
		subLayerStruct->weight = weight;
	}

	EXPORT void _cdecl FWPM_SUBLAYER0_SetFlags(FWPM_SUBLAYER0 *subLayerStruct, DWORD flags)
	{
		subLayerStruct->flags = flags;		
	}

	EXPORT DWORD _cdecl FWPM_SUBLAYER0_Delete(FWPM_SUBLAYER0 *subLayerStruct)
	{
		if (subLayerStruct->providerKey != NULL)
			delete subLayerStruct->providerKey;

		if (subLayerStruct->displayData.name != NULL)
			delete[] subLayerStruct->displayData.name;

		if (subLayerStruct->displayData.description != NULL)
			delete[] subLayerStruct->displayData.description;

		delete subLayerStruct;

		return 0;
	}
}