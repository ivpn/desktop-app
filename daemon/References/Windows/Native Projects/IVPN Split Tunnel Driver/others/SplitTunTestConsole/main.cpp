// test-splittun-drv.cpp : This file contains the 'main' function. Program execution begins and ends there.
//

#include <iostream>
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

#include "../../ivpn-split-tunnel/Public.h"
#include "../../ivpn-split-tunnel/QueueIoctlProtocol/Types.h"


HANDLE   hDevice = INVALID_HANDLE_VALUE;

struct ProcessInfo 
{
	DWORD			PID;
	DWORD			PPID;
	std::wstring	Path;
	FILETIME		CreationTime;
};

std::wstring GetLastErrorStr()
{
	DWORD eNum;
	WCHAR sysMsg[256];
	WCHAR* p;

	eNum = GetLastError();
	FormatMessage(FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
		NULL, eNum,
		MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT), // Default language
		sysMsg, 256, NULL);

	// Trim the end of the line and terminate it with a null
	p = sysMsg;
	while ((*p > 31) || (*p == 9))
		++p;
	do { *p-- = 0; } while ((p >= sysMsg) &&
		((*p == '.') || (*p < 33)));

	std::wostringstream stringStream;
	stringStream << eNum << " (" << sysMsg << ")";
	return stringStream.str();
}

DWORD connect() 
{
	// TODO: implement asynchronous operations 
	// (use FILE_FLAG_OVERLAPPED  attribute; the DeviceIoControl() call have to be updated too)
	hDevice = CreateFileW(DEVICE_NAME_PUBLIC,
		GENERIC_READ | GENERIC_WRITE,
		0,
		NULL,
		OPEN_EXISTING,
		FILE_ATTRIBUTE_NORMAL,
		NULL);

	if (hDevice == INVALID_HANDLE_VALUE)
	{
		DWORD    errNum = GetLastError();
		if (errNum == ERROR_FILE_NOT_FOUND)
			std::wcout << "CreateFile failed!  (ERROR_FILE_NOT_FOUND) " << errNum << std::endl;
		else if (errNum == ERROR_PATH_NOT_FOUND)
			std::wcout << "CreateFile failed!  (ERROR_PATH_NOT_FOUND) " << errNum << std::endl;
		else 
			std::wcout << "CreateFile failed!  " << errNum << std::endl;
	}
	else
		std::wcout << "Connected! (Success)" << std::endl;
	return 0;
}

void disconnect()
{
	if (hDevice != INVALID_HANDLE_VALUE)
		CloseHandle(hDevice);
	hDevice = INVALID_HANDLE_VALUE;
}

bool SendIoctl(DWORD code, 
				LPVOID lpInBuffer, DWORD nInBufferSize, 
				LPVOID lpOutBuffer, DWORD nOutBufferSize, 
				LPDWORD lpBytesReturned)
{
	BOOL bRc;

	std::wcout << " ==> " << std::hex << "0x" << code << std::dec<< " ...";

	bRc = DeviceIoControl(hDevice, code,
		lpInBuffer, nInBufferSize,
		lpOutBuffer, nOutBufferSize,
		lpBytesReturned,
		NULL
	);

	if (!bRc)
		std::wcout << " Error in DeviceIoControl: " << GetLastErrorStr() << std::endl;
	else
		std::wcout << " SUCCESS. Bytes returned: " << *lpBytesReturned << std::endl;
	return bRc;
}

void clean() 
{
	DWORD bytesRead = 0;
	SendIoctl(IOCTL_STOP_ALL_AND_CFG_CLEAN, nullptr, 0, nullptr, 0, &bytesRead);
}

void getState()
{
	DriverStatus s = {};

	DWORD bytesRead = 0;
	if (SendIoctl(IOCTL_GET_STATE, NULL, 0, &s, sizeof(s), &bytesRead))
	{
		std::wcout << "Driver state:"
			<< "\n IsConfigOk			: " << ((s.IsConfigOk)? "Yes": "No")
			<< "\n IsEnabledProcessMonitor	: " << ((s.IsEnabledProcessMonitor) ? "Yes" : "No")
			<< "\n IsEnabledSplitting		: " << ((s.IsEnabledSplitting) ? "Yes" : "No")
			<< std::endl;
	}
	else
		std::wcout << "IOCTL_GET_STATE failed" << std::endl;
}

void pmStart()
{
	DWORD bytesRead = 0;
	SendIoctl(IOCTL_PROCMON_START, nullptr, 0, nullptr, 0, &bytesRead);
}

