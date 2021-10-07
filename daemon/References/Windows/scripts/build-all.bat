@ECHO OFF

setlocal
set SCRIPTDIR=%~dp0
set APPVER=%1

set CERT_SHA1=%2

set COMMIT=""
set DATE=""

set TIMESTAMP_SERVER=http://timestamp.digicert.com

echo ==================================================
echo ============ BUILDING IVPN Service ===============
echo ==================================================

rem Getting info about current date
FOR /F "tokens=* USEBACKQ" %%F IN (`date /T`) DO SET DATE=%%F
rem remove spaces
set DATE=%DATE: =%

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
call :build_obfs4proxy || goto :error
call :build_wireguard || goto :error
call :update_servers_info || goto :error
call :build_agent || goto :error

rem THE END
goto :success

:update_servers_info
	echo [*] Updating servers.json ...
	curl -#fLo %SCRIPTDIR%..\etc\servers.json https://api.ivpn.net/v4/servers.json || exit /b 1
	goto :eof

:build_agent
	cd "%SCRIPTDIR%..\..\.."
	call :build_agent_plat x86_64 amd64 	|| exit /b 1
	goto :eof

:build_agent_plat
	set GOARCH=%~2

	echo [*] Building IVPN service %1

	if exist "bin\%~1\IVPN Service.exe" del "bin\%~1\IVPN Service.exe" || exit /b 1

	go build -tags release -o "bin\%~1\IVPN Service.exe" -trimpath -ldflags "-X github.com/ivpn/desktop-app/daemon/version._version=%APPVER% -X github.com/ivpn/desktop-app/daemon/version._commit=%COMMIT% -X github.com/ivpn/desktop-app/daemon/version._time=%DATE%" || exit /b 1

	if NOT "%CERT_SHA1%" == "" (
		echo.
		echo Signing binary by certificate:  %CERT_SHA1% timestamp: %TIMESTAMP_SERVER%
		echo.
		signtool.exe sign /tr %TIMESTAMP_SERVER% /td sha256 /fd sha256 /sha1 %CERT_SHA1% /v "bin\%~1\IVPN Service.exe" || exit /b 1
		echo.
		echo Signing SUCCES
		echo.
	)

	echo Compiled binary: "bin\%~1\IVPN Service.exe"
	goto :eof

:build_native_libs
	echo [*] Building Native projects x64
	msbuild "%SCRIPTDIR%..\Native Projects\ivpn-windows-native.sln" /verbosity:quiet /t:Build /property:Configuration=Release /property:Platform=x64 || exit /b 1
	goto :eof

:build_obfs4proxy
	if exist "%SCRIPTDIR%..\OpenVPN\obfsproxy\obfs4proxy.exe" (
		echo [ ] obfs4proxy binaries already available. Compilation skipped.
		goto :eof
	)

	echo ### obfs4proxy binary not found ###
	echo ### Buildind obfs4proxy         ###
	call "%SCRIPTDIR%\build-obfs4proxy.bat" || goto error

	if NOT "%CERT_SHA1%" == "" (
		echo.
		echo Signing 'obfs4proxy.exe' binary [certificate:  %CERT_SHA1% timestamp: %TIMESTAMP_SERVER%]
		echo.
		signtool.exe sign /tr %TIMESTAMP_SERVER% /td sha256 /fd sha256 /sha1 %CERT_SHA1% /v "%SCRIPTDIR%..\OpenVPN\obfsproxy\obfs4proxy.exe" || goto :eof
		echo.
		echo Signing SUCCES
		echo.
	)

	goto :eof

:build_wireguard
	if exist "%SCRIPTDIR%..\WireGuard\x86_64\wg.exe" (
 		if exist "%SCRIPTDIR%..\WireGuard\x86_64\wireguard.exe" (
			echo [ ] Wireguard binaries already available. Compilation skipped.
			goto :eof
		)
	)

	echo ### WireGuard binaries not found ###
	call "%SCRIPTDIR%\build-wireguard.bat" || goto error

	if NOT "%CERT_SHA1%" == "" (
		echo.
		echo Signing binaries ['wg.exe', 'wireguard.exe'] [certificate:  %CERT_SHA1% timestamp: %TIMESTAMP_SERVER%]
		echo.
		signtool.exe sign /tr %TIMESTAMP_SERVER% /td sha256 /fd sha256 /sha1 %CERT_SHA1% /v "%SCRIPTDIR%..\WireGuard\x86_64\wg.exe" || goto :eof
		signtool.exe sign /tr %TIMESTAMP_SERVER% /td sha256 /fd sha256 /sha1 %CERT_SHA1% /v "%SCRIPTDIR%..\WireGuard\x86_64\wireguard.exe" || goto :eof
		echo.
		echo Signing SUCCES
		echo.
	)

	goto :eof

:success
	echo [*] Success.
	go version
	exit /b 0

:error
	set ERR=%errorlevel%
	echo [!] IVPN Service build script FAILED with error #%errorlevel%.
	echo [!] Removing files:
	echo [ ] "%SCRIPTDIR%..\OpenVPN\obfsproxy\obfs4proxy.exe"
	echo [ ] "%SCRIPTDIR%..\WireGuard\x86_64\wg.exe"
	echo [ ] "%SCRIPTDIR%..\WireGuard\x86_64\wireguard.exe"
	del "%SCRIPTDIR%..\OpenVPN\obfsproxy\obfs4proxy.exe"
	del "%SCRIPTDIR%..\WireGuard\x86_64\wg.exe"
	del "%SCRIPTDIR%..\WireGuard\x86_64\wireguard.exe"

	exit /b %ERR%
