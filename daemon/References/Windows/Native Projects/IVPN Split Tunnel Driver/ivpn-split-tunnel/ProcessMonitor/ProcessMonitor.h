#pragma once
#include <ntddk.h>

#include "../Trace.h"

#include "../Utils/Timer.h"
#include <minwindef.h>

namespace prc
{	
	NTSTATUS	Start();
	NTSTATUS	Stop();
	bool		IsRunning();

	NTSTATUS	InitPIDs(DWORD* pidPpid, size_t countElement);
} // prc

