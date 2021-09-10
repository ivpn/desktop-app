#include "Callouts.h"
#include "Callouts.tmh"

#include "../ProcessMonitor/ProcessTree.h"
#include "../Config/GlobalConfig.h"

namespace wfp
{
	bool LocalAddress(const IN_ADDR* addr)
	{
		return IN4_IS_ADDR_LOOPBACK(addr)		// 127/8
			|| IN4_IS_ADDR_LINKLOCAL(addr)		// 169.254/16
			|| IN4_IS_ADDR_RFC1918(addr)		// 10/8, 172.16/12, 192.168/16
			|| IN4_IS_ADDR_MC_LINKLOCAL(addr)	// 224.0.0/24
			|| IN4_IS_ADDR_MC_ADMINLOCAL(addr)	// 239.255/16
			|| IN4_IS_ADDR_MC_SITELOCAL(addr)	// 239/8
			|| IN4_IS_ADDR_BROADCAST(addr)		// 255.255.255.255
			;
	}

	bool IN6_IS_ADDR_ULA(const IN6_ADDR* a)
	{
		return (a->s6_bytes[0] == 0xfd);
	}

	bool IN6_IS_ADDR_MC_NON_GLOBAL(const IN6_ADDR* a)
	{
		return IN6_IS_ADDR_MULTICAST(a)
			&& !IN6_IS_ADDR_MC_GLOBAL(a);
	}

	bool LocalAddress(const IN6_ADDR* addr)
	{
		return IN6_IS_ADDR_LOOPBACK(addr)		// ::1/128
			|| IN6_IS_ADDR_LINKLOCAL(addr)		// fe80::/10
			|| IN6_IS_ADDR_SITELOCAL(addr)		// fec0::/10
			|| IN6_IS_ADDR_ULA(addr)			// fd00::/8
			|| IN6_IS_ADDR_MC_NON_GLOBAL(addr)	// ff00::/8 && !(ffxe::/16)
			;
	}

	/// <summary>
	/// // https://docs.microsoft.com/en-us/windows-hardware/drivers/network/using-bind-or-connect-redirection
	/// </summary>
	void CalloutClassifyConnectOrBindRedirect
		(
			_In_ const FWPS_INCOMING_VALUES0* inFixedValues,
			_In_ const FWPS_INCOMING_METADATA_VALUES0* inMetaValues,
			_Inout_opt_ void* layerData,
			_In_opt_ const void* classifyContext,
			_In_ const FWPS_FILTER1* filter,
			_In_ UINT64 flowContext,
			_Inout_ FWPS_CLASSIFY_OUT0* classifyOut
		)
	{
		DEBUG_PrintElapsedTimeEx(20);

		UNREFERENCED_PARAMETER(layerData);
		UNREFERENCED_PARAMETER(flowContext);

		NT_ASSERT(inFixedValues);
		NT_ASSERT(inMetaValues);
		NT_ASSERT(layerData);
		NT_ASSERT(classifyContext);
		NT_ASSERT(filter);
		NT_ASSERT(classifyOut);
		NT_ASSERT(
			inFixedValues->layerId == FWPS_LAYER_ALE_CONNECT_REDIRECT_V4 ||
			inFixedValues->layerId == FWPS_LAYER_ALE_CONNECT_REDIRECT_V6 ||
			inFixedValues->layerId == FWPS_LAYER_ALE_BIND_REDIRECT_V4 ||
			inFixedValues->layerId == FWPS_LAYER_ALE_BIND_REDIRECT_V6);
		
		// https://docs.microsoft.com/en-us/windows-hardware/drivers/network/using-bind-or-connect-redirection
		// 
		// INFO:
		// If a callout must perform additional processing of packet data outside its classifyFn callout function 
		// before it can determine whether the data should be permitted or blocked, 
		// it must pend the packet data until the processing of the data is completed.
		// For information about how to pend packet data, see Types of Calloutsand FwpsPendOperation0.
		// The FwpsPendClassify0() function is used to pend packets 
		// INFO:
		// 1. Call FwpsRedirectHandleCreate0 to obtain a handle that can be used to redirect TCP connections.
		// This handle should be cached and used for all redirections. (This step is omitted for Windows 7 and earlier.)
		// 2. In Windows 8 and later, you must query the redirection state of the connection 
		// by using the FwpsQueryConnectionRedirectState0 function in your callout driver.
		// This must be done to prevent infinite redirecting.
		// 3. Call FwpsAcquireClassifyHandle0 to obtain a handle that will be used for subsequent function calls.
		// 4. Call FwpsAcquireWritableLayerDataPointer0 to get the writable data structure for the layer 
		// in which classifyFn was called. 
		//		FwpsAcquireWritableLayerDataPointer0 sets the following members of the FWPS_CLASSIFY_OUT0 structure :
		//			classifyOut->actionType = FWP_ACTION_BLOCK
		//			classifyOut->rights &= ~FWPS_RIGHT_ACTION_WRITE
		// 5. Make changes to the layer data as needed

		if (!(classifyOut->rights & FWPS_RIGHT_ACTION_WRITE)) 
		{
			//	classifyOut->actionType: specifies the suggested action to be taken as determined by the callout.
			//	If the FWPS_RIGHT_ACTION_WRITE flag is not set, a callout driver should not write to this member 
			//	unless it is vetoing an FWP_ACTION_PERMIT action that was previously returned by a higher weight filter in the filter engine.
			TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) SKIPPING: FWPS_RIGHT_ACTION_WRITE not set.");
			return;
		}

