## Phase 4: Monitoring & Release Dates

**Goal:** Extract monitoring cascade and presets into the module framework's `MonitoringPresets` interface. Extract release date logic into the `ReleaseDateResolver` interface. Extract calendar logic into the `CalendarProvider` interface. Wire the `WantedCollector` interface to the existing wanted/missing collection code. After this phase, the framework can dispatch monitoring, availability transitions, calendar events, and wanted collection through module interfaces.

**Spec sections covered:** §6.1 (Monitoring), §7.1 (Release Dates and Availability)

**Depends on:** Phase 3 (module registry with MetadataProvider wired, `RefreshEntityMetadata` dispatch pattern established)

### Phase 4 Agent Context Management

This phase touches 4 relatively independent interface areas (monitoring, release dates, calendar, wanted). The recommended execution strategy:

1. **Tasks 4.1–4.4** define types and interface implementations — these are independent of each other and can run in **parallel** as separate subagents.
2. **Task 4.5** (availability service refactor) depends on 4.2 (ReleaseDateResolver implementations).
3. **Task 4.6** (calendar service refactor) depends on 4.3 (CalendarProvider implementations).
4. **Task 4.7** (Wire integration) depends on all prior tasks.
5. **Task 4.8** (validation) runs last.

**Context budget:** Each task modifies 2–4 files. Delegate all code changes to Sonnet subagents. Main context only reads verification files (<100 lines each) and runs compile checks.

---

### Phase 4 Architecture Decisions

1. **No database migrations in this phase.** Monitoring columns (`monitored`) already exist on movies, series, seasons, and episodes. Release date columns already exist (`release_date`, `physical_release_date`, `theatrical_release_date` on movies; `air_date` on episodes). The framework interfaces wrap existing data — no schema changes.

2. **Existing code stays functional during transition.** Each module implements the new interfaces by delegating to the existing service code. The framework dispatcher (`availability.Service`, `calendar.Service`, `missing.Service`) gains module awareness incrementally. Legacy paths remain until Phase 10+ when full deduplication happens.

3. **MonitoringPresets lives on the Module struct, not a separate service.** TV's `BulkMonitor` method in `internal/library/tv/seasons.go` already contains all preset logic (all, none, future, first_season, latest_season). The TV module's `MonitoringPresets` implementation wraps those existing service calls. Movie module returns an empty preset list (flat structure — monitoring is just a boolean toggle, no presets needed per spec §6.1).

4. **ReleaseDateResolver is per-module, not a shared utility.** Movies compute the earliest of digital and physical release dates (theatrical is irrelevant for downloads per spec §7.1). Episodes use air_date comparison. The existing `isMovieReleased` function uses a different approach (first-non-nil priority chain including theatrical+90d), but `ComputeAvailabilityDate` aligns with the spec's "earliest of" semantics since its purpose is different: it returns the canonical availability date for calendar display and search gating, not a boolean "is released" check. The existing `CheckReleaseDateTransitions` delegates to the SQL query which encodes its own logic, so both paths coexist. **Spec deviation:** the SQL query `UpdateUnreleasedMoviesToMissing` still uses the priority chain internally — aligning the SQL to the spec's "earliest of" semantics is deferred to Phase 10 cleanup.

5. **WantedCollector wraps existing SQL queries.** `ListMissingMovies`, `ListMissingEpisodes`, `ListUpgradableMoviesWithQuality`, `ListUpgradableEpisodesWithQuality` already exist and enforce the correct monitored+status filters. Module implementations call these queries through the existing service layer and convert results to `module.SearchableItem`. The `SearchableItem` interface (defined in Phase 0) is not yet used by the search pipeline (that's Phase 5) — in this phase we only implement the collection side.

6. **CalendarProvider replaces hard-coded media type dispatch.** The current `calendar.Service` has separate `getMovieEvents` and `getEpisodeEvents` methods. Each module's `CalendarProvider` wraps its corresponding method. The framework `calendar.Service` gains a `registry` field and iterates over enabled modules. Falls back to legacy paths if registry is nil (same pattern as Phase 3's `RefreshEntityMetadata`).

7. **Search eligibility gating on availability date is already enforced by the status model.** Items with future availability dates have `unreleased` status. The `ListMissingMovies` and `ListMissingEpisodes` queries only return items with `status IN ('missing', 'failed')` — unreleased items are never collected for search. The `triggerMovieSearchIfNeeded` method in `librarymanager/add.go:143` also skips unreleased items. No new code is needed — the `ReleaseDateResolver.CheckReleaseDateTransitions` method (Task 4.2) keeps statuses current, and the existing status-based filtering prevents searching future items. When Phase 5 builds the modular search pipeline, it will consume `WantedCollector` results which are already status-filtered.

