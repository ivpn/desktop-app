#include "Filters.h"
#include "Filters.tmh"

namespace wfp
{
	NTSTATUS AddCalloutFilter(HANDLE wfpEngineHandle, const wchar_t* filterName, const wchar_t* filterDescription, 
		const GUID filterKey, const GUID layerKey, const GUID calloutKey)
	{
		FWPM_FILTER0 filter = { 0 };
		UINT64 weight = MAXUINT64;

		filter.filterKey = filterKey;
		filter.displayData.name = const_cast<wchar_t*>(filterName);
		filter.displayData.description = const_cast<wchar_t*>((filterDescription!=NULL)? filterDescription : filterName);
		filter.flags = FWPM_FILTER_FLAG_CLEAR_ACTION_RIGHT;
		filter.providerKey = const_cast<GUID*>(&KEY_IVPN_ST_PROVIDER);
		filter.layerKey = layerKey;
		filter.subLayerKey = KEY_IVPN_ST_SUBLAYER;
		
		filter.weight.type = FWP_UINT64;
		filter.weight.uint64 = const_cast<UINT64*>(&weight);
		//filter.weight.type = FWP_UINT8;
		//filter.weight.uint8 = 0xF;

		filter.action.type = FWP_ACTION_CALLOUT_UNKNOWN;
		filter.action.calloutKey = calloutKey;

		// catch all connections
		filter.numFilterConditions = 0;

		NTSTATUS status = FwpmFilterAdd0(wfpEngineHandle, &filter, NULL, NULL);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) failed':  %!STATUS!", status);

