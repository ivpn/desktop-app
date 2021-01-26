; IVPN Client Installer
; Install script for NSIS 2.x

SetCompressor lzma

; -----------------
; include Modern UI
; -----------------

!include "MUI.nsh"
!include "LogicLib.nsh"
!include "StrFunc.nsh"
!include "x64.nsh"
!include "WinVer.nsh"
; include for some of the windows messages defines
!include "winmessages.nsh"

${StrLoc}

; -------
; general
; -------

; SOURCE_DIR is defined in build.bat

!define PRODUCT_NAME "IVPN Client"
!define PRODUCT_IDENTIFIER "IVPN Client"
!define PRODUCT_PUBLISHER "IVPN Limited"

!define APP_RUN_PATH "$INSTDIR\ui\IVPN Client.exe"
!define PROCESS_NAME "IVPN Client.exe"
!define IVPN_SERVICE_NAME "IVPN Client"
!define PATHDIR "$INSTDIR\cli"

; The following variables will be set from the build.bat script
; !define PRODUCT_VERSION "2.0-b4"
; !define OUT_FILE "bin\${PRODUCT_NAME} ${PRODUCT_VERSION}.exe"

Name "${PRODUCT_NAME}"
OutFile "${OUT_FILE}"
InstallDir "$PROGRAMFILES64\${PRODUCT_IDENTIFIER}"
;InstallDirRegKey HKLM "Software\${PRODUCT_IDENTIFIER}" ""
RequestExecutionLevel admin


; HKLM (all users)
!define env_hklm 'HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"'
; HKCU (current user)
!define env_hkcu 'HKCU "Environment"'

; ---------
; variables
; ---------

var /GLOBAL StartMenuFolder
var /GLOBAL BitDir

Var HEADLINE_FONT

;---------------------------
; StrContains
; This function does a case sensitive searches for an occurrence of a substring in a string.
; It returns the substring if it is found.
; Otherwise it returns null("").
Var STR_HAYSTACK
Var STR_NEEDLE
Var STR_CONTAINS_VAR_1
Var STR_CONTAINS_VAR_2
Var STR_CONTAINS_VAR_3
Var STR_CONTAINS_VAR_4
Var STR_RETURN_VAR

!define StrContains '!insertmacro "StrContains"'
!macro StrContains OUT NEEDLE HAYSTACK
  Push `${HAYSTACK}`
  Push `${NEEDLE}`
  !ifdef __UNINSTALL__
      Call un.StrContains
  !else
      Call StrContains
  !endif
  Pop `${OUT}`
!macroend

!macro Func_StrContains un
  Function ${un}StrContains
    Exch $STR_NEEDLE
    Exch 1
    Exch $STR_HAYSTACK
    ; Uncomment to debug
    ; MessageBox MB_OK 'STR_NEEDLE = $STR_NEEDLE STR_HAYSTACK = $STR_HAYSTACK '
      StrCpy $STR_RETURN_VAR ""
      StrCpy $STR_CONTAINS_VAR_1 -1
      StrLen $STR_CONTAINS_VAR_2 $STR_NEEDLE
      StrLen $STR_CONTAINS_VAR_4 $STR_HAYSTACK
      loop:
        IntOp $STR_CONTAINS_VAR_1 $STR_CONTAINS_VAR_1 + 1
        StrCpy $STR_CONTAINS_VAR_3 $STR_HAYSTACK $STR_CONTAINS_VAR_2 $STR_CONTAINS_VAR_1
        StrCmp $STR_CONTAINS_VAR_3 $STR_NEEDLE found
        StrCmp $STR_CONTAINS_VAR_1 $STR_CONTAINS_VAR_4 done
        Goto loop
      found:
        StrCpy $STR_RETURN_VAR $STR_NEEDLE
        Goto done
      done:
     Pop $STR_NEEDLE ;Prevent "invalid opcode" errors and keep the
     Exch $STR_RETURN_VAR
  FunctionEnd
!macroend
!insertmacro Func_StrContains ""
!insertmacro Func_StrContains "un."

;---------------------------
!define StrRepl "!insertmacro StrRepl"
!macro StrRepl output string old new
    Push `${string}`
    Push `${old}`
    Push `${new}`
    !ifdef __UNINSTALL__
        Call un.StrRepl
    !else
        Call StrRepl
    !endif
    Pop ${output}
