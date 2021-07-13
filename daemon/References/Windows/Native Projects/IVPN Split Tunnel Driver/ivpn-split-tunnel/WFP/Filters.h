#pragma once

#include "../Driver.h"
#include "../trace.h"

#include "Headers.h"
#include "Keys.h"

#include <wdm.h>

namespace wfp
{
	NTSTATUS RegisterFilterBindRedirectIpv4(HANDLE wfpEngineHandle);
	NTSTATUS UnRegisterFilterBindRedirectIpv4(HANDLE wfpEngineHandle);

	NTSTATUS RegisterFilterConnectRedirectIpv4(HANDLE wfpEngineHandle);
	NTSTATUS UnRegisterFilterConnectRedirectIpv4(HANDLE wfpEngineHandle);

	NTSTATUS RegisterFilterBindRedirectIpv6(HANDLE wfpEngineHandle);
	NTSTATUS UnRegisterFilterBindRedirectIpv6(HANDLE wfpEngineHandle);

	NTSTATUS RegisterFilterConnectRedirectIpv6(HANDLE wfpEngineHandle);
	NTSTATUS UnRegisterFilterConnectRedirectIpv6(HANDLE wfpEngineHandle);
}

