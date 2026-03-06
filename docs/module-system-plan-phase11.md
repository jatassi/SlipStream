# Phase 11: Contributor Tooling & Final Cleanup — Implementation Plan

## Overview

Eliminate remaining backend duplication, remove legacy fallback paths introduced during the incremental migration (Phases 0–10), and build contributor tooling for future module authors. By this phase, both movie and TV modules are fully functional through the module framework — all fallback/bridge code is dead weight.

**Spec sections covered:** §19.1 (scaffolding), §19.2 (validation CLI), §19.3 (moduletest harness), Appendix C (backend rows)

**Additionally covers all items deferred to Phase 11 from earlier phases** — see Deferred Items Registry below.

---

## Context Management Instructions

This phase has many independent cleanup tasks ideal for parallel subagents. The executing agent's role is **orchestration and validation only**.

### Delegation Strategy

- **Every numbered task below becomes a subagent.** Use `subagent_type: "general-purpose"` for tasks touching multiple packages. Use Sonnet for targeted single-file tasks.
- Copy the task description verbatim into the subagent prompt. Include the "Depends on" line and all listed file paths.
- Instruct every subagent: **"Do NOT use `git stash` or any commands that affect the entire worktree."**
- For tasks that modify SQL queries: instruct the subagent to run `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate` after changes.

### Parallelism Map

```
Group A (parallel):     11.1, 11.2, 11.3, 11.9, 11.10, 11.11
Group B (after 11.1):   11.4
Group C (after 11.2):   11.5, 11.6
Group D (after 11.5):   11.7
Group E (after all cleanup — 11.1–11.11): 11.12, 11.13, 11.14
Group F (after 11.12):  11.15
Group G (after 11.14):  11.16
Group H (after all):    11.17
```

### Validation Protocol

After each group completes:
1. Run `go build ./...` — catch compile errors.
2. Run `make test` — catch regressions.
3. Run `make lint` — catch style issues.
4. Spot-check 1–2 key modified files (~20 lines around changes). Do NOT read entire service files into main context.
5. If a subagent breaks build/test, resume that subagent with the error output.

After Group E (all cleanup done), run full validation: `make build && make test && make lint && cd web && bun run lint`.

### Context Hygiene

- Do NOT read large files (`service.go`, `wire_gen.go`, `slots.sql`) into the main agent context. Delegate all reads to subagents.
- The main agent should only read small files (<100 lines) for verification.
- When verifying a task, check that the listed "Verify" commands pass — don't manually review every line.

---

## Prerequisites

The following must be complete before Phase 11 begins:

- [ ] Phase 0: `internal/module/` package with all interfaces, types, registry, node schema
- [ ] Phase 2: Module-scoped quality profiles (`quality_profiles.module_type`)
- [ ] Phase 3: Module `MetadataProvider` implementations in `internal/modules/movie/metadata.go` and `internal/modules/tv/metadata.go`
- [ ] Phase 4: Module `MonitoringPresets`, `CalendarProvider`, `WantedCollector`, `ReleaseDateResolver` implementations
- [ ] Phase 5: Module `SearchStrategy` implementations; `module.SearchableItem` interface; bridge from `decisioning.SearchableItem` to module interface
- [ ] Phase 6: Module `FileParser`, `PathGenerator`, `NamingProvider`, `ImportHandler` implementations; `internal/module/parseutil/` package; `module_naming_settings` table
- [ ] Phase 7: Module `NotificationEvents` with JSON event toggles; generic WebSocket `Broadcast()` helper
- [ ] Phase 8: Module `PortalProvisioner`, `SlotSupport`; `slot_root_folders` pivot table
- [ ] Phase 9: Module `MockFactory`, `ArrImportAdapter` implementations
- [ ] Phase 10: Frontend module registry, generic list page, unified views, all frontend deduplication complete

---

## Deferred Items Registry

Every item deferred to Phase 11 from earlier phases, tracked to its task assignment.

### From Phase 10 "Reassigned to Phase 11"

| # | Deferred Item | Source | Assigned Task |
|---|---|---|---|
| 1 | Remove legacy `calendar.Service` helper functions (`getMovieEvents`, `getEpisodeEvents`, `resolveMovieStatus`, `movieRowToEvents`, `createEpisodeEvent`, `createSeasonReleaseEvent`) | Phase 4 (line 5475) | 11.9 |
| 2 | Consolidate `streamingServicesWithEarlyRelease` map duplicated between `internal/calendar/service.go` and `internal/modules/tv/calendar.go` | Phase 4 (line 5618) | 11.9 |
| 3 | Align `UpdateUnreleasedMoviesToMissing` SQL to use earliest-of digital/physical instead of priority chain | Phase 4 (line 5619) | 11.10 |
| 4 | Generic framework cascade (§6.1) — parent `monitored` propagation to descendants via node schema introspection | Phase 4 (line 5620) | 11.11 |
| 5 | Eliminate `decisioning.SearchableItem` struct — migrate autosearch pipeline to use `module.SearchableItem` interface throughout | Phase 5 (lines 6585, 6874–6875) | 11.5 |
| 6 | Refactor `SelectBestRelease` to use `SearchStrategy.FilterRelease` instead of hard-coded `shouldSkipTVRelease` | Phase 5 (line 6878) | 11.6 |
| 7 | Full scanner refactoring — `FileParser` replaces scanner internals instead of wrapping | Phase 6 (line 6978) | 11.7 |
| 8 | Extract shared naming template helpers (`qualityVariables()`, `mediaInfoVariables()`, `metadataVariables()`) to a shared location | Phase 6 (lines 7781, 7944) | 11.8 |
| 9 | Remove unused naming columns from `import_settings` table | Phase 6 (line 8034) | 11.8 |
| 10 | Remove `populateLegacyFields` bridge in renamer | Phase 6 (line 8221) | 11.8 |
| 11 | Remove legacy `availability.Service`, `calendar.Service`, `missing.Service` dispatcher fallback paths | Phase 4 (line 4324) | 11.9 |

### From Phase 3

| # | Deferred Item | Source | Assigned Task |
|---|---|---|---|
| 12 | Deduplicate `convertToSeasonMetadata` between `librarymanager/metadata_refresh.go`, `librarymanager/add.go`, and `internal/modules/tv/metadata.go` | Phase 3 (line 3972) | 11.10 |

### From Appendix C (Backend Rows)

| # | Duplicated Code | Spec Location | Assigned Task |
|---|---|---|---|
| 13 | `generateSortTitle()` in `movies/movie.go` and `tv/series.go` | Appendix C row 1 | 11.1 |
| 14 | `resolveField[T]()` in `movies/service.go` and `tv/service.go` | Appendix C row 2 | 11.1 |
| 15 | `rowToMovieFile` / `rowToEpisodeFile` file record mappers | Appendix C row 3 | 11.2 |
| 16 | File removal → status transition logic | Appendix C row 4 | 11.3 |
| 17 | Service struct layout + constructor | Appendix C row 5 | 11.4 |
| 18 | ~40 duplicated slot queries in `slots.sql` | Appendix C row 6 | 11.12 |
| 19 | List/search query patterns in `movies.sql` / `series.sql` | Appendix C row 7 | 11.16 (via scaffolding templates) |

### From §19 Contributor Tooling

| # | Spec Requirement | Assigned Task |
|---|---|---|
| 20 | `make new-module <id>` scaffolding generator (§19.1) | 11.16 |
| 21 | `slipstream validate-module <id>` CLI validation (§19.2) | 11.15 |
| 22 | `moduletest` package: lifecycle, parsing, naming, schema test harness (§19.3) | 11.14 |

**Total: 22 items, all assigned.**

---

## Phase 11 Architecture Decisions

1. **Shared utilities go in `internal/module/util.go`.** Functions like `GenerateSortTitle` and `ResolveField[T]` are framework-level utilities consumed by modules. They belong in the `internal/module/` package, not in a separate `internal/shared/` package. This keeps imports clean — modules already import `internal/module/`.

2. **Base service is an embeddable struct, not an interface.** `internal/module/base_service.go` defines a `BaseService` struct with common fields (`db`, `queries`, `hub`, `logger`, `statusChangeLogger`, `qualityProfiles`). Movie and TV services embed it. The `NewBaseService()` constructor handles common initialization (sub-logger creation, `sqlc.New(db)` call). This reduces duplication without forcing a rigid inheritance hierarchy.

3. **File record mapper uses generics.** A `MapFileRecord[Row, File any](row Row, mappers ...FieldMapper[Row, File]) File` helper avoids repeating `if row.X.Valid { f.X = row.X.String }` chains. Alternatively, simpler nullable-field helpers (`NullStr`, `NullTime`, `NullInt64`) let each module's mapper stay readable without complex generics. **Decision: nullable-field helpers.** The row-to-domain mapping is module-specific enough (different field names, different nullable columns) that a fully generic mapper adds complexity without clarity. Simple helpers for the repetitive `if .Valid` pattern are sufficient.

4. **Status transition helper is a framework function, not a method.** `TransitionToMissingAfterFileRemoval(ctx, TransitionParams)` in `internal/module/status_transition.go` takes a params struct containing the entity lookup function, status update function, broadcast function, and logger. This avoids coupling the framework to specific service types while eliminating the duplicated transition logic.

