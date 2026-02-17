# Library Import (Migrate from *arr) — Implementation Plan

## Context

SlipStream users migrating from Radarr/Sonarr need a way to import their existing libraries. The spec (`docs/Library-Import-Spec.md`) defines field mappings, quality ID translation tables, and a multi-step workflow. This plan creates a new `internal/arrimport/` backend package and a new frontend wizard tab in Media settings.

---

## Phase 1: Backend — Core Types, Quality Mapping, Reader Interface

### 1.1 Package: `internal/arrimport/`

```
internal/arrimport/
  types.go           -- Source entity types, connection config, import results
  quality_map.go     -- Quality ID mapping tables (spec §2.4, §3.6)
  reader.go          -- Reader interface definition
  reader_sqlite.go   -- SQLite backend (opens source .db read-only)
  reader_api.go      -- API backend (v3 REST with X-Api-Key)
  service.go         -- Import orchestration, preview, session management
  executor.go        -- Import execution (calls movie/tv services)
  handlers.go        -- HTTP handlers for /arrimport endpoints
```

### 1.2 Types (`types.go`)

**Connection:**
```go
type SourceType string // "radarr" | "sonarr"

type ConnectionConfig struct {
    SourceType SourceType `json:"sourceType"`
    DBPath     string     `json:"dbPath,omitempty"`     // SQLite mode
    URL        string     `json:"url,omitempty"`         // API mode
    APIKey     string     `json:"apiKey,omitempty"`      // API mode
}
```

**Source entities** — unified types representing data from either backend:
- `SourceRootFolder{ID, Path}`
- `SourceQualityProfile{ID, Name}`
- `SourceMovie{Title, SortTitle, Year, TmdbID, ImdbID, Overview, Runtime, Path, RootFolderPath, QualityProfileID, Monitored, Status, InCinemas, PhysicalRelease, DigitalRelease, Studio, Certification, Added, HasFile, File *SourceMovieFile}`
- `SourceMovieFile{Path, Size, QualityID, QualityName, VideoCodec, AudioCodec, Resolution, AudioChannels, DynamicRange, OriginalFilePath, DateAdded}`
- `SourceSeries{Title, SortTitle, Year, TvdbID, TmdbID, ImdbID, Overview, Runtime, Path, RootFolderPath, QualityProfileID, Monitored, SeasonFolder, Status, Network, SeriesType, Certification, Added, Seasons[]}`
- `SourceSeason{SeasonNumber, Monitored}`
- `SourceEpisode{SeriesID, SeasonNumber, EpisodeNumber, Title, Overview, AirDateUtc, Monitored, EpisodeFileID, HasFile}`
- `SourceEpisodeFile{ID, SeriesID, SeasonNumber, RelativePath, Size, QualityID, QualityName, VideoCodec, AudioCodec, Resolution, AudioChannels, DynamicRange, OriginalFilePath, DateAdded}`

**Import results:**
- `ImportPreview{Movies[], Series[], Summary, Conflicts[]}`
- `MoviePreview{Title, Year, TmdbID, HasFile, Quality, Status("new"/"duplicate"/"skip"), SkipReason}`
- `SeriesPreview{Title, Year, TvdbID, EpisodeCount, FileCount, Status, SkipReason}`
- `ImportSummary{TotalMovies, TotalSeries, TotalEpisodes, TotalFiles, NewMovies, NewSeries, DuplicateMovies, DuplicateSeries, SkippedMovies, SkippedSeries}`
- `ImportReport{MoviesCreated, MoviesSkipped, MoviesErrored, SeriesCreated, SeriesSkipped, SeriesErrored, FilesImported, Errors[]}`

### 1.3 Quality ID Mapping (`quality_map.go`)

Static mapping tables from spec §2.4 and §3.6. All entries from the spec tables:

```go
var radarrQualityMap = map[int]int{
    1:1, 2:2, 8:3, 12:3, 4:4, 5:6, 14:5, 6:7, 9:8, 3:10,
    15:9, 7:11, 30:12, 16:13, 18:15, 17:14, 19:16, 31:17,
    20:2, 21:2, 23:2,
    // 0,10,22,24-29 -> unmapped, return nil
}

var sonarrQualityMap = map[int]int{
    // Start with all radarrQualityMap entries:
    1:1, 2:2, 8:3, 12:3, 4:4, 5:6, 14:5, 6:7, 9:8, 3:10,
    15:9, 7:11, 30:12, 16:13, 18:15, 17:14, 19:16, 31:17,
    23:2,
    // Sonarr-only additions:
    13:2, 22:2,
    // OVERRIDES (different meaning than Radarr for same IDs):
    20:12,  // Sonarr: Bluray-1080p Remux → Remux-1080p (Radarr: Bluray-480p → DVD)
    21:17,  // Sonarr: Bluray-2160p Remux → Remux-2160p (Radarr: Bluray-576p → DVD)
}
```

