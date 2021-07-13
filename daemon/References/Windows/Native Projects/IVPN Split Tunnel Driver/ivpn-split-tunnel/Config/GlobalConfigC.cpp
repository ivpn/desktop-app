#include "GlobalConfigC.h"
#include "GlobalConfigC.tmh"

#include "../Config/GlobalConfig.h"

NTSTATUS ConfigurationInit()
{
	return cfg::Init();
};