		if (classifyOut->actionType == FWP_ACTION_NONE)
			classifyOut->actionType = FWP_ACTION_CONTINUE;
		
		if (!FWPS_IS_METADATA_FIELD_PRESENT(inMetaValues, FWPS_METADATA_FIELD_PROCESS_ID))
		{
			TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) SKIPPING: Failed to classify connection because PID was not provided");
			return;
		}
		
		// Checking: is it 'known' process
		// ProcessMonitor keep information only about processes that have to be applied to split-tunnel
		if (NULL == prc::FindProcessInfoForPid((HANDLE)inMetaValues->processId))
		{
			// PID unknown. Do nothing. Just go to the next filter.

			//TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) UNKNOWN PID: 0x%llX (on-%s [%s])", inMetaValues->processId,
			//	(inFixedValues->layerId == FWPS_LAYER_ALE_CONNECT_REDIRECT_V4 || inFixedValues->layerId == FWPS_LAYER_ALE_CONNECT_REDIRECT_V6) ? "CONNECT" : "BIND",
			//	(inFixedValues->layerId == FWPS_LAYER_ALE_CONNECT_REDIRECT_V6 || inFixedValues->layerId == FWPS_LAYER_ALE_BIND_REDIRECT_V6)? "IPv6" : "IPv4");
			return;
		}

		TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) 0x%llX (on-%s [%s])", inMetaValues->processId,
			(inFixedValues->layerId == FWPS_LAYER_ALE_CONNECT_REDIRECT_V4 || inFixedValues->layerId == FWPS_LAYER_ALE_CONNECT_REDIRECT_V6) ? "CONNECT" : "BIND",
			(inFixedValues->layerId == FWPS_LAYER_ALE_CONNECT_REDIRECT_V6 || inFixedValues->layerId == FWPS_LAYER_ALE_BIND_REDIRECT_V6) ? "IPv6" : "IPv4");
				
		const IPAddrConfig config = cfg::GetIPs();
		
		switch (inFixedValues->layerId)
		{
			case FWPS_LAYER_ALE_BIND_REDIRECT_V4:
			case FWPS_LAYER_ALE_CONNECT_REDIRECT_V4:
			{
				if (!cfg::IsConfigIPv4AddrOk(config))
				{
					TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) IPv4 configuration unspecified. SKIPPING.");
					break;
				}

				const auto rawLocalAddr = RtlUlongByteSwap(inFixedValues->incomingValue[
					FWPS_FIELD_ALE_CONNECT_REDIRECT_V4_IP_LOCAL_ADDRESS].value.uint32);
				const auto rawRemoteAddr = RtlUlongByteSwap(inFixedValues->incomingValue[
					FWPS_FIELD_ALE_CONNECT_REDIRECT_V4_IP_REMOTE_ADDRESS].value.uint32);

				auto srcAddr = reinterpret_cast<const IN_ADDR*>(&rawLocalAddr);
				auto dstAddr = reinterpret_cast<const IN_ADDR*>(&rawRemoteAddr);
				
				bool isSrcTun = srcAddr->S_un.S_addr == config.IPv4Tunnel.S_un.S_addr;
				bool isConnect = inFixedValues->layerId == FWPS_LAYER_ALE_CONNECT_REDIRECT_V4;

				if (isConnect)
				{	// CONNECT
					TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) KNOWN PID: 0x%llX (IPv4-CONNECT) src:%d.%d.%d.%d dst:%d.%d.%d.%d",
						inMetaValues->processId,
						srcAddr->S_un.S_un_b.s_b1, srcAddr->S_un.S_un_b.s_b2, srcAddr->S_un.S_un_b.s_b3, srcAddr->S_un.S_un_b.s_b4,
						dstAddr->S_un.S_un_b.s_b1, dstAddr->S_un.S_un_b.s_b2, dstAddr->S_un.S_un_b.s_b3, dstAddr->S_un.S_un_b.s_b4
					);

					if (!isSrcTun) 
					{
						TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) Connect SKIPPING (source interface is not tunnel)");
						break;
					}
				}
				else 
				{	// BIND
					TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) KNOWN PID: 0x%llX (IPv4-BIND) src:%d.%d.%d.%d",
						inMetaValues->processId, srcAddr->S_un.S_un_b.s_b1, srcAddr->S_un.S_un_b.s_b2, srcAddr->S_un.S_un_b.s_b3, srcAddr->S_un.S_un_b.s_b4
					);

					bool isSrcNull = IN4_IS_ADDR_UNSPECIFIED(srcAddr);					
					if (!(isSrcNull || isSrcTun))
					{
						TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) Bind SKIPPING: isSrcTun=%d isSrcNull=%d", isSrcTun, isSrcNull);
						break;
					}
				}
				
				UINT64 classifyHandle = 0;
				auto status = FwpsAcquireClassifyHandle0 (
					const_cast<void*>(classifyContext), 
					0, 
					&classifyHandle);

				if (!NT_SUCCESS(status))
				{
					TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) FwpsAcquireClassifyHandle0() failed  %!STATUS!", status);
					break;
				}

				FWPS_CONNECT_REQUEST0* request = NULL;
				status = FwpsAcquireWritableLayerDataPointer0
				(
					classifyHandle,
					filter->filterId,
					0,
					(PVOID*)&request,
					classifyOut
				);

				if (!NT_SUCCESS(status))
				{
					TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) FwpsAcquireWritableLayerDataPointer0() failed  %!STATUS!", status);
					break;
				}
					
				auto localDetails = (SOCKADDR_IN*)&request->localAddressAndPort;
				
				// changing local address
				localDetails->sin_addr = config.IPv4Public;
				
				// apply changes 
				classifyOut->actionType = FWP_ACTION_PERMIT;
				classifyOut->rights &= ~FWPS_RIGHT_ACTION_WRITE;
				FwpsApplyModifiedLayerData0(classifyHandle, request, 0);
				FwpsReleaseClassifyHandle0(classifyHandle);

				TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) REDIRECTED PID: 0x%llX (%s [IPv4])", inMetaValues->processId,	isConnect ? "CONNECT" : "BIND");

				break;
			}

			case FWPS_LAYER_ALE_BIND_REDIRECT_V6:
			case FWPS_LAYER_ALE_CONNECT_REDIRECT_V6:
			{
				static const IN6_ADDR IN6_ADDR_ZERO = { 0 };
				if (!cfg::IsConfigIPv6AddrOk(config))
				{
					TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) IPv6 configuration unspecified. SKIPPING.");
					break;
				}

				auto srcAddr = reinterpret_cast<const IN6_ADDR*>(inFixedValues->incomingValue[
					FWPS_FIELD_ALE_CONNECT_REDIRECT_V6_IP_LOCAL_ADDRESS].value.byteArray16);
				auto dstAddr = reinterpret_cast<const IN6_ADDR*>(inFixedValues->incomingValue[
					FWPS_FIELD_ALE_CONNECT_REDIRECT_V6_IP_REMOTE_ADDRESS].value.byteArray16);

				bool isSrcTun = IN6_ADDR_EQUAL(srcAddr, &config.IPv6Tunnel);
				bool isConnect = inFixedValues->layerId == FWPS_LAYER_ALE_CONNECT_REDIRECT_V6;

				if (isConnect)
				{ // CONNECT
					TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) KNOWN PID: 0x%llX (IPv6-CONNECT) src:%x:%x:%x:%x:%x:%x:%x:%x dst:%x:%x:%x:%x:%x:%x:%x:%x",
						inMetaValues->processId,
						srcAddr->u.Word[0], srcAddr->u.Word[1], srcAddr->u.Word[2], srcAddr->u.Word[3],
						srcAddr->u.Word[4], srcAddr->u.Word[5], srcAddr->u.Word[6], srcAddr->u.Word[7],
						dstAddr->u.Word[0], dstAddr->u.Word[1], dstAddr->u.Word[2], dstAddr->u.Word[3],
						dstAddr->u.Word[4], dstAddr->u.Word[5], dstAddr->u.Word[6], dstAddr->u.Word[7]);

					if (!isSrcTun)
					{
						TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) Connect SKIPPING (source interface is not tunnel)");
						break;
					}
				}
				else
				{ // BIND
					TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) KNOWN PID: 0x%llX (IPv6-BIND) src:%x:%x:%x:%x:%x:%x:%x:%x",
						inMetaValues->processId,
						srcAddr->u.Word[0], srcAddr->u.Word[1], srcAddr->u.Word[2], srcAddr->u.Word[3],
						srcAddr->u.Word[4], srcAddr->u.Word[5], srcAddr->u.Word[6], srcAddr->u.Word[7]);

					bool isSrcNull = IN6_ADDR_EQUAL(srcAddr, &IN6_ADDR_ZERO);
					if (!(isSrcNull || isSrcTun))
					{
						TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) Bind (IPv6) SKIPPING: isSrcTun=%d isSrcNull=%d", isSrcTun, isSrcNull);
						break;
					}
				}

				UINT64 classifyHandle = 0;
				auto status = FwpsAcquireClassifyHandle0(
					const_cast<void*>(classifyContext),
					0,
					&classifyHandle);

				if (!NT_SUCCESS(status))
				{
					TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) FwpsAcquireClassifyHandle0() failed  %!STATUS!", status);
					break;
				}

				FWPS_CONNECT_REQUEST0* request = NULL;
				status = FwpsAcquireWritableLayerDataPointer0
				(
					classifyHandle,
					filter->filterId,
					0,
					(PVOID*)&request,
					classifyOut
				);

				if (!NT_SUCCESS(status))
				{
					TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) FwpsAcquireWritableLayerDataPointer0() failed  %!STATUS!", status);
					break;
				}

				auto localDetails = (SOCKADDR_IN6*)&request->localAddressAndPort;
				localDetails->sin6_addr = config.IPv6Public;
				
				// apply changes 
				classifyOut->actionType = FWP_ACTION_PERMIT;
				classifyOut->rights &= ~FWPS_RIGHT_ACTION_WRITE;
				FwpsApplyModifiedLayerData0(classifyHandle, request, 0);
				FwpsReleaseClassifyHandle0(classifyHandle);

				TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) REDIRECTED PID: 0x%llX (%s [IPv6])", inMetaValues->processId,	isConnect ? "CONNECT" : "BIND");
				
				break;
			}
			default:
			{
				TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) UNSUPPORTED LAYER ID = %d!", inFixedValues->layerId);
				break;
			}
		}
	}

	void CalloutClassifyAuthConnectOrRecv
	(
		_In_ const FWPS_INCOMING_VALUES0* inFixedValues,
		_In_ const FWPS_INCOMING_METADATA_VALUES0* inMetaValues,
		_Inout_opt_ void* layerData,
		_In_opt_ const void* classifyContext,
		_In_ const FWPS_FILTER1* filter,
		_In_ UINT64 flowContext,
		_Inout_ FWPS_CLASSIFY_OUT0* classifyOut
	)
	{
		// NOTE: The callout can be used by external applications 
		// in order to allow all communications for applications which have to be splitted
		// (e.g. it is in use by IVPN firewall to bypass its default blocking rule)
		// 
		// The callout have to be added by external applications (FwpmCalloutAdd0(...))
		// and should be referenced in an appropriate filter (with type FWP_ACTION_CALLOUT_UNKNOWN)

		DEBUG_PrintElapsedTimeEx(20);

		UNREFERENCED_PARAMETER(classifyContext);
		UNREFERENCED_PARAMETER(filter);
		UNREFERENCED_PARAMETER(layerData);
		UNREFERENCED_PARAMETER(flowContext);

		NT_ASSERT(inFixedValues);
		NT_ASSERT(inMetaValues);
		NT_ASSERT(classifyOut);
		NT_ASSERT(
			inFixedValues->layerId == FWPS_LAYER_ALE_AUTH_CONNECT_V4 ||
			inFixedValues->layerId == FWPS_LAYER_ALE_AUTH_CONNECT_V6 ||
			inFixedValues->layerId == FWPS_LAYER_ALE_AUTH_RECV_ACCEPT_V4 ||
			inFixedValues->layerId == FWPS_LAYER_ALE_AUTH_RECV_ACCEPT_V6);
				
		if (classifyOut->actionType == FWP_ACTION_NONE && classifyOut->rights & FWPS_RIGHT_ACTION_WRITE)
			classifyOut->actionType = FWP_ACTION_CONTINUE;
		
		if (!FWPS_IS_METADATA_FIELD_PRESENT(inMetaValues, FWPS_METADATA_FIELD_PROCESS_ID))
		{
			TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) SKIPPING: Failed to classify connection because PID was not provided");
			// TODO: what we should do for the connections when we can not determine PID?
			return;
		}		

		// Checking: is it 'known' process
		// ProcessMonitor keep information only about processes that have to be applied to split-tunnel
		if (NULL == prc::FindProcessInfoForPid((HANDLE)inMetaValues->processId))
		{
			// PID unknown. Do nothing. Just go to the next filter.
			return;
		}
		
		const bool isIPv6 = inFixedValues->layerId == FWPS_LAYER_ALE_AUTH_CONNECT_V6 || inFixedValues->layerId == FWPS_LAYER_ALE_AUTH_RECV_ACCEPT_V6;

		// by default - block connection
		classifyOut->actionType = FWP_ACTION_BLOCK;
		if (!(classifyOut->rights & FWPS_RIGHT_ACTION_WRITE))
			TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) SKIPPING: FWPS_RIGHT_ACTION_WRITE not set (pid=0x%llX %s)", inMetaValues->processId, (isIPv6) ? "IPv6" : "IPv4");
		else
		{
			//	classifyOut->actionType: specifies the suggested action to be taken as determined by the callout.
			//	If the FWPS_RIGHT_ACTION_WRITE flag is not set, a callout driver should not write to this member 
			//	unless it is vetoing an FWP_ACTION_PERMIT action that was previously returned by a higher weight filter in the filter engine.
			// 
			// Traffic can be blocked by a callout filter via a Veto even if a higher priority filter has permitted it.
			// (https://docs.microsoft.com/en-us/windows/win32/fwp/filter-arbitration)
			// 
			// A Veto is a "Block" action returned by the filter when the FWPS_RIGHT_ACTION_WRITE flag was reset prior to calling the filter.
			// A Veto will block traffic that was permitted with a hard permit.

			if (isIPv6)
			{
				// Permit if IPv6 is configured
				if (cfg::IsConfigIPv6PublicAddrOk())
					classifyOut->actionType = FWP_ACTION_PERMIT;
			}
			else
			{
				// Permit if IPv4 is configured
				if (cfg::IsConfigIPv4PublicAddrOk())
					classifyOut->actionType = FWP_ACTION_PERMIT;
			}
			// apply changes		
			classifyOut->rights &= ~FWPS_RIGHT_ACTION_WRITE;

			TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) 0x%llX %s (on-%s [%s])", inMetaValues->processId,
				(classifyOut->actionType == FWP_ACTION_PERMIT) ? "PERMIT" : "BLOCK",
				(inFixedValues->layerId == FWPS_LAYER_ALE_AUTH_CONNECT_V4 || inFixedValues->layerId == FWPS_LAYER_ALE_AUTH_CONNECT_V6) ? "AUTH_CONNECT" : "AUTH_RECV_ACCEPT",
				(isIPv6) ? "IPv6" : "IPv4");
		}
	}

	/// <summary>
	/// The filter engine calls this function to notify the callout driver about events 
	/// that are associated with the callout.
	/// </summary>
	NTSTATUS OnCalloutNotify
		(
			FWPS_CALLOUT_NOTIFY_TYPE notifyType,
			const GUID* filterKey,
			FWPS_FILTER1* filter
		)
	{	
		UNREFERENCED_PARAMETER(notifyType);
		UNREFERENCED_PARAMETER(filterKey);
		UNREFERENCED_PARAMETER(filter);

		return STATUS_SUCCESS;
	}

	/// <summary>
	/// RegisterCallout
	/// </summary>
	NTSTATUS RegisterCallout
	(
		PDEVICE_OBJECT wdfDevObject,
		FWPS_CALLOUT_CLASSIFY_FN1 calloutClassifyFunc,
		const GUID* calloutKey
	)
	{
		FWPS_CALLOUT1 sCallout = { 0 };

		sCallout.calloutKey = *calloutKey;
		sCallout.classifyFn = calloutClassifyFunc;
		sCallout.notifyFn = OnCalloutNotify;
		sCallout.flowDeleteFn = NULL;

		return FwpsCalloutRegister1(wdfDevObject, &sCallout, NULL);
	}

	/// <summary>
	/// AddAndRegisterCallout
	/// </summary>
	NTSTATUS AddAndRegisterCallout
	(
		PDEVICE_OBJECT wdfDevObject,
		HANDLE wfpEngineHandle,
		FWPS_CALLOUT_CLASSIFY_FN1 calloutClassifyFunc,
		const GUID* calloutKey,
		const GUID* applicableLayerKey,
		const wchar_t* calloutName,
		const wchar_t* calloutDescription
	)
	{
		FWPM_CALLOUT0 mCallout = {0};

		mCallout.calloutKey = *calloutKey;
		mCallout.displayData.name = const_cast<wchar_t*>(calloutName);
		mCallout.displayData.description = const_cast<wchar_t*>((calloutDescription!=NULL)? calloutDescription : calloutName);
		mCallout.providerKey = const_cast<GUID*>(&KEY_IVPN_ST_PROVIDER);
		mCallout.applicableLayer = *applicableLayerKey;

		auto status = FwpmCalloutAdd0(wfpEngineHandle, &mCallout, NULL, NULL);
		if (!NT_SUCCESS(status))
		{
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) FwpmCalloutAdd0() failed for '%ls':  %!STATUS!", calloutName, status);
			return status;
		}

		status = RegisterCallout(wdfDevObject, calloutClassifyFunc, calloutKey);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) RegisterCallout() failed for '%ls':  %!STATUS!", calloutName, status);

		return status;
	}

	/// <summary>
	/// UnregisterCallout
	/// </summary>
	NTSTATUS UnregisterCallout(const GUID* calloutKey)
	{
		return FwpsCalloutUnregisterByKey0(calloutKey);
	}

	NTSTATUS RegisterCallouts
	(
		PDEVICE_OBJECT wdfDevObject,
		HANDLE wfpEngineHandle
	)
	{
		//
		// REDIRECTING LAYERS
		// 
		// FWPM_LAYER_ALE_BIND_REDIRECT_V4
		// FWPM_LAYER_ALE_BIND_REDIRECT_V6
		// FWPM_LAYER_ALE_CONNECT_REDIRECT_V4,
		// FWPM_LAYER_ALE_CONNECT_REDIRECT_V6
		
		// IPv4
		auto status = AddAndRegisterCallout(wdfDevObject, wfpEngineHandle,
			CalloutClassifyConnectOrBindRedirect,
			&KEY_CALLOUT_ALE_BIND_REDIRECT_V4,	&FWPM_LAYER_ALE_BIND_REDIRECT_V4,
			L"IVPN Callout for split tunnelling (BIND_REDIRECT_V4)", NULL);
		if (!NT_SUCCESS(status))
			return status;

		status = AddAndRegisterCallout(wdfDevObject,	wfpEngineHandle,
			CalloutClassifyConnectOrBindRedirect,
			&KEY_CALLOUT_ALE_CONNECT_REDIRECT_V4,
			&FWPM_LAYER_ALE_CONNECT_REDIRECT_V4,
			L"IVPN Callout for split tunnelling (CONNECT_REDIRECT_V4)", NULL);
		if (!NT_SUCCESS(status))
			return status;

		// IPv6
		status = AddAndRegisterCallout(wdfDevObject, wfpEngineHandle,
			CalloutClassifyConnectOrBindRedirect,
			&KEY_CALLOUT_ALE_BIND_REDIRECT_V6,
			&FWPM_LAYER_ALE_BIND_REDIRECT_V6,
			L"IVPN Callout for split tunnelling (BIND_REDIRECT_V6)", NULL);
		if (!NT_SUCCESS(status))
			return status;

		status = AddAndRegisterCallout(wdfDevObject, wfpEngineHandle,
			CalloutClassifyConnectOrBindRedirect,
			&KEY_CALLOUT_ALE_CONNECT_REDIRECT_V6,
			&FWPM_LAYER_ALE_CONNECT_REDIRECT_V6,
			L"IVPN Callout for split tunnelling (CONNECT_REDIRECT_V6)", NULL);
		if (!NT_SUCCESS(status))
			return status;

		// permit\block callouts:
		// 
		// FWPM_LAYER_ALE_AUTH_CONNECT_V4
		// FWPM_LAYER_ALE_AUTH_CONNECT_V6
		// FWPM_LAYER_ALE_AUTH_RECV_ACCEPT_V4
		// FWPM_LAYER_ALE_AUTH_RECV_ACCEPT_V6

		status = AddAndRegisterCallout(wdfDevObject, wfpEngineHandle,
			CalloutClassifyAuthConnectOrRecv,
			&KEY_CALLOUT_ALE_AUTH_CONNECT_V4, &FWPM_LAYER_ALE_AUTH_CONNECT_V4,
			L"IVPN Callout for split tunnelling (ALE_AUTH_CONNECT_V4)", NULL);
		if (!NT_SUCCESS(status))
			return status;

		status = AddAndRegisterCallout(wdfDevObject, wfpEngineHandle,
			CalloutClassifyAuthConnectOrRecv,
			&KEY_CALLOUT_ALE_AUTH_CONNECT_V6, &FWPM_LAYER_ALE_AUTH_CONNECT_V6,
			L"IVPN Callout for split tunnelling (ALE_AUTH_CONNECT_V6)", NULL);
		if (!NT_SUCCESS(status))
			return status;

		status = AddAndRegisterCallout(wdfDevObject, wfpEngineHandle,
			CalloutClassifyAuthConnectOrRecv,
			&KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT, &FWPM_LAYER_ALE_AUTH_RECV_ACCEPT_V4,
			L"IVPN Callout for split tunnelling (ALE_AUTH_RECV_ACCEPT_V4)", NULL);
		if (!NT_SUCCESS(status))
			return status;

		status = AddAndRegisterCallout(wdfDevObject, wfpEngineHandle,
			CalloutClassifyAuthConnectOrRecv,
			&KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT_V6, &FWPM_LAYER_ALE_AUTH_RECV_ACCEPT_V6,
			L"IVPN Callout for split tunnelling (ALE_AUTH_RECV_ACCEPT_V6)", NULL);
		if (!NT_SUCCESS(status))
			return status;
				
		return status;
	}

	NTSTATUS UnRegisterCallouts(void)
	{
		NTSTATUS ret = STATUS_SUCCESS;
		NTSTATUS s = STATUS_SUCCESS;

		//
		// REDIRECTING LAYERS
		// 
		s = UnregisterCallout(&KEY_CALLOUT_ALE_BIND_REDIRECT_V4);
		if (!NT_SUCCESS(s))
		{
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) UnregisterCallout failed 'KEY_CALLOUT_ALE_BIND_REDIRECT_V4':  %!STATUS!", s);
			ret = s;
		}
		s = UnregisterCallout(&KEY_CALLOUT_ALE_CONNECT_REDIRECT_V4);
		if (!NT_SUCCESS(s))
		{
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) UnregisterCallout failed 'KEY_CALLOUT_ALE_CONNECT_REDIRECT_V4':  %!STATUS!", s);
			ret = s;
		}
		s = UnregisterCallout(&KEY_CALLOUT_ALE_BIND_REDIRECT_V6);
		if (!NT_SUCCESS(s))
		{
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) UnregisterCallout failed 'KEY_CALLOUT_ALE_BIND_REDIRECT_V6':  %!STATUS!", s);
			ret = s;
		}
		s = UnregisterCallout(&KEY_CALLOUT_ALE_CONNECT_REDIRECT_V6);
		if (!NT_SUCCESS(s))
		{
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) UnregisterCallout failed 'KEY_CALLOUT_ALE_CONNECT_REDIRECT_V6':  %!STATUS!", s);
			ret = s;
		}

		//
		// permit\block callouts: ALE AUTH LAYERS	// 
		s = UnregisterCallout(&KEY_CALLOUT_ALE_AUTH_CONNECT_V4);
		if (!NT_SUCCESS(s))
		{
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) UnregisterCallout failed 'KEY_CALLOUT_ALE_AUTH_CONNECT_V4':  %!STATUS!", s);
			ret = s;
		}
		s = UnregisterCallout(&KEY_CALLOUT_ALE_AUTH_CONNECT_V6);
		if (!NT_SUCCESS(s))
		{
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) UnregisterCallout failed 'KEY_CALLOUT_ALE_AUTH_CONNECT_V6':  %!STATUS!", s);
			ret = s;
		}
		s = UnregisterCallout(&KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT);
		if (!NT_SUCCESS(s))
		{
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) UnregisterCallout failed 'KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT':  %!STATUS!", s);
			ret = s;
		}
		s = UnregisterCallout(&KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT_V6);
		if (!NT_SUCCESS(s))
		{
			TraceEvents(TRACE_LEVEL_WARNING, TRACE_DRIVER, "(%!FUNC!) UnregisterCallout failed 'KEY_CALLOUT_ALE_AUTH_RECV_ACCEPT_V6':  %!STATUS!", s);
			ret = s;
		}
				
		return ret;
	}
}