`MapQualityID(sourceType, sourceQualityID, sourceQualityName) *int64`:
1. Look up in the appropriate static map
2. If miss, try `quality.GetQualityByName(sourceQualityName)`
3. If still miss, return nil (file created without quality_id)

### 1.4 Reader Interface (`reader.go`)

```go
type Reader interface {
    Validate(ctx context.Context) error
    ReadRootFolders(ctx context.Context) ([]SourceRootFolder, error)
    ReadQualityProfiles(ctx context.Context) ([]SourceQualityProfile, error)
    ReadMovies(ctx context.Context) ([]SourceMovie, error)          // Radarr only
    ReadSeries(ctx context.Context) ([]SourceSeries, error)         // Sonarr only
    ReadEpisodes(ctx context.Context, seriesID int64) ([]SourceEpisode, error)
    ReadEpisodeFiles(ctx context.Context, seriesID int64) ([]SourceEpisodeFile, error)
    Close() error
}
```

**Constructor:** `NewReader(cfg ConnectionConfig) (Reader, error)` — returns SQLite or API reader based on config.

### 1.5 SQLite Reader (`reader_sqlite.go`)

- Opens source DB with `?mode=ro&_journal_mode=wal` (read-only, no risk to source)
- **Radarr movies**: `SELECT m.*, mm.* FROM Movies m JOIN MovieMetadata mm ON m.MovieMetadataId = mm.Id WHERE mm.Status != -1` (skip deleted)
- **Radarr movie files**: `SELECT * FROM MovieFiles WHERE MovieId = ?`; parse `Quality` JSON column for `quality.quality.id` and `quality.quality.name`; parse `MediaInfo` JSON for codecs/resolution/channels/HDR
- **Sonarr series**: `SELECT * FROM Series WHERE Status != -1`; parse `Seasons` JSON column for season list; derive `RootFolderPath` from `Path`
- **Sonarr episodes**: `SELECT * FROM Episodes WHERE SeriesId = ?`; `EpisodeFileId > 0` means `HasFile = true`
- **Sonarr episode files**: `SELECT * FROM EpisodeFiles WHERE SeriesId = ?`; `RelativePath` is relative — must NOT prepend series path here (executor handles full path construction); parse `Quality` JSON and `MediaInfo` JSON same as Radarr
- **Root folders**: `SELECT * FROM RootFolders`
- **Quality profiles**: `SELECT Id, Name FROM QualityProfiles`

### 1.6 API Reader (`reader_api.go`)

- Uses `net/http` with `X-Api-Key` header
- Radarr endpoints: `GET /api/v3/movie`, `GET /api/v3/moviefile`, `GET /api/v3/qualityprofile`, `GET /api/v3/rootfolder`
- Sonarr endpoints: `GET /api/v3/series`, `GET /api/v3/episode?seriesId=`, `GET /api/v3/episodefile?seriesId=`, `GET /api/v3/qualityprofile`, `GET /api/v3/rootfolder`
- JSON response fields map directly to Source types (API field names match the spec)
- For Sonarr: `episodeFile.path` from API is already absolute (computed by Sonarr)
- Validate via `GET /api/v3/system/status` — check response for source type confirmation

---

## Phase 2: Backend — Service, Preview, Executor

### 2.1 Service (`service.go`)

```go
type Service struct {
    movieService    *movies.Service
    tvService       *tv.Service
    rootFolderSvc   *rootfolder.Service
    qualityService  *quality.Service
    slotsSvc        *slots.Service       // optional, set via setter
    progressManager *progress.Manager
    hub             *websocket.Hub
    logger          *zerolog.Logger

    // Session state (one active import at a time)
    mu       sync.Mutex
    reader   Reader
    sourceType SourceType
}
```

