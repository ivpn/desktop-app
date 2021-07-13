#include <ntifs.h>

#include "Strings.h"
#include "Strings.tmh"

namespace utils
{

NTSTATUS StringCreateCopy(
	_In_ PCUNICODE_STRING source,
	_Inout_ PUNICODE_STRING destination)
{
	destination->MaximumLength = source->Length;
	destination->Length = 0;
	destination->Buffer = static_cast<PWCH>(ExAllocatePoolWithTag(NonPagedPool, source->Length, POOL_TAG));

	if (NULL == destination->Buffer)
		return STATUS_INSUFFICIENT_RESOURCES;

	RtlCopyUnicodeString(destination, source);

	return STATUS_SUCCESS;
}

void StringFree(_Inout_ PUNICODE_STRING str)
{
	if (str->Buffer != NULL)
		ExFreePoolWithTag(str->Buffer, POOL_TAG);
	str->Length = 0;
	str->MaximumLength = 0;
	str->Buffer = NULL;
}

} // namespace utils