#include "Filters.h"
#include "Filters.tmh"

namespace wfp
{
	NTSTATUS RegisterFilterBindRedirectIpv4 ( HANDLE wfpEngineHandle)
	{
		// In use for non-TCP protocols

		FWPM_FILTER0 filter = { 0 };
		UINT64 weight = MAXUINT64;

		const auto filterName = L"IVPN Split Tunnel filter (BIND_REDIRECT_V4)";
		const auto filterDescription = L"Fits only for non-TCP connections";

		filter.filterKey = KEY_FILTER_CALLOUT_ALE_BIND_REDIRECT_V4;
		filter.displayData.name = const_cast<wchar_t*>(filterName);
		filter.displayData.description = const_cast<wchar_t*>(filterDescription);
		filter.flags = FWPM_FILTER_FLAG_CLEAR_ACTION_RIGHT;
		filter.providerKey = const_cast<GUID*>(&KEY_IVPN_FW_PROVIDER);
		filter.layerKey = FWPM_LAYER_ALE_BIND_REDIRECT_V4;
		filter.subLayerKey = KEY_IVPN_FW_SUBLAYER;
		filter.weight.type = FWP_UINT64;
		filter.weight.uint64 = const_cast<UINT64*>(&weight);
		filter.action.type = FWP_ACTION_CALLOUT_UNKNOWN;
		filter.action.calloutKey = KEY_CALLOUT_ALE_BIND_REDIRECT_V4;

		FWPM_FILTER_CONDITION0 cond;

		cond.fieldKey = FWPM_CONDITION_IP_PROTOCOL;
		cond.matchType = FWP_MATCH_NOT_EQUAL;
		cond.conditionValue.type = FWP_UINT8;
		cond.conditionValue.uint8 = IPPROTO_TCP;

		filter.filterCondition = &cond;
		filter.numFilterConditions = 1;

		NTSTATUS status = FwpmFilterAdd0(wfpEngineHandle, &filter, NULL, NULL);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) failed':  %!STATUS!", status);

