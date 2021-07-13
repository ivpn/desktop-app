#include "ProcessTree.h"
#include "ProcessTree.tmh"

#include "../Public.h"

namespace prc
{
	// The tree of all known processes
	// Using global variable to keep info about all processes
	// (it is ok because only one driver device can be initialised)
	RTL_AVL_TABLE	_processTree = {};
	LOCKER_TYPE		_processTreeLock = NULL;
	
	// =====================================
	// Internal functions block
	// =====================================

	RTL_GENERIC_COMPARE_RESULTS _compareRoutine (
			_In_ struct _RTL_AVL_TABLE* Table,
			_In_ PVOID FirstStruct,
			_In_ PVOID SecondStruct
		)
	{
		UNREFERENCED_PARAMETER(Table);
		auto el1 = ((ProcessInfo*)FirstStruct)->PID;
		auto el2 = ((ProcessInfo*)SecondStruct)->PID;
		if (el1 < el2) return GenericLessThan;
		if (el1 > el2) return GenericGreaterThan;
		return GenericEqual;
	}

	PVOID _allocateRoutine (
			_In_ struct _RTL_AVL_TABLE* Table,
			_In_ CLONG ByteSize
		)
	{
		UNREFERENCED_PARAMETER(Table);
		return ExAllocatePoolWithTag(NonPagedPool, ByteSize, POOL_TAG);
	}

	VOID _freeRoutine (
			_In_ struct _RTL_AVL_TABLE* Table,
			_In_ __drv_freesMem(Mem) _Post_invalid_ PVOID Buffer
		)
	{
		UNREFERENCED_PARAMETER(Table);
		ExFreePoolWithTag(Buffer, POOL_TAG);
	}

	bool _deleteProcessInfo(ProcessInfo* pi)
	{
		//UNICODE_STRING oldPath = &pi->ImageFileName;
		const bool isDeleted = RtlDeleteElementGenericTableAvl(&_processTree, pi);
		//if (isDeleted == TRUE)
		//	utils::StringFree(&oldPath);
		return isDeleted;
	}

	// =====================================
	// Public functions block
	// =====================================

	NTSTATUS InitProcessTree()
	{
		if (_processTreeLock == NULL)
		{
			NTSTATUS status = utils::CreateLockerObj(&_processTreeLock);
			if (!NT_SUCCESS(status))
			{
				TraceEvents(TRACE_LEVEL_CRITICAL, TRACE_DRIVER, "%!FUNC! Unable to initialise locker. Error: %!STATUS!\n", status);
				return status;
			}

			RtlInitializeGenericTableAvl(&_processTree, _compareRoutine, _allocateRoutine, _freeRoutine, NULL);

			return STATUS_SUCCESS;
		}
		
		return DeleteAll();
	}
	
	NTSTATUS		UnInitProcessTree()
	{
		if (_processTreeLock == NULL)
			return STATUS_SUCCESS;

		auto status = DeleteAll();
		if (!NT_SUCCESS(status))
			return status;

		return status;
	}

	NTSTATUS		DeleteAll()
	{
		utils::Locker l(_processTreeLock);

		for (;;)
		{
			auto pi = (ProcessInfo*)RtlGetElementGenericTableAvl(&_processTree, 0);
			if (pi == NULL)
				break;

			if (false == _deleteProcessInfo(pi))
			{
				TraceEvents(TRACE_LEVEL_CRITICAL, TRACE_DRIVER, "%!FUNC! Unable to erase tree. Entry not deleted PID=0x%llX\n", (INT_PTR)pi->PID);
				return STATUS_DATA_ERROR;
			}
		}
		return STATUS_SUCCESS;
	}
	
	NTSTATUS AddNewProcessInfo(HANDLE pid, HANDLE ppid) //, PCUNICODE_STRING imageFileName)
	{
		// initialize new process-info object
		ProcessInfo pi = { };
		pi.PID = pid;
		pi.PPID = ppid;
		
		//if (NULL != imageFileName)
		//{
		//	NTSTATUS status = utils::StringCreateCopy(imageFileName, &pi.ImageFileName);
		//	if (STATUS_SUCCESS != status)
		//		return status;
		//}		

		// If a matching entry already exists in the generic table, 
		// RtlInsertElementGenericTableAvl returns a pointer to the existing entry's data and sets NewElement to FALSE.
		BOOLEAN isNewElement;

		// Add new entry into the tree

		PVOID newEntry;
		{	// locked block
			utils::Locker l(_processTreeLock);
			newEntry = RtlInsertElementGenericTableAvl(&_processTree, &pi, static_cast<CLONG>(sizeof(pi)), &isNewElement);
		}

		// If cannot insert the new entry (for example, because the AllocateRoutine fails), RtlInsertElementGenericTableAvl returns NULL.
		if (newEntry == NULL)
		{
			//utils::StringFree(&pi.ImageFileName);
			return STATUS_INSUFFICIENT_RESOURCES;
		}

		// Check if tree already contain such element
		if (isNewElement == FALSE)
		{
			//utils::StringFree(&pi.ImageFileName);
			return STATUS_DUPLICATE_OBJECTID;
		}

		return STATUS_SUCCESS;
	}

	bool DeleteProcessInfoForPid(HANDLE pid)
	{
		ProcessInfo* pi = FindProcessInfoForPid(pid);
		if (pi == NULL)
			return false;

		utils::Locker l(_processTreeLock);
		return _deleteProcessInfo(pi);
	}

	ProcessInfo* FindProcessInfoForPid(HANDLE pid)
	{
		ProcessInfo pi = { };
		pi.PID = pid;

		utils::Locker l(_processTreeLock);
		return (ProcessInfo*)RtlLookupElementGenericTableAvl(&_processTree, &pi);
	}

	ULONG	GetProcessCount()
	{
		utils::Locker l(_processTreeLock);
		return RtlNumberGenericTableElementsAvl(&_processTree);
	}
}