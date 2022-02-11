#include "stdafx.h"

extern "C" {

	EXPORT DWORD _cdecl WfpGetProviderFlags(HANDLE engineHandle, GUID providerGuid, UINT32 *flags)
	{
		FWPM_PROVIDER0 *provider;		
		DWORD result = FwpmProviderGetByKey0(engineHandle, &providerGuid, &provider);
		if (result != 0)
		{
			*flags = 0;
			return result;
		}

		*flags = provider->flags;
		FwpmFreeMemory0((void **)&provider);

		return result;
	}
	EXPORT DWORD _cdecl WfpGetProviderFlagsPtr(HANDLE engineHandle, GUID *providerGuid, UINT32* flags)	{ return WfpGetProviderFlags(engineHandle, *providerGuid, flags); }

	EXPORT DWORD _cdecl WfpProviderDelete(HANDLE engineHandle, GUID providerGuid)
	{
		return FwpmProviderDeleteByKey0(engineHandle, &providerGuid);
	}
	EXPORT DWORD _cdecl WfpProviderDeletePtr(HANDLE engineHandle, GUID *providerGuid)	{ return WfpProviderDelete(engineHandle, *providerGuid); }

	EXPORT DWORD _cdecl WfpProviderAdd(HANDLE engineHandle, FWPM_PROVIDER0 *provideStruct)
	{
		return FwpmProviderAdd0(engineHandle, provideStruct, NULL);
	}

	EXPORT FWPM_PROVIDER0 * _cdecl FWPM_PROVIDER0_Create(GUID providerKey)
	{
		FWPM_PROVIDER0* provider = new FWPM_PROVIDER0{0};
		provider->providerKey = providerKey;
		return provider;
	}
	EXPORT FWPM_PROVIDER0* _cdecl FWPM_PROVIDER0_CreatePtr(GUID *providerKey)	{ return FWPM_PROVIDER0_Create(*providerKey);	}

	EXPORT DWORD _cdecl FWPM_PROVIDER0_SetFlags(FWPM_PROVIDER0 *providerStruct, UINT32 flags)
	{
		if (providerStruct == NULL)
			return ERROR_INVALID_ADDRESS;

		providerStruct->flags = flags;	
		return 0;
	}

	EXPORT DWORD _cdecl FWPM_PROVIDER0_SetDisplayData(FWPM_PROVIDER0 *provideStruct,
		wchar_t *name, wchar_t *description)
	{
		size_t nameLen = wcslen(name);
		if (nameLen > 256)
			return -1;

		size_t descriptionLen = wcslen(description);
		if (descriptionLen > 256)
			return -1;

		provideStruct->displayData.name = new wchar_t[nameLen + 1];
		provideStruct->displayData.description = new wchar_t[descriptionLen + 1];

		wcscpy_s(provideStruct->displayData.name, nameLen + 1, name);
		wcscpy_s(provideStruct->displayData.description, descriptionLen + 1, description);

		return 0;
	}

	EXPORT DWORD _cdecl FWPM_PROVIDER0_Delete(FWPM_PROVIDER0 *provideStruct)
	{

		if (provideStruct->displayData.name != NULL)
			delete[] provideStruct->displayData.name;

		if (provideStruct->displayData.description != NULL)
			delete[] provideStruct->displayData.description;

		delete provideStruct;

		return 0;
	}
}