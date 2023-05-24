@echo off
setlocal
setlocal enabledelayedexpansion

rem %1 - first argument may specify the folder where all the build operations must be done (if it empty - use current folder)

rem Update this line if using another version of VisualStudio or it is installed in another location
set _VS_VARS_BAT="C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvarsall.bat"

rem -= when _VERSION_LIBOQS not defined - will be used latest sources from github =-
rem set _VERSION_LIBOQS=0.7.2

set _SCRIPTDIR=%~dp0
set _WORK_FOLDER=%_SCRIPTDIR%_out_windows

if "%1" == "" (
    echo [i] Work folder not defined. Using '%_WORK_FOLDER%' as default work folder
) else (
    set _WORK_FOLDER=%1
    echo [i] Using '!_WORK_FOLDER!' as default work folder
    if not exist !_WORK_FOLDER! (
        echo [!] Error: Folder does not exists: '!_WORK_FOLDER!'
        goto :error
    )
)

rem Output subfolder
set _OUT_FOLDER=%_WORK_FOLDER%\kem-helper-bin
set _OUT_FILE=%_OUT_FOLDER%\kem-helper.exe

set _LIBOQS_FOLDER=%_WORK_FOLDER%\liboqs
set _LIBOQS_SOURCES_FOLDER=%_LIBOQS_FOLDER%\liboqs
set _LIBOQS_INSTALL_FOLDER=%_LIBOQS_FOLDER%\INSTALL

call :ensure_build_environment || goto :error
call :compile_or_compile_liboqs_lib || goto :error
call :compile_binary || goto :error

:ensure_build_environment
    if not defined VSCMD_VER (
        goto :init_VS64_build_env
    )
   if "%VSCMD_ARG_TGT_ARCH%" NEQ "x64" (
        goto :init_VS64_build_env
    )    
    goto :eof
    
    :init_VS64_build_env
        echo [*] Initialising x64 VS build environment ...
            if not exist %_VS_VARS_BAT% (
                echo [!] File '%_VS_VARS_BAT%' not exists! 
                echo [!] Please install Visual Studio or update file location in '%~f0'
                goto :error
            )
            call %_VS_VARS_BAT% x64 || goto :error

    goto :eof

:compile_or_compile_liboqs_lib
    if exist "%_LIBOQS_FOLDER%" (
        echo [*] Erasing '%_LIBOQS_FOLDER%' ...
        rmdir /s /q "%_LIBOQS_FOLDER%" || goto :error
    )
    
    echo [*] Creating %_LIBOQS_FOLDER% ...
    mkdir "%_LIBOQS_FOLDER%" || goto :error

    cd %_LIBOQS_FOLDER%

    if defined _VERSION_LIBOQS (
        rem -= Downloading sources of specific version =-
        echo [*] Downloading sources of liboqs v%_VERSION_LIBOQS% ...
        curl -L -o liboqs-%_VERSION_LIBOQS%.zip https://github.com/open-quantum-safe/liboqs/archive/refs/tags/%_VERSION_LIBOQS%.zip || goto :error
        echo [*] Download complete. Extracting files...
        mkdir %_LIBOQS_SOURCES_FOLDER% || goto :error
        tar -xf liboqs-%_VERSION_LIBOQS%.zip --strip-components=1 -C %_LIBOQS_SOURCES_FOLDER% || goto :error
    ) else (
        rem -= Getting latest sources from git repo =-
        echo [*] Cloning liboqs sources...    
        git clone --depth 1 https://github.com/open-quantum-safe/liboqs.git || exit /b 1        
    )
    cd liboqs || goto :error

    rem -= Configure & Build =-
    echo [*] liboqs: Configuring ...
    mkdir build && cd build

    cmake -GNinja .. ^
        -DOQS_MINIMAL_BUILD="KEM_kyber_1024;KEM_classic_mceliece_348864;" ^
        -DCMAKE_BUILD_TYPE=Release ^
        -DCMAKE_INSTALL_PREFIX=%_LIBOQS_INSTALL_FOLDER% ^
        -DOQS_BUILD_ONLY_LIB=ON ^
        -DBUILD_SHARED_LIBS=OFF ^
        -DOQS_USE_OPENSSL=OFF ^
        -DOQS_DIST_BUILD=ON ^
        -DOQS_USE_CPUFEATURE_INSTRUCTIONS=OFF ^
		^	|| goto :error	

    echo [*] liboqs: Compiling ...
    ninja || goto :error
    echo [*] liboqs: Installing ...
    ninja install || goto :error
    
    goto :eof

:compile_binary
    if not exist %_OUT_FOLDER% (
        echo [*] Creating folder '%_OUT_FOLDER%' ...        
        mkdir "%_OUT_FOLDER%" || goto :error        
    ) else (
        echo [*] Erasing '%_OUT_FOLDER%\*' ...
        del /Q %_OUT_FOLDER%\*
    )
    echo Sources '%_SCRIPTDIR%' > %_OUT_FOLDER%\readme.md

    rem Change the current working directory to the location of the source files
    cd %_SCRIPTDIR%

    echo [*] Compiling (%_SCRIPTDIR%)...
    rem The 'Classic-McEliece' consuming a lot of stack, so we specifying stack size manually: "/STACK:5242880"
    cl.exe main.c base64.c /nologo /DWIN32 /D_WINDOWS /W3 /MT /O2 /Ob2 /DNDEBUG  /I "%_LIBOQS_INSTALL_FOLDER%\include" /Fo"%_OUT_FOLDER%/" /link /STACK:5242880 /LIBPATH:"%_LIBOQS_INSTALL_FOLDER%\lib" oqs.lib Advapi32.lib /OUT:"%_OUT_FILE%" || goto :error    
    
    dumpbin /headers "%_OUT_FILE%" | findstr /i "machine.*x64" >nul
    if not %errorlevel% equ 0 (
        echo ERROR: Binary "%_OUT_FILE%" is not compiled for x64 architecture
        goto :error    
    ) 

    goto :success

:success
	echo [ ] Done. Binary location: '%_OUT_FILE%'
	exit /b 0

:error
	set ERR=%errorlevel%
    if %ERR% == 0 (
        echo [!] FAILED
	    exit /b 1    
    )
	echo [!] FAILED with error #%ERR%.    
	exit /b %ERR%