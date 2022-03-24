@ECHO OFF

setlocal

rem TODO: define here dnscrypt-roxy version to build
set _VERSION=2.1.1

set SCRIPTDIR=%~dp0

if exist "%SCRIPTDIR%..\dnscrypt-proxy" (
  echo [*] Erasing dnscrypt-proxy\*.exe ...
  del /f /q /s "%SCRIPTDIR%..\dnscrypt-proxy\*.exe"  >nul 2>&1 || exit /b 1
) else (
  mkdir "%SCRIPTDIR%..\dnscrypt-proxy" || exit /b 1
)

if exist "%SCRIPTDIR%..\.deps\dnscrypt-proxy" (
  echo [*] Erasing '"%SCRIPTDIR%..\.deps\dnscrypt-proxy' ...
  rmdir /s /q "%SCRIPTDIR%..\.deps\dnscrypt-proxy" || exit /b 1
)

echo [*] Creating .deps\dnscrypt-proxy ...
mkdir "%SCRIPTDIR%..\.deps\dnscrypt-proxy" || exit /b 1

echo [*] Cloning dnscrypt-proxy sources...
cd "%SCRIPTDIR%..\.deps\dnscrypt-proxy"
git clone https://github.com/DNSCrypt/dnscrypt-proxy.git || exit /b 1
cd dnscrypt-proxy

echo [*] Checkout version ${_VERSION} of 'dnscrypt-proxy'..."
git checkout tags/%_VERSION%

echo [*] Compiling dnscrypt-proxy ...

go build -o "%SCRIPTDIR%..\dnscrypt-proxy\dnscrypt-proxy.exe" -trimpath -ldflags "-s -w" ./dnscrypt-proxy >nul 2>&1 || exit /b 1

echo [ ] SUCCESS
echo [ ] The compiled 'obfs4proxy.exe' binary located at:
echo [ ] "%SCRIPTDIR%..\dnscrypt-proxy\dnscrypt-proxy.exe"
