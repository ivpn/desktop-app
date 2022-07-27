@ECHO OFF
setlocal
set SCRIPTDIR=%~dp0
set APPVER=%1
set COMMIT=""
set DATE=""
set CERT_SHA1=%2

echo ==================================================
echo ============ BUILDING IVPN CLI ===================
echo ==================================================

rem Getting info about current date
FOR /F "tokens=* USEBACKQ" %%F IN (`date /T`) DO SET DATE=%%F
rem remove spaces
set DATE=%DATE: =%

rem Getting info about commit
cd %SCRIPTDIR%\..\..
FOR /F "tokens=* USEBACKQ" %%F IN (`git rev-list -1 HEAD`) DO SET COMMIT=%%F

if "%APPVER%" == "" set APPVER=unknown
rem Removing spaces from input variables
if NOT "%APPVER%" == "" set APPVER=%APPVER: =%
if NOT "%COMMIT%" == "" set COMMIT=%COMMIT: =%
if NOT "%DATE%" == "" set DATE=%DATE: =%

echo APPVER: %APPVER%
echo COMMIT: %COMMIT%
echo DATE  : %DATE%

call :build || goto :error
goto :success

:build
	echo [*] Building IVPN CLI

	if exist "bin\x86\cli\ivpn.exe" del "bin\x86\cli\ivpn.exe" || exit /b 1
	if exist "bin\x86_64\cli\ivpn.exe" del "bin\x86_64\cli\ivpn.exe" || exit /b 1

	set GOOS=windows

	rem echo [ ] x86 ...
	rem set GOARCH=386
	rem go build -tags release -o "bin\x86\cli\ivpn.exe" -trimpath -ldflags "-X github.com/ivpn/desktop-app/daemon/version._version=%APPVER% -X github.com/ivpn/desktop-app/daemon/version._commit=%COMMIT% -X github.com/ivpn/desktop-app/daemon/version._time=%DATE%" || exit /b 1

	echo [ ] x86_64 ...
	set GOARCH=amd64
	go build -tags release -o "bin\x86_64\cli\ivpn.exe" -trimpath -ldflags "-s -w -X github.com/ivpn/desktop-app/daemon/version._version=%APPVER% -X github.com/ivpn/desktop-app/daemon/version._commit=%COMMIT% -X github.com/ivpn/desktop-app/daemon/version._time=%DATE%" || exit /b 1

	set TIMESTAMP_SERVER=http://timestamp.digicert.com
	if NOT "%CERT_SHA1%" == "" (
		echo.
		echo Signing binary by certificate:  %CERT_SHA1% timestamp: %TIMESTAMP_SERVER%
		echo.
		signtool.exe sign /tr %TIMESTAMP_SERVER% /td sha256 /fd sha256 /sha1 %CERT_SHA1% /v "bin\x86_64\cli\ivpn.exe" || exit /b 1
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
	echo [!] IVPN Service build script FAILED with error #%errorlevel%.
	exit /b %errorlevel%
