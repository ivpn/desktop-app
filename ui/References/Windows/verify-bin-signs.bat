@echo off

setlocal
setlocal EnableDelayedExpansion

set SCRIPTDIR=%~dp0
set HAS_SIGN_ERRORS=0

echo [+] Verifying signature of binaries in '%SCRIPTDIR%\bin\temp' ...
cd %SCRIPTDIR%\bin\temp
for /r %%f in (.\*.exe) do (
  rem echo Checking file: "%%f"
  signtool verify /pa "%%f"  > NUL
  IF not ERRORLEVEL 0 (
    ECHO [!] Error: failed to create installer
    EXIT /B 1
  )

  IF !ERRORLEVEL! NEQ 0 (
    set HAS_SIGN_ERRORS=1
  	echo [***ERROR***] VERIFICATION ERROR FOR FILE: '%%f'
  )
)

IF %HAS_SIGN_ERRORS% NEQ 0 (
  echo ***********************************************
	echo [***ERROR***] ERROR: Some files are not correctly signed
  echo ***********************************************
	exit /b 1
)
echo [ ] Success
exit /b 0
