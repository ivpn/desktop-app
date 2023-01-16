@echo off

setlocal

set FILE_TO_VERIFY=%1
set EXPECTED_SHA256=%2

rem echo File           : %FILE_TO_VERIFY%
rem echo Expected SHA256: %EXPECTED_SHA256%

if "%FILE_TO_VERIFY%" == "" (
	echo [!] Error: no arguments defined [FILE_TO_VERIFY]
	exit /b 1
)
if "%EXPECTED_SHA256%" == "" (
	echo [!] Error: no arguments defined [EXPECTED_SHA256]
	exit /b 1
)

set "SHASUM=" & for /F "skip=1 delims=" %%H in ('
    CertUtil -hashfile %FILE_TO_VERIFY% SHA256
') do if not defined SHASUM set "SHASUM=%%H"

if "%SHASUM%" NEQ "%EXPECTED_SHA256%" (
	echo [!] ERROR File checksum verification %FILE_TO_VERIFY% : %SHASUM% [expected=%EXPECTED_SHA256%]
	exit /b 1	
) 

echo [ ] Checksum is OK ['%FILE_TO_VERIFY%' : %SHASUM%]
exit /b 0