@ECHO OFF

setlocal
set SCRIPTDIR=%~dp0

if exist "%SCRIPTDIR%..\OpenVPN\obfsproxy" (
  echo [*] Erasing OpenVPN\obfsproxy\*.exe ...
  del /f /q /s "%SCRIPTDIR%..\OpenVPN\obfsproxy\*.exe"  >nul 2>&1 || exit /b 1
) else (
  mkdir "%SCRIPTDIR%..\OpenVPN\obfsproxy" || exit /b 1
)

if exist "%SCRIPTDIR%..\.deps\obfsproxy" (
  echo [*] Erasing '"%SCRIPTDIR%..\.deps\obfsproxy' ...
  rmdir /s /q "%SCRIPTDIR%..\.deps\obfsproxy" || exit /b 1
)

echo [*] Creating .deps\obfsproxy ...
mkdir "%SCRIPTDIR%..\.deps\obfsproxy" || exit /b 1

echo [*] Cloning obfs4proxy sources...
cd "%SCRIPTDIR%..\.deps\obfsproxy"
git clone https://github.com/Yawning/obfs4.git || exit /b 1

echo [*] Compiling obfs4proxy ...
cd obfs4

go build -o "%SCRIPTDIR%..\OpenVPN\obfsproxy\obfs4proxy.exe" ./obfs4proxy >nul 2>&1 || exit /b 1

echo [ ] SUCCESS
echo [ ] The compiled 'obfs4proxy.exe' binary located at:
echo [ ] "%SCRIPTDIR%..\OpenVPN\obfsproxy\obfs4proxy.exe"