!macroend

!macro Func_StrRepl un
    Function ${un}StrRepl
        Exch $R2 ;new
        Exch 1
        Exch $R1 ;old
        Exch 2
        Exch $R0 ;string
        Push $R3
        Push $R4
        Push $R5
        Push $R6
        Push $R7
        Push $R8
        Push $R9

        StrCpy $R3 0
        StrLen $R4 $R1
        StrLen $R6 $R0
        StrLen $R9 $R2
        loop:
            StrCpy $R5 $R0 $R4 $R3
            StrCmp $R5 $R1 found
            StrCmp $R3 $R6 done
            IntOp $R3 $R3 + 1 ;move offset by 1 to check the next character
            Goto loop
        found:
            StrCpy $R5 $R0 $R3
            IntOp $R8 $R3 + $R4
            StrCpy $R7 $R0 "" $R8
            StrCpy $R0 $R5$R2$R7
            StrLen $R6 $R0
            IntOp $R3 $R3 + $R9 ;move offset by length of the replacement string
            Goto loop
        done:

        Pop $R9
        Pop $R8
        Pop $R7
        Pop $R6
        Pop $R5
        Pop $R4
        Pop $R3
        Push $R0
        Push $R1
        Pop $R0
        Pop $R1
        Pop $R0
        Pop $R2
        Exch $R1
    FunctionEnd
!macroend
!insertmacro Func_StrRepl ""
!insertmacro Func_StrRepl "un."
;---------------------------

!macro COMMON_INIT
  ; install for  'all users'
  SetShellVarContext all

  SetRegView 64
  StrCpy $BitDir "x86_64"
  StrCpy $StartMenuFolder "IVPN"
  DetailPrint "Running on architecture: $BitDir"
!macroend

Function .onInit
  !insertmacro COMMON_INIT

  CreateFont $HEADLINE_FONT "$(^Font)" "12" "600"

  Call CheckOSSupported

  ClearErrors

  ; hack to not prompt for last 2.12.x releases
  ; It is required for easy migration from 2.x to 3.x version (do not perform logout)
  ReadRegStr $R1 HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\IVPN Client" "DisplayVersion"
  ${StrLoc} $R2 $R1 "2.12." ">"
  StrCmp $R2 "0" done ; R2 must be 0 if upgrading from '2.12.X' version

  ReadRegStr $R0 HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\IVPN Client" "UninstallString"
  StrCmp $R0 "" done
  IfSilent uninst is_not_quiet
is_not_quiet:
  MessageBox MB_OKCANCEL|MB_ICONEXCLAMATION "${PRODUCT_NAME} is already installed.$\n$\nClick OK to uninstall the old version." IDOK uninst
  Abort
uninst:
  ExecWait '$R0 -update'
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\IVPN Client"
done:

FunctionEnd

Function un.onInit
  !insertmacro COMMON_INIT
FunctionEnd

; --------------
; user interface
; --------------
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_RIGHT
!define MUI_HEADERIMAGE_BITMAP "header.bmp"

!define MUI_ICON "application.ico"
!define MUI_UNICON "application.ico"

!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_FINISHPAGE_RUN "$INSTDIR\IVPN Client.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Run ${PRODUCT_NAME} now"
!define MUI_FINISHPAGE_RUN_FUNCTION ExecAppFile

; Checkbox on finish page: create shortcut on desktop
; using unused 'readme' check box for this
!define MUI_FINISHPAGE_SHOWREADME ""
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Create a desktop shortcut"
!define MUI_FINISHPAGE_SHOWREADME_FUNCTION finishpageaction
Function finishpageaction
CreateShortcut "$DESKTOP\IVPN Client.lnk" "${APP_RUN_PATH}"
FunctionEnd

LicenseForceSelection checkbox "I Agree"

!define MUI_STARTMENUPAGE_REGISTRY_ROOT "HKLM"
!define MUI_STARTMENUPAGE_REGISTRY_KEY "Software\${PRODUCT_IDENTIFIER}"
!define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "Start Menu Folder"

!define MUI_WELCOMEPAGE_TITLE "Welcome to the ${PRODUCT_NAME} v.${PRODUCT_VERSION} Setup Wizard"

