#include "ivpnSplitTunDrvControl.h"

namespace splittun
{
	////////////////////////////////////////////////////////////////
	// 	   Private functionality
	////////////////////////////////////////////////////////////////
	HANDLE   hDevice = INVALID_HANDLE_VALUE;
	LoggingCallback cbLog = NULL;

	struct ProcessInfo
	{
		DWORD			PID;
		DWORD			PPID;
		std::wstring	Path;
		FILETIME		CreationTime;
	};

	void _sendToLog(std::wstring str)
	{
		LoggingCallback l = cbLog;
		if (l!=NULL)
			l(str);
	}

	std::wstring _getLastErrorStr()
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

	bool _sendIoctl(DWORD code,
		LPVOID lpInBuffer, DWORD nInBufferSize,
		LPVOID lpOutBuffer, DWORD nOutBufferSize,
		LPDWORD lpBytesReturned)
	{
		BOOL bRc;

		if (cbLog != NULL)
		{
			std::wostringstream ss;
			ss << " ==> " << std::hex << "0x" << code << std::dec << " ...";
			_sendToLog(ss.str());
		}

		DWORD tmpBytesReturned;
		if (lpBytesReturned == NULL)
			lpBytesReturned = &tmpBytesReturned;

		bRc = DeviceIoControl(hDevice, code,
			lpInBuffer, nInBufferSize,
			lpOutBuffer, nOutBufferSize,
			lpBytesReturned,
			NULL
		);

		if (cbLog != NULL) 
		{
			std::wostringstream ss;
			if (!bRc)
				ss << " Error in '_sendIoctl': " << _getLastErrorStr() << std::endl;
			else
				ss << " SUCCESS. Bytes returned: " << *lpBytesReturned << std::endl;
			_sendToLog(ss.str());
		}
		
		return bRc;
	}

	unsigned char* _splitAppMakeRequestData(std::vector<std::wstring> paths, DWORD* createdBuffSize)
	{
		//	DWORD common size bytes
		//	DWORD strings cnt
		//	DWORD str1Len
		//	DWORD str2Len
		//	...
		//	WCHAR[] str1 
		//	WCHAR[] str2
		//	...

		*createdBuffSize = 0;
		if (paths.size() > 0xffff)
			return NULL; // too much strings

		size_t buffSize = sizeof(DWORD) + sizeof(DWORD) * (1 + paths.size());
		DWORD headerOffset = (DWORD)buffSize;
		for (const std::wstring& path : paths)
		{
			if (path.size() > 0xffff)
				return NULL; // string too long

			buffSize += path.size() * sizeof(wchar_t);
		}

		if (buffSize > 0xffffffff)
			return NULL; // too much data

		unsigned char* buff = new unsigned char[buffSize];

		*(DWORD*)buff = (DWORD)buffSize;
		*(DWORD*)(buff + sizeof(DWORD)) = (DWORD)paths.size();

		unsigned char* sptr = buff + headerOffset;
		for (size_t i = 0; i < paths.size(); i++)// const std::wstring& path : paths)
		{
			// string size
			*(DWORD*)(buff + sizeof(DWORD) + sizeof(DWORD) + sizeof(DWORD) * i) = (DWORD)paths[i].length();

			// string data
			size_t strBSize = paths[i].length() * sizeof(wchar_t);
			memcpy(sptr, paths[i].c_str(), strBSize);
			sptr += strBSize;
		}
		*createdBuffSize = (DWORD)buffSize;
		return buff;
	}

	bool _splitAppParseData(unsigned char* buff, DWORD bufSize, std::vector<std::wstring>* appImages)
	{
		//	DWORD common size bytes
		//	DWORD strings cnt
		//	DWORD str1Len
		//	DWORD str2Len
		//	...
		//	WCHAR[] str1 
		//	WCHAR[] str2
		//	...

		if (appImages != NULL)
			appImages->clear();

		if (bufSize == sizeof(DWORD) + sizeof(DWORD))
		{
			DWORD stringsCnt = *(DWORD*)(buff + sizeof(DWORD));
			if (stringsCnt == 0)
			{
				_sendToLog(L"App configuration is empty\n");
				return true;
			}
			else
				_sendToLog(L"Bad data\n");
			return false; // buffer is too small
		}

		if (bufSize < sizeof(DWORD) + sizeof(DWORD) * 2 + sizeof(wchar_t) * 1)
		{
			_sendToLog(L"Bad data: buffer is too small\n");
			return false; // buffer is too small
		}

		if (*(DWORD*)buff != bufSize)
		{
			_sendToLog (L"Bad data: buffer size error\n");
			return false; // bad data
		}

		bool isOK = true;

		DWORD stringsCnt = *(DWORD*)(buff + sizeof(DWORD));
		DWORD headerSize = sizeof(DWORD) + sizeof(DWORD) + sizeof(DWORD) * stringsCnt;
		unsigned char* strPtr = buff + headerSize;

		for (DWORD i = 0; i < stringsCnt; i++)
		{
			DWORD strLen = *(DWORD*)(buff + sizeof(DWORD) + sizeof(DWORD) + sizeof(DWORD) * i);
			std::wstring str = std::wstring((wchar_t*)strPtr, (wchar_t*)(strPtr + strLen * sizeof(wchar_t)));
			strPtr += strLen * sizeof(wchar_t);
						
			if (strLen != str.length())
			{
				_sendToLog(L"String length error\n");
				isOK = false;
			}

			if (appImages != NULL)
				appImages->push_back(str);
		}

		return isOK;
	}


