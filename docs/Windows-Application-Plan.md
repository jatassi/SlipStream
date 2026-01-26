# Windows Application Features Implementation Plan

**Status: IMPLEMENTED** (2026-01-26)

## Overview

Implement Windows-specific features for SlipStream including system tray integration, startup registration, first-run behavior, and installer improvements.

## Requirements (from Checklist)

1. **System tray integration** - Tray icon, left-click opens browser, right-click menu with "Open SlipStream" / "Quit"
2. **Startup on boot** - Place shortcut in Windows Startup folder (`%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`)
3. **NSIS installer updates** - Register startup entry option, uninstaller option to remove all data
4. **First run behavior** - Auto-open browser to localhost:PORT on initial launch
5. **Minimum Windows version** - Windows 10+ only

## Implementation Approach

### Key Decision: CGO with `tailscale/walk` for Native Windows Integration

Use `github.com/tailscale/walk` for native Windows system tray support. This is a maintained fork of `lxn/walk` used by Tailscale for their Windows tray application.

**Why walk:**
- Clean, idiomatic Go API (no manual syscall wrangling)
- Native Windows look and feel
- Robust message pump handling
- Easy context menu creation with Action objects
- Proper event handling
- Battle-tested in production (Tailscale)

**Build requirement:** MinGW-w64 GCC toolchain for CGO compilation

**Note:** Previous implementation used pure Go syscalls. This refactor provides cleaner code and better maintainability.

## Phase 0: Remove Existing Syscall Implementation

Before implementing the walk-based solution, completely remove the existing pure Go syscall implementation from `internal/platform/app_windows.go`:

**Remove all of the following:**
- All `syscall.NewLazyDLL` and `NewProc` declarations (user32, shell32, kernel32)
- All Windows API constant definitions (WM_*, NIM_*, NIF_*, etc.)
- All manual struct definitions (`wndClassExW`, `msg`, `point`, `notifyIconDataW`)
- The `wndProc` callback function
- All direct syscall invocations (`pShellNotifyIconW.Call`, `pCreateWindowExW.Call`, etc.)

**Start fresh** with a clean `app_windows.go` that only imports `github.com/tailscale/walk` and standard library packages. Do not attempt to preserve or adapt any of the syscall-based code - the walk API is fundamentally different and mixing approaches will create maintenance burden.

## Phase 1: Platform Package Structure

Files in `internal/platform/`:

```
internal/
  platform/
    app.go              # Cross-platform interface
    app_windows.go      # Windows implementation (walk/CGO)
    app_darwin.go       # macOS implementation (macdriver/CGO)
    app_linux.go        # Linux stub (no-op)
    util.go             # Shared utilities
```

### `app.go` - Interface Definition
```go
type AppConfig struct {
    ServerURL string
    DataPath  string
    Port      int
    NoTray    bool
    OnQuit    func()
}

type App interface {
    Run() error
    OpenBrowser(url string) error
    Stop()
}
```

### `app_windows.go` - Windows Implementation (walk)

```go
//go:build windows

package platform

import (
    "os/exec"
    "sync"

    "github.com/tailscale/walk"
)

type windowsApp struct {
    config     AppConfig
    mainWindow *walk.MainWindow
    notifyIcon *walk.NotifyIcon
    running    bool
    mu         sync.Mutex
    stopOnce   sync.Once
}

func NewApp(cfg AppConfig) App {
    return &windowsApp{config: cfg}
}

func (a *windowsApp) Run() error {
    if a.config.NoTray {
        a.mu.Lock()
        a.running = true
        a.mu.Unlock()
        select {} // Block forever in no-tray mode
    }

    var err error
    a.mainWindow, err = walk.NewMainWindow()
    if err != nil {
        return err
    }

    a.notifyIcon, err = walk.NewNotifyIcon(a.mainWindow)
    if err != nil {
        return err
    }

    a.notifyIcon.SetToolTip("SlipStream - Media Management")
    a.notifyIcon.SetVisible(true)

    // Load icon from executable
    if icon, err := walk.Resources.Icon("1"); err == nil {
        a.notifyIcon.SetIcon(icon)
    }

    // Left-click opens browser
    a.notifyIcon.MouseUp().Attach(func(x, y int, button walk.MouseButton) {
        if button == walk.LeftButton {
            a.OpenBrowser(a.config.ServerURL)
        }
    })

    // Context menu
    openAction := walk.NewAction()
    openAction.SetText("Open SlipStream")
    openAction.Triggered().Attach(func() {
        a.OpenBrowser(a.config.ServerURL)
    })

    quitAction := walk.NewAction()
    quitAction.SetText("Quit")
    quitAction.Triggered().Attach(func() {
        a.Stop()
    })

    a.notifyIcon.ContextMenu().Actions().Add(openAction)
    a.notifyIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction())
    a.notifyIcon.ContextMenu().Actions().Add(quitAction)

    a.mu.Lock()
    a.running = true
    a.mu.Unlock()

    a.mainWindow.Run()
    return nil
}

func (a *windowsApp) OpenBrowser(url string) error {
    cmd := exec.Command("cmd", "/c", "start", "", url)
    return cmd.Start()
}

func (a *windowsApp) Stop() {
    a.stopOnce.Do(func() {
        a.mu.Lock()
        if a.running {
            if a.notifyIcon != nil {
                a.notifyIcon.Dispose()
            }
            if a.mainWindow != nil {
                a.mainWindow.Close()
            }
        }
        a.mu.Unlock()

        if a.config.OnQuit != nil {
            a.config.OnQuit()
        }
    })
}
```