!insertmacro MUI_DEFAULT MUI_WELCOMEFINISHPAGE_BITMAP "startfinish.bmp"
!insertmacro MUI_DEFAULT MUI_UNWELCOMEFINISHPAGE_BITMAP "startfinish.bmp"

!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE License.txt
;!insertmacro MUI_PAGE_STARTMENU Application $StartMenuFolder
!insertmacro MUI_PAGE_INSTFILES

;===============================
; FINISH page modification
!define MUI_PAGE_CUSTOMFUNCTION_PRE fin_pre
!define MUI_PAGE_CUSTOMFUNCTION_SHOW fin_show
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE fin_leave
;===============================
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

;===============================
; FINISH page modification handlers (add additional checkbox "Add IVPN CLI binary to the path" to the 'finish' page)
Function fin_show
	ReadINIStr $0 "$PLUGINSDIR\iospecial.ini" "Field 6" "HWND"
	SetCtlColors $0 0x000000 0xFFFFFF
FunctionEnd

Function fin_pre
	WriteINIStr "$PLUGINSDIR\iospecial.ini" "Settings" "NumFields" "6"
	WriteINIStr "$PLUGINSDIR\iospecial.ini" "Field 6" "Type" "CheckBox"
	WriteINIStr "$PLUGINSDIR\iospecial.ini" "Field 6" "Text" "Add IVPN CLI binary to the path"
	WriteINIStr "$PLUGINSDIR\iospecial.ini" "Field 6" "Left" "120"
	WriteINIStr "$PLUGINSDIR\iospecial.ini" "Field 6" "Right" "315"
	WriteINIStr "$PLUGINSDIR\iospecial.ini" "Field 6" "Top" "130"
	WriteINIStr "$PLUGINSDIR\iospecial.ini" "Field 6" "Bottom" "140"
	WriteINIStr "$PLUGINSDIR\iospecial.ini" "Field 6" "State" "0"
FunctionEnd

Function fin_leave
	ReadINIStr $0 "$PLUGINSDIR\iospecial.ini" "Field 6" "State"
	StrCmp $0 "0" end

	; UPDATING %PATH% VARIABLE
	ReadRegStr $0 ${env_hklm} "PATH"

	; check if PATH already updated
	${StrContains} $1 "${PATHDIR}" $0
	StrCmp $1 "${PATHDIR}" end ; do nothing

	; remove last symbol ';' from %PATH% (if exists)
	StrCpy $2 $0 "" -1
	StrCmp $2 ";" 0 +2
	StrCpy $0 $0 -1

	; set variable for local machine
	StrCpy $0 "$0;${PATHDIR}"
	WriteRegExpandStr ${env_hklm} PATH "$0"

	; make sure windows knows about the change
	SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=100

	end:
FunctionEnd
;===============================

; ------------------
; installer sections
; ------------------

!define DEVCON_BASENAME "devcon.exe"
!define PRODUCT_TAP_WIN_COMPONENT_ID "tapivpn"

