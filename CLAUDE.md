# CLAUDE.md

### Making Changes
When making changes to this file, ALWAYS make the same update to AGENTS.md in the same directory. Same applies to all scoped CLAUDE.md/AGENTS.md pairs (`web/`, `internal/`).

## Project Overview

SlipStream is a unified media management system (similar to Sonarr/Radarr) with a Go backend and React frontend. It manages movies and TV shows, integrates with metadata providers (TMDB/TVDB), and supports torrent/usenet download clients. Still under initial development — prioritize clean code over backward compatibility.

**Cross-platform:** Targets Windows, macOS, Linux and all mainstream browsers. Handle both `\` and `/` path separators — external data (e.g., Radarr/Sonarr DBs) may originate from a different OS. Avoid platform- or browser-specific APIs without fallbacks.

## Scoped Documentation

- `web/CLAUDE.md` — Frontend patterns, design system, component gotchas
- `internal/CLAUDE.md` — Backend patterns, service layer, module framework, auto search pipeline
- `docs/adding-a-module.md` — Step-by-step guide for adding a new media module
- `docs/` — Feature specs and design documents

## Project Structure

- `cmd/slipstream/` — Application entrypoint
- `internal/` — All backend packages (services, handlers, database, etc.)
- `internal/module/` — Module framework: interfaces, registry, schema, migrations
- `internal/modules/` — Module implementations (`movie/`, `tv/`, `shared/`)
- `web/` — React frontend (Vite + TanStack Router + TanStack Query)
- `web/src/modules/` — Frontend module configs and registry
- `configs/` — Config files and .env
- `internal/domain/contracts/` — Shared service interfaces
- `internal/portal/provisioner/` — Media provisioning service
- `scripts/` — Build, lint, and utility scripts
- `scripts/new-module/` — Scaffolding tool for new modules

## Mandatory Directives

- Avoid frivolous comments - code should be self-documenting
- After completing major changes, run the relevant linter(s): `make lint` for Go, `cd web && bun run lint` for frontend. Fix any issues introduced by your changes.
- At the end of each message, put **Restart Backend** if a backend restart is required
- If following an external plan or to-do list, always update it once you finish a task/phase/feature
- When completing a major unit of work, check for and remove legacy codepaths that have been replaced by your changes
- **Do NOT commit, tag, or release unless explicitly instructed by the user**
- **Strongly avoid** fallback code, "defense-in-depth", and backwards compatibility. Instead, code should be predictable so overly defensive code isn't required.

## Common Commands

Do not start servers. Assume they're running on default ports (backend :8080, frontend :3000). Prompt user to restart as needed.

**Important:** The frontend uses `bun` as its package manager and runtime. Always use `bun` instead of `npm`, `npx`, or `node` for frontend commands (e.g., `bun run`, `bun add`, `bun install`).

### Build & Test
```bash
make build                # Build both backend and frontend
make test                 # Run all Go tests
make test-verbose         # Verbose output
go test -v -run TestName ./internal/package/...  # Single test
make wire                 # Regenerate Wire DI (after changing wire.go)
```

### Database
```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```
After modifying `internal/database/queries/*.sql`, run sqlc generate to update `internal/database/sqlc/`.

### Linting
```bash
make lint                  # Go: golangci-lint
make lint-fix              # Go: with auto-fix
cd web && bun run lint     # Frontend: ESLint
cd web && bun run lint:fix # Frontend: with auto-fix
```
Additional lint helper scripts in `scripts/lint/`.

## Key Patterns

### Module System
Media types (movie, TV, etc.) are implemented as pluggable modules. Each module is a Go struct satisfying the composite `module.Module` interface (16 required sub-interfaces) registered in a central `module.Registry`. The frontend mirrors this with `ModuleConfig` objects in `web/src/modules/`. Modules own their own DB migrations, quality definitions, search strategies, naming templates, and scheduled tasks. See `docs/adding-a-module.md` for the full guide, `internal/module/interfaces.go` for interface definitions, and `scripts/new-module/` for scaffolding.

### Database
SQLite with WAL mode. Framework migrations via Goose (embedded). Per-module migrations in `internal/modules/<id>/migrations/` with independent version tables. Queries via sqlc (type-safe generated Go).

### Configuration
Priority: environment variables > `.env` file > config.yaml > defaults
- Env vars use `SLIPSTREAM_` prefix: `SLIPSTREAM_SERVER_PORT`, `SLIPSTREAM_METADATA_TMDB_API_KEY`, etc.
- Config file: `--config` flag or `configs/config.yaml`

### Logging
Development (`go run`): log level forced to `debug` (detected via "go-build" in executable path). Production: configured level (default `info`).

### API
All endpoints under `/api/v1`. Route definitions in `internal/api/routes.go`.

### Frontend-Backend Communication
- HTTP: Vite proxies `/api` to backend (:8080)
- WebSocket: `/ws` for real-time library/progress updates
- State: TanStack Query for server data, Zustand for client state

## Developer Mode

Toggle via hammer icon in header. Switches to separate dev database (`slipstream_dev.db`), creates mock services (metadata, indexer, download client, notifications), mock root folders with virtual filesystem. Backend check: `dbManager.IsDevMode()`.

## External Requests Portal

Separate auth from admin: admin uses session/cookies, portal users use JWT/localStorage. Request lifecycle: `pending` -> `approved` -> `searching` -> `downloading` -> `available` (or `denied`/`cancelled`). The `searching` status is set during auto search after approval; manual search grabs skip it. Quality profiles with `allowAutoApprove` skip the approval queue.

## Production Debugging

Logs and database from a production SlipStream instance are available for debugging purposes. You may only read from the database. Do not access these files unless the user explicitly mentions an error happening in production or otherwise instructs you to access them.

- `/Volumes/Users/jacks/AppData/Local/SlipStream/slipstream.db`
- `/Volumes/Users/jacks/AppData/Local/SlipStream/logs/slipstream.log`

## Subagents

When launching subagents, prefer launching in synchronous mode (especially when parallelizing). When lauching subagents to perform targeted code changes, prefer using Sonnet. ALWAYS instruct subagents that they are forbidden to use git stash commands or similar commands that affect the entire worktree and could interfere with the work of other agents.