8. **Generic framework cascade is deferred to Phase 10.** The spec §6.1 describes framework-level cascade infrastructure (setting a parent's `monitored` to false automatically propagates to descendants for any module). The existing TV-specific cascade in `internal/library/tv/seasons.go` (lines 50–56) already handles this for TV. Movie has no hierarchy, so cascade is a no-op. Building a generic framework cascade requires node schema introspection and generic parent-child traversal — complexity that provides no value until a 3rd module (music, books) is added. The existing module-specific cascade code continues to work through this transition. Phase 10 builds the generic version when all modules are in place to validate against.

9. **`existing` monitoring preset is NOT added in this phase.** The spec §6.1 mentions `existing` as a TV preset. The current codebase has 5 presets: `all`, `none`, `future`, `first_season`, `latest_season`. The `existing` preset (monitor only episodes that currently have files) is a new feature not present in the codebase and would require a new SQL query. **Spec deviation:** defer `existing` to a follow-up task. The module interface supports it — the implementation just doesn't include it yet. Document this in the plan.

---

### Task 4.1: Implement MonitoringPresets on Module Stubs

**Depends on:** Phase 3 complete (module stubs exist in `internal/modules/movie/module.go` and `internal/modules/tv/module.go`)

**Flesh out** the `MonitoringPreset` type in `internal/module/parse_types.go` (where it was defined as a skeleton in Task 0.4):

```go
// MonitoringPreset defines a named monitoring strategy that a module supports.
type MonitoringPreset struct {
    ID          string // e.g., "all", "none", "future", "first_season", "latest_season"
    Label       string // Human-readable label for UI, e.g., "All Episodes"
    Description string // e.g., "Monitor all episodes including specials"
    HasOptions  bool   // Whether this preset has additional options (e.g., "includeSpecials")
}
```

Verify the `MonitoringPresets` interface in `internal/module/interfaces.go` matches (defined in Task 0.3):

```go
type MonitoringPresets interface {
    AvailablePresets() []MonitoringPreset
    ApplyPreset(ctx context.Context, rootEntityID int64, presetID string) error
}
```

If the `ApplyPreset` signature needs an options parameter for `includeSpecials`, update it:

```go
ApplyPreset(ctx context.Context, rootEntityID int64, presetID string, options map[string]any) error
```

**This is a spec-aligned deviation** — the spec says `ApplyPreset(ctx, rootEntityID, presetID)` but the TV module's `BulkMonitor` takes an `IncludeSpecials` bool. The `options map[string]any` parameter is the simplest way to support preset-specific options without module-specific interfaces. Update `internal/module/interfaces.go` accordingly.

---

**Create** `internal/modules/movie/monitoring.go`:

```go
package movie

import (
    "context"

    "github.com/slipstream/slipstream/internal/module"
)

// AvailablePresets returns an empty list — movies are flat entities with no hierarchy.
// Monitoring is a simple boolean toggle, not a preset-based strategy.
func (m *Module) AvailablePresets() []module.MonitoringPreset {
    return nil
}

// ApplyPreset is a no-op for movies. Use the movie service's BulkUpdateMonitored directly.
func (m *Module) ApplyPreset(_ context.Context, _ int64, _ string, _ map[string]any) error {
    return nil
}
```

---

**Create** `internal/modules/tv/monitoring.go`:

```go
package tv

import (
    "context"
    "fmt"

    "github.com/slipstream/slipstream/internal/module"
    tvlib "github.com/slipstream/slipstream/internal/library/tv"
)

// TV monitoring presets matching the current BulkMonitor implementation
// in internal/library/tv/seasons.go.
var tvMonitoringPresets = []module.MonitoringPreset{
    {ID: "all", Label: "All Episodes", Description: "Monitor all episodes (excludes specials by default)", HasOptions: true},
    {ID: "none", Label: "None", Description: "Unmonitor all episodes and seasons"},
    {ID: "future", Label: "Future Episodes", Description: "Only monitor episodes that haven't aired yet", HasOptions: true},
    {ID: "first_season", Label: "First Season", Description: "Only monitor season 1 episodes"},
    {ID: "latest_season", Label: "Latest Season", Description: "Only monitor the most recent season"},
}

func (m *Module) AvailablePresets() []module.MonitoringPreset {
    return tvMonitoringPresets
}

// ApplyPreset delegates to the existing TV service's BulkMonitor method.
// Options: "includeSpecials" (bool) — applies to "all" and "future" presets.
func (m *Module) ApplyPreset(ctx context.Context, rootEntityID int64, presetID string, options map[string]any) error {
    monitorType, err := toMonitorType(presetID)
    if err != nil {
        return err
    }

    includeSpecials := false
    if v, ok := options["includeSpecials"]; ok {
        if b, ok := v.(bool); ok {
            includeSpecials = b
        }
    }

    return m.tvService.BulkMonitor(ctx, rootEntityID, tvlib.BulkMonitorInput{
        MonitorType:     monitorType,
        IncludeSpecials: includeSpecials,
    })
}

func toMonitorType(presetID string) (tvlib.MonitorType, error) {
    switch presetID {
    case "all":
        return tvlib.MonitorTypeAll, nil
    case "none":
        return tvlib.MonitorTypeNone, nil
    case "future":
        return tvlib.MonitorTypeFuture, nil
    case "first_season":
        return tvlib.MonitorTypeFirstSeason, nil
    case "latest_season":
        return tvlib.MonitorTypeLatest, nil
    default:
        return "", fmt.Errorf("unknown TV monitoring preset: %s", presetID)
    }
}
```

**Important:** The TV `Module` struct (defined in `internal/modules/tv/module.go` from Phase 3) needs a `tvService` field of type `*tv.Service`. This should already be present from Phase 3's `NewModule` constructor (which receives the TV service for metadata delegation). If not, add it:

```go
type Module struct {
    descriptor       *Descriptor
    metadataProvider *metadataProvider
    tvService        *tvlib.Service // needed for MonitoringPresets, WantedCollector
}
```

Verify the `NewModule` constructor passes `tvService` to the struct.

---

**Update** `internal/modules/movie/module.go` — ensure the Module struct satisfies `module.MonitoringPresets`:

The `Module` struct must already embed or satisfy all required interfaces. Verify the compile-time assertion exists or add:

```go
var _ module.MonitoringPresets = (*Module)(nil)
```

Do the same in `internal/modules/tv/module.go`.

---

**Verify:**
- `go build ./internal/modules/movie/...` compiles
- `go build ./internal/modules/tv/...` compiles
- `go vet ./internal/modules/...` clean

---

### Task 4.2: Implement ReleaseDateResolver on Module Stubs

**Depends on:** Phase 3 complete

The `ReleaseDateResolver` interface (defined in Task 0.3):

```go
type ReleaseDateResolver interface {
    ComputeAvailabilityDate(ctx context.Context, entityID int64) (*time.Time, error)
    CheckReleaseDateTransitions(ctx context.Context) (transitioned int, err error)
}
```

---

**Create** `internal/modules/movie/release_dates.go`:

```go
package movie

import (
    "context"
    "database/sql"
    "time"

    "github.com/slipstream/slipstream/internal/database/sqlc"
)

// ComputeAvailabilityDate returns the date a movie becomes downloadable.
// Returns the earliest of digital and physical release dates per spec §7.1.
// Theatrical release date is irrelevant for downloads and is not considered.
func (m *Module) ComputeAvailabilityDate(ctx context.Context, entityID int64) (*time.Time, error) {
    movie, err := m.movieService.Get(ctx, entityID)
    if err != nil {
        return nil, err
    }

    var earliest *time.Time

    if movie.ReleaseDate != nil {
        earliest = movie.ReleaseDate
    }
    if movie.PhysicalReleaseDate != nil {
        if earliest == nil || movie.PhysicalReleaseDate.Before(*earliest) {
            earliest = movie.PhysicalReleaseDate
        }
    }

    return earliest, nil // nil if no release date known
}

// CheckReleaseDateTransitions transitions unreleased movies to missing
// when their availability date has passed. Delegates to the existing SQL query
// UpdateUnreleasedMoviesToMissing which encodes the same priority chain.
func (m *Module) CheckReleaseDateTransitions(ctx context.Context) (int, error) {
    result, err := m.queries.UpdateUnreleasedMoviesToMissing(ctx)
    if err != nil {
        return 0, err
    }
    rows, _ := result.RowsAffected()
    return int(rows), nil
}
```

**Note:** The `Module` struct needs `movieService *movies.Service` and `queries *sqlc.Queries` fields. The `movieService` should already exist from Phase 3's metadata provider. The `queries` field provides direct SQL access for the bulk transition query. Add it to `NewModule` if not already present:

```go
func NewModule(movieSvc *movies.Service, metadataSvc *metadata.Service, db *sql.DB, logger *zerolog.Logger) *Module {
    return &Module{
        // ... existing fields ...
        movieService: movieSvc,
        queries:      sqlc.New(db),
    }
}
```

---

**Create** `internal/modules/tv/release_dates.go`:

```go
package tv

import (
    "context"
    "time"
)

// ComputeAvailabilityDate returns the date an episode becomes downloadable.
// For TV, this is simply the air date.
// entityID is an episode ID (the leaf node level).
func (m *Module) ComputeAvailabilityDate(ctx context.Context, entityID int64) (*time.Time, error) {
    episode, err := m.tvService.GetEpisode(ctx, entityID)
    if err != nil {
        return nil, err
    }
    return episode.AirDate, nil
}

// CheckReleaseDateTransitions transitions unreleased episodes to missing
// when their air date has passed. Delegates to the existing SQL queries.
func (m *Module) CheckReleaseDateTransitions(ctx context.Context) (int, error) {
    // UpdateUnreleasedEpisodesToMissing: air_date <= date('now')
    result1, err := m.queries.UpdateUnreleasedEpisodesToMissing(ctx)
    if err != nil {
        return 0, err
    }
    rows1, _ := result1.RowsAffected()

    // UpdateUnreleasedEpisodesToMissingDateOnly: same check (kept for backward compat)
    result2, err := m.queries.UpdateUnreleasedEpisodesToMissingDateOnly(ctx)
    if err != nil {
        return int(rows1), err
    }
    rows2, _ := result2.RowsAffected()

    return int(rows1 + rows2), nil
}
```

**Note:** Verify that `tvService.GetEpisode` exists. Check `internal/library/tv/episodes.go` — it likely has a `GetEpisode(ctx, id)` method. If it doesn't exist, the TV service's `ListEpisodes` filtered by ID can be used instead, or a `GetEpisode` method should be added to the TV service. Read `internal/library/tv/episodes.go` to confirm.

Also, the TV `Module` struct needs `queries *sqlc.Queries`. Add to constructor if not present.

---

**Add compile-time assertions** in both module files:

```go
var _ module.ReleaseDateResolver = (*Module)(nil)
```

**Verify:**
- `go build ./internal/modules/movie/...` compiles
- `go build ./internal/modules/tv/...` compiles
- `go vet ./internal/modules/...` clean

---

### Task 4.3: Implement CalendarProvider on Module Stubs

**Depends on:** Phase 3 complete

The `CalendarProvider` interface (defined in Task 0.3):

```go
type CalendarProvider interface {
    GetItemsInDateRange(ctx context.Context, start, end time.Time) ([]CalendarItem, error)
}
```

First, flesh out the `CalendarItem` type in `internal/module/calendar_types.go` (skeleton from Task 0.4). This type must carry enough data for the unified calendar UI:

```go
package module

import "time"

// CalendarItem represents a single item with a date for the unified calendar view.
type CalendarItem struct {
    ID            int64     // Entity ID
    Title         string    // Display title
    ModuleType    Type      // Module discriminator (e.g., TypeMovie, TypeTV)
    EntityType    EntityType // e.g., EntityMovie, EntityEpisode
    EventType     string    // Module-specific event type (e.g., "digital", "physical", "airDate")
    Date          time.Time // The calendar date
    Status        string    // Current status (missing, available, etc.)
    Monitored     bool      // Whether the entity is monitored

    // Metadata for display — all optional
    ExternalIDs map[string]string // e.g., {"tmdb": "123", "tvdb": "456"}
    Year        int
    ParentID    int64  // e.g., series_id for episodes
    ParentTitle string // e.g., series title for episodes
    Extra       map[string]any // Module-specific extra data (season/episode numbers, network, etc.)
}
```

---

**Create** `internal/modules/movie/calendar.go`:

```go
package movie

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/slipstream/slipstream/internal/database/sqlc"
    "github.com/slipstream/slipstream/internal/module"
)

// GetItemsInDateRange returns movie release events within the date range.
// Produces separate events for digital and physical releases (same as existing
// calendar.Service.getMovieEvents logic).
func (m *Module) GetItemsInDateRange(ctx context.Context, start, end time.Time) ([]module.CalendarItem, error) {
    rows, err := m.queries.GetMoviesInDateRange(ctx, sqlc.GetMoviesInDateRangeParams{
        FromReleaseDate:           sql.NullTime{Time: start, Valid: true},
        ToReleaseDate:             sql.NullTime{Time: end, Valid: true},
        FromPhysicalReleaseDate:   sql.NullTime{Time: start, Valid: true},
        ToPhysicalReleaseDate:     sql.NullTime{Time: end, Valid: true},
        FromTheatricalReleaseDate: sql.NullTime{Time: start, Valid: true},
        ToTheatricalReleaseDate:   sql.NullTime{Time: end, Valid: true},
    })
    if err != nil {
        return nil, err
    }

    var items []module.CalendarItem
    startStr := start.Format("2006-01-02")
    endStr := end.Format("2006-01-02")

    for _, row := range rows {
        status := m.resolveMovieStatus(ctx, row)

        if event, ok := m.movieReleaseItem(row, "digital", row.ReleaseDate, status, startStr, endStr); ok {
            items = append(items, event)
        }
        if event, ok := m.movieReleaseItem(row, "physical", row.PhysicalReleaseDate, status, startStr, endStr); ok {
            items = append(items, event)
        }
    }

    return items, nil
}

func (m *Module) resolveMovieStatus(ctx context.Context, row *sqlc.Movie) string {
    count, err := m.queries.CountMovieFiles(ctx, row.ID)
    if err == nil && count > 0 {
        return "available"
    }
    return row.Status
}

func (m *Module) movieReleaseItem(row *sqlc.Movie, eventType string, date sql.NullTime, status, startStr, endStr string) (module.CalendarItem, bool) {
    if !date.Valid {
        return module.CalendarItem{}, false
    }
    dateStr := date.Time.Format("2006-01-02")
    if dateStr < startStr || dateStr > endStr {
        return module.CalendarItem{}, false
    }

    externalIDs := make(map[string]string)
    if row.TmdbID.Valid {
        externalIDs["tmdb"] = fmt.Sprintf("%d", row.TmdbID.Int64)
    }
    if row.ImdbID.Valid {
        externalIDs["imdb"] = row.ImdbID.String
    }

    year := 0
    if row.Year.Valid {
        year = int(row.Year.Int64)
    }

    return module.CalendarItem{
        ID:          row.ID,
        Title:       row.Title,
        ModuleType:  module.TypeMovie,
        EntityType:  module.EntityMovie,
        EventType:   eventType,
        Date:        date.Time,
        Status:      status,
        Monitored:   row.Monitored,
        ExternalIDs: externalIDs,
        Year:        year,
    }, true
}
```

---

**Create** `internal/modules/tv/calendar.go`:

```go
package tv

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/slipstream/slipstream/internal/database/sqlc"
    "github.com/slipstream/slipstream/internal/module"
)

// streamingServicesWithEarlyRelease lists networks that release content the night before.
// Duplicated from internal/calendar/service.go — will be consolidated in Phase 10.
var streamingServicesWithEarlyRelease = map[string]bool{
    "Apple TV+": true,
    "Apple TV":  true,
}

// GetItemsInDateRange returns episode air date events within the date range.
// Groups 3+ same-season same-day episodes into a single season release event
// (same logic as existing calendar.Service.getEpisodeEvents).
func (m *Module) GetItemsInDateRange(ctx context.Context, start, end time.Time) ([]module.CalendarItem, error) {
    rows, err := m.queries.GetEpisodesInDateRange(ctx, sqlc.GetEpisodesInDateRangeParams{
        AirDate:   sql.NullTime{Time: start, Valid: true},
        AirDate_2: sql.NullTime{Time: end, Valid: true},
    })
    if err != nil {
        return nil, err
    }

    // Group by series+season+date to detect full season releases
    type seasonKey struct {
        SeriesID     int64
        SeasonNumber int64
        Date         string
    }
    grouped := make(map[seasonKey][]*sqlc.GetEpisodesInDateRangeRow)
    for _, row := range rows {
        key := seasonKey{
            SeriesID:     row.SeriesID,
            SeasonNumber: row.SeasonNumber,
            Date:         row.AirDate.Time.Format("2006-01-02"),
        }
        grouped[key] = append(grouped[key], row)
    }

    var items []module.CalendarItem
    for key, episodes := range grouped {
        if len(episodes) >= 3 {
            items = append(items, m.seasonReleaseItem(ctx, key.SeriesID, key.SeasonNumber, key.Date, episodes))
        } else {
            for _, row := range episodes {
                items = append(items, m.episodeItem(ctx, row))
            }
        }
    }

    return items, nil
}

func (m *Module) seasonReleaseItem(ctx context.Context, seriesID, seasonNumber int64, date string, episodes []*sqlc.GetEpisodesInDateRangeRow) module.CalendarItem {
    first := episodes[0]

    availableCount := 0
    for _, ep := range episodes {
        files, err := m.queries.ListEpisodeFilesByEpisode(ctx, ep.ID)
        if err == nil && len(files) > 0 {
            availableCount++
        }
    }

    status := "missing"
    if availableCount == len(episodes) {
        status = "available"
    } else if availableCount > 0 {
        status = "downloading"
    }

    network := ""
    if first.Network.Valid {
        network = first.Network.String
    }

    externalIDs := make(map[string]string)
    if first.SeriesTmdbID.Valid {
        externalIDs["tmdb"] = fmt.Sprintf("%d", first.SeriesTmdbID.Int64)
    }

    dateTime, _ := time.Parse("2006-01-02", date)

    return module.CalendarItem{
        ID:          first.ID,
        Title:       fmt.Sprintf("Season %d", seasonNumber),
        ModuleType:  module.TypeTV,
        EntityType:  module.EntityEpisode,
        EventType:   "airDate",
        Date:        dateTime,
        Status:      status,
        Monitored:   first.Monitored,
        ExternalIDs: externalIDs,
        ParentID:    seriesID,
        ParentTitle: first.SeriesTitle,
        Extra: map[string]any{
            "seasonNumber":  int(seasonNumber),
            "episodeNumber": len(episodes), // episode count stored in episodeNumber (matches existing CalendarEvent.EpisodeNumber usage for season releases)
            "network":       network,
            "earlyAccess":   streamingServicesWithEarlyRelease[network],
        },
    }
}

func (m *Module) episodeItem(ctx context.Context, row *sqlc.GetEpisodesInDateRangeRow) module.CalendarItem {
    hasFile := false
    files, err := m.queries.ListEpisodeFilesByEpisode(ctx, row.ID)
    if err == nil && len(files) > 0 {
        hasFile = true
    }

    status := "missing"
    if hasFile {
        status = "available"
    }

    network := ""
    if row.Network.Valid {
        network = row.Network.String
    }

    title := row.Title.String
    if title == "" {
        title = fmt.Sprintf("Episode %d", row.EpisodeNumber)
    }

    externalIDs := make(map[string]string)
    if row.SeriesTmdbID.Valid {
        externalIDs["tmdb"] = fmt.Sprintf("%d", row.SeriesTmdbID.Int64)
    }

    return module.CalendarItem{
        ID:          row.ID,
        Title:       title,
        ModuleType:  module.TypeTV,
        EntityType:  module.EntityEpisode,
        EventType:   "airDate",
        Date:        row.AirDate.Time,
        Status:      status,
        Monitored:   row.Monitored,
        ExternalIDs: externalIDs,
        ParentID:    row.SeriesID,
        ParentTitle: row.SeriesTitle,
        Extra: map[string]any{
            "seasonNumber":  int(row.SeasonNumber),
            "episodeNumber": int(row.EpisodeNumber),
            "network":       network,
            "earlyAccess":   streamingServicesWithEarlyRelease[network],
        },
    }
}
```

**Add compile-time assertions** in both module files:

```go
var _ module.CalendarProvider = (*Module)(nil)
```

**Verify:**
- `go build ./internal/modules/movie/...` compiles
- `go build ./internal/modules/tv/...` compiles

---

### Task 4.4: Implement WantedCollector on Module Stubs

**Depends on:** Phase 3 complete

The `WantedCollector` interface (defined in Task 0.3):

```go
type WantedCollector interface {
    CollectMissing(ctx context.Context) ([]SearchableItem, error)
    CollectUpgradable(ctx context.Context) ([]SearchableItem, error)
}
```

This task implements the collection side only. The `module.SearchableItem` interface is defined in Phase 0's `search_types.go` but is NOT yet consumed by the search pipeline (that's Phase 5). In this phase we return a concrete struct satisfying the interface.

