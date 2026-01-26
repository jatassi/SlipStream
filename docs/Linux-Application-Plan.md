# Linux Application Features Implementation Plan

## Overview

Implement Linux-specific features for SlipStream including systemd service integration, package formats, and XDG-compliant data paths.

## Requirements (from Checklist)

1. **Systemd service**
   - Unit file with `After=network-online.target`
   - User service (not system-wide)
   - Restart on failure
2. **Package formats**
   - `.deb` package (Debian/Ubuntu)
   - `.rpm` package (Fedora/RHEL)
   - AppImage (universal)
3. **Data path** - `~/.config/slipstream` (XDG compliant)
4. **Architectures** - x86_64, arm64 (Raspberry Pi 4+, ARM servers)

## Implementation Approach

### Key Decision: Pure Go, No Tray Icon

Linux desktop environments have inconsistent tray/indicator support:
- GNOME removed system tray (extensions can add it back)
- KDE Plasma has full support
- XFCE, MATE, Cinnamon have varying support

**Approach:** No tray icon for Linux. Use systemd user service + browser access.

This is the standard pattern for self-hosted media apps (Sonarr, Radarr, Jellyfin all work this way on Linux).

## Phase 1: Platform Package Extension

Extend existing `internal/platform/` package:

```
internal/
  platform/
    app.go              # Cross-platform interface
    app_windows.go      # Windows implementation (syscalls)
    app_darwin.go       # macOS implementation (macdriver)
    app_linux.go        # Linux implementation (systemd helpers)
```

### `app_linux.go` - Linux Implementation

```go
//go:build linux

package platform

import (
    "os"
    "os/exec"
    "os/signal"
    "syscall"
)

type linuxApp struct {
    config AppConfig
    done   chan struct{}
}

func NewApp(cfg AppConfig) App {
    return &linuxApp{config: cfg, done: make(chan struct{})}
}

func (a *linuxApp) Run() error {
    // No tray - just wait for signals
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    select {
    case <-quit:
    case <-a.done:
    }

    if a.config.OnQuit != nil {
        a.config.OnQuit()
    }
    return nil
}

func (a *linuxApp) OpenBrowser() error {
    // xdg-open is the standard way on Linux
    return exec.Command("xdg-open", a.config.ServerURL).Start()
}

func (a *linuxApp) Stop() {
    close(a.done)
}
```

## Phase 2: Systemd User Service

### Unit File Template

Create `scripts/linux/slipstream.service`:

```ini
[Unit]
Description=SlipStream Media Management
Documentation=https://github.com/slipstream/slipstream
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%h/.local/bin/slipstream
WorkingDirectory=%h/.config/slipstream
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=default.target
```

**Notes:**
- `%h` expands to user's home directory
- `default.target` for user services (not `multi-user.target`)
- Logs go to journald (view with `journalctl --user -u slipstream`)

### Installation Paths

| Component | Path |
|-----------|------|
| Binary | `~/.local/bin/slipstream` |
| Service file | `~/.config/systemd/user/slipstream.service` |
| Config | `~/.config/slipstream/config.yaml` |
| Database | `~/.config/slipstream/slipstream.db` |
| Logs | `~/.config/slipstream/logs/` (or journald) |

### Startup Management

```go
func EnableStartup() error {
    // Copy service file to ~/.config/systemd/user/
    // Run: systemctl --user daemon-reload
    // Run: systemctl --user enable slipstream
    // Run: systemctl --user start slipstream
}

func DisableStartup() error {
    // Run: systemctl --user stop slipstream
    // Run: systemctl --user disable slipstream
}

func IsStartupEnabled() (bool, error) {
    // Run: systemctl --user is-enabled slipstream
}
```

## Phase 3: Desktop Entry (Optional)

For users who want a launcher icon:

Create `scripts/linux/slipstream.desktop`:

```ini
[Desktop Entry]
Type=Application
Name=SlipStream
Comment=Media Management System
Exec=xdg-open http://localhost:8080
Icon=slipstream
Categories=AudioVideo;Network;
Terminal=false
StartupNotify=false
```

Install to: `~/.local/share/applications/slipstream.desktop`

## Phase 4: Package Formats

### AppImage (Already Exists)

Current `scripts/linux/create-appimage.sh` creates AppImage. Updates needed:
- Include systemd service file in AppImage
- Add post-extraction script to install service

### .deb Package

Create `scripts/linux/create-deb.sh`:

```
slipstream_VERSION_amd64/
├── DEBIAN/
│   ├── control
│   ├── postinst      # Install systemd service
│   └── prerm         # Stop service before removal
└── usr/
    ├── bin/
    │   └── slipstream
    └── share/
        ├── applications/
        │   └── slipstream.desktop
        └── doc/
            └── slipstream/
                └── slipstream.service
```

