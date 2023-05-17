@echo off

setlocal

set _SCRIPTDIR=%~dp0

echo ### Buildind KEM-helper binaries  ###

PUSHD
call %_SCRIPTDIR%..\..\common\kem-helper\build.bat %_SCRIPTDIR%..\.deps || goto :error
POPD

SET _KEM_BIN_DIR=%SCRIPTDIR%..\kem
    if exist "%_KEM_BIN_DIR%" (
        echo [*] Deleting '%_KEM_BIN_DIR%\*' ...
        rmdir /s /q "%_KEM_BIN_DIR%"
    )

mkdir "%_KEM_BIN_DIR%" || goto :error

copy /Y %_SCRIPTDIR%..\.deps\kem-helper-bin\kem-helper.exe  %_SCRIPTDIR%..\kem || goto :error

set _theResult_binary_path=%_SCRIPTDIR%..\kem-helper.exe
for %%i in (%_theResult_binary_path%) do set _theResult_binary_path=%%~fi
echo [ ] RESULT BINARY:  %_theResult_binary_path%

:success
	echo [*] Success.    
	exit /b 0

:error
	set ERR=%errorlevel%
    if %ERR% == 0 (
        echo [!] FAILED
	    exit /b 1    
    )
	echo [!] FAILED with error #%ERR%.    
	exit /b %ERR%