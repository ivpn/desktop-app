#pragma once

#include "../Driver.h"
#include "../trace.h"

#include "Headers.h"

#include "Callouts.h"
#include "Filters.h"

namespace wfp
{
	NTSTATUS	Start(WDFDEVICE wdfDevice);
	NTSTATUS	Stop();
	bool		IsRunning();
}