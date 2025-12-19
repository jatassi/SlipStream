# SlipStream - Agent Quick Start Guide

> This document provides AI agents with comprehensive context about the SlipStream codebase to minimize exploration time at the start of each session.

## Project Overview

**SlipStream** is a unified media management application that consolidates the functionality of Sonarr (TV management), Radarr (Movie management), and Prowlarr (Indexer management) into a single, user-friendly interface.

**Current Version:** 0.0.1-dev

### Design Goals
1. **Single binary distribution** - No external dependencies, all assets embedded
2. **Cross-platform support** - Windows, macOS, Linux
3. **User-friendly** - Designed for non-technical users
4. **Graceful degradation** - External service failures don't crash the app
5. **API-first design** - Enables future CLI and mobile apps

---

## Tech Stack

### Backend
| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | Go 1.25+ | Main backend |
| HTTP Framework | Echo v4 | REST API server |
| Database | SQLite 3 (WAL mode) | Data storage |
| SQLite Driver | modernc.org/sqlite | Pure Go (no CGO) |
| Query Builder | sqlc | Type-safe SQL |
| Migrations | Goose v3 | Schema management |
| WebSocket | gorilla/websocket | Real-time updates |
| Config | Viper | Configuration |
| Logging | zerolog | Structured logging |
| Auth | golang-jwt/jwt v5 | JWT tokens |

### Frontend
| Component | Technology | Purpose |
|-----------|------------|---------|
| Framework | React 19 | UI library |
| Build Tool | Vite 7.2 | Dev server & bundler |
| Language | TypeScript 5.9 | Type safety |
| Router | TanStack Router | Client-side routing |
| Server State | TanStack Query | API caching |
| Client State | Zustand | Local state |
| UI Components | shadcn/ui (Base UI) | Component library |
| Styling | Tailwind CSS v4 | Utility CSS |
| Forms | React Hook Form + Zod | Form handling |
| Icons | Lucide React | Icon library |

---

## Directory Structure

```
SlipStream/
├── cmd/slipstream/
│   └── main.go                      # Application entry point
│
├── internal/                         # Private application code
│   ├── api/
│   │   ├── server.go                # HTTP server setup, all route registration
│   │   ├── handlers/                # Generic handlers (health, status)
│   │   └── middleware/              # HTTP middleware (auth, logging, CORS)
│   │
│   ├── config/
│   │   └── config.go                # Viper configuration loading
│   │
│   ├── database/
│   │   ├── database.go              # SQLite connection, migrations
│   │   ├── migrations/
│   │   │   └── 001_initial_schema.sql
│   │   ├── queries/                 # sqlc SQL query definitions
│   │   │   ├── movies.sql
│   │   │   ├── series.sql
│   │   │   ├── download_clients.sql
│   │   │   ├── indexers.sql
│   │   │   └── settings.sql
│   │   └── sqlc/                    # Auto-generated Go code (DO NOT EDIT)
│   │       ├── db.go
│   │       ├── models.go
│   │       └── *.sql.go
│   │
│   ├── library/
│   │   ├── movies/
│   │   │   ├── service.go           # Movie business logic
│   │   │   ├── handlers.go          # Movie HTTP handlers
│   │   │   └── movie.go             # Movie domain models
│   │   │
│   │   ├── tv/
│   │   │   ├── service.go           # TV series business logic
│   │   │   ├── handlers.go          # TV HTTP handlers
│   │   │   └── series.go            # Series/Season/Episode models
│   │   │
│   │   ├── scanner/
│   │   │   ├── scanner.go           # File system scanning
│   │   │   ├── parser.go            # Filename parsing (title, year, quality)
│   │   │   └── extensions.go        # Video file detection
│   │   │
│   │   ├── quality/
│   │   │   ├── profile.go           # Quality tier definitions (17 levels)
│   │   │   ├── service.go           # Quality profile CRUD
│   │   │   └── handlers.go          # Quality profile HTTP handlers
│   │   │
│   │   ├── rootfolder/
│   │   │   ├── service.go           # Root folder operations
│   │   │   └── handlers.go          # Root folder HTTP handlers
│   │   │
│   │   └── organizer/
│   │       ├── organizer.go         # File organization logic
│   │       └── templates.go         # Naming templates
│   │
│   ├── metadata/                    # TMDB/TVDB integration (planned)
│   │   ├── tmdb/
│   │   └── tvdb/
│   │
│   ├── downloader/                  # Download client integration (planned)
│   │   ├── client.go                # Client interface
│   │   ├── torrent/                 # qBittorrent, Transmission, etc.
│   │   └── usenet/                  # SABnzbd, NZBGet
│   │
│   ├── indexer/                     # Indexer integration (planned)
│   │   ├── indexer.go
│   │   ├── torznab/
│   │   └── newznab/
│   │
│   ├── websocket/
│   │   └── hub.go                   # WebSocket hub for real-time updates
│   │
│   ├── auth/                        # JWT authentication
│   └── logger/                      # Structured logging setup
│
├── web/                             # React frontend
│   ├── src/
│   │   ├── main.tsx                 # React entry point
│   │   ├── App.tsx                  # Root component
│   │   ├── index.css                # Tailwind styles
│   │   ├── components/ui/           # shadcn/ui components
│   │   ├── features/                # Feature modules (planned)
│   │   ├── hooks/                   # Custom React hooks
│   │   ├── lib/                     # Utilities, API client
│   │   └── routes/                  # TanStack Router (planned)
│   ├── package.json
│   └── public/
│
├── configs/
│   └── config.example.yaml          # Example configuration
│
├── data/                            # Runtime data (gitignored)
│   └── slipstream.db                # SQLite database
│
├── bin/                             # Build output (gitignored)
│
├── Makefile                         # Build commands
├── go.mod / go.sum                  # Go dependencies
├── sqlc.yaml                        # sqlc configuration
├── package.json                     # Root npm scripts
├── ARCHITECTURE.md                  # Architecture documentation
└── TODO.md                          # Development progress tracker
```

