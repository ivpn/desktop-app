@ECHO OFF

setlocal

rem TODO: define here version to build
set _VERSION=v5.7.0

set SCRIPTDIR=%~dp0

if exist "%SCRIPTDIR%..\v2ray" (
  echo [*] Erasing v2ray\*.exe ...
  del /f /q /s "%SCRIPTDIR%..\v2ray\*.exe"  >nul 2>&1 || exit /b 1
) else (
  mkdir "%SCRIPTDIR%..\v2ray" || exit /b 1
)

if exist "%SCRIPTDIR%..\.deps\v2ray" (
  echo [*] Erasing '"%SCRIPTDIR%..\.deps\v2ray' ...
  rmdir /s /q "%SCRIPTDIR%..\.deps\v2ray" || exit /b 1
)

echo [*] Creating .deps\v2ray ...
mkdir "%SCRIPTDIR%..\.deps\v2ray" || exit /b 1

echo [*] Cloning V2Ray sources...
cd "%SCRIPTDIR%..\.deps\v2ray"
git clone  --depth 1 --branch %_VERSION% https://github.com/v2fly/v2ray-core.git || exit /b 1
cd v2ray-core/main

echo [*] Compiling V2Ray ...

go build -o "%SCRIPTDIR%..\v2ray\v2ray.exe" -trimpath -ldflags "-s -w" >nul 2>&1 || exit /b 1

echo [ ] SUCCESS
echo [ ] The compiled 'v2ray.exe' binary located at:
echo [ ] "%SCRIPTDIR%..\v2ray\v2ray.exe"
