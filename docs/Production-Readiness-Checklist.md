# Production Readiness Checklist

This document tracks all work required to prepare SlipStream for production beta release.

## Release Strategy

| Decision | Choice |
|----------|--------|
| Initial beta platform | Windows only |
| Beta distribution | Private releases with gated downloads |
| Code signing | Deferred (accept security warnings for beta) |
| Timeline | When ready (no artificial deadline) |
| Support channel | GitHub Issues |

---

## Phase 1: Core Infrastructure (Windows Beta)

### Build System

- [x] **Switch to pure Go SQLite** - Using `modernc.org/sqlite` (pure Go, no CGO dependency)
- [x] **Build-time versioning** - Version injected via Makefile `VERSION` parameter (ldflags)
- [ ] **Static binary** - Ensure fully self-contained executable with no external dependencies

### CI/CD Pipeline

- [ ] **Release workflow** - Trigger on every push to main
- [ ] **Build artifacts**:
  - [ ] Windows x64 NSIS/Inno Setup installer (.exe)
  - [ ] Windows x64 portable ZIP
  - [ ] MacOS DMG Installer
  - [ ] `.deb` package (Debian/Ubuntu)
  - [ ] `.rpm` package (Fedora/RHEL)
  - [ ] AppImage (universal)
- [ ] **Changelog** - Manual changelog in release notes (markdown format)

### Windows Application

- [ ] **System tray integration**
  - [ ] Tray icon with SlipStream branding
  - [ ] Left-click: Open browser to web UI
  - [ ] Right-click menu: "Open SlipStream" / "Quit"
  - [ ] Minimize to tray on window close
  - [ ] Explicit "Quit" required to stop service
- [ ] **Startup on boot** - Register with Windows Task Scheduler or Registry Run key
- [ ] **NSIS/Inno Setup installer**
  - [ ] Install to Program Files
  - [ ] Create Start Menu shortcuts
  - [ ] Register startup entry
  - [ ] Uninstaller that removes all data (config, database, logs)
- [ ] **First run behavior** - Auto-open browser to localhost:PORT on initial launch
- [ ] **Minimum Windows version** - Windows 10+ only

### Runtime Behavior

- [ ] **Standard data paths** - Use `%APPDATA%\SlipStream` for config/database/logs
- [ ] **Port conflict handling** - Auto-find next available port if 8888 is in use, notify user of actual port
- [ ] **Network binding** - Bind to localhost (127.0.0.1) by default
- [ ] **Boot timing** - Retry network-dependent features with exponential backoff if network unavailable
- [ ] **Log rotation** - Built-in rotation (e.g., 5 x 10MB files, delete oldest)

### Auto-Update System

- [ ] **Update check and auto install** - On startup + periodic (every 24 hours) -> use task system
- [ ] **Auto-install toggle** - add toggle on update page - on/off enables/disables auto update task - default on - part of config (persists across restart)
- [ ] **GitHub Releases integration** - Check for new releases via GitHub API
- [ ] **Always latest** - No version skipping, updates always go to newest
- [ ] **Offline handling** - use update page update error state
- [ ] **Pre-update backup** - Automatically backup database before applying update -> placeholder only, need to build out backup system
- [ ] **Recovery strategy** - On failed update, user downloads fresh installer (simple approach)

### Security

- [x] **API rate limiting** - IP-based rate limiting on auth endpoints (10 req/min)
- [x] **Account lockout** - Progressive lockout after 5 failed login attempts (15min → 30min → 1hr)
- [x] **Security headers** - X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, Referrer-Policy, CSP frame-ancestors
- [x] **Request body size limit** - 2MB limit to prevent memory exhaustion
- [x] **CORS restrictions** - Same-origin only (origin hostname must match request hostname)
- [x] **Embedded API keys** - TMDB/TVDB keys embedded at build time via ldflags
- [ ] **Credential encryption** - Encryption package ready (`internal/crypto/secrets.go`), service integration deferred

#### Secrets Storage (Following Sonarr/Radarr Pattern)

User-specific credentials (download client passwords, indexer API keys) are stored in plaintext in the SQLite database, following the same pattern as Sonarr/Radarr. This is acceptable for self-hosted applications where:
- The database file is on the user's own machine
- File system permissions (600) restrict access
- Credentials are never exposed via API responses

---

## Phase 2: Multi-Platform Expansion (Post-Beta)

### macOS Support

- [ ] **GitHub-hosted runner** - Use free GitHub Actions macOS runner
- [ ] **Menu bar integration** - Similar to Windows tray (open/quit)
- [ ] **Launch agent** - `~/Library/LaunchAgents` plist for startup on boot
- [ ] **DMG installer** - Drag-to-Applications style installer
- [ ] **Data path** - `~/Library/Application Support/SlipStream`
- [ ] **Apple Silicon** - arm64 native build
- [ ] **Intel** - x86_64 build (or universal binary)
- [ ] **Code signing** - Apple Developer account for notarization (when budget allows)

### Linux Support

- [ ] **GitHub-hosted runner** - Use free GitHub Actions Linux runner
- [ ] **Systemd service**
  - [ ] Unit file with `After=network-online.target`
  - [ ] User service (not system-wide)
  - [ ] Restart on failure
- [ ] **Package formats**:
  - [ ] `.deb` package (Debian/Ubuntu)
  - [ ] `.rpm` package (Fedora/RHEL)
  - [ ] AppImage (universal)
