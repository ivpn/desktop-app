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

	bool StartDriverAsService(LPCTSTR serviceExe);
	bool StopDriverAsService();

	/// <summary>
	/// Connects to driver
	/// </summary>
	/// <returns> false - in case of error.
	/// GetLastError() can be used for error details (only if RegisterLoggingCallback() was not used before) 
	/// </returns>
	bool Connect();
	bool Disconnect();

	/// <summary>
	/// Stop and clean everything:
	///		Stop splitting
	///		Stop processes monitoring 
	///		Clean all configuration/statuses
	/// </summary>
	bool StopAndClean();
	
	bool GetState(DriverStatus& _out_state);

	bool ConfigSetAddresses(IPAddrConfig cfg);
	bool ConfigGetAddresses(IPAddrConfig& _out_cfg);

	/// <summary>
	/// Update applications (full paths) which have to be monitored (splitted).
	/// The current configuration will remain unchanged. Will be added only new elements.
	/// </summary>
	bool ConfigSetSplitApp(std::vector<std::wstring> appPaths);
	bool ConfigGetSplitApp(std::vector<std::wstring>& retAppImages);
	/// <summary>
	/// Analogs of ConfigSetSplitApp()/ConfigGetSplitApp()
	/// Using RAW data buffer as argument.
	/// RAW data format:
	///		DWORD common size bytes
	///		DWORD strings cnt
	///		DWORD str1Len
	///		DWORD str2Len
	///		...
	///		WCHAR[] str1
	///		WCHAR[] str2
	///		...
	/// </summary>
	bool ConfigSetSplitAppRaw(unsigned char* buff, DWORD _in_buffSize);
	bool ConfigGetSplitAppRaw(unsigned char* buff, DWORD* _in_out_buffSize);
	
	bool ProcMonStart();
	bool ProcMonStop();
	/// <summary>
	/// Set application PID\PPIDs which have to be splitted.
	/// It adds new info to internal process tree but not erasing current known PID\PPIDs.
	/// Operaion fails when 'process monitor' not running
	/// </summary>
	bool ProcMonInitRunningApps();

	/// <summary>
	/// Start splitting.
	/// If "process monitor" not running - it will be started.
	///	
	/// Operation fails when configuration not complete:
	///		- no splitting apps defined
	///		- no IP configuration (IP-public + IP-tunnel) defined at least for one protocol type (IPv4\IPv6)
	/// 
	/// If only IPv4 configuration defined - splitting will work only for IPv4
	/// If only IPv6 configuration defined - splitting will work only for IPv6
	/// </summary>
	bool SplitStart();
	bool SplitStop();
}