5. **`decisioning.SearchableItem` struct is fully replaced.** Phase 5 introduced a bridge (`collectFromModules` converting `module.SearchableItem` to `decisioning.SearchableItem`). Phase 11 removes the bridge — the entire autosearch pipeline operates on `module.SearchableItem` directly. The `decisioning` package's `SearchableItem` struct is deleted. All consumers (`SelectBestRelease`, `scorer`, `search router`) are updated to accept the interface.

6. **Scanner delegates to FileParser, not the other way around.** After Phase 6, `FileParser` wraps the scanner. Phase 11 inverts this: the scanner's `ParseFilename` calls the module's `FileParser.ParseFilename` via the registry. The scanner retains directory-walking and file-discovery logic but delegates all filename interpretation to modules. The `internal/module/parseutil/` utilities (extracted in Phase 6) remain the shared parsing building blocks.

7. **Generic monitoring cascade adds a `CascadeMonitored` method to the module interface.** Rather than the framework walking node schemas with generic queries (which would need module-specific SQL), each module implements `CascadeMonitored(ctx, entityType, entityID, monitored bool) error`. The framework calls this method. Movie's implementation is a no-op (flat). TV's implementation delegates to the existing cascade logic in `tv/seasons.go`. This is a thin abstraction that enables the framework to cascade for any module without knowing its internals. The generic node-schema-walk version is deferred until a 3rd module makes the existing approach insufficient.

8. **`validate-module` is a subcommand, not a separate binary.** The existing `main.go` already uses `os.Args` dispatch for `--complete-update`. Phase 11 adds `validate-module` as another early dispatch: `if len(os.Args) >= 2 && os.Args[1] == "validate-module"`. This avoids adding a CLI framework for a single command. The validation logic lives in `internal/module/validate/` — the main.go handler is a thin entry point.

9. **`make new-module` runs a Go script.** `scripts/new-module/main.go` is a standalone Go program that generates backend and frontend stubs from embedded templates. Templates are embedded via `//go:embed` and use `text/template`. The Makefile target is `new-module`: `go run ./scripts/new-module $(MODULE_ID)`.

10. **Slot query template generation uses `go generate`.** A `slots_generate.go` file in `internal/database/queries/` uses `//go:generate` to run a Go script that produces slot queries from a single template. The template has `{{.EntityType}}`, `{{.EntityTable}}`, `{{.FileTable}}`, `{{.JoinClause}}` placeholders. Output overwrites `slots.sql`. This keeps sqlc type safety while maintaining a single source of truth. The generator is run manually when slot query patterns change — not on every build.

11. **List/search query patterns are addressed through scaffolding, not generation.** Movie and TV queries differ structurally (TV has season/episode joins, cascading monitoring). Generating series.sql from a movie.sql template is not feasible. Instead, the scaffolding tool (Task 11.16) includes SQL query templates for common CRUD patterns (`Get`, `List`, `Create`, `Update`, `Delete`, `Search`, `ListPaginated`, file operations). New modules get these templates pre-filled. Existing movie/TV queries remain as-is.

12. **`moduletest` operates at the Go test level, not as a CLI.** The `internal/module/moduletest/` package provides test helper functions called from standard `_test.go` files. Contributors write `func TestMyModule(t *testing.T) { moduletest.RunLifecycleTest(t, myModule) }`. This integrates with `go test` and CI naturally. No custom test runner needed.

---

### Task 11.1: Extract Shared Utility Functions

**Depends on:** Phase 0 complete (`internal/module/` package exists)

**Create** `internal/module/util.go`

Extract the two utility functions duplicated between movie and TV services:

```go
package module

// GenerateSortTitle strips leading articles for sort ordering.
// Used by module services when creating/updating entities.
func GenerateSortTitle(title string) string {
    prefixes := []string{"The ", "A ", "An "}
    for _, prefix := range prefixes {
        if len(title) > len(prefix) && title[:len(prefix)] == prefix {
            return title[len(prefix):]
        }
    }
    return title
}

// ResolveField returns the input value if non-nil, otherwise the current value.
// Used in Update operations to apply optional fields.
func ResolveField[T any](current T, input *T) T {
    if input != nil {
        return *input
    }
    return current
}
```

**Modify** `internal/library/movies/movie.go`:
- Delete the local `generateSortTitle` function (currently at ~line 131).
- Update all call sites to use `module.GenerateSortTitle`.

**Modify** `internal/library/tv/series.go`:
- Delete the local `generateSortTitle` function (currently at ~line 219).
- Update all call sites to use `module.GenerateSortTitle`.

**Modify** `internal/library/movies/service.go`:
- Delete the local `resolveField[T]` function (currently at ~line 780).
- Update all call sites to use `module.ResolveField`.

**Modify** `internal/library/tv/service.go`:
- Delete the local `resolveField[T]` function (currently at ~line 791).
- Update all call sites to use `module.ResolveField`.

**Implementation notes:**
- The function is renamed from `generateSortTitle` (unexported) to `GenerateSortTitle` (exported) since it moves to a shared package.
- Search for ALL call sites of both functions before deleting. In movies: `generateSortTitle` is called in `Create` and `Update`. In TV: called in `CreateSeries` and `UpdateSeries`. There may be additional call sites added by earlier phases (e.g., in module stub packages `internal/modules/movie/` or `internal/modules/tv/`).
- `ResolveField` is a generic function — verify the Go compiler handles cross-package generic instantiation correctly (it does, but confirm with `go build`).

**Verify:**
- `go build ./internal/module/...`
- `go build ./internal/library/movies/...`
- `go build ./internal/library/tv/...`
- `go build ./internal/modules/...`
- `make test` — all existing tests pass

---

### Task 11.2: Extract Nullable-Field Helpers and Standardize File Record Mapping

**Depends on:** Phase 0 complete

**Create** `internal/module/nullable.go`

Extract helpers for the repetitive `if row.X.Valid` pattern in file record mappers:

```go
package module

import (
    "database/sql"
    "time"
)

// NullStr returns the string value if valid, empty string otherwise.
func NullStr(ns sql.NullString) string {
    if ns.Valid {
        return ns.String
    }
    return ""
}

// NullTime returns the time value if valid, zero time otherwise.
func NullTime(nt sql.NullTime) time.Time {
    if nt.Valid {
        return nt.Time
    }
    return time.Time{}
}

// NullInt64Ptr returns a pointer to the int64 value if valid, nil otherwise.
func NullInt64Ptr(ni sql.NullInt64) *int64 {
    if ni.Valid {
        v := ni.Int64
        return &v
    }
    return nil
}
```

**Modify** `internal/library/movies/service.go`:
- Refactor `rowToMovieFile` (currently at ~line 825) to use the nullable helpers:

```go
func (s *Service) rowToMovieFile(row *sqlc.MovieFile) MovieFile {
    return MovieFile{
        ID:            row.ID,
        MovieID:       row.MovieID,
        Path:          row.Path,
        Size:          row.Size,
        Quality:       module.NullStr(row.Quality),
        VideoCodec:    module.NullStr(row.VideoCodec),
        AudioCodec:    module.NullStr(row.AudioCodec),
        AudioChannels: module.NullStr(row.AudioChannels),
        DynamicRange:  module.NullStr(row.DynamicRange),
        Resolution:    module.NullStr(row.Resolution),
        CreatedAt:     module.NullTime(row.CreatedAt),
        SlotID:        module.NullInt64Ptr(row.SlotID),
    }
}
```

**Modify** `internal/library/tv/service.go`:
- If `rowToEpisodeFile` exists by Phase 11 (Phase 6 may have added it), refactor it the same way.
- If it doesn't exist yet, no change needed for TV.

**Implementation notes:**
- The nullable helpers reduce each mapper from ~30 lines to ~15 lines and eliminate the repeated `if .Valid` pattern.
- These helpers are useful beyond file records — any sqlc row-to-domain conversion can use them. Future modules will benefit.
- Do NOT create a generic `MapFileRecord` function — the field mapping is module-specific. The helpers address the repetitive part without over-abstracting.

**Verify:**
- `go build ./internal/module/...`
- `go build ./internal/library/movies/...`
- `make test`

---

### Task 11.3: Extract Framework Status Transition Helper

**Depends on:** Phase 0 complete

**Create** `internal/module/status_transition.go`

Extract the "file removed → transition to missing" pattern:

