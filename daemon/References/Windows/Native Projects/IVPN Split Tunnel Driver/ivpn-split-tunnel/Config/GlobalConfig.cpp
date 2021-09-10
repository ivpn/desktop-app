#include "GlobalConfig.h"
#include "GlobalConfig.tmh"

#include "../WFP/Headers.h"
#include "../Utils/Locker.h"
#include "../Utils/Strings.h"

namespace cfg
{
	// Using global variable to keep info about all processes
	// (it is ok because only one driver device can be initialised)

	// configuration: local interfaces
	IPAddrConfig	_cfg_IPs		= {};
	LOCKER_TYPE		_cfg_IPsLock	= NULL;
	// configuration: paths to applications binaries which have to be splitted
	RTL_AVL_TABLE	_cfg_Images		= {};
	LOCKER_TYPE		_cfg_ImagesLock = NULL;

	static const IN6_ADDR IN6_ADDR_ZERO = { 0 };
	// =====================================
	// Internal functions block
	// =====================================
	struct ImageInfo
	{
		UNICODE_STRING ImagePath;
	};

	RTL_GENERIC_COMPARE_RESULTS _compareRoutine(
		_In_ struct _RTL_AVL_TABLE* Table,
		_In_ PVOID FirstStruct,
		_In_ PVOID SecondStruct
	)
	{
		UNREFERENCED_PARAMETER(Table);
		
		LONG result = RtlCompareUnicodeString(
			&((ImageInfo*)FirstStruct)->ImagePath,
			&((ImageInfo*)SecondStruct)->ImagePath,
			true
		);
		if (result < 0) return GenericLessThan;
		if (result > 0) return GenericGreaterThan;
		return GenericEqual;
	}

	PVOID _allocateRoutine(
		_In_ struct _RTL_AVL_TABLE* Table,
		_In_ CLONG ByteSize
	)
	{
		UNREFERENCED_PARAMETER(Table);
		return ExAllocatePoolWithTag(NonPagedPool, ByteSize, POOL_TAG);
	}

	VOID _freeRoutine(
		_In_ struct _RTL_AVL_TABLE* Table,
		_In_ __drv_freesMem(Mem) _Post_invalid_ PVOID Buffer
	)
	{
		UNREFERENCED_PARAMETER(Table);
		ExFreePoolWithTag(Buffer, POOL_TAG);
	}

	// -------------------------------------
	bool _deleteImageInfo(ImageInfo* img)
	{
		UNICODE_STRING oldPath = img->ImagePath;
		const bool isDeleted = RtlDeleteElementGenericTableAvl(&_cfg_Images, img);
		if (isDeleted == TRUE)
			utils::StringFree(&oldPath);
		return isDeleted;
	}

	NTSTATUS _deleteAllImageInfo()
	{
		for (;;)
		{
			auto ii = (ImageInfo*)RtlGetElementGenericTableAvl(&_cfg_Images, 0);
			if (ii == NULL)
				break;

			if (false == _deleteImageInfo(ii))
			{
				TraceEvents(TRACE_LEVEL_CRITICAL, TRACE_DRIVER, "%!FUNC! Unable to erase configuration tree. Entry not deleted %wZ\n", &ii->ImagePath);
				return STATUS_DATA_ERROR;
			}
		}

		TraceEvents(TRACE_LEVEL_INFORMATION, TRACE_DRIVER, "%!FUNC! Configuration erased: images to split\n");

		return STATUS_SUCCESS;
	}

	// =====================================
	// Public functions block
	// =====================================

	NTSTATUS Init()
	{
		_cfg_IPs = {};

		NTSTATUS status;

		if (_cfg_IPsLock == NULL)
		{
			status = utils::CreateLockerObj(&_cfg_IPsLock);
			if (!NT_SUCCESS(status))
			{
				TraceEvents(TRACE_LEVEL_CRITICAL, TRACE_DRIVER, "%!FUNC! Unable to initialise locker. Error: %!STATUS!\n", status);
				return status;
			}
		}

		if (_cfg_ImagesLock == NULL)
		{
			status = utils::CreateLockerObj(&_cfg_ImagesLock);
			if (!NT_SUCCESS(status))
			{
				TraceEvents(TRACE_LEVEL_CRITICAL, TRACE_DRIVER, "%!FUNC! Unable to initialise locker. Error: %!STATUS!\n", status);
				return status;
			}
			// initialize tree to collect images
			RtlInitializeGenericTableAvl(&_cfg_Images, _compareRoutine, _allocateRoutine, _freeRoutine, NULL);
		}
		TraceEvents(TRACE_LEVEL_INFORMATION , TRACE_DRIVER, "%!FUNC! Configuration Initialized.\n");

		return STATUS_SUCCESS;
	}

