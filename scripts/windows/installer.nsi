; SlipStream Windows Installer Script (NSIS)
; Requires NSIS 3.x

!include "MUI2.nsh"
!include "FileFunc.nsh"

; Basic Info
Name "SlipStream"
OutFile "slipstream_${VERSION}_windows_amd64_setup.exe"
InstallDir "$PROGRAMFILES64\SlipStream"
InstallDirRegKey HKLM "Software\SlipStream" "InstallDir"
RequestExecutionLevel admin

; Version Info
VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "SlipStream"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "FileDescription" "SlipStream Media Management"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "LegalCopyright" "SlipStream"

; Modern UI Settings
!define MUI_ABORTWARNING
; Icons are optional - NSIS will use default if not present

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Language
!insertmacro MUI_LANGUAGE "English"

; Installer Section
Section "Install"
    SetOutPath "$INSTDIR"

    ; Copy files
    File "..\..\dist\slipstream_windows_amd64_v1\slipstream.exe"

    ; Create data directory
    CreateDirectory "$LOCALAPPDATA\SlipStream"
    CreateDirectory "$LOCALAPPDATA\SlipStream\data"

    ; Create blank config.yaml if it doesn't exist
    IfFileExists "$LOCALAPPDATA\SlipStream\config.yaml" +2
    FileOpen $0 "$LOCALAPPDATA\SlipStream\config.yaml" w
    FileClose $0

    ; Create Start Menu shortcuts
    CreateDirectory "$SMPROGRAMS\SlipStream"
    CreateShortcut "$SMPROGRAMS\SlipStream\SlipStream.lnk" "$INSTDIR\slipstream.exe"
    CreateShortcut "$SMPROGRAMS\SlipStream\Uninstall.lnk" "$INSTDIR\uninstall.exe"

    ; Create Desktop shortcut
    CreateShortcut "$DESKTOP\SlipStream.lnk" "$INSTDIR\slipstream.exe"

    ; Write registry keys
    WriteRegStr HKLM "Software\SlipStream" "InstallDir" "$INSTDIR"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SlipStream" "DisplayName" "SlipStream"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SlipStream" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SlipStream" "DisplayVersion" "${VERSION}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SlipStream" "Publisher" "SlipStream"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SlipStream" "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SlipStream" "NoRepair" 1

    ; Get installed size
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SlipStream" "EstimatedSize" "$0"

    ; Create uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

; Uninstaller Section
Section "Uninstall"
    ; Stop service if running (future: when service support added)
    ; nsExec::ExecToLog 'sc stop SlipStream'

    ; Remove files (keep user data in $LOCALAPPDATA\SlipStream)
    Delete "$INSTDIR\slipstream.exe"
    Delete "$INSTDIR\uninstall.exe"

    ; Remove shortcuts
    Delete "$SMPROGRAMS\SlipStream\SlipStream.lnk"
    Delete "$SMPROGRAMS\SlipStream\Uninstall.lnk"
    RMDir "$SMPROGRAMS\SlipStream"
    Delete "$DESKTOP\SlipStream.lnk"

    ; Remove install directory
    RMDir "$INSTDIR"

    ; Remove registry keys
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\SlipStream"
    DeleteRegKey HKLM "Software\SlipStream"

    ; Note: Don't remove user data in $LOCALAPPDATA\SlipStream
SectionEnd
