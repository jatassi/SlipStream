; SlipStream Windows Installer Script (NSIS)
; Requires NSIS 3.x

!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "Sections.nsh"

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

; Finish Page Settings
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Start SlipStream"
!define MUI_FINISHPAGE_RUN_NOTCHECKED
!define MUI_FINISHPAGE_RUN_FUNCTION LaunchSlipStream

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Language
!insertmacro MUI_LANGUAGE "English"

; Function to launch SlipStream with correct working directory
Function LaunchSlipStream
    SetOutPath "$LOCALAPPDATA\SlipStream"
    Exec "$INSTDIR\slipstream.exe"
FunctionEnd

; Component descriptions
LangString DESC_SecMain ${LANG_ENGLISH} "Install SlipStream application files."
LangString DESC_SecStartup ${LANG_ENGLISH} "Start SlipStream automatically when Windows starts."

; Main Install Section (required)
Section "SlipStream (required)" SEC_MAIN
    SectionIn RO
    SetOutPath "$INSTDIR"

    ; Copy files
    File "..\..\dist\slipstream_windows_amd64\slipstream.exe"
    File "slipstream.manifest"
    Rename "$INSTDIR\slipstream.manifest" "$INSTDIR\slipstream.exe.manifest"

    ; Create data directory
    CreateDirectory "$LOCALAPPDATA\SlipStream"
    CreateDirectory "$LOCALAPPDATA\SlipStream\data"

    ; Create blank config.yaml if it doesn't exist
    IfFileExists "$LOCALAPPDATA\SlipStream\config.yaml" +2
    FileOpen $0 "$LOCALAPPDATA\SlipStream\config.yaml" w
    FileClose $0

    ; Create Start Menu shortcuts with working directory set to data folder
    CreateDirectory "$SMPROGRAMS\SlipStream"
    CreateShortcut "$SMPROGRAMS\SlipStream\SlipStream.lnk" "$INSTDIR\slipstream.exe" "" "$INSTDIR\slipstream.exe" 0 SW_SHOWNORMAL "" "SlipStream Media Management"
    ; Set working directory for Start Menu shortcut
    SetOutPath "$LOCALAPPDATA\SlipStream"
    CreateShortcut "$SMPROGRAMS\SlipStream\SlipStream.lnk" "$INSTDIR\slipstream.exe"
    SetOutPath "$INSTDIR"
    CreateShortcut "$SMPROGRAMS\SlipStream\Uninstall.lnk" "$INSTDIR\uninstall.exe"

    ; Create Desktop shortcut with working directory
    SetOutPath "$LOCALAPPDATA\SlipStream"
    CreateShortcut "$DESKTOP\SlipStream.lnk" "$INSTDIR\slipstream.exe"
    SetOutPath "$INSTDIR"

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

    ; Grant current user write access to install directory for seamless auto-updates
    ; This allows the app to update itself without requiring elevation
    nsExec::ExecToLog 'icacls "$INSTDIR" /grant "$USERNAME:(OI)(CI)F" /T'
SectionEnd

; Optional: Start with Windows
Section "Start with Windows" SEC_STARTUP
    SetOutPath "$LOCALAPPDATA\SlipStream"
    CreateShortcut "$SMSTARTUP\SlipStream.lnk" "$INSTDIR\slipstream.exe"
SectionEnd

; Component descriptions
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SEC_MAIN} $(DESC_SecMain)
    !insertmacro MUI_DESCRIPTION_TEXT ${SEC_STARTUP} $(DESC_SecStartup)
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; Uninstaller Section
Section "Uninstall"
    ; Remove startup shortcut if it exists
    Delete "$SMSTARTUP\SlipStream.lnk"

    ; Remove files
    Delete "$INSTDIR\slipstream.exe"
    Delete "$INSTDIR\slipstream.exe.manifest"
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

    ; Ask user if they want to remove data
    MessageBox MB_YESNO "Remove SlipStream data (database, logs, config)?$\n$\nLocation: $LOCALAPPDATA\SlipStream" IDNO skip_data
        RMDir /r "$LOCALAPPDATA\SlipStream"
    skip_data:
SectionEnd
