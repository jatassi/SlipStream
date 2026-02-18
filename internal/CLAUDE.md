# Backend Agent Documentation

### Making Changes
When making changes to this file, ALWAYS make the same update to AGENTS.md in the same directory.

## Architecture

Handlers -> Services -> sqlc queries. Services are injected during server setup in `internal/server/`.

## Logging

Use `github.com/rs/zerolog` (`*zerolog.Logger`) for all logging. Do NOT use `log/slog`. Zerolog syntax:
```go
s.logger.Info().Str("key", val).Int("count", n).Msg("message")
s.logger.Warn().Err(err).Str("key", val).Msg("something failed")
```

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

## Statuses

- Media: unreleased, missing, downloading, failed, upgradable, available
- Aggregate priority: downloading > failed > missing > upgradable > available > unreleased
- Requests: pending, approved, denied, downloading, failed, available
