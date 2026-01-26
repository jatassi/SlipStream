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

### Key Decision: Pure Go Implementation (No CGO)

All menu bar and macOS API functionality will be implemented using pure Go to maintain the current `CGO_ENABLED=0` build configuration.

**Challenge:** Unlike Windows, macOS menu bar apps typically require Objective-C/Cocoa frameworks. Pure Go options are limited.

**Options:**
1. **Use `os/exec` to call AppleScript** - Simple but limited functionality
2. **Embed a minimal Swift helper** - More capable but adds build complexity
3. **Accept headless mode** - No menu bar, just launch agent + browser access

**Recommended:** Start with option 1 (AppleScript) for basic menu bar, can enhance later.

## Phase 1: Platform Package Extension

Extend existing `internal/platform/` package:

```
internal/
  platform/
    app.go              # Cross-platform interface (exists)
    app_windows.go      # Windows implementation (exists)
    app_darwin.go       # macOS implementation (NEW)
    app_other.go        # Linux stub (rename from app_other.go)
```

### `app_darwin.go` - macOS Implementation

**Menu Bar (via AppleScript/osascript):**
- Display status item in menu bar
- Click shows menu: "Open SlipStream" / "Quit"
- Uses `osascript` to create native dialogs

**Alternative (simpler initial approach):**
- No persistent menu bar icon
- Just launch agent + first-run browser open
- Users access via browser bookmark

**Browser Launch:**
```go
func OpenBrowser(url string) error {
    return exec.Command("open", url).Start()
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
- `internal/platform/app_darwin.go` - macOS menu bar/startup implementation
- `scripts/macos/com.slipstream.slipstream.plist` - Launch agent template
- `assets/icons/slipstream.icns` - macOS app icon

### Modify
- `internal/platform/app_other.go` → rename to `app_linux.go` (Linux-only stub)
- `scripts/macos/create-dmg.sh` - Add launch agent to bundle
- `internal/config/config.go` - Add macOS default paths

## Implementation Order

1. Create `app_darwin.go` with browser launch and startup functions
2. Create launch agent plist template
3. Update config.go for macOS paths
4. Update create-dmg.sh to include launch agent
5. Test on macOS (Intel and Apple Silicon)
6. Create .icns icon file

## Verification

1. **Build test:** `GOOS=darwin GOARCH=arm64 go build ./cmd/slipstream`
2. **Browser test:** Run app, verify browser opens on first run
3. **Startup test:** Enable launch agent, reboot, verify app starts
4. **DMG test:** Create DMG, install, verify app bundle works
5. **Data path test:** Verify config/db created in correct location

## Notes

- Menu bar icon deferred to Phase 2 (requires more complex Cocoa integration)
- Code signing deferred until Apple Developer account available
- Universal binary (arm64 + amd64) possible but increases build complexity
- macOS Catalina (10.15) minimum for modern Swift/security features
