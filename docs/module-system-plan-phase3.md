## Phase 3: Metadata

**Goal:** Extract the monolithic `internal/metadata/Service` into module-provided `MetadataProvider` implementations. After this phase, each module owns its metadata search, fetch, and refresh logic. The shared `metadata.Service` becomes a thin delegation layer that routes to the correct module, or the framework calls module providers directly.

**Spec sections covered:** §3.1 (MetadataProvider interface: `Search`, `GetByID`, `GetExtendedInfo`, `RefreshMetadata`), §3.2 (External IDs generalization: `GetExternalIDs() map[string]string`), §18.5 (Metadata refresh with existing data: structured diff, preservation rules for existing children)

**Depends on:** Phase 0 (module interfaces defined, registry exists), Phase 2 (module-scoped quality profiles — needed because add-media flows reference quality profiles)

**Required Phase 0 types (must exist before Phase 3 tasks compile):**
- `module.Type` type with constants `module.TypeMovie`, `module.TypeTV`
- `module.EntityType` type with constants `module.EntityMovie`, `module.EntitySeries`, `module.EntitySeason`, `module.EntityEpisode`
- `module.Module` interface (must embed `module.MetadataProvider` so `Registry.Get().RefreshMetadata()` works)
- `module.MetadataProvider` interface (defined in Task 0.3, verified/updated in Task 3.2)
- `module.Registry` type with `Register(Module)` and `Get(Type) Module` methods
- `module.SearchOptions`, `module.SearchResult`, `module.MediaMetadata`, `module.ExtendedMetadata` (skeleton types from Task 0.4, fleshed out in Task 3.1)

### Phase 3 Architecture Decisions

1. **Metadata service stays, gains module dispatch:** The existing `internal/metadata/Service` is NOT removed — it continues to own the TMDB/TVDB/OMDb API clients, caching, provider health registration, and artwork coordination. What changes is that the `MetadataProvider` interface methods are implemented on the movie and TV module stubs, and these implementations **delegate to the existing metadata service methods**. This is a refactor-from-the-outside-in approach: the module interface wraps the existing service rather than rewriting it. This minimizes risk in this phase. **Spec deviation note:** Spec §3.1 states "modules own their provider logic entirely, including API clients, rate limiting, and caching." Phase 3 intentionally deviates from this — modules delegate to the shared `metadata.Service` rather than owning clients directly. This is the correct intermediate architecture for an incremental migration. Full module ownership of API clients (each module creating/managing its own TMDB/TVDB/OMDb client instances) is deferred to Phase 11 (Final Cleanup) or may be retained as the permanent architecture if the delegation approach proves sufficient.

2. **`MetadataProvider` implementations live in module packages:** `internal/modules/movie/metadata.go` and `internal/modules/tv/metadata.go`. Each receives `*metadata.Service` (and optionally `*metadata.ArtworkDownloader`) via constructor injection. The module method implementations call through to the shared service's existing methods (e.g., `SearchMovies`, `GetMovie`, `GetExtendedMovie`, `GetSeriesSeasons`).