Section "${PRODUCT_NAME}" SecIVPN
  SetRegView 64
  SetOutPath "$INSTDIR"

  ; Stop IVPN service
  stopservcice:
  Call StopService
  Pop $0 ; 1 - SUCCESS;
  ${if} $0 != 1
		DetailPrint "ERROR: Failed to stop 'IVPN Client' service."
		MessageBox MB_ABORTRETRYIGNORE|MB_ICONEXCLAMATION "Failed to stop 'IVPN Client' service.$\nIgnoring this problem can lead to issues with IVPN Client software in the future." IDRETRY stopservcice IDIGNORE ignoreservicestop
		DetailPrint "Aborted"
		Abort
  ${EndIf}
  ignoreservicestop:

  ; When service stopping - IVPN Client must also Close automatically
  ; anyway, there could be situations when IVPN Client not connected to service (cannot receive 'service exiting' notification.)
  ; Therefore, here we try to stop IVPN Client process manually.
  ; Stop IVPN Client application
  stopclient:
  Call StopClient
  Pop $0 ; 1 - SUCCESS
  ${if} $0 != 1
		DetailPrint "ERROR: Failed to stop 'IVPN Client' application."
		MessageBox MB_ABORTRETRYIGNORE|MB_ICONEXCLAMATION "Failed to stop 'IVPN Client' application.$\nIgnoring this problem can lead to issues with IVPN Client software in the future." IDRETRY stopclient IDIGNORE ignoreclientstop
		DetailPrint "Aborted"
		Abort
  ${EndIf}
  ignoreclientstop:

  ; check is library can be overwritten
  Push "$INSTDIR\IVPN Firewall Native x64.dll" ; file to check for writting
  Push 15000 ; 15 seconds
  Call WaitFileOpenForWritting

  ; check is library can be overwritten
  Push "$INSTDIR\IVPN Helpers Native x64.dll" ; file to check for writting
  Push 15000 ; 15 seconds
  Call WaitFileOpenForWritting

  ; hack to not prompt for last 2.12.x releases
  ; It is required for easy migration from 2.x to 3.x version (do not perform logout)
  ReadRegStr $R1 HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\IVPN Client" "DisplayVersion"
  ${StrLoc} $R2 $R1 "2.12." ">"
  ${If} $R2 == 0 ; R2 must be 0 if upgrading from '2.12.X' version
    ; Remove files from old installations
    ; TODO: not required if upgarding from v3.x.x
    ; (only necessay for v2.12.x because uninstaller for old versions does not support '-update' argument)
    DetailPrint "Removing files from previous installation 2.12.x ..."
    Delete "$DESKTOP\IVPN Client.lnk"
    Delete "$INSTDIR\*.*"
    RMDir /r "$INSTDIR\OpenVPN"
    RMDir /r "$INSTDIR\WireGuard"
  ${EndIf}

  ; extract all files from source dir (it is important that IVPN Client Application must be stopped on this moment)
  File /r "${SOURCE_DIR}\*.*"

  CreateDirectory "$INSTDIR\log"

  WriteRegStr HKLM "Software\${PRODUCT_IDENTIFIER}" "" $INSTDIR
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_IDENTIFIER}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegExpandStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_IDENTIFIER}" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_IDENTIFIER}" "DisplayIcon" "$INSTDIR\icon.ico"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_IDENTIFIER}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_IDENTIFIER}" "Publisher" "${PRODUCT_PUBLISHER}"

  ; create StartMenu shortcuts
  CreateDirectory "$SMPROGRAMS\$StartMenuFolder"
  CreateShortCut "$SMPROGRAMS\$StartMenuFolder\Uninstall ${PRODUCT_NAME}.lnk" "$INSTDIR\Uninstall.exe"
  CreateShortCut "$SMPROGRAMS\$StartMenuFolder\${PRODUCT_NAME}.lnk" "$INSTDIR\ui\IVPN Client.exe"

  Call CheckIsWin7DriverInstalled

  ; ============ TAP driver ======================================================================
  DetailPrint "Installing TAP driver..."

  ; check if TUN/TAP driver is installed
  IntOp $R5 0 & 0
  nsExec::ExecToStack '"$INSTDIR\devcon\$BitDir\${DEVCON_BASENAME}" hwids ${PRODUCT_TAP_WIN_COMPONENT_ID}'
  Pop $R0 # return value/error/timeout
  IntOp $R5 $R5 | $R0
  DetailPrint "${DEVCON_BASENAME} hwids returned: $R0"

  ; if output contains the component id, then it's installed already
  Push "${PRODUCT_TAP_WIN_COMPONENT_ID}"
  Push ">"
  Call StrLoc
  Pop $R0

  ; if it's installed, do an update
  ${If} $R5 == 0
    ${If} $R0 == ""
      StrCpy $R1 "install"
    ${Else}
      StrCpy $R1 "update"
    ${EndIf}

    DetailPrint "TAP $R1 (${PRODUCT_TAP_WIN_COMPONENT_ID}) (May require confirmation)"
    nsExec::ExecToLog '"$INSTDIR\devcon\$BitDir\${DEVCON_BASENAME}" $R1 "$INSTDIR\OpenVPN\$BitDir\tap\OemVista.inf" ${PRODUCT_TAP_WIN_COMPONENT_ID}'
    Pop $R0 # return value/error/timeout

    ${If} $R0 == ""
      IntOp $R0 0 & 0
      SetRebootFlag true
      DetailPrint "REBOOT flag set"
    ${EndIf}

    IntOp $R5 $R5 | $R0
    DetailPrint "${DEVCON_BASENAME} returned: $R0"
  ${EndIf}

  DetailPrint "${DEVCON_BASENAME} cumulative status: $R5"

  ${If} $R5 != 0
    MessageBox MB_OK "An error occurred installing the TAP device driver."
    Abort
  ${EndIf}

  ; ============ Service ======================================================================
  ; install service
  DetailPrint "Installing IVPN Client service..."
  nsExec::ExecToLog '"$SYSDIR\sc.exe" create "IVPN Client" binPath= "\"$INSTDIR\IVPN Service.exe\"" start= auto'
  nsExec::ExecToLog '"$SYSDIR\sc.exe" sdset "IVPN Client" "D:(A;;CCLCSWRPWPDTLOCRRC;;;SY)(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;BA)(A;;CCLCSWLOCRRC;;;IU)(A;;CCLCSWLOCRRC;;;SU)(A;;RPWPDTLO;;;S-1-1-0)"'

  ; add service to firewall
  ;nsExec::ExecToLog '"$SYSDIR\netsh.exe" firewall add allowedprogram "$INSTDIR\IVPN Service.exe" "IVPN Service" ENABLE'

  ; start service
  DetailPrint "Starting IVPN Client service..."
  nsExec::ExecToLog '"$SYSDIR\sc.exe" start "IVPN Client"'
