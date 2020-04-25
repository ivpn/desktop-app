@ECHO OFF
setlocal
set SCRIPTDIR=%~dp0
set APPVER=%1
set COMMIT=""
set DATE=""

echo ==================================================
echo ============ BUILDING IVPN CLI ===================
echo ==================================================

rem Getting info about current date
FOR /F "tokens=* USEBACKQ" %%F IN (`date /T`) DO SET DATE=%%F
rem Getting info about commit
cd %SCRIPTDIR%\..\..
FOR /F "tokens=* USEBACKQ" %%F IN (`git rev-list -1 HEAD`) DO SET COMMIT=%%F

echo APPVER: %APPVER%
echo COMMIT: %COMMIT%
echo DATE  : %DATE%

call :build || goto :error
goto :success

:build
	echo [*] Building IVPN CLI

	IF "%IVPN_GOROOT%" == "" goto :build_skip_ivpn_vars

	rem There are Go variables saved by another IVPN script (to be able to compile CLI with the same Go version)
	echo [!] Using IVPN environment variables IVPN_GOROOT and IVPN_PATH
	set "GOROOT=%IVPN_GOROOT%"
	set "PATH=%IVPN_PATH%"
	echo *    GOROOT= %GOROOT%
	echo *    PATH  = %PATH%

:build_skip_ivpn_vars

	if exist "bin\x86\ivpn.exe" del "bin\x86\ivpn.exe" || exit /b 1
	if exist "bin\x86_64\ivpn.exe" del "bin\x86_64\ivpn.exe" || exit /b 1

	set GOOS=windows

	echo [ ] x86 ...
	set GOARCH=386

	go build -tags release -o "bin\x86\ivpn.exe" -ldflags "-X github.com/ivpn/desktop-app-daemon/version._version=%APPVER% -X github.com/ivpn/desktop-app-daemon/version._commit=%COMMIT% -X github.com/ivpn/desktop-app-daemon/version._time=%DATE%" || exit /b 1

	echo [ ] x86_64 ...
	set GOARCH=amd64
	go build -tags release -o "bin\x86_64\ivpn.exe" -ldflags "-X github.com/ivpn/desktop-app-daemon/version._version=%APPVER% -X github.com/ivpn/desktop-app-daemon/version._commit=%COMMIT% -X github.com/ivpn/desktop-app-daemon/version._time=%DATE%" || exit /b 1

	goto :eof

:success
	echo [*] Success.
	go version
	exit /b 0

:error
	echo [!] IVPN Service build script FAILED with error #%errorlevel%.
	exit /b %errorlevel%
