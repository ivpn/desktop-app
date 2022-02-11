#include "stdafx.h"

extern "C" {

	EXPORT FWPM_FILTER0 * _cdecl FWPM_FILTER_Create(
		GUID filterKey, 
		GUID layerKey, GUID subLayerKey, 
		UINT8 weight, UINT32 flags)
	{
		FWPM_FILTER0* filter = new FWPM_FILTER0{0};

		filter->filterKey = filterKey;		
		filter->layerKey = layerKey;
		filter->subLayerKey = subLayerKey;

		filter->weight.type = FWP_UINT8;
		filter->weight.uint8 = weight;
		filter->flags = flags;

		return filter;
	}
	EXPORT FWPM_FILTER0* _cdecl FWPM_FILTER_CreatePtr(
		GUID *filterKey,
		GUID *layerKey, GUID *subLayerKey,
		UINT8 weight, UINT32 flags) { return FWPM_FILTER_Create(*filterKey, *layerKey, *subLayerKey, weight, flags); }

	EXPORT void _cdecl FWPM_FILTER_Delete(FWPM_FILTER0 *filter)
	{
		if (filter->providerKey != NULL)
			delete filter->providerKey;

		if (filter->displayData.name != NULL)
			delete[] filter->displayData.name;

		if (filter->displayData.description != NULL)
			delete[] filter->displayData.description;


		for (UINT8 i = 0; i < filter->numFilterConditions; i++)
		{
			switch (filter->filterCondition[i].conditionValue.type)
			{
			case FWP_V4_ADDR_MASK:
				if (filter->filterCondition[i].conditionValue.v4AddrMask!=0)
					delete filter->filterCondition[i].conditionValue.v4AddrMask;
				break;

			case FWP_V6_ADDR_MASK:
				if (filter->filterCondition[i].conditionValue.v6AddrMask != 0)
					delete filter->filterCondition[i].conditionValue.v6AddrMask;
				break;

			case FWP_BYTE_BLOB_TYPE:
				if (filter->filterCondition[i].conditionValue.byteBlob != 0)
				{
					FwpmFreeMemory0((void**)&(filter->filterCondition[i].conditionValue.byteBlob));					
				}
				break;
			default:
				// unknown filter type (bug?)
				filter = filter;// just for breakpoint
				break;
			}			
		}

		if (filter->numFilterConditions != 0)
			delete[] filter->filterCondition;

		delete filter;
	}

	EXPORT DWORD _cdecl FWPM_FILTER_SetProviderKey(FWPM_FILTER0 *filter, GUID providerKey)
	{
		if (filter == NULL)
			return -1;
		
		filter->providerKey = new GUID();
		*(filter->providerKey) = providerKey;

		return ERROR_SUCCESS;
	}
	EXPORT DWORD _cdecl FWPM_FILTER_SetProviderKeyPtr(FWPM_FILTER0* filter, GUID *providerKey) { return FWPM_FILTER_SetProviderKey(filter, *providerKey); }

	EXPORT DWORD _cdecl FWPM_FILTER_SetDisplayData(FWPM_FILTER0 *filter,
		wchar_t *name, wchar_t *description)
	{
		size_t nameLen = wcslen(name);
		if (nameLen > 256)
			return -1;

		size_t descriptionLen = wcslen(description);
		if (descriptionLen > 256)
			return -1;

		filter->displayData.name = new wchar_t[nameLen + 1]{0};
		filter->displayData.description = new wchar_t[descriptionLen + 1]{0};

		wcscpy_s(filter->displayData.name, nameLen + 1, name);
		wcscpy_s(filter->displayData.description, descriptionLen + 1, description);

		return ERROR_SUCCESS;
	}
	
	EXPORT DWORD _cdecl FWPM_FILTER_AllocateConditions(FWPM_FILTER0 *filter, INT32 numFilterConditions)
	{
		if (filter == NULL)
			return -1;

		if (numFilterConditions > 10)
			return -1;

		filter->numFilterConditions = numFilterConditions;
		filter->filterCondition = new FWPM_FILTER_CONDITION0[numFilterConditions]{0};
		
		return ERROR_SUCCESS;
	}

	int CheckFilter(FWPM_FILTER0 *filter, UINT32 conditionIndex)
	{
		if (filter == NULL)
			return -1;

		if (conditionIndex >= filter->numFilterConditions)
			return -2;

		return ERROR_SUCCESS;
	}

	EXPORT DWORD _cdecl FWPM_FILTER_SetConditionFieldKey(FWPM_FILTER0 *filter,
		UINT32 conditionIndex, GUID fieldKey)		
	{
		DWORD checkFilterResult = CheckFilter(filter, conditionIndex);
		if (checkFilterResult != ERROR_SUCCESS)
			return checkFilterResult;

		filter->filterCondition[conditionIndex].fieldKey = fieldKey;		

		return ERROR_SUCCESS;
	}
	EXPORT DWORD _cdecl FWPM_FILTER_SetConditionFieldKeyPtr(FWPM_FILTER0* filter,
		UINT32 conditionIndex, GUID *fieldKey) { return FWPM_FILTER_SetConditionFieldKey(filter, conditionIndex, *fieldKey); }

	EXPORT DWORD _cdecl FWPM_FILTER_SetConditionMatchType(FWPM_FILTER0 *filter,
		UINT32 conditionIndex, FWP_MATCH_TYPE matchType)
	{
		DWORD checkFilterResult = CheckFilter(filter, conditionIndex);
		if (checkFilterResult != ERROR_SUCCESS)
			return checkFilterResult;
		
		filter->filterCondition[conditionIndex].matchType = matchType;

		return ERROR_SUCCESS;
	}

	EXPORT DWORD _cdecl FWPM_FILTER_SetConditionV4AddrMask(FWPM_FILTER0 *filter,
		UINT32 conditionIndex, UINT32 address, UINT32 mask)
	{
		DWORD checkFilterResult = CheckFilter(filter, conditionIndex);
		if (checkFilterResult != ERROR_SUCCESS)
			return checkFilterResult;

		filter->filterCondition[conditionIndex].conditionValue.type = FWP_V4_ADDR_MASK;
		filter->filterCondition[conditionIndex].conditionValue.v4AddrMask = new FWP_V4_ADDR_AND_MASK{0};
		filter->filterCondition[conditionIndex].conditionValue.v4AddrMask->addr = address;
		filter->filterCondition[conditionIndex].conditionValue.v4AddrMask->mask = mask;

		return ERROR_SUCCESS;
	}	

	EXPORT DWORD _cdecl FWPM_FILTER_SetConditionV6AddrMask(FWPM_FILTER0 *filter,
		UINT32 conditionIndex, UINT8 address[FWP_V6_ADDR_SIZE], UINT8 prefixLength)
	{
		DWORD checkFilterResult = CheckFilter(filter, conditionIndex);
		if (checkFilterResult != ERROR_SUCCESS)
			return checkFilterResult;

		filter->filterCondition[conditionIndex].conditionValue.type = FWP_V6_ADDR_MASK;
		filter->filterCondition[conditionIndex].conditionValue.v6AddrMask = new FWP_V6_ADDR_AND_MASK{0};
		filter->filterCondition[conditionIndex].conditionValue.v6AddrMask->prefixLength = prefixLength;

		for (int i = 0; i < FWP_V6_ADDR_SIZE; i++)
			filter->filterCondition[conditionIndex].conditionValue.v6AddrMask->addr[i] = address[i];

		return ERROR_SUCCESS;
	}

	EXPORT DWORD _cdecl FWPM_FILTER_SetConditionUINT16(FWPM_FILTER0 *filter, 
				UINT32 conditionIndex, UINT16 port)
	{
		DWORD checkFilterResult = CheckFilter(filter, conditionIndex);
		if (checkFilterResult != 0)
			return checkFilterResult;

		filter->filterCondition[conditionIndex].conditionValue.type = FWP_UINT16;
		filter->filterCondition[conditionIndex].conditionValue.uint16 = port;

		return ERROR_SUCCESS;
	}

	EXPORT DWORD _cdecl FWPM_FILTER_SetConditionBlobString(FWPM_FILTER0 *filter, 
		UINT32 conditionIndex, wchar_t *blobString)
	{
		DWORD checkFilterResult = CheckFilter(filter, conditionIndex);
		if (checkFilterResult != 0)
			return checkFilterResult;

		FWP_BYTE_BLOB * blob;		
		// NOTE! The caller must free the returned object by a call to FwpmFreeMemory0.
		// We are doing that in FWPM_FILTER_Delete()
		DWORD result = FwpmGetAppIdFromFileName0(blobString, &blob);
		if (result != ERROR_SUCCESS)
			return result;
			
		filter->filterCondition[conditionIndex].conditionValue.type = FWP_BYTE_BLOB_TYPE;
		filter->filterCondition[conditionIndex].conditionValue.byteBlob = blob;
		

		return ERROR_SUCCESS;
	}

	EXPORT DWORD _cdecl FWPM_FILTER_SetAction(FWPM_FILTER0 *filter, FWP_ACTION_TYPE filterActionType)
	{
		if (filter == NULL)
			return -1;
		
		filter->action.type = filterActionType;

		return ERROR_SUCCESS;
	}

	EXPORT DWORD _cdecl FWPM_FILTER_SetFlags(FWPM_FILTER0 *filter, UINT32 flags)
	{
		if (filter == NULL)
			return ERROR_INVALID_ADDRESS;

		filter->flags = flags;

		return ERROR_SUCCESS;
	}

	EXPORT DWORD _cdecl WfpFilterAdd(HANDLE engineHandle, FWPM_FILTER0 *filter, UINT64 *id)
	{
		return FwpmFilterAdd0(engineHandle, filter, 0, id);
	}

	EXPORT DWORD _cdecl WfpFilterDeleteById(HANDLE engineHandle, UINT64 id)
	{
		return FwpmFilterDeleteById0(engineHandle, id);
	}

	DWORD FindMatchingCallouts(
		HANDLE engine,
		GUID providerKey,
		GUID layerKey,
		FWPM_CALLOUT0*** callouts,
		UINT32* numCallouts
	)
	{
		DWORD result = ERROR_SUCCESS;
		HANDLE enumHandle = NULL;
		FWPM_CALLOUT_ENUM_TEMPLATE0 enumTempl;

		memset(&enumTempl, 0, sizeof(enumTempl));
		enumTempl.layerKey = layerKey;
		enumTempl.providerKey = new GUID();
		*(enumTempl.providerKey) = providerKey;

		result = FwpmCalloutCreateEnumHandle0(
			engine,
			&enumTempl,
			&enumHandle
		);

		if (result != ERROR_SUCCESS)
		{
			delete enumTempl.providerKey;
			return result;
		}

		result = FwpmCalloutEnum0(engine,
			enumHandle,
			INFINITE,			
			callouts,
			numCallouts
		);

		delete enumTempl.providerKey;
		FwpmCalloutDestroyEnumHandle0(engine, enumHandle);

		return result;
	}

	DWORD FindMatchingFilters(
		HANDLE engine,
		GUID providerKey,
		GUID layerKey,
		FWPM_FILTER0*** filters,
		UINT32* numFilters
		)
	{

		DWORD result = ERROR_SUCCESS;
		HANDLE enumHandle = NULL;
		FWPM_FILTER_ENUM_TEMPLATE0 enumTempl;

		memset(&enumTempl, 0, sizeof(enumTempl));
		enumTempl.layerKey = layerKey;
		enumTempl.providerKey = new GUID();
		enumTempl.flags = FWP_FILTER_ENUM_FLAG_INCLUDE_BOOTTIME;
		*(enumTempl.providerKey) = providerKey;

		enumTempl.actionMask = 0xFFFFFFFF;

		result = FwpmFilterCreateEnumHandle0(
			engine,
			&enumTempl,
			&enumHandle
			);

		if (result != ERROR_SUCCESS)
		{
			delete enumTempl.providerKey;
			return result;
		}

		result = FwpmFilterEnum0(engine,
								 enumHandle,
								 INFINITE,
								 filters,
								 numFilters);

		delete enumTempl.providerKey;

		FwpmFilterDestroyEnumHandle0(engine, enumHandle);
		return result;
	}

	EXPORT DWORD _cdecl WfpFiltersDeleteByProviderKey(HANDLE engineHandle, GUID providerKey, GUID layerKey)
	{
		// Delete all filters by providerKey+layerKey
		UINT32 numFilters = 0;
		FWPM_FILTER0** filters = 0;
		DWORD result;

		result = FindMatchingFilters(engineHandle, providerKey, layerKey, &filters, &numFilters);
		if (result != ERROR_SUCCESS)				
			return result;
		
		for (UINT32 i = 0; i < numFilters; i++)
		{
			FWPM_FILTER0* filter = filters[i];
			if (filter->providerKey != NULL && 
				*(filter->providerKey) == providerKey) 
			{				
				result = FwpmFilterDeleteById0(engineHandle, filter->filterId);
				if (result != ERROR_SUCCESS)
				{
					FwpmFreeMemory0((void**)&filters);
					return result;
				}
			}
		}

		FwpmFreeMemory0((void**)&filters);

		// Delete all callouts by providerKey+layerKey
		UINT32 numCallouts = 0;
		FWPM_CALLOUT0** callouts = 0;
		
		result = FindMatchingCallouts(engineHandle, providerKey, layerKey, &callouts, &numCallouts);
		if (result != ERROR_SUCCESS)
			return result;

		for (UINT32 i = 0; i < numCallouts; i++)
		{
			FWPM_CALLOUT0* callout = callouts[i];
			if (callout->providerKey != NULL &&
				*(callout->providerKey) == providerKey)
			{
				result = FwpmCalloutDeleteById0(engineHandle, callout->calloutId);
				if (result != ERROR_SUCCESS)
				{
					FwpmFreeMemory0((void**)&callouts);
					return result;
				}
			}
		}

		FwpmFreeMemory0((void**)&callouts);

		return ERROR_SUCCESS;

	}
	EXPORT DWORD _cdecl WfpFiltersDeleteByProviderKeyPtr(HANDLE engineHandle, GUID *providerKey, GUID *layerKey) { return WfpFiltersDeleteByProviderKey(engineHandle, *providerKey, *layerKey); }
}