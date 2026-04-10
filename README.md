# SlipStream

A unified media management system in the spirit of Sonarr/Radarr, built around a pluggable **module system** so new media types can be added without forking the core. Go backend, React frontend, single SQLite database, cross-platform (Windows, macOS, Linux).

> **Status:** early development. Clean design is prioritized over backward compatibility — APIs, schemas and module contracts are expected to change.

## Highlights

- **Modular media types.** Movies and TV are first-class today; each is a self-contained module owning its schema migrations, quality definitions, search strategy, file naming, scheduled tasks, and frontend config. Music, books, etc. can be added the same way.
- **Metadata providers.** TMDB (movies/TV), TVDB (TV), OMDb (ratings).
- **Automatic search & grab pipeline.** Quality profiles with upgrades and cutoffs, multi-quality version slots, season-pack ⇢ per-episode fallback for TV, grab locks to prevent duplicates.
- **Indexers.** Prowlarr (with caching, rate limiting, capability polling), Cardigann-style YAML indexers, and generic RSS.
- **Download clients.** qBittorrent, Transmission, Deluge, rTorrent, Aria2, SABnzbd, and more.
- **Importing.** Post-grab matching & import, plus Radarr / Sonarr library import.
- **Requests portal.** Separate JWT-authenticated user surface with quotas, auto-approval, and per-module settings. Lifecycle: `pending → approved → searching → downloading → available` (or `denied`/`cancelled`).
- **Auth.** Session cookies for admin; JWT + WebAuthn passkeys for portal users.
- **Real-time UI.** WebSocket broadcasts drive TanStack Query invalidation for library and progress updates.
- **Notifications.** Discord, Telegram, Slack, Email, Webhook, Pushover, Plex.
- **Calendar & history.** Unified calendar across modules; status-change and grab/import history.
- **Health monitoring.** Indexer and download-client health surface warnings/errors in the UI.
- **Developer mode.** Toggleable sandbox with its own SQLite DB (`slipstream_dev.db`), mock metadata/indexer/download/notification services, and a virtual filesystem.

## Tech Stack