void pmStop()
{
	DWORD bytesRead = 0;
	SendIoctl(IOCTL_PROCMON_STOP, nullptr, 0, nullptr, 0, &bytesRead);
}

void splitStart()
{
	DWORD bytesRead = 0;
	SendIoctl(IOCTL_SPLITTING_START, nullptr, 0, nullptr, 0, &bytesRead);
}
void splitStop()
{
	DWORD bytesRead = 0;
	SendIoctl(IOCTL_SPLITTING_STOP, nullptr, 0, nullptr, 0, &bytesRead);
}

void setAddresses() {
	IPAddrConfig cfg = {};

	std::wstring pubStrIPv4, tunStrIPv4, pubStrIPv6, tunStrIPv6;

	std::wcout << L" PUBLIC interface IPv4: ";
	std::getline(std::wcin, pubStrIPv4);
	std::wcout << L" TUNNEL interface IPv4: ";
	std::getline(std::wcin, tunStrIPv4);

	std::wcout << L" PUBLIC interface IPv6: ";
	std::getline(std::wcin, pubStrIPv6);
	std::wcout << L" TUNNEL interface IPv6: ";
	std::getline(std::wcin, tunStrIPv6);
	
	if (pubStrIPv4.length() <= 0 || tunStrIPv4.length() <= 0)
	{
		std::wcout << L"Error: IPv4 configuration not defined" << std::endl;
		return;
	}
	else
	{
		if (InetPtonW(AF_INET, pubStrIPv4.c_str(), &(cfg.IPv4Public)) != 1
			|| InetPtonW(AF_INET, tunStrIPv4.c_str(), &(cfg.IPv4Tunnel)) != 1)
		{
			std::wcout << L"Error: IPv4 configuration" << std::endl;
			return;
		}
	}

	if (pubStrIPv6.length() <= 0 || tunStrIPv6.length() <= 0)
		std::wcout << L"IPv6 configuration skipped" << std::endl;
	else
	{
		if (InetPtonW(AF_INET6, pubStrIPv6.c_str(), &(cfg.IPv6Public)) != 1
			|| InetPtonW(AF_INET6, tunStrIPv6.c_str(), &(cfg.IPv6Tunnel)) != 1)
		{
			std::wcout << L"Error parsing IPv6 addresses" << std::endl;
		}			
	}
	
	DWORD bytesRead = 0;
	SendIoctl(IOCTL_CFG_SET_ADDRESSES, &cfg, sizeof(cfg), nullptr, 0, &bytesRead);
}

void getAddresses() 
{
	IPAddrConfig cfg = {};

	DWORD bytesRead = 0;
	if (SendIoctl(IOCTL_CFG_GET_ADDRESSES, NULL, 0, &cfg, sizeof(cfg), &bytesRead))
	{
		auto pub = cfg.IPv4Public.S_un.S_un_b;
		auto tun = cfg.IPv4Tunnel.S_un.S_un_b;

		WCHAR ipv4Pub[16] = {0}, ipv4tun[16] = { 0 }, ipv6Pub[46] = { 0 }, ipv6tun[46] = { 0 };
		
		InetNtopW(AF_INET, &cfg.IPv4Public,	&ipv4Pub[0], 16);
		InetNtopW(AF_INET, &cfg.IPv4Tunnel,	&ipv4tun[0], 16);
		InetNtopW(AF_INET6, &cfg.IPv6Public, &ipv6Pub[0], 46);
		InetNtopW(AF_INET6, &cfg.IPv6Tunnel, &ipv6tun[0], 46);

		std::wcout << "Driver configuration:"
			<< "\n pub IPv4: " << ipv4Pub
			<< "\n tun IPv4: " << ipv4tun
			<< "\n pub IPv6: " << ipv6Pub
			<< "\n tun IPv6: " << ipv6tun
			<< std::endl;
	}
	else
		std::wcout << "IOCTL_CFG_GET_ADDRESSES failed" << std::endl;
}

