# CLAUDE.md

### Making Changes
When making changes to this file, ALWAYS make the same update to AGENTS.md in the same directory. Same applies to all scoped CLAUDE.md/AGENTS.md pairs (`web/`, `internal/`).

## Project Overview

SlipStream is a unified media management system (similar to Sonarr/Radarr) with a Go backend and React frontend. It manages movies and TV shows, integrates with metadata providers (TMDB/TVDB), and supports torrent/usenet download clients. Still under initial development — prioritize clean code over backward compatibility.

## Scoped Documentation

- `web/CLAUDE.md` — Frontend patterns, design system, component gotchas
- `internal/CLAUDE.md` — Backend patterns, service layer, auto search pipeline
- `docs/` — Feature specs and design documents

## Project Structure

- `cmd/slipstream/` — Application entrypoint
- `internal/` — All backend packages (services, handlers, database, etc.)
- `web/` — React frontend (Vite + TanStack Router + TanStack Query)
- `configs/` — Config files and .env
- `scripts/` — Build, lint, and utility scripts

## Mandatory Directives

- Code should be self-documenting — avoid frivolous comments
- Do not conduct your own testing after implementation unless explicitly instructed. Suggest testing to the user if necessary.
- After completing major changes, run the relevant linter(s): `make lint` for Go, `cd web && bun run lint` for frontend. Fix any issues introduced by your changes.
- At the end of each message, put **Restart Backend** if a backend restart is required
- If following an external plan or to-do list, always update it once you finish a task/phase/feature
- Deviation from the plan or spec requires pausing to ask the user's approval. If approved, update the plan/spec before making code changes
- When a tool use fails, think carefully about the root cause, find the issue, and improve the common commands sections of AGENTS.md and CLAUDE.md to avoid the failure in the future
- When completing a major unit of work, check for and remove legacy codepaths that have been replaced by your changes
- **Do NOT commit, tag, or release unless explicitly instructed by the user**

## Common Commands

Do not start servers. Assume they're running on default ports (backend :8080, frontend :3000). Prompt user to restart as needed.

### Build & Test
```bash
make build                # Build both backend and frontend
make test                 # Run all Go tests
make test-verbose         # Verbose output
go test -v -run TestName ./internal/package/...  # Single test
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

### Database
SQLite with WAL mode. Migrations via Goose (embedded). Queries via sqlc (type-safe generated Go).

### Configuration
Priority: environment variables > `.env` file > config.yaml > defaults
- Env vars use `SLIPSTREAM_` prefix: `SLIPSTREAM_SERVER_PORT`, `SLIPSTREAM_METADATA_TMDB_API_KEY`, etc.
- Config file: `--config` flag or `configs/config.yaml`

### Logging
Development (`go run`): log level forced to `debug` (detected via "go-build" in executable path). Production: configured level (default `info`).

### API
All endpoints under `/api/v1`. Route definitions in `internal/server/routes.go`.

### Frontend-Backend Communication
- HTTP: Vite proxies `/api` to backend (:8080)
- WebSocket: `/ws` for real-time library/progress updates
- State: TanStack Query for server data, Zustand for client state

## Developer Mode

Toggle via hammer icon in header. Switches to separate dev database (`slipstream_dev.db`), creates mock services (metadata, indexer, download client, notifications), mock root folders with virtual filesystem. Backend check: `dbManager.IsDevMode()`.

## External Requests Portal

Separate auth from admin: admin uses session/cookies, portal users use JWT/localStorage. Request lifecycle: `pending` -> `approved` -> `downloading` -> `available` (or `denied`/`cancelled`). Quality profiles with `allowAutoApprove` skip the approval queue.