**First**, verify or flesh out the `SearchableItem` interface in `internal/module/search_types.go`. The spec §5.3 defines:

```go
type SearchableItem interface {
    GetModuleType() string
    GetMediaType() string
    GetEntityID() int64
    GetTitle() string
    GetExternalIDs() map[string]string
    GetQualityProfile() *QualityProfile
    GetCurrentQuality() *QualityItem
    GetSearchParams() SearchParams
}
```

This was defined as a skeleton in Task 0.4. If `SearchableItem` is currently a struct, convert it to an interface as specified. If it's already an interface, verify the methods match. **Note:** The existing `autosearch.SearchableItem` struct (in `internal/autosearch/types.go` or `internal/decisioning/types.go`) is a concrete struct used by the current search pipeline. The module system's `SearchableItem` is an *interface*. The two will coexist until Phase 5, when the search pipeline is refactored to use the interface.

For now, create a concrete type that satisfies the interface for WantedCollector results:

**Create** `internal/module/wanted_item.go`:

```go
package module

// WantedItem is a concrete SearchableItem implementation returned by WantedCollector.
// In Phase 5, the search pipeline will consume these through the SearchableItem interface.
type WantedItem struct {
    ModuleType_     Type
    MediaType_      string
    EntityID_       int64
    Title_          string
    ExternalIDs_    map[string]string
    QualityProfileID_ int64
    CurrentQualityID_ *int64 // nil if no file (missing), non-nil for upgradable
    SearchParams_   SearchParams
}

func (w *WantedItem) GetModuleType() string              { return string(w.ModuleType_) }
func (w *WantedItem) GetMediaType() string               { return w.MediaType_ }
func (w *WantedItem) GetEntityID() int64                 { return w.EntityID_ }
func (w *WantedItem) GetTitle() string                   { return w.Title_ }
func (w *WantedItem) GetExternalIDs() map[string]string  { return w.ExternalIDs_ }
func (w *WantedItem) GetQualityProfileID() int64         { return w.QualityProfileID_ }
func (w *WantedItem) GetCurrentQualityID() *int64        { return w.CurrentQualityID_ }
func (w *WantedItem) GetSearchParams() SearchParams      { return w.SearchParams_ }
```