char* makeSplitAppRequestData(std::vector<std::wstring> paths, size_t* createdBuffSize)
{
	//	size_t common size bytes
	//	size_t strings cnt
	//	size_t str1Len
	//	size_t str2Len
	//	...
	//	WCHAR[] str1 
	//	WCHAR[] str2
	//	...

	*createdBuffSize = 0;
	if (paths.size() > 0xffff)
		return NULL; // too much strings

	size_t buffSize = sizeof(size_t) + sizeof(size_t) * (1 + paths.size());
	size_t headerOffset = (size_t)buffSize;
	for (const std::wstring& path : paths)
	{
		if (path.size() > 0xffff)
			return NULL; // string too long

		buffSize += path.size() * sizeof(wchar_t);
	}
	
	if (buffSize > 0xffffffff)
		return NULL; // too much data

	char* buff = new char[buffSize];

	*(size_t*)buff = (size_t)buffSize;
	*(size_t*)(buff + sizeof(size_t)) = (size_t)paths.size();

	char* sptr = buff + headerOffset;
	for (size_t i=0 ; i< paths.size(); i++)// const std::wstring& path : paths)
	{
		// string size
		*(size_t*)(buff + sizeof(size_t) + sizeof(size_t) + sizeof(size_t)*i) = (size_t) paths[i].length();

		// string data
		size_t strBSize = paths[i].length() * sizeof(wchar_t);
		memcpy(sptr, paths[i].c_str(), strBSize);
		sptr += strBSize;
	}
	*createdBuffSize = (size_t)buffSize;
	return buff;
}

bool parseSplitAppRequestData(char* buff, size_t bufSize, std::vector<std::wstring> *appImages  )
{
	//	size_t common size bytes
	//	size_t strings cnt
	//	size_t str1Len
	//	size_t str2Len
	//	...
	//	WCHAR[] str1 
	//	WCHAR[] str2
	//	...

	if (appImages != NULL)
		appImages->clear();
		
	if (bufSize == sizeof(size_t) + sizeof(size_t))
	{
		size_t stringsCnt = *(size_t*)(buff + sizeof(size_t));
		if (stringsCnt == 0)
		{
			std::wcout << L"App configuration is empty\n";
			return true;
		}
		else
			std::wcout << L"Bad data\n";
		return false; // buffer is too small
	}
		
	if (bufSize < sizeof(size_t) + sizeof(size_t) * 2 + sizeof(wchar_t) * 1)
	{
		std::wcout << L"Bad data: buffer is too small\n";
		return false; // buffer is too small
	}
	
	if (*(size_t*)buff != bufSize)
	{
		std::wcout << L"Bad data: buffer size error\n";
		return false; // bad data
	}
	
	bool isOK = true;

	size_t stringsCnt = *(size_t*)(buff + sizeof(size_t));
	size_t headerSize = sizeof(size_t) + sizeof(size_t) + sizeof(size_t) * stringsCnt;
	char* strPtr = buff + headerSize;
	
	for (auto i = 0; i < stringsCnt; i++)
	{
		size_t strLen = *(size_t*)(buff + sizeof(size_t) + sizeof(size_t) + sizeof(size_t) * i);
		std::wstring str = std::wstring((wchar_t*)strPtr, (wchar_t*)(strPtr + strLen*sizeof(wchar_t)));
		strPtr += strLen* sizeof(wchar_t);

		std::wcout << L" app: '" << str << "'\n";
		if (strLen != str.length())
		{
			std::wcout << L"String length error\n";
			isOK = false;
		}

		if (appImages != NULL)
		{
			std::wstring prefix = L"\\??\\";
			if (str.find(prefix) != std::string::npos)
				str = str.substr(prefix.length());

			appImages->push_back(str);
		}		
	}

	return isOK;
}

void setSplitApp()
{
	std::vector<std::wstring> paths;

	std::wstring appPath;

	for (;;)
	{
		std::wcout << L" Application path to split: ";
		std::getline(std::wcin, appPath);
		if (appPath.length() <= 0)
			break;

		if (appPath.find(L"\\??\\") != 0)
			appPath = L"\\??\\" + appPath;

		paths.push_back(appPath);
	}
	// C:\Program Files\Mozilla Firefox\firefox.exe
	// C:\Windows\system32\cmd.exe
	//paths.push_back(L"\\??\\C:\\Program Files\\Mozilla Firefox\\firefox.exe");
	//paths.push_back(L"\\??\\C:\\Windows\\system32\\cmd.exe");

	size_t bufSize;
	char* buff = makeSplitAppRequestData(paths, &bufSize);

	if (parseSplitAppRequestData(buff, bufSize, NULL) == false)
	{
		std::wcout << L"Request not sent due to errors in prepared buffer\n";
	} 
	else
	{
		DWORD bytesRead = 0;
		SendIoctl(IOCTL_CFG_SET_IMAGES_TO_SPLIT, buff, (DWORD)bufSize, nullptr, 0, &bytesRead);
	}

	delete[] buff;
}

