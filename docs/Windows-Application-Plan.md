# Windows Application Features Implementation Plan

## Overview

Implement Windows-specific features for SlipStream including system tray integration, startup registration, first-run behavior, and installer improvements.

## Requirements (from Checklist)

1. **System tray integration** - Tray icon, left-click opens browser, right-click menu with "Open SlipStream" / "Quit"
2. **Startup on boot** - Place shortcut in Windows Startup folder (`%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`)
3. **NSIS installer updates** - Register startup entry option, uninstaller option to remove all data
4. **First run behavior** - Auto-open browser to localhost:PORT on initial launch
5. **Minimum Windows version** - Windows 10+ only

## Implementation Approach

### Key Decision: Pure Go Implementation (No CGO)

All system tray and Windows API functionality will be implemented using pure Go syscalls to maintain the current `CGO_ENABLED=0` build configuration. This avoids MinGW/GCC dependencies.

## Phase 1: Platform Package Structure

Create new files in `internal/platform/`:

```
internal/
  platform/
    app.go              # Cross-platform interface
    app_windows.go      # Windows implementation (tray, startup, browser)
    app_other.go        # Unix/macOS stub (no-op)
```

### `app.go` - Interface Definition
```go
type AppConfig struct {
    ServerAddress string
    DataPath      string
    Port          int
    NoTray        bool
    OnQuit        func()
}

type App interface {
    Run() error
    OpenBrowser(url string) error
    Stop()
}
```

### `app_windows.go` - Windows Implementation
- System tray using `Shell_NotifyIconW` via syscall
- Invisible message window to receive tray callbacks
- Context menu with "Open SlipStream" and "Quit"
- Left-click and double-click open browser
- Startup folder shortcut management (`%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`)
- Browser launch via `cmd /c start`

## Phase 2: Windows Console Hiding

**DEFERRED** - Keep console visible for debugging during development. Will implement later using `-ldflags "-H windowsgui"` for production builds.

## Phase 3: First-Run Detection

In `internal/platform/firstrun.go`:
```go
func IsFirstRun(dbPath string) bool {
    _, err := os.Stat(dbPath)
    return os.IsNotExist(err)
}
```

If first run, open browser after server starts.

## Phase 4: Main.go Integration

Modify `cmd/slipstream/main.go`:
1. Add `--no-tray` flag for console mode
2. Create platform App with quit callback
3. Check first-run and open browser if needed
4. Run platform message loop (Windows) or wait for signals (Unix)

## Phase 5: NSIS Installer Updates

Update `scripts/windows/installer.nsi`:

1. **Set working directory** for shortcuts to `$LOCALAPPDATA\SlipStream`
2. **Optional startup registration** (checkbox during install):
   ```nsis
   Section "Start with Windows" SEC_STARTUP
       CreateShortcut "$SMSTARTUP\SlipStream.lnk" "$INSTDIR\slipstream.exe" "" "$INSTDIR\slipstream.exe" 0
   SectionEnd
   ```
3. **Uninstaller option** to remove user data:
   ```nsis
   MessageBox MB_YESNO "Remove SlipStream data (database, logs, config)?" IDNO skip_data
       RMDir /r "$LOCALAPPDATA\SlipStream"
   skip_data:
   ```
4. **Remove startup shortcut** in uninstaller:
   ```nsis
   Delete "$SMSTARTUP\SlipStream.lnk"
   ```

## Phase 6: Icon Assets

Create Windows icon file:
- Location: `assets/icons/slipstream.ico`
- Sizes: 16x16, 32x32, 48x48, 256x256
- Embed in binary or load from install directory

## Files to Create/Modify

### Create
- `internal/platform/app.go` - Interface definition
- `internal/platform/app_windows.go` - Windows tray/startup implementation
- `internal/platform/app_other.go` - Unix stub
- `assets/icons/slipstream.ico` - Windows icon (placeholder needed)

### Modify
- `cmd/slipstream/main.go` - Integrate platform app
- `scripts/windows/installer.nsi` - Startup and uninstall options
- `.goreleaser.yaml` - Add `-H windowsgui` for Windows builds (optional)

## Implementation Order

1. Create `internal/platform/` package with interface and stubs
2. Implement Windows tray basics (icon, click handlers)
3. Add context menu (Open/Quit)
4. Implement startup registration
5. Add first-run detection and browser launch
6. Integrate with main.go
7. Update NSIS installer
8. Create icon assets
9. Test on Windows

## Verification

1. **Build test:** `go build ./cmd/slipstream` on Windows
2. **Tray test:** Run app, verify tray icon appears
3. **Click test:** Left-click opens browser, right-click shows menu
4. **Quit test:** "Quit" menu item stops the application
5. **Startup test:** Enable startup, reboot, verify app starts
6. **First-run test:** Delete database, run app, verify browser opens
7. **Installer test:** Fresh install, verify shortcuts work with correct working directory
8. **Uninstall test:** Verify startup entry removed, optional data removal works

## Notes

- Windows 10+ requirement enforced via manifest (not runtime check)
- Tray tooltip shows "SlipStream - Media Management"
- Console mode (`--no-tray`) available for debugging/service usage
- Startup uses Startup folder shortcut (user-visible, easy to manage via Task Manager > Startup)
