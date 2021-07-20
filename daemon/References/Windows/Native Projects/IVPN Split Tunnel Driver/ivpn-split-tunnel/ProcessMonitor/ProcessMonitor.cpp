#include "ProcessMonitor.h"
#include "ProcessMonitor.tmh"

#include "ProcessTree.h"
#include "../Config/GlobalConfig.h"

namespace prc
{
	bool _isRunning = false;

void OnCreateProcessNotify(
		PEPROCESS Process,
		HANDLE ProcessId,
		PPS_CREATE_NOTIFY_INFO CreateInfo)
	{
		UNREFERENCED_PARAMETER(Process);

		const UNICODE_STRING testPathToSplit1	= RTL_CONSTANT_STRING(L"\\??\\C:\\Program Files\\Mozilla Firefox\\firefox.exe");
		const UNICODE_STRING testPathToSplit2	= RTL_CONSTANT_STRING(L"\\??\\C:\\Windows\\system32\\cmd.exe");
		const UNICODE_STRING testFNNotAvail		= RTL_CONSTANT_STRING(L"(fileName not available)");
		
		DEBUG_PrintElapsedTimeEx(20);
				
		// We are keeping in the process tree only the information about processes:
		// 1. If process path equals to configuration 
		//		(available in the list of applications which has to be splitted)
		// 2. If process is an owner of configured application 
		//		We can simple check if the PPID is already available in a process tree
				
		if (CreateInfo != NULL)
		{
			bool isParent = FindProcessInfoForPid(CreateInfo->ParentProcessId)==NULL? FALSE : TRUE;
			bool isInConfiguration = false;

			if (isParent == FALSE && CreateInfo->FileOpenNameAvailable)
				isInConfiguration = cfg::GetIsImageToSplit(CreateInfo->ImageFileName);
			
			if (isParent || isInConfiguration)
			{
				auto status = AddNewProcessInfo(ProcessId, CreateInfo->ParentProcessId); // , CreateInfo->FileOpenNameAvailable ? CreateInfo->ImageFileName : NULL);
				if (status == STATUS_DUPLICATE_OBJECTID)
				{
					DeleteProcessInfoForPid(ProcessId);
					AddNewProcessInfo(ProcessId, CreateInfo->ParentProcessId);
				}

				TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) STARTED process: PID=0x%llX PPID=0x%llX PROC='%wZ' (Processes TOTAL: %lu)\n", 
					(INT_PTR)ProcessId, (INT_PTR)CreateInfo->ParentProcessId,
					(CreateInfo->FileOpenNameAvailable ? CreateInfo->ImageFileName : &testFNNotAvail),
					GetProcessCount());
			}
		}
		else
		{
			bool isDeleted = DeleteProcessInfoForPid(ProcessId);
			
			if (isDeleted)
				TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) STOPPED process: PID=0x%llX (Processes TOTAL: %lu)\n", 
					(INT_PTR)ProcessId, GetProcessCount());
		}
	}

	NTSTATUS	InitPIDs(DWORD* pidPpid, size_t countElement)
	{
		if (!_isRunning)
			return STATUS_INVALID_DEVICE_STATE;

		TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) Initialization info about currently running processes (processes: %llu)...\n", countElement);

		for (size_t i = 0; i < countElement; i++)
		{
			DWORD PID = *pidPpid++;
			DWORD PPID = *pidPpid++;
			AddNewProcessInfo((HANDLE)PID, (HANDLE)PPID);

			TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) INIT process: PID=0x%llX PPID=0x%llX' (Processes TOTAL: %lu)\n",
				(INT_PTR)PID, (INT_PTR)PPID, GetProcessCount());
		}

		return STATUS_SUCCESS;
	}

	NTSTATUS Start()
	{
		if (IsRunning())
			return STATUS_SUCCESS;

		InitProcessTree();

		// NOTE: the '/INTEGRITYCHECK' parameter should be defined in additional options of the linker
		// Otherwise PsSetCreateProcessNotifyRoutineEx() will faile with STATUS_ACCESS_DENIED	
		NTSTATUS status = PsSetCreateProcessNotifyRoutineEx(OnCreateProcessNotify, FALSE);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_FATAL, TRACE_DRIVER, "(%!FUNC!) OnCreateProcessNotify(register) failed %!STATUS!\n", status);
		else
		{
			TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) ProcessMonitor started.\n");
			_isRunning = true;
		}
		return status;
	};

	NTSTATUS Stop() 
	{
		if (!IsRunning())
			return STATUS_SUCCESS;

		NTSTATUS status = PsSetCreateProcessNotifyRoutineEx(OnCreateProcessNotify, TRUE);
		if (!NT_SUCCESS(status))
			TraceEvents(TRACE_LEVEL_FATAL, TRACE_DRIVER, "(%!FUNC!) OnCreateProcessNotify(remove) failed %!STATUS!\n", TRUE);
		else
			TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "(%!FUNC!) ProcessMonitor stopped.\n");

		UnInitProcessTree();
		
		_isRunning = false;

		return status;
	};

	bool		IsRunning()
	{
		return _isRunning;
	}
} // prc