**Implementation note:** The `GetQualityProfile()` and `GetCurrentQuality()` methods from the spec return profile/item objects. But in this phase, the quality system hasn't been wired through the module interface yet — quality profiles are module-scoped (Phase 2) but not yet connected to `SearchableItem`. Instead, use `GetQualityProfileID() int64` and `GetCurrentQualityID() *int64` which reference IDs. The search pipeline (Phase 5) will resolve these to profile objects. Update the `SearchableItem` interface signature in `internal/module/search_types.go` accordingly:

```go
type SearchableItem interface {
    GetModuleType() string
    GetMediaType() string
    GetEntityID() int64
    GetTitle() string
    GetExternalIDs() map[string]string
    GetQualityProfileID() int64
    GetCurrentQualityID() *int64  // nil if no file
    GetSearchParams() SearchParams
}
```

**Deviation from spec:** The spec §5.3 has `GetQualityProfile() *QualityProfile` and `GetCurrentQuality() *QualityItem`. We use ID-based accessors instead because resolving to full objects here would create a circular dependency (WantedCollector would need to import quality service). The Phase 5 search pipeline can resolve IDs to objects. Note this deviation in a comment on the interface.

---

**Create** `internal/modules/movie/wanted.go`:

```go
package movie

import (
    "context"
    "fmt"

    "github.com/slipstream/slipstream/internal/module"
)

// CollectMissing returns all monitored movies with status 'missing' or 'failed'.
// Wraps the existing ListMissingMovies query (status IN ('missing', 'failed')).
// Both statuses are collected — the search pipeline (Phase 5) decides retry policy.
func (m *Module) CollectMissing(ctx context.Context) ([]module.SearchableItem, error) {
    rows, err := m.queries.ListMissingMovies(ctx)
    if err != nil {
        return nil, fmt.Errorf("movie: collect missing: %w", err)
    }

    items := make([]module.SearchableItem, 0, len(rows))
    for _, row := range rows {
        externalIDs := make(map[string]string)
        if row.TmdbID.Valid {
            externalIDs["tmdb"] = fmt.Sprintf("%d", row.TmdbID.Int64)
        }
        if row.ImdbID.Valid && row.ImdbID.String != "" {
            externalIDs["imdb"] = row.ImdbID.String
        }
        if row.TvdbID.Valid {
            externalIDs["tvdb"] = fmt.Sprintf("%d", row.TvdbID.Int64)
        }

        var profileID int64
        if row.QualityProfileID.Valid {
            profileID = row.QualityProfileID.Int64
        }

        items = append(items, &module.WantedItem{
            ModuleType_:       module.TypeMovie,
            MediaType_:        "movie",
            EntityID_:         row.ID,
            Title_:            row.Title,
            ExternalIDs_:      externalIDs,
            QualityProfileID_: profileID,
            CurrentQualityID_: nil, // missing — no file
        })
    }

    return items, nil
}

// CollectUpgradable returns all monitored movies with status 'upgradable'.
// Wraps the existing ListUpgradableMoviesWithQuality query.
func (m *Module) CollectUpgradable(ctx context.Context) ([]module.SearchableItem, error) {
    rows, err := m.queries.ListUpgradableMoviesWithQuality(ctx)
    if err != nil {
        return nil, fmt.Errorf("movie: collect upgradable: %w", err)
    }

    items := make([]module.SearchableItem, 0, len(rows))
    for _, row := range rows {
        externalIDs := make(map[string]string)
        if row.TmdbID.Valid {
            externalIDs["tmdb"] = fmt.Sprintf("%d", row.TmdbID.Int64)
        }
        if row.ImdbID.Valid && row.ImdbID.String != "" {
            externalIDs["imdb"] = row.ImdbID.String
        }

        var profileID int64
        if row.QualityProfileID.Valid {
            profileID = row.QualityProfileID.Int64
        }

        var currentQIDPtr *int64
        if row.CurrentQualityID.Valid {
            v := row.CurrentQualityID.Int64
            currentQIDPtr = &v
        }

        items = append(items, &module.WantedItem{
            ModuleType_:       module.TypeMovie,
            MediaType_:        "movie",
            EntityID_:         row.ID,
            Title_:            row.Title,
            ExternalIDs_:      externalIDs,
            QualityProfileID_: profileID,
            CurrentQualityID_: currentQIDPtr,
        })
    }

    return items, nil
}
```

