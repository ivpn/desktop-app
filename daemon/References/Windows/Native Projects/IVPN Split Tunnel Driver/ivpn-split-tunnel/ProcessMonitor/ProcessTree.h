#pragma once
#include <ntddk.h>

#include "../Trace.h"

#include "../Utils/Timer.h"
#include "../Utils/Strings.h"
#include "../Utils/Locker.h"

namespace prc
{
	// Information about process
	struct ProcessInfo {
		HANDLE PID;
		HANDLE PPID;
		// UNICODE_STRING ImageFileName;
	};

	NTSTATUS		InitProcessTree();
	NTSTATUS		UnInitProcessTree();

	NTSTATUS		DeleteAll();

	NTSTATUS		AddNewProcessInfo(HANDLE pid, HANDLE ppid); //, PCUNICODE_STRING imageFileName);

	bool			DeleteProcessInfoForPid(HANDLE pid);
	ProcessInfo*	FindProcessInfoForPid(HANDLE pid);

	ULONG			GetProcessCount();

} // prc

