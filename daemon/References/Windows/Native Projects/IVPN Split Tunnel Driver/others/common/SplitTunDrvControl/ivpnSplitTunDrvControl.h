#pragma once

#include <string>
#include <algorithm>
#include <vector>
#include <sstream>

#include <algorithm>    // std::sort
#include <unordered_set>

#include <ws2tcpip.h>
#include <windows.h>
#include <subauth.h>
#include <tlhelp32.h>
#include <format>

#include "../../../ivpn-split-tunnel/Public.h"
#include "../../../ivpn-split-tunnel/QueueIoctlProtocol/Types.h"

namespace splittun 
{
	typedef void (*LoggingCallback)(std::wstring logMes);
	void RegisterLoggingCallback(LoggingCallback cb);
	void UnRegisterLoggingCallback();

	/// <summary>
	/// Connects to driver
	/// </summary>
	/// <returns> false - in case of error (GetLastError() can be used for error details) </returns>
	bool Connect();
	bool Disconnect();

	bool StopAndClean();
	
	bool GetState(DriverStatus& _out_state);

	bool ConfigSetAddresses(IPAddrConfig cfg);
	bool ConfigGetAddresses(IPAddrConfig& _out_cfg);

	bool ConfigSetSplitApp(std::vector<std::wstring> appPaths);
	bool ConfigGetSplitApp(std::vector<std::wstring>& retAppImages);

	bool ProcMonStart();
	bool ProcMonStop();
	bool ProcMonInitRunningApps();

	bool SplitStart();
	bool SplitStop();
}