**control file:**
```
Package: slipstream
Version: VERSION
Section: video
Priority: optional
Architecture: amd64
Maintainer: SlipStream <support@slipstream.app>
Description: Media management system
 SlipStream is a unified media management system for movies and TV shows.
```

### .rpm Package

Create `scripts/linux/slipstream.spec`:

```spec
Name:           slipstream
Version:        %{version}
Release:        1%{?dist}
Summary:        Media management system

License:        MIT
URL:            https://github.com/slipstream/slipstream

%description
SlipStream is a unified media management system for movies and TV shows.

%install
mkdir -p %{buildroot}/usr/bin
cp slipstream %{buildroot}/usr/bin/

%files
/usr/bin/slipstream

%post
# Install user service template

%preun
# Stop service if running
```

## Phase 5: Data Path Configuration

### XDG Base Directory Spec

Update `internal/config/config.go`:

```go
func defaultConfigPaths() []string {
    if runtime.GOOS == "linux" {
        // XDG_CONFIG_HOME or default
        configHome := os.Getenv("XDG_CONFIG_HOME")
        if configHome == "" {
            home, _ := os.UserHomeDir()
            configHome = filepath.Join(home, ".config")
        }
        return []string{
            filepath.Join(configHome, "slipstream", "config.yaml"),
            "./config.yaml",
        }
    }
    // ... other platforms
}

func defaultDataPath() string {
    if runtime.GOOS == "linux" {
        dataHome := os.Getenv("XDG_DATA_HOME")
        if dataHome == "" {
            home, _ := os.UserHomeDir()
            dataHome = filepath.Join(home, ".local", "share")
        }
        return filepath.Join(dataHome, "slipstream")
    }
    // ... other platforms
}
```

### Directory Structure

```
~/.config/slipstream/
├── config.yaml
├── slipstream.db
└── logs/
    └── slipstream.log

~/.local/bin/
└── slipstream

~/.config/systemd/user/
└── slipstream.service

~/.local/share/applications/
└── slipstream.desktop
```

## Files to Create/Modify

### Create
- `internal/platform/app_linux.go` - Linux implementation (signal handling, xdg-open)
- `scripts/linux/slipstream.service` - Systemd user service unit
- `scripts/linux/slipstream.desktop` - Desktop entry file
- `scripts/linux/create-deb.sh` - Debian package builder
- `scripts/linux/slipstream.spec` - RPM spec file

### Modify
- `internal/config/config.go` - Add XDG path detection
- `scripts/linux/create-appimage.sh` - Include service file
- `.goreleaser.yaml` - Add deb/rpm package generation

## Build Configuration

### GoReleaser

```yaml
builds:
  - id: slipstream-linux
    goos: [linux]
    goarch: [amd64, arm64]
    env:
      - CGO_ENABLED=0  # Pure Go

nfpms:
  - id: slipstream-packages
    package_name: slipstream
    vendor: SlipStream
    homepage: https://github.com/slipstream/slipstream
    maintainer: SlipStream
    description: Media management system
    formats:
      - deb
      - rpm
    contents:
      - src: scripts/linux/slipstream.service
        dst: /usr/share/doc/slipstream/slipstream.service
      - src: scripts/linux/slipstream.desktop
        dst: /usr/share/applications/slipstream.desktop
```

## Implementation Order

1. ~~Create `app_linux.go` with signal handling and xdg-open~~ **DONE**
2. ~~Create systemd service unit file~~ **DONE**
3. ~~Update config.go for XDG paths~~ **DONE**
4. ~~Create desktop entry file~~ **DONE**
5. ~~Update create-appimage.sh to include service file~~ **DONE**
6. ~~Add nfpms config to .goreleaser.yaml for deb/rpm~~ **DONE**
7. Test on Ubuntu and Fedora

## Verification

1. **Build test:** `GOOS=linux GOARCH=amd64 go build ./cmd/slipstream`
2. **Service test:** Install service, `systemctl --user start slipstream`
3. **Restart test:** Kill process, verify systemd restarts it
4. **Boot test:** Enable service, reboot, verify app starts
5. **Browser test:** Run `xdg-open http://localhost:8080`
6. **Package test:** Install .deb on Ubuntu, .rpm on Fedora
7. **AppImage test:** Download and run AppImage

## Notes

- No tray icon (inconsistent Linux desktop support)
- User service, not system service (no root required)
- XDG compliant paths for config and data
- journald integration for logs (alternative to file logging)
- Pure Go build (CGO_ENABLED=0)