SectionEnd

; -----------
; uninstaller
; -----------

Section "Uninstall"
  SetRegView 64
  DetailPrint "Ensure firewall is disabled..."
  nsExec::ExecToLog '"${PATHDIR}\ivpn.exe" firewall -off'
  DetailPrint "Ensure VPN is disconnected..."
  nsExec::ExecToLog '"${PATHDIR}\ivpn.exe" disconnect'

  ${StrContains} $0 " -update" $CMDLINE
  ${If} $0 == ""
      ; uninstall
      DetailPrint "Logout..."
      nsExec::ExecToLog '"${PATHDIR}\ivpn.exe" logout'
  ${Else}
      ; update
  ${EndIf}

  ; stop service
  nsExec::ExecToLog '"$SYSDIR\sc.exe" stop "${IVPN_SERVICE_NAME}"'

  ; wait a little (give change for IVPN Client application to stop)
  Sleep 1500
  ; When service stopping - IVPN Client must also Close automatically
  ; anyway, there could be situations when IVPN Client not connected to service (cannot receive 'service exiting' notification.)
  ; Therefore, here we try to stop IVPN Client process manually.
  nsExec::ExecToStack '"$SYSDIR\taskkill" /IM "${PROCESS_NAME}" /T /F'
  ; give some time to stop the process
  Sleep 1500

  ; remove service
  nsExec::ExecToLog '"$SYSDIR\sc.exe" delete "IVPN Client"'

  ; removing firewall rules
  nsExec::ExecToLog '"$INSTDIR\ivpncli.exe" firewall disable'

  ; uninstall TUN/TAP driver
  DetailPrint "Removing TUN/TAP device..."

  nsExec::ExecToLog '"$INSTDIR\devcon\$BitDir\${DEVCON_BASENAME}" remove ${PRODUCT_TAP_WIN_COMPONENT_ID}'
  Pop $R0 # return value/error/timeout
  DetailPrint "${DEVCON_BASENAME} remove returned: $R0"

  DetailPrint "Removing files..."

  ; remove all
  Delete "$DESKTOP\IVPN Client.lnk"
  RMDir /r "$INSTDIR\mutable"
  RMDir /r "$INSTDIR\log"
  RMDir /r "$INSTDIR\devcon"
  RMDir /r "$INSTDIR\OpenVPN"
  RMDir /r "$INSTDIR\WireGuard"
  RMDir /r "$INSTDIR\cli"
  RMDir /r "$INSTDIR\ui"

  Delete "$INSTDIR\*.*"

  ${StrContains} $0 " -update" $CMDLINE
  ${If} $0 == ""
      ; uninstall
      RMDir /r "$INSTDIR\etc"
      RMDir "$INSTDIR"
  ${Else}
      ; update
  ${EndIf}



  SetShellVarContext current ; To be able to get environment variables of current user ("$LOCALAPPDATA", "$APPDATA")
  RMDir /r "$APPDATA\ivpn-ui"
  SetShellVarContext all
  RMDir /r "$APPDATA\ivpn-ui"

  ;!insertmacro MUI_STARTMENU_GETFOLDER Application $StartMenuFolder
  StrCpy $StartMenuFolder "IVPN"

  Delete "$SMPROGRAMS\$StartMenuFolder\Uninstall ${PRODUCT_NAME}.lnk"
  Delete "$SMPROGRAMS\$StartMenuFolder\${PRODUCT_NAME}.lnk"
  RMDir "$SMPROGRAMS\$StartMenuFolder"
  DeleteRegKey /ifempty HKLM "Software\${PRODUCT_IDENTIFIER}"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_IDENTIFIER}"

  ; UPDATING %PATH% VARIABLE
  ; read PATH variable value (current user)
  ReadRegStr $0 ${env_hkcu} "PATH"
  ; remove all references to $INSTDIR
  ${StrRepl} $1 $0 "${PATHDIR};" ""
  ${StrRepl} $1 $1 ";${PATHDIR}" ""
  ${StrRepl} $1 $1 ${PATHDIR} ""

  ; read PATH variable value (all users)
  ReadRegStr $0 ${env_hklm} "PATH"
  ; remove all references to $INSTDIR
  ${StrRepl} $1 $0 "${PATHDIR};" ""
  ${StrRepl} $1 $1 ";${PATHDIR}" ""
  ${StrRepl} $1 $1 ${PATHDIR} ""
  ${If} $1 != $0
  	WriteRegExpandStr ${env_hklm} PATH "$1"
  	; make sure windows knows about the change
  	SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=100
  ${EndIf}

