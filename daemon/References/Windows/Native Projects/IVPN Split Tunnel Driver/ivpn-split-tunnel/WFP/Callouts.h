#pragma once

#include "../Driver.h"
#include "../trace.h"

#include "../Utils/Timer.h"

#include "Headers.h"
#include "Keys.h"

//
// https://docs.microsoft.com/en-us/windows-hardware/drivers/network/using-bind-or-connect-redirection
//
namespace wfp
{
	NTSTATUS RegisterCallouts( PDEVICE_OBJECT wdfDevObject, HANDLE wfpEngineHandle );
	NTSTATUS UnRegisterCallouts(void);
}