Key methods:
- `Connect(ctx, cfg ConnectionConfig) error` — creates and validates a Reader, stores in session
- `GetSourceRootFolders(ctx) ([]SourceRootFolder, error)` — delegates to reader
- `GetSourceQualityProfiles(ctx) ([]SourceQualityProfile, error)` — delegates to reader
- `Preview(ctx, mappings ImportMappings) (*ImportPreview, error)` — reads source, checks for duplicates against existing SlipStream library
- `Execute(ctx, mappings ImportMappings) (*ImportReport, error)` — runs the import in a goroutine, returns immediately, progress via WebSocket
- `Disconnect()` — closes reader, clears session
- `SetSlotsService(svc *slots.Service)` — optional setter

**ImportMappings:**
```go
type ImportMappings struct {
    RootFolderMapping      map[string]int64  `json:"rootFolderMapping"`      // source path -> SS root folder ID
    QualityProfileMapping  map[int64]int64   `json:"qualityProfileMapping"`  // source profile ID -> SS profile ID
}
```

### 2.2 Preview Logic

For each source movie/series:
1. Check TMDB/TVDB ID against existing library using `movies.Service.GetByTmdbID()` / `tv.Service.GetSeriesByTvdbID()`
2. Mark as "duplicate" if exists, "new" if doesn't, "skip" if deleted status or missing required IDs
3. Count files, compute summary stats
4. Identify conflicts (duplicate external IDs)

### 2.3 Executor (`executor.go`)

**Movie import flow (per movie):**
1. Skip if Radarr status is deleted (-1)
2. Skip if `GetByTmdbID` returns existing movie (duplicate)
3. Resolve root folder ID from `mappings.RootFolderMapping[movie.RootFolderPath]`
4. Resolve quality profile ID from `mappings.QualityProfileMapping[movie.QualityProfileID]`
5. Call `movies.Service.Create(ctx, &movies.CreateMovieInput{...})` — maps all 20 fields per spec §2.1
   - Format dates as "YYYY-MM-DD" strings
   - `updated_at` = import timestamp (handled by service)
   - Status auto-derived from release dates
6. If `slots.Service.IsMultiVersionEnabled(ctx)`: call `slots.Service.InitializeSlotAssignments(ctx, "movie", movieID)`
7. If movie has file:
   a. Map quality ID via `MapQualityID()`
   b. Call `movies.Service.AddFile(ctx, movieID, &movies.CreateMovieFileInput{...})` — maps all 14 fields per spec §2.2
      - `OriginalFilename` = `path.Base(file.OriginalFilePath)`
   c. If multi-version: `scanner.ParsePath(file.Path)` → `slots.Service.DetermineTargetSlot()` → `slots.Service.AssignFileToSlot()`
8. Update progress: `progressManager.UpdateActivity(id, "Importing: "+movie.Title, pct)`

**TV series import flow (per series):**
1. Skip if Sonarr status is "deleted"
2. Skip if `GetSeriesByTvdbID` returns existing series
3. Resolve root folder and quality profile same as movies
4. Read episodes and episode files from source
5. Build season→episode tree from episodes list, paired with source seasons for monitored flag
6. Call `tv.Service.CreateSeries(ctx, &tv.CreateSeriesInput{...})` — maps all 19 fields per spec §3.1
   - `ProductionStatus` = Sonarr's `Status` (required field)
   - `FormatType` = Sonarr's `SeriesType`
   - Seasons include episodes with `AirDate` set (status auto-derived)
   - Episode titles wrapped for `sql.NullString` (handled internally by tv.Service)
7. If multi-version: iterate created episodes → `InitializeSlotAssignments(ctx, "episode", episodeID)` for each
8. For each episode file:
   a. Find source episode by matching `EpisodeFileID` from source episodes
   b. Look up SlipStream episode via `tv.Service.GetEpisodeByNumber(seriesID, seasonNum, episodeNum)`
   c. Map quality ID
   d. Compute absolute path: `series.Path + "/" + file.RelativePath` (for SQLite reads; API already provides absolute)
   e. Call `tv.Service.AddEpisodeFile(ctx, episodeID, &tv.CreateEpisodeFileInput{...})`
   f. If multi-version: determine target slot and assign
9. Update progress

**Progress reporting:**
```go
activity := progressManager.StartActivity("arrimport", progress.ActivityTypeImport, "Library Import")
// Per item: progressManager.UpdateActivity("arrimport", "Importing: The Dark Knight", 45)
// Done: progressManager.CompleteActivity("arrimport", "Import complete: 120 movies, 35 series")
```

### 2.4 Handlers (`handlers.go`)

