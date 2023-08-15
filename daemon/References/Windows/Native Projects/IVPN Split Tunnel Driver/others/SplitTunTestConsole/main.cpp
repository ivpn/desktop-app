//
//  The test console for Split-Tunnel Driver (Windows)
//  https://github.com/ivpn/desktop-app/daemon/References/Windows/Native%20Projects/IVPN%20Split%20Tunnel%20Driver
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
// 
//  This file is part of the IVPN Client Desktop project.
//  https://github.com/ivpn/desktop-app
//

#include <iostream>

#include "../common/SplitTunDrvControl/ivpnSplitTunDrvControl.h"

void Log(std::wstring logMes)
{
	// ensure the string ends with a new line
	if (!logMes.empty() && logMes[logMes.length() - 1] != '\n') {
		logMes += L"\n";
	}

	wprintf(L"    [log]: %s", logMes.c_str());
}

void doStartDriverAsService()
{
	std::wstring sysFilePath;

	std::wcout << L" Full path to driver *.sys file: ";
	std::getline(std::wcin, sysFilePath);
	if (sysFilePath.length() <= 0)
	{
		std::wcout << L"Error: Not defined path to driver binary" << std::endl;
		return;
	}

	if (!splittun::StartDriverAsService(sysFilePath.c_str()))
	{
		std::wcout << L"Failed" << std::endl;
	}
}

void doGetState()
{
	DriverStatus s = {};
		
	if (splittun::GetState(s))
	{
		std::wcout << "Driver state:"
			<< "\n IsConfigOk			: " << ((s.IsConfigOk) ? "Yes" : "No")
			<< "\n IsEnabledProcessMonitor	: " << ((s.IsEnabledProcessMonitor) ? "Yes" : "No")
			<< "\n IsEnabledSplitting		: " << ((s.IsEnabledSplitting) ? "Yes" : "No")
			<< std::endl;
	}
	else
		std::wcout << "IOCTL_GET_STATE failed" << std::endl;
}

void doConfigGetAddresses()
{
	IPAddrConfig cfg = {};

	if (splittun::ConfigGetAddresses(cfg))
	{
		auto pub = cfg.IPv4Public.S_un.S_un_b;
		auto tun = cfg.IPv4Tunnel.S_un.S_un_b;

		WCHAR ipv4Pub[16] = { 0 }, ipv4tun[16] = { 0 }, ipv6Pub[46] = { 0 }, ipv6tun[46] = { 0 };

		InetNtopW(AF_INET, &cfg.IPv4Public, &ipv4Pub[0], 16);
		InetNtopW(AF_INET, &cfg.IPv4Tunnel, &ipv4tun[0], 16);
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


void doConfigSetAddresses() {
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

	splittun::ConfigSetAddresses(cfg);
}

void doConfigGetSplitApp()
{
	std::vector<std::wstring> retAppImages;

	splittun::ConfigGetSplitApp(retAppImages);

	std::wcout << "Configuration (applications to split):" << std::endl;
	for (std::wstring path : retAppImages)
		std::wcout << " " << path << std::endl;
}

void doConfigSetSplitApp()
{
	std::vector<std::wstring> paths;

	std::wstring appPath;
	
	std::wcout << L" Applications to split (press Enter to finish)\n";
	for (;;)
	{
		std::wcout << L"   application path: ";
		std::getline(std::wcin, appPath);
		if (appPath.length() <= 0)
			break;

		paths.push_back(appPath);
	}
	// C:\Program Files\Mozilla Firefox\firefox.exe
	// C:\Windows\system32\cmd.exe
	//paths.push_back(L"\\??\\C:\\Program Files\\Mozilla Firefox\\firefox.exe");
	//paths.push_back(L"\\??\\C:\\Windows\\system32\\cmd.exe");

	splittun::ConfigSetSplitApp(paths);
}

int main()
{
	std::wcout << L"Test console for ivpn-split-tunnel driver" << std::endl;
	splittun::RegisterLoggingCallback(Log);

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
			break;
				
		try
		{
			if (0 == _wcsicmp(command.c_str(), L"start-driver"))
			{
				doStartDriverAsService();
				continue;
			}
			if (0 == _wcsicmp(command.c_str(), L"stop-driver"))
			{
				splittun::StopDriverAsService();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"connect"))
			{
				splittun::Connect();
				continue;
			}
			if (0 == _wcsicmp(command.c_str(), L"disconnect"))
			{
				splittun::Disconnect();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"get-state")) // IOCTL_GET_STATE
			{
				doGetState();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"clean")) // IOCTL_STOP_ALL_AND_CFG_CLEAN
			{
				splittun::StopAndClean();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"split-start")) // IOCTL_SPLITTING_START
			{
				if (splittun::SplitStart())	// IOCTL_SPLITTING_START
					splittun::ProcMonInitRunningApps();	// IOCTL_CFG_GET_IMAGES_TO_SPLIT + IOCTL_PROCMON_SET_PID_DATA

				continue;
			}
			if (0 == _wcsicmp(command.c_str(), L"split-stop")) // IOCTL_SPLITTING_STOP
			{
				splittun::SplitStop();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"pm-start")) // IOCTL_PROCMON_START
			{
				splittun::ProcMonStart();				// IOCTL_PROCMON_START
				splittun::ProcMonInitRunningApps();	// IOCTL_CFG_GET_IMAGES_TO_SPLIT + IOCTL_PROCMON_SET_PID_DATA
				continue;
			}
			if (0 == _wcsicmp(command.c_str(), L"pm-stop")) // IOCTL_PROCMON_STOP
			{
				splittun::ProcMonStop();
				continue;
			}
			if (0 == _wcsicmp(command.c_str(), L"pm-init")) // IOCTL_CFG_GET_IMAGES_TO_SPLIT + IOCTL_PROCMON_SET_PID_DATA
			{
				splittun::ProcMonInitRunningApps();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"get-config")) // IOCTL_CFG_SET_ADDRESSES
			{
				doConfigGetAddresses();
				doConfigGetSplitApp();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"set-addr")) // IOCTL_CFG_SET_ADDRESSES
			{
				doConfigSetAddresses();
				continue;
			}
			if (0 == _wcsicmp(command.c_str(), L"get-addr")) // IOCTL_CFG_GET_ADDRESSES
			{
				doConfigGetAddresses();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"set-app")) // IOCTL_CFG_SET_IMAGES_TO_SPLIT
			{
				doConfigSetSplitApp();
				continue;
			}

			if (0 == _wcsicmp(command.c_str(), L"get-app")) // IOCTL_CFG_GET_IMAGES_TO_SPLIT 
			{
				doConfigGetSplitApp();
				continue;
			}
		}
		catch (const std::exception& ex)
		{
			std::cout << "Error: " << ex.what() << std::endl;
			continue;
		}

		std::wcout << L"Invalid command!" << std::endl;
		std::wcout << L"Allowed commands:\n\
 start-driver\n stop-driver\n\
 connect\n disconnect\n get-state\n clean\n split-start\n split-stop\n\
 pm-start\n pm-stop\n pm-init\n get-config\n set-addr\n get-addr\n\
 set-app\n get-app\n" << std::endl;

	}

	std::wcout << L"Exiting ..." << std::endl;
	splittun::Disconnect();
	
	return 0;
}