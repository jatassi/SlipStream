## Phase 9: Arr Import, Dev Mode & Health

**Goal:** Refactor the arr import pipeline to use module-provided adapters, extract dev mode mock data creation into module `MockFactory` implementations, and document the health monitoring system's module usage patterns (already generic — no code changes needed for health).

**Spec sections covered:** §12 (ArrImportAdapter interface, framework orchestration with module adapters), §14 (MockFactory per module, dev mode integration), §15 (health monitoring — already generic)

**Spec items deferred from Phase 9:**
- §14.1 `CreateMockMetadataProvider()` on `MockFactory` — deferred. The current dev mode already switches to mock TMDB/TVDB/OMDB clients via `metadata.Service.SetClients()`. The spec envisions each module providing its own mock metadata provider, but the current architecture has a single `metadata.Service` that serves both movie and TV. Splitting metadata into per-module providers happens in Phase 3; until then, the existing `switchMetadataClients` mechanism continues to work. `MockFactory` in Phase 9 covers `CreateSampleLibraryData` and `CreateTestRootFolders` only. The `CreateMockMetadataProvider` method signature exists on the interface (defined in Phase 0) but returns `nil` — the framework ignores nil returns.
- §12 `ArrImportAdapter.ReadExternalDB(ctx, dbPath) ([]ArrImportItem, error)` with generic `ArrImportItem` — deferred. The spec envisions a single generic item type, but the current Radarr/Sonarr import pipelines have deeply divergent schemas (movies have files with paths; series have episodes, seasons, and episode files with relative paths). Forcing both into `[]ArrImportItem` would require a complex union type or `any` assertions, losing type safety for no gain with only two modules. Phase 9 instead uses **module-specific adapter interfaces** that the framework dispatches to, preserving the current type-safe `SourceMovie`/`SourceSeries` types. The generic `ArrImportAdapter` interface from Phase 0 remains defined but is not implemented by movie/TV modules — they implement richer, type-safe adapter interfaces instead.

**Depends on:** Phase 0 (module types, interfaces, registry), Phase 7 (notification event toggles JSON — arr import config creates notifications with new event format). Phase 2 is NOT a dependency — arr import quality profile mapping translates external *arr quality IDs to SlipStream's global quality IDs via `MapQualityID`, which is independent of module-scoped quality profiles.

### Phase 9 Context Management

**Parallelism map** (tasks with no dependency arrow can run concurrently):

```
Group A (parallel):  9.1, 9.5, 9.9
Group B (after A):   9.2 (needs 9.1), 9.6 (needs 9.5)
Group C (after B):   9.3 (needs 9.2), 9.7 (needs 9.6)
Group D (after C):   9.4 (needs 9.3), 9.8 (needs 9.7)
Group E (after D):   9.10 (needs 9.4, 9.8)
Group F:             9.11 (validation — needs all)
```

**Delegation notes:**
- Tasks 9.1 + 9.2 (arr import adapter interfaces + movie adapter) should be one subagent — the movie adapter implements the interface defined in 9.1.
- Tasks 9.3 + 9.4 (TV adapter + framework orchestrator rewrite) can be two parallel subagents if Group C is split, but 9.4 depends on both adapters existing. Safer as sequential in one subagent.
- Tasks 9.5 + 9.6 + 9.7 (MockFactory interface refinement + movie mock + TV mock) should be one subagent — they all touch the dev mode pipeline.
- Task 9.8 (DevModeManager refactor) is the largest task — dedicate a single subagent.
- Task 9.9 (health documentation) is pure documentation — can run in background from Group A.
- Task 9.10 (frontend updates) — dedicate a single subagent.

**Validation checkpoints:** After Groups B and D, run `go build ./...` to catch compile errors early. Full validation in 9.11.

---

### Phase 9 Architecture Decisions

1. **Module-specific adapter interfaces instead of generic `ArrImportAdapter`.** The spec (§12) defines a single `ArrImportAdapter` with `ReadExternalDB(ctx, dbPath) ([]ArrImportItem, error)` and `ConvertToCreateInput(item ArrImportItem) (any, error)`. This is too generic for the current two modules — Radarr returns movies with files; Sonarr returns series with seasons, episodes, and episode files. A generic `ArrImportItem` would be a lowest-common-denominator union type.

   Instead, Phase 9 defines **two framework-level adapter interfaces** in `internal/module/arr_import.go`:

   ```
   MovieArrImportAdapter interface {
       ExternalAppName() string
       PreviewMovies(ctx, reader) ([]ArrImportPreviewItem, error)
       ImportMovie(ctx, movie ArrSourceMovie, mappings) (*ImportedEntity, error)
   }

   TVArrImportAdapter interface {
       ExternalAppName() string
       PreviewSeries(ctx, reader) ([]ArrImportPreviewItem, error)
       ImportSeries(ctx, series ArrSourceSeries, reader, mappings) (*ImportedEntity, error)
   }
   ```

   The framework reads all entities once and passes each source entity directly to the adapter — adapters never re-read from the external DB. Movie adapters receive `ArrSourceMovie` (which already includes the file); TV adapters additionally receive the `ArrReader` because they need to read episodes and episode files per series. The `Registry` provides `GetMovieArrAdapter()` and `GetTVArrAdapter()` lookups. The arr import `Service` dispatches to the correct adapter based on `SourceType`. This preserves full type safety and matches the existing code structure. The generic `ArrImportAdapter` from Phase 0 remains defined but unimplemented — it will be useful when a 3rd module (e.g., Lidarr → Music) is added with its own schema.

   **Return convention for `ImportMovie`/`ImportSeries`:** Non-nil entity with nil error = created successfully. Nil entity with nil error = skipped (duplicate, no external ID, etc.). Nil entity with non-nil error = import failed. The framework uses this to update `ImportReport` counters (MoviesCreated/Skipped/Errored) without needing to know module-specific skip reasons.

   **Key insight:** The current `arrimport.Service` directly imports `movies.Service` and `tv.Service`. After Phase 9, it imports zero module-specific packages — all module interaction flows through the adapter interfaces.

2. **Reader stays framework-owned.** The `Reader` interface (`arrimport/reader.go`) and its implementations (`reader_sqlite.go`, `reader_api.go`) stay in the `arrimport` package. They handle the mechanics of connecting to and reading from external *arr databases/APIs. The `SourceMovie`, `SourceSeries`, etc. types also stay in `arrimport` — they represent the external *arr schema, not SlipStream's internal model. Module adapters receive these types and convert them to SlipStream entities.