## Phase 2: Windows Console Hiding

**DEFERRED** - Keep console visible for debugging during development. Will implement later using `-ldflags "-H windowsgui"` for production builds.

## Phase 3: First-Run Detection

In `internal/platform/util.go`:
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
- Embed in binary via `rsrc` or `goversioninfo` for walk to access

## Build Configuration

### Dependencies

Add to `go.mod`:
```
github.com/tailscale/walk v0.0.0-latest
```

### Windows Build Requirements

**Local development:**
- Install MinGW-w64: `choco install mingw` or download from [mingw-w64.org](https://www.mingw-w64.org/)
- Ensure `gcc` is in PATH
- Build: `CGO_ENABLED=1 go build ./cmd/slipstream`

**Cross-compilation (from Linux/macOS):**
```bash
# Install mingw-w64 cross-compiler
# Ubuntu: apt install mingw-w64
# macOS: brew install mingw-w64

CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build ./cmd/slipstream
```

### GoReleaser Configuration

```yaml
builds:
  - id: slipstream-windows
    goos: [windows]
    goarch: [amd64]
    env:
      - CGO_ENABLED=1
      - CC=x86_64-w64-mingw32-gcc
    ldflags:
      - -s -w
      - -H windowsgui  # Hide console in production
    # ... other settings

  - id: slipstream-darwin
    goos: [darwin]
    goarch: [amd64, arm64]
    env:
      - CGO_ENABLED=1
    # ... other settings

  - id: slipstream-linux
    goos: [linux]
    goarch: [amd64, arm64]
    env:
      - CGO_ENABLED=0  # Keep pure Go for Linux (no tray)
    # ... other settings
```

### CI/CD Pipeline

Windows builds can cross-compile from Linux with mingw-w64:
```yaml
windows-build:
  runs-on: ubuntu-latest
  steps:
    - name: Install MinGW
      run: sudo apt-get install -y mingw-w64

    - name: Build Windows binary
      run: |
        CGO_ENABLED=1 \
        CC=x86_64-w64-mingw32-gcc \
        GOOS=windows \
        GOARCH=amd64 \
        go build -ldflags="-H windowsgui" -o slipstream.exe ./cmd/slipstream
```

Or native Windows build:
```yaml
windows-build:
  runs-on: windows-latest
  steps:
    - name: Install MinGW
      run: choco install mingw -y

    - name: Build
      run: |
        $env:CGO_ENABLED=1
        go build -ldflags="-H windowsgui" -o slipstream.exe ./cmd/slipstream
```

## Files to Create/Modify

### Create
- `assets/icons/slipstream.ico` - Windows icon (for tray and installer)

### Modify
- `internal/platform/app_windows.go` - **Complete rewrite**: delete all syscall code, replace with walk implementation
- `go.mod` - Add `github.com/tailscale/walk` dependency
- `.goreleaser.yaml` - Enable CGO for Windows builds, add `-H windowsgui`
- `.github/workflows/release.yml` - Add MinGW installation step

## Implementation Order

1. **Remove existing syscall implementation** - Delete all syscall-based code from `app_windows.go` (see Phase 0)
2. Add `github.com/tailscale/walk` to go.mod
3. Implement fresh `app_windows.go` using walk API
4. Update .goreleaser.yaml for CGO on Windows
5. Update CI/CD for MinGW installation
6. Test on Windows
7. Create .ico icon file (optional, can use exe embedded icon)

## Verification

1. **Build test:** `CGO_ENABLED=1 go build ./cmd/slipstream` (on Windows with MinGW)
2. **Cross-compile test:** Build from Linux with mingw-w64
3. **Tray test:** Run app, verify tray icon appears
4. **Click test:** Left-click opens browser, right-click shows menu
5. **Quit test:** "Quit" menu item stops the application
6. **Startup test:** Enable startup, reboot, verify app starts
7. **First-run test:** Delete database, run app, verify browser opens
8. **Installer test:** Fresh install, verify shortcuts work with correct working directory
9. **Uninstall test:** Verify startup entry removed, optional data removal works

## Notes

- Windows 10+ requirement enforced via manifest (not runtime check)
- Tray tooltip shows "SlipStream - Media Management"
- Console mode (`--no-tray`) available for debugging/service usage
- Startup uses Startup folder shortcut (user-visible, easy to manage via Task Manager > Startup)
- CGO links against Windows system DLLs (no runtime dependencies for users)
- walk uses standard Windows APIs under the hood, just provides cleaner Go interface

## Implementation Summary

**Implemented on:** 2026-01-26

### Files Modified
- `internal/platform/app_windows.go` - Complete rewrite using tailscale/walk for tray support
- `internal/platform/app.go` - Added `Port` field to `AppConfig`
- `cmd/slipstream/main.go` - Added `--no-tray` flag, platform App integration, first-run browser open
- `scripts/windows/installer.nsi` - Added startup registration (optional section), data removal prompt on uninstall
- `.goreleaser.yaml` - Split builds: Windows with CGO+walk, Linux/macOS without CGO
- `.github/workflows/release.yml` - Added mingw-w64 installation for Windows cross-compilation
- `go.mod` - Added `github.com/tailscale/walk` dependency

### Deferred Items
- Phase 2 (Console hiding) - Using `-H windowsgui` ldflags for production; console visible in development
- Phase 6 (Icon assets) - Using exe embedded icon; custom .ico file creation deferred