---

## Development Commands

### Makefile Targets
```bash
make dev              # Run backend (8080) + frontend (3000) in parallel
make dev-backend      # Run Go backend only
make dev-frontend     # Run Vite dev server only

make build            # Build both backend and frontend
make build-backend    # Build Go binary to bin/slipstream
make build-frontend   # Build React app to web/dist/

make install          # Install Go + npm dependencies
make clean            # Remove build artifacts
make help             # Show all targets
```

### Database Commands
```bash
# Regenerate sqlc code after modifying queries/*.sql
sqlc generate
# or
~/go/bin/sqlc generate
```

### Frontend Commands (from web/ directory)
```bash
npm run dev           # Start Vite dev server
npm run build         # Production build
npm run lint          # Run ESLint
```

---

## Configuration

### File Locations
- **User config:** `config.yaml` (root, gitignored)
- **Example:** `configs/config.example.yaml`
- **Environment prefix:** `SLIPSTREAM_*`

### Configuration Structure
```yaml
server:
  host: 0.0.0.0          # Bind address
  port: 8080             # HTTP port

database:
  path: ./data/slipstream.db

logging:
  level: info            # debug, info, warn, error
  format: console        # console or json

auth:
  jwt_secret: ""         # Auto-generated if empty
```

### Loading Priority (highest to lowest)
1. Environment variables (`SLIPSTREAM_SERVER_PORT`)
2. Config file
3. Default values

---

## Database Schema

### Core Tables (15 total)

**Authentication & Settings:**
- `auth` - Single-user authentication (password_hash)
- `settings` - Key-value system configuration

**Library Management:**
- `quality_profiles` - Quality definitions with cutoff thresholds
- `root_folders` - Library storage paths (movie or tv type)

**Movies:**
- `movies` - Movie library (tmdb_id, path, status, monitored)
- `movie_files` - File tracking (path, size, codec, resolution)

**TV Shows:**
- `series` - TV shows (tvdb_id, tmdb_id, path, status)
- `seasons` - Season tracking with monitoring
- `episodes` - Episode tracking (air_date, monitored)
- `episode_files` - Episode file metadata

**Integrations:**
- `indexers` - Search indexer configs (torznab/newznab)
- `download_clients` - Download client configs

**Activity:**
- `downloads` - Download queue tracking
- `history` - Activity log
- `jobs` - Scheduled task definitions

### Key Relationships
- `movies.root_folder_id` → `root_folders.id`
- `movies.quality_profile_id` → `quality_profiles.id`
- `movie_files.movie_id` → `movies.id` (CASCADE delete)
- `series.root_folder_id` → `root_folders.id`
- `seasons.series_id` → `series.id` (CASCADE delete)
- `episodes.series_id` → `series.id` (CASCADE delete)
- `episode_files.episode_id` → `episodes.id` (CASCADE delete)
- `downloads.client_id` → `download_clients.id`

### SQLite Configuration
- **WAL mode** enabled for concurrent reads
- **Foreign keys** enforced
- **Busy timeout:** 5 seconds
- **Single connection** (SQLite limitation)

---

## API Endpoints

**Base URL:** `http://localhost:8080`