3. **Config import stays framework-owned.** The config import pipeline (download clients, indexers, notifications, quality profiles, naming config) is media-type-agnostic — it imports infrastructure settings, not media. It stays entirely in the `arrimport` package. The only Phase 9 change is that notification config import must use the Phase 7 event toggles JSON format instead of individual boolean fields (if Phase 7 has been completed). If Phase 7 hasn't been completed yet, the notification import code stays as-is and Phase 7 updates it.

4. **Metadata refresh stays on module adapters.** The current `Executor` calls `metadataRefresher.RefreshMovieMetadata` and `RefreshSeriesMetadata`. After Phase 9, each module's adapter handles its own metadata refresh internally. The `MetadataRefresher` interface on `arrimport.Service` is removed — adapters receive whatever services they need via their constructor.

5. **`MockFactory` scope in Phase 9.** Each module implements two of the three `MockFactory` methods:
   - `CreateSampleLibraryData(ctx)` — creates mock media entities (movies/series) with artwork and files
   - `CreateTestRootFolders(ctx)` — creates mock root folders with virtual filesystem entries
   - `CreateMockMetadataProvider()` — returns `nil` in Phase 9 (see deferred items above)

   The `DevModeManager` calls these methods on each enabled module instead of containing the movie/TV creation logic directly. This moves ~400 lines of media-specific code out of `devmode.go` into module packages.

6. **`DevModeManager` keeps framework responsibilities.** After Phase 9, `DevModeManager.OnToggle` still handles:
   - Database switching (`dbManager.SetDevMode`)
   - Copying JWT secret, settings, quality profiles, portal users to dev DB
   - Switching metadata clients (until Phase 3 changes this)
   - Creating mock indexer, download client, notification
   - Configuring dev mode slots

   What moves to modules:
   - `switchRootFolders` → calls each module's `CreateTestRootFolders`
   - `populateMockMedia` (+ `populateMockMovies`, `populateMockSeries`, all helper methods) → calls each module's `CreateSampleLibraryData`

7. **MockFactory receives a `MockContext` struct.** Rather than passing individual services, `CreateSampleLibraryData` and `CreateTestRootFolders` receive a `MockContext` struct containing the additional context the factory needs beyond what the module struct already provides:

   ```
   type MockContext struct {
       DB                *sql.DB
       Logger            *zerolog.Logger
       RootFolderCreator MockRootFolderCreator
       QualityProfiles   []MockQualityProfile
       DefaultProfileID  int64
   }
   ```

   Modules get metadata service and artwork downloader from their own struct fields (set in earlier phases). `MockContext` provides only the services that aren't already available on the module: the dev DB connection, logger, root folder creation, and quality profile info. This keeps the interface minimal.

8. **Health monitoring — no code changes.** The spec (§15) confirms the health system is already generic. The `HealthService` contract interface uses string-based categories and IDs. Modules can call `RegisterItemStr`, `SetErrorStr`, `ClearStatusStr`, `SetWarningStr` with any category/ID strings. Phase 9 documents the patterns for future module authors and adds no code changes to the health system.

9. **Slot initialization stays in adapters.** The current `Executor` calls `slotsService.InitializeSlotAssignments` and `assignSlot` for imported media. After Phase 9, the movie adapter handles slot init for movies (mediaType `"movie"`), and the TV adapter handles slot init for episodes (mediaType `"episode"`, NOT `"tv"` or `"series"`). The `SlotsService` interface is passed to adapters via their constructor (or via the `ImportContext` — see task 9.1).

10. **Progress tracking stays framework-owned.** The `progress.Manager` calls (`StartActivity`, `UpdateActivity`, `CompleteActivity`, `FailActivity`) remain in the framework's `Executor.Run` method. Module adapters are called per-entity and don't manage progress directly — the framework updates progress between adapter calls.

---

### Task 9.1: Define Arr Import Adapter Interfaces in Module Framework ✅

**Create** `internal/module/arr_import.go`

Define the module-specific arr import adapter interfaces and supporting types:

