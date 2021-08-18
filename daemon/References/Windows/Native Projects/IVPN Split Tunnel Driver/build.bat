@echo off
setlocal

echo [ ] The scipt is building IVPN Split-Tunnelling Driver
echo [ ] and preparing a CAB file to be send to Microsoft Partner portal for Attestation signing.
echo.
rem The scipt is building IVPN Split-Tunnelling Driver
rem and preparing a CAB file to be send to Microsoft Partner portal for Attestation signing.
rem
rem The Driver signed by Attestation signing on Microsoft Partner portal
rem will work only on Windows versions since Windows 10
rem
rem Useful links:
rem     https://docs.microsoft.com/en-us/windows-hardware/drivers/dashboard/attestation-signing-a-kernel-driver-for-public-release
rem     http://billauer.co.il/blog/2021/05/windows-drivers-attestation-signing/

rem Checking if msbuild available
WHERE msbuild >nul 2>&1
IF %ERRORLEVEL% NEQ 0 (
	echo [!] 'msbuild' is not recognized as an internal or external command
	echo [!] Ensure you are running this script from Developer Cammand Prompt for Visual Studio
	goto :error
)

if [%1]==[] goto show_usage
set CERT_SHA1=%1

set SCRIPTDIR=%~dp0

cd %SCRIPTDIR%

set RELEASE_PATH=%SCRIPTDIR%ivpn-split-tunnel\x64\Release\ivpn-split-tunnel
set DIST_PATH=%SCRIPTDIR%_out_bin\x64\Windows10
set DIST_PATH_DRV=%DIST_PATH%\drv
set DIST_PATH_OUT=%DIST_PATH%\out

rem Building driver (signing not necessary on this atage)
echo [+] Building ...
msbuild "%SCRIPTDIR%ivpn-split-tunnel\ivpn-split-tunnel.vcxproj" /p:Configuration=Release /p:Platform=x64 /p:SignMode=Off || goto :error

echo [+] Preparing files to build CAB ...
rem Erasing work directory
rmdir /S /Q "%DIST_PATH%"
mkdir "%DIST_PATH_DRV%" || goto :error
mkdir "%DIST_PATH_OUT%" || goto :error

echo [+] Copying files to build CAB ...
rem Copying required files
rem The *.cat file is not required (it will be automatically cretaded by Microsoft Partner portal after Attestation signing)
rem NOTE: The *.pdb file is not required
rem       But but Microsoft Partner portal will warn that symbols is missing during Attestation.
rem       We can ignore this warning
copy "%RELEASE_PATH%\ivpn-split-tunnel.inf" "%DIST_PATH_DRV%\ivpn-split-tunnel.inf" || goto :error
copy "%RELEASE_PATH%\ivpn-split-tunnel.sys" "%DIST_PATH_DRV%\ivpn-split-tunnel.sys" || goto :error
copy "%RELEASE_PATH%\WdfCoinstaller01009.dll" "%DIST_PATH_DRV%\WdfCoinstaller01009.dll" || goto :error
copy "%RELEASE_PATH%\..\ivpn-split-tunnel.pdb" "%DIST_PATH_DRV%\ivpn-split-tunnel.pdb" || goto :error

rem echo [+] Signing SYS file by EV Certificate ...
rem set TIMESTAMP_SERVER=http://timestamp.digicert.com
rem signtool sign /tr %TIMESTAMP_SERVER% /td sha256 /fd sha256 /sha1 %CERT_SHA1% /v "%DIST_PATH_DRV%\ivpn-split-tunnel.sys" || goto :error

echo [+] Preparing CAB file configuration...
>"%DIST_PATH_OUT%\ivpn-split-tunnel.ddf" (
  echo .Set CabinetFileCountThreshold=0
  echo .Set FolderFileCountThreshold=0
  echo .Set FolderSizeThreshold=0
  echo .Set MaxCabinetSize=0
  echo .Set MaxDiskFileCount=0
  echo .Set MaxDiskSize=0
  echo .Set CompressionType=MSZIP
  echo .Set Cabinet=on
  echo .Set Compress=on
  echo .Set CabinetNameTemplate=ivpn-split-tunnel.cab
  echo .Set DestinationDir=Package
  echo .Set DiskDirectoryTemplate="%DIST_PATH_OUT%"
  echo "%DIST_PATH_DRV%\ivpn-split-tunnel.inf"
  echo "%DIST_PATH_DRV%\ivpn-split-tunnel.sys"
  echo "%DIST_PATH_DRV%\WdfCoinstaller01009.dll"
	echo "%DIST_PATH_DRV%\ivpn-split-tunnel.pdb"
)
IF %ERRORLEVEL% NEQ 0 goto error

echo [+] Generating CAB file ...
cd "%DIST_PATH_OUT%"
makecab /f "%DIST_PATH_OUT%\ivpn-split-tunnel.ddf" || goto :error

echo [+] Signing CAB file by EV Certificate ...
rem Timestamping is not required here (since the driver will be signed by Microsoft on Microsoft Partner portal)
signtool sign /fd sha256 /sha1 %CERT_SHA1% /v "%DIST_PATH_OUT%\ivpn-split-tunnel.cab" || goto :error

echo.
echo [ ] IVPN Split-Tunnelling Driver: Build SUCCESS
echo [ ] CAB file: %DIST_PATH_OUT%\ivpn-split-tunnel.cab
echo.
echo [ ] Now you can send the CAB file to Microsoft Partner portal for Attestation signing
echo [ ]      https://partner.microsoft.com/en-US/dashboard/home
echo.
echo [ ] The signed driver files should be placed at:
echo [ ]      daemon\References\Windows\SplitTunnelDriver\x86_64
echo.

exit /b 0

:show_usage
echo Usage:  %0 ^<EV_Certificate_SHA1_hash^>

exit /b 1

:error
echo.
echo [!] FAILED building IVPN Split-Tunnelling Driver. Error #%errorlevel%.
echo.
exit /b %errorlevel%
