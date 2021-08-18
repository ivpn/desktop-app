#pragma once

#include "../Driver.h"
#include "../trace.h"

#include "Headers.h"
#include "Keys.h"

#include <wdm.h>

namespace wfp
{
	//
	// 'Callout' filters for splitting
	//
	NTSTATUS RegisterFilters(HANDLE wfpEngineHandle);
	NTSTATUS UnRegisterFilters(HANDLE wfpEngineHandle);
}