```go
package module

import (
    "context"

    "github.com/slipstream/slipstream/internal/indexer/scanner"
    "github.com/slipstream/slipstream/internal/library/slots"
)

// ArrImportSlotsService is the subset of slot operations needed during arr import.
// Matches the existing arrimport.SlotsService interface exactly.
type ArrImportSlotsService interface {
    IsMultiVersionEnabled(ctx context.Context) bool
    InitializeSlotAssignments(ctx context.Context, mediaType string, mediaID int64) error
    DetermineTargetSlot(ctx context.Context, parsed *scanner.ParsedMedia, mediaType string, mediaID int64) (*slots.SlotAssignment, error)
    AssignFileToSlot(ctx context.Context, mediaType string, mediaID, slotID, fileID int64) error
}

// ArrImportMappings holds user-provided mapping choices (root folders, quality profiles, selected items).
type ArrImportMappings struct {
    RootFolderMapping     map[string]int64 // source root folder path → SlipStream root folder ID
    QualityProfileMapping map[int64]int64  // source quality profile ID → SlipStream profile ID
    SelectedIDs           []int            // external IDs of selected items (TMDB IDs for movies, TVDB IDs for series)
}

// ImportedEntity is the result of importing a single entity from an external *arr app.
// Return convention: non-nil entity + nil error = created; nil + nil = skipped; nil + error = failed.
type ImportedEntity struct {
    EntityType    EntityType
    EntityID      int64
    Title         string
    FilesImported int
    Errors        []string // per-file errors (non-fatal); entity was still created
}

// ArrImportPreviewItem represents a single item in the import preview.
// Contains a superset of fields needed by both MoviePreview and SeriesPreview API types.
// The framework converts these to the appropriate API response types.
type ArrImportPreviewItem struct {
    Title            string `json:"title"`
    Year             int    `json:"year"`
    TmdbID           int    `json:"tmdbId"`                     // TMDB ID (movies + series)
    TvdbID           int    `json:"tvdbId,omitempty"`            // TVDB ID (series only)
    HasFile          bool   `json:"hasFile"`                     // movies only
    Quality          string `json:"quality,omitempty"`            // movies only (from file's QualityName)
    EpisodeCount     int    `json:"episodeCount,omitempty"`      // series only
    FileCount        int    `json:"fileCount"`                   // series: episode files; movies: 0 or 1
    QualityProfileID int64  `json:"qualityProfileId"`            // source quality profile ID
    Monitored        bool   `json:"monitored"`
    Status           string `json:"status"`                      // "new", "duplicate", "skip"
    SkipReason       string `json:"skipReason,omitempty"`
    PosterURL        string `json:"posterUrl,omitempty"`
}

// MovieArrImportAdapter enables import of movies from external *arr apps (Radarr). (§12)
// Implemented by the movie module as an optional interface.
type MovieArrImportAdapter interface {
    // ExternalAppName returns the name of the external app (e.g., "Radarr").
    ExternalAppName() string
    // PreviewMovies generates a preview of movies that would be imported.
    // The reader provides access to the external database.
    PreviewMovies(ctx context.Context, reader ArrReader) ([]ArrImportPreviewItem, error)
    // ImportMovie imports a single movie from the external source.
    // The framework reads all movies once and passes each source movie directly.
    // Returns nil, nil if the movie should be skipped (duplicate, no TMDB ID, etc.).
    ImportMovie(ctx context.Context, movie ArrSourceMovie, mappings ArrImportMappings) (*ImportedEntity, error)
}

// TVArrImportAdapter enables import of TV series from external *arr apps (Sonarr). (§12)
// Implemented by the TV module as an optional interface.
type TVArrImportAdapter interface {
    // ExternalAppName returns the name of the external app (e.g., "Sonarr").
    ExternalAppName() string
    // PreviewSeries generates a preview of series that would be imported.
    PreviewSeries(ctx context.Context, reader ArrReader) ([]ArrImportPreviewItem, error)
    // ImportSeries imports a single series from the external source.
    // The framework reads all series once and passes each source series directly.
    // The reader is also provided because the adapter needs to read episodes and
    // episode files per series (not included in ArrSourceSeries).
    // Returns nil, nil if the series should be skipped (duplicate, no TVDB ID, etc.).
    ImportSeries(ctx context.Context, series ArrSourceSeries, reader ArrReader, mappings ArrImportMappings) (*ImportedEntity, error)
}

// ArrReader abstracts reading data from an external *arr database or API.
// This is implemented by the framework (arrimport package) and passed to module adapters.
// Note: ReadRootFolders and ReadQualityProfiles are included for completeness but are
// primarily used by the framework (GetSourceRootFolders/GetSourceQualityProfiles endpoints).
// Module adapters mainly use ReadMovies/ReadSeries/ReadEpisodes/ReadEpisodeFiles.
type ArrReader interface {
    ReadRootFolders(ctx context.Context) ([]ArrSourceRootFolder, error)
    ReadQualityProfiles(ctx context.Context) ([]ArrSourceQualityProfile, error)
    ReadMovies(ctx context.Context) ([]ArrSourceMovie, error)
    ReadSeries(ctx context.Context) ([]ArrSourceSeries, error)
    ReadEpisodes(ctx context.Context, seriesID int64) ([]ArrSourceEpisode, error)
    ReadEpisodeFiles(ctx context.Context, seriesID int64) ([]ArrSourceEpisodeFile, error)
}
```

**Note on `ArrReader`:** This interface mirrors the current `arrimport.Reader` but uses types defined in the `module` package (prefixed with `Arr` to avoid collisions). The `arrimport` package's `Reader` implementation will satisfy this interface via thin wrapper methods. The config-import methods (`ReadDownloadClients`, `ReadIndexers`, `ReadNotifications`, `ReadQualityProfilesFull`, `ReadNamingConfig`) are NOT on `ArrReader` — they remain framework-internal since config import is not module-dispatched.

**Create** `internal/module/arr_source_types.go`

Define the source types that mirror existing `arrimport.Source*` types. These are needed so module adapters don't import the `arrimport` package:

```go
package module

import "time"

// ArrSourceRootFolder represents a root folder from an external *arr app.
type ArrSourceRootFolder struct {
    ID   int64
    Path string
}

// ArrSourceQualityProfile represents a quality profile from an external *arr app.
type ArrSourceQualityProfile struct {
    ID    int64
    Name  string
    InUse bool
}

// ArrSourceMovie represents a movie from an external *arr app (Radarr).
type ArrSourceMovie struct {
    ID               int64
    Title            string
    SortTitle        string
    Year             int
    TmdbID           int
    ImdbID           string
    Overview         string
    Runtime          int
    Path             string
    RootFolderPath   string
    QualityProfileID int64
    Monitored        bool
    Status           string
    InCinemas        time.Time
    PhysicalRelease  time.Time
    DigitalRelease   time.Time
    Studio           string
    Certification    string
    Added            time.Time
    HasFile          bool
    PosterURL        string
    File             *ArrSourceMovieFile
}

// ArrSourceMovieFile represents a movie file from an external *arr app.
type ArrSourceMovieFile struct {
    ID               int64
    Path             string
    Size             int64
    QualityID        int
    QualityName      string
    VideoCodec       string
    AudioCodec       string
    Resolution       string
    AudioChannels    string
    DynamicRange     string
    OriginalFilePath string
    DateAdded        time.Time
}

// ArrSourceSeries represents a TV series from an external *arr app (Sonarr).
type ArrSourceSeries struct {
    ID               int64
    Title            string
    SortTitle        string
    Year             int
    TvdbID           int
    TmdbID           int
    ImdbID           string
    Overview         string
    Runtime          int
    Path             string
    RootFolderPath   string
    QualityProfileID int64
    Monitored        bool
    SeasonFolder     bool
    Status           string
    Network          string
    SeriesType       string
    Certification    string
    Added            time.Time
    PosterURL        string
    Seasons          []ArrSourceSeason
}

// ArrSourceSeason represents a season from an external *arr app.
type ArrSourceSeason struct {
    SeasonNumber int
    Monitored    bool
}

// ArrSourceEpisode represents an episode from an external *arr app.
type ArrSourceEpisode struct {
    ID            int64
    SeriesID      int64
    SeasonNumber  int
    EpisodeNumber int
    Title         string
    Overview      string
    AirDateUtc    string
    Monitored     bool
    EpisodeFileID int64
    HasFile       bool
}

// ArrSourceEpisodeFile represents an episode file from an external *arr app.
type ArrSourceEpisodeFile struct {
    ID               int64
    SeriesID         int64
    SeasonNumber     int
    RelativePath     string
    Size             int64
    QualityID        int
    QualityName      string
    VideoCodec       string
    AudioCodec       string
    Resolution       string
    AudioChannels    string
    DynamicRange     string
    OriginalFilePath string
    DateAdded        time.Time
}
```