	bool _processListGet(std::vector<ProcessInfo>& ret)
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

	bool _processInfoSortFunction(ProcessInfo i, ProcessInfo j)
	{
		LONG ret = CompareFileTime(&i.CreationTime, &j.CreationTime);
		if (ret == 0) {
			return i.PID < j.PID;
		}
		return ret < 0;
	}


	std::vector<ProcessInfo> _processListFilter(std::vector<std::wstring> appsToMonitor, std::vector<ProcessInfo> allApps)
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
	////////////////////////////////////////////////////////////////
	// 	   Public functionality
	////////////////////////////////////////////////////////////////
	
	void RegisterLoggingCallback(LoggingCallback cb)
	{
		cbLog = cb;
	}

	void UnRegisterLoggingCallback()
	{
		cbLog = NULL;
	};

	bool Connect()
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

		bool isSuccess = hDevice != INVALID_HANDLE_VALUE;
		
		if (cbLog != NULL)
		{
			if (isSuccess)
				_sendToLog(L"Success: connected\n");
			else
				_sendToLog(L"Error: " + _getLastErrorStr() + L"\n");
		}

		return isSuccess;
	}

	bool Disconnect()
	{
		if (hDevice == INVALID_HANDLE_VALUE)
			return true;

		bool ret = CloseHandle(hDevice);
		hDevice = INVALID_HANDLE_VALUE;
		
		if (cbLog != NULL)
		{
			if (ret)
				_sendToLog(L"Success: disconnected\n");
			else
				_sendToLog(L"Error: " + _getLastErrorStr() + L"\n");
		}

		return ret;
	}

	bool StopAndClean()
	{
		return _sendIoctl(IOCTL_STOP_ALL_AND_CFG_CLEAN, nullptr, 0, nullptr, 0, NULL);
	}

	bool GetState(DriverStatus &s)
	{
		DWORD bytesRead = 0;

		return _sendIoctl(IOCTL_GET_STATE, NULL, 0, &s, sizeof(s), &bytesRead) 
			&& bytesRead == sizeof(s);
	}

	bool ConfigSetAddresses(IPAddrConfig cfg) 
	{	
		return _sendIoctl(IOCTL_CFG_SET_ADDRESSES, &cfg, sizeof(cfg), nullptr, 0, NULL);
	}

	bool ConfigGetAddresses(IPAddrConfig& _out_cfg)
	{
		DWORD bytesRead = 0;
		return _sendIoctl(IOCTL_CFG_GET_ADDRESSES, NULL, 0, &_out_cfg, sizeof(_out_cfg), &bytesRead) && bytesRead == sizeof(_out_cfg);
	}

	bool ConfigSetSplitApp(std::vector<std::wstring> appPaths)
	{
		bool ret = false;
		DWORD bufSize;
		
		for (size_t i = 0; i < appPaths.size(); i++)
		{
			if (appPaths[i].find(L"\\??\\") != 0)
				appPaths[i] = L"\\??\\" + appPaths[i];
		}

		unsigned char* buff = _splitAppMakeRequestData(appPaths, &bufSize);
		ret = _sendIoctl(IOCTL_CFG_SET_IMAGES_TO_SPLIT, buff, (DWORD)bufSize, nullptr, 0, NULL);
		delete[] buff;

		return ret;
	}

	bool ConfigGetSplitApp(std::vector<std::wstring> &retAppImages)
	{
		DWORD bytesRead = 0;
		DWORD buffSize = 0;

		if (!_sendIoctl(IOCTL_CFG_GET_IMAGES_TO_SPLIT_BUFF_SIZE, NULL, 0, &buffSize, sizeof(buffSize), &bytesRead)
			|| bytesRead != sizeof(buffSize))
		{
			_sendToLog(L"IOCTL_CFG_GET_IMAGES_TO_SPLIT_BUFF_SIZE failed");
			return false;
		}

		unsigned char* buff = new unsigned char[buffSize];

		bool ret = false;
		if (!_sendIoctl(IOCTL_CFG_GET_IMAGES_TO_SPLIT, NULL, 0, buff, (DWORD)buffSize, &bytesRead) || bytesRead != buffSize)
			_sendToLog(L"IOCTL_CFG_GET_IMAGES_TO_SPLIT failed");
		else
		{
			if (_splitAppParseData(buff, buffSize, &retAppImages))
				ret = true; 
			else 
				_sendToLog(L"ERROR parsing of received data\n");
		}
		delete[] buff;

		if (ret == true) 
		{
			const std::wstring prefix = L"\\??\\";
			for (size_t i = 0; i < retAppImages.size(); i++)
			{
				if(retAppImages[i].find(prefix) != std::string::npos)
					retAppImages[i] = retAppImages[i].substr(prefix.length());
			}
		}

		return ret;
	}

	bool ConfigSetSplitAppRaw(unsigned char* buff, DWORD _in_buffSize)
	{
		bool ret = false;

		std::vector<std::wstring> appImages;
		if (_splitAppParseData(buff, _in_buffSize, &appImages) == false)
			_sendToLog(L"Request not sent due to errors in prepared buffer\n");
		else
		{
			// We should ensure that app images starts from  L"\\??\\"
			// Therefore we are not calling IOCTL_CFG_SET_IMAGES_TO_SPLIT directly
			// ret = _sendIoctl(IOCTL_CFG_SET_IMAGES_TO_SPLIT, buff, (DWORD)_in_buffSize, nullptr, 0, NULL);
			ret = ConfigSetSplitApp(appImages);
		}

		return ret;
	}
	bool ConfigGetSplitAppRaw(unsigned char* buff, DWORD* _in_out_buffSize)
	{
		DWORD bytesRead = 0;
		DWORD buffSize = 0;

		if (!_sendIoctl(IOCTL_CFG_GET_IMAGES_TO_SPLIT_BUFF_SIZE, NULL, 0, &buffSize, sizeof(buffSize), &bytesRead)
			|| bytesRead != sizeof(buffSize))
		{
			_sendToLog(L"IOCTL_CFG_GET_IMAGES_TO_SPLIT_BUFF_SIZE failed");
			return false;
		}

		if (*_in_out_buffSize < buffSize)
		{
			*_in_out_buffSize = buffSize;
			return false;
		}
		*_in_out_buffSize = buffSize;
				
		bool ret = false;
		if (!_sendIoctl(IOCTL_CFG_GET_IMAGES_TO_SPLIT, NULL, 0, buff, (DWORD)buffSize, &bytesRead) || bytesRead != buffSize)
		{
			_sendToLog(L"IOCTL_CFG_GET_IMAGES_TO_SPLIT failed");
			return false;
		}

		return true;
	}

	bool ProcMonStart()
	{
		return _sendIoctl(IOCTL_PROCMON_START, nullptr, 0, nullptr, 0, NULL);
	}

	bool ProcMonStop()
	{
		return _sendIoctl(IOCTL_PROCMON_STOP, nullptr, 0, nullptr, 0, NULL);
	}

	bool ProcMonInitRunningApps()
	{
		std::vector<std::wstring> imagesToMonitor;
		if (!ConfigGetSplitApp(imagesToMonitor))
		{
			_sendToLog(L"Error: Unable to retrieve current configuration about splitting apps\n");
			return false;
		}

		std::vector<ProcessInfo> pl;

		_processListGet(pl);
		pl = _processListFilter(imagesToMonitor, pl);

		if (pl.size() == 0)
		{
			_sendToLog(L"Nothing to send. No processes is running (which we are interesting)\n");
			return true;
		}

		// DWORD PID
		// DWORD PPID
		// ...

		const size_t bufSize = (DWORD)pl.size() * sizeof(DWORD) * 2;
		if (bufSize > 0xffffffff)
		{
			_sendToLog (L"Error: too much data to send\n");
			return false; // too much data
		}

		char* buff = new char[bufSize];
		DWORD* writePtr = (DWORD*)buff;
		for (ProcessInfo pi : pl)
		{
			*writePtr++ = pi.PID;
			*writePtr++ = pi.PPID;
		}

		bool ret = _sendIoctl(IOCTL_PROCMON_SET_PID_DATA, buff, (DWORD)bufSize, nullptr, 0, NULL);
		delete[] buff;

		return ret;
	}

	bool SplitStart()
	{
		return _sendIoctl(IOCTL_SPLITTING_START, nullptr, 0, nullptr, 0, NULL);
	}

	bool SplitStop()
	{
		return _sendIoctl(IOCTL_SPLITTING_STOP, nullptr, 0, nullptr, 0, NULL);
	}
}