		return status;
	}

	NTSTATUS DeleteFilter(HANDLE wfpEngineHandle, const GUID filterKey)
	{
		NTSTATUS status = FwpmFilterDeleteByKey0(wfpEngineHandle, &filterKey);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) failed':  %!STATUS!", status);

		return status;
	}

	NTSTATUS RegisterFilters(HANDLE wfpEngineHandle)
	{
		NTSTATUS status;

		//
		// REDIRECTION CALLOUTS
		// 
		
		// BIND_REDIRECT_V4
		status = AddCalloutFilter(wfpEngineHandle, 
			L"IVPN Split Tunnel filter (BIND_REDIRECT_V4)", NULL,
			KEY_FILTER_CALLOUT_ALE_BIND_REDIRECT_V4, 
			FWPM_LAYER_ALE_BIND_REDIRECT_V4, 
			KEY_CALLOUT_ALE_BIND_REDIRECT_V4);
		if (!NT_SUCCESS(status))
			return status;

		// CONNECT_REDIRECT_V4
		status = AddCalloutFilter(wfpEngineHandle,
			L"IVPN Split Tunnel filter (CONNECT_REDIRECT_V4)", NULL,
			KEY_FILTER_CALLOUT_ALE_CONNECT_REDIRECT_V4, 
			FWPM_LAYER_ALE_CONNECT_REDIRECT_V4, 
			KEY_CALLOUT_ALE_CONNECT_REDIRECT_V4);
		if (!NT_SUCCESS(status))
			return status;
				
		// BIND_REDIRECT_V6
		status = AddCalloutFilter(wfpEngineHandle,
			L"IVPN Split Tunnel filter (BIND_REDIRECT_V6)", NULL,
			KEY_FILTER_CALLOUT_ALE_BIND_REDIRECT_V6, 
			FWPM_LAYER_ALE_BIND_REDIRECT_V6, 
			KEY_CALLOUT_ALE_BIND_REDIRECT_V6);
		if (!NT_SUCCESS(status))
			return status;

		// CONNECT_REDIRECT_V6
		status = AddCalloutFilter(wfpEngineHandle,
			L"IVPN Split Tunnel filter (CONNECT_REDIRECT_V6)", NULL,
			KEY_FILTER_CALLOUT_ALE_CONNECT_REDIRECT_V6, 
			FWPM_LAYER_ALE_CONNECT_REDIRECT_V6, 
			KEY_CALLOUT_ALE_CONNECT_REDIRECT_V6);
		if (!NT_SUCCESS(status))
			return status;

		//
		// PERMIT/BLOCK CALLOUTS
		// 

		// ALE_AUTH_CONNECT_V4
		status = AddCalloutFilter(wfpEngineHandle,
			L"IVPN Split Tunnel filter (ALE_AUTH_CONNECT_V4)", NULL,
			KEY_FILTER_CALLOUT_ALE_AUTH_CONNECT_V4_ST_INTERNAL,
			FWPM_LAYER_ALE_AUTH_CONNECT_V4,
			KEY_CALLOUT_ALE_AUTH_CONNECT_V4);
		if (!NT_SUCCESS(status))
			return status;

		// ALE_AUTH_CONNECT_V6
		status = AddCalloutFilter(wfpEngineHandle,
			L"IVPN Split Tunnel filter (ALE_AUTH_CONNECT_V6)", NULL,
			KEY_FILTER_CALLOUT_ALE_AUTH_CONNECT_V6_ST_INTERNAL,
			FWPM_LAYER_ALE_AUTH_CONNECT_V6,
			KEY_CALLOUT_ALE_AUTH_CONNECT_V6);
		if (!NT_SUCCESS(status))
			return status;

		// ALE_AUTH_RECV_ACCEPT_V4
		status = AddCalloutFilter(wfpEngineHandle,
			L"IVPN Split Tunnel filter (ALE_AUTH_RECV_ACCEPT_V4)", NULL,
			KEY_FILTER_CALLOUT_ALE_AUTH_RECV_ACCEPT_V4_ST_INTERNAL,
			FWPM_LAYER_ALE_AUTH_RECV_ACCEPT_V4,
			KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT);
		if (!NT_SUCCESS(status))
			return status;

		// ALE_AUTH_RECV_ACCEPT_V6
		status = AddCalloutFilter(wfpEngineHandle,
			L"IVPN Split Tunnel filter (ALE_AUTH_RECV_ACCEPT_V6)", NULL,
			KEY_FILTER_CALLOUT_ALE_AUTH_RECV_ACCEPT_V6_ST_INTERNAL,
			FWPM_LAYER_ALE_AUTH_RECV_ACCEPT_V6,
			KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT_V6);
		if (!NT_SUCCESS(status))
			return status;

		return status;
	}

	NTSTATUS UnRegisterFilters(HANDLE wfpEngineHandle)
	{
		NTSTATUS ret = STATUS_SUCCESS, status;

		// REDIRECTION CALLOUTS

		status = DeleteFilter(wfpEngineHandle, KEY_FILTER_CALLOUT_ALE_BIND_REDIRECT_V4);
		if (!NT_SUCCESS(status) && (NT_SUCCESS(ret)))
			ret = status;

		status = DeleteFilter(wfpEngineHandle, KEY_FILTER_CALLOUT_ALE_CONNECT_REDIRECT_V4);
		if (!NT_SUCCESS(status) && (NT_SUCCESS(ret)))
			ret = status;

		status = DeleteFilter(wfpEngineHandle, KEY_FILTER_CALLOUT_ALE_BIND_REDIRECT_V6);
		if (!NT_SUCCESS(status) && (NT_SUCCESS(ret)))
			ret = status;

		status = DeleteFilter(wfpEngineHandle, KEY_FILTER_CALLOUT_ALE_CONNECT_REDIRECT_V6);
		if (!NT_SUCCESS(status) && (NT_SUCCESS(ret)))
			ret = status;
				
		// PERMIT/BLOCK CALLOUTS

		status = DeleteFilter(wfpEngineHandle, KEY_FILTER_CALLOUT_ALE_AUTH_CONNECT_V4_ST_INTERNAL);
		if (!NT_SUCCESS(status) && (NT_SUCCESS(ret)))
			ret = status;

		status = DeleteFilter(wfpEngineHandle, KEY_FILTER_CALLOUT_ALE_AUTH_CONNECT_V6_ST_INTERNAL);
		if (!NT_SUCCESS(status) && (NT_SUCCESS(ret)))
			ret = status;

		status = DeleteFilter(wfpEngineHandle, KEY_FILTER_CALLOUT_ALE_AUTH_RECV_ACCEPT_V4_ST_INTERNAL);
		if (!NT_SUCCESS(status) && (NT_SUCCESS(ret)))
			ret = status;

		status = DeleteFilter(wfpEngineHandle, KEY_FILTER_CALLOUT_ALE_AUTH_RECV_ACCEPT_V6_ST_INTERNAL);
		if (!NT_SUCCESS(status) && (NT_SUCCESS(ret)))
			ret = status;

		return ret;
	}
}