SectionEnd

; ----------------
; helper functions
; ----------------

Function CheckOSSupported
    ${If} ${AtLeastWin7}
        goto archcheck
    ${EndIf}
    MessageBox MB_ICONSTOP|MB_OK "Unsupported Windows Version.$\nThis version of IVPN Client can only be installed on Windows 7 and above."
    Quit
archcheck:
    ${If} ${RunningX64}
        goto end
    ${EndIf}
    MessageBox MB_ICONSTOP|MB_OK "Unsupported architecture.$\nThis version of IVPN Client can only be installed on 64-bit Windows."
    Quit
end:
FunctionEnd

; Return values:
;	<0 - Error
;	0 - NOT STOPPED
; 	1 - Stopped (SECCUSS)
Function StopService
	DetailPrint "Checking is IVPN Client service is running..."
	Call IsServiceStopped
	Pop $0
	${If} $0 == 1
		Push 1 ; Stopped OK
		Return
	${EndIf}

	DetailPrint "Stopping IVPN Client service..."

	; stop service
	nsExec::ExecToStack '"$SYSDIR\sc.exe" stop "${IVPN_SERVICE_NAME}"'
	Pop $0 ; Return
	Pop $1 ; Output
	${If} $0 == '1060'
		DetailPrint "IVPN Client service does not exist as an installed service [1060]"
		Push 1 		; Stopped OK
		Return
	${EndIf}
	${If} $0 != '0'
		DetailPrint "Failed to execute 'sc stop' command: $0; $1"
		Goto killservice
	${EndIf}

	; R1 - counter
	StrCpy	$R1 0
	; waiting to stop 8 seconds (500ms*16)
	${While} $R1 < 16
		Sleep 500
		IntOp $R1 $R1 + 1

		Call IsServiceStopped
		Pop $0
		${If} $0 < 0
			Goto killservice
		${EndIf}
		${If} $0 == 1
			Push 1 ; stooped OK
			Return
		${EndIf}

	${EndWhile}

	killservice:
	; if we still here - service still not stopped. Killing it manually
	DetailPrint "WARNING: Unable to stop service. Killing process ..."
	nsExec::ExecToStack '"$SYSDIR\taskkill" /fi "Services eq ${IVPN_SERVICE_NAME}" /F'
	Pop $0 ; Return
	Pop $1 ; Output
	${If} $0 < 0
		DetailPrint "Failed to execute 'taskkill' command: $0; $1"
		Push -1 ; Error
		Return
	${EndIf}

	Sleep 500

	Call IsServiceStopped
	Pop $0
	${If} $0 < 0
		Push -1 ; Error
		Return
	${EndIf}
	${If} $0 == 1
		Push 1 ; stooped OK
		Return
	${EndIf}

	Push 0 ; if we are here, service is NOT STOPPED
