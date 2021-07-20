#include "Firewall.h"
#include "Firewall.tmh"

namespace wfp
{
	HANDLE gWfpEngineHandle = NULL;

	NTSTATUS Start(WDFDEVICE wdfDevice) 
	{
		if (IsRunning())
			return STATUS_SUCCESS;

		// Start WFP session
		
		FWPM_SESSION0 sessionInfo = { 0 };
		sessionInfo.flags = FWPM_SESSION_FLAG_DYNAMIC;

		NTSTATUS status = FwpmEngineOpen0(NULL, RPC_C_AUTHN_DEFAULT, NULL, &sessionInfo, &gWfpEngineHandle);

		if (!NT_SUCCESS(status))
		{
			gWfpEngineHandle = NULL;
			return status;
		}

		// TODO: do we need transaction start here? ( status = FwpmTransactionBegin0(gEngineHandle, 0); ...)

		// --------------------------------------------
		// Provider + layoults
		// --------------------------------------------

		FWPM_PROVIDER0 provider = { 0 };
		provider.providerKey = KEY_IVPN_ST_PROVIDER;
		provider.displayData.name = const_cast<wchar_t*>(L"IVPN Split Tunnel");
		provider.displayData.description = const_cast<wchar_t*>(L"IVPN Split Tunnel filters + callouts");

		status = FwpmProviderAdd0(gWfpEngineHandle, &provider, NULL);
		if (!NT_SUCCESS(status))
		{
			return status;
		}

		FWPM_SUBLAYER subLayer;
		RtlZeroMemory(&subLayer, sizeof(FWPM_SUBLAYER));

		subLayer.subLayerKey = KEY_IVPN_ST_SUBLAYER;
		subLayer.displayData.name = L"IVPN Split Tunnel sub-Layer";
		subLayer.displayData.description = L"IVPN Split Tunnel sub-Layer for use callouts";
		subLayer.flags = 0;
		subLayer.weight = FWP_EMPTY; // auto-weight;

		status = FwpmSubLayerAdd(gWfpEngineHandle, &subLayer, NULL);
		if (!NT_SUCCESS(status))
		{
			return status;
		}

		// --------------------------------------------
		// callouts
		// --------------------------------------------
		
		PDEVICE_OBJECT wdfDevObject = WdfDeviceWdmGetDeviceObject(wdfDevice);

		UNREFERENCED_PARAMETER(wdfDevObject);
		
		status = RegisterCallouts(wdfDevObject, gWfpEngineHandle);
		if (!NT_SUCCESS(status)) 
		{
			Stop();
			return status;
		}
		
		// --------------------------------------------
		// filters
		// --------------------------------------------
		status = RegisterFilters(gWfpEngineHandle);
		if (!NT_SUCCESS(status))
		{
			Stop();
			return status;
		}
				
		TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) Splitting started");
		return status;
	}

	NTSTATUS Stop()
	{
		if (gWfpEngineHandle == NULL)
			return STATUS_SUCCESS;

		NTSTATUS ret = STATUS_SUCCESS, status;
		
		status = UnRegisterFilters(gWfpEngineHandle);
		if (!NT_SUCCESS(status) && (NT_SUCCESS(ret)))
			ret = status;

		status = UnRegisterCallouts();
		if (!NT_SUCCESS(status) && (NT_SUCCESS(ret)))			
			ret = status;

		status = FwpmEngineClose0(gWfpEngineHandle);
		if (!NT_SUCCESS(status) && (NT_SUCCESS(ret)))			
			ret = status;
				
		gWfpEngineHandle = NULL;

		TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) Splitting stopped (%!STATUS!)", ret);
		return ret;
	}

	bool		IsRunning()
	{
		return gWfpEngineHandle!=NULL;
	}
}