#include <ntifs.h>

#include "Locker.h"
#include "Locker.tmh"

namespace utils
{
	NTSTATUS CreateLockerObj(LOCKER_TYPE* l)
	{
		return WdfSpinLockCreate(WDF_NO_OBJECT_ATTRIBUTES, l);
	}
} // namespace utils