**Note:** `row.CurrentQualityID` is `sql.NullInt64` in the generated sqlc type (`ListUpgradableMoviesWithQualityRow`). The query JOINs movie_files so in practice the value is always present, but the generated type is nullable. Extract with `.Int64` after checking `.Valid`.

---

**Create** `internal/modules/tv/wanted.go`:

```go
package tv

import (
    "context"
    "fmt"

    "github.com/slipstream/slipstream/internal/module"
)

// CollectMissing returns all monitored episodes with status 'missing' or 'failed'.
// Wraps the existing ListMissingEpisodes query which enforces the 3-level
// monitoring check: series.monitored AND season.monitored AND episode.monitored.
// Both statuses are collected — the search pipeline (Phase 5) decides retry policy.
func (m *Module) CollectMissing(ctx context.Context) ([]module.SearchableItem, error) {
    rows, err := m.queries.ListMissingEpisodes(ctx)
    if err != nil {
        return nil, fmt.Errorf("tv: collect missing: %w", err)
    }

    items := make([]module.SearchableItem, 0, len(rows))
    for _, row := range rows {
        externalIDs := make(map[string]string)
        if row.SeriesTvdbID.Valid {
            externalIDs["tvdb"] = fmt.Sprintf("%d", row.SeriesTvdbID.Int64)
        }
        if row.SeriesTmdbID.Valid {
            externalIDs["tmdb"] = fmt.Sprintf("%d", row.SeriesTmdbID.Int64)
        }
        if row.SeriesImdbID.Valid && row.SeriesImdbID.String != "" {
            externalIDs["imdb"] = row.SeriesImdbID.String
        }

        var profileID int64
        if row.SeriesQualityProfileID.Valid {
            profileID = row.SeriesQualityProfileID.Int64
        }

        items = append(items, &module.WantedItem{
            ModuleType_:       module.TypeTV,
            MediaType_:        "episode",
            EntityID_:         row.ID,
            Title_:            row.SeriesTitle,
            ExternalIDs_:      externalIDs,
            QualityProfileID_: profileID,
            CurrentQualityID_: nil,
            SearchParams_: module.SearchParams{
                Extra: map[string]any{
                    "seriesId":      row.SeriesID,
                    "seasonNumber":  int(row.SeasonNumber),
                    "episodeNumber": int(row.EpisodeNumber),
                },
            },
        })
    }

    return items, nil
}

// CollectUpgradable returns all monitored episodes with status 'upgradable'.
// Wraps the existing ListUpgradableEpisodesWithQuality query.
func (m *Module) CollectUpgradable(ctx context.Context) ([]module.SearchableItem, error) {
    rows, err := m.queries.ListUpgradableEpisodesWithQuality(ctx)
    if err != nil {
        return nil, fmt.Errorf("tv: collect upgradable: %w", err)
    }

    items := make([]module.SearchableItem, 0, len(rows))
    for _, row := range rows {
        externalIDs := make(map[string]string)
        if row.SeriesTvdbID.Valid {
            externalIDs["tvdb"] = fmt.Sprintf("%d", row.SeriesTvdbID.Int64)
        }
        if row.SeriesTmdbID.Valid {
            externalIDs["tmdb"] = fmt.Sprintf("%d", row.SeriesTmdbID.Int64)
        }
        if row.SeriesImdbID.Valid && row.SeriesImdbID.String != "" {
            externalIDs["imdb"] = row.SeriesImdbID.String
        }

        var profileID int64
        if row.SeriesQualityProfileID.Valid {
            profileID = row.SeriesQualityProfileID.Int64
        }

        var currentQIDPtr *int64
        if row.CurrentQualityID.Valid {
            v := row.CurrentQualityID.Int64
            currentQIDPtr = &v
        }

        items = append(items, &module.WantedItem{
            ModuleType_:       module.TypeTV,
            MediaType_:        "episode",
            EntityID_:         row.ID,
            Title_:            row.SeriesTitle,
            ExternalIDs_:      externalIDs,
            QualityProfileID_: profileID,
            CurrentQualityID_: currentQIDPtr,
            SearchParams_: module.SearchParams{
                Extra: map[string]any{
                    "seriesId":      row.SeriesID,
                    "seasonNumber":  int(row.SeasonNumber),
                    "episodeNumber": int(row.EpisodeNumber),
                },
            },
        })
    }

    return items, nil
}
```

