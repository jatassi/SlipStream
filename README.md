# SlipStream

SlipStream is a self-hosted media management system — the kind of tool that watches for new releases, pulls them from your indexers, hands them off to your download client, and files the results away under a tidy library. It's built around a pluggable **module system**, so support for a new media type is a self-contained package rather than a fork.

Today it ships with first-class Movie and TV modules. Music, books, comics, games and anything else live as future modules that slot into the same machinery.

## What it does

- **Library management** — track what you have, what you're missing, and what can still be upgraded.
- **Metadata** from TMDB, TVDB, and OMDb (for Rotten Tomatoes / IMDB ratings).
- **Search and grab** across indexers with quality profiles, cutoffs, and multiple simultaneous quality versions per item. For TV, season packs fall back cleanly to individual episodes.
- **Indexers** — native Prowlarr integration (with caching, rate limiting, and capability polling), Cardigann-style YAML definitions, and generic RSS feeds.
- **Download clients** — qBittorrent, Transmission, Deluge, rTorrent, Aria2, SABnzbd, and more.
- **Importing** — post-download file matching, rename, and move, plus one-shot library import from existing Radarr and Sonarr installs.
- **Requests portal** — a separate, JWT-authenticated surface where other users can request media, with per-user quotas, auto-approval rules, and module-specific settings.
- **Real-time UI** — a WebSocket feed keeps library and download progress live without polling.
- **Notifications** — Discord, Telegram, Slack, Email, Webhook, Pushover, and Plex.
- **Calendar, history, and health monitoring** for indexers and download clients.
- **Developer mode** — a sandbox with its own database, mock services, and a virtual filesystem, toggleable from the UI.

## Tech stack

**Backend:** Go 1.25, Echo for HTTP, Google Wire for DI, SQLite in WAL mode with Goose migrations and sqlc-generated queries, zerolog, Viper, gocron. Auth uses session cookies for admins and JWT + WebAuthn passkeys for portal users.

**Frontend:** React 19 and TypeScript on Vite, TanStack Router and TanStack Query, Zustand for client state, Tailwind CSS 4, and shadcn components built on Base UI. Package management is Bun.

Everything runs from a single SQLite file and targets Windows, macOS, and Linux.

## Getting started

You'll need Go 1.25+, Bun, and Git.

```bash
git clone <repo-url> && cd SlipStream
make install                # go mod download + bun install

cp .env.example .env        # add your TMDB / TVDB API keys
make dev                    # backend on :8080, frontend on :3000
```

Then open <http://localhost:3000>.

First-time setup scripts are available if you'd rather have the toolchain installed for you:

| Platform | Setup               | Run          |
| -------- | ------------------- | ------------ |
| Windows  | `.\setup.bat`       | `.\dev.bat`  |
| macOS    | `bash setup-mac.sh` | `make dev`   |
| Linux    | `make install`      | `make dev`   |

## Configuration

Config is resolved from, in order: environment variables → `.env` → `configs/config.yaml` → defaults. Env vars use the `SLIPSTREAM_` prefix with underscores for nested keys:

```bash
SLIPSTREAM_SERVER_PORT=8080
SLIPSTREAM_DATABASE_PATH=./data/slipstream.db
SLIPSTREAM_LOGGING_LEVEL=info                 # debug | info | warn | error
SLIPSTREAM_METADATA_TMDB_API_KEY=...
SLIPSTREAM_METADATA_TVDB_API_KEY=...
SLIPSTREAM_METADATA_OMDB_API_KEY=...          # optional
SLIPSTREAM_AUTH_JWT_SECRET=...                # auto-generated if unset
```

See `configs/config.example.yaml` for the full schema. Metadata API keys and a `VERSION` string can also be baked into the binary at build time:

```bash
make build-backend VERSION=1.0.0 TMDB_API_KEY=... TVDB_API_KEY=... OMDB_API_KEY=...
```

## How it's built

