@ECHO OFF

setlocal
set SCRIPTDIR=%~dp0
set WGVER=v0.4.9

echo ### Buildind WireGuard binaries  ###

if exist "%SCRIPTDIR%..\WireGuard\x86_64" (
  echo [*] Erasing WireGuard\x86_64\* ...
  del /f /q /s "%SCRIPTDIR%..\WireGuard\x86_64\*" >nul 2>&1 || exit /b 1
)

if not exist "%SCRIPTDIR%..\.deps\wireguard-windows\.deps\prepared" (
  if not exist "%SCRIPTDIR%..\.deps" (
    echo [*] Creating .deps ...
    mkdir "%SCRIPTDIR%..\.deps" || exit /b 1
    cd "%SCRIPTDIR%..\.deps" 	|| exit /b 1
  )

  if exist "%SCRIPTDIR%..\.deps\wireguard-windows" (
    echo [*] Erasing .deps ...
    rd /s /q "%SCRIPTDIR%..\.deps\wireguard-windows" || exit /b 1
    sleep 2
  )

  cd "%SCRIPTDIR%..\.deps"

  echo [*] Cloning wireguard-windows...
  git clone https://git.zx2c4.com/wireguard-windows || exit /b 1
  cd wireguard-windows || exit /b 1

  echo [*] Checking out wireguard-windows version [%WGVER%]...
  git checkout %WGVER% >nul 2>&1 || exit /b 1
    echo [*] Building wireguard-windows from NEW sources...
) else (
  echo [*] Building wireguard-windows from ALREADY DOWNLOADED sources...
  cd "%SCRIPTDIR%..\.deps\wireguard-windows" 	|| exit /b 1
)

call build.bat
if not %errorlevel%==0 (
    echo [!] ERROR: Building WireGuard from official sources
    echo [ ]        You can skip building WireGuard binaries.
    echo [ ]        To skip build, copy correspond precompiled official WireGuard binaries to locations:
    echo [ ]        	%SCRIPTDIR%..\WireGuard\x86_64\wg.exe
    echo [ ]        	%SCRIPTDIR%..\WireGuard\x86_64\wireguard.exe
    exit /b 1
)

echo [*] WireGuard build DONE. Copying compiled binaries ...

if not exist "%SCRIPTDIR%..\WireGuard" 			mkdir "%SCRIPTDIR%..\WireGuard" 		|| exit /b 1
if not exist "%SCRIPTDIR%..\WireGuard\x86_64" 	mkdir "%SCRIPTDIR%..\WireGuard\x86_64"	|| exit /b 1

copy /y "%SCRIPTDIR%..\.deps/wireguard-windows\amd64\wg.exe" 		"%SCRIPTDIR%..\WireGuard\x86_64\wg.exe" 		>nul 2>&1 || exit /b 1
copy /y "%SCRIPTDIR%..\.deps\wireguard-windows\amd64\wireguard.exe" "%SCRIPTDIR%..\WireGuard\x86_64\wireguard.exe" 	>nul 2>&1 || exit /b 1