std::vector<std::wstring> getSplitApp()
{
	std::vector<std::wstring> retAppImages;

	DWORD bytesRead = 0;
	size_t buffSize = 0;
	if (!SendIoctl(IOCTL_CFG_GET_IMAGES_TO_SPLIT_BUFF_SIZE, NULL, 0, &buffSize, sizeof(buffSize), &bytesRead)
		|| bytesRead != sizeof(buffSize))
	{
		std::wcout << "IOCTL_CFG_GET_IMAGES_TO_SPLIT_BUFF_SIZE failed" << std::endl;
		return retAppImages;
	}

	char* buff = new char[buffSize];

	if (!SendIoctl(IOCTL_CFG_GET_IMAGES_TO_SPLIT, NULL, 0, buff, (DWORD)buffSize, &bytesRead))
		std::wcout << "IOCTL_CFG_GET_IMAGES_TO_SPLIT failed" << std::endl;
	else
	{
		if (parseSplitAppRequestData(buff, buffSize, &retAppImages) == false)
			std::wcout << L"ERROR parsing of received data\n";
	}
	delete[] buff;

	return retAppImages;
}

bool _processInfoSortFunction(ProcessInfo i, ProcessInfo j) 
{ 
	LONG ret = CompareFileTime(&i.CreationTime,&j.CreationTime);
	if (ret == 0) {
		return i.PID < j.PID;
	}
	return ret < 0;
}

std::vector<ProcessInfo> _filterProcesses(std::vector<std::wstring> appsToMonitor, std::vector<ProcessInfo> allApps)
{
	// sort processes by creation time
	std::sort(allApps.begin(), allApps.end(), _processInfoSortFunction);

	// prepare hash set of images to monitor
	std::unordered_set <std::wstring> appsToMonitorSet;
	for (std::wstring imj : appsToMonitor)
	{
		// to lower
		transform(imj.begin(), imj.end(), imj.begin(), towlower);
		appsToMonitorSet.insert(imj);
	}

	std::unordered_set <DWORD> pidToMonitorSet;
	std::vector<ProcessInfo> ret;
		
	for (ProcessInfo pi : allApps)
	{
		if (pidToMonitorSet.find(pi.PPID) != pidToMonitorSet.end())
		{
			pidToMonitorSet.insert(pi.PID);
			ret.push_back(pi);
			continue;
		}

		std::wstring imj = pi.Path;
		if (imj.length() > 0)
		{
			transform(imj.begin(), imj.end(), imj.begin(), towlower);

			if (appsToMonitorSet.find(imj) != appsToMonitorSet.end())
			{
				pidToMonitorSet.insert(pi.PID);
				ret.push_back(pi);
				continue;
			}
		}
	}

	return ret;
}

bool _getProcessList(std::vector<ProcessInfo> &ret)
{
	HANDLE hProcessSnap;
	
	PROCESSENTRY32 pe32;
		
	ret.clear();

	// Take a snapshot of all processes in the system.
	hProcessSnap = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);
	if (hProcessSnap == INVALID_HANDLE_VALUE)
		return false;
	
	// Set the size of the structure before using it.
	pe32.dwSize = sizeof(PROCESSENTRY32);

	// Retrieve information about the first process,
	// and exit if unsuccessful
	if (!Process32First(hProcessSnap, &pe32))
	{
		CloseHandle(hProcessSnap);          // clean the snapshot object
		return false;
	}

	// Now walk the snapshot of processes, and
	// display information about each process in turn
	do
	{
		ProcessInfo pi = {};
		HANDLE hProcess = OpenProcess(PROCESS_ALL_ACCESS, FALSE, pe32.th32ProcessID);
		if (hProcess != NULL)
		{
			// image path
			DWORD pathSize = MAX_PATH * 2;
			WCHAR path[MAX_PATH * 2 + 1] = { 0 };
			if (QueryFullProcessImageName(hProcess, 0, path, &pathSize))
				pi.Path = path;
			
			// creation time
			FILETIME creationTime, tmp;
			if (GetProcessTimes(hProcess, &creationTime, &tmp, &tmp, &tmp))
				pi.CreationTime = creationTime;

			CloseHandle(hProcess);
		}
			
		pi.PID = pe32.th32ProcessID;
		pi.PPID = pe32.th32ParentProcessID;

		ret.push_back(pi);
	} while (Process32Next(hProcessSnap, &pe32));

	CloseHandle(hProcessSnap);

	return true;
}

