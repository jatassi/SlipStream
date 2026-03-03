# Backend Agent Documentation

### Making Changes
When making changes to this file, ALWAYS make the same update to AGENTS.md in the same directory.

## Architecture

Handlers -> Services -> sqlc queries. Services wired via google/wire in `internal/api/` (`wire.go` -> `wire_gen.go`). Circular/late-binding deps in `setters.go`. Routes in `routes.go` are purely declarative.

## Logging

Use `github.com/rs/zerolog` (`*zerolog.Logger`) for all logging. Do NOT use `log/slog`. Zerolog syntax:
```go
s.logger.Info().Str("key", val).Int("count", n).Msg("message")
s.logger.Warn().Err(err).Str("key", val).Msg("something failed")
```

**Component tags:** Every logger must have a `Str("component", "xxx")` tag identifying its top-level subsystem (e.g., `api`, `database`, `scheduler`, `cardigann`, `startup`, `updater`). Use `Str("service", "xxx")` for sub-granularity within a component. Pre-zerolog bootstrap logs use `[component]` prefix in the format string (e.g., `[bootstrap]`, `[api]`).

## Database

SQLite, WAL mode, Goose migrations (embedded), sqlc-generated queries. After modifying `internal/database/queries/*.sql`, run:
```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

### Testing Helpers
- `testutil.NewTestDB(t)` creates isolated test DB with all migrations applied
- `sqlc.New(tdb.Conn)` for direct DB manipulation in tests
- `defer tdb.Close()` for cleanup

### Common Gotchas
- Episodes table has NO `updated_at` column
- `CreateSeriesParams` has no `Status` field (status is computed from episodes)
- `CreateEpisodeParams.Title` is `sql.NullString`, not `string`
- `CreateSeriesParams.ProductionStatus` is a required `string` field ("ended", "continuing", etc.)
- `LinkRequestToMedia` SQL sets `status = 'available'` as a side effect — reset after if testing download lifecycle
- Portal requests require a portal user first (FK on `requests.user_id` -> `portal_users.id`)
- Nil slices serialize to JSON `null`, not `[]` — always initialize slice fields that reach the frontend (e.g., `Errors: []string{}`)

## Auto Search & Upgrade Pipeline

Bugs here cause repeated erroneous downloads. Understand the full flow before modifying.

**Key files:**
- `internal/autosearch/service.go` — Search/grab logic, SearchableItem construction
- `internal/autosearch/scheduled.go` — Scheduled task, item collection
- `internal/indexer/scoring/scorer.go` — Release scoring, quality matching
- `internal/indexer/search/router.go` — Search routing (Prowlarr vs direct)
- `internal/import/pipeline.go` — Post-download import, upgrade validation

**TV upgrade search flow (most complex path):**
1. Scheduled task groups upgradable episodes by season
2. All eps upgradable in a season -> `SearchSeasonUpgrade` (season pack + upgrade checks)
3. Creates `SearchableItem` with `HasFile=true` and `CurrentQualityID` set
4. `selectBestRelease`: quality filter -> upgrade check (skipped if `!HasFile`)
5. No season pack found -> fallback to individual `SearchEpisode`
6. `SearchEpisode` E01 fallback retries as season pack for **missing** episodes only (`!item.HasFile`)

**Critical invariant:** When media has a file, `SearchableItem` MUST have `HasFile=true` and `CurrentQualityID` set to the highest quality across all file records. Without this, upgrade check is silently skipped.

**Pitfalls:**
- `seriesToSeasonPackItem` does NOT set `HasFile`/`CurrentQualityID` — callers must set explicitly
- Use highest `quality_id` across all files (duplicates exist from re-imports), never `files[0]`
- E01 season pack fallback must NOT run for upgradable episodes
- `selectBestRelease` only checks upgrades when `item.HasFile` is true

**Quality system:**
- Each quality has ID and weight (e.g., WEBDL-2160p = ID 15, weight 15)
- `profile.IsUpgrade(currentID, releaseID)` — strictly better check
- `profile.IsAcceptable(qualityID)` — allowed in profile check
- Cutoff determines "upgradable" vs "available" boundary

## Dependency Injection (Wire)

Key files in `internal/api/`: `wire.go` (providers, source of truth), `wire_gen.go` (generated — DO NOT EDIT), `setters.go` (circular/late-binding deps), `service_groups.go` (group structs), `switchable.go` (dev mode DB-switching registry).

**Adding a service:** constructor with all deps as params -> add to `wire.Build()` in `wire.go` -> add field to group struct -> `make wire`. If switchable: add tagged field in `switchable.go`. If circular: add setter in `setters.go`.

**Shared interfaces** in `internal/domain/contracts/`: `Broadcaster`, `HealthService`, `StatusChangeLogger`, `FileDeleteHandler`, `QueueTrigger`. Use instead of local copies. Add compile-time checks: `var _ contracts.X = (*MyType)(nil)`

## Dev Mode Switching

`DevModeManager` in `devmode.go` handles toggle logic. `SwitchableServices` in `switchable.go` uses struct tags (`switchable:"db"` / `switchable:"queries"`) with reflection-based `UpdateAll()`. New switchable service: add tagged field — reflection handles the rest. `Validate()` catches unassigned fields at startup.

**Bridge files** (`internal/api/`): `notification_bridge.go`, `portal_bridge.go`, `history_bridge.go`, `slot_bridge.go`

## Statuses

- Media: unreleased, missing, downloading, failed, upgradable, available
- Aggregate priority: downloading > failed > missing > upgradable > available > unreleased
- Requests: pending, approved, searching, denied, downloading, failed, available