### Health & Status
| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/status` | App status (version, counts) |

### Movies (`/api/v1/movies`)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | List movies (pagination) |
| POST | `/` | Create movie |
| GET | `/:id` | Get movie details |
| PUT | `/:id` | Update movie |
| DELETE | `/:id` | Delete movie |
| GET | `/:id/files` | Get movie files |
| POST | `/:id/files` | Add file to movie |
| DELETE | `/:id/files/:fileId` | Remove file |

### Series (`/api/v1/series`)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | List series (pagination) |
| POST | `/` | Add series |
| GET | `/:id` | Get series with seasons/episodes |
| PUT | `/:id` | Update series |
| DELETE | `/:id` | Delete series |
| GET | `/:id/seasons` | Get seasons |
| GET | `/:id/seasons/:season/episodes` | Get episodes |
| PUT | `/:id/seasons/:season/monitor` | Toggle season monitoring |

### Quality Profiles (`/api/v1/qualityprofiles`)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | List profiles |
| POST | `/` | Create profile |
| GET | `/:id` | Get profile |
| PUT | `/:id` | Update profile |
| DELETE | `/:id` | Delete profile |

### Root Folders (`/api/v1/rootfolders`)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | List folders |
| POST | `/` | Add folder |
| GET | `/:id` | Get folder |
| PUT | `/:id` | Update folder |
| DELETE | `/:id` | Delete folder |
| POST | `/:id/scan` | Scan for media |

### Settings (`/api/v1/settings`)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Get all settings |
| PUT | `/` | Update settings |

### Indexers (`/api/v1/indexers`) - Placeholder
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | List indexers |
| POST | `/` | Add indexer |
| GET | `/:id` | Get indexer |
| PUT | `/:id` | Update indexer |
| DELETE | `/:id` | Delete indexer |
| POST | `/:id/test` | Test indexer |

### Download Clients (`/api/v1/downloadclients`) - Placeholder
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | List clients |
| POST | `/` | Add client |
| GET | `/:id` | Get client |
| PUT | `/:id` | Update client |
| DELETE | `/:id` | Delete client |
| POST | `/:id/test` | Test client |

### Other
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/queue` | Download queue |
| DELETE | `/api/v1/queue/:id` | Remove from queue |
| GET | `/api/v1/history` | Activity history |
| GET | `/api/v1/search` | Search media |
| WS | `/ws` | WebSocket connection |

---

## Architecture Patterns

### Backend Service Pattern
Each domain follows a consistent structure:
```
internal/library/{domain}/
├── service.go    # Business logic, database calls
├── handlers.go   # HTTP handlers (Echo context)
└── {model}.go    # Domain models and types
```

**Service dependencies are injected:**
```go
type MovieService struct {
    queries *sqlc.Queries
    hub     *websocket.Hub
}

func NewMovieService(queries *sqlc.Queries, hub *websocket.Hub) *MovieService
```

### Error Handling
- Typed errors: `var ErrMovieNotFound = errors.New("movie not found")`
- Errors wrapped with context: `fmt.Errorf("failed to get movie: %w", err)`
- Services return `(*Model, error)` tuples

### Null Handling
Database optional fields use `sql.Null*` types:
```go
sql.NullInt64, sql.NullString, sql.NullTime
```

### WebSocket Events
The Hub broadcasts events on data changes:
- `movie:added`, `movie:updated`, `movie:deleted`
- `series:added`, `series:updated`, `series:deleted`
- `scan:progress`, `scan:complete`
- `download:started`, `download:completed`, `download:failed`

### Database Workflow
1. Write SQL in `internal/database/queries/*.sql`
2. Run `sqlc generate` to create Go code
3. Use generated `*sqlc.Queries` in services

---

## Quality Profiles

17 predefined quality tiers (lowest to highest):
1. SDTV
2. DVD
3. WEBRip-480p
4. WEBDL-480p
5. Bluray-480p
6. HDTV-720p
7. WEBRip-720p
8. WEBDL-720p
9. Bluray-720p
10. HDTV-1080p
11. WEBRip-1080p
12. WEBDL-1080p
13. Bluray-1080p
14. Remux-1080p
15. HDTV-2160p
16. WEBDL-2160p
17. Bluray-2160p / Remux-2160p

## Important Notes

1. **sqlc generated code** in `internal/database/sqlc/` should never be edited manually
2. **Frontend is scaffold only** - basic layout exists but no connected functionality
3. **Single-user auth** - no multi-user support currently
4. **Metadata providers not integrated** - TMDB/TVDB work is pending
5. **Download clients are placeholders** - interfaces defined but not implemented
6. **SQLite is single-writer** - only one connection for writes at a time