	NTSTATUS Clean()
	{
		utils::Locker la(_cfg_IPsLock);
		utils::Locker li(_cfg_ImagesLock);

		_cfg_IPs = {};		
		return _deleteAllImageInfo();
	}

	bool IsConfigIPv4AddrOk(const IPAddrConfig& cfgIPs)
	{
		if (IN4_IS_ADDR_UNSPECIFIED(&cfgIPs.IPv4Public) || IN4_IS_ADDR_UNSPECIFIED(&cfgIPs.IPv4Tunnel))
			return false;
		return true;
	}

	bool IsConfigIPv6AddrOk(const IPAddrConfig& cfgIPs)
	{
		if (IN6_ADDR_EQUAL(&cfgIPs.IPv6Public, &IN6_ADDR_ZERO) || IN6_ADDR_EQUAL(&cfgIPs.IPv6Tunnel, &IN6_ADDR_ZERO))
			return false;
		return true;
	}
	
	bool IsConfigIPv4PublicAddrOk()
	{
		utils::Locker la(_cfg_IPsLock);
		if (IN4_IS_ADDR_UNSPECIFIED(&_cfg_IPs.IPv4Public))
			return false;
		return true;
	}

	bool IsConfigIPv6PublicAddrOk()
	{
		utils::Locker la(_cfg_IPsLock);
		if (IN6_ADDR_EQUAL(&_cfg_IPs.IPv6Public, &IN6_ADDR_ZERO))
			return false;
		return true;
	}

	bool IsConfigIPv4AddrOk()
	{
		utils::Locker la(_cfg_IPsLock);
		return IsConfigIPv4AddrOk(_cfg_IPs);
	}

	bool IsConfigIPv6AddrOk()
	{
		utils::Locker la(_cfg_IPsLock);
		return IsConfigIPv6AddrOk(_cfg_IPs);
	}

	bool IsConfigOk()
	{
		utils::Locker la(_cfg_IPsLock);
		utils::Locker li(_cfg_ImagesLock);
		
		if ((IN4_IS_ADDR_UNSPECIFIED(&_cfg_IPs.IPv4Public) || IN4_IS_ADDR_UNSPECIFIED(&_cfg_IPs.IPv4Tunnel))
			&& (IN6_ADDR_EQUAL(&_cfg_IPs.IPv6Public, &IN6_ADDR_ZERO) || IN6_ADDR_EQUAL(&_cfg_IPs.IPv6Tunnel, &IN6_ADDR_ZERO)))
			return false;

		if (RtlNumberGenericTableElementsAvl(&_cfg_Images) == 0)
			return false;

		return true;
	}

	void SetIPs(const IPAddrConfig& ips)
	{
		utils::Locker l(_cfg_IPsLock);
		_cfg_IPs = ips;
	}

	const IPAddrConfig GetIPs()
	{
		utils::Locker l(_cfg_IPsLock);
		return _cfg_IPs;
	}
	
