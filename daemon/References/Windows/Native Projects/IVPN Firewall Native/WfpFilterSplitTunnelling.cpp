#include "stdafx.h"

#include <initguid.h>

// {F6BF8F75-1BCC-43D2-9BD9-FD2922588F50}
DEFINE_GUID(KEY_FILTER_CALLOUT_ALE_AUTH_CONNECT_V4,
	0xf6bf8f75, 0x1bcc, 0x43d2, 0x9b, 0xd9, 0xfd, 0x29, 0x22, 0x58, 0x8f, 0x50);
// {EA063DEC-FDBA-4994-8002-75EA2146F909}
DEFINE_GUID(KEY_FILTER_CALLOUT_ALE_AUTH_CONNECT_V6,
	0xea063dec, 0xfdba, 0x4994, 0x80, 0x2, 0x75, 0xea, 0x21, 0x46, 0xf9, 0x9);
// {2A8E7616-414E-4C7A-B479-E16C625ACC00}
DEFINE_GUID(KEY_FILTER_CALLOUT_ALE_AUTH_RECV_ACCEPT_V4,
	0x2a8e7616, 0x414e, 0x4c7a, 0xb4, 0x79, 0xe1, 0x6c, 0x62, 0x5a, 0xcc, 0x0);
// {1507C6F3-5B27-4997-8368-954A2097A8D5}
DEFINE_GUID(KEY_FILTER_CALLOUT_ALE_AUTH_RECV_ACCEPT_V6,
	0x1507c6f3, 0x5b27, 0x4997, 0x83, 0x68, 0x95, 0x4a, 0x20, 0x97, 0xa8, 0xd5);

//
// NOTE!: The callouts with the GUIDs (bellow) are registered by split-tunnelling driver
//

// {100DD8BC-5C6C-4989-99CF-EB93B14AFA69}
DEFINE_GUID(KEY_CALLOUT_ALE_AUTH_CONNECT_V4,
	0x100dd8bc, 0x5c6c, 0x4989, 0x99, 0xcf, 0xeb, 0x93, 0xb1, 0x4a, 0xfa, 0x69);
// {7C4E6A94-7284-4592-B394-B3369770F30D}
DEFINE_GUID(KEY_CALLOUT_ALE_AUTH_CONNECT_V6,
	0x7c4e6a94, 0x7284, 0x4592, 0xb3, 0x94, 0xb3, 0x36, 0x97, 0x70, 0xf3, 0xd);
// {D7FD0B39-89FE-4E13-9FE4-52F97170F098}
DEFINE_GUID(KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT_V4,
	0xd7fd0b39, 0x89fe, 0x4e13, 0x9f, 0xe4, 0x52, 0xf9, 0x71, 0x70, 0xf0, 0x98);
// {67C57157-8A6B-4AF2-8DAA-5F06372F5DAB}
DEFINE_GUID(KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT_V6,
	0x67c57157, 0x8a6b, 0x4af2, 0x8d, 0xaa, 0x5f, 0x6, 0x37, 0x2f, 0x5d, 0xab);

extern "C" {

	DWORD registerSTFilter(HANDLE wfpEngineHandle, GUID providerKey, GUID subLayerKey, GUID filterKey, GUID calloutKey)
	{
		// add callout

		FWPM_CALLOUT0 mCallout = { 0 };

		const auto calloutName = L"IVPN Firewall callout for Split-Tunnelling";
		const auto calloutDescription = L"Allow communications for splitted appls";

		mCallout.calloutKey = calloutKey;
		mCallout.displayData.name = const_cast<wchar_t*>(calloutName);
		mCallout.displayData.description = const_cast<wchar_t*>(calloutDescription);
		mCallout.providerKey = const_cast<GUID*>(&providerKey);
		mCallout.applicableLayer = FWPM_LAYER_ALE_AUTH_CONNECT_V4;

		auto status = FwpmCalloutAdd0(wfpEngineHandle, &mCallout, NULL, NULL);

		if (status!=0)
			return status;

		// add filter

		FWPM_FILTER0 filter = { 0 };
		UINT64 weight = MAXUINT64;

		const auto filterName = L"IVPN Firewall filter for Split-Tunnel callout";
		const auto filterDescription = L"Allow communications for splitted appls";

		filter.filterKey = filterKey;
		filter.displayData.name = const_cast<wchar_t*>(filterName);
		filter.displayData.description = const_cast<wchar_t*>(filterDescription);
		filter.flags = FWPM_FILTER_FLAG_CLEAR_ACTION_RIGHT;
		filter.providerKey = const_cast<GUID*>(&providerKey);
		filter.layerKey = FWPM_LAYER_ALE_AUTH_CONNECT_V4;
		filter.subLayerKey = subLayerKey;
		filter.weight.type = FWP_UINT64;
		filter.weight.uint64 = const_cast<UINT64*>(&weight);
		filter.action.type = FWP_ACTION_CALLOUT_UNKNOWN; 
		filter.action.calloutKey = calloutKey;

		filter.numFilterConditions = 0;

		return FwpmFilterAdd0(wfpEngineHandle, &filter, NULL, NULL);		
	}

	EXPORT DWORD _cdecl WfpRegisterSplitTunFilters(HANDLE wfpEngineHandle, GUID* providerKey, GUID* subLayerKey)
	{
		DWORD ret = 0, r;
		r = registerSTFilter(wfpEngineHandle, *providerKey, *subLayerKey,	KEY_FILTER_CALLOUT_ALE_AUTH_CONNECT_V4, KEY_CALLOUT_ALE_AUTH_CONNECT_V4	);
		if (ret == 0 && r != 0) ret = r;
		r = registerSTFilter(wfpEngineHandle, *providerKey,	*subLayerKey,	KEY_FILTER_CALLOUT_ALE_AUTH_CONNECT_V6,	KEY_CALLOUT_ALE_AUTH_CONNECT_V6);
		if (ret == 0 && r != 0) ret = r;
		r = registerSTFilter(wfpEngineHandle, *providerKey,	*subLayerKey,	KEY_FILTER_CALLOUT_ALE_AUTH_RECV_ACCEPT_V4,	KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT_V4);
		if (ret == 0 && r != 0) ret = r;
		r = registerSTFilter(wfpEngineHandle, *providerKey,	*subLayerKey,	KEY_FILTER_CALLOUT_ALE_AUTH_RECV_ACCEPT_V6,	KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT_V6);
		if (ret == 0 && r != 0) ret = r;

		return ret;
	}

	/*
	EXPORT DWORD _cdecl WfpUnRegisterSplitTunFilters(HANDLE wfpEngineHandle)
	{
		DWORD ret = 0, r;
		r = FwpmFilterDeleteByKey0(wfpEngineHandle, &KEY_FILTER_CALLOUT_ALE_AUTH_CONNECT_V4);
		if (ret == 0 && r != 0) ret = r;
		r = FwpmFilterDeleteByKey0(wfpEngineHandle, &KEY_FILTER_CALLOUT_ALE_AUTH_CONNECT_V6);
		if (ret == 0 && r != 0) ret = r;
		r = FwpmFilterDeleteByKey0(wfpEngineHandle, &KEY_FILTER_CALLOUT_ALE_AUTH_RECV_ACCEPT_V4);
		if (ret == 0 && r != 0) ret = r;
		r = FwpmFilterDeleteByKey0(wfpEngineHandle, &KEY_FILTER_CALLOUT_ALE_AUTH_RECV_ACCEPT_V6);
		if (ret == 0 && r != 0) ret = r;

		return ret;
	}*/
}