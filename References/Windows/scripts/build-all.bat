@ECHO OFF

setlocal
set SCRIPTDIR=%~dp0
set APPVER=%1

rem E.g. 'exclude32bit'
set EXTRA_ARG=%2
set COMMIT=""
set DATE=""

rem ==================================================
rem DEFINE required WireGuard version here
set WGVER=v0.3.4
rem ==================================================

echo ==================================================
echo ============ BUILDING IVPN Service ===============
echo ==================================================

rem Getting info about current date
FOR /F "tokens=* USEBACKQ" %%F IN (`date /T`) DO SET DATE=%%F
rem Getting info about commit
cd %SCRIPTDIR%\..\..\..
FOR /F "tokens=* USEBACKQ" %%F IN (`git rev-list -1 HEAD`) DO SET COMMIT=%%F

echo APPVER: %APPVER%
echo COMMIT: %COMMIT%
echo DATE  : %DATE%

rem Checking if msbuild available
WHERE msbuild >nul 2>&1
IF %ERRORLEVEL% NEQ 0 (
	echo [!] 'msbuild' is not recognized as an internal or external command
	echo [!] Ensure you are running this script from Developer Cammand Prompt for Visual Studio
	goto :error
)

call :build_native_libs || goto :error

set needRebuildWireGuard=0
if not exist "%SCRIPTDIR%..\WireGuard\x86\wg.exe" 					set needRebuildWireGuard=1
if not exist "%SCRIPTDIR%..\WireGuard\x86\wireguard.exe" 			set needRebuildWireGuard=1
if not exist "%SCRIPTDIR%..\WireGuard\x86_64\wg.exe" 				set needRebuildWireGuard=1
if not exist "%SCRIPTDIR%..\WireGuard\x86_64\wireguard.exe" 		set needRebuildWireGuard=1
if not exist "%SCRIPTDIR%..\.deps\wireguard-windows\.deps\prepared" set needRebuildWireGuard=1
if %needRebuildWireGuard% == 1 call :build_wireguard || goto :error

call :update_servers_info || goto :error
call :build_agent || goto :error

(
	rem Save Go variables (to be able to compile CLI with the same Go version)
	rem parenthesis "()" are important here !
	endlocal
	set "IVPN_GOROOT=%GOROOT%"
	set "IVPN_PATH=%PATH%"
)

rem THE END
goto :success

:update_servers_info
	echo [*] Updating servers.json ...
	curl -#fLo %SCRIPTDIR%..\etc\servers.json https://api.ivpn.net/v4/servers.json || exit /b 1
	goto :eof

:build_agent
	set GOOS=windows
	set GOPATH=%SCRIPTDIR%..\.deps\wireguard-windows\.deps\gopath
	set GOROOT=%SCRIPTDIR%..\.deps\wireguard-windows\.deps\go
	set PATH=%SCRIPTDIR%..\.deps\wireguard-windows\.deps\go\bin;%PATH%
	cd "%SCRIPTDIR%..\..\.."

	IF not "%EXTRA_ARG%" == "exclude32bit" (
		call :build_agent_plat x86 386 		|| exit /b 1
	)
	IF not "%EXTRA_ARG%" == "exclude64bit" (
		call :build_agent_plat x86_64 amd64 	|| exit /b 1
	)

	goto :eof

:build_agent_plat
	set GOARCH=%~2

	echo [*] Building IVPN service %1

	if exist "bin\%~1\IVPN Service.exe" del "bin\%~1\IVPN Service.exe" || exit /b 1

	go build -tags release -o "bin\%~1\IVPN Service.exe" -trimpath -ldflags "-X github.com/ivpn/desktop-app-daemon/version._version=%APPVER% -X github.com/ivpn/desktop-app-daemon/version._commit=%COMMIT% -X github.com/ivpn/desktop-app-daemon/version._time=%DATE%" || exit /b 1
	goto :eof

