#include "stdafx.h"
#include "../IVPN Split Tunnel Driver/others/common/SplitTunDrvControl/ivpnSplitTunDrvControl.h"

extern "C" {
	EXPORT DWORD _cdecl  SplitTun_Connect(wchar_t* driverPath)
	{
		// If driver file is defined - start driver as a service
		// If driver file is not defined - just try to connect driver object (it could be that driver already registered in a system)
		bool isStartedAsService = false;
		if (driverPath != NULL && wcslen(driverPath) > 0)
		{
			isStartedAsService = true;
						
			if (!splittun::StartDriverAsService(driverPath))
			{
				// It could be that service already started (which is not expected for us)
				// Therefore, we stopping it. And do a clean start then.
				DWORD lastError = GetLastError();
				if (lastError == ERROR_SERVICE_EXISTS || lastError == ERROR_SERVICE_MARKED_FOR_DELETE)
				{
					splittun::StopDriverAsService();
					if (!splittun::StartDriverAsService(driverPath))
						return false;
				}
				else 
					return false;
			}
		}

		if (!splittun::Connect())
		{
			if (isStartedAsService)
				splittun::StopDriverAsService();
			return false;
		}
		return true;
	}
	EXPORT DWORD _cdecl  SplitTun_Disconnect()
	{
		bool isDisconnected = splittun::Disconnect();
		splittun::StopDriverAsService();
		return isDisconnected;
	}

	EXPORT DWORD _cdecl  SplitTun_StopAndClean()
	{
		return splittun::StopAndClean();
	}
		
	EXPORT DWORD _cdecl  SplitTun_ProcMonStart()
	{
		return splittun::ProcMonStart();
	}
	EXPORT DWORD _cdecl  SplitTun_ProcMonStop()
	{
		return splittun::ProcMonStop();
	}
	EXPORT DWORD _cdecl  SplitTun_ProcMonInitRunningApps()
	{
		return splittun::ProcMonInitRunningApps();
	}

	EXPORT DWORD _cdecl  SplitTun_SplitStart()
	{
		return splittun::SplitStart();
	}
	EXPORT DWORD _cdecl  SplitTun_SplitStop()
	{
		return splittun::SplitStop();
	}
		
	EXPORT DWORD _cdecl  SplitTun_GetState(
		DWORD* _out_IsConfigOk, 
		DWORD* _out_IsEnabledProcessMonitor,
		DWORD* _out_IsEnabledSplitting)
	{
		DriverStatus _out_state = {};
		DWORD isSuccess = splittun::GetState(_out_state);
		*_out_IsConfigOk = _out_state.IsConfigOk;
		*_out_IsEnabledProcessMonitor = _out_state.IsEnabledProcessMonitor;
		*_out_IsEnabledSplitting = _out_state.IsEnabledSplitting;
		return isSuccess;
	}
	
	EXPORT DWORD _cdecl  SplitTun_ConfigSetAddresses(
		const unsigned char* IPv4Public, // 4 bytes
		const unsigned char* IPv4Tunnel, // 4 bytes
		const unsigned char* IPv6Public, // 16 bytes
		const unsigned char* IPv6Tunnel) // 16 bytes
	{
		IPAddrConfig cfg = {};
		cfg.IPv4Public = *(IN_ADDR*)IPv4Public;
		cfg.IPv4Tunnel = *(IN_ADDR*)IPv4Tunnel;
		cfg.IPv6Public = *(IN6_ADDR*)IPv6Public;
		cfg.IPv6Tunnel = *(IN6_ADDR*)IPv6Tunnel;
		return splittun::ConfigSetAddresses(cfg);
	}
	
	EXPORT DWORD _cdecl  SplitTun_ConfigGetAddresses(
		unsigned char* IPv4Public, // 4 bytes
		unsigned char* IPv4Tunnel, // 4 bytes
		unsigned char* IPv6Public, // 16 bytes
		unsigned char* IPv6Tunnel) // 16 bytes
	{
		IPAddrConfig _out_cfg;
		DWORD ret = splittun::ConfigGetAddresses(_out_cfg);
		if (ret != 1)
			return ret;
		
		if (0 != memcpy_s(IPv4Public, 4, &_out_cfg.IPv4Public, 4)) return 0;		
		if (0 != memcpy_s(IPv4Tunnel, 4, &_out_cfg.IPv4Tunnel, 4)) return 0;		
		if (0 != memcpy_s(IPv6Public, 16, &_out_cfg.IPv6Public, 16)) return 0;
		if (0 != memcpy_s(IPv6Tunnel, 16, &_out_cfg.IPv6Tunnel, 16)) return 0;

		return ret;
	}
		
	EXPORT DWORD _cdecl  fSplitTun_ConfigSetSplitAppRaw(unsigned char* buff, DWORD _in_buffSize)
	{
		return splittun::ConfigSetSplitAppRaw(buff, _in_buffSize);
	};

	EXPORT DWORD _cdecl  fSplitTun_ConfigGetSplitAppRaw(unsigned char* buff, DWORD* _in_out_buffSize)
	{
		return splittun::ConfigGetSplitAppRaw(buff, _in_out_buffSize);
	}
}