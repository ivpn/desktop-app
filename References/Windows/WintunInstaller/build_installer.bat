@echo off
setlocal

set BUILDDIR=%~dp0
cd /d %BUILDDIR% || exit /b 1

if exist .deps\prepared goto :build
:installdeps
	rmdir /s /q .deps 2> NUL
	mkdir .deps || goto :error
	cd .deps || goto :error
	call :download wintun-x86.msm https://www.wintun.net/builds/wintun-x86-0.8.msm 7ff5fcca21be75584fea830a4624ff52305ebb6982c3ec1b294a22b20ee5c1fc || goto :error
	call :download wintun-amd64.msm https://www.wintun.net/builds/wintun-amd64-0.8.msm 14e94f3151e425d80fc262b4bb3f351df9d3b3dde5d9cf39aad2e94c39944435 || goto :error
	call :download wix-binaries.zip https://github.com/wixtoolset/wix3/releases/download/wix3112rtm/wix311-binaries.zip 2c1888d5d1dba377fc7fa14444cf556963747ff9a0a289a3599cf09da03b9e2e || goto :error
	echo [+] Extracting wix-binaries.zip
	mkdir wix\bin || goto :error
	tar -xf wix-binaries.zip -C wix\bin || goto :error
	echo [+] Cleaning up wix-binaries.zip
	del wix-binaries.zip || goto :error
	copy /y NUL prepared > NUL || goto :error
	cd .. || goto :error

:build
	set WIX=%BUILDDIR%.deps\wix\
	call :msi x86 		|| goto :error
	call :msi x86_64 	|| goto :error

:success
	echo [+] Success.
	exit /b 0

:download
	echo [+] Downloading %1
	curl -#fLo %1 %2 || exit /b 1
	echo [+] Verifying %1
	for /f %%a in ('CertUtil -hashfile %1 SHA256 ^| findstr /r "^[0-9a-f]*$"') do if not "%%a"=="%~3" exit /b 1
	goto :eof

:msi
	echo [+] Compiling %1
	"%WIX%bin\candle" WintunInstaller_%~1.wxs -o obj\ || goto error
	echo [+] Linking %1
	"%WIX%bin\light" obj\WintunInstaller_%~1.wixobj -o bin\WintunInstaller_%~1.msi || goto error
	goto :eof

:error
	echo [-] Failed with error #%errorlevel%.
	cmd /c exit %errorlevel%