:build_native_libs
	IF not "%EXTRA_ARG%" == "exclude32bit" (
		echo [*] Building Native projects x86
		msbuild "%SCRIPTDIR%..\Native Projects\ivpn-windows-native.sln" /verbosity:quiet /t:Build /property:Configuration=Release /property:Platform=x86 || exit /b 1
	)
	IF not "%EXTRA_ARG%" == "exclude64bit" (
		echo [*] Building Native projects x64
		msbuild "%SCRIPTDIR%..\Native Projects\ivpn-windows-native.sln" /verbosity:quiet /t:Build /property:Configuration=Release /property:Platform=x64 || exit /b 1
	)
	goto :eof

:build_wireguard
	echo ### WireGuard binaries not found ###
	echo ### Buildind WireGuard binaries  ###

	if exist "%SCRIPTDIR%..\WireGuard\x86" (
		echo [*] Erasing WireGuard\x86\*.exe ...
		del /f /q /s "%SCRIPTDIR%..\WireGuard\x86\*.exe"  	>nul 2>&1 || exit /b 1
	)

	if exist "%SCRIPTDIR%..\WireGuard\x86_64" (
		echo [*] Erasing WireGuard\x86_64\*.exe ...
		del /f /q /s "%SCRIPTDIR%..\WireGuard\x86_64\*.exe" >nul 2>&1 || exit /b 1
	)

	if not exist "%SCRIPTDIR%..\.deps\wireguard-windows\.deps\prepared" (
		if exist "%SCRIPTDIR%..\.deps" (
			echo [*] Erasing .deps ...
			rd /s /q "%SCRIPTDIR%..\.deps" || exit /b 1
			sleep 2
		)

		echo [*] Creating .deps ...
		mkdir "%SCRIPTDIR%..\.deps" || exit /b 1
		cd "%SCRIPTDIR%..\.deps" 	|| exit /b 1

		echo [*] Cloning wireguard-windows...
		git clone https://git.zx2c4.com/wireguard-windows || exit /b 1
		cd wireguard-windows || exit /b 1

		echo [*] Checking out wireguard-windows version [%WGVER%]...
		git checkout %WGVER% >nul 2>&1 || exit /b 1
	) else (
		cd "%SCRIPTDIR%..\.deps\wireguard-windows" 	|| exit /b 1
	)

	echo [*] Building wireguard-windows ...
	call build.bat || exit /b 1

	echo [*] WireGuard build DONE. Copying compiled binaries ...

	if not exist "%SCRIPTDIR%..\WireGuard" 			mkdir "%SCRIPTDIR%..\WireGuard" 		|| exit /b 1
	if not exist "%SCRIPTDIR%..\WireGuard\x86" 		mkdir "%SCRIPTDIR%..\WireGuard\x86"		|| exit /b 1
	if not exist "%SCRIPTDIR%..\WireGuard\x86_64" 	mkdir "%SCRIPTDIR%..\WireGuard\x86_64"	|| exit /b 1

	copy /y "%SCRIPTDIR%..\.deps\wireguard-windows\x86\wg.exe" 			"%SCRIPTDIR%..\WireGuard\x86\wg.exe" 			>nul 2>&1 || exit /b 1
	copy /y "%SCRIPTDIR%..\.deps\wireguard-windows\x86\wireguard.exe" 	"%SCRIPTDIR%..\WireGuard\x86\wireguard.exe" 	>nul 2>&1 || exit /b 1
	copy /y "%SCRIPTDIR%..\.deps/wireguard-windows\amd64\wg.exe" 		"%SCRIPTDIR%..\WireGuard\x86_64\wg.exe" 		>nul 2>&1 || exit /b 1
	copy /y "%SCRIPTDIR%..\.deps\wireguard-windows\amd64\wireguard.exe" "%SCRIPTDIR%..\WireGuard\x86_64\wireguard.exe" 	>nul 2>&1 || exit /b 1
	goto :eof

:success
	echo [*] Success.
	go version
	exit /b 0

:error
	echo [!] IVPN Service build script FAILED with error #%errorlevel%.
	exit /b %errorlevel%