```go
package module

import (
    "context"

    "github.com/rs/zerolog"
)

// FileRemovalTransitionParams provides the callbacks needed for the framework
// to transition an entity to "missing" after its last file is removed.
type FileRemovalTransitionParams struct {
    ModuleType        Type
    EntityType        EntityType
    EntityID          int64
    Logger            *zerolog.Logger

    // GetCurrentStatus returns the entity's current status. Returns "" if entity not found.
    GetCurrentStatus  func(ctx context.Context, entityID int64) (string, error)
    // UpdateStatus sets the entity's status to "missing" and unmonitors it.
    UpdateStatus      func(ctx context.Context, entityID int64) error
    // LogStatusChange records the status transition in history (optional, may be nil).
    LogStatusChange   func(ctx context.Context, moduleType string, entityID int64, oldStatus, newStatus, reason string) error
    // Broadcast sends a WebSocket update (optional, may be nil).
    Broadcast         func(event string, data map[string]any)
}

// TransitionToMissingAfterFileRemoval handles the common pattern of transitioning
// an entity to "missing" status when its last file is removed.
// Call this after confirming the entity has zero remaining files.
func TransitionToMissingAfterFileRemoval(ctx context.Context, p FileRemovalTransitionParams) {
    oldStatus, _ := p.GetCurrentStatus(ctx, p.EntityID)

    if err := p.UpdateStatus(ctx, p.EntityID); err != nil {
        p.Logger.Error().Err(err).
            Str("moduleType", string(p.ModuleType)).
            Int64("entityId", p.EntityID).
            Msg("Failed to transition entity to missing after file removal")
        return
    }

    if p.LogStatusChange != nil && oldStatus != "" && oldStatus != "missing" {
        _ = p.LogStatusChange(ctx, string(p.ModuleType), p.EntityID, oldStatus, "missing", "File removed")
    }

    if p.Broadcast != nil {
        p.Broadcast(string(p.EntityType)+":updated", map[string]any{
            string(p.EntityType) + "Id": p.EntityID,
        })
    }
}
```

**Modify** `internal/library/movies/service.go`:
- Replace `transitionMovieToMissingAfterFileRemoval` (currently at ~line 802) with a call to `module.TransitionToMissingAfterFileRemoval`, passing the appropriate callbacks.

**Modify** `internal/library/tv/service.go`:
- If TV has equivalent file-removal transition logic by Phase 11, replace it the same way.

**Implementation notes:**
- The callback-based approach avoids the framework depending on specific service types or sqlc query signatures.
- The `Broadcast` event format follows the convention established in Phase 7 (`entityType + ":updated"`). Verify the actual WebSocket event names used by Phase 7 before implementation — the main agent should check one of the module's broadcast calls.
- The `UpdateStatus` callback should both set status to "missing" AND set monitored to false (the current movie implementation does both).

**Verify:**
- `go build ./internal/module/...`
- `go build ./internal/library/movies/...`
- `make test`

---

### Task 11.4: Base Service Struct Extraction

**Depends on:** Task 11.1 complete (shared utilities used by services)

**Create** `internal/module/base_service.go`

```go
package module

import (
    "database/sql"

    "github.com/rs/zerolog"

    "github.com/slipstream/slipstream/internal/database/sqlc"
    "github.com/slipstream/slipstream/internal/domain/contracts"
    "github.com/slipstream/slipstream/internal/library/quality"
    "github.com/slipstream/slipstream/internal/websocket"
)

// BaseService provides the common fields and initialization shared by all
// module service implementations. Module services embed this struct.
type BaseService struct {
    DB                 *sql.DB
    Queries            *sqlc.Queries
    Hub                *websocket.Hub
    Logger             *zerolog.Logger
    StatusChangeLogger contracts.StatusChangeLogger
    QualityProfiles    *quality.Service
}

// NewBaseService creates a BaseService with standard initialization.
// componentName is used for the logger's "component" field (e.g., "movies", "tv").
func NewBaseService(db *sql.DB, hub *websocket.Hub, logger *zerolog.Logger, qualityService *quality.Service, statusChangeLogger contracts.StatusChangeLogger, componentName string) BaseService {
    subLogger := logger.With().Str("component", componentName).Logger()
    return BaseService{
        DB:                 db,
        Queries:            sqlc.New(db),
        Hub:                hub,
        Logger:             &subLogger,
        QualityProfiles:    qualityService,
        StatusChangeLogger: statusChangeLogger,
    }
}

// Broadcast sends a WebSocket event if the hub is available.
func (bs *BaseService) Broadcast(event string, data map[string]any) {
    if bs.Hub != nil {
        bs.Hub.Broadcast(event, data)
    }
}
```

**Modify** `internal/library/movies/service.go`:
- Replace the `Service` struct's common fields with an embedded `module.BaseService`.
- Replace the `NewService` constructor to call `module.NewBaseService`.
- Update all field accesses: `s.db` → `s.DB`, `s.queries` → `s.Queries`, `s.hub` → `s.Hub`, `s.logger` → `s.Logger`, etc.
- Keep movie-specific fields (e.g., `fileDeleteHandler`, `notifier`) on the movie `Service` struct directly.
- Keep movie-specific setter methods (e.g., `SetFileDeleteHandler`, `SetNotificationDispatcher`).

**Modify** `internal/library/tv/service.go`:
- Same refactoring as movies.

**Implementation notes:**
- The `BaseService` fields are exported (uppercase) so the embedding service can access them directly. This is conventional for Go embedded structs.
- The `Broadcast` convenience method eliminates the repeated nil-check pattern (`if s.hub != nil { s.hub.Broadcast(...) }`).
- Verify that Wire DI still works after changing the constructor signatures. Run `make wire` if needed.
- This task touches the most code of any cleanup task. The subagent should do a careful find-and-replace, then verify every test passes.
- **Critical:** search for ALL references to the old field names (`s.db`, `s.queries`, `s.hub`, `s.logger`, `s.qualityProfiles`, `s.statusChangeLogger`) across both service files. Missing a reference causes a compile error (good — the compiler catches it).

**Verify:**
- `go build ./internal/module/...`
- `go build ./internal/library/movies/...`
- `go build ./internal/library/tv/...`
- `make wire` (if constructor signatures changed)
- `make test`

---

### Task 11.5: Eliminate `decisioning.SearchableItem` Struct

**Depends on:** Task 11.2 complete (for stable module package), Phase 5 complete (bridge exists)

**Goal:** Remove the `decisioning.SearchableItem` struct and the `collectFromModules` bridge converter. The entire autosearch pipeline operates on `module.SearchableItem` interface directly.

**Modify** `internal/autosearch/service.go` (and related files in `internal/autosearch/`):
- Replace all references to `decisioning.SearchableItem` with `module.SearchableItem`.
- Remove the `collectFromModules` converter function that was introduced in Phase 5 as a bridge.
- Update function signatures: `searchItem(ctx, item module.SearchableItem)`, etc.

**Modify** `internal/indexer/scoring/scorer.go`:
- Update `ScoreRelease` and related functions to accept `module.SearchableItem` instead of `decisioning.SearchableItem`.

**Modify** `internal/indexer/search/router.go`:
- Update search functions to accept `module.SearchableItem`.

**Modify** `internal/decisioning/types.go`:
- Delete the `SearchableItem` struct.
- Keep other types in `decisioning` that are still used (e.g., `Release`, `GrabResult`). Do NOT delete the entire package.

**Implementation notes:**
- The `module.SearchableItem` is an interface; `decisioning.SearchableItem` was a struct. All call sites that construct a `decisioning.SearchableItem{}` literal must change to constructing the module's concrete type (e.g., `movie.SearchableItem{}` or `tv.SearchableItem{}`). These concrete types were created in Phase 5's module implementations.
- The Phase 5 bridge in `autosearch/scheduled.go` has a `collectFromModules` function — delete it entirely. The module `WantedCollector` implementations already return `module.SearchableItem` values; they no longer need conversion.
- Search for ALL imports of `decisioning.SearchableItem` across the codebase. Every import must be removed or redirected.
- If `decisioning.SearchableItem` is referenced in test files, update those too.

**Verify:**
- `grep -r "decisioning.SearchableItem" internal/` returns zero matches (excluding the deleted file itself)
- `go build ./...`
- `make test`

---

### Task 11.6: Refactor `SelectBestRelease` for Module-Provided Filtering

**Depends on:** Task 11.5 complete (`module.SearchableItem` used throughout)

**Modify** `internal/indexer/scoring/scorer.go` (or wherever `SelectBestRelease` lives — verify actual location):
- Remove the hard-coded `shouldSkipTVRelease` function.
- Replace it with a call to the module's `SearchStrategy.FilterRelease(release, item)`.
- The `SelectBestRelease` function receives the `SearchStrategy` as a parameter (or retrieves it from the registry via `item.GetModuleType()`).

**Before (Phase 5 state):**
```go
func SelectBestRelease(releases []Release, item SearchableItem, ...) *Release {
    for _, r := range releases {
        // hard-coded TV filter
        if item.GetMediaType() == "episode" {
            if skip, _ := shouldSkipTVRelease(r, item); skip {
                continue
            }
        }
        // ... scoring
    }
}
```

**After:**
```go
func SelectBestRelease(releases []Release, item module.SearchableItem, strategy module.SearchStrategy, ...) *Release {
    for _, r := range releases {
        if reject, reason := strategy.FilterRelease(r, item); reject {
            // log rejection reason
            continue
        }
        // ... scoring
    }
}
```

**Modify** all call sites of `SelectBestRelease` to pass the `SearchStrategy`:
- The strategy is obtained from the module registry: `registry.Get(item.GetModuleType()).SearchStrategy()` (or however the module exposes it — check the Phase 5 implementation).

**Modify** `internal/autosearch/` to pass the strategy through to scoring.

**Delete** the `shouldSkipTVRelease` function.