**Note:** `row.CurrentQualityID` is `sql.NullInt64` in the generated sqlc type (`ListUpgradableEpisodesWithQualityRow`). Extract with `.Int64` after checking `.Valid`.

**Add compile-time assertions** in both module files:

```go
var _ module.WantedCollector = (*Module)(nil)
```

**Verify:**
- `go build ./internal/modules/movie/...` compiles
- `go build ./internal/modules/tv/...` compiles

---

### Task 4.5: Refactor Availability Service to Dispatch Through Module Providers

**Depends on:** Task 4.2 (ReleaseDateResolver implementations)

**Modify** `internal/availability/service.go` — add module registry dependency:

```go
import "github.com/slipstream/slipstream/internal/module"

type Service struct {
    db       *sql.DB
    queries  *sqlc.Queries
    logger   *zerolog.Logger
    registry *module.Registry // NEW — nil during transition period
}

func NewService(db *sql.DB, logger *zerolog.Logger) *Service {
    // ... existing code ...
}

// SetRegistry sets the module registry for dispatch. Called after DI setup.
func (s *Service) SetRegistry(registry *module.Registry) {
    s.registry = registry
}
```

**Update** `RefreshAll` to dispatch through module providers when the registry is available:

```go
func (s *Service) RefreshAll(ctx context.Context) error {
    s.logger.Info().Msg("Starting status refresh for all media")

    if s.registry != nil {
        return s.refreshViaModules(ctx)
    }
    return s.refreshLegacy(ctx)
}

func (s *Service) refreshViaModules(ctx context.Context) error {
    totalTransitioned := 0
    for _, mod := range s.registry.Enabled() {
        resolver, ok := mod.(module.ReleaseDateResolver)
        if !ok {
            continue
        }
        count, err := resolver.CheckReleaseDateTransitions(ctx)
        if err != nil {
            s.logger.Error().Err(err).Str("module", string(mod.ID())).Msg("Failed to check release date transitions")
            continue // Don't fail the whole refresh for one module
        }
        totalTransitioned += count
    }
    s.logger.Info().Int("transitioned", totalTransitioned).Msg("Status refresh completed via modules")
    return nil
}

func (s *Service) refreshLegacy(ctx context.Context) error {
    // Move existing RefreshAll body here
    moviesUpdated, err := s.RefreshMovies(ctx)
    if err != nil {
        s.logger.Error().Err(err).Msg("Failed to refresh movie status")
        return err
    }
    episodesUpdated, err := s.RefreshEpisodes(ctx)
    if err != nil {
        s.logger.Error().Err(err).Msg("Failed to refresh episode status")
        return err
    }
    s.logger.Info().
        Int64("movies", moviesUpdated).
        Int64("episodes", episodesUpdated).
        Msg("Status refresh completed (legacy)")
    return nil
}
```