**Modify** `internal/module/registry.go` — add lookup methods for arr import adapters:

```go
// GetMovieArrAdapter returns the MovieArrImportAdapter from the registered modules, or nil if none.
func (r *Registry) GetMovieArrAdapter() MovieArrImportAdapter {
    r.mu.RLock()
    defer r.mu.RUnlock()
    for _, mod := range r.modules {
        if adapter, ok := mod.(MovieArrImportAdapter); ok {
            return adapter
        }
    }
    return nil
}

// GetTVArrAdapter returns the TVArrImportAdapter from the registered modules, or nil if none.
func (r *Registry) GetTVArrAdapter() TVArrImportAdapter {
    r.mu.RLock()
    defer r.mu.RUnlock()
    for _, mod := range r.modules {
        if adapter, ok := mod.(TVArrImportAdapter); ok {
            return adapter
        }
    }
    return nil
}
```

**Modify** `internal/module/optional_interfaces.go` — update the comment to list `MovieArrImportAdapter` and `TVArrImportAdapter` as additional optional interfaces alongside the existing `ArrImportAdapter`.

**Verify:**
- `go build ./internal/module/...` compiles
- No existing tests broken

---

### Task 9.2: Implement MovieArrImportAdapter on Movie Module ✅

**Modify** `internal/modules/movie/module.go` — add `MovieArrImportAdapter` implementation.

The movie module needs access to its `MovieService` (for `Create`, `GetByTmdbID`, `AddFile`) and supporting services. These should already be available on the module struct from earlier phases.

**Create** `internal/modules/movie/arr_import.go`:

```go
package movie

// Compile-time assertion
var _ module.MovieArrImportAdapter = (*Module)(nil)

func (m *Module) ExternalAppName() string { return "Radarr" }
```