**Implementation notes:**
- Movie's `SearchStrategy.FilterRelease` returns `(false, "")` for everything (no filtering beyond quality, which is handled separately).
- TV's `SearchStrategy.FilterRelease` contains the logic currently in `shouldSkipTVRelease` — verify this was implemented in Phase 5. If not, move it now.
- This is a medium-risk change. Ensure test coverage exists for the TV filtering logic (season pack matching, daily/anime format filtering, etc.).

**Verify:**
- `grep -r "shouldSkipTVRelease" internal/` returns zero matches
- `go build ./...`
- `make test`

---

### Task 11.7: Full Scanner Refactoring — FileParser as Primary Parser

**Depends on:** Task 11.5 complete (module interfaces stable)

**Goal:** Invert the scanner/FileParser relationship. Currently (after Phase 6), `FileParser` wraps the scanner. After this task, the scanner calls `FileParser` via the module registry.

**Modify** `internal/library/scanner/parser.go` (or equivalent):
- Remove media-type-specific parsing logic (S01E02 patterns, Title (Year) patterns) from the scanner.
- Replace with a call to the module registry's FileParser:

```go
func (p *Parser) ParseFilename(filename string, registry *module.Registry) *module.ParseResult {
    var bestResult *module.ParseResult
    var bestConfidence float64

    for _, mod := range registry.All() {
        confidence, result := mod.FileParser().TryMatch(filename)
        if confidence > bestConfidence {
            bestConfidence = confidence
            bestResult = result
        }
    }
    return bestResult
}
```

- Retain shared parsing infrastructure in `internal/module/parseutil/` (quality detection, codec detection, etc.) — these are called BY module FileParser implementations, not by the scanner directly.
- Retain directory-walking and file-discovery logic in the scanner package.

**Modify** `internal/library/scanner/` call sites:
- Update all callers of the scanner's parse functions to pass the registry.
- The scanner is used in: library scans (`librarymanager`), orphan detection (`ScanForPendingImports` in import pipeline), and potentially manual import.

**Implementation notes:**
- The `TryMatch` method on each module's `FileParser` was defined in Phase 6 (spec §8.4). It returns `(confidence float64, match *ParseResult)`. The scanner iterates all enabled modules and picks the highest confidence.
- The `internal/module/parseutil/` package (created in Phase 6) continues to provide `ExtractYear`, `NormalizeTitle`, `DetectQualityAttributes`, `ExtractEdition`. Module FileParser implementations compose these utilities.
- After this refactoring, the scanner package should have ZERO imports of `internal/modules/movie/` or `internal/modules/tv/`. All module-specific logic flows through the `module.FileParser` interface.
- **Risk:** This is the largest single refactoring task. The subagent should first catalog all scanner parse functions and their callers, then refactor incrementally, verifying compilation after each change.

**Verify:**
- `grep -r "internal/modules/" internal/library/scanner/` returns zero matches (no direct module imports)
- `go build ./...`
- `make test` — especially scanner tests and import pipeline tests

---

### Task 11.8: Naming Pipeline Cleanup

**Depends on:** Phase 6 complete (naming infrastructure exists)

Three sub-tasks, all related to the naming/renamer pipeline:

#### 11.8a: Extract Shared Naming Template Helpers

**Move** `qualityVariables()`, `mediaInfoVariables()`, `metadataVariables()` (currently in `internal/modules/movie/path_naming.go` or a shared file) to `internal/module/naming_helpers.go`:

```go
package module

// QualityVariables returns the template variables related to quality attributes.
// Shared by all video-type modules.
func QualityVariables() []TemplateVariable { ... }

// MediaInfoVariables returns the template variables from media info probing.
func MediaInfoVariables() []TemplateVariable { ... }

// MetadataVariables returns the common metadata template variables.
func MetadataVariables() []TemplateVariable { ... }
```

Update `internal/modules/movie/path_naming.go` and `internal/modules/tv/path_naming.go` to call the shared helpers instead of defining them locally.

#### 11.8b: Remove `populateLegacyFields` Bridge in Renamer

**Modify** `internal/import/renamer/` (location of the renamer):
- Delete the `populateLegacyFields` function that bridges new per-module settings to old typed fields.
- The renamer should use the `ResolveContext(contextName, ctx, ext)` method exclusively (introduced in Phase 6).
- Remove any remaining references to the old type-specific methods (`ResolveMovieFolderName`, `ResolveSeriesFolderName`, etc.) if they are no longer called by anything.

#### 11.8c: Drop Unused `import_settings` Naming Columns

**Create** a new migration file in `internal/database/migrations/`:

```sql
-- +goose Up
-- Remove naming columns that became unused when module_naming_settings was introduced in Phase 6.
-- SQLite doesn't support DROP COLUMN in older versions, so we recreate the table.

CREATE TABLE import_settings_new (
    id INTEGER PRIMARY KEY,
    rename_enabled INTEGER NOT NULL DEFAULT 0,
    -- Keep only non-naming columns that are still used.
    -- List the remaining columns here based on what import_settings
    -- actually contains beyond naming formats.
);

INSERT INTO import_settings_new (id, rename_enabled, ...)
SELECT id, rename_enabled, ... FROM import_settings;

DROP TABLE import_settings;
ALTER TABLE import_settings_new RENAME TO import_settings;

-- +goose Down
-- Recreate with original columns (data loss on naming columns is acceptable for down migration)
```

**Implementation notes:**
- Before writing the migration, the subagent MUST read the current `import_settings` table schema to know exactly which columns exist and which are unused.
- The columns to drop are the naming format columns that were migrated to `module_naming_settings` in Phase 6 (movie folder/file format, series folder format, season folder format, episode file format variants, colon replacement, multi-episode style).
- Columns that are NOT naming-related (e.g., `rename_enabled`, any import-behavior flags) must be preserved.
- Run `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate` after updating the migration and any affected queries.

**Verify:**
- `go build ./internal/import/renamer/...`
- `make test`
- `grep -r "populateLegacyFields" internal/` returns zero matches

---

### Task 11.9: Calendar, Availability & Legacy Fallback Cleanup

**Depends on:** Phase 4 and Phase 10 complete (module CalendarProvider validated, frontend uses module framework)

**Goal:** Remove the legacy code paths that were kept as fallbacks during the incremental migration.

#### 11.9a: Remove Legacy Calendar Helper Functions

**Modify** `internal/calendar/service.go`:
- Delete `getMovieEvents`, `getEpisodeEvents`, `resolveMovieStatus`, `movieRowToEvents`, `createEpisodeEvent`, `createSeasonReleaseEvent` and any other legacy helper functions.
- The service should now ONLY iterate over module `CalendarProvider` implementations via the registry. The `if registry == nil` fallback path is removed.

#### 11.9b: Remove Legacy Dispatcher Fallback Paths

**Modify** `internal/calendar/service.go`, `internal/availability/service.go` (if exists), `internal/missing/service.go` (if exists):
- Remove all `if s.registry == nil { /* legacy path */ }` branches.
- The registry is guaranteed to be set after Phase 10 — fallback code is dead.
- If removing the fallback makes the `registry` field always required, make it a constructor parameter (not a setter).

#### 11.9c: Consolidate `streamingServicesWithEarlyRelease` Map

**Move** the `streamingServicesWithEarlyRelease` map to a single canonical location:
- If it's only used by the TV module's calendar provider, keep it in `internal/modules/tv/calendar.go` and delete the copy from `internal/calendar/service.go`.
- If it's used by both framework and module code, put it in `internal/module/` as an exported variable.

**Implementation notes:**
- The subagent should first verify which functions are actually still present — earlier phases may have already removed some during their own cleanup. Read `internal/calendar/service.go` before attempting deletions.
- After removing the legacy helpers, check for any orphaned imports or unused types.
- The `availability.Service` and `missing.Service` may have been partially cleaned up by Phase 4. The subagent should check if these services still exist and what fallback paths remain.

**Verify:**
- `grep -r "getMovieEvents\|getEpisodeEvents\|resolveMovieStatus\|movieRowToEvents" internal/` returns zero matches
- `grep -r "registry == nil" internal/calendar/ internal/availability/ internal/missing/` returns zero matches (no fallback paths)
- `go build ./...`
- `make test`

---

### Task 11.10: Miscellaneous Deduplication

**Depends on:** Phase 3 and Phase 4 complete

Two independent sub-tasks:

#### 11.10a: Align `UpdateUnreleasedMoviesToMissing` SQL

**Modify** `internal/database/queries/movies.sql`:
- Find the `UpdateUnreleasedMoviesToMissing` query.
- Change the release date logic from the priority chain (`digital → physical → theatrical+90d`) to "earliest of digital and physical" per spec §7.1.
- The query should transition movies to "missing" when `MIN(COALESCE(digital_release_date, '9999'), COALESCE(physical_release_date, '9999')) <= date('now')`.
- Theatrical release date is irrelevant for download availability — do not include it.

Run `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate` after.

#### 11.10b: Deduplicate `convertToSeasonMetadata`

**Find** all occurrences of `convertToSeasonMetadata`:
- `internal/library/librarymanager/metadata_refresh.go` (~line 387)
- `internal/library/librarymanager/add.go` (verify if it exists here)
- `internal/modules/tv/metadata.go` (created in Phase 3)

