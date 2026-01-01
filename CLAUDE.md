# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SlipStream is a unified media management system (similar to Sonarr/Radarr) with a Go backend and React frontend. It manages movies and TV shows, integrates with metadata providers (TMDB/TVDB), and supports torrent/usenet download clients. This project is still undergoing initial development meaning that it is not necessary to maintain backward compatibility with existing implementations. Always prioritize neat, clean code over cumbersome backward compatible workarounds.

## Additional Documentation

Various documents detailing specific aspects of the application may be available in the `docs/` directory. When creating new documents, ALWAYS put them in this directory.

## Common Commands

### Development

**Windows:**
```powershell
.\dev.bat             # Run both backend (:8080) and frontend (:3000)
.\dev.ps1             # PowerShell alternative

# Or manually in separate terminals:
go run ./cmd/slipstream        # Backend only
cd web && npm run dev          # Frontend only
```

**Unix/Mac (with Make):**
```bash
make dev              # Run both backend (:8080) and frontend (:3000)
make dev-backend      # Run Go backend only
make dev-frontend     # Run Vite frontend only
```

### Building
```bash
make build            # Build both backend and frontend
make build-backend    # Build Go binary to bin/slipstream
make build-frontend   # Build frontend to web/dist/
```

### Testing
```bash
make test             # Run all Go tests
make test-verbose     # Run tests with verbose output
make test-unit        # Run scanner, quality, organizer tests
make test-integration # Run movies, tv, api tests
make test-coverage    # Generate coverage report (coverage/coverage.html)

# Run a single test
go test -v -run TestFunctionName ./internal/package/...
```

### Dependencies
```bash
make install          # Install Go and npm dependencies
go mod download       # Go dependencies only
cd web && npm install # Frontend dependencies only
```

### Database
```bash
sqlc generate         # Regenerate Go code from SQL queries
```

After modifying `internal/database/queries/*.sql`, run `sqlc generate` to update `internal/database/sqlc/`.

### Frontend
```bash
cd web
npm run dev           # Start dev server
npm run build         # Production build (runs tsc first)
npm run lint          # ESLint
```

## Architecture

```
cmd/slipstream/       # Application entry point (main.go)
internal/
├── api/              # Echo HTTP server, routes, middleware
├── auth/             # JWT authentication
├── config/           # Viper config (env → .env → yaml → defaults)
├── database/
│   ├── migrations/   # Goose SQL migrations
│   ├── queries/      # sqlc query definitions
│   └── sqlc/         # Generated query code (DO NOT EDIT)
├── library/
│   ├── movies/       # Movie CRUD service & handlers
│   ├── tv/           # Series/seasons/episodes service
│   ├── scanner/      # Filesystem media file detection
│   ├── organizer/    # File naming/path templates
│   ├── quality/      # Quality profile management
│   └── rootfolder/   # Library path management
├── metadata/
│   ├── tmdb/         # TMDB API client
│   └── tvdb/         # TVDB API client
├── indexer/          # Search provider abstraction (Torznab/Newznab)
├── downloader/       # Download client abstraction
│   ├── torrent/      # qBittorrent, Transmission, etc.
│   └── usenet/       # SABnzbd, NZBGet, etc.
├── websocket/        # Real-time updates hub
├── scheduler/        # Background jobs
└── logger/           # Zerolog setup
web/                  # React frontend
├── src/
│   ├── components/   # UI components (shadcn/ui)
│   └── lib/          # Utilities
└── vite.config.ts    # Dev server proxies /api→:8080, /ws→ws://:8080
```

## Key Patterns

### API Routes
All API endpoints are under `/api/v1`. Route groups: `/auth`, `/movies`, `/series`, `/qualityprofiles`, `/rootfolders`, `/metadata`, `/indexers`, `/downloadclients`, `/queue`, `/history`, `/search`.

### Database
- SQLite with WAL mode
- Migrations via Goose (embedded in binary)
- Queries via sqlc (type-safe generated Go)

### Configuration
Priority: environment variables > `.env` file > config.yaml > defaults
- Config file: `--config` flag or `configs/config.yaml`
- Env file: `configs/.env` or project root `.env`
- Env vars: `SERVER_PORT`, `METADATA_TMDB_API_KEY`, etc.

### Frontend-Backend Communication
- HTTP API: Vite dev server proxies `/api` to backend
- WebSocket: `/ws` endpoint for real-time library/progress updates
- State: TanStack Query for data fetching, Zustand for local state

### Service Layer
Handlers delegate to service structs (e.g., `movies.Service`, `metadata.Service`) which wrap sqlc queries. Services are injected into handlers during server setup.

## Testing Notes

Unit tests are in `*_test.go` files alongside source. Integration tests may use `internal/testutil` helpers. Scanner tests parse media filenames; quality tests validate profile matching logic.

## Developer Mode

SlipStream supports a developer mode for testing and debugging features. When enabled:
- Debug buttons are visible in the UI (e.g., download client dialogs)
- Mock download queue items can be created based on library content
- The `/api/v1/status` endpoint includes `developerMode: true`

### Best Practices

- Always check `developerMode` before exposing debug/testing features
- Mock data should use realistic values from the actual library
- Debug endpoints should return 403 Forbidden when not in developer mode

### Enabling Developer Mode

Set `DEVELOPER_MODE=true` in your `.env` file

### Frontend Hook

Use the `useDeveloperMode()` hook to check developer mode status in React components:
```typescript
import { useDeveloperMode } from '@/hooks'

function MyComponent() {
  const developerMode = useDeveloperMode()

  if (developerMode) {
    // Show debug features
  }
}
```

## Claude Code Notes

When running bash commands on Windows, use forward slashes for paths:
```bash
cd c:/Git/SlipStream/web && npm run build 2>&1
```
