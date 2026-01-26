# macOS Application Features Implementation Plan

## Overview

Implement macOS-specific features for SlipStream including menu bar integration, launch agent for startup, and DMG installer improvements.

## Requirements (from Checklist)

1. **Menu bar integration** - Similar to Windows tray (open/quit)
2. **Launch agent** - `~/Library/LaunchAgents` plist for startup on boot
3. **DMG installer** - Drag-to-Applications style installer
4. **Data path** - `~/Library/Application Support/SlipStream`
5. **Apple Silicon** - arm64 native build
6. **Intel** - x86_64 build (or universal binary)
7. **Code signing** - Deferred (Apple Developer account for notarization when budget allows)

## Implementation Approach

### Key Decision: Use `progrium/macdriver` for Native Cocoa Integration

Use `github.com/progrium/macdriver` for full native macOS menu bar support via Cocoa bindings.

**Why macdriver:**
- Direct access to NSStatusBar, NSMenu, NSMenuItem
- Native macOS look and feel
- Full feature parity with Windows tray
- CGO required but only links system frameworks (no runtime dependencies for users)

**Build requirement:** macOS runner with Xcode Command Line Tools (GitHub Actions `macos-latest` has this)

## Phase 1: Platform Package Extension

Extend existing `internal/platform/` package:

```
internal/
  platform/
    app.go              # Cross-platform interface (exists)
    app_windows.go      # Windows implementation (pure Go syscalls)
    app_darwin.go       # macOS implementation (macdriver/Cocoa)
    app_linux.go        # Linux stub
```

### `app_darwin.go` - macOS Implementation

**Menu Bar (via macdriver):**
```go
import (
    "github.com/progrium/macdriver/macos/appkit"
    "github.com/progrium/macdriver/macos/foundation"
    "github.com/progrium/macdriver/objc"
)

type macApp struct {
    config     AppConfig
    statusItem appkit.StatusItem
    done       chan struct{}
}

func (a *macApp) Run() error {
    // Create status bar item
    a.statusItem = appkit.StatusBar_SystemStatusBar().StatusItemWithLength(appkit.VariableStatusItemLength)
    a.statusItem.Button().SetTitle("SlipStream")
    // Or set icon: a.statusItem.Button().SetImage(icon)

    // Create menu
    menu := appkit.NewMenu()

    openItem := appkit.NewMenuItem()
    openItem.SetTitle("Open SlipStream")
    openItem.SetAction(objc.Sel("openBrowser:"))
    menu.AddItem(openItem)

    menu.AddItem(appkit.MenuItem_SeparatorItem())

    quitItem := appkit.NewMenuItem()
    quitItem.SetTitle("Quit")
    quitItem.SetAction(objc.Sel("quit:"))
    menu.AddItem(quitItem)

    a.statusItem.SetMenu(menu)

    // Run the app
    app := appkit.Application_SharedApplication()
    app.Run()
    return nil
}
```

**Browser Launch:**
```go
func (a *macApp) OpenBrowser() error {
    url := foundation.URL_URLWithString(a.config.ServerURL)
    appkit.Workspace_SharedWorkspace().OpenURL(url)
    return nil
}
```

**Startup Registration:**
- Create/remove plist in `~/Library/LaunchAgents/`
- Plist file: `com.slipstream.slipstream.plist`

## Phase 2: Launch Agent

### Plist Template

Create `scripts/macos/com.slipstream.slipstream.plist`:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.slipstream.slipstream</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Applications/SlipStream.app/Contents/MacOS/slipstream</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
    <key>StandardOutPath</key>
    <string>~/Library/Logs/SlipStream/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>~/Library/Logs/SlipStream/stderr.log</string>
</dict>
</plist>
```

### Startup Management Functions

```go
func EnableStartup(appPath string) error {
    // Copy plist to ~/Library/LaunchAgents/
    // Run: launchctl load ~/Library/LaunchAgents/com.slipstream.slipstream.plist
}

func DisableStartup() error {
    // Run: launchctl unload ~/Library/LaunchAgents/com.slipstream.slipstream.plist
    // Remove plist file
}