| Method | Path | Body | Description |
|--------|------|------|-------------|
| POST | `/arrimport/connect` | `ConnectionConfig` | Validate + store reader session |
| GET | `/arrimport/source/rootfolders` | — | Read source root folders |
| GET | `/arrimport/source/qualityprofiles` | — | Read source quality profiles |
| POST | `/arrimport/preview` | `ImportMappings` | Generate import preview |
| POST | `/arrimport/execute` | `ImportMappings` | Start async import, returns immediately |
| DELETE | `/arrimport/session` | — | Disconnect and cleanup |

Register in `internal/api/routes.go` under admin-protected group.

### 2.5 Server Wiring (`internal/api/server.go`)

Add `arrImportService *arrimport.Service` to Server struct. Initialize:
```go
s.arrImportService = arrimport.NewService(s.movieService, s.tvService, s.rootFolderService,
    s.qualityService, s.progressManager, s.hub, logger)
s.arrImportService.SetSlotsService(s.slotsService)
```

---

## Phase 3: Frontend — Route, Navigation, API Module

### 3.1 Navigation

**File:** `web/src/routes/settings/media/media-nav.tsx`

Add new tab after "Import & Naming":
```ts
{ title: 'Migrate from *arr', href: '/settings/media/arr-import', icon: ArrowRightLeft }
```

### 3.2 Route Registration

**File:** `web/src/routes-config.tsx` — add route definition
**File:** `web/src/router.tsx` — add to media settings route tree

### 3.3 API Module

**New file:** `web/src/api/arr-import.ts`
```ts
export const arrImportApi = {
    connect: (config: ConnectionConfig) => apiFetch('/arrimport/connect', { method: 'POST', body: ... }),
    getSourceRootFolders: () => apiFetch('/arrimport/source/rootfolders'),
    getSourceQualityProfiles: () => apiFetch('/arrimport/source/qualityprofiles'),
    preview: (mappings: ImportMappings) => apiFetch('/arrimport/preview', { method: 'POST', body: ... }),
    execute: (mappings: ImportMappings) => apiFetch('/arrimport/execute', { method: 'POST', body: ... }),
    disconnect: () => apiFetch('/arrimport/session', { method: 'DELETE' }),
}
```

**New file:** `web/src/types/arr-import.ts` — TypeScript types matching backend types

### 3.4 Hooks

**New file:** `web/src/hooks/use-arr-import.ts`

Manages wizard state: `step: 'connect' | 'mapping' | 'preview' | 'importing' | 'report'`

Uses TanStack Query mutations for connect/preview/execute. Listens for WebSocket `progress:update` events during import step.

---

## Phase 4: Frontend — Wizard Components

### 4.1 Component Structure

```
web/src/components/arr-import/
  index.tsx                -- Wizard shell (step router)
  connect-step.tsx         -- Step 1: Source selection + connection
  mapping-step.tsx         -- Step 2: Root folder + quality profile mapping
  preview-step.tsx         -- Step 3: Preview (reuses DryRunModal patterns)
  import-step.tsx          -- Step 4: Progress bar + final report
```

### 4.2 Connect Step

- Radio group: "Radarr (Movies)" / "Sonarr (TV Series)"
- Radio group: "Direct Database (SQLite)" / "API Connection"
- SQLite mode: Text input for DB path + folder browser button (reuse `FolderBrowser` from `web/src/components/forms/folder-browser.tsx`, adapt to allow `.db` file selection)
- API mode: URL input + API key input
- "Connect" button → fires `POST /arrimport/connect`
- On success: auto-fetches source root folders + quality profiles, advances to mapping step

### 4.3 Mapping Step

Two table sections:

**Root Folder Mapping:**
- Each row: source path (read-only) | SlipStream root folder dropdown (Select component)
- Auto-match by path where possible
- "Create new" option to create missing root folders

**Quality Profile Mapping:**
- Each row: source profile name (read-only) | SlipStream profile dropdown
- All profiles must be mapped before "Next" is enabled

### 4.4 Preview Step

Reuses patterns from `web/src/components/slots/DryRunModal/`:
- Summary cards at top (total, new, duplicates, files) — click to filter
- Tabs: Movies | TV Shows (line variant)
- Scrollable list of items with status badges (new=green, duplicate=yellow, skip=gray)
- "Import" button at bottom

### 4.5 Import Step

- Progress bar component using `web/src/components/ui/progress.tsx`
- Listens to WebSocket for `progress:update` events
- Shows current item: "Importing movie 45/120: The Dark Knight"
- On completion: shows ImportReport summary
  - Cards: movies created, series created, files imported, errors
  - Expandable error list if any

---