**Consolidate** into a single function:
- Put the canonical version in the TV module's metadata package: `internal/modules/tv/metadata.go` (or a helper file in the same package).
- If `librarymanager` still calls it, have it delegate to the TV module's version via the module registry (or move the function to a shared location both can import without circular deps).
- If `librarymanager` no longer calls it directly (because `RefreshMetadata` was moved to the module in Phase 3), simply delete the `librarymanager` copy.

**Implementation notes:**
- Check import paths carefully. `librarymanager` importing `internal/modules/tv` may create circular dependencies. If so, extract the conversion function to a shared types package (e.g., `internal/library/tv/types.go` or `internal/module/tv_helpers.go`).
- The function converts `[]metadata.SeasonResult` to `[]tv.SeasonMetadata`. If both types are in different packages, the function naturally belongs near the target type (`tv.SeasonMetadata`).

**Verify:**
- Only ONE copy of `convertToSeasonMetadata` exists in the codebase.
- `go build ./...`
- `make test`

---

### Task 11.11: Generic Framework Monitoring Cascade

**Depends on:** Phase 4 complete (MonitoringPresets implemented)

**Goal:** Add a framework-level monitoring cascade that works for any module, replacing the need for module-specific cascade logic in future modules.

**Add** method to the module interface in `internal/module/interfaces.go`:

```go
// CascadeMonitored propagates a monitored value to all descendants of the
// given entity. For flat modules (e.g., Movie), this is a no-op.
// For hierarchical modules (e.g., TV), this propagates to children
// (e.g., setting series.monitored=false cascades to seasons and episodes).
type MonitoringCascader interface {
    CascadeMonitored(ctx context.Context, entityType EntityType, entityID int64, monitored bool) error
}
```

**Note:** This is defined as a separate interface (not added to an existing required interface) because not all modules need cascade logic. Movie implements it as a no-op. TV wraps the existing cascade code in `tv/seasons.go`.

**Implement** on movie module in `internal/modules/movie/monitoring.go`:
```go
func (m *Module) CascadeMonitored(ctx context.Context, entityType module.EntityType, entityID int64, monitored bool) error {
    return nil // flat module, no children to cascade to
}
```

**Implement** on TV module in `internal/modules/tv/monitoring.go`:
- Delegate to the existing cascade logic in `internal/library/tv/seasons.go` (BulkMonitor or the direct cascade code at ~lines 50–56).

**Add** framework dispatcher in `internal/module/`:
```go
// CascadeMonitoredForModule propagates monitoring state through a module's
// node hierarchy. Falls back to no-op if the module doesn't implement MonitoringCascader.
func CascadeMonitoredForModule(ctx context.Context, registry *Registry, moduleType Type, entityType EntityType, entityID int64, monitored bool) error {
    mod := registry.Get(moduleType)
    if cascader, ok := mod.(MonitoringCascader); ok {
        return cascader.CascadeMonitored(ctx, entityType, entityID, monitored)
    }
    return nil
}
```

**Modify** any framework code that currently calls TV-specific cascade logic directly — route it through `CascadeMonitoredForModule` instead.