**Backend** — Go 1.25, [Echo](https://echo.labstack.com) HTTP, [Google Wire](https://github.com/google/wire) DI, [sqlc](https://sqlc.dev) + [Goose](https://github.com/pressly/goose) over SQLite (WAL), [zerolog](https://github.com/rs/zerolog), [Viper](https://github.com/spf13/viper) config, [gocron](https://github.com/go-co-op/gocron) scheduling, [gorilla/websocket](https://github.com/gorilla/websocket), [go-webauthn](https://github.com/go-webauthn/webauthn), [modernc.org/sqlite](https://modernc.org/sqlite) (pure-Go, no CGO).

**Frontend** — React 19, TypeScript 5.9, Vite 7, [TanStack Router](https://tanstack.com/router) + [TanStack Query](https://tanstack.com/query), [Zustand](https://zustand.docs.pmnd.rs) for client state, Tailwind CSS 4, [Base UI](https://base-ui.com) via shadcn, [react-hook-form](https://react-hook-form.com) + [Zod](https://zod.dev). **Package manager: [Bun](https://bun.sh)** — not npm.

## Prerequisites

- **Go** ≥ 1.25
- **Bun** (used for all frontend commands — `bun`, not `npm`/`npx`)
- **Git**
- No SQLite system package needed — `modernc.org/sqlite` is pure Go.

## Quick Start

```bash
# 1. Clone and install deps
git clone <repo-url> && cd SlipStream
make install                 # go mod download + bun install

# 2. Configure metadata keys
cp .env.example .env
$EDITOR .env                 # set SLIPSTREAM_METADATA_TMDB_API_KEY, _TVDB_API_KEY

# 3. Run both servers
make dev                     # backend :8080, frontend :3000
```

Then open http://localhost:3000.

**Platform helper scripts** (install toolchains + first run):

| Platform | Setup               | Dev                  |
| -------- | ------------------- | -------------------- |
| Windows  | `.\setup.bat`       | `.\dev.bat`          |
| macOS    | `bash setup-mac.sh` | `make dev`           |
| Linux    | `make install`      | `make dev`           |

## Configuration

Resolution order (highest priority first): **environment variables → `.env` → `configs/config.yaml` → defaults**.

Environment variables are prefixed `SLIPSTREAM_` and nested keys use `_`:

```bash
SLIPSTREAM_SERVER_HOST=0.0.0.0
SLIPSTREAM_SERVER_PORT=8080
SLIPSTREAM_DATABASE_PATH=./data/slipstream.db
SLIPSTREAM_LOGGING_LEVEL=info               # debug | info | warn | error
SLIPSTREAM_METADATA_TMDB_API_KEY=...
SLIPSTREAM_METADATA_TVDB_API_KEY=...
SLIPSTREAM_METADATA_OMDB_API_KEY=...        # optional, adds Rotten Tomatoes / IMDB ratings
SLIPSTREAM_AUTH_JWT_SECRET=...              # auto-generated if unset
```

See `configs/config.example.yaml` for the full schema and `.env.example` for a starter env file.

Metadata keys (and `VERSION`) can also be **baked into the binary** at build time:

```bash
make build-backend VERSION=1.0.0 TMDB_API_KEY=xxx TVDB_API_KEY=yyy OMDB_API_KEY=zzz
```

## Make Targets

```bash
make dev                 # Run backend and frontend in parallel
make build               # make build-backend + make build-frontend
make build-backend       # → bin/slipstream (supports VERSION / *_API_KEY ldflags)
make build-frontend      # Vite production build
make install             # go mod download && bun install
make clean               # Remove bin/, web/dist/, coverage/

make test                # go test -race ./...
make test-unit           # scanner / quality / organizer
make test-integration    # library services + API
make test-coverage       # → coverage/coverage.html
make test-verbose

make lint                # golangci-lint
make lint-fix            # with --fix
make lint-new            # diff vs origin/main
make deadcode            # whole-program unreachable-function analysis

make wire                # regenerate Wire DI after editing internal/api/wire.go
make generate-slots      # regenerate slot SQL from templates
make new-module MODULE_ID=music   # scaffold a new media module
```

Frontend linting lives in `web/`:

```bash
cd web
bun run lint             # ESLint
bun run lint:fix
bun run format           # Prettier
```

After editing `internal/database/queries/*.sql`, regenerate sqlc output:

```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

## Project Layout

```
cmd/slipstream/             Application entrypoint
internal/
  api/                      HTTP handlers, routes, middleware, Wire DI (wire.go / wire_gen.go)
  module/                   Module framework: interfaces, registry, schema, migrations
  modules/                  Module implementations: movie/, tv/, shared/
  database/                 SQLite manager, Goose migrations, sqlc queries
  domain/contracts/         Cross-service interfaces (Broadcaster, HealthService, ...)
  library/                  Core services: movies, tv, rootfolder, quality, slots, scanner, organizer
  autosearch/               Scheduled auto-search + grab pipeline
  indexer/                  Search routing, scoring, grab; prowlarr / cardigann / genericrss clients
  downloader/               Torrent & usenet client integrations
  import/                   Post-download matching and file import
  metadata/                 TMDB / TVDB / OMDb clients
  portal/                   Portal auth, requests, users, quotas, provisioner
  notification/             Event dispatch + Discord/Telegram/Slack/Email/Webhook/Pushover/Plex
  auth/                     Session + JWT + WebAuthn (passkeys)
  websocket/                Real-time broadcast hub
  health/                   Indexer & download-client health tracking
  calendar/  history/       Unified calendar, event history
  prowlarr/  rsssync/       Prowlarr cache & RSS sync services
  config/  logger/  startup/  scheduler/  filesystem/  preferences/  update/
web/
  src/
    api/                    HTTP client functions (admin + portal)
    routes/                 TanStack Router routes (lazy-loaded)
    modules/                ModuleConfig registry (movie, tv)
    components/             React components by feature
    hooks/                  Per-feature custom hooks (admin + portal)
    stores/                 Zustand client-state stores
    lib/  types/
configs/                    config.example.yaml
docs/                       Feature specs, design documents, implementation plans
scripts/                    Build, lint and utility scripts; scripts/new-module/ scaffolder
resources/                  Static resources
```

## Module System (in brief)

A media type is a Go struct composed of ~16 sub-interfaces from `internal/module/interfaces.go`:

`ModuleDescriptor`, `MetadataProvider`, `SearchStrategy`, `ImportHandler`, `PathGenerator`, `NamingProvider`, `CalendarProvider`, `QualityDefinition`, `WantedCollector`, `MonitoringPresets`, `FileParser`, `MockFactory`, `NotificationEvents`, `ReleaseDateResolver`, `RouteProvider`, `TaskProvider` — plus optional `PortalProvisioner`, `SlotSupport`, `ArrImportAdapter`.

Each module:

- declares a **node schema** (flat for Movie, `series → season → episode` for TV),
- ships its **own Goose migrations** in `internal/modules/<id>/migrations/` with an independent `goose_db_version_<id>` table,
- is registered in a central `module.Registry`, and
- has a mirrored `ModuleConfig` in `web/src/modules/<id>/` exposing routes, query keys, WebSocket invalidation rules, filter/sort options, and table columns.

To add a module: `make new-module MODULE_ID=music`, then follow `docs/adding-a-module.md`.

## Documentation

Project-level agent instructions:

- `CLAUDE.md` / `AGENTS.md` (root) — mandatory directives and style guide
- `internal/CLAUDE.md` — backend patterns, service layer, module framework, auto-search
- `web/CLAUDE.md` — frontend patterns, design system (movie = orange, TV = blue), Base UI gotchas

Selected design docs in `docs/`:

- `adding-a-module.md` — guide for new modules
- `module-system-spec.md` + `module-system-plan-phase*.md` — module architecture and phased rollout
- `automatic-search-spec.md` — auto search and upgrade pipeline
- `multiple-quality-versions-spec.md` — quality version slots
- `external-requests-spec.md` — portal requests system
- `Prowlarr-integration-spec.md`, `RSS-Sync-Design.md`
- `system-health-spec.md`
- `Library-Import-Spec.md`, `release-importing-spec.md`
- `passkey-implementation-plan.md`
- `Windows-Application-Plan.md`, `macOS-Application-Plan.md`, `Linux-Application-Plan.md`

## Architecture Notes

- **API** — all endpoints under `/api/v1`; routes registered in `internal/api/routes.go`. WebSocket on `/ws`.
- **Database** — single SQLite file in WAL mode. Framework migrations via embedded Goose; each module runs its own migrations against an independent version table. All reads/writes use sqlc-generated type-safe queries.
- **DI** — Google Wire generates `internal/api/wire_gen.go` from `wire.go`. Circular/late-binding dependencies are handled in `setters.go`. Re-run `make wire` after changing providers.
- **Dev mode switching** — `DevModeManager` swaps in mock services at runtime (`SwitchableServices` in `internal/api/switchable.go`) and points at `slipstream_dev.db`. Toggle via the hammer icon in the header.
- **Logging** — zerolog everywhere with mandatory `component` tags. In `go run` the level is forced to `debug`; in built binaries it honors config.
- **Cross-platform paths** — both `\` and `/` separators are handled; imports from foreign Radarr/Sonarr DBs may originate on a different OS than the running server.
- **Frontend code-splitting** — routes are lazy-loaded via `lazyRouteComponent()` in `web/src/routes-config.tsx`; never eagerly import page components.

## Default Ports

| Service  | Port   |
| -------- | ------ |
| Backend  | `8080` |
| Frontend | `3000` |
| WebSocket| `/ws` on the backend |

The Vite dev server proxies `/api` and `/ws` to the backend.