- [ ] **Data path** - `~/.config/slipstream` (XDG compliant)
- [ ] **Architectures**:
  - [ ] x86_64
  - [ ] arm64 (Raspberry Pi 4+, ARM servers)

### Docker Support

- [ ] **Official Docker image** - Published to GitHub Container Registry (ghcr.io)
- [ ] **Base image** - Debian slim for compatibility and debuggability
- [ ] **Multi-arch** - linux/amd64 + linux/arm64
- [ ] **Docker Compose example** - Sample compose file in repo
- [ ] **Volume mounts** - `/config` for persistent data
- [ ] **Environment variables** - All config options available as env vars
- [ ] **Health check** - Built-in healthcheck endpoint

---

## Phase 3: Production Polish

### Code Signing (When Budget Allows)

- [ ] **Windows EV certificate** - Eliminates SmartScreen warnings
- [ ] **Apple Developer account** - Required for notarization
- [ ] **Signed installers** - All distributed binaries signed

### Migration Tools

- [ ] **Sonarr import** - Import library, quality profiles, download clients
- [ ] **Radarr import** - Import library, quality profiles, download clients

---

## Technical Decisions Summary

| Area | Decision | Rationale |
|------|----------|-----------|
| SQLite | `modernc.org/sqlite` (pure Go) | Eliminates CGO, simplifies cross-compilation |
| Data location | Standard OS paths | Follows platform conventions |
| Update policy | Auto-download, prompt install | Balance between convenience and user control |
| Background mode | Always run, close to tray | Media server should be always-on |
| Log management | Built-in rotation | Self-contained, no external dependencies |
| HTTPS | Reverse proxy assumed | Standard for self-hosted apps |
| Telemetry | None | Privacy-focused |
| Network binding | Localhost only by default | Security-first, user enables remote access |
| Uninstall | Remove everything | Clean slate for reinstalls |
| Versioning | Build script parameter (ldflags) | Explicit control over version numbers |
| Release trigger | Every push to main | Fast iteration during beta |
| Auth rate limiting | IP + account lockout | Prevents brute force attacks |
| Secrets storage | Plaintext (like Sonarr/Radarr) | Standard for self-hosted apps, rely on file permissions |
| API keys | Embedded at build time | Zero-config for users, ldflags injection |

---

## CI/CD Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Push to main                             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Self-hosted Windows Runner                      │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  1. Checkout code                                    │    │
│  │  2. Set version (from workflow input or tag)         │    │
│  │  3. Build frontend (bun)                             │    │
│  │  4. Build backend (go build with VERSION=x.y.z)      │    │
│  │  5. Create NSIS installer                            │    │
│  │  6. Create portable ZIP                              │    │
│  │  7. Create draft GitHub Release                      │    │
│  │  8. Upload artifacts                                 │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│           Manual: Edit changelog, publish release            │
└─────────────────────────────────────────────────────────────┘
```

### Future Multi-Platform Pipeline

```
Push to main
     │
     ├──► Self-hosted Windows ──► Windows x64 installer + portable
     │
     ├──► GitHub macOS ──► macOS arm64 + x64 DMG
     │
     ├──► GitHub Linux ──► Linux x64/arm64 .deb/.rpm/AppImage
     │
     └──► GitHub Linux ──► Docker multi-arch image
                │
                ▼
         Draft GitHub Release (all artifacts)
                │
                ▼
         Manual publish after changelog review
```

---

## Update Flow

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  App Start  │────►│ Check GitHub │────►│ New version?    │
└─────────────┘     │   Releases   │     └────────┬────────┘
                    └──────────────┘              │
                           ▲                      │ Yes
                           │                      ▼
                    ┌──────┴───────┐     ┌─────────────────┐
                    │ Every 24hrs  │     │ Download in     │
                    │   (silent)   │     │ background      │
                    └──────────────┘     └────────┬────────┘
                                                  │
                                                  ▼
                                         ┌─────────────────┐
                                         │ Notify user:    │
                                         │ "Update ready"  │
                                         └────────┬────────┘
                                                  │
                                                  ▼
                                         ┌─────────────────┐
                                         │ User clicks     │
                                         │ "Install"       │
                                         └────────┬────────┘
                                                  │
                                                  ▼
                                         ┌─────────────────┐
                                         │ Backup database │
                                         └────────┬────────┘
                                                  │
                                                  ▼
                                         ┌─────────────────┐
                                         │ Launch updater, │
                                         │ quit app        │
                                         └────────┬────────┘
                                                  │
                                                  ▼
                                         ┌─────────────────┐
                                         │ Replace binary, │
                                         │ restart app     │
                                         └─────────────────┘
```

---

## File Locations

### Windows
```
%APPDATA%\SlipStream\
├── config.yaml
├── slipstream.db
├── slipstream.db.backup    (pre-update backup)
└── logs\
    ├── slipstream.log
    ├── slipstream.log.1
    └── ...
```

### macOS (Future)
```
~/Library/Application Support/SlipStream/
├── config.yaml
├── slipstream.db
└── logs/
```

### Linux (Future)
```
~/.config/slipstream/
├── config.yaml
├── slipstream.db
└── logs/
```

### Docker (Future)
```
/config/                    (mounted volume)
├── config.yaml
├── slipstream.db
└── logs/
```