func IsStartupEnabled() (bool, error) {
    // Check if plist exists in ~/Library/LaunchAgents/
}
```

## Phase 3: DMG Installer Updates

### Current State

Existing `scripts/macos/create-dmg.sh` creates:
- `.app` bundle with proper Info.plist
- DMG with drag-to-Applications layout

### Enhancements Needed

1. **Post-install script** (optional, runs on first launch):
   - Create data directory: `~/Library/Application Support/SlipStream/`
   - Optionally install launch agent

2. **Uninstall instructions** in README or help:
   ```
   1. Drag SlipStream.app to Trash
   2. Remove ~/Library/Application Support/SlipStream/ (optional, keeps data)
   3. Remove ~/Library/LaunchAgents/com.slipstream.slipstream.plist
   ```

## Phase 4: Data Path Configuration

### Standard macOS Paths

| Type | Path |
|------|------|
| Config | `~/Library/Application Support/SlipStream/config.yaml` |
| Database | `~/Library/Application Support/SlipStream/slipstream.db` |
| Logs | `~/Library/Logs/SlipStream/` |
| Cache | `~/Library/Caches/SlipStream/` |

### Config Detection

Update `internal/config/config.go` to check macOS paths:
```go
func defaultConfigPaths() []string {
    if runtime.GOOS == "darwin" {
        home, _ := os.UserHomeDir()
        return []string{
            filepath.Join(home, "Library", "Application Support", "SlipStream", "config.yaml"),
            "./config.yaml",
        }
    }
    // ... existing paths
}
```

## Phase 5: App Bundle Structure

### Current Bundle (from create-dmg.sh)

```
SlipStream.app/
├── Contents/
│   ├── Info.plist
│   ├── MacOS/
│   │   └── slipstream          # Main binary
│   └── Resources/
│       └── slipstream.icns     # App icon (needs creation)
```

### Info.plist Requirements

```xml
<key>LSUIElement</key>
<true/>  <!-- Hides from Dock, menu bar app only -->

<key>LSMinimumSystemVersion</key>
<string>10.15</string>  <!-- macOS Catalina minimum -->
```

## Files to Create/Modify

### Create
- `internal/platform/app_darwin.go` - macOS menu bar/startup via macdriver
- `scripts/macos/com.slipstream.slipstream.plist` - Launch agent template
- `assets/icons/slipstream.icns` - macOS app icon

### Modify
- `internal/platform/app_other.go` → rename to `app_linux.go` (Linux-only stub)
- `scripts/macos/create-dmg.sh` - Add launch agent to bundle
- `internal/config/config.go` - Add macOS default paths
- `go.mod` - Add `github.com/progrium/macdriver` dependency
- `.goreleaser.yaml` - Enable CGO for darwin builds

## Build Configuration

### GoReleaser Changes

```yaml
builds:
  - id: slipstream-darwin
    goos: [darwin]
    goarch: [amd64, arm64]
    env:
      - CGO_ENABLED=1
    # ... other settings

  - id: slipstream-windows
    goos: [windows]
    goarch: [amd64]
    env:
      - CGO_ENABLED=0  # Keep pure Go for Windows
    # ... other settings

  - id: slipstream-linux
    goos: [linux]
    goarch: [amd64, arm64]
    env:
      - CGO_ENABLED=0  # Keep pure Go for Linux
    # ... other settings
```

### CI/CD Pipeline

macOS builds must run on macOS runner (not cross-compile):
```yaml
macos-build:
  runs-on: macos-latest  # Has Xcode pre-installed
  steps:
    - run: CGO_ENABLED=1 go build -o slipstream ./cmd/slipstream
```

## Implementation Order

1. Add `github.com/progrium/macdriver` to go.mod
2. Create `app_darwin.go` with menu bar and startup functions
3. Create launch agent plist template
4. Update config.go for macOS paths
5. Update .goreleaser.yaml for CGO on darwin
6. Update CI/CD for native macOS builds
7. Test on macOS (Intel and Apple Silicon)
8. Create .icns icon file

## Verification

1. **Build test:** `CGO_ENABLED=1 go build ./cmd/slipstream` (on macOS)
2. **Menu bar test:** Run app, verify icon appears in menu bar
3. **Click test:** Click icon shows menu, "Open SlipStream" opens browser
4. **Quit test:** "Quit" menu item stops the application
5. **Startup test:** Enable launch agent, reboot, verify app starts
6. **DMG test:** Create DMG, install, verify app bundle works
7. **Data path test:** Verify config/db created in correct location

## Notes

- CGO links against system Cocoa frameworks (no runtime dependencies for users)
- Code signing deferred until Apple Developer account available
- Separate builds for arm64 and amd64 (not universal binary initially)
- macOS Catalina (10.15) minimum for macdriver compatibility