**Important:** The existing `RefreshMovies` and `RefreshEpisodes` methods are NOT removed — they remain for the legacy fallback path and for any direct callers.

**Implementation notes for the subagent:**
- Read `internal/availability/service.go` first to see the full current implementation.
- Read `internal/module/registry.go` (created in Phase 0) to understand the `Registry.Enabled()` method — it should return all enabled modules. If it returns `[]module.Module` or similar, the subagent needs to type-assert to `module.ReleaseDateResolver`.
- The `module.Module` interface bundle (defined in Phase 0's Task 0.5) embeds all required interfaces. So every module returned by `Enabled()` satisfies `ReleaseDateResolver`. The type assertion is still good practice for safety and for optional interfaces.
- Keep both paths working — `registry == nil` means legacy.

**Verify:**
- `go build ./internal/availability/...` compiles
- `go vet ./internal/availability/...` clean

---

### Task 4.6: Refactor Calendar Service to Dispatch Through Module Providers

**Depends on:** Task 4.3 (CalendarProvider implementations)

**Modify** `internal/calendar/service.go` — add module registry dependency:

```go
import "github.com/slipstream/slipstream/internal/module"

type Service struct {
    db       *sql.DB
    queries  *sqlc.Queries
    logger   *zerolog.Logger
    registry *module.Registry // NEW — nil during transition period
}
```

Update the constructor and `SetDB`:

```go
func NewService(db *sql.DB, logger *zerolog.Logger) *Service {
    // ... existing ...
}

func (s *Service) SetRegistry(registry *module.Registry) {
    s.registry = registry
}
```

**Update** `GetEvents` to dispatch through module providers:

```go
func (s *Service) GetEvents(ctx context.Context, start, end time.Time) ([]CalendarEvent, error) {
    if s.registry != nil {
        return s.getEventsViaModules(ctx, start, end)
    }
    return s.getEventsLegacy(ctx, start, end)
}

func (s *Service) getEventsViaModules(ctx context.Context, start, end time.Time) ([]CalendarEvent, error) {
    var events []CalendarEvent
    for _, mod := range s.registry.Enabled() {
        provider, ok := mod.(module.CalendarProvider)
        if !ok {
            continue
        }
        items, err := provider.GetItemsInDateRange(ctx, start, end)
        if err != nil {
            s.logger.Error().Err(err).Str("module", string(mod.ID())).Msg("Failed to get calendar items")
            continue
        }
        for _, item := range items {
            events = append(events, calendarItemToEvent(item))
        }
    }
    return events, nil
}

func (s *Service) getEventsLegacy(ctx context.Context, start, end time.Time) ([]CalendarEvent, error) {
    // Move existing GetEvents body here
    var events []CalendarEvent
    movieEvents, err := s.getMovieEvents(ctx, start, end)
    if err != nil {
        s.logger.Error().Err(err).Msg("Failed to get movie events")
        return nil, err
    }
    events = append(events, movieEvents...)
    episodeEvents, err := s.getEpisodeEvents(ctx, start, end)
    if err != nil {
        s.logger.Error().Err(err).Msg("Failed to get episode events")
        return nil, err
    }
    events = append(events, episodeEvents...)
    return events, nil
}
```

**Add** the conversion function `calendarItemToEvent` that maps `module.CalendarItem` to the existing `CalendarEvent` struct:

```go
func calendarItemToEvent(item module.CalendarItem) CalendarEvent {
    event := CalendarEvent{
        ID:        item.ID,
        Title:     item.Title,
        MediaType: string(item.EntityType), // "movie" or "episode"
        EventType: item.EventType,
        Date:      item.Date.Format("2006-01-02"),
        Status:    item.Status,
        Monitored: item.Monitored,
        Year:      item.Year,
    }

    // Extract TMDB ID from external IDs
    if tmdb, ok := item.ExternalIDs["tmdb"]; ok {
        event.TmdbID, _ = strconv.Atoi(tmdb)
    }

    // Episode-specific fields from Extra
    if item.ParentID != 0 {
        event.SeriesID = item.ParentID
        event.SeriesTitle = item.ParentTitle
    }
    if extra := item.Extra; extra != nil {
        if sn, ok := extra["seasonNumber"].(int); ok {
            event.SeasonNumber = sn
        }
        if en, ok := extra["episodeNumber"].(int); ok {
            event.EpisodeNumber = en
        }
        if net, ok := extra["network"].(string); ok {
            event.Network = net
        }
        if ea, ok := extra["earlyAccess"].(bool); ok {
            event.EarlyAccess = ea
        }
    }

    return event
}
```

**Important:** The existing `CalendarEvent` struct, `getMovieEvents`, `getEpisodeEvents`, and all helper functions (e.g., `resolveMovieStatus`, `movieRowToEvents`, `createEpisodeEvent`, `createSeasonReleaseEvent`) are NOT removed — they serve the legacy path. They can be removed in Phase 10 when the module framework is fully validated.

**Implementation notes for the subagent:**
- Read `internal/calendar/service.go` fully first — it's ~330 lines.
- The existing `getMovieEvents` and `getEpisodeEvents` private methods stay. `GetEvents` becomes a router.
- Add `"strconv"` to imports for `Atoi` in `calendarItemToEvent`.
- The `module.CalendarItem` type was fleshed out in Task 4.3.

**Verify:**
- `go build ./internal/calendar/...` compiles
- `go vet ./internal/calendar/...` clean

---

### Task 4.7: Wire Integration — Connect Module Providers to Services

**Depends on:** Tasks 4.1–4.6

**Modify** `internal/modules/movie/module.go` — update the `Module` struct and `NewModule` constructor to hold all Phase 4 dependencies:

```go
type Module struct {
    descriptor       *Descriptor
    metadataProvider *metadataProvider // from Phase 3
    movieService     *movies.Service
    queries          *sqlc.Queries
}
```

If `movieService` and `queries` are not already on the struct (they may have been added in Phase 3 for MetadataProvider), add them. The `NewModule` constructor should accept `*movies.Service` and `*sql.DB`:

```go
func NewModule(movieSvc *movies.Service, metadataSvc *metadata.Service, db *sql.DB, logger *zerolog.Logger) *Module {
    return &Module{
        descriptor:       &Descriptor{},
        metadataProvider: newMetadataProvider(metadataSvc, movieSvc, logger),
        movieService:     movieSvc,
        queries:          sqlc.New(db),
    }
}
```

**Do the same for** `internal/modules/tv/module.go`:

```go
type Module struct {
    descriptor       *Descriptor
    metadataProvider *metadataProvider // from Phase 3
    tvService        *tvlib.Service
    queries          *sqlc.Queries
}
```

---

**Modify** `internal/api/wire.go` — update the module constructors if their signatures changed:

If `NewModule` now accepts `*sql.DB`, ensure Wire provides it. The `*sql.DB` is already available in the Wire graph (from `database.NewConnection` or `manager.DB()`).

---

**Modify** `internal/api/setters.go` or `internal/api/switchable.go` — add `SetRegistry` calls for availability and calendar services:

After the module registry is constructed and modules are registered, call:

```go
availabilityService.SetRegistry(registry)
calendarService.SetRegistry(registry)
```

This should happen in the `setCircularDependencies` function (or equivalent) in `internal/api/setters.go`, where other late-binding dependencies are wired. Read `setters.go` to find the right place.

**Alternative:** If `SetRegistry` feels wrong for the DI approach, pass `*module.Registry` directly to the `NewService` constructors via Wire. This is cleaner but requires updating the Wire graph. The subagent should evaluate which approach is simpler and matches the existing patterns in `setters.go`.

---

**Regenerate Wire:**

```bash
make wire
```

Verify `internal/api/wire_gen.go` regenerates cleanly.

---

**Add compile-time interface assertions** for all Phase 4 interfaces in both module files. These may already exist from earlier tasks but verify they're all present:

In `internal/modules/movie/module.go`:
```go
var _ module.MonitoringPresets = (*Module)(nil)
var _ module.ReleaseDateResolver = (*Module)(nil)
var _ module.CalendarProvider = (*Module)(nil)
var _ module.WantedCollector = (*Module)(nil)
```

In `internal/modules/tv/module.go`:
```go
var _ module.MonitoringPresets = (*Module)(nil)
var _ module.ReleaseDateResolver = (*Module)(nil)
var _ module.CalendarProvider = (*Module)(nil)
var _ module.WantedCollector = (*Module)(nil)
```

**Verify:**
- `make wire` succeeds
- `go build ./...` compiles (full project)
- `go vet ./...` clean

---

### Task 4.8: Phase 4 Validation

**Run all of these after all Phase 4 tasks complete:**

1. `make build` — full project builds (backend + frontend)
2. `make test` — all Go tests pass
3. `make lint` — no new Go lint issues
4. `cd web && bun run lint` — no new frontend lint issues (no frontend changes in this phase)

**Integration verification:**
- Start the app → both modules register with all Phase 4 interfaces (check logs for startup messages)
- Movie add → status correctly set to `unreleased` or `missing` based on release dates (existing behavior preserved)
- Series add → episodes get correct `unreleased`/`missing` status based on air dates (existing behavior preserved)
- Calendar page → shows movie and episode events (module dispatch path if registry is wired)
- Wanted/Missing page → shows missing movies and episodes (existing queries still work)
- Wait for hourly "status-refresh" task (or trigger manually) → unreleased items with past dates transition to missing (module dispatch path)
- TV monitoring presets → bulk monitor still works: test "all", "future", "first_season", "latest_season", "none"
- Movie bulk monitor → toggle monitored on/off for multiple movies still works

**Specific regression checks:**
- `ListMissingMovies` query still returns correct results (status IN ('missing', 'failed') AND monitored = 1)
- `ListMissingEpisodes` query still enforces 3-level monitoring check (series + season + episode)
- `UpdateUnreleasedMoviesToMissing` still handles the digital→physical→theatrical+90d priority chain
- `UpdateUnreleasedEpisodesToMissing` still transitions based on air_date <= date('now')
- Dev mode toggle → services' `SetDB` calls still work alongside `SetRegistry`
- Calendar API returns events in the same format (CalendarEvent JSON structure unchanged)
- Monitoring stats endpoint (`GET /api/v1/series/:id/monitor/stats`) still works

**Spec deviations documented in this phase:**
1. `MonitoringPresets.ApplyPreset` takes `options map[string]any` parameter (not in spec) to support TV's `includeSpecials` flag
2. `SearchableItem` uses `GetQualityProfileID() int64` and `GetCurrentQualityID() *int64` instead of spec's `GetQualityProfile() *QualityProfile` and `GetCurrentQuality() *QualityItem` (avoids circular dependency)
3. TV `existing` monitoring preset deferred — not present in current codebase
4. `streamingServicesWithEarlyRelease` map is duplicated between `internal/calendar/service.go` and `internal/modules/tv/calendar.go` — consolidation deferred to Phase 10
5. `ComputeAvailabilityDate` for movies returns earliest of digital/physical per spec §7.1, but the existing SQL query `UpdateUnreleasedMoviesToMissing` still uses the priority chain (digital→physical→theatrical+90d) — SQL alignment deferred to Phase 10
6. Generic framework cascade (§6.1: parent `monitored` propagation to descendants) deferred to Phase 10 — existing TV-specific cascade in `tv/seasons.go` continues to work

---

### Phase 4 File Inventory

### New files created in Phase 4
```
internal/module/wanted_item.go (WantedItem concrete SearchableItem implementation)
internal/modules/movie/monitoring.go (Movie MonitoringPresets — empty presets, no-op apply)
internal/modules/movie/release_dates.go (Movie ReleaseDateResolver)
internal/modules/movie/calendar.go (Movie CalendarProvider)
internal/modules/movie/wanted.go (Movie WantedCollector)
internal/modules/tv/monitoring.go (TV MonitoringPresets — 5 presets, delegates to BulkMonitor)
internal/modules/tv/release_dates.go (TV ReleaseDateResolver)
internal/modules/tv/calendar.go (TV CalendarProvider)
internal/modules/tv/wanted.go (TV WantedCollector)
```

### Existing files modified in Phase 4

**Backend — module framework types:**
```
internal/module/parse_types.go (flesh out MonitoringPreset struct fields)
internal/module/calendar_types.go (flesh out CalendarItem struct fields)
internal/module/search_types.go (update SearchableItem interface to use ID-based quality accessors)
internal/module/interfaces.go (update ApplyPreset signature to include options parameter)
```

**Backend — module stubs:**
```
internal/modules/movie/module.go (add movieService, queries fields; add compile-time assertions)
internal/modules/tv/module.go (add tvService, queries fields; add compile-time assertions)
```

**Backend — framework services:**
```
internal/availability/service.go (add registry field, SetRegistry, dispatch to modules)
internal/calendar/service.go (add registry field, SetRegistry, dispatch to modules, calendarItemToEvent converter)
```

**Backend — Wire DI:**
```
internal/api/wire.go (update module constructors if signatures changed)
internal/api/setters.go (add SetRegistry calls for availability and calendar services)
internal/api/wire_gen.go (regenerated via make wire)
```

---