## Phase 5: Page Shell

**New file:** `web/src/routes/settings/media/arr-import.tsx`

Follows standard settings page pattern:
```tsx
export function ArrImportPage() {
  return (
    <div className="space-y-6">
      <PageHeader title="Media Management" description="..." breadcrumbs={...} />
      <MediaNav />
      <ArrImportWizard />
    </div>
  )
}
```

---

## Implementation Sequence

| # | Scope | Files |
|---|-------|-------|
| 1 | Types + quality mapping | `internal/arrimport/types.go`, `quality_map.go` |
| 2 | Reader interface + SQLite reader (Radarr) | `reader.go`, `reader_sqlite.go` |
| 3 | SQLite reader (Sonarr) | `reader_sqlite.go` (extend) |
| 4 | API reader (both) | `reader_api.go` |
| 5 | Service + preview logic | `service.go` |
| 6 | Import executor | `executor.go` |
| 7 | Handlers + route registration | `handlers.go`, `internal/api/routes.go` |
| 8 | Server wiring | `internal/api/server.go` |
| 9 | Frontend types + API module | `web/src/types/arr-import.ts`, `web/src/api/arr-import.ts` |
| 10 | Frontend route + nav | `media-nav.tsx`, `routes-config.tsx`, `router.tsx` |
| 11 | Frontend hooks | `web/src/hooks/use-arr-import.ts` |
| 12 | Frontend connect step | `connect-step.tsx` |
| 13 | Frontend mapping step | `mapping-step.tsx` |
| 14 | Frontend preview step | `preview-step.tsx` |
| 15 | Frontend import + report step | `import-step.tsx` |
| 16 | Wizard shell + page | `web/src/components/arr-import/index.tsx`, `arr-import.tsx` |
| 17 | Linting + cleanup | Run `make lint`, `cd web && bun run lint` |

---

## Key Existing Functions to Reuse

| Function | Location | Usage |
|----------|----------|-------|
| `movies.Service.Create()` | `internal/library/movies/service.go:200` | Create each imported movie |
| `movies.Service.AddFile()` | `internal/library/movies/service.go` | Attach file to movie |
| `movies.Service.GetByTmdbID()` | `internal/library/movies/service.go:147` | Duplicate detection |
| `tv.Service.CreateSeries()` | `internal/library/tv/service.go:200` | Create series + seasons + episodes |
| `tv.Service.AddEpisodeFile()` | `internal/library/tv/episodes.go` | Attach file to episode |
| `tv.Service.GetSeriesByTvdbID()` | `internal/library/tv/service.go:129` | Duplicate detection |
| `tv.Service.GetEpisodeByNumber()` | `internal/library/tv/episodes.go:66` | Look up episode for file attachment |
| `quality.GetQualityByName()` | `internal/library/quality/profile.go` | Fallback quality mapping |
| `slots.Service.IsMultiVersionEnabled()` | `internal/library/slots/service.go` | Check multi-version mode |
| `slots.Service.InitializeSlotAssignments()` | `internal/library/slots/status.go:365` | Create slot assignment rows |
| `slots.Service.DetermineTargetSlot()` | `internal/library/slots/assignment.go:153` | Find best slot for file |
| `slots.Service.AssignFileToSlot()` | `internal/library/slots/assignment.go` | Assign file to slot |
| `scanner.ParsePath()` | `internal/library/scanner/parser.go` | Parse file path for slot matching |
| `rootfolder.Service.GetByPath()` | `internal/library/rootfolder/service.go` | Match root folders by path |
| `progress.Manager.StartActivity/UpdateActivity/CompleteActivity` | `internal/progress/progress.go` | WebSocket progress reporting |

---

## Verification

1. **Backend unit tests**: Quality mapping (every entry from spec), SQLite reader (mock DB), executor (mock services)
2. **Manual integration test**: Use a real `radarr.db` / `sonarr.db` file to test end-to-end flow
3. **Frontend**: Navigate to Settings → Media → Migrate from *arr, complete the full wizard flow
4. **Linting**: `make lint` + `cd web && bun run lint`
5. **Multi-version**: Test with slots enabled — verify slot assignments created and files assigned

---

## Agent Implementation Guide

See `docs/Library-Import-Agent-Instructions.md` for:
- Optimized task decomposition and parallelization strategy
- Exact function signatures and struct definitions for all APIs
- Comprehensive gotchas/pitfalls list (24 items)
- Pre-written test specifications
- Per-task instructions with file lists and step-by-step guidance
