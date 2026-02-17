# Library Import — Agent Implementation Instructions

> **Purpose:** This document provides optimized instructions for an AI agent implementing the Library Import feature. It is designed to maximize context efficiency, prevent common mistakes, and ensure deterministic quality through tests.
>
> **Reference documents:**
> - `docs/Library-Import-Spec.md` — Source of truth for field mappings and quality ID tables
> - `docs/Library-Import-Plan.md` — Implementation plan (architecture, phases, file structure)

---

## Table of Contents

1. [Task Decomposition & Agent Strategy](#1-task-decomposition--agent-strategy)
2. [Verification Script](#2-verification-script)
3. [API Reference (Exact Signatures)](#3-api-reference-exact-signatures)
4. [Gotchas & Pitfalls](#4-gotchas--pitfalls)
5. [Test Specifications](#5-test-specifications)
6. [Frontend Reference](#6-frontend-reference)
7. [Per-Task Instructions](#7-per-task-instructions)

---

## 1. Task Decomposition & Agent Strategy

### Execution Order & Dependency Graph

```
Task 1: Types + Quality Map ──────────────────┐
                                               ├─→ Task 5: Service + Preview
Task 2: Reader Interface + SQLite Reader ──────┤
                                               ├─→ Task 6: Executor
Task 3: SQLite Reader (Sonarr extension) ──────┘

Task 4: API Reader ────────────────────────────→ (independent after Task 1)

Task 5: Service + Preview ─────┐
Task 6: Executor ──────────────┼─→ Task 7: Handlers
                               │
                               └─→ Task 8: Server Wiring + Routes

Task 9:  Frontend Types + API ──────┐
Task 10: Frontend Route + Nav ──────┼─→ Task 11: Frontend Hooks
                                    │
                                    └─→ Tasks 12-16: Wizard Components

Task 17: Quality Map Tests (run alongside Task 1)
Task 18: Linting + Cleanup (final)
```

### Agent Assignment

| Task | Model | Rationale |
|------|-------|-----------|
| 1: Types + Quality Map | Sonnet | Mechanical mapping, well-specified |
| 2: Reader Interface + SQLite (Radarr) | Sonnet | Medium complexity, clear SQL patterns |
| 3: SQLite Reader (Sonarr) | Sonnet | Extension of Task 2, same patterns |
| 4: API Reader | Sonnet | Medium complexity, HTTP client boilerplate |
| 5: Service + Preview | Sonnet | Orchestration, clear interfaces |
| 6: Executor | **Opus** | Highest complexity — field mappings, slot integration, error handling |
| 7: Handlers | Sonnet | Simple HTTP wrapper |
| 8: Server Wiring | Sonnet | Boilerplate, follow established pattern |
| 9-10: Frontend types/routes | Sonnet | Low complexity |
| 11: Frontend hooks | Sonnet | Medium complexity |
| 12-16: Wizard components | Sonnet | Medium complexity, clear UI patterns |
| 17: Tests | Sonnet | Follow patterns |
| 18: Lint cleanup | Sonnet | Mechanical |

### Parallelization Strategy

**Round 1 (parallel):** Tasks 1, 17 (test shell)
**Round 2 (parallel after Task 1):** Tasks 2+3, 4
**Round 3 (parallel after Tasks 2-4):** Tasks 5, 6
**Round 4 (after Tasks 5-6):** Tasks 7, 8
**Round 5 (parallel, independent of backend):** Tasks 9, 10
**Round 6 (after Tasks 9-10):** Task 11
**Round 7 (parallel after Task 11):** Tasks 12, 13, 14, 15
**Round 8 (after Tasks 12-15):** Task 16
**Round 9 (final):** Task 18

### Context Management Rules

1. **Each subagent receives:** This document + the specific task instructions from §7
2. **Subagents do NOT need to read** the full spec or plan — relevant excerpts are embedded in task instructions
3. **File reads per task:** Each task lists exactly which files to read before writing
4. **Cross-task dependencies:** When a task depends on output from a previous task, the orchestrator should verify the dependency files exist before launching

---

## 2. Verification Script

Create this test file early (Task 17) to catch quality mapping errors:

**File:** `internal/arrimport/quality_map_test.go`

```go
package arrimport

import (
	"testing"
)

// TestRadarrQualityMap verifies every entry from spec §2.4
func TestRadarrQualityMap(t *testing.T) {
	tests := []struct {
		radarrID    int
		name        string
		expectedSSID *int
	}{
		// Direct matches (spec §2.4, rows 1-17)
		{1, "SDTV", intPtr(1)},
		{2, "DVD", intPtr(2)},
		{8, "WEBDL-480p", intPtr(3)},       // nearest: WEBRip-480p
		{12, "WEBRip-480p", intPtr(3)},
		{4, "HDTV-720p", intPtr(4)},
		{5, "WEBDL-720p", intPtr(6)},
		{14, "WEBRip-720p", intPtr(5)},
		{6, "Bluray-720p", intPtr(7)},
		{9, "HDTV-1080p", intPtr(8)},
		{3, "WEBDL-1080p", intPtr(10)},
		{15, "WEBRip-1080p", intPtr(9)},
		{7, "Bluray-1080p", intPtr(11)},
		{30, "Remux-1080p", intPtr(12)},
		{16, "HDTV-2160p", intPtr(13)},
		{18, "WEBDL-2160p", intPtr(15)},
		{17, "WEBRip-2160p", intPtr(14)},
		{19, "Bluray-2160p", intPtr(16)},
		{31, "Remux-2160p", intPtr(17)},
		// Approximate matches
		{20, "Bluray-480p", intPtr(2)},      // nearest: DVD
		{21, "Bluray-576p", intPtr(2)},      // nearest: DVD
		{23, "DVD-R", intPtr(2)},            // nearest: DVD
		// Unmapped (should return nil)
		{0, "Unknown", nil},
		{10, "Raw-HD", nil},
		{22, "BR-DISK", nil},
		{24, "WORKPRINT", nil},
		{25, "CAM", nil},
		{26, "TELESYNC", nil},
		{27, "TELECINE", nil},
		{28, "DVDSCR", nil},
		{29, "REGIONAL", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapRadarrQualityID(tt.radarrID)
			if tt.expectedSSID == nil {
				if result != nil {
					t.Errorf("radarr ID %d (%s): expected nil, got %d", tt.radarrID, tt.name, *result)
				}
			} else {
				if result == nil {
					t.Errorf("radarr ID %d (%s): expected %d, got nil", tt.radarrID, tt.name, *tt.expectedSSID)
				} else if *result != int64(*tt.expectedSSID) {
					t.Errorf("radarr ID %d (%s): expected %d, got %d", tt.radarrID, tt.name, *tt.expectedSSID, *result)
				}
			}
		})
	}
}

// TestSonarrQualityMap verifies Sonarr-specific additions from spec §3.6
func TestSonarrQualityMap(t *testing.T) {
	sonarrOnlyTests := []struct {
		sonarrID    int
		name        string
		expectedSSID *int
	}{
		{13, "Bluray-480p", intPtr(2)},          // nearest: DVD
		{20, "Bluray-1080p Remux", intPtr(12)},  // Remux-1080p
		{21, "Bluray-2160p Remux", intPtr(17)},  // Remux-2160p
		{22, "Bluray-576p", intPtr(2)},          // nearest: DVD
	}

	for _, tt := range sonarrOnlyTests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapSonarrQualityID(tt.sonarrID)
			if tt.expectedSSID == nil {
				if result != nil {
					t.Errorf("sonarr ID %d (%s): expected nil, got %d", tt.sonarrID, tt.name, *result)
				}
			} else {
				if result == nil {
					t.Errorf("sonarr ID %d (%s): expected %d, got nil", tt.sonarrID, tt.name, *tt.expectedSSID)
				} else if *result != int64(*tt.expectedSSID) {
					t.Errorf("sonarr ID %d (%s): expected %d, got %d", tt.sonarrID, tt.name, *tt.expectedSSID, *result)
				}
			}
		})
	}

	// Verify Sonarr includes all Radarr mappings EXCEPT overridden IDs 20, 21
	radarrMapped := []int{1, 2, 8, 12, 4, 5, 14, 6, 9, 3, 15, 7, 30, 16, 18, 17, 19, 31, 23}
	sonarrOverrides := map[int]bool{20: true, 21: true} // These differ between Radarr and Sonarr
	for _, id := range radarrMapped {
		if sonarrOverrides[id] {
			continue // Skip IDs that are intentionally different
		}
		r := mapRadarrQualityID(id)
		s := mapSonarrQualityID(id)
		if r == nil || s == nil {
			continue // Skip unmapped
		}
		if *r != *s {
			t.Errorf("ID %d: Radarr maps to %d but Sonarr maps to %d (should match)", id, *r, *s)
		}
	}

	// Verify Sonarr overrides are different from Radarr
	r20 := mapRadarrQualityID(20)
	s20 := mapSonarrQualityID(20)
	if r20 != nil && s20 != nil && *r20 == *s20 {
		t.Errorf("ID 20: Radarr and Sonarr should differ (Radarr=Bluray-480p→DVD, Sonarr=Remux-1080p)")
	}
	r21 := mapRadarrQualityID(21)
	s21 := mapSonarrQualityID(21)
	if r21 != nil && s21 != nil && *r21 == *s21 {
		t.Errorf("ID 21: Radarr and Sonarr should differ (Radarr=Bluray-576p→DVD, Sonarr=Remux-2160p)")
	}
}

func intPtr(v int) *int { return &v }
```

**Important:** The test references unexported functions `mapRadarrQualityID` and `mapSonarrQualityID`. The `quality_map.go` implementation should expose these as unexported helpers called by the exported `MapQualityID`:

```go
func MapQualityID(sourceType SourceType, sourceQualityID int, sourceQualityName string) *int64 {
    var result *int64
    switch sourceType {
    case SourceTypeRadarr:
        result = mapRadarrQualityID(sourceQualityID)
    case SourceTypeSonarr:
        result = mapSonarrQualityID(sourceQualityID)
    }
    if result != nil {
        return result
    }
    // Fallback: try matching by name
    if q, ok := quality.GetQualityByName(sourceQualityName); ok {
        id := int64(q.ID)
        return &id
    }
    return nil
}
```

---

## 3. API Reference (Exact Signatures)

### Movie Service (`internal/library/movies/`)

```go
// service.go
func (s *Service) Create(ctx context.Context, input *CreateMovieInput) (*Movie, error)
func (s *Service) AddFile(ctx context.Context, movieID int64, input *CreateMovieFileInput) (*MovieFile, error)
func (s *Service) GetByTmdbID(ctx context.Context, tmdbID int) (*Movie, error)
// Returns ErrMovieNotFound (wraps sql.ErrNoRows) if not found

// movie.go — CreateMovieInput
type CreateMovieInput struct {
    Title                 string `json:"title"`                          // REQUIRED
    Year                  int    `json:"year,omitempty"`
    TmdbID                int    `json:"tmdbId,omitempty"`               // ErrDuplicateTmdbID if exists
    ImdbID                string `json:"imdbId,omitempty"`
    Overview              string `json:"overview,omitempty"`
    Runtime               int    `json:"runtime,omitempty"`
    Path                  string `json:"path,omitempty"`
    RootFolderID          int64  `json:"rootFolderId"`
    QualityProfileID      int64  `json:"qualityProfileId"`
    Monitored             bool   `json:"monitored"`
    Studio                string `json:"studio,omitempty"`
    TvdbID                int    `json:"tvdbId,omitempty"`
    ContentRating         string `json:"contentRating,omitempty"`
    ReleaseDate           string `json:"releaseDate,omitempty"`           // "YYYY-MM-DD"
    PhysicalReleaseDate   string `json:"physicalReleaseDate,omitempty"`   // "YYYY-MM-DD"
    TheatricalReleaseDate string `json:"theatricalReleaseDate,omitempty"` // "YYYY-MM-DD"
    AddedBy               *int64 `json:"-"`
}

// movie.go — CreateMovieFileInput
type CreateMovieFileInput struct {
    Path             string `json:"path"`                // REQUIRED
    Size             int64  `json:"size"`
    Quality          string `json:"quality,omitempty"`
    QualityID        *int64 `json:"qualityId,omitempty"` // pointer — nil means unknown
    VideoCodec       string `json:"videoCodec,omitempty"`
    AudioCodec       string `json:"audioCodec,omitempty"`
    AudioChannels    string `json:"audioChannels,omitempty"`
    DynamicRange     string `json:"dynamicRange,omitempty"`
    Resolution       string `json:"resolution,omitempty"`
    OriginalPath     string `json:"originalPath,omitempty"`
    OriginalFilename string `json:"originalFilename,omitempty"`
}
// NOTE: When OriginalPath != "", imported_at is set to time.Now() automatically.
// NOTE: After AddFile, movie status auto-updates to "available" or "upgradable".
```

### TV Service (`internal/library/tv/`)

```go
// service.go
func (s *Service) CreateSeries(ctx context.Context, input *CreateSeriesInput) (*Series, error)
func (s *Service) GetSeriesByTvdbID(ctx context.Context, tvdbID int) (*Series, error)
// Returns ErrSeriesNotFound if not found

// episodes.go
func (s *Service) AddEpisodeFile(ctx context.Context, episodeID int64, input *CreateEpisodeFileInput) (*EpisodeFile, error)
func (s *Service) GetEpisodeByNumber(ctx context.Context, seriesID int64, seasonNumber, episodeNumber int) (*Episode, error)
// Returns ErrEpisodeNotFound if not found

// series.go — CreateSeriesInput
type CreateSeriesInput struct {
    Title            string        `json:"title"`                             // REQUIRED
    Year             int           `json:"year,omitempty"`
    TvdbID           int           `json:"tvdbId,omitempty"`                  // ErrDuplicateTvdbID if exists
    TmdbID           int           `json:"tmdbId,omitempty"`
    ImdbID           string        `json:"imdbId,omitempty"`
    Overview         string        `json:"overview,omitempty"`
    Runtime          int           `json:"runtime,omitempty"`
    Path             string        `json:"path,omitempty"`
    RootFolderID     int64         `json:"rootFolderId"`
    QualityProfileID int64         `json:"qualityProfileId"`
    Monitored        bool          `json:"monitored"`
    SeasonFolder     bool          `json:"seasonFolder"`
    Network          string        `json:"network,omitempty"`
    NetworkLogoURL   string        `json:"networkLogoUrl,omitempty"`
    FormatType       string        `json:"formatType,omitempty"`
    ProductionStatus string        `json:"productionStatus,omitempty"`        // defaults to "continuing" if empty
    Seasons          []SeasonInput `json:"seasons,omitempty"`
    AddedBy          *int64        `json:"-"`
}

type SeasonInput struct {
    SeasonNumber int            `json:"seasonNumber"`
    Monitored    bool           `json:"monitored"`
    Episodes     []EpisodeInput `json:"episodes,omitempty"`
}

type EpisodeInput struct {
    EpisodeNumber int        `json:"episodeNumber"`
    Title         string     `json:"title"`           // plain string — service wraps in sql.NullString internally
    Overview      string     `json:"overview,omitempty"`
    AirDate       *time.Time `json:"airDate,omitempty"`
    Monitored     bool       `json:"monitored"`
}

// series.go — CreateEpisodeFileInput
type CreateEpisodeFileInput struct {
    Path             string `json:"path"`                // REQUIRED
    Size             int64  `json:"size"`
    Quality          string `json:"quality,omitempty"`
    QualityID        *int64 `json:"qualityId,omitempty"` // pointer — nil means unknown
    VideoCodec       string `json:"videoCodec,omitempty"`
    AudioCodec       string `json:"audioCodec,omitempty"`
    AudioChannels    string `json:"audioChannels,omitempty"`
    DynamicRange     string `json:"dynamicRange,omitempty"`
    Resolution       string `json:"resolution,omitempty"`
    OriginalPath     string `json:"originalPath,omitempty"`
    OriginalFilename string `json:"originalFilename,omitempty"`
}
```

### Quality (`internal/library/quality/`)

```go
// profile.go — package-level function (NOT a method)
func GetQualityByName(name string) (Quality, bool)

// All 17 predefined quality IDs:
// 1=SDTV, 2=DVD, 3=WEBRip-480p, 4=HDTV-720p, 5=WEBRip-720p,
// 6=WEBDL-720p, 7=Bluray-720p, 8=HDTV-1080p, 9=WEBRip-1080p,
// 10=WEBDL-1080p, 11=Bluray-1080p, 12=Remux-1080p, 13=HDTV-2160p,
// 14=WEBRip-2160p, 15=WEBDL-2160p, 16=Bluray-2160p, 17=Remux-2160p
```

### Slots (`internal/library/slots/`)

```go
func (s *Service) IsMultiVersionEnabled(ctx context.Context) bool
func (s *Service) InitializeSlotAssignments(ctx context.Context, mediaType string, mediaID int64) error
// mediaType: "movie" or "episode"

func (s *Service) DetermineTargetSlot(ctx context.Context, parsed *scanner.ParsedMedia, mediaType string, mediaID int64) (*SlotAssignment, error)
// Returns ErrNoMatchingSlot if no slots match

func (s *Service) AssignFileToSlot(ctx context.Context, mediaType string, mediaID, slotID, fileID int64) error
// Also updates file's slot_id column
```

### Progress (`internal/progress/`)

```go
func (m *Manager) StartActivity(id string, activityType ActivityType, title string) *Activity
func (m *Manager) UpdateActivity(id, subtitle string, progress int)  // progress: 0-100, -1=indeterminate
func (m *Manager) CompleteActivity(id, subtitle string)
func (m *Manager) FailActivity(id, errorMsg string)

// ActivityType constants:
const ActivityTypeImport ActivityType = "import"

// WebSocket event types:
// "progress:started", "progress:update", "progress:completed", "progress:error"
```

### Scanner (`internal/library/scanner/`)

```go
func ParsePath(filePath string) *ParsedMedia
// Used for slot matching — parses resolution, codec, source from file path
```

---

## 4. Gotchas & Pitfalls

### Backend

1. **`import` is a Go keyword.** The existing package at `internal/import/` uses `package importer` and is imported as `importer "github.com/slipstream/slipstream/internal/import"`. Our new package is `internal/arrimport/` with `package arrimport` — no conflict.

2. **Sonarr `Seasons` are JSON embedded in the `Series` row**, NOT a separate table. Parse the JSON column. The JSON structure is: `[{"seasonNumber": 1, "monitored": true}, ...]`.

3. **Sonarr `EpisodeFiles.RelativePath` is relative** to the series path. For SQLite reads, compute absolute path as `series.Path + "/" + file.RelativePath`. API reads already provide absolute paths.

4. **Radarr uses a split schema**: `Movies` + `MovieMetadata` tables. Must JOIN on `Movies.MovieMetadataId = MovieMetadata.Id`. The `Title`, `SortTitle`, `Year`, `TmdbId`, `ImdbId`, `Overview`, `Runtime`, `Status`, `Studio`, `Certification`, `InCinemas`, `PhysicalRelease`, `DigitalRelease` fields are on `MovieMetadata`. The `Path`, `QualityProfileId`, `Monitored`, `Added`, `MovieFileId` fields are on `Movies`.

5. **Radarr `Quality` column in `MovieFiles`** is a JSON string. Structure: `{"quality":{"id":7,"name":"Bluray-1080p","source":"bluray","resolution":1080},...}`. Extract `quality.quality.id` and `quality.quality.name`.

6. **Radarr/Sonarr `MediaInfo`** is also JSON. Key fields: `videoCodec`, `audioCodec`, `resolution`, `audioChannels`, `videoDynamicRange` (Radarr) / `videoDynamicRangeType` (Sonarr).

7. **`AudioChannels` type mismatch**: Source is a decimal number (5.1), SlipStream field is a string. Convert with `fmt.Sprintf("%.1f", channels)` or `strconv.FormatFloat(channels, 'f', -1, 64)`.

8. **Date formatting**: Radarr/Sonarr dates are ISO 8601 (e.g., `"2023-06-15T00:00:00Z"`). SlipStream release dates are `"YYYY-MM-DD"` strings. Parse and reformat. `AirDate` for episodes is `*time.Time`.

9. **`EpisodeInput.Title` is a plain `string`**, NOT `sql.NullString`. The service wraps it internally. However, `CreateEpisodeParams.Title` in the database layer IS `sql.NullString` — you only interact with the service layer, so use plain strings.

10. **`CreateSeriesInput.ProductionStatus`** defaults to `"continuing"` if empty. Map directly from Sonarr's `Status` field (values: `"continuing"`, `"ended"`, `"upcoming"`).

11. **Duplicate detection uses errors**: `GetByTmdbID` returns `ErrMovieNotFound`, `GetSeriesByTvdbID` returns `ErrSeriesNotFound`. Check with `errors.Is(err, movies.ErrMovieNotFound)`.

12. **`AddFile`/`AddEpisodeFile` auto-update status**. Do NOT manually set status after creating files — it's handled by the service.

13. **`OriginalFilename`** must be derived from `OriginalFilePath` using `path.Base()`. Do NOT pass the full path.

14. **Multi-version slot flow**: After creating media → `InitializeSlotAssignments` → After creating file → `scanner.ParsePath(file.Path)` → `DetermineTargetSlot(parsed, ...)` → if non-nil → `AssignFileToSlot(...)`. If `DetermineTargetSlot` returns `ErrNoMatchingSlot`, skip slot assignment (file still gets created, just without a slot).

15. **`added_at` preservation requires post-creation UPDATE.** The spec (§2.1 row 19, §3.1 row 18) says `added_at` should be mapped from the source `added` field. However, `CreateMovieInput` and `CreateSeriesInput` do NOT have an `AddedAt` field — the database defaults to `CURRENT_TIMESTAMP`. To preserve the original date, run a direct SQL UPDATE after creation:
    ```go
    // After movies.Service.Create():
    if !movie.Added.IsZero() {
        _, _ = db.ExecContext(ctx, "UPDATE movies SET added_at = ? WHERE id = ?", movie.Added, createdMovie.ID)
    }
    ```
    This requires passing a `*sql.DB` (or `*sqlc.Queries`) to the executor. Alternatively, add `AddedAt *time.Time` to `CreateMovieInput` and modify the service — but the direct UPDATE is less invasive.

16. **`sort_title` is auto-generated.** The spec says "Direct" mapping for `sort_title`, but `movies.Service.Create()` calls `generateSortTitle(input.Title)` and ignores any sort title from the source. This is correct — SlipStream generates its own sort titles. Do NOT try to override this.

17. **Sonarr quality ID 20 maps differently than Radarr ID 20.** Radarr 20=Bluray-480p→DVD(2). Sonarr 20=Bluray-1080p Remux→Remux-1080p(12). The `sonarrQualityMap` must OVERRIDE Radarr's entry for ID 20, not just add to it. Same for ID 21: Radarr 21=Bluray-576p→DVD(2), Sonarr 21=Bluray-2160p Remux→Remux-2160p(17).

### Frontend

18. **Base UI, NOT Radix.** Use `render` prop instead of `asChild`. SelectValue renders the raw value — render labels manually.

19. **No nested ternaries.** Use early returns or component maps.

20. **Hook extraction.** All logic in `use-<feature>.ts`. Component JSX < 50 lines.

21. **`void` prefix** on `queryClient.invalidateQueries(...)` calls.

22. **Never make `onSuccess` async.**

23. **Icons from `lucide-react`.** `ArrowRightLeft` for the "Migrate from *arr" nav icon.

24. **`bun` not `npm`.** Use `bun run lint`, `bun add`, etc.

---

## 5. Test Specifications

### Test 1: Quality Map (Task 17)

**File:** `internal/arrimport/quality_map_test.go`

See §2 above for the full test. Tests every single entry from spec §2.4 and §3.6, including:
- All 18 direct-match Radarr qualities
- 3 approximate-match Radarr qualities (Bluray-480p, Bluray-576p, DVD-R → DVD)
- 9 unmapped Radarr qualities (Unknown, Raw-HD, BR-DISK, WORKPRINT, CAM, etc.) → nil
- 4 Sonarr-only qualities (Bluray-480p, Bluray-1080p Remux, Bluray-2160p Remux, Bluray-576p)
- Cross-validation: shared IDs produce same result in both maps (except overrides for 20, 21)

### Test 2: Executor Movie Import (Task 6)

**File:** `internal/arrimport/executor_test.go`

Test the movie import flow with mock services:

```go
func TestExecuteMovieImport(t *testing.T) {
    // Setup: mock movie service, quality service, slots service
    // Input: SourceMovie with all 20 fields populated, SourceMovieFile with all fields
    // Mappings: root folder mapping, quality profile mapping
    //
    // Verify:
    // 1. movies.Create called with correct CreateMovieInput (all 16 fields mapped)
    // 2. movies.AddFile called with correct CreateMovieFileInput (all 11 fields mapped)
    // 3. QualityID correctly mapped through MapQualityID
    // 4. OriginalFilename = path.Base(OriginalFilePath)
    // 5. Duplicate movies (GetByTmdbID returns non-error) are skipped
    // 6. Movies with deleted status are skipped
    // 7. Progress updates fired
}
```

### Test 3: Executor TV Import (Task 6)

```go
func TestExecuteTVImport(t *testing.T) {
    // Setup: mock tv service, quality service, slots service
    // Input: SourceSeries with seasons, SourceEpisodes, SourceEpisodeFiles
    //
    // Verify:
    // 1. tv.CreateSeries called with correct CreateSeriesInput (all 16 fields)
    // 2. SeasonInput array correctly built from SourceSeasons
    // 3. EpisodeInput correctly populated (title as plain string, AirDate as *time.Time)
    // 4. Episode files linked by matching EpisodeFileID → episode → GetEpisodeByNumber
    // 5. For SQLite: absolute path = series.Path + "/" + file.RelativePath
    // 6. Duplicate series (GetSeriesByTvdbID returns non-error) are skipped
    // 7. ProductionStatus maps from Sonarr status field
}
```

### Test 4: Executor Multi-Version (Task 6)

```go
func TestExecuteWithMultiVersion(t *testing.T) {
    // Setup: IsMultiVersionEnabled returns true
    //
    // Verify:
    // 1. InitializeSlotAssignments called after each movie/episode creation
    // 2. ParsePath called on file path
    // 3. DetermineTargetSlot called with parsed result
    // 4. AssignFileToSlot called when slot found
    // 5. No error when DetermineTargetSlot returns ErrNoMatchingSlot (file still created)
}
```

### Test 5: MapQualityID Fallback (Task 1)

```go
func TestMapQualityIDFallback(t *testing.T) {
    // Verify fallback to GetQualityByName when static map misses
    // Input: sourceType=radarr, qualityID=999 (unmapped), qualityName="Bluray-1080p"
    // Expected: returns pointer to 11 (GetQualityByName("Bluray-1080p") → Quality{ID:11})

    // Verify nil when both map and name fail
    // Input: qualityID=999, qualityName="SomeUnknownQuality"
    // Expected: nil
}
```

---

## 6. Frontend Reference

### Settings Page Template

Every settings/media page follows this exact pattern:

```tsx
import { PageHeader } from '@/components/layout/page-header'
import { MediaNav } from './media-nav'
// ... component import

export function ArrImportPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Media Management"
        description="Configure root folders, quality profiles, version slots, and file naming"
        breadcrumbs={[
          { label: 'Settings', href: '/settings/media' },
          { label: 'Media Management' },
        ]}
      />
      <MediaNav />
      <ArrImportWizard />
    </div>
  )
}
```

### MediaNav — Adding New Tab

Current items array in `web/src/routes/settings/media/media-nav.tsx`:

```tsx
const mediaNavItems = [
  { title: 'Root Folders',      href: '/settings/media/root-folders',       icon: FolderOpen },
  { title: 'Quality Profiles',  href: '/settings/media/quality-profiles',   icon: Sliders    },
  { title: 'Version Slots',     href: '/settings/media/version-slots',      icon: Layers     },
  { title: 'Import & Naming',   href: '/settings/media/file-naming',        icon: FileInput  },
]
```

Add after "Import & Naming":
```tsx
{ title: 'Migrate from *arr', href: '/settings/media/arr-import', icon: ArrowRightLeft },
```

Import `ArrowRightLeft` from `lucide-react`.

### Route Registration

**`web/src/routes-config.tsx`:**
1. Import the page component at the top
2. Export: `export const arrImportRoute = route('/settings/media/arr-import', ArrImportPage)`

**`web/src/router.tsx`:**
1. Import `arrImportRoute` from `@/routes-config`
2. Add to `routeTree` in the `// Media settings` section (after `fileNamingRoute`)

### API Module Pattern

```tsx
// web/src/api/arr-import.ts
import { apiFetch } from './client'
import type { ConnectionConfig, ImportMappings, ImportPreview, SourceRootFolder, SourceQualityProfile } from '@/types/arr-import'

export const arrImportApi = {
  connect: (config: ConnectionConfig) =>
    apiFetch<void>('/arrimport/connect', { method: 'POST', body: JSON.stringify(config) }),
  getSourceRootFolders: () =>
    apiFetch<SourceRootFolder[]>('/arrimport/source/rootfolders'),
  getSourceQualityProfiles: () =>
    apiFetch<SourceQualityProfile[]>('/arrimport/source/qualityprofiles'),
  preview: (mappings: ImportMappings) =>
    apiFetch<ImportPreview>('/arrimport/preview', { method: 'POST', body: JSON.stringify(mappings) }),
  execute: (mappings: ImportMappings) =>
    apiFetch<void>('/arrimport/execute', { method: 'POST', body: JSON.stringify(mappings) }),
  disconnect: () =>
    apiFetch<void>('/arrimport/session', { method: 'DELETE' }),
}
```

### TanStack Query Mutation Pattern

```tsx
import { useMutation, useQueryClient } from '@tanstack/react-query'

export function useConnectSource() {
  return useMutation({
    mutationFn: (config: ConnectionConfig) => arrImportApi.connect(config),
  })
}
```

### WebSocket Progress Listening

Progress events are already handled globally. The frontend just needs to read from the progress store:

```tsx
import { useProgressStore } from '@/stores/progress'

// Inside a component:
const activities = useProgressStore((s) => s.activities)
const importActivity = activities.find((a) => a.id === 'arrimport')
```

### Component Library Quick Reference

| Component | Import | Key Props |
|-----------|--------|-----------|
| `RadioGroup` + `RadioGroupItem` | `@/components/ui/radio-group` | `value`, `onValueChange` |
| `Select` + `SelectTrigger` + `SelectContent` + `SelectItem` | `@/components/ui/select` | `value`, `onValueChange` |
| `Input` | `@/components/ui/input` | Standard HTML input props, `h-8` |
| `Button` | `@/components/ui/button` | `variant`, `size`, `disabled` |
| `Progress` | `@/components/ui/progress` | `value` (0-100), `variant` |
| `ScrollArea` | `@/components/ui/scroll-area` | `className="h-[50vh]"` |
| `Badge` | `@/components/ui/badge` | `variant` |
| `FolderBrowser` | `@/components/forms/folder-browser` | `open`, `onOpenChange`, `initialPath`, `onSelect` |
| `Label` | `@/components/ui/label` | Standard HTML label props |

---

## 7. Per-Task Instructions

### Task 1: Types + Quality Map

**Files to create:**
- `internal/arrimport/types.go`
- `internal/arrimport/quality_map.go`

**Files to read first:**
- `docs/Library-Import-Spec.md` §2.1, §2.2, §3.1-§3.4 (field lists)
- `docs/Library-Import-Spec.md` §2.4, §3.6 (quality mapping tables)
- `internal/library/quality/profile.go` (for `GetQualityByName` import)

**Instructions:**

Create `types.go` with:
- `package arrimport`
- `SourceType` as `string` type with constants `SourceTypeRadarr = "radarr"`, `SourceTypeSonarr = "sonarr"`
- `ConnectionConfig` struct (see plan §1.2)
- All `Source*` entity types (see plan §1.2) — every field from the spec tables must be present
- All `Import*` result types (see plan §1.2)
- `ImportMappings` struct (see plan §2.1)

Create `quality_map.go` with:
- Two unexported map variables: `radarrQualityMap`, `sonarrQualityMap` (`map[int]int64`)
- **CRITICAL**: Sonarr map must start as a COPY of the Radarr map, then OVERRIDE entries for IDs 20, 21, and ADD entries for IDs 13, 22 (see Gotcha #15)
- Unexported helpers: `mapRadarrQualityID(id int) *int64`, `mapSonarrQualityID(id int) *int64`
- Exported: `MapQualityID(sourceType SourceType, sourceQualityID int, sourceQualityName string) *int64`
  - Static map lookup → name fallback via `quality.GetQualityByName` → nil

Every single ID from the spec tables must appear in the maps. Count: Radarr has entries for IDs 1,2,3,4,5,6,7,8,9,12,14,15,16,17,18,19,20,21,23,30,31 (21 entries). Sonarr has all Radarr entries plus overrides/additions for IDs 13,20,21,22.

---

### Task 2: Reader Interface + SQLite Reader (Radarr)

**Files to create:**
- `internal/arrimport/reader.go`
- `internal/arrimport/reader_sqlite.go`

**Files to read first:**
- `internal/arrimport/types.go` (created in Task 1)
- `docs/Library-Import-Spec.md` §1 (data access), §2.1-§2.2 (movie fields)

**Instructions:**

Create `reader.go` with the `Reader` interface (see plan §1.4). Include a `NewReader(cfg ConnectionConfig) (Reader, error)` constructor that dispatches to SQLite or API reader based on config fields.

Create `reader_sqlite.go` with `sqliteReader` struct implementing `Reader`. Use `database/sql` with `modernc.org/sqlite` driver (same as the rest of the project — check `go.mod`). If the project uses a different SQLite driver, match it.

**SQLite connection:** Open with `sql.Open("sqlite", cfg.DBPath+"?mode=ro&_journal_mode=wal")`

**Radarr movie query (split schema):**
```sql
SELECT m.Id, m.Path, m.QualityProfileId, m.Monitored, m.Added, m.MovieFileId,
       mm.Title, mm.SortTitle, mm.Year, mm.TmdbId, mm.ImdbId, mm.Overview,
       mm.Runtime, mm.Status, mm.Studio, mm.Certification,
       mm.InCinemas, mm.PhysicalRelease, mm.DigitalRelease
FROM Movies m
JOIN MovieMetadata mm ON m.MovieMetadataId = mm.Id
```

Skip rows where `mm.Status = -1` (deleted).

**Radarr movie files query:**
```sql
SELECT Id, MovieId, Path, Size, Quality, MediaInfo, OriginalFilePath, DateAdded
FROM MovieFiles WHERE MovieId = ?
```

Parse `Quality` JSON: extract `.quality.id` (int) and `.quality.name` (string).
Parse `MediaInfo` JSON: extract `.videoCodec`, `.audioCodec`, `.resolution`, `.audioChannels` (float→string), `.videoDynamicRange`.

**Root folders:** `SELECT Id, Path FROM RootFolders`
**Quality profiles:** `SELECT Id, Name FROM QualityProfiles`

For Radarr, `ReadSeries`/`ReadEpisodes`/`ReadEpisodeFiles` should return empty slices (Radarr has no TV data). `ReadMovies` returns the populated list.

**Derive `RootFolderPath`:** Radarr movies store `Path` (e.g., `/movies/The Dark Knight (2008)`). The root folder path is the parent — find the matching root folder from `ReadRootFolders` that is a prefix of the movie path.

**Validate:** Open the DB, run `SELECT name FROM sqlite_master WHERE type='table' AND name='Movies'` to verify it's a Radarr DB.

---

### Task 3: SQLite Reader (Sonarr Extension)

**Files to modify:**
- `internal/arrimport/reader_sqlite.go`

**Files to read first:**
- `docs/Library-Import-Spec.md` §3.1-§3.4 (TV fields)
- Existing `reader_sqlite.go` from Task 2

**Instructions:**

Add Sonarr-specific queries to `sqliteReader`:

**Series query:**
```sql
SELECT Id, Title, SortTitle, Year, TvdbId, TmdbId, ImdbId, Overview,
       Runtime, Path, QualityProfileId, Monitored, SeasonFolder,
       Status, Network, SeriesType, Certification, Added, Seasons
FROM Series
```

Skip rows where `Status = -1` (deleted).

Parse `Seasons` JSON column: `[{"seasonNumber":1,"monitored":true},...]` → `[]SourceSeason`.

**Derive `RootFolderPath`:** Same approach as Radarr — match against root folders.

**Episodes query:**
```sql
SELECT Id, SeriesId, SeasonNumber, EpisodeNumber, Title, Overview,
       AirDateUtc, Monitored, EpisodeFileId
FROM Episodes WHERE SeriesId = ?
```

`EpisodeFileId > 0` means `HasFile = true`. `AirDateUtc` may be empty string or NULL — handle both.

**Episode files query:**
```sql
SELECT Id, SeriesId, SeasonNumber, RelativePath, Size, Quality, MediaInfo,
       OriginalFilePath, DateAdded
FROM EpisodeFiles WHERE SeriesId = ?
```

Parse `Quality` and `MediaInfo` JSON same as Radarr.

**IMPORTANT:** `RelativePath` is relative — store as-is. Do NOT prepend series path. The executor handles path construction.

**Validate:** Check for `SELECT name FROM sqlite_master WHERE type='table' AND name='Series'` for Sonarr.

For Sonarr, `ReadMovies` should return empty slice. `ReadSeries`/`ReadEpisodes`/`ReadEpisodeFiles` return populated data.

---

### Task 4: API Reader

**Files to create:**
- `internal/arrimport/reader_api.go`

**Files to read first:**
- `internal/arrimport/types.go` (Task 1)
- `internal/arrimport/reader.go` (Task 2)
- `docs/Library-Import-Spec.md` §1 (API endpoints)

**Instructions:**

Create `apiReader` struct implementing `Reader`. Uses `net/http.Client` with `X-Api-Key` header.

**Radarr API endpoints:**
- `GET /api/v3/movie` → array of movie objects (includes embedded `movieFile` when `hasFile=true`)
- `GET /api/v3/qualityprofile` → array of profiles
- `GET /api/v3/rootfolder` → array of root folders
- `GET /api/v3/system/status` → `{"appName":"Radarr",...}` for validation

**Sonarr API endpoints:**
- `GET /api/v3/series` → array of series (includes embedded `seasons` array)
- `GET /api/v3/episode?seriesId={id}` → array of episodes for a series
- `GET /api/v3/episodefile?seriesId={id}` → array of episode files for a series
- `GET /api/v3/qualityprofile` → array of profiles
- `GET /api/v3/rootfolder` → array of root folders
- `GET /api/v3/system/status` → `{"appName":"Sonarr",...}` for validation

**API JSON field names** match the spec's field names directly (camelCase). The API response for episode files includes an absolute `path` field (not relative like SQLite).

**Validate:** `GET /api/v3/system/status` — check `appName` matches expected source type.

For movie files from Radarr API: the movie object includes a `movieFile` field when `hasFile=true`. Quality info is nested as `movieFile.quality.quality.id` and `movieFile.quality.quality.name`. MediaInfo is `movieFile.mediaInfo.*`.

---

### Task 5: Service + Preview

**Files to create:**
- `internal/arrimport/service.go`

**Files to read first:**
- `internal/arrimport/types.go` (Task 1)
- `internal/arrimport/reader.go` (Task 2)
- `internal/library/movies/service.go` — for `GetByTmdbID` signature
- `internal/library/tv/service.go` — for `GetSeriesByTvdbID` signature
- `internal/progress/progress.go` — for `Manager` type and methods

**Instructions:**

Create the `Service` struct with dependencies (see plan §2.1). Follow the existing pattern of constructor + optional setter methods. Include a `*sql.DB` field for direct queries needed by the executor (e.g., `added_at` preservation).

`NewService(db, movieService, tvService, rootFolderService, qualityService, progressManager, hub, logger)` — required deps in constructor.
`SetSlotsService(svc)` — optional setter (same pattern as `importService.SetSlotsService`).

**Connect:** Create reader via `NewReader(cfg)`, call `reader.Validate(ctx)`, store reader and sourceType in service.

**Preview logic:**
1. Read all source data via reader
2. For each movie: check `movieService.GetByTmdbID(ctx, tmdbID)` — if `errors.Is(err, movies.ErrMovieNotFound)` → "new", else → "duplicate"
3. For each series: check `tvService.GetSeriesByTvdbID(ctx, tvdbID)` — if `errors.Is(err, tv.ErrSeriesNotFound)` → "new", else → "duplicate"
4. Build `ImportPreview` with summary stats

**Disconnect:** Close reader, clear session state.

Thread-safety: Use `sync.Mutex` to protect session state (reader, sourceType). Only one import session at a time.

---

### Task 6: Executor (USE OPUS)

**Files to create:**
- `internal/arrimport/executor.go`

**Files to read first:**
- `internal/arrimport/types.go` (Task 1)
- `internal/arrimport/quality_map.go` (Task 1)
- `internal/library/movies/movie.go` — for `CreateMovieInput`, `CreateMovieFileInput` struct definitions
- `internal/library/tv/series.go` — for `CreateSeriesInput`, `CreateEpisodeFileInput`, `SeasonInput`, `EpisodeInput`
- `internal/library/tv/episodes.go` — for `GetEpisodeByNumber`, `AddEpisodeFile`
- `internal/library/slots/assignment.go` — for `DetermineTargetSlot`, `AssignFileToSlot`
- `internal/library/slots/status.go` — for `InitializeSlotAssignments`
- `internal/library/scanner/parser.go` — for `ParsePath`
- `internal/progress/progress.go` — for progress reporting

**Instructions:**

This is the most complex file. Create an `Executor` struct that receives all service dependencies, a `*sql.DB` for direct queries (needed for `added_at` preservation), and `ImportMappings`.

**Movie import flow (per movie):**

```go
func (e *Executor) importMovie(ctx context.Context, movie SourceMovie, mappings ImportMappings) error {
    // 1. Skip deleted
    // 2. Skip if GetByTmdbID doesn't return ErrMovieNotFound
    // 3. Resolve root folder ID: mappings.RootFolderMapping[movie.RootFolderPath]
    // 4. Resolve quality profile ID: mappings.QualityProfileMapping[movie.QualityProfileID]
    // 5. Create movie:
    input := &movies.CreateMovieInput{
        Title:                 movie.Title,
        Year:                  movie.Year,
        TmdbID:                movie.TmdbID,
        ImdbID:                movie.ImdbID,
        Overview:              movie.Overview,
        Runtime:               movie.Runtime,
        Path:                  movie.Path,
        RootFolderID:          rootFolderID,
        QualityProfileID:      qualityProfileID,
        Monitored:             movie.Monitored,
        Studio:                movie.Studio,
        ContentRating:         movie.Certification,
        ReleaseDate:           formatDate(movie.DigitalRelease),    // "YYYY-MM-DD"
        PhysicalReleaseDate:   formatDate(movie.PhysicalRelease),
        TheatricalReleaseDate: formatDate(movie.InCinemas),
    }
    createdMovie, err := e.movieService.Create(ctx, input)

    // 5b. Preserve original added_at date (spec §2.1 row 19)
    //     CREATE doesn't support AddedAt, so UPDATE directly
    if !movie.Added.IsZero() {
        _, _ = e.db.ExecContext(ctx, "UPDATE movies SET added_at = ? WHERE id = ?",
            movie.Added, createdMovie.ID)
    }

    // 6. Multi-version: InitializeSlotAssignments(ctx, "movie", createdMovie.ID)

    // 7. If movie.File != nil (has file):
    qualityID := MapQualityID(SourceTypeRadarr, movie.File.QualityID, movie.File.QualityName)
    fileInput := &movies.CreateMovieFileInput{
        Path:             movie.File.Path,
        Size:             movie.File.Size,
        QualityID:        qualityID,           // *int64, can be nil
        VideoCodec:       movie.File.VideoCodec,
        AudioCodec:       movie.File.AudioCodec,
        AudioChannels:    movie.File.AudioChannels,
        DynamicRange:     movie.File.DynamicRange,
        Resolution:       movie.File.Resolution,
        OriginalPath:     movie.File.OriginalFilePath,
        OriginalFilename: path.Base(movie.File.OriginalFilePath),
    }
    createdFile, err := e.movieService.AddFile(ctx, createdMovie.ID, fileInput)

    // 8. Multi-version slot assignment:
    //    parsed := scanner.ParsePath(movie.File.Path)
    //    slot, err := e.slotsService.DetermineTargetSlot(ctx, parsed, "movie", createdMovie.ID)
    //    if err == nil: e.slotsService.AssignFileToSlot(ctx, "movie", createdMovie.ID, slot.SlotID, createdFile.ID)
}
```

**TV series import flow (per series):**

```go
func (e *Executor) importSeries(ctx context.Context, series SourceSeries, episodes []SourceEpisode, files []SourceEpisodeFile, mappings ImportMappings) error {
    // 1. Skip deleted status
    // 2. Skip if GetSeriesByTvdbID doesn't return ErrSeriesNotFound
    // 3. Resolve root folder and quality profile from mappings
    // 4. Build SeasonInput array:
    //    - Group episodes by season
    //    - For each SourceSeason: create SeasonInput with Monitored flag
    //    - For each episode in that season: create EpisodeInput with:
    //      - EpisodeNumber, Title (plain string), Overview
    //      - AirDate: parse AirDateUtc string to *time.Time (nil if empty/null)
    //      - Monitored

    input := &tv.CreateSeriesInput{
        Title:            series.Title,
        Year:             series.Year,
        TvdbID:           series.TvdbID,
        TmdbID:           series.TmdbID,
        ImdbID:           series.ImdbID,
        Overview:         series.Overview,
        Runtime:          series.Runtime,
        Path:             series.Path,
        RootFolderID:     rootFolderID,
        QualityProfileID: qualityProfileID,
        Monitored:        series.Monitored,
        SeasonFolder:     series.SeasonFolder,
        Network:          series.Network,
        FormatType:       series.SeriesType,          // "standard", "daily", "anime"
        ProductionStatus: series.Status,               // "continuing", "ended", "upcoming"
        Seasons:          seasonInputs,
    }
    createdSeries, err := e.tvService.CreateSeries(ctx, input)

    // 4b. Preserve original added_at date (spec §3.1 row 18)
    if !series.Added.IsZero() {
        _, _ = e.db.ExecContext(ctx, "UPDATE series SET added_at = ? WHERE id = ?",
            series.Added, createdSeries.ID)
    }

    // 5. Multi-version: for each created episode, InitializeSlotAssignments
    //    (need to iterate episodes of the created series)

    // 6. For each SourceEpisodeFile:
    //    a. Find the source episode that has EpisodeFileID == file.ID
    //    b. Look up SlipStream episode: GetEpisodeByNumber(seriesID, seasonNum, episodeNum)
    //    c. Compute path:
    //       - If SQLite: series.Path + "/" + file.RelativePath
    //       - If API: file path is already absolute (store as-is)
    //    d. Map quality ID
    //    e. Call AddEpisodeFile
    //    f. Multi-version: DetermineTargetSlot → AssignFileToSlot
}
```

**CRITICAL:** Build a map from source `EpisodeFileID` → `SourceEpisode` for efficient lookup when matching files to episodes.

**Progress reporting:**
```go
activity := e.progressManager.StartActivity("arrimport", progress.ActivityTypeImport, "Library Import")
// Loop: e.progressManager.UpdateActivity("arrimport", fmt.Sprintf("Importing: %s", title), pct)
// Done: e.progressManager.CompleteActivity("arrimport", fmt.Sprintf("Import complete: %d movies, %d series", ...))
// Error: e.progressManager.FailActivity("arrimport", err.Error())
```

---

### Task 7: Handlers

**Files to create:**
- `internal/arrimport/handlers.go`

**Files to read first:**
- `internal/arrimport/service.go` (Task 5)
- `internal/arrimport/types.go` (Task 1)
- `internal/import/handlers.go` — for pattern reference (existing handler style)

**Instructions:**

Follow the established `Handlers` + `NewHandlers` + `RegisterRoutes` pattern:

```go
type Handlers struct {
    service *Service
}

func NewHandlers(service *Service) *Handlers {
    return &Handlers{service: service}
}

func (h *Handlers) RegisterRoutes(g *echo.Group) {
    g.POST("/connect", h.Connect)
    g.GET("/source/rootfolders", h.GetSourceRootFolders)
    g.GET("/source/qualityprofiles", h.GetSourceQualityProfiles)
    g.POST("/preview", h.Preview)
    g.POST("/execute", h.Execute)
    g.DELETE("/session", h.Disconnect)
}
```

Each handler: parse request body (if any) → call service method → return JSON response. Use `echo.Context` for binding and response. Return `http.StatusOK` for success, `http.StatusBadRequest` for validation errors, `http.StatusInternalServerError` for service errors.

`Execute` should return `http.StatusAccepted` (202) since it starts an async operation.

---

### Task 8: Server Wiring + Route Registration

**Files to modify:**
- `internal/api/server.go`
- `internal/api/routes.go`

**Files to read first:**
- Both files above (to find exact insertion points)
- `internal/arrimport/handlers.go` (Task 7)

**Instructions:**

In `server.go`:
1. Add import: `"github.com/slipstream/slipstream/internal/arrimport"`
2. Add field to `Server` struct: `arrImportService *arrimport.Service`
3. In `NewServer`, after other service initializations:
   ```go
   s.arrImportService = arrimport.NewService(db, s.movieService, s.tvService, s.rootFolderService,
       s.qualityService, s.progressManager, s.hub, &logger)
   s.arrImportService.SetSlotsService(s.slotsService)
   ```
   Use the same `db` instance available in `NewServer` (check how other services get their `*sql.DB` — e.g., `s.dbManager.GetDB()` or `s.startupDB`).

In `routes.go`:
1. Add a new setup function or inline in existing setup:
   ```go
   arrImportHandlers := arrimport.NewHandlers(s.arrImportService)
   arrImportHandlers.RegisterRoutes(protected.Group("/arrimport"))
   ```
   Place this in the `protected` group (admin-only).

---

### Task 9: Frontend Types + API Module

**Files to create:**
- `web/src/types/arr-import.ts`
- `web/src/api/arr-import.ts`

**Files to read first:**
- `web/src/api/client.ts` — for `apiFetch` import path
- `web/src/api/index.ts` — to add re-export
- `web/src/types/index.ts` — to add re-export (if exists)

**Instructions:**

Create TypeScript types matching the backend types from Task 1. Use `camelCase` field names (matching JSON tags).

Create API module following the exact pattern from `web/src/api/quality-profiles.ts` (see §6).

Add re-export to `web/src/api/index.ts`:
```ts
export { arrImportApi } from './arr-import'
```

---

### Task 10: Frontend Route + Nav

**Files to modify:**
- `web/src/routes/settings/media/media-nav.tsx`
- `web/src/routes-config.tsx`
- `web/src/router.tsx`

**Files to create:**
- `web/src/routes/settings/media/arr-import.tsx`

**Files to read first:**
- All three files to modify (to find exact insertion points)
- `web/src/routes/settings/media/file-naming.tsx` — for page pattern reference

**Instructions:**

1. Create `arr-import.tsx` page shell (see §6 template)
2. Add nav item to `media-nav.tsx` (add `ArrowRightLeft` import from `lucide-react`)
3. Import page + add route in `routes-config.tsx`
4. Add to `routeTree` in `router.tsx`

For now, the `ArrImportWizard` component can be a placeholder div. It will be created in Tasks 12-16.

---

### Task 11: Frontend Hooks

**Files to create:**
- `web/src/hooks/use-arr-import.ts`

**Files to read first:**
- `web/src/api/arr-import.ts` (Task 9)
- `web/src/types/arr-import.ts` (Task 9)
- `web/src/hooks/use-movies.ts` — for mutation pattern reference
- `web/src/stores/progress.ts` — for progress store usage

**Instructions:**

Create a custom hook managing the wizard state machine:

```tsx
type WizardStep = 'connect' | 'mapping' | 'preview' | 'importing' | 'report'

export function useArrImport() {
  const [step, setStep] = useState<WizardStep>('connect')
  // ... state for each step's data

  const connectMutation = useMutation({ mutationFn: arrImportApi.connect })
  const previewMutation = useMutation({ mutationFn: arrImportApi.preview })
  const executeMutation = useMutation({ mutationFn: arrImportApi.execute })

  // Progress from global store
  const activities = useProgressStore((s) => s.activities)
  const importActivity = activities.find((a) => a.id === 'arrimport')

  // Step transition handlers
  // Data fetching for source root folders + quality profiles after connect

  return { step, /* ...all state and handlers */ }
}
```

---

### Tasks 12-16: Wizard Components

**Files to create:**
- `web/src/components/arr-import/index.tsx` (Task 16)
- `web/src/components/arr-import/connect-step.tsx` (Task 12)
- `web/src/components/arr-import/mapping-step.tsx` (Task 13)
- `web/src/components/arr-import/preview-step.tsx` (Task 14)
- `web/src/components/arr-import/import-step.tsx` (Task 15)

**Files to read first (for each component task):**
- `web/src/hooks/use-arr-import.ts` (Task 11)
- `web/src/components/slots/DryRunModal/` — for preview patterns (Task 14)
- `web/src/components/forms/folder-browser.tsx` — for file selection (Task 12)
- `web/src/components/ui/` — relevant UI components

Each component receives state/handlers from the parent wizard shell via props. Keep JSX under 50 lines per component — extract logic into the hook.

**Connect Step (Task 12):**
- RadioGroup for source type (Radarr/Sonarr)
- RadioGroup for connection method (SQLite/API)
- Conditional: SQLite → Input + FolderBrowser, API → URL Input + API Key Input
- Connect button fires mutation

**Mapping Step (Task 13):**
- Two sections: Root Folder Mapping, Quality Profile Mapping
- Each row: source item (read-only) + Select dropdown for SlipStream equivalent
- Auto-match logic: compare paths/names
- "Next" disabled until all mappings set

**Preview Step (Task 14):**
- Summary cards (Total, New, Duplicates, Skipped)
- Filtered list of movies/series with status badges
- "Start Import" button

**Import Step (Task 15):**
- Progress bar reading from `useProgressStore`
- Current item display
- On completion: show ImportReport summary cards + error list

**Wizard Shell (Task 16):**
- Step router: renders the active step component
- Passes state from `useArrImport()` hook
- Step indicator/breadcrumbs (optional)

---

### Task 18: Lint + Cleanup

Run:
```bash
make lint       # Go linting
cd web && bun run lint  # Frontend linting
```

Fix any issues introduced by the new code. Common fixes:
- Unused imports
- Missing error checks (Go)
- TypeScript type errors
- ESLint rule violations

---

## Appendix: File Checklist

Backend files (8 new, 2 modified):
- [ ] `internal/arrimport/types.go`
- [ ] `internal/arrimport/quality_map.go`
- [ ] `internal/arrimport/quality_map_test.go`
- [ ] `internal/arrimport/reader.go`
- [ ] `internal/arrimport/reader_sqlite.go`
- [ ] `internal/arrimport/reader_api.go`
- [ ] `internal/arrimport/service.go`
- [ ] `internal/arrimport/executor.go`
- [ ] `internal/arrimport/handlers.go`
- [ ] `internal/api/server.go` (modified)
- [ ] `internal/api/routes.go` (modified)

Frontend files (9 new, 3 modified):
- [ ] `web/src/types/arr-import.ts`
- [ ] `web/src/api/arr-import.ts`
- [ ] `web/src/hooks/use-arr-import.ts`
- [ ] `web/src/components/arr-import/index.tsx`
- [ ] `web/src/components/arr-import/connect-step.tsx`
- [ ] `web/src/components/arr-import/mapping-step.tsx`
- [ ] `web/src/components/arr-import/preview-step.tsx`
- [ ] `web/src/components/arr-import/import-step.tsx`
- [ ] `web/src/routes/settings/media/arr-import.tsx`
- [ ] `web/src/routes/settings/media/media-nav.tsx` (modified)
- [ ] `web/src/routes-config.tsx` (modified)
- [ ] `web/src/router.tsx` (modified)