`PreviewMovies` logic — port from `arrimport.Service.previewMovies`:
- Read movies from the `ArrReader`
- For each movie: check if `TmdbID == 0` (skip), check if exists via `movieService.GetByTmdbID` (duplicate vs new)
- Return `[]ArrImportPreviewItem` with status, poster URL, quality info
- Populate all preview fields: `TmdbID` (from source movie), `HasFile`, `Quality` (from file's QualityName if file exists), `QualityProfileID` (source profile ID), `Monitored`, `PosterURL`

`ImportMovie` logic — port from `arrimport.Executor.importMovie`:
- Receive the `ArrSourceMovie` directly from the framework (framework reads all movies once and passes each one)
- Return `nil, nil` if movie should be skipped (TmdbID == 0, already exists via `GetByTmdbID`)
- Resolve root folder and quality profile mappings
- Build `CreateMovieInput`, call `movieService.Create`
- Preserve `added_at` timestamp via raw SQL (same as current `preserveMovieAddedAt`)
- Initialize slot assignments if multi-version enabled
- Trigger metadata refresh
- Import movie file if present (quality ID mapping via `MapQualityID`, file stat check, `AddFile`, slot assignment)
- Return `*ImportedEntity` with entity details and any per-file errors

**Key imports moved INTO module adapter:**
- `movies.CreateMovieInput`, `movies.CreateMovieFileInput` — the module owns these types
- `arrimport.MapQualityID` — the quality mapping function stays in `arrimport` (shared utility)
- `pathutil.NormalizePath` — shared utility
- `scanner.ParsePath` — shared utility for slot assignment

**Services the adapter needs** (received via constructor or module struct fields set in earlier phases):
- `movieService MovieService` (same interface as current `arrimport.MovieService`)
- `slotsService` (for slot init/assignment)
- `metadataRefresher` (for post-import metadata fetch, or direct metadata service access)
- `db *sql.DB` (for raw `UPDATE movies SET added_at` query)
- `logger *zerolog.Logger`

**Verify:**
- `go build ./internal/modules/movie/...` compiles
- The adapter methods have the correct signatures matching `module.MovieArrImportAdapter`

---

### Task 9.3: Implement TVArrImportAdapter on TV Module ✅

**Modify** `internal/modules/tv/module.go` — add `TVArrImportAdapter` implementation.

**Create** `internal/modules/tv/arr_import.go`:

```go
package tv

var _ module.TVArrImportAdapter = (*Module)(nil)

func (m *Module) ExternalAppName() string { return "Sonarr" }
```

`PreviewSeries` logic — port from `arrimport.Service.previewSeries`:
- Read series from the `ArrReader`
- For each series: check `TvdbID == 0` (skip), read episodes + episode files for counts, check existence via `tvService.GetSeriesByTvdbID`
- Return `[]ArrImportPreviewItem`
- Populate all preview fields: `TmdbID`, `TvdbID` (both from source series), `EpisodeCount`, `FileCount`, `QualityProfileID` (source profile ID), `Monitored`, `PosterURL`

`ImportSeries` logic — port from `arrimport.Executor.importSeries` + `importAllSeries`:
- Receive the `ArrSourceSeries` directly from the framework plus the `ArrReader` (needed for reading episodes/files)
- Return `nil, nil` if series should be skipped (TvdbID == 0, already exists via `GetSeriesByTvdbID`)
- Read episodes and episode files for this series via reader
- Build `fileIDToEpisode` lookup, `buildSeasonInputs` (port the helper functions)
- Build `CreateSeriesInput`, call `tvService.CreateSeries`
  - Map `series.SeriesType` → `FormatType` field on CreateSeriesInput
  - Map `series.Status` → `ProductionStatus` field on CreateSeriesInput (required string field)
- Preserve `added_at` timestamp
- Trigger metadata refresh
- Import episode files: for each file, find matching episode via `tvService.GetEpisodeByNumber`, stat the file, map quality ID, create episode file record, init slots (mediaType `"episode"`, NOT `"series"`), assign slots
- Return `*ImportedEntity`

**Helper functions to port into module package:**
- `buildFileEpisodeLookup` → `internal/modules/tv/arr_import.go`
- `buildSeasonInputs` → `internal/modules/tv/arr_import.go`
- `parseAirDate` → `internal/modules/tv/arr_import.go`
- `formatDate` → shared utility or duplicated (it's 4 lines)
- `resolveFilePath` → port from current `arrimport` package

**Services the adapter needs:**
- `tvService TVService` (same interface as current `arrimport.TVService`)
- `slotsService`
- `metadataRefresher`
- `db *sql.DB`
- `logger *zerolog.Logger`

**Verify:**
- `go build ./internal/modules/tv/...` compiles
- The adapter methods match `module.TVArrImportAdapter`

---

### Task 9.4: Refactor Arr Import Service to Use Module Adapters ✅

**Modify** `internal/arrimport/service.go`:

1. **Remove direct movie/TV service dependencies.** Remove `MovieService`, `TVService`, and `MetadataRefresher` interfaces and fields from `Service`. Remove `movieService`, `tvService`, `metadataRefresher` constructor params.

2. **Add module registry dependency.** Add a `registry ModuleRegistry` field where `ModuleRegistry` is a local interface:

   ```go
   type ModuleRegistry interface {
       GetMovieArrAdapter() module.MovieArrImportAdapter
       GetTVArrAdapter() module.TVArrImportAdapter
   }
   ```

3. **Add `ArrReader` wrapper.** Create a thin wrapper that adapts the existing `Reader` to `module.ArrReader`. The wrapper converts `arrimport.Source*` types to `module.ArrSource*` types:

   ```go
   type readerAdapter struct { inner Reader }

   func (a *readerAdapter) ReadMovies(ctx context.Context) ([]module.ArrSourceMovie, error) {
       movies, err := a.inner.ReadMovies(ctx)
       if err != nil { return nil, err }
       result := make([]module.ArrSourceMovie, len(movies))
       for i, m := range movies {
           result[i] = convertSourceMovie(m)
       }
       return result, nil
   }
   // ... same for ReadSeries, ReadEpisodes, ReadEpisodeFiles, ReadRootFolders, ReadQualityProfiles
   ```

4. **Refactor `Preview`** — dispatch to module adapters:

   ```go
   func (s *Service) Preview(ctx context.Context, mappings ImportMappings) (*ImportPreview, error) {
       // ... reader/lock as before ...
       adapted := &readerAdapter{inner: reader}
       preview := &ImportPreview{Movies: []MoviePreview{}, Series: []SeriesPreview{}, Summary: ImportSummary{}}

       if sourceType == SourceTypeRadarr {
           adapter := s.registry.GetMovieArrAdapter()
           if adapter == nil { return nil, fmt.Errorf("movie module not available") }
           items, err := adapter.PreviewMovies(ctx, adapted)
           // Convert []ArrImportPreviewItem to []MoviePreview + update Summary
       }
       if sourceType == SourceTypeSonarr {
           adapter := s.registry.GetTVArrAdapter()
           if adapter == nil { return nil, fmt.Errorf("TV module not available") }
           items, err := adapter.PreviewSeries(ctx, adapted)
           // Convert []ArrImportPreviewItem to []SeriesPreview + update Summary
       }
       return preview, nil
   }
   ```

5. **Refactor `Executor`** — remove movie/TV service deps, use module adapters:

   **Modify** `internal/arrimport/executor.go`:
   - Remove `movieService`, `tvService`, `metadataRefresher` fields and constructor params
   - Add `registry ModuleRegistry` field
   - `importMovies`: get `MovieArrImportAdapter` from registry, read all movies from reader via `readerAdapter` once, then iterate. For each source movie, call `adapter.ImportMovie(ctx, movie, moduleMappings)`. The framework converts `ImportMappings` → `ArrImportMappings` before the loop: set `SelectedIDs` from `mappings.SelectedMovieTmdbIDs`. Use the return convention: non-nil entity → `MoviesCreated++`, `report.FilesImported += entity.FilesImported`, append entity.Errors; nil+nil → `MoviesSkipped++`; nil+error → `MoviesErrored++`, append error. Framework handles progress tracking between calls.
   - `importAllSeries`: same pattern with `TVArrImportAdapter` — read all series once, iterate, call `adapter.ImportSeries(ctx, series, reader, moduleMappings)` for each. Set `SelectedIDs` from `mappings.SelectedSeriesTvdbIDs`.
   - Remove ALL helper methods that are now in module adapters: `importMovie`, `movieExists`, `buildMovieInput`, `preserveMovieAddedAt`, `initMovieSlots`, `importMovieFile`, `assignSlot`, `importSeries`, `seriesExists`, `readSeriesData`, `buildSeriesInput`, `preserveSeriesAddedAt`, `importEpisodeFiles`, `importSingleEpisodeFile`, `refreshMovieMetadata`, `refreshSeriesMetadata`
   - Keep `resolveMappings` — export as `ResolveMappings` so module adapters can call it (or move to `module` package as it's generic)
   - Keep `formatDate`, `parseAirDate` as shared utilities — export them for use by TV adapter

6. **Update `NewService` constructor** — replace `movieService`/`tvService` params with `registry ModuleRegistry`.

7. **Update `NewExecutor` constructor** — replace `movieService`/`tvService`/`metadataRefresher` params with `registry ModuleRegistry`.

**Modify** `internal/arrimport/types.go`:
- `ImportMappings`: keep as-is (the framework type). The framework converts to `module.ArrImportMappings` before calling adapters:
  - `RootFolderMapping` and `QualityProfileMapping` copy directly
  - For movies: `SelectedIDs` = `mappings.SelectedMovieTmdbIDs`
  - For series: `SelectedIDs` = `mappings.SelectedSeriesTvdbIDs`
- `ImportPreview`, `MoviePreview`, `SeriesPreview`, `ImportSummary`, `ImportReport`: keep as-is (API response types owned by framework).
- Add a `convertToModuleMappings(mappings ImportMappings, sourceType SourceType) module.ArrImportMappings` helper.

**Modify** `internal/api/wire.go`:
- Update `arrimport.NewService` provider call: remove `movies.Service` and `tv.Service` params, add `*module.Registry` param.
- Remove the interface bindings that are no longer needed (lines 212-213):
  - `wire.Bind(new(arrimport.MovieService), new(*movies.Service))` — remove
  - `wire.Bind(new(arrimport.TVService), new(*tv.Service))` — remove
  - Keep `arrimport.RootFolderService`, `arrimport.QualityService`, `arrimport.SlotsService` bindings (still used by framework for preview/config)
- No `MetadataRefresher` binding exists in wire.go (it's set via `setters.go`) — handled in setters.go changes above.

**Modify** `internal/api/setters.go`:
- Remove the `s.automation.ArrImport.SetMetadataRefresher(s.library.LibraryManager)` call (line 36). Each module's adapter handles metadata refresh internally via its own service references.

**Verify:**
- `go build ./...` compiles
- Arr import API endpoints still register correctly
- No compile errors from removed dependencies

---

### Task 9.5: Refine MockFactory Interface and Define MockContext ✅

**Modify** `internal/module/interfaces.go` — update `MockFactory` to use a `MockContext` parameter:

```go
// MockFactory creates mock data for dev mode. (§14.1)
type MockFactory interface {
    // CreateMockMetadataProvider returns a mock metadata provider for this module.
    // May return nil if the module reuses a shared mock provider (current movie/TV behavior).
    CreateMockMetadataProvider() MetadataProvider
    // CreateSampleLibraryData creates sample media entities in the dev database.
    CreateSampleLibraryData(ctx context.Context, mctx *MockContext) error
    // CreateTestRootFolders creates mock root folders with virtual filesystem entries.
    CreateTestRootFolders(ctx context.Context, mctx *MockContext) error
}
```

**Create** `internal/module/mock_types.go`:

```go
package module

import (
    "context"
    "database/sql"

    "github.com/rs/zerolog"
)

// MockContext provides services and state to MockFactory methods.
type MockContext struct {
    DB     *sql.DB
    Logger *zerolog.Logger

    // Services available to mock factories
    RootFolderCreator MockRootFolderCreator
    QualityProfiles   []MockQualityProfile
    DefaultProfileID  int64
}

// MockRootFolderCreator creates root folders during dev mode setup.
type MockRootFolderCreator interface {
    Create(ctx context.Context, path, name, mediaType string) (rootFolderID int64, err error)
}

// MockQualityProfile is a minimal quality profile representation for mock factories.
type MockQualityProfile struct {
    ID   int64
    Name string
}
```

Note: No `MockMetadataService` or `MockArtworkDownloader` interfaces are needed here — modules access metadata and artwork services through their own struct fields (set in earlier phases via constructor injection).

**Note:** `MockContext` deliberately uses zerolog directly (breaking the "no internal deps" rule for the `module` package) because `MockFactory` is only called during dev mode — it's not a hot path and the ergonomic benefit of structured logging outweighs the purity. This matches the existing Phase 0 decision that `register.go` already imports zerolog.

**Verify:**
- `go build ./internal/module/...` compiles
- `MockFactory` interface signature is updated

---

### Task 9.6: Implement MockFactory on Movie Module ✅

**Create** `internal/modules/movie/mock_factory.go`:

```go
package movie

var _ module.MockFactory = (*Module)(nil)
```

`CreateMockMetadataProvider()` — returns `nil` (Phase 9 scope — see deferred items).

`CreateTestRootFolders(ctx, mctx)` — port from `DevModeManager.createMockMovieFolder`:
- **Check for existing mock folder first:** list root folders via the module's root folder service, check if any have `Path == fsmock.MockMoviesPath`. Skip creation if already exists (port from `checkExistingMockFolders` logic).
- Use `mctx.RootFolderCreator.Create(ctx, fsmock.MockMoviesPath, "Mock Movies", "movie")`
- Initialize the virtual filesystem with mock movie directories and files
- Port the VFS setup from `fsmock.GetInstance().initialize()` for the movies subtree (the `initialize` method in `virtualfs.go` creates the full tree — the module should only create its portion)

`CreateSampleLibraryData(ctx, mctx)` — port from `DevModeManager.populateMockMovies`:
- **Check for existing data first:** call `movieService.List(ctx, movies.ListMoviesOptions{})` — if movies already exist, skip population (port from `populateMockMedia` existing-check logic). This prevents duplicate mock data when dev mode is toggled multiple times.
- Define the mock movie TMDB IDs list: `{603, 27205, 438631, 680, 550, 693134, 872585, 346698}`
- Look up the movie root folder ID by listing root folders and matching `fsmock.MockMoviesPath`
- For each: fetch metadata from metadata service, build `CreateMovieInput`, create movie, download artwork, create mock files from VFS
- Port `createMockMovieFiles` helper
- Port `parseQualityFromFilename` utility (or import from a shared location — see Task 9.8)

**Module needs access to:**
- `movieService` (for `Create`, `AddFile`, `List`)
- Metadata service (for `GetMovie`) — in Phase 9, accessed via the module's metadata reference
- Artwork downloader — for downloading mock poster/backdrop images
- `fsmock` package — for virtual filesystem access
- `quality.GetQualityByName` — for mapping quality names to IDs

**Verify:**
- `go build ./internal/modules/movie/...` compiles
- Compile-time assertion passes

---

### Task 9.7: Implement MockFactory on TV Module ✅

**Create** `internal/modules/tv/mock_factory.go`:

```go
package tv

var _ module.MockFactory = (*Module)(nil)
```

`CreateMockMetadataProvider()` — returns `nil`.

`CreateTestRootFolders(ctx, mctx)` — port from `DevModeManager.createMockTVFolder`:
- **Check for existing mock folder first:** list root folders, check if any have `Path == fsmock.MockTVPath`. Skip creation if already exists.
- Use `mctx.RootFolderCreator.Create(ctx, fsmock.MockTVPath, "Mock TV", "tv")`
- Initialize VFS with mock TV directories

`CreateSampleLibraryData(ctx, mctx)` — port from `DevModeManager.populateMockSeries`:
- **Check for existing data first:** call `tvService.ListSeries(ctx)` or equivalent — if series already exist, skip population.
- Define mock series TVDB IDs: `{81189, 121361, 305288, 361753, 355567}`
- Look up the TV root folder ID by listing root folders and matching `fsmock.MockTVPath`
- For each: port `createOneMockSeries` logic — fetch series metadata, fetch seasons, build season inputs, create series (include all fields from current code: `NetworkLogoURL`, `SeasonFolder: true`, etc.), download artwork, create episode files from VFS
- Port all helper methods: `convertSeasonsMetadata`, `convertEpisodesMetadata`, `createMockEpisodeFiles`, `buildEpisodeMap`, `processSeasonDirectory`, `processEpisodeFile`
- Port parsing helpers: `parseSeasonNumber`, `parseEpisodeNumber`, `isEpisodeMarker`, `extractEpisodeDigits`

**Module needs access to:**
- `tvService` (for `CreateSeries`, `ListEpisodes`)
- Metadata service (for `GetSeriesByTVDB`, `GetSeriesSeasons`)
- Artwork downloader
- `fsmock` package
- `quality.GetQualityByName`
- `sqlc.Queries` (for direct `CreateEpisodeFile`, `UpdateEpisodeStatusWithDetails` — same as current code)

**Verify:**
- `go build ./internal/modules/tv/...` compiles
- Compile-time assertion passes

---

### Task 9.8: Refactor DevModeManager to Use Module MockFactory ✅

**Modify** `internal/api/devmode.go`:

1. **Add module registry dependency.** Add `registry *module.Registry` field to `DevModeManager`. Update `NewDevModeManager` constructor to accept `*module.Registry`.

2. **Create `MockRootFolderCreator` adapter.** Implement the `module.MockRootFolderCreator` interface using the existing `rootfolder.Service`:

   ```go
   type mockRootFolderAdapter struct {
       service *rootfolder.Service
   }

   func (a *mockRootFolderAdapter) Create(ctx context.Context, path, name, mediaType string) (int64, error) {
       rf, err := a.service.Create(ctx, rootfolder.CreateRootFolderInput{
           Path: path, Name: name, MediaType: mediaType,
       })
       if err != nil { return 0, err }
       return rf.ID, nil
   }
   ```

3. **Replace `switchRootFolders`** with module-dispatched version:

   ```go
   func (d *DevModeManager) switchRootFolders(devMode bool) {
       if !devMode { return }
       ctx := context.Background()
       fsmock.ResetInstance()

       mctx := d.buildMockContext(ctx)
       for _, mod := range d.registry.EnabledModules() {
           if err := mod.CreateTestRootFolders(ctx, mctx); err != nil {
               d.logger.Error().Err(err).Str("module", string(mod.ID())).Msg("Failed to create test root folders")
           }
       }
   }
   ```

4. **Replace `populateMockMedia`** with module-dispatched version:

   ```go
   func (d *DevModeManager) populateMockMedia() {
       ctx := context.Background()
       mctx := d.buildMockContext(ctx)
       for _, mod := range d.registry.EnabledModules() {
           if err := mod.CreateSampleLibraryData(ctx, mctx); err != nil {
               d.logger.Error().Err(err).Str("module", string(mod.ID())).Msg("Failed to create sample library data")
           }
       }
   }
   ```

   **Behavioral change:** The current `populateMockMedia` requires BOTH movie and TV root folders to exist before populating ANY media, and checks for existing movies globally. After refactoring, each module independently checks its own root folder and existing data. This removes an unnecessary cross-module dependency and allows partial population if one module fails.

5. **Add `buildMockContext` helper:**

   ```go
   func (d *DevModeManager) buildMockContext(ctx context.Context) *module.MockContext {
       profiles, _ := d.library.Quality.List(ctx)
       mockProfiles := make([]module.MockQualityProfile, len(profiles))
       for i, p := range profiles {
           mockProfiles[i] = module.MockQualityProfile{ID: p.ID, Name: p.Name}
       }
       var defaultProfileID int64
       if len(profiles) > 0 { defaultProfileID = profiles[0].ID }

       return &module.MockContext{
           DB:                d.dbManager.Conn(),
           Logger:            d.logger,
           RootFolderCreator: &mockRootFolderAdapter{service: d.library.RootFolder},
           QualityProfiles:   mockProfiles,
           DefaultProfileID:  defaultProfileID,
       }
   }
   ```

6. **Remove methods that moved to modules:**
   - `checkExistingMockFolders` — each module's `CreateTestRootFolders` checks for existing folders
   - `createMockMovieFolder` — moved to movie module
   - `createMockTVFolder` — moved to TV module
   - `populateMockMovies` — moved to movie module
   - `populateMockSeries` — moved to TV module
   - `createOneMockSeries` — moved to TV module
   - `convertSeasonsMetadata` — moved to TV module
   - `convertEpisodesMetadata` — moved to TV module
   - `createMockMovieFiles` — moved to movie module
   - `createMockEpisodeFiles` — moved to TV module
   - `buildEpisodeMap` — moved to TV module
   - `processSeasonDirectory` — moved to TV module
   - `processEpisodeFile` — moved to TV module
   - `parseQualityFromFilename` — moved to shared utility or module
   - `parseSeasonNumber` — moved to TV module
   - `parseEpisodeNumber` — moved to TV module
   - `isEpisodeMarker` — moved to TV module
   - `extractEpisodeDigits` — moved to TV module

7. **Keep in `devmode.go`:**
   - `OnToggle` — updated to call module factories instead of direct movie/TV methods
   - `switchMetadataClients` — unchanged (Phase 3 will modularize this)
   - `switchIndexer` — unchanged
   - `updateServicesDB` — unchanged
   - `switchDownloadClient` — unchanged
   - `switchNotification` — unchanged
   - `copyJWTSecretToDevDB` — unchanged
   - `copySettingsToDevDB` — unchanged
   - `copyPortalUsersToDevDB` — unchanged
   - `copyPortalUserNotificationsToDevDB` — unchanged
   - `copyQualityProfilesToDevDB` — unchanged
   - `setupDevModeSlots` — unchanged
   - `mapProfileID` — unchanged
   - `itoa` — can be removed if no longer used in devmode.go (moved to modules)

**Modify** `internal/api/wire.go`:
- Update `NewDevModeManager` provider to pass `*module.Registry`.

**Verify:**
- `go build ./...` compiles
- `devmode.go` is significantly smaller (~250 lines removed)
- The `OnToggle` flow calls module factories correctly

---

### Task 9.9: Document Health Monitoring Module Usage Patterns ✅

**Create** `internal/module/HEALTH_PATTERNS.md`:

Document how modules should use the health monitoring system. This is a reference for future module authors.

Content:

```markdown
# Health Monitoring for Modules

The health monitoring system (`internal/health/`) is already fully generic.
Modules interact with it via the `contracts.HealthService` interface, which uses
string-based categories and IDs. No code changes are needed to support new modules.

## Interface

Modules receive `contracts.HealthService` via constructor injection:

```go
type HealthService interface {
    RegisterItemStr(category, id, name string)
    UnregisterItemStr(category, id string)
    SetErrorStr(category, id, message string)
    ClearStatusStr(category, id string)
    SetWarningStr(category, id, message string)
}
```

## Categories

Current categories (defined in `health/types.go`):
- `downloadClients` — download client connectivity
- `indexers` — indexer connectivity and search health
- `prowlarr` — Prowlarr integration health
- `rootFolders` — root folder accessibility
- `metadata` — metadata provider connectivity
- `storage` — disk space warnings
- `import` — import pipeline health

Modules may use existing categories or define new ones. New categories are
automatically created when `RegisterItemStr` is called with an unknown category.
However, the `GetAll()` response struct (`HealthResponse`) has hard-coded fields
per category — a future phase should make this dynamic (use a map instead of
named fields) when a 3rd module adds a new category.

## Usage Patterns

### Metadata provider health
```go
// On service startup
s.health.RegisterItemStr("metadata", "musicbrainz", "MusicBrainz")

// On API call failure
s.health.SetErrorStr("metadata", "musicbrainz", "API returned 503")

// On successful API call after failure
s.health.ClearStatusStr("metadata", "musicbrainz")
```

### Root folder health
```go
// Root folder health is checked by the storage health scheduled task
// (`scheduler/tasks/storagehealth.go`). It iterates all root folders and
// checks accessibility. No module-specific code needed — the root folder's
// `media_type` (module_type after Phase 1) associates it with a module.
```

### Import pipeline health
```go
// The import service uses dedicated convenience methods:
s.health.RegisterImportItem("music-import", "Music Import Pipeline")
s.health.SetImportError("music-import", "FFprobe not found")
s.health.ClearImportStatus("music-import")
```

## Notifications

Health status changes automatically trigger notifications via the
`NotificationDispatcher` interface. When a health item transitions from
OK → Error/Warning, `DispatchHealthIssue` is called. When it transitions
back to OK, `DispatchHealthRestored` is called. Modules don't need to handle
notification dispatch — the health service does it automatically.
```

**Verify:**
- File exists and is readable
- No code changes needed

---

### Task 9.10: Frontend Updates for Module-Dispatched Arr Import ✅ (no changes needed)

The frontend arr import wizard currently has hard-coded "Radarr" and "Sonarr" options. Phase 9 backend changes are backward-compatible with the existing API contract — the same endpoints return the same response shapes. However, two frontend adjustments are needed:

**Modify** `web/src/types/arr-import.ts`:
- No type changes needed — the API response types (`ImportPreview`, `MoviePreview`, `SeriesPreview`, `ImportReport`) are unchanged.

**Modify** `web/src/routes/settings/media/arr-import.tsx`:
- The source type selector currently shows "Radarr" and "Sonarr". For Phase 9, this remains hard-coded (the available source types are determined by which modules have arr import adapters, but the UI doesn't yet query this dynamically). **No changes needed.**

**Modify** `web/src/components/arr-import/import-step.tsx`:
- No changes needed — the component renders preview data generically.

**Modify** `web/src/api/arr-import.ts`:
- No changes needed — API endpoints and request/response shapes are unchanged.

**Summary:** Phase 9 frontend changes are **zero** for arr import. The backend refactoring is transparent to the frontend because the API contract is preserved.

**Dev mode frontend:**

**Modify** `web/src/components/layout/header-dev-mode.tsx`:
- No changes needed — the dev mode toggle sends a WebSocket message to the backend. The backend's `OnToggle` handles everything.

**Modify** `web/src/stores/devmode.ts`:
- No changes needed.

**Summary:** Phase 9 frontend changes are **zero** overall. The refactoring is entirely backend.

**Verify:**
- `cd web && bun run lint` passes
- No frontend build errors

---

### Task 9.11: Phase 9 Validation ✅

Run full validation:

```bash
make build       # Backend + frontend compile
make test        # All Go tests pass
make lint        # Go linting clean
cd web && bun run lint  # Frontend linting clean
```

**Specific checks:**

1. **Arr import flow works end-to-end:**
   - `arrimport.Service` no longer imports `movies.Service` or `tv.Service` directly
   - `arrimport.Executor` no longer imports `movies.Service` or `tv.Service` directly
   - Module adapters are the only code that calls movie/TV service methods during import
   - All existing arr import API endpoints (`/api/v1/arrimport/*`) still register correctly

2. **Dev mode flow works end-to-end:**
   - `devmode.go` no longer contains movie-specific or TV-specific media creation code
   - `DevModeManager` dispatches to module `MockFactory` implementations
   - The `OnToggle(true)` path creates root folders and sample data via modules
   - `devmode.go` is ~250 lines shorter than before

3. **No circular dependencies:**
   - `internal/module/` does not import `internal/arrimport/`
   - `internal/module/` imports `internal/indexer/scanner` and `internal/library/slots` (for `ArrImportSlotsService` — these are leaf packages with no reverse deps)
   - `internal/arrimport/` does not import `internal/library/movies/` or `internal/library/tv/`
   - `internal/modules/movie/` imports `internal/module/` (not the reverse)
   - `internal/modules/tv/` imports `internal/module/` (not the reverse)

4. **Health system unchanged:**
   - No files modified in `internal/health/`
   - `HEALTH_PATTERNS.md` documentation created

**Verify all expected files are touched/created:**

**New files:**
```
internal/module/arr_import.go
internal/module/arr_source_types.go
internal/module/mock_types.go
internal/module/HEALTH_PATTERNS.md
internal/modules/movie/arr_import.go
internal/modules/movie/mock_factory.go
internal/modules/tv/arr_import.go
internal/modules/tv/mock_factory.go
```

**Deleted files:**
```
(none — old code in service.go and executor.go is modified, not deleted)
```

**Modified files:**
```
internal/module/interfaces.go (MockFactory signature update)
internal/module/optional_interfaces.go (comment update for arr adapters)
internal/module/registry.go (GetMovieArrAdapter, GetTVArrAdapter methods)
internal/arrimport/service.go (remove movie/TV deps, add registry, add reader adapter)
internal/arrimport/executor.go (remove movie/TV deps, dispatch to module adapters)
internal/api/devmode.go (remove ~20 movie/TV-specific methods, dispatch to MockFactory)
internal/api/wire.go (update NewService/NewDevModeManager providers)
internal/api/setters.go (remove SetMetadataRefresher call on line 36)
internal/modules/movie/module.go (compile-time assertions for new interfaces)
internal/modules/tv/module.go (compile-time assertions for new interfaces)
```

- Zero changes to: health system, notification system, search pipeline, download pipeline, portal system, quality profiles, database schema (no migrations in this phase)
- Config import (`arrimport/config_map.go`, `config_import_test.go`, etc.) is unchanged — it remains framework-owned
- The `Reader` interface and implementations (`reader_sqlite.go`, `reader_api.go`) are unchanged — only wrapped via `readerAdapter`
- `arrimport/types.go` (API response types) is unchanged, but a `convertToModuleMappings` helper is added to `executor.go` or `service.go`
- `arrimport/handlers.go` is unchanged — same routes, same handler methods
- Frontend is unchanged