**Implementation notes:**
- This is the thin-abstraction approach (AD #7). The full generic node-schema-walk cascade is deferred until a 3rd module makes the current approach insufficient.
- The existing TV cascade code is NOT deleted — it's just wrapped by the module's `CascadeMonitored` implementation.
- The `MonitoringCascader` interface is optional — check with type assertion. This follows the pattern of other optional interfaces (`PortalProvisioner`, `SlotSupport`).

**Verify:**
- `go build ./...`
- `make test` — monitoring cascade tests still pass

---

### Task 11.12: Slot Query Template Generation

**Depends on:** Phase 8 complete (slot system refactored)

**Goal:** Create a single-source-of-truth template for slot assignment queries, reducing the ~40 duplicated movie/episode query pairs.

**Create** `internal/database/queries/slots_template.go`:

```go
//go:generate go run ./gen_slots.go

package queries
```

**Create** `internal/database/queries/gen_slots.go`:

```go
//go:build ignore

package main

import (
    "fmt"
    "os"
    "strings"
    "text/template"
)

type SlotEntity struct {
    EntityType   string // "Movie" or "Episode"
    EntityTable  string // "movies" or "episodes"
    FileTable    string // "movie_files" or "episode_files"
    AssignTable  string // "movie_slot_assignments" or "episode_slot_assignments"
    EntityIDCol  string // "movie_id" or "episode_id"
    TitleExpr    string // "m.title" or "s.title || ' - S' || printf('%02d', se.season_number) || 'E' || printf('%02d', e.episode_number)"
    TitleJoins   string // "" or "JOIN seasons se ON ... JOIN series s ON ..."
}

var entities = []SlotEntity{
    {
        EntityType:  "Movie",
        EntityTable: "movies",
        FileTable:   "movie_files",
        AssignTable: "movie_slot_assignments",
        EntityIDCol: "movie_id",
        TitleExpr:   "m.title",
        TitleJoins:  "",
    },
    {
        EntityType:  "Episode",
        EntityTable: "episodes",
        FileTable:   "episode_files",
        AssignTable: "episode_slot_assignments",
        EntityIDCol: "episode_id",
        TitleExpr:   `s.title || ' - S' || printf('%02d', se.season_number) || 'E' || printf('%02d', e.episode_number)`,
        TitleJoins:  "JOIN seasons se ON e.season_id = se.id JOIN series s ON se.series_id = s.id",
    },
}

// Template contains the slot query patterns with {{.EntityType}} placeholders.
// The subagent should extract this from the current slots.sql by generalizing
// one set of queries (movie or episode) into a template.

func main() {
    tmplContent, _ := os.ReadFile("slots_template.sql.tmpl")
    tmpl := template.Must(template.New("slots").Parse(string(tmplContent)))

    out, _ := os.Create("slots.sql")
    defer out.Close()

    fmt.Fprintln(out, "-- Code generated by gen_slots.go. DO NOT EDIT.")
    fmt.Fprintln(out, "-- Edit slots_template.sql.tmpl instead and run: go generate ./internal/database/queries/")
    fmt.Fprintln(out)

    // Write shared (non-templated) slot queries first
    shared, _ := os.ReadFile("slots_shared.sql")
    out.Write(shared)
    fmt.Fprintln(out)

    // Generate per-entity queries
    for _, e := range entities {
        tmpl.Execute(out, e)
        fmt.Fprintln(out)
    }
}
```

**Create** `internal/database/queries/slots_template.sql.tmpl`:
- Extract the repeating query patterns from the current `slots.sql`.
- Replace `movie_slot_assignments` → `{{.AssignTable}}`, `Movie` → `{{.EntityType}}`, etc.
- One template block per query pattern.

**Create** `internal/database/queries/slots_shared.sql`:
- Extract the non-duplicated slot queries (version_slots CRUD, multi_version_settings, etc.) into this file.
- These are appended to the generated output unchanged.

**Verify the generated output matches the current `slots.sql`:**
- Run `go generate ./internal/database/queries/`
- Diff the generated `slots.sql` against the pre-existing version. They should be functionally identical (whitespace differences are OK).
- Run `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate` — sqlc should produce identical Go code.

**Update** `Makefile`: add a comment noting that `slots.sql` is generated. Optionally add a target:
```makefile
generate-slots: ## Regenerate slot SQL queries from template
	@cd internal/database/queries && go generate .
```

**Implementation notes:**
- The subagent should first read the entire current `slots.sql` and categorize each query as "shared" vs "per-entity-type pair".
- The template approach keeps sqlc type safety — sqlc sees the fully expanded SQL and generates typed Go code as before.
- The generator is run manually (`go generate` or `make generate-slots`), NOT on every build. It only needs re-running when the slot query pattern itself changes.
- If the current `slots.sql` has queries that are NOT symmetric between movie and episode (e.g., episode has extra joins for series/season info), the template must handle this via `{{.TitleJoins}}` and `{{.TitleExpr}}` fields.

**Verify:**
- `diff <(go generate ./internal/database/queries/ && cat slots.sql) slots.sql.bak` shows no semantic differences
- `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate` succeeds
- `go build ./...`
- `make test`

---

### Task 11.13: Validation Checkpoint — All Cleanup Complete

**Depends on:** Tasks 11.1–11.12 all complete

This is not a code task — it's a validation gate.

**Run:**
```bash
make build && make test && make lint
cd web && bun run lint
```

**Check that no legacy code remains:**
```bash
# No decisioning.SearchableItem references
grep -r "decisioning\.SearchableItem" internal/

# No legacy calendar helpers
grep -r "getMovieEvents\|getEpisodeEvents\|movieRowToEvents" internal/

# No fallback paths
grep -r "registry == nil" internal/calendar/ internal/availability/ internal/missing/

# No renamer bridge
grep -r "populateLegacyFields" internal/

# No duplicated utility functions
grep -rn "func generateSortTitle" internal/   # should be 0 (only GenerateSortTitle in module/)
grep -rn "func resolveField" internal/        # should be 0 (only ResolveField in module/)

# shouldSkipTVRelease removed
grep -r "shouldSkipTVRelease" internal/
```

All grep commands should return zero matches. If any return matches, file a follow-up fix before proceeding to contributor tooling.

---

### Task 11.14: Module Test Harness (`moduletest` Package)

**Depends on:** Task 11.13 complete (cleanup validated — test harness should test the final architecture)

**Create** `internal/module/moduletest/` package

This package provides integration test helpers per spec §19.3. Contributors use these in standard `_test.go` files.

#### File: `internal/module/moduletest/schema_test_helpers.go`

```go
package moduletest

import (
    "testing"

    "github.com/slipstream/slipstream/internal/module"
)

// RunSchemaTest validates a module's node schema for consistency.
// Checks: exactly one root, exactly one leaf, ordered levels, unique names,
// format variants are non-empty strings, HasMonitored set on searchable levels.
func RunSchemaTest(t *testing.T, mod module.Module) {
    t.Helper()
    t.Run("NodeSchema", func(t *testing.T) {
        schema := mod.Descriptor().NodeSchema()

        // Validate using the schema's own Validate method
        if err := schema.Validate(); err != nil {
            t.Fatalf("NodeSchema.Validate() failed: %v", err)
        }

        // Additional checks beyond basic Validate
        root, _ := schema.Root()
        leaf, _ := schema.Leaf()

        if !root.HasMonitored {
            t.Error("Root level should have HasMonitored=true")
        }
        if !leaf.HasMonitored {
            t.Error("Leaf level should have HasMonitored=true")
        }

        // Entity types should match level names
        entityTypes := mod.Descriptor().EntityTypes()
        if len(entityTypes) != len(schema.Levels) {
            t.Errorf("EntityTypes count (%d) != schema levels count (%d)",
                len(entityTypes), len(schema.Levels))
        }
    })
}
```

#### File: `internal/module/moduletest/parsing_test_helpers.go`

```go
package moduletest

import (
    "testing"

    "github.com/slipstream/slipstream/internal/module"
)

// ParsingFixture defines an expected parse result for a filename.
type ParsingFixture struct {
    Filename       string
    ExpectedTitle  string
    ExpectedYear   int
    ExpectMatch    bool    // false = module should not match this filename
    MinConfidence  float64 // minimum confidence from TryMatch
}

// RunParsingTest validates a module's FileParser against fixture data.
func RunParsingTest(t *testing.T, parser module.FileParser, fixtures []ParsingFixture) {
    t.Helper()
    for _, fix := range fixtures {
        t.Run(fix.Filename, func(t *testing.T) {
            confidence, result := parser.TryMatch(fix.Filename)

            if !fix.ExpectMatch {
                if confidence > 0 {
                    t.Errorf("Expected no match, got confidence=%f", confidence)
                }
                return
            }

            if confidence < fix.MinConfidence {
                t.Errorf("Confidence %f < minimum %f", confidence, fix.MinConfidence)
            }
            if result == nil {
                t.Fatal("Expected match but got nil result")
            }
            if result.Title != fix.ExpectedTitle {
                t.Errorf("Title = %q, want %q", result.Title, fix.ExpectedTitle)
            }
            if fix.ExpectedYear > 0 && result.Year != fix.ExpectedYear {
                t.Errorf("Year = %d, want %d", result.Year, fix.ExpectedYear)
            }
        })
    }
}
```

#### File: `internal/module/moduletest/naming_test_helpers.go`

```go
package moduletest

import (
    "testing"

    "github.com/slipstream/slipstream/internal/module"
)

// NamingFixture defines expected output for a naming template resolution.
type NamingFixture struct {
    ContextName    string         // e.g., "movie-file", "episode-file.daily"
    TokenData      map[string]any // template variable values
    Extension      string         // e.g., ".mkv"
    ExpectedOutput string
}

// RunNamingTest validates a module's NamingProvider templates against fixtures.
func RunNamingTest(t *testing.T, provider module.NamingProvider, pathGen module.PathGenerator, fixtures []NamingFixture) {
    t.Helper()

    // Verify token contexts are declared
    contexts := provider.TokenContexts()
    if len(contexts) == 0 {
        t.Fatal("NamingProvider declares no token contexts")
    }

    // Verify default templates exist for all contexts
    defaults := provider.DefaultFileTemplates()
    for _, ctx := range contexts {
        if _, ok := defaults[ctx.Name]; !ok {
            t.Errorf("No default template for context %q", ctx.Name)
        }
    }

    // Run fixture tests
    for _, fix := range fixtures {
        t.Run(fix.ContextName+"/"+fix.ExpectedOutput, func(t *testing.T) {
            template, ok := defaults[fix.ContextName]
            if !ok {
                t.Skipf("No default template for context %q", fix.ContextName)
                return
            }
            result, err := pathGen.ResolveTemplate(template, fix.TokenData)
            if err != nil {
                t.Fatalf("ResolveTemplate failed: %v", err)
            }
            if fix.Extension != "" {
                result += fix.Extension
            }
            if result != fix.ExpectedOutput {
                t.Errorf("got %q, want %q", result, fix.ExpectedOutput)
            }
        })
    }
}
```

#### File: `internal/module/moduletest/lifecycle_test_helpers.go`

```go
package moduletest

import (
    "context"
    "testing"

    "github.com/slipstream/slipstream/internal/module"
    "github.com/slipstream/slipstream/internal/testutil"
)

// LifecycleConfig configures the lifecycle test.
type LifecycleConfig struct {
    // SampleSearchQuery is a search query that the mock metadata provider should match.
    SampleSearchQuery string
    // SampleExternalID is the external ID the mock metadata provider returns.
    SampleExternalID string
}

// RunLifecycleTest runs the full add → search → grab → import → status lifecycle.
// Uses mock infrastructure (test DB, mock metadata results).
// This test validates that all module interfaces work together correctly.
func RunLifecycleTest(t *testing.T, mod module.Module, cfg LifecycleConfig) {
    t.Helper()

    tdb := testutil.NewTestDB(t)
    defer tdb.Close()
    ctx := context.Background()

    // 1. Validate module schema
    RunSchemaTest(t, mod)

    // 2. Metadata search
    t.Run("MetadataSearch", func(t *testing.T) {
        results, err := mod.MetadataProvider().Search(ctx, cfg.SampleSearchQuery, module.SearchOptions{})
        if err != nil {
            t.Fatalf("Search failed: %v", err)
        }
        if len(results) == 0 {
            t.Fatal("Search returned no results")
        }
    })

    // 3. Quality definition
    t.Run("QualityDefinition", func(t *testing.T) {
        items := mod.QualityDefinition().QualityItems()
        if len(items) == 0 {
            t.Fatal("Module declares no quality items")
        }
        // Verify unique IDs
        seen := make(map[int]bool)
        for _, item := range items {
            if seen[item.ID] {
                t.Errorf("Duplicate quality item ID: %d", item.ID)
            }
            seen[item.ID] = true
        }
    })

    // 4. Wanted collection (empty library → no results, but should not error)
    t.Run("WantedCollector", func(t *testing.T) {
        missing, err := mod.WantedCollector().CollectMissing(ctx)
        if err != nil {
            t.Fatalf("CollectMissing failed: %v", err)
        }
        // Empty library should have no missing items
        if len(missing) != 0 {
            t.Logf("Note: empty library returned %d missing items", len(missing))
        }
    })

    // 5. Search strategy declares categories
    t.Run("SearchStrategy", func(t *testing.T) {
        cats := mod.SearchStrategy().Categories()
        if len(cats) == 0 {
            t.Fatal("Module declares no search categories")
        }
    })

    // 6. Notification events
    t.Run("NotificationEvents", func(t *testing.T) {
        events := mod.NotificationEvents().DeclareEvents()
        if len(events) == 0 {
            t.Fatal("Module declares no notification events")
        }
        for _, e := range events {
            if e.ID == "" {
                t.Error("Notification event has empty ID")
            }
        }
    })

    // 7. Calendar provider (should not error on empty library)
    t.Run("CalendarProvider", func(t *testing.T) {
        _, err := mod.CalendarProvider().GetItemsInDateRange(ctx,
            testutil.ParseTime("2020-01-01"), testutil.ParseTime("2030-01-01"))
        if err != nil {
            t.Fatalf("GetItemsInDateRange failed: %v", err)
        }
    })
}
```

**Implementation notes:**
- The `module.Module` type referenced here is the combined interface that holds all required interfaces (defined in Phase 0). Verify the exact type name — it may be `module.Module` or the module struct that implements all interfaces.
- `RunLifecycleTest` is a **smoke test**, not a full end-to-end integration test. It verifies that all interfaces are callable and return reasonable results. The full lifecycle (search → grab → import → status transition) requires mock indexer and download client infrastructure that is complex to set up. The smoke test covers interface completeness; detailed behavior is tested by each module's own test suite.
- `testutil.ParseTime` may not exist yet. If not, add a simple helper: `func ParseTime(s string) time.Time { t, _ := time.Parse("2006-01-02", s); return t }` to `internal/testutil/testutil.go`.
- Add a `moduletest/doc.go` with package documentation explaining how contributors use these helpers.

**Create** test files that exercise the helpers against the existing movie and TV modules:
- `internal/modules/movie/module_test.go` — calls `moduletest.RunSchemaTest`, `moduletest.RunParsingTest`, `moduletest.RunNamingTest`
- `internal/modules/tv/module_test.go` — same

**Verify:**
- `go build ./internal/module/moduletest/...`
- `go test ./internal/modules/movie/...`
- `go test ./internal/modules/tv/...`

---

### Task 11.15: Module Validation CLI

**Depends on:** Task 11.14 complete (moduletest provides validation logic to reuse)

**Goal:** Add a `validate-module` subcommand per spec §19.2.

**Create** `internal/module/validate/validate.go`:

```go
package validate

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/slipstream/slipstream/internal/module"
)

// Result holds the validation outcome.
type Result struct {
    ModuleID string
    Errors   []string
    Warnings []string
}

func (r *Result) AddError(format string, args ...any) {
    r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

func (r *Result) AddWarning(format string, args ...any) {
    r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

func (r *Result) OK() bool { return len(r.Errors) == 0 }

// ValidateModule checks a module implementation for correctness.
func ValidateModule(registry *module.Registry, moduleID string) *Result {
    r := &Result{ModuleID: moduleID}

    // 1. Module exists in registry
    modType := module.Type(moduleID)
    mod := registry.Get(modType)
    if mod == nil {
        r.AddError("Module %q not found in registry. Is it registered in RegisterAll()?", moduleID)
        return r
    }

    // 2. Node schema validation
    schema := mod.Descriptor().NodeSchema()
    if err := schema.Validate(); err != nil {
        r.AddError("Node schema invalid: %v", err)
    }

    // 3. Entity type uniqueness (no conflicts with other modules)
    for _, m := range registry.All() {
        if m.Descriptor().ID() == modType {
            continue
        }
        for _, et := range mod.Descriptor().EntityTypes() {
            for _, otherET := range m.Descriptor().EntityTypes() {
                if et == otherET {
                    r.AddError("Entity type %q conflicts with module %q", et, m.Descriptor().ID())
                }
            }
        }
    }

    // 4. Quality items have unique IDs and weights
    qd := mod.QualityDefinition()
    if qd != nil {
        items := qd.QualityItems()
        seenIDs := make(map[int]bool)
        for _, item := range items {
            if seenIDs[item.ID] {
                r.AddError("Duplicate quality item ID: %d", item.ID)
            }
            seenIDs[item.ID] = true
        }
    }

    // 5. Notification event IDs follow module:event convention
    ne := mod.NotificationEvents()
    if ne != nil {
        events := ne.DeclareEvents()
        prefix := moduleID + ":"
        for _, e := range events {
            if !strings.HasPrefix(e.ID, prefix) {
                r.AddError("Notification event ID %q doesn't follow %s convention", e.ID, prefix+"<event>")
            }
        }
    }

    // 6. Migration files exist
    migrationsDir := filepath.Join("internal", "modules", moduleID, "migrations")
    if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
        r.AddWarning("No migrations directory at %s", migrationsDir)
    }

    // 7. Frontend module config exists
    frontendDir := filepath.Join("web", "src", "modules", moduleID)
    if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
        r.AddWarning("No frontend directory at %s", frontendDir)
    }

    // 8. Search categories are in a valid range
    if ss := mod.SearchStrategy(); ss != nil {
        cats := ss.Categories()
        if len(cats) == 0 {
            r.AddWarning("Module declares no search categories")
        }
    }

    return r
}
```

**Modify** `cmd/slipstream/main.go`:
- Add early dispatch for `validate-module`:

```go
func main() {
    // Handle --complete-update before anything else
    if len(os.Args) >= 4 && os.Args[1] == "--complete-update" {
        // ... existing code
    }

    // Handle validate-module subcommand
    if len(os.Args) >= 3 && os.Args[1] == "validate-module" {
        runValidateModule(os.Args[2])
        return
    }

    // ... rest of existing main()
}

func runValidateModule(moduleID string) {
    registry := module.NewRegistry()
    module.RegisterAll(registry, nil) // nil logger OK for validation

    result := validate.ValidateModule(registry, moduleID)

    for _, w := range result.Warnings {
        fmt.Printf("  WARN: %s\n", w)
    }
    for _, e := range result.Errors {
        fmt.Printf("  ERROR: %s\n", e)
    }

    if result.OK() {
        fmt.Printf("Module %q: all checks passed (%d warnings)\n", moduleID, len(result.Warnings))
    } else {
        fmt.Printf("Module %q: %d errors, %d warnings\n", moduleID, len(result.Errors), len(result.Warnings))
        os.Exit(1)
    }
}
```

**Implementation notes:**
- The spec §19.2 mentions checking that "All required interfaces are implemented (compile-time check via `var _ Interface = (*Type)(nil)` assertions)." This is a compile-time check, not a runtime check — the validation CLI cannot verify this. It's enforced by the scaffolding tool generating `var _` assertions in the stub code (Task 11.16). The CLI documents this: "Interface compliance is checked at compile time."
- The spec mentions "Frontend `ModuleConfig` matches backend schema (route IDs align, theme color is valid)." This requires reading the TypeScript config and comparing to Go. This is complex and fragile. **Spec deviation:** Phase 11 defers frontend config validation to a separate linting rule or manual checklist. The CLI validates backend-only. Document this deviation.
- The spec mentions "Migration files parse correctly and are sequentially numbered." This can be implemented by reading the migration directory and checking filenames. Add this check.

**Verify:**
- `go build ./cmd/slipstream/...`
- `./bin/slipstream validate-module movie` — should pass
- `./bin/slipstream validate-module tv` — should pass
- `./bin/slipstream validate-module nonexistent` — should fail with "not found in registry"

---

### Task 11.16: Module Scaffolding Tool

**Depends on:** Task 11.15 complete (scaffolding should generate code that passes validation)

**Goal:** `make new-module <id>` generates the full skeleton per spec §19.1.

**Create** `scripts/new-module/main.go`:

```go
package main

import (
    "embed"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "text/template"
)

//go:embed templates/*
var templateFS embed.FS

type ModuleData struct {
    ID              string // e.g., "music"
    Name            string // e.g., "Music"
    PluralName      string // e.g., "Music" (or "Artists" etc.)
    PackageName     string // e.g., "music"
    ModuleType      string // e.g., "TypeMusic"
    GoModulePath    string // "github.com/slipstream/slipstream"
}

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintln(os.Stderr, "Usage: go run ./scripts/new-module <module-id>")
        fmt.Fprintln(os.Stderr, "Example: go run ./scripts/new-module music")
        os.Exit(1)
    }

    id := strings.ToLower(os.Args[1])
    data := ModuleData{
        ID:           id,
        Name:         strings.Title(id),
        PluralName:   strings.Title(id),
        PackageName:  id,
        ModuleType:   "Type" + strings.Title(id),
        GoModulePath: "github.com/slipstream/slipstream",
    }

    // Generate backend files
    generateFromTemplate(data, "module.go.tmpl",
        filepath.Join("internal", "modules", id, "module.go"))
    generateFromTemplate(data, "metadata.go.tmpl",
        filepath.Join("internal", "modules", id, "metadata.go"))
    generateFromTemplate(data, "search.go.tmpl",
        filepath.Join("internal", "modules", id, "search.go"))
    generateFromTemplate(data, "import.go.tmpl",
        filepath.Join("internal", "modules", id, "import_handler.go"))
    generateFromTemplate(data, "path_naming.go.tmpl",
        filepath.Join("internal", "modules", id, "path_naming.go"))
    generateFromTemplate(data, "quality.go.tmpl",
        filepath.Join("internal", "modules", id, "quality.go"))
    generateFromTemplate(data, "calendar.go.tmpl",
        filepath.Join("internal", "modules", id, "calendar.go"))
    generateFromTemplate(data, "wanted.go.tmpl",
        filepath.Join("internal", "modules", id, "wanted.go"))
    generateFromTemplate(data, "monitoring.go.tmpl",
        filepath.Join("internal", "modules", id, "monitoring.go"))
    generateFromTemplate(data, "file_parser.go.tmpl",
        filepath.Join("internal", "modules", id, "file_parser.go"))
    generateFromTemplate(data, "mock.go.tmpl",
        filepath.Join("internal", "modules", id, "mock.go"))
    generateFromTemplate(data, "notifications.go.tmpl",
        filepath.Join("internal", "modules", id, "notifications.go"))
    generateFromTemplate(data, "release_dates.go.tmpl",
        filepath.Join("internal", "modules", id, "release_dates.go"))
    generateFromTemplate(data, "routes.go.tmpl",
        filepath.Join("internal", "modules", id, "routes.go"))
    generateFromTemplate(data, "tasks.go.tmpl",
        filepath.Join("internal", "modules", id, "tasks.go"))

    // Migration directory with initial migration
    generateFromTemplate(data, "initial_migration.sql.tmpl",
        filepath.Join("internal", "modules", id, "migrations", "00001_initial.sql"))

    // Test file
    generateFromTemplate(data, "module_test.go.tmpl",
        filepath.Join("internal", "modules", id, "module_test.go"))

    // Frontend files
    generateFromTemplate(data, "frontend_config.ts.tmpl",
        filepath.Join("web", "src", "modules", id, "config.ts"))
    generateFromTemplate(data, "frontend_card.tsx.tmpl",
        filepath.Join("web", "src", "modules", id, "components", id+"-card.tsx"))
    generateFromTemplate(data, "frontend_detail.tsx.tmpl",
        filepath.Join("web", "src", "modules", id, "components", id+"-detail.tsx"))

    // SQL query templates
    generateFromTemplate(data, "queries.sql.tmpl",
        filepath.Join("internal", "database", "queries", id+"s.sql"))

    fmt.Printf("Module %q scaffolded successfully.\n", id)
    fmt.Println()
    fmt.Println("Next steps:")
    fmt.Printf("  1. Register the module in internal/module/register.go\n")
    fmt.Printf("  2. Add module.Type%s constant to internal/module/types.go\n", data.Name)
    fmt.Printf("  3. Implement TODO markers in each generated file\n")
    fmt.Printf("  4. Run: slipstream validate-module %s\n", id)
    fmt.Printf("  5. Run: go test ./internal/modules/%s/...\n", id)
}

func generateFromTemplate(data ModuleData, tmplName, outputPath string) {
    // ... template execution, directory creation, file writing
    // Each template includes TODO markers per spec §19.1
}
```

**Create** template files in `scripts/new-module/templates/`:

Each template generates a stub file with:
- Correct package declaration
- Required imports
- Struct definitions
- Interface method stubs with `// TODO: implement` markers
- Compile-time interface assertions (`var _ module.X = (*Y)(nil)`)

Example `module.go.tmpl`:
```
package {{.PackageName}}

import (
    "github.com/google/wire"
    "{{.GoModulePath}}/internal/module"
)

// Descriptor implements module.Descriptor for the {{.Name}} module.
type Descriptor struct{}

var _ module.Descriptor = (*Descriptor)(nil)

func (d *Descriptor) ID() module.Type     { return module.{{.ModuleType}} }
func (d *Descriptor) Name() string        { return "{{.Name}}" }
func (d *Descriptor) PluralName() string  { return "{{.PluralName}}" }
func (d *Descriptor) Icon() string        { return "{{.ID}}" } // TODO: choose icon
func (d *Descriptor) ThemeColor() string  { return "{{.ID}}" } // TODO: choose theme color

func (d *Descriptor) NodeSchema() module.NodeSchema {
    // TODO: Define your module's node hierarchy.
    // Examples:
    //   Flat (1 level):  [Root+Leaf]        e.g., Movie
    //   2 levels:        [Root, Leaf]        e.g., Author → Book
    //   3 levels:        [Root, Mid, Leaf]   e.g., Artist → Album → Track
    return module.NodeSchema{
        Levels: []module.NodeLevel{
            {
                Name:         "{{.ID}}",
                PluralName:   "{{.ID}}s",
                IsRoot:       true,
                IsLeaf:       true,
                HasMonitored: true,
                Searchable:   true,
            },
        },
    }
}

func (d *Descriptor) EntityTypes() []module.EntityType {
    return []module.EntityType{ module.EntityType("{{.ID}}") }
}

func (d *Descriptor) Wire() wire.ProviderSet {
    return wire.NewSet() // TODO: add service constructors
}
```

Example `queries.sql.tmpl` (common CRUD patterns — addresses Appendix C row 7):
```
-- {{.Name}} module queries
-- Generated by: make new-module {{.ID}}

-- name: Get{{.Name}} :one
SELECT * FROM {{.ID}}s WHERE id = ?;

-- name: List{{.Name}}s :many
SELECT * FROM {{.ID}}s ORDER BY sort_title ASC;

-- name: Create{{.Name}} :one
INSERT INTO {{.ID}}s (title, sort_title, path, root_folder_id, quality_profile_id, monitored, status, added_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: Update{{.Name}} :exec
UPDATE {{.ID}}s SET title = ?, sort_title = ?, monitored = ?, quality_profile_id = ?, updated_at = ? WHERE id = ?;

-- name: Delete{{.Name}} :exec
DELETE FROM {{.ID}}s WHERE id = ?;

-- name: Search{{.Name}}s :many
SELECT * FROM {{.ID}}s WHERE title LIKE ? ORDER BY sort_title ASC LIMIT 20;

-- name: List{{.Name}}sPaginated :many
-- TODO: Add pagination, filtering, and sorting parameters
SELECT * FROM {{.ID}}s ORDER BY sort_title ASC LIMIT ? OFFSET ?;

-- name: Get{{.Name}}ByPath :one
SELECT * FROM {{.ID}}s WHERE path = ?;

-- name: Update{{.Name}}Status :exec
UPDATE {{.ID}}s SET status = ? WHERE id = ?;

-- name: Update{{.Name}}Monitored :exec
UPDATE {{.ID}}s SET monitored = ? WHERE id = ?;
```

**Update** `Makefile`:
```makefile
new-module: ## Scaffold a new module (usage: make new-module MODULE_ID=music)
	@if [ -z "$(MODULE_ID)" ]; then echo "Usage: make new-module MODULE_ID=<id>"; exit 1; fi
	@go run ./scripts/new-module $(MODULE_ID)
```

**Implementation notes:**
- The subagent should study the existing movie and TV module files (created in Phases 0–9) to understand the exact patterns that templates should generate. The templates must produce code that matches the established conventions.
- Each generated file should have a header comment: `// Code generated by "make new-module". Edit as needed.`
- The `// TODO:` markers should be specific: `// TODO: implement TMDB/MusicBrainz/etc. search` not just `// TODO: implement`.
- The `queries.sql.tmpl` provides the common CRUD query patterns (Appendix C row 7: "per-module but generated from patterns"). More complex queries (joins, cascading, etc.) are module-specific and must be hand-written.
- The frontend templates generate minimal stubs that register with the module system (Phase 10's `ModuleConfig`). The card and detail components are empty shells with TODO markers.

**Verify:**
- `make new-module MODULE_ID=testmod` generates all expected files
- `go build ./internal/modules/testmod/...` compiles (stubs satisfy interfaces via `var _` checks)
- `./bin/slipstream validate-module testmod` passes (or shows only expected warnings for unimplemented features)
- Clean up: `rm -rf internal/modules/testmod/ web/src/modules/testmod/ internal/database/queries/testmods.sql`

---

### Task 11.17: Final Validation & Documentation

**Depends on:** All prior tasks complete

#### 11.17a: Full Build & Test Validation

```bash
make build && make test && make lint
cd web && bun run lint
```

All commands must pass with zero errors.

#### 11.17b: Verify Deferred Items Registry

Go through every item in the Deferred Items Registry (section above) and confirm it was addressed:

- Items 1–12: Legacy code removed, deduplication complete
- Items 13–19: Appendix C backend duplication eliminated
- Items 20–22: Contributor tooling implemented

For each, run the specific grep/verification command listed in the task's "Verify" section.

#### 11.17c: Update Documentation

**Modify** `docs/module-system-plan-structure.md`:
- Mark Phase 11 as having a detailed plan in `docs/module-system-plan-phase11.md`.

**Create or update** `internal/module/moduletest/README.md`:
- Brief usage guide for contributors showing how to write module tests using the harness.

**Verify** the scaffolding tool's "Next steps" output is accurate — a contributor following those steps should reach a compiling module.

#### 11.17d: Deferred to Phase 10+ (Unchanged)

Confirm these items from Phase 10's "Deferred to Phase 10+" section remain intentionally unaddressed:

| Deferred Item | Reason |
|---|---|
| Generic portal duplicate detection via module external IDs | No value for 2 modules |
| Generic hierarchy walk for request completion inference | No value for 2 modules |
| Full module ownership of metadata API clients (Phase 3 AD #1) | Delegation approach is sufficient |
| Frontend `ModuleConfig` ↔ backend schema cross-validation in CLI | Complex, fragile, deferred to lint rule |

---

## Validation Checklist

Run after all Phase 11 tasks are complete:

- [ ] `make build` succeeds (Go backend + frontend)
- [ ] `make test` passes (all Go tests)
- [ ] `make lint` clean (no new warnings)
- [ ] `cd web && bun run lint` clean
- [ ] Zero instances of `decisioning.SearchableItem` in codebase
- [ ] Zero instances of `shouldSkipTVRelease` in codebase
- [ ] Zero instances of `populateLegacyFields` in codebase
- [ ] Zero instances of local `generateSortTitle` or `resolveField` (only shared versions in `internal/module/`)
- [ ] Zero `registry == nil` fallback branches in calendar/availability/missing services
- [ ] Zero legacy calendar helpers (`getMovieEvents`, `movieRowToEvents`, etc.)
- [ ] `import_settings` table no longer has naming format columns
- [ ] `slots.sql` is generated from template (verify with `go generate ./internal/database/queries/ && git diff`)
- [ ] `streamingServicesWithEarlyRelease` exists in exactly one location
- [ ] `convertToSeasonMetadata` exists in exactly one location
- [ ] `make new-module MODULE_ID=test_validation` generates a compiling module skeleton
- [ ] `./bin/slipstream validate-module movie` passes
- [ ] `./bin/slipstream validate-module tv` passes
- [ ] `go test ./internal/module/moduletest/...` passes
- [ ] `go test ./internal/modules/movie/...` passes (including moduletest harness tests)
- [ ] `go test ./internal/modules/tv/...` passes (including moduletest harness tests)
- [ ] Backend items deferred from earlier phases (Deferred Items Registry) are all addressed or explicitly documented as further-deferred with rationale
