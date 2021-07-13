#pragma once

#include "../Driver.h"

namespace utils
{
	NTSTATUS StringCreateCopy(
		_In_ PCUNICODE_STRING source,
		_Inout_ PUNICODE_STRING destination);

	void StringFree(_Inout_ PUNICODE_STRING str);

} // namespace utils