void pmInitRunningApps()
{
	std::vector<std::wstring> imagesToMonitor = getSplitApp();
	//imagesToMonitor.push_back(L"C:\\Program Files\\Mozilla Firefox\\firefox.exe");
	//imagesToMonitor.push_back(L"C:\\Windows\\system32\\cmd.exe");

	std::vector<ProcessInfo> pl;

	_getProcessList(pl);
	pl = _filterProcesses(imagesToMonitor, pl);

	if (pl.size() == 0)
	{
		std::wcout << L"Nothing to send. No processes is running (which we are interesting)\n";
		return;
	}

	// DWORD PID
	// DWORD PPID
	// ...

	const size_t bufSize = (DWORD) pl.size() * sizeof(DWORD) * 2;
	if (bufSize > 0xffffffff)
	{
		std::wcout << L"Error: too much data to send\n";
		return; // too much data
	}

	char* buff = new char[bufSize];
	DWORD* writePtr = (DWORD*)buff;
	for (ProcessInfo pi : pl)
	{
		*writePtr++ = pi.PID;
		*writePtr++ = pi.PPID;
	}

	DWORD bytesRead = 0;
	SendIoctl(IOCTL_PROCMON_SET_PID_DATA, buff, (DWORD)bufSize, nullptr, 0, &bytesRead);
	delete[] buff;
}

int main()
{
	std::wcout << L"Test console for ivpn-split-tunnel driver" << std::endl;
	for (;;)
	{
		std::wcout << L">> ";

		std::wstring userInput, temp;
		std::getline(std::wcin, userInput);

		std::vector<std::wstring> args;
		std::wstringstream wss(userInput);
		while (std::getline(wss, temp, L' ')) {
			if (temp.empty()) continue;
			args.push_back(temp);
		}

		if (args.empty()) continue;

		std::wstring command = args[0];
		std::transform(command.begin(), command.end(), command.begin(), towlower);

		if (0 == _wcsicmp(command.c_str(), L"exit") || 0 == _wcsicmp(command.c_str(), L"quit"))
		{
			break;
		}
				
		try
		{
			if (0 == _wcsicmp(command.c_str(), L"connect"))
			{
				connect();
				continue;
			}
			if (0 == _wcsicmp(command.c_str(), L"disconnect"))
			{
				disconnect();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"get-state")) // IOCTL_GET_STATE
			{
				getState();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"clean")) // IOCTL_STOP_ALL_AND_CFG_CLEAN
			{
				clean();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"split-start")) // IOCTL_SPLITTING_START
			{
				splitStart();
				pmInitRunningApps();	// IOCTL_CFG_GET_IMAGES_TO_SPLIT + IOCTL_PROCMON_SET_PID_DATA

				continue;
			}
			if (0 == _wcsicmp(command.c_str(), L"split-stop")) // IOCTL_SPLITTING_STOP
			{
				splitStop();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"pm-start")) // IOCTL_PROCMON_START
			{
				pmStart();				// IOCTL_PROCMON_START
				pmInitRunningApps();	// IOCTL_CFG_GET_IMAGES_TO_SPLIT + IOCTL_PROCMON_SET_PID_DATA
				continue;
			}
			if (0 == _wcsicmp(command.c_str(), L"pm-stop")) // IOCTL_PROCMON_STOP
			{
				pmStop();
				continue;
			}
			if (0 == _wcsicmp(command.c_str(), L"pm-init")) // IOCTL_CFG_GET_IMAGES_TO_SPLIT + IOCTL_PROCMON_SET_PID_DATA
			{
				pmInitRunningApps();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"get-config")) // IOCTL_CFG_SET_ADDRESSES
			{
				getAddresses(); 
				getSplitApp();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"set-addr")) // IOCTL_CFG_SET_ADDRESSES
			{
				setAddresses();
				continue;
			}
			if (0 == _wcsicmp(command.c_str(), L"get-addr")) // IOCTL_CFG_GET_ADDRESSES
			{
				getAddresses();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"set-app")) // IOCTL_CFG_SET_IMAGES_TO_SPLIT
			{
				setSplitApp();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"get-app")) // IOCTL_CFG_GET_IMAGES_TO_SPLIT 
			{
				getSplitApp();
				continue;
			}			
		}
		catch (const std::exception& ex)
		{
			std::cout << "Error: " << ex.what() << std::endl;
			continue;
		}

		std::wcout << L"Invalid command!" << std::endl;
		std::wcout << L"Allowed commands:\n connect\n disconnect\n get-state\n clean\n split-start\n split-stop\n pm-start\n pm-stop\n pm-init\n get-config\n set-addr\n get-addr\n set-app\n get-app" << std::endl;

	}

	std::wcout << L"Exiting ..." << std::endl;
	disconnect();

	return 0;
}