	NTSTATUS SetImagesToSplit(const char* buff, DWORD bufSize)
	{
		utils::Locker l(_cfg_ImagesLock);

		//	DWORD common size bytes
		//	DWORD strings cnt
		//	DWORD str1Len
		//	DWORD str2Len
		//	...
		//	WCHAR[] str1 
		//	WCHAR[] str2
		//	...

		if (bufSize < sizeof(DWORD) + sizeof(DWORD) * 2 + sizeof(wchar_t) * 1)
			return STATUS_INVALID_PARAMETER; // buffer is too small

		if (*(DWORD*)buff != bufSize)
			return STATUS_INVALID_PARAMETER; // bad data

		DWORD stringsCnt = *(DWORD*)(buff + sizeof(DWORD));
		DWORD headerSize = sizeof(DWORD) + sizeof(DWORD) + sizeof(DWORD) * stringsCnt;
		const char* strPtr = buff + headerSize;

		for (DWORD i = 0; i < stringsCnt; i++)
		{
			DWORD strLen = *(DWORD*)(buff + sizeof(DWORD) + sizeof(DWORD) + sizeof(DWORD) * i);
			
			ImageInfo ii = {};
			
			ii.ImagePath.Length = (USHORT) strLen * sizeof(WCHAR);
			ii.ImagePath.MaximumLength = ii.ImagePath.Length;
			ii.ImagePath.Buffer = static_cast<PWCH>(ExAllocatePoolWithTag(NonPagedPool, ii.ImagePath.Length, POOL_TAG));
			if (ii.ImagePath.Buffer != NULL)
				RtlCopyMemory(ii.ImagePath.Buffer, strPtr, ii.ImagePath.Length);
			else
				return STATUS_INSUFFICIENT_RESOURCES;

			strPtr += strLen * sizeof(wchar_t);

			BOOLEAN isNewElement;
			ImageInfo* newEntry = (ImageInfo*)RtlInsertElementGenericTableAvl(&_cfg_Images, &ii, static_cast<CLONG>(sizeof(ii)), &isNewElement);
			if (newEntry == NULL)
			{
				utils::StringFree(&ii.ImagePath);
				return STATUS_INSUFFICIENT_RESOURCES;
			}
			// Check if tree already contain such element
			if (isNewElement == FALSE)
			{
				utils::StringFree(&ii.ImagePath);
				continue;
			}
			TraceEvents(TRACE_LEVEL_CRITICAL, TRACE_DRIVER, "%!FUNC! config: new image to split '%wZ'\n", &newEntry->ImagePath);
		}
				
		return STATUS_SUCCESS;
	}

	NTSTATUS GetImagesToSplit(char* buff_inOut, DWORD *bufSize_inOut)
	{
		utils::Locker l(_cfg_ImagesLock);

		//	DWORD common size bytes
		//	DWORD strings cnt
		//	DWORD str1Len
		//	DWORD str2Len
		//	...
		//	WCHAR[] str1 
		//	WCHAR[] str2
		//	...

		if (bufSize_inOut == NULL)
			return STATUS_INVALID_PARAMETER;

		ULONG elCntUL = RtlNumberGenericTableElementsAvl(&_cfg_Images);
		if (elCntUL > 0xffffffff)
		{
			*bufSize_inOut = (DWORD)0;
			return STATUS_INTERNAL_ERROR;
		}
		DWORD elementsCnt = elCntUL;
		
		size_t buffSize = sizeof(DWORD) +sizeof(DWORD) * ((size_t)1 + elementsCnt);
		DWORD headerOffset = (DWORD)buffSize;

		for (ImageInfo* p = (ImageInfo*)RtlEnumerateGenericTableAvl(&_cfg_Images, TRUE); p != NULL; p = (ImageInfo*)RtlEnumerateGenericTableAvl(&_cfg_Images, FALSE))
			buffSize += p->ImagePath.Length;

		if (*bufSize_inOut < buffSize)
		{
			*bufSize_inOut = (DWORD)buffSize;
			return STATUS_BUFFER_TOO_SMALL;
		}
		*bufSize_inOut = (DWORD)buffSize;
		
		if (buff_inOut == NULL)
			return STATUS_INVALID_PARAMETER;

		*(DWORD*)buff_inOut = (DWORD)buffSize;
		*(DWORD*)(buff_inOut + sizeof(DWORD)) = (DWORD)elementsCnt;

		char* sptr = buff_inOut + headerOffset;
		DWORD i = 0;
		for (ImageInfo* p = (ImageInfo*)RtlEnumerateGenericTableAvl(&_cfg_Images, TRUE); p != NULL; i++, p = (ImageInfo*)RtlEnumerateGenericTableAvl(&_cfg_Images, FALSE))
		{
			// string size (characters count)
			*(DWORD*)(buff_inOut + sizeof(DWORD) + sizeof(DWORD) + sizeof(DWORD) * i) = (DWORD)p->ImagePath.Length/sizeof(wchar_t);

			// string data
			size_t strBSize = p->ImagePath.Length;
			memcpy(sptr, p->ImagePath.Buffer, strBSize);
			sptr += strBSize;
		}

		return STATUS_SUCCESS;
	}

	bool GetIsImageToSplit(PCUNICODE_STRING img)
	{
		utils::Locker l(_cfg_ImagesLock);
		
		ImageInfo ii = {};
		ii.ImagePath = *img;

		ImageInfo* found = (ImageInfo*)RtlLookupElementGenericTableAvl(&_cfg_Images, &ii);

		return found != NULL;
	}

	bool GetIsNoImagesToSplit()
	{
		utils::Locker l(_cfg_ImagesLock);
		return RtlNumberGenericTableElementsAvl(&_cfg_Images) == 0;
	}
}