		return status;
	}

	NTSTATUS UnRegisterFilterBindRedirectIpv4 ( HANDLE wfpEngineHandle)
	{
		NTSTATUS status = FwpmFilterDeleteByKey0(wfpEngineHandle, &KEY_FILTER_CALLOUT_ALE_BIND_REDIRECT_V4);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) failed':  %!STATUS!", status);

		return status;
	}

	
	NTSTATUS RegisterFilterConnectRedirectIpv4 ( HANDLE wfpEngineHandle)
	{
		// In use for TCP protocols

		FWPM_FILTER0 filter = { 0 };
		UINT64 weight = MAXUINT64;

		const auto filterName = L"IVPN Split Tunnel filter (CONNECT_REDIRECT_V4)";
		const auto filterDescription = L"Fits only for TCP connections";

		filter.filterKey = KEY_FILTER_CALLOUT_ALE_CONNECT_REDIRECT_V4;
		filter.displayData.name = const_cast<wchar_t*>(filterName);
		filter.displayData.description = const_cast<wchar_t*>(filterDescription);
		filter.flags = FWPM_FILTER_FLAG_CLEAR_ACTION_RIGHT;
		filter.providerKey = const_cast<GUID*>(&KEY_IVPN_FW_PROVIDER);
		filter.layerKey = FWPM_LAYER_ALE_CONNECT_REDIRECT_V4;
		filter.subLayerKey = KEY_IVPN_FW_SUBLAYER;
		filter.weight.type = FWP_UINT64;
		filter.weight.uint64 = const_cast<UINT64*>(&weight);

		// TODO: https://docs.microsoft.com/en-us/windows-hardware/drivers/network/types-of-callouts
		//The filter action type for this type of callout should be set to FWP_ACTION_PERMIT.
		filter.action.type = FWP_ACTION_CALLOUT_UNKNOWN;

		filter.action.calloutKey = KEY_CALLOUT_ALE_CONNECT_REDIRECT_V4;

		FWPM_FILTER_CONDITION0 cond;

		cond.fieldKey = FWPM_CONDITION_IP_PROTOCOL;
		cond.matchType = FWP_MATCH_EQUAL;
		cond.conditionValue.type = FWP_UINT8;
		cond.conditionValue.uint8 = IPPROTO_TCP;

		filter.filterCondition = &cond;
		filter.numFilterConditions = 1;

		NTSTATUS status = FwpmFilterAdd0(wfpEngineHandle, &filter, NULL, NULL);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) failed':  %!STATUS!", status);

		return status;
	}

	NTSTATUS UnRegisterFilterConnectRedirectIpv4 ( HANDLE wfpEngineHandle)
	{
		NTSTATUS status = FwpmFilterDeleteByKey0(wfpEngineHandle, &KEY_FILTER_CALLOUT_ALE_CONNECT_REDIRECT_V4);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) failed':  %!STATUS!", status);

		return status;
	}

	NTSTATUS RegisterFilterBindRedirectIpv6 ( HANDLE wfpEngineHandle)
	{
		// In use for non-TCP protocols

		FWPM_FILTER0 filter = { 0 };
		UINT64 weight = MAXUINT64;

		const auto filterName = L"IVPN Split Tunnel filter (BIND_REDIRECT_V6)";
		const auto filterDescription = L"Fits only for non-TCP connections";

		filter.filterKey = KEY_FILTER_CALLOUT_ALE_BIND_REDIRECT_V6;
		filter.displayData.name = const_cast<wchar_t*>(filterName);
		filter.displayData.description = const_cast<wchar_t*>(filterDescription);
		filter.flags = FWPM_FILTER_FLAG_CLEAR_ACTION_RIGHT;
		filter.providerKey = const_cast<GUID*>(&KEY_IVPN_FW_PROVIDER);
		filter.layerKey = FWPM_LAYER_ALE_BIND_REDIRECT_V6;
		filter.subLayerKey = KEY_IVPN_FW_SUBLAYER;
		filter.weight.type = FWP_UINT64;
		filter.weight.uint64 = const_cast<UINT64*>(&weight);
		filter.action.type = FWP_ACTION_CALLOUT_UNKNOWN;
		filter.action.calloutKey = KEY_CALLOUT_ALE_BIND_REDIRECT_V6;

		FWPM_FILTER_CONDITION0 cond;

		cond.fieldKey = FWPM_CONDITION_IP_PROTOCOL;
		cond.matchType = FWP_MATCH_NOT_EQUAL;
		cond.conditionValue.type = FWP_UINT8;
		cond.conditionValue.uint8 = IPPROTO_TCP;

		filter.filterCondition = &cond;
		filter.numFilterConditions = 1;

		NTSTATUS status = FwpmFilterAdd0(wfpEngineHandle, &filter, NULL, NULL);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) failed':  %!STATUS!", status);

		return status;
	}

	NTSTATUS UnRegisterFilterBindRedirectIpv6 ( HANDLE wfpEngineHandle)
	{
		NTSTATUS status = FwpmFilterDeleteByKey0(wfpEngineHandle, &KEY_FILTER_CALLOUT_ALE_BIND_REDIRECT_V6);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) failed':  %!STATUS!", status);

		return status;
	}

	NTSTATUS RegisterFilterConnectRedirectIpv6 ( HANDLE wfpEngineHandle)
	{
		// In use for TCP protocols

		FWPM_FILTER0 filter = { 0 };
		UINT64 weight = MAXUINT64;

		const auto filterName = L"IVPN Split Tunnel filter (CONNECT_REDIRECT_V6)";
		const auto filterDescription = L"Fits only for TCP connections";

		filter.filterKey = KEY_FILTER_CALLOUT_ALE_CONNECT_REDIRECT_V6;
		filter.displayData.name = const_cast<wchar_t*>(filterName);
		filter.displayData.description = const_cast<wchar_t*>(filterDescription);
		filter.flags = FWPM_FILTER_FLAG_CLEAR_ACTION_RIGHT;
		filter.providerKey = const_cast<GUID*>(&KEY_IVPN_FW_PROVIDER);
		filter.layerKey = FWPM_LAYER_ALE_CONNECT_REDIRECT_V6;
		filter.subLayerKey = KEY_IVPN_FW_SUBLAYER;
		filter.weight.type = FWP_UINT64;
		filter.weight.uint64 = const_cast<UINT64*>(&weight);

		// TODO: https://docs.microsoft.com/en-us/windows-hardware/drivers/network/types-of-callouts
		//The filter action type for this type of callout should be set to FWP_ACTION_PERMIT.
		filter.action.type = FWP_ACTION_CALLOUT_UNKNOWN;

		filter.action.calloutKey = KEY_CALLOUT_ALE_CONNECT_REDIRECT_V6;

		FWPM_FILTER_CONDITION0 cond;

		cond.fieldKey = FWPM_CONDITION_IP_PROTOCOL;
		cond.matchType = FWP_MATCH_EQUAL;
		cond.conditionValue.type = FWP_UINT8;
		cond.conditionValue.uint8 = IPPROTO_TCP;

		filter.filterCondition = &cond;
		filter.numFilterConditions = 1;

		NTSTATUS status = FwpmFilterAdd0(wfpEngineHandle, &filter, NULL, NULL);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) failed':  %!STATUS!", status);

		return status;
	}

	NTSTATUS UnRegisterFilterConnectRedirectIpv6 ( HANDLE wfpEngineHandle)
	{
		NTSTATUS status = FwpmFilterDeleteByKey0(wfpEngineHandle, &KEY_FILTER_CALLOUT_ALE_CONNECT_REDIRECT_V6);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) failed':  %!STATUS!", status);

		return status;
	}
}