The backend is a set of services wired together by Google Wire (`internal/api/wire.go` → `wire_gen.go`). HTTP routing, middleware, and handlers live in `internal/api/`; all endpoints are mounted under `/api/v1`, with a WebSocket hub at `/ws`.

Persistence is a single SQLite database. Framework schema migrations are managed with Goose and embedded into the binary. Each media module runs its own migrations against an independent version table, so modules evolve their schemas on their own schedule. Queries go through sqlc-generated code in `internal/database/sqlc/`.

Supporting services live in their own packages — metadata providers (`internal/metadata`), indexer clients and search routing (`internal/indexer`, `internal/prowlarr`, `internal/rsssync`), download clients (`internal/downloader`), the grab and import pipeline (`internal/autosearch`, `internal/import`), notifications (`internal/notification`), health monitoring (`internal/health`), the requests portal (`internal/portal`), and authentication (`internal/auth`). Cross-service interfaces like broadcasting and health reporting live in `internal/domain/contracts/` so individual services don't depend on each other's concrete types.

The frontend mirrors this shape under `web/src/`:

- `routes/` — TanStack Router routes, lazy-loaded
- `modules/` — one config object per media type, declaring routes, query keys, WebSocket invalidation rules, filter and sort options, table columns, and lazy-loaded page components
- `components/` — React components organized by feature
- `hooks/` — custom hooks split by admin vs. portal
- `stores/` — Zustand stores for client state
- `api/` — typed HTTP clients for the admin and portal APIs

## The module system

A media type is a Go struct that satisfies `module.Module` in `internal/module/interfaces.go` — a composite of sixteen smaller interfaces covering identity, metadata, search, import, path and naming templates, quality, monitoring, calendar, notifications, filename parsing, dev-mode mocks, HTTP routes, and scheduled tasks. Optional interfaces add portal provisioning, multi-version slot support, and Radarr/Sonarr import adapters.

Each module declares its own **node schema** (Movie is flat; TV nests `series → season → episode`), ships its own database migrations, and registers itself in a central `module.Registry` at startup. On the frontend, every module exports a matching `ModuleConfig` from `web/src/modules/<id>/index.ts`.

To scaffold a new module:

```bash
make new-module MODULE_ID=music
```

Then follow `docs/adding-a-module.md` for the full walkthrough. `docs/module-system-spec.md` has the deeper reference.

## Development commands

```bash
make dev                 # run backend and frontend together
make build               # build both (→ bin/slipstream + web/dist)
make test                # go test -race ./...
make test-coverage       # HTML report at coverage/coverage.html
make lint                # golangci-lint
make lint-fix            # ... with auto-fix
make wire                # regenerate Wire DI after editing providers
make new-module MODULE_ID=<id>
```

From `web/`:

```bash
bun run lint
bun run format
```

After editing `internal/database/queries/*.sql`, regenerate sqlc output:

```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

`make help` lists every target.

## Project layout

```
cmd/slipstream/     Application entrypoint
internal/           Backend packages
  module/           Module framework: interfaces, registry, schema
  modules/          Module implementations (movie, tv, shared)
  api/              HTTP handlers, routes, Wire DI
  library/          Core services: movies, tv, rootfolder, quality, scanner, organizer
  autosearch/       Scheduled search and grab pipeline
  indexer/          Search routing and indexer clients
  downloader/       Torrent and usenet client integrations
  import/           Post-download matching and file import
  metadata/         TMDB, TVDB, OMDb clients
  portal/           Requests portal, users, quotas, provisioner
  notification/     Event dispatch and notification channels
  auth/             Session, JWT, and WebAuthn
  database/         SQLite manager, Goose migrations, sqlc queries
  domain/contracts/ Cross-service interfaces
web/                React frontend (Vite)
  src/
    routes/  modules/  components/  hooks/  stores/  api/
configs/            config.example.yaml
docs/               Feature specs and design documents
scripts/            Build, lint, and scaffolding utilities
```

Most of the deeper reading lives in `docs/` — start with `adding-a-module.md` and `module-system-spec.md` for the extension story, or `automatic-search-spec.md` for the search and grab pipeline.
