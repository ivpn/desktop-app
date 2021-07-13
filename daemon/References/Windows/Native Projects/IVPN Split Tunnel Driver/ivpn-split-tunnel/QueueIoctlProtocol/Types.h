#pragma once

#ifdef THE_DRIVER_PROJECT
#include "../Driver.h"
#endif

#include <inaddr.h>
#include <in6addr.h>

typedef struct _IPAddrConfig 
{
	IN_ADDR IPv4Public;
	IN_ADDR IPv4Tunnel;

	IN6_ADDR IPv6Public;
	IN6_ADDR IPv6Tunnel;
} IPAddrConfig;

typedef struct _DriverStatus
{
	unsigned char IsEnabledProcessMonitor;
	unsigned char IsEnabledSplitting;
	unsigned char IsConfigOk;
} DriverStatus;