3. **`RefreshResult` structured diff — new concept:** Today, `RefreshMovieMetadata` and `RefreshSeriesMetadata` in `librarymanager/metadata_refresh.go` perform the refresh and return the updated entity directly. The spec §18.5 requires a structured diff (`added`, `updated`, `removed` entities). We introduce a `RefreshResult` type in the module package that captures this diff. For movies (flat structure), this is trivial — just "updated fields". For TV, the diff is computed by comparing existing seasons/episodes with metadata results and categorizing them as added/updated/removed. The **upsert-based preservation** already exists in `UpsertSeason`/`UpsertEpisode` (they don't overwrite `monitored` or `status` on conflict) — the diff is an additional reporting layer, not a behavioral change.

4. **Refresh moves from librarymanager to module providers:** `RefreshMovieMetadata` and `RefreshSeriesMetadata` currently live in `internal/library/librarymanager/metadata_refresh.go`. They contain module-specific logic (best-match selection, field mapping, release date fetching, artwork downloading, season/episode upsert). These functions are extracted into the module `MetadataProvider` implementations. The librarymanager gains a generic `RefreshMetadata(moduleType, entityID)` method that dispatches to the correct module's `RefreshMetadata`. **However**, the artwork download and folder scan operations remain in the librarymanager since they are framework concerns, not module concerns. The module's `RefreshMetadata` returns a `RefreshResult` and the librarymanager handles post-refresh actions (artwork, scan, broadcast).

5. **`GetExternalIDs()` on library entities — not on metadata results:** The spec §3.2 says `SearchableItem` gets `GetExternalIDs() map[string]string`. This is implemented on the **library entity level**, not on metadata provider types. Each module adds a method to its entity types (or implements the `SearchableItem` interface in Phase 5). In Phase 3, we add `ExternalIDs() map[string]string` methods to `movies.Movie` and `tv.Series` as utilities. These return `{"tmdb": "12345", "imdb": "tt1234567"}` etc. This is a small, additive change.

6. **Metadata handlers stay in `internal/metadata/handlers.go`:** The HTTP API endpoints (`/metadata/movie/search`, `/metadata/series/search`, etc.) are unchanged in this phase. They still call `metadata.Service` methods directly. Migrating them to call through module providers happens in Phase 5 (RouteProvider) or later when modules register their own routes. For now, the existing API is preserved.

7. **Metadata search in add-media flows — unchanged:** The librarymanager's `AddMovie`/`AddSeries` methods call `metadata.Service.GetMovieReleaseDates`, `GetMovieContentRating`, `GetSeriesSeasons`, etc. These calls are NOT migrated in Phase 3 — the add-media flow continues using `metadata.Service` directly. Migration to module providers happens when the librarymanager becomes module-aware (Phase 6+). The priority in Phase 3 is establishing the interface contract and implementing it for refresh, which is the most complex metadata operation.

8. **Dev mode mocks — deferred:** The `MockFactory.CreateMockMetadataProvider()` interface method was defined in Phase 0. Actual implementation is deferred to Phase 9 (Dev Mode). The movie and TV module stubs return `nil` from `CreateMockMetadataProvider()` for now. The existing `internal/metadata/mock/` package continues to work unchanged via `metadata.Service.SetClients()`.

9. **No database schema changes in Phase 3:** All metadata columns already exist on the movie, series, season, and episode tables. No migrations needed. The work is purely Go code restructuring.

10. **Artwork coordination note:** Artwork downloading is tightly coupled to the metadata result types (`MovieResult`, `SeriesResult`). The `MetadataProvider.RefreshMetadata` returns enough data in `RefreshResult` for the caller (librarymanager) to trigger artwork downloads. The module does NOT download artwork itself — it returns the metadata (including poster/backdrop URLs) and the framework handles artwork.

11. **New episode monitoring inheritance — known gap:** Spec §18.5 says new child entities should "inherit monitoring state from the parent's monitoring preset (or from the parent's current `monitored` value)." Currently, `UpsertEpisode` hardcodes `monitored=true` for new episodes. This is the **existing behavior** and is preserved as-is in Phase 3. The proper fix (inheriting from parent preset) requires the monitoring infrastructure from Phase 4 (`MonitoringPresets`). A spec-conformant implementation will be addressed in Phase 4 when the monitoring cascade system is built.

### Phase 3 Dependency Graph

```
Task 3.1 (RefreshResult type + ExternalIDs) ─────→ Task 3.3 (movie MetadataProvider)
                                                          ↓
Task 3.2 (flesh out Phase 0 metadata types) ──→ Task 3.4 (TV MetadataProvider)
  (can start in parallel with 3.1)                        ↓
                                              Task 3.5 (librarymanager dispatch)
                                                          ↓
                                              Task 3.6 (update callers & wire)
                                                          ↓
                                              Task 3.7 (phase validation)
```

**Parallelism:** Tasks 3.1 and 3.2 can run in parallel. Tasks 3.3 and 3.4 can run in parallel once 3.1 and 3.2 are complete (they touch different module packages with no file overlap).

---

### Task 3.1: Define RefreshResult Type and ExternalIDs Utility ✅

**Create** `internal/module/metadata_types.go`

This task fleshes out the metadata-related types in the module package that were left as minimal skeletons in Phase 0.

**`RefreshResult`** — the structured diff returned by `MetadataProvider.RefreshMetadata`:

```go
package module

// RefreshResult contains the structured diff from a metadata refresh operation.
// The framework uses this to determine post-refresh actions (artwork download, status updates, broadcasting).
type RefreshResult struct {
    // EntityID is the root entity that was refreshed.
    EntityID int64

    // Updated is true if any metadata fields changed on the root entity.
    Updated bool

    // FieldsChanged lists which fields were updated (for logging/auditing).
    FieldsChanged []string

    // ChildrenAdded lists child entities that were newly created from metadata.
    // For TV: new seasons/episodes discovered. For flat modules (Movie): always empty.
    ChildrenAdded []RefreshChildEntry

    // ChildrenUpdated lists child entities whose metadata was updated.
    ChildrenUpdated []RefreshChildEntry

    // ChildrenRemoved lists child entities present in the library but absent from metadata.
    // Per spec §18.5, these are NOT auto-deleted — they are flagged for user review.
    ChildrenRemoved []RefreshChildEntry

    // ArtworkURLs contains the poster/backdrop/logo URLs from the refreshed metadata,
    // allowing the caller to trigger artwork downloads without re-fetching metadata.
    ArtworkURLs ArtworkURLs

    // Metadata contains the raw metadata result for additional caller use.
    // The concrete type depends on the module (MovieResult, SeriesResult, etc.)
    Metadata any
}

// RefreshChildEntry represents a child entity in a refresh diff.
type RefreshChildEntry struct {
    EntityType EntityType
    // Identifier is the natural key of the child (e.g., "S01E05" for TV, track number for music).
    Identifier string
    // EntityID is the database ID if the entity exists (0 for newly added from metadata).
    EntityID int64
    // Title of the child entity.
    Title string
}

// ArtworkURLs holds URLs for artwork that should be downloaded after a metadata refresh.
type ArtworkURLs struct {
    PosterURL     string
    BackdropURL   string
    LogoURL       string
    StudioLogoURL string // Movie-specific
}
```

**Also update** `internal/module/search_types.go` (created in Phase 0 Task 0.4) — flesh out `SearchResult` and `MediaMetadata` types:

```go
// SearchOptions configures a metadata search request.
type SearchOptions struct {
    Year int    // Filter by year (0 = no filter)
    Limit int   // Max results (0 = provider default)
}

// SearchResult represents a single result from a metadata search.
type SearchResult struct {
    ExternalID  string            // Primary external ID value (e.g., TMDB ID as string)
    Title       string
    Year        int
    Overview    string
    PosterURL   string
    BackdropURL string
    ExternalIDs map[string]string // All external IDs (e.g., {"tmdb": "550", "imdb": "tt0137523"})
    Extra       any               // Module-specific extra data (cast to module's result type)
}

// MediaMetadata represents full metadata for an entity fetched by external ID.
type MediaMetadata struct {
    ExternalID  string
    Title       string
    Year        int
    Overview    string
    PosterURL   string
    BackdropURL string
    ExternalIDs map[string]string
    Extra       any // Module-specific (e.g., release dates, runtime, studio, genres, network)
}

// ExtendedMetadata represents additional metadata (credits, ratings, etc.)
type ExtendedMetadata struct {
    Credits       any // Module-specific credits structure
    Ratings       any // Module-specific ratings
    ContentRating string
    TrailerURL    string
    Extra         any // Module-specific extended data
}
```

**Also add** `ExternalIDs()` methods to movie and TV entity types:

**Modify** `internal/library/movies/movie.go` — add method:

```go
// ExternalIDs returns a map of external ID provider names to their values.
func (m *Movie) ExternalIDs() map[string]string {
    ids := make(map[string]string)
    if m.TmdbID > 0 {
        ids["tmdb"] = strconv.Itoa(m.TmdbID)
    }
    if m.ImdbID != "" {
        ids["imdb"] = m.ImdbID
    }
    if m.TvdbID > 0 {
        ids["tvdb"] = strconv.Itoa(m.TvdbID)
    }
    return ids
}
```

**Modify** `internal/library/tv/service.go` (or create `internal/library/tv/series.go` if `Series` struct is defined there) — add method to `Series`:

```go
// ExternalIDs returns a map of external ID provider names to their values.
func (s *Series) ExternalIDs() map[string]string {
    ids := make(map[string]string)
    if s.TvdbID > 0 {
        ids["tvdb"] = strconv.Itoa(s.TvdbID)
    }
    if s.TmdbID > 0 {
        ids["tmdb"] = strconv.Itoa(s.TmdbID)
    }
    if s.ImdbID != "" {
        ids["imdb"] = s.ImdbID
    }
    return ids
}
```

**Implementation notes for the subagent:**
- Read `internal/module/search_types.go` first (created in Phase 0) — it has skeleton types. Update them in place, don't create a new file.
- Read `internal/library/movies/movie.go` to see the `Movie` struct — add the `ExternalIDs()` method there. Import `strconv`.
- Read `internal/library/tv/service.go` to find where `Series` is defined — it may be in a separate `types.go` or `series.go`. Add the `ExternalIDs()` method on that struct. Import `strconv`.
- The `RefreshResult.Metadata` field uses `any` to avoid the module package depending on `internal/metadata` types. Callers type-assert to the concrete type.

**Verify:**
- `go build ./internal/module/...` compiles
- `go build ./internal/library/movies/...` compiles
- `go build ./internal/library/tv/...` compiles
- `go vet ./internal/module/... ./internal/library/movies/... ./internal/library/tv/...` clean

---

### Task 3.2: Flesh Out Phase 0 MetadataProvider Interface Types ✅

**Depends on:** Task 3.1 (RefreshResult type must be defined first)

**Modify** `internal/module/interfaces.go` — verify the `MetadataProvider` interface matches the spec and uses the types from Task 3.1:

```go
// MetadataProvider handles search and metadata retrieval for a module. (§3.1)
type MetadataProvider interface {
    Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
    GetByID(ctx context.Context, externalID string) (*MediaMetadata, error)
    GetExtendedInfo(ctx context.Context, externalID string) (*ExtendedMetadata, error)
    RefreshMetadata(ctx context.Context, entityID int64) (*RefreshResult, error)
}
```

This should already be defined from Phase 0 Task 0.3. Verify the method signatures match and the types referenced (`SearchOptions`, `SearchResult`, `MediaMetadata`, `ExtendedMetadata`, `RefreshResult`) are all defined and compile-compatible.

**Implementation notes for the subagent:**
- Read `internal/module/interfaces.go` and verify the `MetadataProvider` interface. If it matches, no changes needed — just verify compilation.
- If the return types were left as placeholder names in Phase 0 that don't match the types defined in Task 3.1, update the interface to reference the correct types.
- Run `go build ./internal/module/...` to verify.

**Verify:**
- `go build ./internal/module/...` compiles with zero errors.

---

### Task 3.3: Implement Movie MetadataProvider ✅

**Depends on:** Tasks 3.1 and 3.2

**Create** `internal/modules/movie/metadata.go`

This implements `module.MetadataProvider` for the movie module. It wraps the existing `metadata.Service` methods.

```go
package movie

import (
    "context"
    "errors"
    "fmt"
    "strconv"
    "strings"

    "github.com/rs/zerolog"

    "github.com/slipstream/slipstream/internal/library/movies"
    "github.com/slipstream/slipstream/internal/metadata"
    "github.com/slipstream/slipstream/internal/module"
)

// ErrNoMetadataProvider is returned when no metadata provider is configured.
// Matches the sentinel used by librarymanager for consistency with error-checking callers.
var ErrNoMetadataProvider = errors.New("no metadata provider configured")

// metadataProvider implements module.MetadataProvider for the Movie module.
type metadataProvider struct {
    metadataSvc *metadata.Service
    movieSvc    *movies.Service
    logger      zerolog.Logger
}

func newMetadataProvider(metadataSvc *metadata.Service, movieSvc *movies.Service, logger *zerolog.Logger) *metadataProvider {
    return &metadataProvider{
        metadataSvc: metadataSvc,
        movieSvc:    movieSvc,
        logger:      logger.With().Str("component", "movie-metadata").Logger(),
    }
}
```

**`Search` — wraps `metadata.Service.SearchMovies`:**

```go
func (p *metadataProvider) Search(ctx context.Context, query string, opts module.SearchOptions) ([]module.SearchResult, error) {
    results, err := p.metadataSvc.SearchMovies(ctx, query, opts.Year)
    if err != nil {
        return nil, err
    }

    out := make([]module.SearchResult, len(results))
    for i, r := range results {
        out[i] = module.SearchResult{
            ExternalID:  strconv.Itoa(r.ID),
            Title:       r.Title,
            Year:        r.Year,
            Overview:    r.Overview,
            PosterURL:   r.PosterURL,
            BackdropURL: r.BackdropURL,
            ExternalIDs: map[string]string{
                "tmdb": strconv.Itoa(r.ID),
                "imdb": r.ImdbID,
            },
            Extra: &r, // Preserve the full MovieResult for callers that need it
        }
    }
    return out, nil
}
```

**`GetByID` — wraps `metadata.Service.GetMovie`:**

```go
func (p *metadataProvider) GetByID(ctx context.Context, externalID string) (*module.MediaMetadata, error) {
    tmdbID, err := strconv.Atoi(externalID)
    if err != nil {
        return nil, fmt.Errorf("invalid TMDB ID: %w", err)
    }

    result, err := p.metadataSvc.GetMovie(ctx, tmdbID)
    if err != nil {
        return nil, err
    }

    return &module.MediaMetadata{
        ExternalID:  externalID,
        Title:       result.Title,
        Year:        result.Year,
        Overview:    result.Overview,
        PosterURL:   result.PosterURL,
        BackdropURL: result.BackdropURL,
        ExternalIDs: map[string]string{
            "tmdb": strconv.Itoa(result.ID),
            "imdb": result.ImdbID,
        },
        Extra: result,
    }, nil
}
```

**`GetExtendedInfo` — wraps `metadata.Service.GetExtendedMovie`:**

Note: `GetExtendedMovie` returns `*metadata.ExtendedMovieResult` (which embeds `MovieResult` and adds `Credits`, `ContentRating`, `Ratings`, `TrailerURL`). The `Extra` field stores the full `*ExtendedMovieResult` — callers needing movie-specific extended data type-assert to `*metadata.ExtendedMovieResult`.

```go
func (p *metadataProvider) GetExtendedInfo(ctx context.Context, externalID string) (*module.ExtendedMetadata, error) {
    tmdbID, err := strconv.Atoi(externalID)
    if err != nil {
        return nil, fmt.Errorf("invalid TMDB ID: %w", err)
    }

    // Returns *metadata.ExtendedMovieResult (embeds MovieResult).
    result, err := p.metadataSvc.GetExtendedMovie(ctx, tmdbID)
    if err != nil {
        return nil, err
    }

    return &module.ExtendedMetadata{
        Credits:       result.Credits,
        Ratings:       result.Ratings,
        ContentRating: result.ContentRating,
        TrailerURL:    result.TrailerURL,
        Extra:         result, // *metadata.ExtendedMovieResult
    }, nil
}
```

**`RefreshMetadata` — extracted from `librarymanager.RefreshMovieMetadata`:**

This is the most complex method. It extracts the movie refresh logic from `internal/library/librarymanager/metadata_refresh.go` into the module provider. The key steps:

1. Get movie from DB
2. Search metadata providers for the movie
3. Select best match
4. Fetch full details + release dates + content rating
5. Update the movie entity
6. Return a `RefreshResult` with diff info + artwork URLs

```go
func (p *metadataProvider) RefreshMetadata(ctx context.Context, entityID int64) (*module.RefreshResult, error) {
    movie, err := p.movieSvc.Get(ctx, entityID)
    if err != nil {
        return nil, fmt.Errorf("failed to get movie: %w", err)
    }

    if !p.metadataSvc.HasMovieProvider() {
        return nil, ErrNoMetadataProvider
    }

    results, err := p.metadataSvc.SearchMovies(ctx, movie.Title, movie.Year)
    if err != nil {
        return nil, fmt.Errorf("metadata search failed: %w", err)
    }

    if len(results) == 0 {
        p.logger.Warn().Str("title", movie.Title).Int("year", movie.Year).Msg("No metadata results found")
        return &module.RefreshResult{EntityID: entityID}, nil
    }

    bestMatch := p.selectBestMatch(movie, results)
    p.enrichDetails(ctx, bestMatch)

    fieldsChanged := p.computeFieldsChanged(movie, bestMatch)
    if err := p.updateMovieFromMetadata(ctx, movie.ID, bestMatch); err != nil {
        return nil, err
    }

    return &module.RefreshResult{
        EntityID:      entityID,
        Updated:       len(fieldsChanged) > 0,
        FieldsChanged: fieldsChanged,
        ArtworkURLs: module.ArtworkURLs{
            PosterURL:     bestMatch.PosterURL,
            BackdropURL:   bestMatch.BackdropURL,
            LogoURL:       bestMatch.LogoURL,
            StudioLogoURL: bestMatch.StudioLogoURL,
        },
        Metadata: bestMatch,
    }, nil
}
```

**Helper methods** — extracted from `librarymanager/metadata_refresh.go`:

```go
func (p *metadataProvider) selectBestMatch(movie *movies.Movie, results []metadata.MovieResult) *metadata.MovieResult {
    // Exact title + year match
    movieTitleLower := strings.ToLower(movie.Title)
    for i := range results {
        if results[i].Year == movie.Year && strings.EqualFold(results[i].Title, movieTitleLower) {
            return &results[i]
        }
    }
    // Title prefix + year match
    for i := range results {
        if results[i].Year == movie.Year && strings.HasPrefix(strings.ToLower(results[i].Title), movieTitleLower) {
            return &results[i]
        }
    }
    // Year match only
    for i := range results {
        if results[i].Year == movie.Year {
            return &results[i]
        }
    }
    return &results[0]
}

func (p *metadataProvider) enrichDetails(ctx context.Context, match *metadata.MovieResult) {
    if match.ID > 0 {
        if details, err := p.metadataSvc.GetMovie(ctx, match.ID); err == nil {
            *match = *details
        }
    }
    if match.ID > 0 {
        if logoURL, err := p.metadataSvc.GetMovieLogoURL(ctx, match.ID); err == nil && logoURL != "" {
            match.LogoURL = logoURL
        }
    }
}

func (p *metadataProvider) computeFieldsChanged(movie *movies.Movie, match *metadata.MovieResult) []string {
    var changed []string
    if movie.Title != match.Title {
        changed = append(changed, "title")
    }
    if movie.Year != match.Year {
        changed = append(changed, "year")
    }
    if movie.TmdbID != match.ID {
        changed = append(changed, "tmdbId")
    }
    if movie.ImdbID != match.ImdbID {
        changed = append(changed, "imdbId")
    }
    if movie.Overview != match.Overview {
        changed = append(changed, "overview")
    }
    if movie.Runtime != match.Runtime {
        changed = append(changed, "runtime")
    }
    if movie.Studio != match.Studio {
        changed = append(changed, "studio")
    }
    return changed
}

func (p *metadataProvider) updateMovieFromMetadata(ctx context.Context, movieID int64, match *metadata.MovieResult) error {
    title := match.Title
    year := match.Year
    tmdbID := match.ID
    imdbID := match.ImdbID
    overview := match.Overview
    runtime := match.Runtime
    studio := match.Studio

    var releaseDate, physicalReleaseDate, theatricalReleaseDate, contentRating string
    if tmdbID > 0 {
        digital, physical, theatrical, err := p.metadataSvc.GetMovieReleaseDates(ctx, tmdbID)
        if err != nil {
            p.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to fetch release dates")
        } else {
            releaseDate = digital
            physicalReleaseDate = physical
            theatricalReleaseDate = theatrical
        }

        if cr, err := p.metadataSvc.GetMovieContentRating(ctx, tmdbID); err == nil && cr != "" {
            contentRating = cr
        }
    }

    _, err := p.movieSvc.Update(ctx, movieID, &movies.UpdateMovieInput{
        Title:                 &title,
        Year:                  &year,
        TmdbID:                &tmdbID,
        ImdbID:                &imdbID,
        Overview:              &overview,
        Runtime:               &runtime,
        Studio:                &studio,
        ReleaseDate:           &releaseDate,
        PhysicalReleaseDate:   &physicalReleaseDate,
        TheatricalReleaseDate: &theatricalReleaseDate,
        ContentRating:         &contentRating,
    })
    return err
}
```

**Implementation notes for the subagent:**
- Read `internal/library/librarymanager/metadata_refresh.go` — specifically `RefreshMovieMetadata`, `searchMovieMetadata`, `selectBestMovieMatch`, `enrichMovieDetails`, `updateMovieWithMetadata`, and `downloadMovieArtworkAsync`. The logic being extracted comes from these functions.
- Read `internal/modules/movie/module.go` — this is the existing module stub from Phase 0. Add the metadata provider as a field on the module struct (or as a separate struct composed into the module).
- The `metadataProvider` struct needs `*metadata.Service` and `*movies.Service`. These come via constructor injection but are NOT wired via Wire yet (that happens in Task 3.6). For now, just write the constructor and methods.
- Do NOT import `internal/library/librarymanager` — this would create a circular dependency. The extraction is one-directional: logic moves FROM librarymanager INTO the module.
- The `selectBestMatch` function uses `strings.EqualFold` — note the existing code in librarymanager has a redundancy where it pre-lowers the title (`movieTitleLower`) then passes it to `strings.EqualFold` (which is already case-insensitive). The lowering IS needed for the `HasPrefix` comparison. Reproduce the existing behavior exactly — do not fix cosmetic issues in this phase.

**Verify:**
- `go build ./internal/modules/movie/...` compiles
- `go vet ./internal/modules/movie/...` clean

---

### Task 3.4: Implement TV MetadataProvider ✅

**Depends on:** Tasks 3.1 and 3.2 (can run in parallel with Task 3.3)

**Create** `internal/modules/tv/metadata.go`

Implements `module.MetadataProvider` for the TV module. Significantly more complex than movies because of the hierarchical structure (series → seasons → episodes).

```go
package tv

import (
    "context"
    "errors"
    "fmt"
    "strconv"

    "github.com/rs/zerolog"

    tvlib "github.com/slipstream/slipstream/internal/library/tv"
    "github.com/slipstream/slipstream/internal/metadata"
    "github.com/slipstream/slipstream/internal/module"
)

// ErrNoMetadataProvider is returned when no metadata provider is configured.
var ErrNoMetadataProvider = errors.New("no metadata provider configured")

type metadataProvider struct {
    metadataSvc *metadata.Service
    tvSvc       *tvlib.Service
    logger      zerolog.Logger
}

func newMetadataProvider(metadataSvc *metadata.Service, tvSvc *tvlib.Service, logger *zerolog.Logger) *metadataProvider {
    return &metadataProvider{
        metadataSvc: metadataSvc,
        tvSvc:       tvSvc,
        logger:      logger.With().Str("component", "tv-metadata").Logger(),
    }
}
```

**`Search` — wraps `metadata.Service.SearchSeries`:**

```go
func (p *metadataProvider) Search(ctx context.Context, query string, opts module.SearchOptions) ([]module.SearchResult, error) {
    // Note: SearchSeries does not accept a year parameter — opts.Year is ignored.
    // The underlying TVDB/TMDB series search APIs don't support year filtering.
    results, err := p.metadataSvc.SearchSeries(ctx, query)
    if err != nil {
        return nil, err
    }

    out := make([]module.SearchResult, len(results))
    for i, r := range results {
        externalIDs := make(map[string]string)
        if r.TvdbID > 0 {
            externalIDs["tvdb"] = strconv.Itoa(r.TvdbID)
        }
        if r.TmdbID > 0 {
            externalIDs["tmdb"] = strconv.Itoa(r.TmdbID)
        }
        if r.ImdbID != "" {
            externalIDs["imdb"] = r.ImdbID
        }

        // Primary external ID is TVDB for TV module (spec §3.2)
        primaryID := strconv.Itoa(r.TvdbID)
        if r.TvdbID == 0 {
            primaryID = strconv.Itoa(r.TmdbID)
        }

        out[i] = module.SearchResult{
            ExternalID:  primaryID,
            Title:       r.Title,
            Year:        r.Year,
            Overview:    r.Overview,
            PosterURL:   r.PosterURL,
            BackdropURL: r.BackdropURL,
            ExternalIDs: externalIDs,
            Extra:       &r,
        }
    }
    return out, nil
}
```

**`GetByID` — wraps `metadata.Service.GetSeries`:**

Note: The externalID for TV could be a TVDB ID or TMDB ID. The module uses the convention that TVDB is primary.

```go
func (p *metadataProvider) GetByID(ctx context.Context, externalID string) (*module.MediaMetadata, error) {
    id, err := strconv.Atoi(externalID)
    if err != nil {
        return nil, fmt.Errorf("invalid external ID: %w", err)
    }

    // GetSeries(ctx, tvdbID, tmdbID) tries TVDB first, falls back to TMDB.
    // We pass the same ID for both because the module's generic externalID doesn't
    // carry type information. The mismatched lookup fails gracefully and the fallback
    // succeeds. This is a pragmatic trade-off of the single-externalID interface.
    result, err := p.metadataSvc.GetSeries(ctx, id, id)
    if err != nil {
        return nil, err
    }

    externalIDs := make(map[string]string)
    if result.TvdbID > 0 {
        externalIDs["tvdb"] = strconv.Itoa(result.TvdbID)
    }
    if result.TmdbID > 0 {
        externalIDs["tmdb"] = strconv.Itoa(result.TmdbID)
    }
    if result.ImdbID != "" {
        externalIDs["imdb"] = result.ImdbID
    }

    return &module.MediaMetadata{
        ExternalID:  externalID,
        Title:       result.Title,
        Year:        result.Year,
        Overview:    result.Overview,
        PosterURL:   result.PosterURL,
        BackdropURL: result.BackdropURL,
        ExternalIDs: externalIDs,
        Extra:       result,
    }, nil
}
```

**`GetExtendedInfo` — wraps `metadata.Service.GetExtendedSeries`:**

Note: `GetExtendedSeries` returns `*metadata.ExtendedSeriesResult` (which embeds `SeriesResult` and adds `Credits`, `ContentRating`, `Ratings`, `Seasons`, `TrailerURL`). The `Extra` field stores the full `*ExtendedSeriesResult` — callers needing TV-specific extended data type-assert to `*metadata.ExtendedSeriesResult`.

```go
func (p *metadataProvider) GetExtendedInfo(ctx context.Context, externalID string) (*module.ExtendedMetadata, error) {
    tmdbID, err := strconv.Atoi(externalID)
    if err != nil {
        return nil, fmt.Errorf("invalid external ID: %w", err)
    }

    // Returns *metadata.ExtendedSeriesResult (embeds SeriesResult).
    result, err := p.metadataSvc.GetExtendedSeries(ctx, tmdbID)
    if err != nil {
        return nil, err
    }

    return &module.ExtendedMetadata{
        Credits:       result.Credits,
        Ratings:       result.Ratings,
        ContentRating: result.ContentRating,
        TrailerURL:    result.TrailerURL,
        Extra:         result, // *metadata.ExtendedSeriesResult
    }, nil
}
```

**`RefreshMetadata` — extracted from `librarymanager.RefreshSeriesMetadata`:**

This is the most complex method. Key differences from movie refresh:
- Must fetch season/episode metadata
- Must compute a structured diff of added/updated/removed seasons and episodes
- Uses `UpdateSeasonsFromMetadata` which preserves existing `monitored`/`status` via upsert
- Per spec §18.5, removed episodes are NOT auto-deleted — flagged via `ChildrenRemoved`

```go
func (p *metadataProvider) RefreshMetadata(ctx context.Context, entityID int64) (*module.RefreshResult, error) {
    series, err := p.tvSvc.GetSeries(ctx, entityID)
    if err != nil {
        return nil, fmt.Errorf("failed to get series: %w", err)
    }

    if !p.metadataSvc.HasSeriesProvider() {
        return nil, ErrNoMetadataProvider
    }

    // Search for the series
    results, err := p.metadataSvc.SearchSeries(ctx, series.Title)
    if err != nil {
        return nil, fmt.Errorf("metadata search failed: %w", err)
    }

    if len(results) == 0 {
        p.logger.Warn().Str("title", series.Title).Msg("No metadata results found")
        return &module.RefreshResult{EntityID: entityID}, nil
    }

    bestMatch := p.enrichSeriesDetails(ctx, &results[0])
    fieldsChanged := p.computeSeriesFieldsChanged(series, bestMatch)

    if err := p.updateSeriesFromMetadata(ctx, series.ID, bestMatch); err != nil {
        return nil, err
    }

    // Fetch and reconcile seasons/episodes
    childDiff := p.refreshSeasons(ctx, series, bestMatch.TmdbID, bestMatch.TvdbID)

    return &module.RefreshResult{
        EntityID:        entityID,
        Updated:         len(fieldsChanged) > 0,
        FieldsChanged:   fieldsChanged,
        ChildrenAdded:   childDiff.added,
        ChildrenUpdated: childDiff.updated,
        ChildrenRemoved: childDiff.removed,
        ArtworkURLs: module.ArtworkURLs{
            PosterURL:   bestMatch.PosterURL,
            BackdropURL: bestMatch.BackdropURL,
            LogoURL:     bestMatch.LogoURL,
        },
        Metadata: bestMatch,
    }, nil
}
```

**Season/episode diff computation:**

```go
type seasonEpisodeDiff struct {
    added   []module.RefreshChildEntry
    updated []module.RefreshChildEntry
    removed []module.RefreshChildEntry
}

func (p *metadataProvider) refreshSeasons(ctx context.Context, series *tvlib.Series, tmdbID, tvdbID int) seasonEpisodeDiff {
    if tmdbID == 0 && tvdbID == 0 {
        return seasonEpisodeDiff{}
    }

    seasonResults, err := p.metadataSvc.GetSeriesSeasons(ctx, tmdbID, tvdbID)
    if err != nil {
        p.logger.Warn().Err(err).Int("tmdbId", tmdbID).Int("tvdbId", tvdbID).Msg("Failed to fetch season metadata")
        return seasonEpisodeDiff{}
    }

    // Build lookup of existing episodes by (season, episode) key.
    // NOTE: Season struct has no Episodes field — must fetch via ListEpisodes.
    existingEpisodes, err := p.buildExistingEpisodeMap(ctx, series.ID)
    if err != nil {
        p.logger.Warn().Err(err).Int64("seriesId", series.ID).Msg("Failed to list existing episodes for diff")
        return seasonEpisodeDiff{}
    }

    // Track which existing episodes are seen in metadata
    seenKeys := make(map[string]bool)

    var diff seasonEpisodeDiff

    // Convert metadata and upsert (preserves existing monitored/status)
    seasonMeta := p.convertToSeasonMetadata(seasonResults)
    if err := p.tvSvc.UpdateSeasonsFromMetadata(ctx, series.ID, seasonMeta); err != nil {
        p.logger.Warn().Err(err).Int64("seriesId", series.ID).Msg("Failed to update seasons from metadata")
        return seasonEpisodeDiff{}
    }

    // Compute diff
    for _, sm := range seasonMeta {
        for _, ep := range sm.Episodes {
            key := fmt.Sprintf("S%02dE%02d", ep.SeasonNumber, ep.EpisodeNumber)
            seenKeys[key] = true

            if _, existed := existingEpisodes[key]; existed {
                diff.updated = append(diff.updated, module.RefreshChildEntry{
                    EntityType: module.EntityEpisode,
                    Identifier: key,
                    Title:      ep.Title,
                })
            } else {
                diff.added = append(diff.added, module.RefreshChildEntry{
                    EntityType: module.EntityEpisode,
                    Identifier: key,
                    Title:      ep.Title,
                })
            }
        }
    }

    // Find removed episodes (in library but not in metadata)
    for key, entry := range existingEpisodes {
        if !seenKeys[key] {
            diff.removed = append(diff.removed, entry)
        }
    }

    return diff
}

func (p *metadataProvider) buildExistingEpisodeMap(ctx context.Context, seriesID int64) (map[string]module.RefreshChildEntry, error) {
    // The Season struct has no Episodes field — episodes are a separate entity.
    // Fetch all episodes for the series via ListEpisodes(ctx, seriesID, nil).
    episodes, err := p.tvSvc.ListEpisodes(ctx, seriesID, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to list episodes: %w", err)
    }

    existing := make(map[string]module.RefreshChildEntry, len(episodes))
    for _, ep := range episodes {
        key := fmt.Sprintf("S%02dE%02d", ep.SeasonNumber, ep.EpisodeNumber)
        existing[key] = module.RefreshChildEntry{
            EntityType: module.EntityEpisode,
            Identifier: key,
            EntityID:   ep.ID,
            Title:      ep.Title,
        }
    }
    return existing, nil
}

func (p *metadataProvider) convertToSeasonMetadata(seasonResults []metadata.SeasonResult) []tvlib.SeasonMetadata {
    seasonMeta := make([]tvlib.SeasonMetadata, len(seasonResults))
    for i, sr := range seasonResults {
        episodes := make([]tvlib.EpisodeMetadata, len(sr.Episodes))
        for j, ep := range sr.Episodes {
            episodes[j] = tvlib.EpisodeMetadata{
                EpisodeNumber: ep.EpisodeNumber,
                SeasonNumber:  ep.SeasonNumber,
                Title:         ep.Title,
                Overview:      ep.Overview,
                AirDate:       ep.AirDate,
                Runtime:       ep.Runtime,
            }
        }
        seasonMeta[i] = tvlib.SeasonMetadata{
            SeasonNumber: sr.SeasonNumber,
            Name:         sr.Name,
            Overview:     sr.Overview,
            PosterURL:    sr.PosterURL,
            AirDate:      sr.AirDate,
            Episodes:     episodes,
        }
    }
    return seasonMeta
}
```

**Other helper methods** (extracted from librarymanager):

```go
func (p *metadataProvider) enrichSeriesDetails(ctx context.Context, match *metadata.SeriesResult) *metadata.SeriesResult {
    if match.TmdbID > 0 {
        if detail, err := p.metadataSvc.GetSeriesByTMDB(ctx, match.TmdbID); err == nil {
            match = detail
        }
    }
    if match.TmdbID > 0 {
        if logoURL, err := p.metadataSvc.GetSeriesLogoURL(ctx, match.TmdbID); err == nil && logoURL != "" {
            match.LogoURL = logoURL
        }
    }
    return match
}

func (p *metadataProvider) computeSeriesFieldsChanged(series *tvlib.Series, match *metadata.SeriesResult) []string {
    var changed []string
    if series.Title != match.Title {
        changed = append(changed, "title")
    }
    if series.Year != match.Year {
        changed = append(changed, "year")
    }
    if series.TvdbID != match.TvdbID {
        changed = append(changed, "tvdbId")
    }
    if series.TmdbID != match.TmdbID {
        changed = append(changed, "tmdbId")
    }
    if series.ImdbID != match.ImdbID {
        changed = append(changed, "imdbId")
    }
    if series.Overview != match.Overview {
        changed = append(changed, "overview")
    }
    return changed
}

func (p *metadataProvider) updateSeriesFromMetadata(ctx context.Context, seriesID int64, match *metadata.SeriesResult) error {
    title := match.Title
    year := match.Year
    tvdbID := match.TvdbID
    tmdbID := match.TmdbID
    imdbID := match.ImdbID
    overview := match.Overview
    runtime := match.Runtime
    status := match.Status
    network := match.Network
    networkLogoURL := match.NetworkLogoURL

    _, err := p.tvSvc.UpdateSeries(ctx, seriesID, &tvlib.UpdateSeriesInput{
        Title:            &title,
        Year:             &year,
        TvdbID:           &tvdbID,
        TmdbID:           &tmdbID,
        ImdbID:           &imdbID,
        Overview:         &overview,
        Runtime:          &runtime,
        ProductionStatus: &status,
        Network:          &network,
        NetworkLogoURL:   &networkLogoURL,
    })
    return err
}
```

**Implementation notes for the subagent:**
- Read `internal/library/librarymanager/metadata_refresh.go` — specifically `RefreshSeriesMetadata`, `enrichSeriesDetails`, `updateSeriesWithMetadata`, `updateSeriesSeasons`, and `convertToSeasonMetadata`. The series update and season upsert logic is extracted from there. **However**, the diff computation (`seasonEpisodeDiff`, `buildExistingEpisodeMap`, added/updated/removed categorization) is **entirely new logic** implementing spec §18.5 — the existing code does a pure upsert with no diff tracking. Do not look for diff logic in the librarymanager; implement it from scratch per the code below.
- Read `internal/modules/tv/module.go` — this is the existing TV module stub from Phase 0. The `metadataProvider` is a separate struct, not embedded in the module struct. It will be composed into the module in Task 3.6.
- The TV service import must use an alias (`tvlib`) to avoid conflict with the package name `tv`.
- **Critical:** The `Season` struct does NOT have an `Episodes` field — episodes are a separate entity fetched via `tvSvc.ListEpisodes(ctx, seriesID, nil)`. The `buildExistingEpisodeMap` function must call `ListEpisodes` to fetch all episodes for the series. Do NOT iterate `series.Seasons[].Episodes` — that field does not exist.
- The `convertToSeasonMetadata` function duplicates code that exists in `librarymanager/metadata_refresh.go` (`convertToSeasonMetadata`) and `librarymanager/add.go`. This is intentional — deduplication happens in Phase 11 (Final Cleanup).
- Do NOT import `internal/library/librarymanager` — this would create a circular dependency.

**Verify:**
- `go build ./internal/modules/tv/...` compiles
- `go vet ./internal/modules/tv/...` clean

---

### Task 3.5: Update librarymanager to Dispatch Refresh Through Module Providers ✅

**Depends on:** Tasks 3.3 and 3.4

**Modify** `internal/library/librarymanager/service.go` — add module registry dependency:

```go
import "github.com/slipstream/slipstream/internal/module"

type Service struct {
    // ... existing fields ...
    registry *module.Registry // NEW
}
```

Update the constructor `NewService` to accept `*module.Registry` as an optional parameter. If nil, the service operates in legacy mode (uses existing metadata refresh code). If non-nil, refresh operations dispatch through module providers.

**Modify** `internal/library/librarymanager/metadata_refresh.go` — add a generic dispatch method:

```go
// RefreshEntityMetadata refreshes metadata for any module entity by dispatching
// to the module's MetadataProvider. Falls back to legacy refresh if the module
// registry is not available.
func (s *Service) RefreshEntityMetadata(ctx context.Context, moduleType module.Type, entityID int64) (*module.RefreshResult, error) {
    if s.registry == nil {
        return s.legacyRefresh(ctx, moduleType, entityID)
    }

    mod := s.registry.Get(moduleType)
    if mod == nil {
        return nil, fmt.Errorf("unknown module type: %s", moduleType)
    }

    result, err := mod.RefreshMetadata(ctx, entityID)
    if err != nil {
        return nil, err
    }

    // Post-refresh framework actions: artwork download and folder scan
    s.handlePostRefreshArtwork(ctx, moduleType, result)
    s.handlePostRefreshScan(ctx, moduleType, entityID)

    return result, nil
}

func (s *Service) legacyRefresh(ctx context.Context, moduleType module.Type, entityID int64) (*module.RefreshResult, error) {
    switch moduleType {
    case module.TypeMovie:
        movie, err := s.RefreshMovieMetadata(ctx, entityID)
        if err != nil {
            return nil, err
        }
        return &module.RefreshResult{
            EntityID: movie.ID,
            Updated:  true,
        }, nil
    case module.TypeTV:
        series, err := s.RefreshSeriesMetadata(ctx, entityID)
        if err != nil {
            return nil, err
        }
        return &module.RefreshResult{
            EntityID: series.ID,
            Updated:  true,
        }, nil
    default:
        return nil, fmt.Errorf("unsupported module type for legacy refresh: %s", moduleType)
    }
}
```

**Framework post-refresh actions:**

```go
func (s *Service) handlePostRefreshArtwork(ctx context.Context, moduleType module.Type, result *module.RefreshResult) {
    if s.artwork == nil || result == nil {
        return
    }

    urls := result.ArtworkURLs
    if urls.PosterURL == "" && urls.BackdropURL == "" && urls.LogoURL == "" && urls.StudioLogoURL == "" {
        return
    }

    // NOTE: The Metadata field uses `any`, so these type assertions are the coupling
    // point between the module MetadataProvider (which sets the concrete type) and the
    // framework post-refresh handler (which reads it). Each module's RefreshMetadata MUST
    // store the correct pointer type: *metadata.MovieResult for movie, *metadata.SeriesResult
    // for TV. A mismatch silently skips artwork. Future modules must document their concrete
    // Metadata type in their MetadataProvider implementation.
    go func() {
        switch moduleType {
        case module.TypeMovie:
            movieResult, ok := result.Metadata.(*metadata.MovieResult)
            if !ok {
                s.logger.Error().Str("actualType", fmt.Sprintf("%T", result.Metadata)).
                    Msg("Post-refresh artwork: expected *metadata.MovieResult in RefreshResult.Metadata")
                return
            }
            if err := s.artwork.DownloadMovieArtwork(context.Background(), movieResult); err != nil {
                s.logger.Warn().Err(err).Msg("Failed to download movie artwork after refresh")
            }
        case module.TypeTV:
            seriesResult, ok := result.Metadata.(*metadata.SeriesResult)
            if !ok {
                s.logger.Error().Str("actualType", fmt.Sprintf("%T", result.Metadata)).
                    Msg("Post-refresh artwork: expected *metadata.SeriesResult in RefreshResult.Metadata")
                return
            }
            if err := s.artwork.DownloadSeriesArtwork(context.Background(), seriesResult); err != nil {
                s.logger.Warn().Err(err).Msg("Failed to download series artwork after refresh")
            }
        }
    }()
}

func (s *Service) handlePostRefreshScan(ctx context.Context, moduleType module.Type, entityID int64) {
    switch moduleType {
    case module.TypeMovie:
        movie, err := s.movies.Get(ctx, entityID)
        if err == nil {
            s.scanMovieFolder(ctx, movie)
        }
    case module.TypeTV:
        series, err := s.tv.GetSeries(ctx, entityID)
        if err == nil {
            s.scanSeriesFolder(ctx, series)
        }
    }
}
```

**Update existing refresh callers to use the new dispatch method:**

The existing `RefreshMovieMetadata` and `RefreshSeriesMetadata` methods are NOT removed — they remain as the legacy path and are also called by the new `legacyRefresh` fallback. However, the **handlers** and **scheduled tasks** that call these should be updated to call `RefreshEntityMetadata` instead when the registry is available.

**Modify** `internal/library/librarymanager/handlers.go` — update the refresh endpoints:

Find the handler methods for movie/series refresh (likely `HandleRefreshMovie` and `HandleRefreshSeries` or similar) and update them to call `RefreshEntityMetadata` if the registry is available, falling back to the direct methods:

```go
func (h *Handlers) RefreshMovie(c echo.Context) error {
    // ... parse ID ...
    if h.service.registry != nil {
        result, err := h.service.RefreshEntityMetadata(c.Request().Context(), module.TypeMovie, id)
        if err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        // Return the refreshed movie
        movie, _ := h.service.movies.Get(c.Request().Context(), id)
        return c.JSON(http.StatusOK, movie)
    }
    // Legacy path
    movie, err := h.service.RefreshMovieMetadata(c.Request().Context(), id)
    // ...
}
```

**Modify** `RefreshMonitoredSeriesMetadata` and `RefreshAllMovies` / `RefreshAllSeries` — update the inner loop to call `RefreshEntityMetadata` when registry is available. The bulk refresh methods iterate over entities and call per-entity refresh — swap the inner call.

**Implementation notes for the subagent:**
- Read `internal/library/librarymanager/service.go` to see the full `Service` struct and `NewService` constructor. Add `registry *module.Registry` as a new field.
- Read `internal/library/librarymanager/handlers.go` to find the refresh handler methods. Update them to dispatch through `RefreshEntityMetadata`.
- The `scanMovieFolder` and `scanSeriesFolder` private methods already exist — find them in `scanning.go`.
- Keep both paths (registry dispatch + legacy) working. The registry may be nil during the transition period.
- Import `"github.com/slipstream/slipstream/internal/module"` in the service file.
- DO NOT modify the `metadata_refresh.go` file's existing `RefreshMovieMetadata` and `RefreshSeriesMetadata` functions — they serve as the legacy path. Add the new dispatch code alongside them.

**Verify:**
- `go build ./internal/library/librarymanager/...` compiles
- `go vet ./internal/library/librarymanager/...` clean

---

### Task 3.6: Wire Integration — Connect Module Providers to Services ✅

**Depends on:** Task 3.5

**Modify** `internal/modules/movie/module.go` — add constructor that wires the metadata provider:

```go
func NewModule(metadataSvc *metadata.Service, movieSvc *movies.Service, logger *zerolog.Logger) *Module {
    return &Module{
        descriptor:       &Descriptor{},
        metadataProvider: newMetadataProvider(metadataSvc, movieSvc, logger),
    }
}
```

The `Module` struct needs to hold the metadata provider and expose it through the `MetadataProvider` interface methods. Since `Module` must satisfy `module.Module` (which embeds `module.MetadataProvider`), the `MetadataProvider` methods must be on `*Module`:

```go
type Module struct {
    descriptor       *Descriptor
    metadataProvider *metadataProvider
}

// Delegate MetadataProvider methods to the internal provider
func (m *Module) Search(ctx context.Context, query string, opts module.SearchOptions) ([]module.SearchResult, error) {
    return m.metadataProvider.Search(ctx, query, opts)
}
func (m *Module) GetByID(ctx context.Context, externalID string) (*module.MediaMetadata, error) {
    return m.metadataProvider.GetByID(ctx, externalID)
}
func (m *Module) GetExtendedInfo(ctx context.Context, externalID string) (*module.ExtendedMetadata, error) {
    return m.metadataProvider.GetExtendedInfo(ctx, externalID)
}
func (m *Module) RefreshMetadata(ctx context.Context, entityID int64) (*module.RefreshResult, error) {
    return m.metadataProvider.RefreshMetadata(ctx, entityID)
}
```

**Do the same for** `internal/modules/tv/module.go`.

**Modify** `internal/api/wire.go` — update module constructors in the Wire build:

The movie and TV modules were registered with stub constructors in Phase 0. Update them to pass the required dependencies:

```go
// In wire.Build(...):
movie.NewModule,    // Now requires *metadata.Service, *movies.Service, *zerolog.Logger
tv.NewModule,       // Now requires *metadata.Service, *tv.Service, *zerolog.Logger
```

Wire will resolve these dependencies from the existing providers.

**Modify** `internal/api/wire.go` — pass `*module.Registry` to `librarymanager.NewService`:

The librarymanager constructor needs the registry. Update the Wire build to provide it.

**Run** `make wire` to regenerate `wire_gen.go`.

**Implementation notes for the subagent:**
- Read `internal/modules/movie/module.go` and `internal/modules/tv/module.go` — these are the Phase 0 stubs. They satisfy `module.Module` via stub methods that panic or return nil. Replace the `MetadataProvider` stub methods with delegations to the real provider.
- Read `internal/api/wire.go` to understand the current Wire build. The module constructors may be registered via `module.RegisterAll` or individually. Update accordingly.
- After modifying `wire.go`, run `make wire`. If it fails, the error message will tell you which providers are missing.
- The `Module` struct currently satisfies ALL required interfaces with stubs. Only replace the `MetadataProvider` methods — leave all other interface stubs unchanged.
- **DO NOT use git stash or any commands that affect the entire worktree.**

**Verify:**
- `make wire` succeeds
- `go build ./...` compiles (full project)
- `go vet ./...` clean

---

### Task 3.7: Phase 3 Validation ✅

**Run all of these after all Phase 3 tasks complete:**

1. `make build` — full project builds (backend + frontend)
2. `make test` — all Go tests pass
3. `make lint` — no new Go lint issues
4. `cd web && bun run lint` — no new frontend lint issues (no frontend changes in this phase, but verify no regressions)

**Integration verification:**
- Start the app → movie and TV modules register with metadata providers (check logs for "movie-metadata" and "tv-metadata" component tags)
- Add a movie → metadata search still works (existing flow unchanged)
- Add a series → season/episode metadata still fetched (existing flow unchanged)
- Refresh a movie's metadata → uses module provider path if registry is wired, produces `RefreshResult` with `FieldsChanged`
- Refresh a series' metadata → uses module provider path, `RefreshResult` includes `ChildrenAdded`/`ChildrenUpdated`/`ChildrenRemoved` data
- Bulk refresh all movies → completes without error
- Bulk refresh monitored series → completes without error
- Dev mode toggle → existing mock metadata still works (SetClients path unchanged)

**Specific regression checks:**
- `ExternalIDs()` methods on `Movie` and `Series` return correct maps
- `RefreshEntityMetadata` fallback to legacy path works when registry is nil
- Artwork downloads still happen after metadata refresh
- Season/episode upsert still preserves `monitored` and `status` on existing episodes

---

### Phase 3 File Inventory

### New files created in Phase 3
```
internal/module/metadata_types.go (RefreshResult, ArtworkURLs, RefreshChildEntry)
internal/modules/movie/metadata.go (movie MetadataProvider implementation)
internal/modules/tv/metadata.go (TV MetadataProvider implementation)
```

### Existing files modified in Phase 3

**Backend — module framework types:**
```
internal/module/search_types.go (flesh out SearchOptions, SearchResult, MediaMetadata, ExtendedMetadata)
internal/module/interfaces.go (verify MetadataProvider signature matches new types)
```

**Backend — library entity types:**
```
internal/library/movies/movie.go (ExternalIDs() method)
internal/library/tv/series.go or service.go (ExternalIDs() method on Series — wherever Series struct is defined)
```

**Backend — module stubs:**
```
internal/modules/movie/module.go (NewModule constructor, MetadataProvider delegation methods)
internal/modules/tv/module.go (NewModule constructor, MetadataProvider delegation methods)
```

**Backend — librarymanager:**
```
internal/library/librarymanager/service.go (add registry field, update constructor)
internal/library/librarymanager/metadata_refresh.go (add RefreshEntityMetadata dispatch, post-refresh handlers)
internal/library/librarymanager/handlers.go (update refresh handlers to use dispatch)
```

**Backend — Wire DI:**
```
internal/api/wire.go (update module constructors, pass registry to librarymanager)
internal/api/wire_gen.go (regenerated via make wire)
```

---

