// NativeHelpers.cpp : Defines the exported functions for the DLL application.
//

#include "stdafx.h"

extern "C" {

	EXPORT DWORD _cdecl WfpEngineOpen(const FWPM_SESSION0 *session, HANDLE *engineHandle)
	{
		DWORD result = FwpmEngineOpen0(0,
			RPC_C_AUTHN_WINNT,
			0,
			session,
			engineHandle);

		return result;
	}

	EXPORT DWORD _cdecl WfpEngineClose(HANDLE engineHandle)
	{
		return FwpmEngineClose0(engineHandle);
	}

	EXPORT FWPM_SESSION0 * _cdecl CreateWfpSessionObject(bool isDynamic)
	{
		FWPM_SESSION0 *session = new FWPM_SESSION0();
		if (isDynamic)
			session->flags = FWPM_SESSION_FLAG_DYNAMIC;

		return session;
	}

	EXPORT void _cdecl DeleteWfpSessionObject(FWPM_SESSION0 *sessionObject)
	{
		delete sessionObject;
	}

	EXPORT DWORD _cdecl WfpTransactionBegin(HANDLE engineHandle)
	{
		return FwpmTransactionBegin0(engineHandle, 0);
	}

	EXPORT DWORD _cdecl WfpTransactionCommit(HANDLE engineHandle)
	{
		return FwpmTransactionCommit0(engineHandle);			
	}

	EXPORT DWORD _cdecl WfpTransactionAbort(HANDLE engineHandle)
	{
		return FwpmTransactionAbort0(engineHandle);
	}
}

BOOL APIENTRY DllMain(HMODULE hModule,
	DWORD  ul_reason_for_call,
	LPVOID lpReserved
	)
{
	switch (ul_reason_for_call)
	{
	case DLL_PROCESS_ATTACH:
	case DLL_THREAD_ATTACH:
	case DLL_THREAD_DETACH:
	case DLL_PROCESS_DETACH:
		break;
	}
	return TRUE;
}