FunctionEnd

Function IsServiceStopped
	nsExec::ExecToStack '"$SYSDIR\sc.exe" query "${IVPN_SERVICE_NAME}"'
	Pop $0 ; Return
	Pop $1 ; Output
	${If} $0 == '1060'
		DetailPrint "IVPN Client service does not exist as an installed service [1060]"
		Push 1 		; Stopped OK
		Return
	${EndIf}
	${If} $0 != '0'
		DetailPrint "Failed to execute 'sc query' command: $0; $1"
		Push -1 ; Error
		Return
	${EndIf}

	; An example of an expected result:
	; 	SERVICE_NAME: IVPN Client
    ;    TYPE               : 10  WIN32_OWN_PROCESS
    ;    STATE              : 4  RUNNING
    ;                            (STOPPABLE, NOT_PAUSABLE, ACCEPTS_SHUTDOWN)
    ;    WIN32_EXIT_CODE    : 0  (0x0)
    ;    SERVICE_EXIT_CODE  : 0  (0x0)
    ;    CHECKPOINT         : 0x0
    ;    WAIT_HINT          : 0x0

	; Another example:
	;	SERVICE_NAME: [service_name]
    ;    TYPE               : 10  WIN32_OWN_PROCESS
    ;    STATE              : 1  STOPPED
    ;    WIN32_EXIT_CODE    : 0  (0x0)
    ;    SERVICE_EXIT_CODE  : 0  (0x0)
    ;    CHECKPOINT         : 0x0
    ;    WAIT_HINT          : 0x0

	${StrContains} $0 "STOPPED" $1
	${If} $0 == "STOPPED"
		Push 1 		; Stopped OK
		Return
	${EndIf}

	Push 0 ; if we are here, service is NOT STOPPED
FunctionEnd

; Return values:
;	<0 - Error
;	0 - NOT STOPPED
; 	1 - Stopped (SECCUSS)
Function StopClient
	DetailPrint "Checking is IVPN Client application is running..."
	Call IsClientStopped
	Pop $0
	${If} $0 == 1
		Push 1 ; Stopped OK
		Return
	${EndIf}

	DetailPrint "Terminating IVPN Client application..."

	; stop client
	nsExec::ExecToStack '"$SYSDIR\taskkill" /IM "${PROCESS_NAME}" /F'
	Pop $0 ; Return
	Pop $1 ; Output
	${If} $0 != '0'
		DetailPrint "Failed to execute taskkill command: $0; $1"
	${EndIf}

	; R1 - counter
	StrCpy	$R1 0
	; waiting to stop 3 seconds (500ms*6)
	${While} $R1 < 6
		Sleep 500
		IntOp $R1 $R1 + 1

		Call IsClientStopped
		Pop $0
		${If} $0 < 0
			Push -1 ; Error
			Return
		${EndIf}
		${If} $0 == 1
			Push 1 ; Stopped OK
			Return
		${EndIf}

	${EndWhile}

	Push 0 ; Not stopped
FunctionEnd

Function IsClientStopped
	nsExec::ExecToStack '"$SYSDIR\tasklist" /FI "IMAGENAME eq ${PROCESS_NAME}"'
	Pop $0 ; Return
	Pop $1 ; Output
	${If} $0 != '0'
		DetailPrint "Failed to execute tasklist command: $0; $1"
		Push -1 ; return execution error
		Return
	${EndIf}

	${StrContains} $0 "${PROCESS_NAME}" $1
	${If} $0 == ""
		Push 1 ; stopped
		Return
	${EndIf}

	Push 0	; running
FunctionEnd

Function WaitFileOpenForWritting
	Pop $1 ; wait milliseconds
	Pop $0 ; filname

	StrCpy	$R1 0
	${While} $R1 < $1
		FileOpen $4 "$0" w
		FileClose $4

		${If} $4 > 0
			Return
		${EndIf}

		DetailPrint "File '$0' is in use. Waiting..."

		Sleep 1000
		IntOp $R1 $R1 + 1000
	${EndWhile}
FunctionEnd

