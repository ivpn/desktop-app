#pragma once

#include "../Driver.h"

namespace utils
{

#define LOCKER_TYPE WDFSPINLOCK

	NTSTATUS CreateLockerObj(LOCKER_TYPE* l);

	class Locker {
	public:
		Locker(WDFSPINLOCK l) : _l(l)
		{
			WdfSpinLockAcquire(_l);
		};

		~Locker()
		{
			WdfSpinLockRelease(_l);
		}

	private:
		WDFSPINLOCK _l;
	};	
} // namespace utils