Function ExecAppFile
    Exec "${APP_RUN_PATH}"
    Sleep 500

    StrCpy $R1 0
    ${While} $R1 < 50  ; Wait application launch for 5 seconds max
        IntOp $R1 $R1 + 1
        System::Call user32::GetForegroundWindow()i.r0

        ${If} $0 != $hwndparent
            Return
        ${EndIf}

        Sleep 100
    ${EndWhile}

FunctionEnd

; For Windows 7 there is requirements:
; - Windows7 SP1 should be installed
; - security update KB3033929 should be installed (info: https://docs.microsoft.com/en-us/security-updates/securityadvisories/2015/3033929 )
Function CheckIsWin7DriverInstalled

	; check is it Windows7
	${WinVerGetMajor} $0
	${WinVerGetMinor} $1
	StrCmp '$0.$1' '6.1' label_win7
	Goto end

	label_win7:
		; check is driver works fine
		nsExec::ExecToStack '"$INSTDIR\OpenVPN\$BitDir\tap\${DEVCON_BASENAME}" status ${PRODUCT_TAP_WIN_COMPONENT_ID}'
		Pop $0 ; Return
		Pop $1 ; Output
		${If} $0 != '0'
			; command execution failed - do nothing
			Goto end
		${Else}
			; In case of driver installation problem, 'devcon.exe' returns error.
			; 	e.g.: 'The device has the following problem: 52'
			${StrContains} $0 "problem" $1
			StrCmp $0 "" end ; do nothing if driver has no problems
		${EndIf}

		; check service pack version
		${WinVerGetServicePackLevel} $0
		StrCmp $0 '0' win7_SP1_required
		Goto checkRequiredWinUpdate

		win7_SP1_required:
			; inform user that Windows7 SP1 required
			MessageBox MB_ICONINFORMATION|MB_OK  "Windows 7 Service Pack 1 is not installed on your PC.$\nPlease, install ServicePack1.$\n$\nProbably, you would need to reinstall the application then.\
				$\n$\nhttps://www.microsoft.com/en-us/download/details.aspx?id=5842" IDOK true ;IDCANCEL next
				true:
					;ExecShell "" "iexplore.exe" "https://www.microsoft.com/en-us/download/details.aspx?id=5842"
					;nsExec::ExecToStack 'cmd /Q /C start /Q https://www.microsoft.com/en-us/download/details.aspx?id=5842'
			;	next:
			;Quit
			Goto end

		checkRequiredWinUpdate:
			; check is KB3033929 security update installed (if not - notify to user)
			nsExec::ExecToStack '"$SYSDIR\cmd" /Q /C "%SYSTEMROOT%\System32\wbem\wmic.exe qfe get hotfixid"'
			Pop $0 ; Return
			Pop $1 ; Output

			${If} $0 != '0'
				; command execution failed - do nothing
				Goto end
			${Else}
				${StrContains} $0 "KB3033929" $1
				StrCmp $0 "" notfound
					; security update is installed
					Goto end
				notfound:
					; security update not installed
					${If} ${RunningX64}
						MessageBox MB_ICONINFORMATION|MB_OK  "Security Update for Windows 7 for x64-based Systems (KB3033929) is not installed on your PC.\
							$\nPlease, install Security Update(KB3033929). \
							$\n$\nhttps://www.microsoft.com/en-us/download/details.aspx?id=46148" IDOK yes_x64 ;IDCANCEL quit
							yes_x64:
								;ExecShell "" "iexplore.exe" "https://www.microsoft.com/en-us/download/details.aspx?id=46148"
								;nsExec::ExecToStack 'cmd start /Q https://www.microsoft.com/en-us/download/details.aspx?id=46148'
					${Else}
						MessageBox MB_ICONINFORMATION|MB_OK  "Security Update for Windows 7 (KB3033929) is not installed on your PC.\
							$\nPlease, install Security Update(KB3033929). \
							$\n$\nhttps://www.microsoft.com/en-in/download/details.aspx?id=46078" IDOK yes_x32 ;IDCANCEL quit
							yes_x32:
								;ExecShell "" "iexplore.exe" "https://www.microsoft.com/en-in/download/details.aspx?id=46078"
								;nsExec::ExecToStack 'cmd start /Q https://www.microsoft.com/en-in/download/details.aspx?id=46078'
					${EndIf}
				;quit:
				;Quit
				Goto end
			${EndIf}
	end:
FunctionEnd
