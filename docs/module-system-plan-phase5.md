## Phase 5: Search & Indexer

**Goal:** Modularize the search pipeline so that search criteria construction, release filtering, title matching, category selection, and group search logic flow through module-provided `SearchStrategy` implementations. After this phase, the search system dispatches through the module framework instead of hard-coding movie/TV branches, and new modules can participate in search by implementing `SearchStrategy`.

**Spec sections covered:** §5.1 (Category Mappings), §5.2/§16 (Indexer Capability Auto-Detection), §5.3 (SearchableItem Interface), §5.4 (Custom Filter Functions), §5.5 (Group Search), §5.6 (RSS Sync)

**Depends on:** Phase 4 (WantedCollector implementations exist, `module.SearchableItem` interface is defined and has concrete implementations via `WantedItem`, module registry is functional)

### Phase 5 Agent Context Management

This phase refactors the search pipeline across multiple packages. The key constraint is that **the autosearch pipeline is the most bug-sensitive area of the codebase** (see `internal/CLAUDE.md` "Auto Search & Upgrade Pipeline" section). Every change must preserve existing behavior exactly.

**Execution strategy:**

1. **Task 5.1** (interface types) and **Task 5.3** (title matching extraction) can run in **parallel** — neither depends on the other.
2. **Task 5.2** (module implementations) depends on 5.1 + 5.3 (needs interface types and titleutil package).
3. **Task 5.4** (criteria refactor) depends on 5.1 + 5.2.
4. **Task 5.5** (indexer capability refactor) depends on 5.1.
5. **Task 5.6** (group search formalization) depends on 5.1 + 5.2.
6. Tasks 5.4, 5.5, and 5.6 have no mutual dependencies and can run in **parallel** after 5.2 completes.
7. **Task 5.7** (RSS sync refactor) depends on 5.3 + 5.4 + 5.6.
8. **Task 5.8** (autosearch refactor) depends on 5.4 + 5.6.
9. Tasks 5.7 and 5.8 can run in **parallel**.
10. **Task 5.9** (Wire integration) depends on all prior tasks.
11. **Task 5.10** (validation) runs last.

**Context budget:** The autosearch and RSS sync services are large files (~1000 lines). Do NOT read these into main context. Delegate all code changes to Sonnet subagents. Give each subagent the exact function signatures to produce and the exact files to modify. Main context only reads verification files (<100 lines) and runs compile checks.

**Critical safety rule for all subagents:** Include in every subagent prompt: "This is the autosearch pipeline — the most bug-sensitive area. Read `internal/CLAUDE.md` 'Auto Search & Upgrade Pipeline' section before making changes. Do NOT use git stash or any commands that affect the entire worktree."

---

### Deferred from Phase 2

These items were deferred from Phase 2 (Quality System) and must be addressed in this phase:

1. **Call `RegisterModuleQualities()` at startup for each enabled module.** Phase 2 added the registration method on the quality service and `QualityItems()` on both module descriptors, but never wired the call at startup. Currently `GetQualitiesForModule()` falls back to `PredefinedQualities` (correct today since both modules use identical video tiers, but will silently break when a non-video module is added). Wire the call during module bootstrap so module-registered items are used instead of the fallback.

2. **Wire the scoring pipeline through module `QualityDefinition` interfaces.** The scoring pipeline (`internal/indexer/scoring/`) still references `quality.PredefinedQualities` directly. Per Architecture Decision #9 below, scoring remains framework-owned — but the quality *items* it operates on should come from the module's registered definitions (via `GetQualitiesForModule`) rather than the hardcoded global. This ensures new modules with different quality tiers (e.g., music bitrate tiers) are scored correctly.

---

### Phase 5 Architecture Decisions

1. **`SearchStrategy` is implemented on the Module struct, not a separate service.** Like `MonitoringPresets` and `WantedCollector`, each module's `Module` struct (in `internal/modules/movie/` and `internal/modules/tv/`) implements the `SearchStrategy` interface. The methods wrap existing logic that's currently scattered across `autosearch/service.go` (buildSearchCriteria), `indexer/search/aggregator.go` (FilterByCriteria, getTVFilterReason, getMovieFilterReason), and `indexer/search/title_match.go` (TitlesMatch, TVTitlesMatch).

2. **The `decisioning.SearchableItem` struct is NOT removed in this phase.** The spec §5.3 calls for replacing the struct with an interface. Phase 4 already defined `module.SearchableItem` as an interface and `module.WantedItem` as a concrete implementation. In this phase, we make the search pipeline consume `module.SearchableItem` for module-dispatched searches (RSS sync, scheduled autosearch) while keeping the `decisioning.SearchableItem` struct for internal service calls (`SearchMovie`, `SearchEpisode`) that don't flow through the module framework. Full elimination of the struct is deferred to Phase 10 when module services replace the movie/TV services entirely. This avoids a massive refactor of `autosearch/service.go` in this phase.

3. **Dual-path dispatch: module-aware + legacy.** The search service gains a `registry` field. When set:
   - `getIndexersForSearch` uses module-declared categories instead of hard-coded `ListEnabledForMovies`/`ListEnabledForTV`.
   - `FilterByCriteria` dispatches to the module's `FilterRelease` method.
   - `buildSearchCriteria` dispatches to the module's `BuildSearchCriteria` method.
   When the registry is nil, all paths fall back to the existing hard-coded behavior. This matches the Phase 3/4 pattern.

4. **Category mappings move to module declarations but the `indexer` package constants remain.** The `indexer.MovieCategories()`, `indexer.TVCategories()`, `indexer.IsMovieCategory()`, `indexer.IsTVCategory()` functions stay as convenience helpers. Module `SearchStrategy.Categories()` implementations return these same slices. When the search service has a registry, it resolves categories from the module instead of calling `indexer.MovieCategories()` directly. This is a low-risk incremental change.

5. **Indexer capability auto-detection uses category ranges, not flags.** Currently, `IndexerDefinition` has `SupportsMovies` and `SupportsTV` boolean fields set at indexer setup time based on the Cardigann definition's declared categories. This phase adds a `SupportsModule(moduleCategories []int) bool` method that checks whether the indexer's category list includes any categories in the module's declared range. The existing `SupportsMovies`/`SupportsTV` booleans remain as a fast path and are set during indexer creation using the same category-range logic. When a new module is added, the framework automatically detects support from the indexer's categories without needing new boolean columns.

6. **Group search formalization.** The current codebase has season pack eligibility checks (`decisioning.IsSeasonPackEligible`, `decisioning.IsSeasonPackUpgradeEligible`) and implicit suppression via dedup keys (`seasonKeyForEpisode` in `rsssync/matcher.go`). The `SearchStrategy` interface formalizes these: `IsGroupSearchEligible` wraps the eligibility checks (TV module delegates to `decisioning.IsSeasonPackEligible`/`IsSeasonPackUpgradeEligible` via the `forUpgrade` parameter). `SuppressChildSearches` is a **new** method that makes the implicit suppression explicit and module-owned — no existing function with this name exists today. Movie module returns `false`/`nil` for all group search methods.

7. **RSS sync's `WantedIndex` gains module awareness.** The `WantedIndex` currently indexes `decisioning.SearchableItem` structs. In this phase, the RSS sync service uses the module framework's `WantedCollector` to gather wanted items (returning `module.SearchableItem` interface values). The `WantedIndex` is updated to work with `module.SearchableItem` instead of `decisioning.SearchableItem`. **Important distinction:** The RSS matcher does NOT dispatch to `SearchStrategy.FilterRelease` or `TitlesMatch` — those are used by the autosearch aggregator (`FilterByCriteria` in `indexer/search/aggregator.go`). The RSS matcher has its own candidate-matching logic (`matchMovie`, `matchEpisode`, `matchSeasonPack`) that uses index lookups by ID/title and simpler season/episode checks. The only module dispatch in the matcher is `SearchStrategy.IsGroupSearchEligible` for season pack eligibility (replacing direct calls to `decisioning.IsSeasonPackEligible`). Existing title match helpers (`TitlesMatch`, `TVTitlesMatch`, `NormalizeTitle`) move to a shared utility package (`internal/module/titleutil/`) since both module implementations and the autosearch aggregator need them.

8. **The `SearchCriteria` struct gains a `ModuleType` field.** This field is populated by the module's `BuildSearchCriteria` method and used by the search service to dispatch to the correct indexers. The existing `Type` field (`"movie"`, `"tvsearch"`) is still set by the module — it represents the Newznab search type, which is indexer-protocol-specific and must remain.

9. **Scoring remains framework-owned.** The `scoring.Scorer` is not module-specific. All modules share the same scoring algorithm (quality score + health score + indexer score + match score + age score). The `ScoringContext` struct already has `SearchYear`, `SearchSeason`, `SearchEpisode` fields that handle movie vs TV scoring differences. Future modules (music) may need additional scoring parameters — but that's a future concern, not a Phase 5 change.

10. **`SelectBestRelease` stays in `decisioning` package — NOT refactored in this phase.** Currently, `SelectBestRelease` has hard-coded branches for `MediaTypeEpisode`/`MediaTypeSeason` that parse filenames and check season/episode matching via `shouldSkipTVRelease`. Refactoring this to use `SearchStrategy.FilterRelease` is deferred to Phase 10 when the decisioning package is fully module-aware. In this phase, `SelectBestRelease` continues to use its existing hard-coded TV logic. This avoids a complex signature change to a function called from multiple paths (autosearch `SearchMovie`/`SearchEpisode`, RSS sync grab flow).

---

### Task 5.1: Flesh Out SearchStrategy Interface and Supporting Types

**Depends on:** Phase 4 complete

**Update** `internal/module/interfaces.go` — verify the `SearchStrategy` interface matches the spec and the codebase needs. The Phase 0 skeleton defined:

```go
type SearchStrategy interface {
    Categories() []int
    FilterRelease(release Release, item SearchableItem) (reject bool, reason string)
    TitlesMatch(releaseTitle, mediaTitle string) bool
    BuildSearchCriteria(item SearchableItem) SearchCriteria
    IsGroupSearchEligible(parentEntityType EntityType, parentID int64) bool
    SuppressChildSearches(parentEntityType EntityType, parentID int64, grabbedRelease Release) []int64
}
```

**Update to final form:**

```go
// SearchStrategy defines search behavior for a module. (spec §5.4, §5.5, §5.6)
type SearchStrategy interface {
    // Categories returns the Newznab/Torznab category ranges for this module.
    // Movie: 2000-2999, TV: 5000-5999, Music: 3000-3999, Book: 7000-7999.
    Categories() []int

    // DefaultSearchCategories returns the default subset of categories used for searches.
    // This may exclude niche subcategories. Used for Prowlarr config defaults, NOT for
    // autosearch criteria (which uses Categories() for full coverage).
    DefaultSearchCategories() []int

    // FilterRelease determines whether a release should be rejected for the given item.
    // Returns (true, reason) to reject, (false, "") to accept.
    // Called during autosearch result filtering (aggregator.FilterByCriteria path).
    // NOT called during RSS sync matching — the RSS matcher has its own candidate
    // filtering logic (see Task 5.7 architecture note).
    FilterRelease(ctx context.Context, release ReleaseForFilter, item SearchableItem) (reject bool, reason string)

    // TitlesMatch checks whether a parsed release title matches a media title.
    // Implementations define their own normalization and comparison logic.
    TitlesMatch(releaseTitle, mediaTitle string) bool

    // BuildSearchCriteria constructs indexer search parameters for a wanted item.
    // Returns criteria with the Newznab search type, external IDs, and categories.
    BuildSearchCriteria(item SearchableItem) SearchCriteria

    // IsGroupSearchEligible determines if a parent node should be searched as a group.
    // For TV: checks if all episodes in a season are missing (forUpgrade=false) or
    // all upgradable (forUpgrade=true). childIdentifier carries the season number
    // for TV, 0 for flat modules like Movie.
    // Merges spec §5.5's IsGroupSearchEligible and IsGroupUpgradeEligible into one
    // method via the forUpgrade parameter (see spec deviation note at end of phase).
    IsGroupSearchEligible(ctx context.Context, parentEntityType EntityType, parentID int64, childIdentifier int, forUpgrade bool) bool

    // SuppressChildSearches returns entity IDs of children that should be suppressed
    // after a successful group search (e.g., all episodes in a grabbed season pack).
    // parentID is the series ID, seasonNumber identifies the season.
    // This is a new formalization — the current codebase handles suppression implicitly
    // via dedup keys (seasonKeyForEpisode in rsssync/matcher.go). This method makes
    // the suppression logic explicit and module-owned.
    SuppressChildSearches(parentEntityType EntityType, parentID int64, seasonNumber int) []int64
}
```

**Key differences from Phase 0 skeleton:**
- `FilterRelease` takes `context.Context` and uses `ReleaseForFilter` (a read-only view of release data) instead of `Release` to avoid depending on `indexer/types` from the module package.
- `BuildSearchCriteria` returns `module.SearchCriteria` (defined below), not `indexer/types.SearchCriteria`, to keep the `module` package dependency-free.
- `IsGroupSearchEligible` merges the spec's `IsGroupSearchEligible` and `IsGroupUpgradeEligible` via a `forUpgrade bool` parameter, and takes `childIdentifier int` for the season number (see Task 5.6 for rationale). `IsUpgradeGroupEligible` from the spec is NOT a separate method.
- `SuppressChildSearches` takes `seasonNumber int` instead of a release — the suppression logic only needs the season number, not the full release.
- `SuppressChildSearches` is a new formalization of behavior that currently happens implicitly via dedup keys in the RSS sync matcher.

---

**Update** `internal/module/search_types.go` — add the types referenced by `SearchStrategy`:

```go
// ReleaseForFilter provides a read-only view of a release for module filtering.
// This avoids the module package depending on indexer/types.
type ReleaseForFilter struct {
    Title      string
    Year       int
    Season     int
    EndSeason  int
    Episode    int
    EndEpisode int
    IsTV       bool
    IsSeasonPack    bool
    IsCompleteSeries bool
    Quality    string
    Source     string
    Languages  []string
    Size       int64
    Categories []int
}

// SearchCriteria defines parameters for an indexer search, constructed by a module.
// The framework converts this to the indexer-specific types.SearchCriteria before dispatch.
type SearchCriteria struct {
    ModuleType Type   // Which module produced this criteria
    Query      string // Title to search for
    SearchType string // Newznab search type: "movie", "tvsearch", "audio", "book"
    Categories []int  // Newznab category IDs

    // External IDs (module sets whichever are relevant)
    ExternalIDs map[string]string // "imdbId" -> "tt1234567", "tmdbId" -> "550", etc.

    // Structured search parameters (TV season/episode, year, etc.)
    Year    int
    Season  int
    Episode int
}
```

Also add a `SearchParams` struct if not already defined (Phase 0 may have a skeleton):

```go
// SearchParams holds module-specific search parameters on a SearchableItem.
type SearchParams struct {
    Extra map[string]any // Module-specific fields (e.g., seriesId, seasonNumber for TV)
}
```

**Verify:** `go build ./internal/module/...` compiles cleanly.

---

### Task 5.2: Implement SearchStrategy on Movie and TV Modules

**Depends on:** Phase 4 complete (module stubs exist), Task 5.1 (interface finalized)

**Create** `internal/modules/movie/search.go`:

```go
package movie

import (
    "context"

    "github.com/slipstream/slipstream/internal/module"
    "github.com/slipstream/slipstream/internal/module/titleutil"
)

// Categories returns all movie Newznab categories (2000-2999).
func (m *Module) Categories() []int {
    return []int{2000, 2010, 2020, 2030, 2040, 2045, 2050, 2060, 2070, 2080}
}

// DefaultSearchCategories returns the default movie search categories.
func (m *Module) DefaultSearchCategories() []int {
    return []int{2000, 2030, 2040, 2045, 2050, 2080}
}

// FilterRelease rejects releases that don't match the target movie.
// Mirrors the logic in aggregator.go:getMovieFilterReason.
func (m *Module) FilterRelease(_ context.Context, release module.ReleaseForFilter, item module.SearchableItem) (bool, string) {
    if release.IsTV {
        return true, "is TV content, not movie"
    }
    if !titleutil.TitlesMatch(release.Title, item.GetTitle()) {
        return true, "title mismatch"
    }
    year := item.GetSearchParams().Extra["year"]
    if yearInt, ok := year.(int); ok && yearInt > 0 && release.Year > 0 {
        diff := yearInt - release.Year
        if diff < 0 {
            diff = -diff
        }
        if diff > 1 {
            return true, "year mismatch (>1 year difference)"
        }
    }
    return false, ""
}

// TitlesMatch checks if a release title matches a movie title.
func (m *Module) TitlesMatch(releaseTitle, mediaTitle string) bool {
    return titleutil.TitlesMatch(releaseTitle, mediaTitle)
}

// BuildSearchCriteria constructs Newznab search criteria for a movie.
// Uses Categories() (all movie categories), NOT DefaultSearchCategories().
// The current codebase sends ALL categories to indexers during autosearch
// (indexer.MovieCategories()). DefaultSearchCategories is for Prowlarr config only.
func (m *Module) BuildSearchCriteria(item module.SearchableItem) module.SearchCriteria {
    criteria := module.SearchCriteria{
        ModuleType:  module.TypeMovie,
        Query:       item.GetTitle(),
        SearchType:  "movie",
        Categories:  m.Categories(),
        ExternalIDs: item.GetExternalIDs(),
    }

    if year, ok := item.GetSearchParams().Extra["year"].(int); ok && year > 0 {
        criteria.Year = year
    }

    return criteria
}

// IsGroupSearchEligible — movies have no hierarchy, always false.
func (m *Module) IsGroupSearchEligible(_ context.Context, _ module.EntityType, _ int64, _ int, _ bool) bool {
    return false
}

// SuppressChildSearches — movies have no children.
func (m *Module) SuppressChildSearches(_ module.EntityType, _ int64, _ int) []int64 {
    return nil
}
```

---

**Create** `internal/modules/tv/search.go`:

```go
package tv

import (
    "context"

    "github.com/slipstream/slipstream/internal/decisioning"
    "github.com/slipstream/slipstream/internal/module"
    "github.com/slipstream/slipstream/internal/module/titleutil"
)

// Categories returns all TV Newznab categories (5000-5999).
func (m *Module) Categories() []int {
    return []int{5000, 5010, 5020, 5030, 5040, 5045, 5060, 5070, 5080, 5090}
}

// DefaultSearchCategories returns the default TV search categories.
func (m *Module) DefaultSearchCategories() []int {
    return []int{5000, 5030, 5040, 5045, 5070, 5090}
}

// FilterRelease rejects releases that don't match the target episode/season.
// Mirrors aggregator.go:getTVFilterReason — must preserve exact behavior.
func (m *Module) FilterRelease(_ context.Context, release module.ReleaseForFilter, item module.SearchableItem) (bool, string) {
    mediaType := item.GetMediaType()

    if !release.IsTV {
        return true, "not TV content"
    }
    if !titleutil.TVTitlesMatch(release.Title, item.GetTitle()) {
        return true, "title mismatch"
    }

    seasonNumber := m.extractSeasonNumber(item)
    episodeNumber := m.extractEpisodeNumber(item)

    // Season matching
    if seasonNumber > 0 {
        if release.IsSeasonPack || release.IsCompleteSeries {
            if release.Season > 0 && release.Season != seasonNumber {
                if release.EndSeason > 0 {
                    if seasonNumber < release.Season || seasonNumber > release.EndSeason {
                        return true, "season not in pack range"
                    }
                } else {
                    return true, "wrong season pack"
                }
            }
        } else if release.Season != seasonNumber {
            return true, "wrong season"
        }
    }

    // Episode matching (only for individual episode searches, not season packs)
    if mediaType == "episode" && episodeNumber > 0 {
        if release.IsSeasonPack || release.IsCompleteSeries {
            // Season packs cover all episodes — don't filter by episode
            return false, ""
        }
        if release.Episode > 0 && release.Episode != episodeNumber {
            if release.EndEpisode > 0 {
                if episodeNumber < release.Episode || episodeNumber > release.EndEpisode {
                    return true, "episode not in multi-episode range"
                }
            } else {
                return true, "wrong episode"
            }
        }
    }

    return false, ""
}

// TitlesMatch checks if a release title matches a series title using TV-specific matching.
func (m *Module) TitlesMatch(releaseTitle, mediaTitle string) bool {
    return titleutil.TVTitlesMatch(releaseTitle, mediaTitle)
}

// BuildSearchCriteria constructs Newznab search criteria for a TV episode or season.
func (m *Module) BuildSearchCriteria(item module.SearchableItem) module.SearchCriteria {
    mediaType := item.GetMediaType()
    criteria := module.SearchCriteria{
        ModuleType:  module.TypeTV,
        Query:       item.GetTitle(),
        SearchType:  "tvsearch",
        ExternalIDs: item.GetExternalIDs(),
    }

    seasonNumber := m.extractSeasonNumber(item)
    episodeNumber := m.extractEpisodeNumber(item)

    switch mediaType {
    case "episode":
        // Uses Categories() (all TV categories), NOT DefaultSearchCategories().
        // The current codebase sends ALL categories (indexer.TVCategories()) during
        // autosearch. DefaultSearchCategories is for Prowlarr config only.
        criteria.Categories = m.Categories()
        criteria.Season = seasonNumber
        criteria.Episode = episodeNumber
    case "season":
        // Don't set categories or season for season pack searches.
        // Some indexers categorize season packs differently, and setting
        // season would cause server-side filtering that excludes boxsets.
        // Client-side SelectBestRelease handles filtering.
    }

    return criteria
}

// IsGroupSearchEligible checks if a season is eligible for season pack search.
// childIdentifier is the season number. forUpgrade distinguishes missing vs upgrade.
func (m *Module) IsGroupSearchEligible(ctx context.Context, _ module.EntityType, seriesID int64, seasonNumber int, forUpgrade bool) bool {
    return false // Placeholder — actual dispatch happens in Task 5.6
}

// SuppressChildSearches returns episode IDs covered by a season pack grab.
func (m *Module) SuppressChildSearches(_ module.EntityType, _ int64, _ int) []int64 {
    return nil // Placeholder — actual dispatch happens in Task 5.6
}

func (m *Module) extractSeasonNumber(item module.SearchableItem) int {
    if v, ok := item.GetSearchParams().Extra["seasonNumber"].(int); ok {
        return v
    }
    return 0
}

func (m *Module) extractEpisodeNumber(item module.SearchableItem) int {
    if v, ok := item.GetSearchParams().Extra["episodeNumber"].(int); ok {
        return v
    }
    return 0
}
```

**Important:** The subagent must:
- Read the existing `internal/modules/tv/module.go` and `internal/modules/movie/module.go` to understand the Module struct
- Add compile-time assertions: `var _ module.SearchStrategy = (*Module)(nil)` in both module packages
- Ensure the Module struct has access to `*sqlc.Queries` and `*zerolog.Logger` for the TV group search methods (these should already be present from Phase 4)

**Verify:**
- `go build ./internal/modules/movie/...` compiles
- `go build ./internal/modules/tv/...` compiles
- `go vet ./internal/modules/...` clean

---

### Task 5.3: Extract Title Matching to Shared Utility Package

**Depends on:** Task 5.1

**Create** `internal/module/titleutil/` package with shared title matching functions.

Currently, title matching lives in `internal/indexer/search/title_match.go` with functions `TitlesMatch`, `TVTitlesMatch`, and `NormalizeTitle`. These are needed by:
- Module `SearchStrategy` implementations (Task 5.2)
- RSS sync `Matcher` (uses `search.NormalizeTitle`, `search.TVTitlesMatch`)
- Aggregator filter logic (uses `search.TitlesMatch`, `search.TVTitlesMatch`)

**Create** `internal/module/titleutil/titleutil.go`:

Move the following functions from `internal/indexer/search/title_match.go`:
- `NormalizeTitle(title string) string`
- `TitlesMatch(a, b string) bool`
- `TVTitlesMatch(a, b string) bool`
- Private helper `stripTrailingYear(normalized string) string`
- Module-level regex vars: `apostropheRegex`, `specialCharsRegex`, `multipleSpaceRegex`, `trailingYearRegex`

**Do NOT move** `CalculateTitleSimilarity` and its helpers (`buildTokenSet`, `calculateIntersection`, `calculateUnion`, `tokenize`) — they stay in the search package and use `NormalizeTitle` via the wrapper. They are debug/logging utilities, not module-relevant.

Read `internal/indexer/search/title_match.go` first to get the exact implementations.

**Update** `internal/indexer/search/title_match.go`:
- Keep the existing functions but change them to thin wrappers that delegate to `titleutil`:

```go
package search

import "github.com/slipstream/slipstream/internal/module/titleutil"

func NormalizeTitle(title string) string { return titleutil.NormalizeTitle(title) }
func TitlesMatch(a, b string) bool      { return titleutil.TitlesMatch(a, b) }
func TVTitlesMatch(a, b string) bool     { return titleutil.TVTitlesMatch(a, b) }
```

This ensures all existing callers (`rsssync/matcher.go`, `indexer/search/aggregator.go`, tests) continue to work without import changes.

**Verify:**
- `go build ./internal/module/titleutil/...` compiles
- `go build ./internal/indexer/search/...` compiles
- `go test ./internal/indexer/search/...` passes (existing title_match_test.go)
- `go build ./internal/rsssync/...` compiles

---

### Task 5.4: Add ModuleType to SearchCriteria and Refactor Criteria Building

**Depends on:** Tasks 5.1, 5.2, 5.3

**Update** `internal/indexer/types/types.go` — add `ModuleType` field to `SearchCriteria`:

```go
type SearchCriteria struct {
    ModuleType string `json:"moduleType,omitempty"` // NEW: module that generated this criteria
    Query      string `json:"query,omitempty"`
    Type       string `json:"type"`
    Categories []int  `json:"categories,omitempty"`
    // ... rest unchanged
}
```

**Add** a conversion function in `internal/module/search_types.go` or a new `internal/module/search_convert.go`:

```go
// ToIndexerCriteria converts module SearchCriteria to indexer types.SearchCriteria.
func (c *SearchCriteria) ToIndexerCriteria() *indexertypes.SearchCriteria {
    ic := &indexertypes.SearchCriteria{
        ModuleType: string(c.ModuleType),
        Query:      c.Query,
        Type:       c.SearchType,
        Categories: c.Categories,
        Year:       c.Year,
        Season:     c.Season,
        Episode:    c.Episode,
    }
    if v, ok := c.ExternalIDs["imdbId"]; ok {
        ic.ImdbID = v
    }
    if v, ok := c.ExternalIDs["tmdbId"]; ok {
        if id, err := strconv.Atoi(v); err == nil {
            ic.TmdbID = id
        }
    }
    if v, ok := c.ExternalIDs["tvdbId"]; ok {
        if id, err := strconv.Atoi(v); err == nil {
            ic.TvdbID = id
        }
    }
    return ic
}
```

**Important:** The `internal/module` package MUST NOT import `internal/indexer/types`. The `ToIndexerCriteria` example shown above is for illustration only — do NOT place it in the `module` package. Instead, define the conversion as a standalone function in the consuming code. The cleanest location is `internal/autosearch/convert.go` (already created in this phase for `moduleItemToSearchableItem`), where both `module` and `indexer/types` are available:

```go
// convertModuleCriteria converts module.SearchCriteria to indexer types.SearchCriteria.
// Lives in autosearch package (not module) to avoid module→indexer/types dependency.
func convertModuleCriteria(mc *module.SearchCriteria) types.SearchCriteria { ... }
```

**Update** `internal/autosearch/service.go` — refactor `buildSearchCriteria` to support module dispatch:

The existing `buildSearchCriteria(item *SearchableItem) types.SearchCriteria` method hard-codes movie/TV branches. Add a parallel method that uses the module registry:

```go
// buildSearchCriteriaFromModule constructs criteria using the module's SearchStrategy.
func (s *Service) buildSearchCriteriaFromModule(item module.SearchableItem) types.SearchCriteria {
    moduleType := module.Type(item.GetModuleType())
    mod := s.registry.Get(moduleType)
    if mod == nil {
        // Fallback: should not happen if registry is properly configured
        return types.SearchCriteria{Query: item.GetTitle()}
    }

    strategy, ok := mod.(module.SearchStrategy)
    if !ok {
        return types.SearchCriteria{Query: item.GetTitle()}
    }

    moduleCriteria := strategy.BuildSearchCriteria(item)
    // Convert module.SearchCriteria to types.SearchCriteria
    return convertModuleCriteria(&moduleCriteria)
}
```

The existing `buildSearchCriteria` method for `decisioning.SearchableItem` remains unchanged — it's still used by `SearchMovie` and `SearchEpisode` direct calls.

**Verify:**
- `go build ./internal/indexer/types/...` compiles
- `go build ./internal/autosearch/...` compiles

---

### Task 5.5: Refactor Indexer Selection for Module Awareness

**Depends on:** Task 5.1

**Update** `internal/indexer/types/types.go` — add `SupportsModule` method to `IndexerDefinition`:

```go
// SupportsModule checks if this indexer supports a module type by examining
// whether its category list includes any categories in the module's declared range.
// moduleCategories is the full list returned by SearchStrategy.Categories().
// Category ranges: movie=2000-2999, tv=5000-5999, audio=3000-3999, book=7000-7999.
func (d *IndexerDefinition) SupportsModule(moduleCategories []int) bool {
    if len(d.Categories) == 0 || len(moduleCategories) == 0 {
        return false
    }
    // Build a set of the module's declared categories for fast lookup
    moduleCatSet := make(map[int]bool, len(moduleCategories))
    for _, c := range moduleCategories {
        moduleCatSet[c] = true
    }
    // Check if any of the indexer's categories match a module category
    // or fall in the module's main range (e.g., 2000-2999 for movies).
    // Range check catches subcategories the module didn't explicitly list.
    moduleMin := moduleCategories[0] / 1000 * 1000 // e.g., 2000 for movies
    moduleMax := moduleMin + 999                    // e.g., 2999
    for _, c := range d.Categories {
        if moduleCatSet[c] || (c >= moduleMin && c <= moduleMax) {
            return true
        }
    }
    return false
}
```

**Update** `internal/indexer/search/service.go` — add module-aware indexer filtering:

Add a `registry` field to `Service` and a `SetRegistry` method (same pattern as Phase 3/4):

```go
type Service struct {
    indexerService *indexer.Service
    statusService  *status.Service
    rateLimiter    *ratelimit.Limiter
    broadcaster    contracts.Broadcaster
    logger         *zerolog.Logger
    registry       *module.Registry // NEW
}

func (s *Service) SetRegistry(r *module.Registry) {
    s.registry = r
}
```

Update `getIndexersForSearch` to support module-based lookup:

```go
func (s *Service) getIndexersForSearch(ctx context.Context, criteria *types.SearchCriteria) ([]*types.IndexerDefinition, error) {
    var indexers []*types.IndexerDefinition
    var err error

    // Module-aware path: use module's categories to determine support
    if criteria.ModuleType != "" && s.registry != nil {
        mod := s.registry.Get(module.Type(criteria.ModuleType))
        if mod != nil {
            if strategy, ok := mod.(module.SearchStrategy); ok {
                indexers, err = s.indexerService.ListEnabled(ctx)
                if err != nil {
                    return nil, err
                }
                cats := strategy.Categories()
                indexers = filterByModuleSupport(indexers, cats)
            }
        }
    }

    // Fallback: existing hard-coded path
    if indexers == nil {
        switch criteria.Type {
        case searchTypeMovie:
            indexers, err = s.indexerService.ListEnabledForMovies(ctx)
        case searchTypeTVSearch:
            indexers, err = s.indexerService.ListEnabledForTV(ctx)
        default:
            indexers, err = s.indexerService.ListEnabled(ctx)
        }
        if err != nil {
            return nil, err
        }
    }

    // ... rest of filtering unchanged (search support, disabled, rate-limited, sort by priority)
}

func filterByModuleSupport(indexers []*types.IndexerDefinition, moduleCategories []int) []*types.IndexerDefinition {
    filtered := make([]*types.IndexerDefinition, 0, len(indexers))
    for _, idx := range indexers {
        if idx.SupportsModule(moduleCategories) {
            filtered = append(filtered, idx)
        }
    }
    return filtered
}
```

**Verify:**
- `go build ./internal/indexer/...` compiles
- Existing `filterBySearchSupport` logic is not modified
- `go test ./internal/indexer/search/...` passes

---

### Task 5.6: Formalize Group Search via SearchStrategy

**Depends on:** Tasks 5.1, 5.2

**Update** `internal/modules/tv/search.go` — replace the placeholder group search methods from Task 5.2 with real implementations:

```go
// IsGroupSearchEligible checks if a season is eligible for season pack search.
// childIdentifier is the season number. forUpgrade selects the check:
// - forUpgrade=false: all monitored episodes are missing (IsSeasonPackEligible)
// - forUpgrade=true: all monitored episodes are upgradable (IsSeasonPackUpgradeEligible)
func (m *Module) IsGroupSearchEligible(ctx context.Context, _ module.EntityType, seriesID int64, seasonNumber int, forUpgrade bool) bool {
    if forUpgrade {
        return decisioning.IsSeasonPackUpgradeEligible(ctx, m.queries, m.logger, seriesID, seasonNumber)
    }
    return decisioning.IsSeasonPackEligible(ctx, m.queries, m.logger, seriesID, seasonNumber)
}
```

The `childIdentifier` parameter (third int) carries the season number. This was settled in the Task 5.1 interface definition with the merged `forUpgrade` parameter approach.

**Spec deviation:** The spec §5.5 has separate `IsGroupSearchEligible` and `IsGroupUpgradeEligible`. We merge them with a `forUpgrade` parameter to match the codebase's two functions (`decisioning.IsSeasonPackEligible` and `decisioning.IsSeasonPackUpgradeEligible`) that differ only in the check logic.

**Verify:**
- Updated interface compiles
- Movie module's no-op implementation compiles
- TV module's implementation delegates correctly to `decisioning.IsSeasonPackEligible` / `IsSeasonPackUpgradeEligible`

---

### Task 5.7: Refactor RSS Sync for Module Dispatch

**Depends on:** Tasks 5.3, 5.4, 5.6

This is the most complex task in Phase 5. The RSS sync service (`internal/rsssync/`) must be updated to:
1. Collect wanted items from module `WantedCollector` implementations instead of `decisioning.CollectWantedItems`
2. Re-type the `WantedIndex` and `Matcher` to use `module.SearchableItem` interface instead of `decisioning.SearchableItem` struct
3. Use module `SearchStrategy.IsGroupSearchEligible` for season pack eligibility checks instead of calling `decisioning.IsSeasonPackEligible` directly
4. **Note:** The matcher does NOT use `SearchStrategy.FilterRelease` or `TitlesMatch` — it has its own candidate-matching logic (see architecture note below)

**Update** `internal/rsssync/service.go`:

Add `registry` field and `SetRegistry` method:

```go
type Service struct {
    // ... existing fields
    registry *module.Registry // NEW
}

func (s *Service) SetRegistry(r *module.Registry) {
    s.registry = r
}
```

Modify `collectWanted` to use module WantedCollectors when registry is available:

```go
func (s *Service) collectWanted(ctx context.Context, start time.Time) ([]module.SearchableItem, error) {
    if s.registry != nil {
        return s.collectWantedFromModules(ctx, start)
    }
    // Legacy fallback
    return s.collectWantedLegacy(ctx, start)
}

func (s *Service) collectWantedFromModules(ctx context.Context, start time.Time) ([]module.SearchableItem, error) {
    var allItems []module.SearchableItem
    for _, mod := range s.registry.Enabled() {
        collector, ok := mod.(module.WantedCollector)
        if !ok {
            continue
        }
        missing, err := collector.CollectMissing(ctx)
        if err != nil {
            s.logger.Warn().Err(err).Str("module", string(mod.(module.Descriptor).ID())).Msg("failed to collect missing items")
            continue
        }
        allItems = append(allItems, missing...)

        upgradable, err := collector.CollectUpgradable(ctx)
        if err != nil {
            s.logger.Warn().Err(err).Str("module", string(mod.(module.Descriptor).ID())).Msg("failed to collect upgradable items")
            continue
        }
        allItems = append(allItems, upgradable...)
    }

    if len(allItems) == 0 {
        s.logger.Info().Msg("no wanted items from modules, skipping RSS sync")
        s.setStatus(&SyncStatus{LastRun: start})
        s.broadcast(EventCompleted, CompletedEvent{ElapsedMs: int(time.Since(start).Milliseconds())})
        return nil, nil
    }
    return allItems, nil
}
```

**Update** `internal/rsssync/matcher.go`:

The `WantedIndex` and `Matcher` need to work with `module.SearchableItem` instead of `decisioning.SearchableItem`. This is a type change that ripples through the matcher.

**Architecture note on RSS matcher vs. aggregator filtering:**

The RSS sync matcher (`rsssync/matcher.go`) and the autosearch aggregator (`indexer/search/aggregator.go`) have DIFFERENT filtering approaches:
- **Aggregator** (`FilterByCriteria`): Parses each release title via `scanner.ParseFilename`, then calls `getMovieFilterReason`/`getTVFilterReason` to validate title match, year, season, episode. This is where `SearchStrategy.FilterRelease` replaces the hard-coded logic.
- **RSS matcher** (`Matcher.Match`): Uses a two-phase approach: (1) index lookup by external ID or normalized title to find candidates, (2) candidate filtering by media type + year/season/episode match. The matcher does NOT use `FilterRelease` — it has its own simpler candidate-checking logic in `matchMovie`, `matchEpisode`, `matchSeasonPack`.

The RSS matcher's `findCandidates` uses `search.NormalizeTitle` for index building but NOT `TitlesMatch`/`TVTitlesMatch` for matching — exact normalized equality is used via map lookup. The module `TitlesMatch` is only relevant for the aggregator path.

**Key changes:**
- `WantedIndex` stores `module.SearchableItem` slices in its maps
- `BuildWantedIndex` accepts `[]module.SearchableItem`
- `MatchResult` uses `module.SearchableItem`
- `Matcher.matchMovie`, `matchEpisode`, `matchSeasonPack` access item data via `module.SearchableItem` getters instead of struct fields
- `Matcher` needs access to the module `Registry` for `checkSeasonPackEligibility` (to call `SearchStrategy.IsGroupSearchEligible` instead of `decisioning.IsSeasonPackEligible` directly)
- The matcher does NOT dispatch to `SearchStrategy.FilterRelease` — its candidate filtering logic is preserved as-is, just re-typed

```go
type Matcher struct {
    index    *WantedIndex
    queries  *sqlc.Queries
    logger   *zerolog.Logger
    registry *module.Registry // NEW
}

func NewMatcher(index *WantedIndex, queries *sqlc.Queries, logger *zerolog.Logger, registry *module.Registry) *Matcher {
    return &Matcher{
        index:    index,
        queries:  queries,
        logger:   logger,
        registry: registry,
    }
}
```

The `findCandidates` method accesses external IDs via `item.GetExternalIDs()` instead of direct struct fields. The `matchMovie` method accesses year via `item.GetSearchParams().Extra["year"]`, `matchEpisode` accesses season/episode via `Extra["seasonNumber"]`/`Extra["episodeNumber"]`, etc. The `BuildWantedIndex` function uses `titleutil.NormalizeTitle` (via the search package wrapper) for title indexing — this does not change.

**Important constraint:** The existing `Matcher.Match` method has nuanced logic for season pack matching (checking `IsSeasonPackEligible` for episode candidates, constructing synthetic season items). This logic MUST be preserved exactly. The refactor changes the type plumbing, not the matching logic.

**Synthetic season item handling:** The `checkSeasonPackEligibility` method currently creates synthetic `decisioning.SearchableItem` structs with `MediaTypeSeason` when a season pack release matches episode candidates. When `WantedIndex` is updated to use `module.SearchableItem`, these synthetic items must become `module.WantedItem` instances instead. The conversion is straightforward — populate the `WantedItem` fields from the episode item's data:

```go
func (m *Matcher) checkSeasonPackEligibility(ctx context.Context, episodeItem module.SearchableItem, season int) module.SearchableItem {
    params := episodeItem.GetSearchParams()
    seriesID, _ := params.Extra["seriesId"].(int64)

    // Use module SearchStrategy for eligibility check when registry is available
    var eligible bool
    if m.registry != nil {
        mod := m.registry.Get(module.Type(episodeItem.GetModuleType()))
        if strategy, ok := mod.(module.SearchStrategy); ok {
            eligible = strategy.IsGroupSearchEligible(ctx, "series", seriesID, season, false)
            if !eligible {
                // Check upgrade eligibility
                eligible = strategy.IsGroupSearchEligible(ctx, "series", seriesID, season, true)
                if eligible {
                    // Build upgrade season item with highest quality
                    return m.buildSyntheticSeasonItem(episodeItem, seriesID, season, true)
                }
            }
        }
    } else {
        // Legacy fallback
        eligible = decisioning.IsSeasonPackEligible(ctx, m.queries, m.logger, seriesID, season)
        if !eligible {
            eligible = decisioning.IsSeasonPackUpgradeEligible(ctx, m.queries, m.logger, seriesID, season)
            if eligible {
                return m.buildSyntheticSeasonItem(episodeItem, seriesID, season, true)
            }
        }
    }

    if !eligible {
        return nil
    }
    return m.buildSyntheticSeasonItem(episodeItem, seriesID, season, false)
}

func (m *Matcher) buildSyntheticSeasonItem(episodeItem module.SearchableItem, seriesID int64, season int, forUpgrade bool) module.SearchableItem {
    item := &module.WantedItem{
        ModuleType_:       module.Type(episodeItem.GetModuleType()),
        MediaType_:        "season",
        EntityID_:         seriesID,
        Title_:            episodeItem.GetTitle(),
        ExternalIDs_:      episodeItem.GetExternalIDs(),
        QualityProfileID_: episodeItem.GetQualityProfileID(),
        SearchParams_:     module.SearchParams{Extra: map[string]any{
            "seriesId":     seriesID,
            "seasonNumber": season,
        }},
    }
    if forUpgrade {
        highestQID := m.highestQualityForSeason(episodeItem, seriesID, season)
        item.CurrentQualityID_ = &highestQID
    }
    return item
}
```

**Backward compatibility:** When `registry` is nil, fall back to the existing hard-coded matching (calling `decisioning.IsSeasonPackEligible` / `IsSeasonPackUpgradeEligible` directly). This ensures the service works even if the module system is partially initialized.

**Verify:**
- `go build ./internal/rsssync/...` compiles
- `go test ./internal/rsssync/...` passes (existing matcher_test.go, scoring_selection_test.go)
- The existing test helpers in `testhelpers_test.go` may need updates for the new types

---

### Task 5.8: Refactor Autosearch Scheduled Task for Module Dispatch

**Depends on:** Tasks 5.3, 5.4, 5.6

The scheduled autosearch (`internal/autosearch/scheduled.go`) currently collects items directly via sqlc queries. In this phase, when a module registry is available, it uses the module `WantedCollector` implementations instead.

**Critical constraint:** The current `collectSearchableItems` does three things beyond raw collection that MUST be preserved:
1. **Backoff checking** — each item is checked via `shouldSkipItem` (queries `autosearch_status` table) before inclusion
2. **Release date extraction** — items are sorted by release date (newest first) for search priority
3. **Season pack grouping** — missing/upgradable episodes are grouped by season; if all episodes in a season qualify, a single `MediaTypeSeason` item replaces the individual episodes (via `tryAddMissingSeasonPackSearch` for missing, `isSeasonPackUpgradeEligible` for upgrades)

The module `WantedCollector` returns raw items without backoff or season grouping. The `collectFromModules` method must apply these transformations after collection. Additionally, `module.SearchableItem` has no `GetReleaseDate()` method — release dates must be extracted from `SearchParams.Extra` (the WantedCollector implementations should populate `Extra["releaseDate"]` as `time.Time`).

**Update** `internal/autosearch/scheduled.go`:

Add `registry` field to `ScheduledSearcher`:

```go
type ScheduledSearcher struct {
    // ... existing fields
    registry *module.Registry // NEW
}

func (s *ScheduledSearcher) SetRegistry(r *module.Registry) {
    s.registry = r
}
```

Modify `collectSearchableItems` to use module WantedCollectors:

```go
func (s *ScheduledSearcher) collectSearchableItems(ctx context.Context) ([]SearchableItem, error) {
    if s.registry != nil {
        return s.collectFromModules(ctx)
    }
    return s.collectLegacy(ctx)
}
```

`collectFromModules` converts `module.SearchableItem` results to `decisioning.SearchableItem` (the type the rest of autosearch expects). This is a temporary bridge — full conversion to `module.SearchableItem` throughout autosearch is deferred to Phase 10.

**Important:** The conversion must preserve backoff checking, release date sorting, and season pack grouping. The implementation follows the same structure as the existing `collectMissingMovies` / `collectMissingEpisodes` / `collectUpgradeMovies` / `collectUpgradeEpisodes` methods, but sourcing from module WantedCollectors instead of sqlc queries.

```go
func (s *ScheduledSearcher) collectFromModules(ctx context.Context) ([]SearchableItem, error) {
    var items []searchableItemWithPriority

    for _, mod := range s.registry.Enabled() {
        collector, ok := mod.(module.WantedCollector)
        if !ok {
            continue
        }
        strategy, hasStrategy := mod.(module.SearchStrategy)

        modType := mod.(module.Descriptor).ID()

        // Collect missing items
        missing, err := collector.CollectMissing(ctx)
        if err != nil {
            s.logger.Warn().Err(err).Str("module", string(modType)).Msg("failed to collect missing items from module")
            continue
        }
        missingItems := s.convertAndFilterModuleItems(ctx, missing, "missing")

        // Season pack grouping for modules that support group search
        if hasStrategy {
            missingItems = s.groupIntoSeasonPacks(ctx, missingItems, strategy, false)
        }
        items = append(items, missingItems...)

        // Collect upgradable items
        upgradable, err := collector.CollectUpgradable(ctx)
        if err != nil {
            s.logger.Warn().Err(err).Str("module", string(modType)).Msg("failed to collect upgradable items from module")
            continue
        }
        upgradeItems := s.convertAndFilterModuleItems(ctx, upgradable, "upgrade")

        // Season pack grouping for upgrades
        if hasStrategy {
            upgradeItems = s.groupIntoSeasonPacks(ctx, upgradeItems, strategy, true)
        }
        items = append(items, upgradeItems...)
    }

    // Sort by release date (newest first) — same as existing collectSearchableItems
    sort.Slice(items, func(i, j int) bool {
        return items[i].releaseDate.After(items[j].releaseDate)
    })

    result := make([]SearchableItem, len(items))
    for i := range items {
        result[i] = items[i].item
    }
    return result, nil
}

// convertAndFilterModuleItems converts module items to legacy SearchableItems,
// applying backoff filtering. Extracts release dates from SearchParams.Extra["releaseDate"].
func (s *ScheduledSearcher) convertAndFilterModuleItems(
    ctx context.Context,
    moduleItems []module.SearchableItem,
    searchType string,
) []searchableItemWithPriority {
    var items []searchableItemWithPriority
    for _, mi := range moduleItems {
        converted := moduleItemToSearchableItem(mi)

        // Backoff check — same logic as existing shouldSkipItem
        itemType := string(converted.MediaType)
        shouldSkip, err := s.shouldSkipItem(ctx, itemType, converted.MediaID, searchType)
        if err != nil {
            s.logger.Warn().Err(err).Str("itemType", itemType).Int64("mediaId", converted.MediaID).Msg("Failed to check backoff status")
        }
        if shouldSkip {
            continue
        }

        // Extract release date from SearchParams for priority sorting
        var releaseDate time.Time
        if rd, ok := mi.GetSearchParams().Extra["releaseDate"].(time.Time); ok {
            releaseDate = rd
        }

        items = append(items, searchableItemWithPriority{
            item:        converted,
            releaseDate: releaseDate,
        })
    }
    return items
}

// groupIntoSeasonPacks groups episode items by series+season and replaces them
// with a single season pack item when all episodes qualify for group search.
// Mirrors the existing collectMissingEpisodes / collectUpgradeEpisodes logic.
func (s *ScheduledSearcher) groupIntoSeasonPacks(
    ctx context.Context,
    items []searchableItemWithPriority,
    strategy module.SearchStrategy,
    forUpgrade bool,
) []searchableItemWithPriority {
    // Separate non-episode items (pass through) from episode items (group candidates)
    var result []searchableItemWithPriority
    episodesBySeason := make(map[seasonKey][]searchableItemWithPriority)

    for _, item := range items {
        if item.item.MediaType != MediaTypeEpisode {
            result = append(result, item)
            continue
        }
        key := seasonKey{seriesID: item.item.SeriesID, seasonNumber: int64(item.item.SeasonNumber)}
        episodesBySeason[key] = append(episodesBySeason[key], item)
    }

    for key, episodes := range episodesBySeason {
        // Use module's SearchStrategy to check group eligibility
        if strategy.IsGroupSearchEligible(ctx, "series", key.seriesID, int(key.seasonNumber), forUpgrade) {
            // Build a synthetic season pack item from first episode's data
            first := episodes[0].item
            seasonItem := SearchableItem{
                MediaType:        MediaTypeSeason,
                MediaID:          key.seriesID,
                SeriesID:         key.seriesID,
                Title:            first.Title,
                SeasonNumber:     int(key.seasonNumber),
                TvdbID:           first.TvdbID,
                TmdbID:           first.TmdbID,
                ImdbID:           first.ImdbID,
                QualityProfileID: first.QualityProfileID,
            }
            if forUpgrade {
                seasonItem.HasFile = true
                // Find highest quality among episodes for upgrade check
                maxQID := 0
                for _, ep := range episodes {
                    if ep.item.CurrentQualityID > maxQID {
                        maxQID = ep.item.CurrentQualityID
                    }
                }
                seasonItem.CurrentQualityID = maxQID
            }
            result = append(result, searchableItemWithPriority{
                item:         seasonItem,
                releaseDate:  episodes[0].releaseDate,
                isSeasonPack: true,
            })
        } else {
            // Not eligible for group search — keep individual episodes
            result = append(result, episodes...)
        }
    }

    return result
}
```

**WantedCollector contract update:** Module `WantedCollector.CollectMissing` and `CollectUpgradable` implementations MUST populate `SearchParams.Extra["releaseDate"]` with a `time.Time` value for priority sorting. For movies: earliest of physical/digital release date. For TV episodes: air date. This was implicit in the existing sqlc-based collection but must be explicit for the module interface.

**Create** `internal/autosearch/convert.go` — conversion helper:

```go
package autosearch

import (
    "strconv"

    "github.com/slipstream/slipstream/internal/decisioning"
    "github.com/slipstream/slipstream/internal/module"
)

// moduleItemToSearchableItem converts a module.SearchableItem to the legacy
// decisioning.SearchableItem struct. This is a temporary bridge — full migration
// to module.SearchableItem throughout the autosearch pipeline is deferred to Phase 10.
func moduleItemToSearchableItem(item module.SearchableItem) decisioning.SearchableItem {
    result := decisioning.SearchableItem{
        MediaType: decisioning.MediaType(item.GetMediaType()),
        MediaID:   item.GetEntityID(),
        Title:     item.GetTitle(),
    }

    // Quality
    result.QualityProfileID = item.GetQualityProfileID()
    if qid := item.GetCurrentQualityID(); qid != nil {
        result.HasFile = true
        result.CurrentQualityID = *qid
    }

    // External IDs
    ids := item.GetExternalIDs()
    if v, ok := ids["imdbId"]; ok {
        result.ImdbID = v
    }
    if v, ok := ids["tmdbId"]; ok {
        if id, err := strconv.Atoi(v); err == nil {
            result.TmdbID = id
        }
    }
    if v, ok := ids["tvdbId"]; ok {
        if id, err := strconv.Atoi(v); err == nil {
            result.TvdbID = id
        }
    }

    // Search params
    params := item.GetSearchParams()
    if v, ok := params.Extra["year"].(int); ok {
        result.Year = v
    }
    if v, ok := params.Extra["seriesId"].(int64); ok {
        result.SeriesID = v
    }
    if v, ok := params.Extra["seasonNumber"].(int); ok {
        result.SeasonNumber = v
    }
    if v, ok := params.Extra["episodeNumber"].(int); ok {
        result.EpisodeNumber = v
    }

    return result
}
```

**Verify:**
- `go build ./internal/autosearch/...` compiles
- The existing `SearchMovie`, `SearchEpisode`, `SearchSeason`, `SearchSeries` methods are NOT modified — they continue to use the direct service calls and `decisioning.SearchableItem`
- Only the `ScheduledSearcher.collectSearchableItems` gains the new module path

---

### Task 5.9: Wire Integration

**Depends on:** All prior tasks

**Update** `internal/api/setters.go` — add `SetRegistry` calls for the new services:

```go
// In the SetRegistry function (or wherever registry is distributed):
searchService.SetRegistry(registry)
rssSyncService.SetRegistry(registry)
scheduledSearcher.SetRegistry(registry)
```

**Update** `internal/api/wire.go` if constructor signatures changed (unlikely — we're using setter injection for the registry, not constructor injection).

**Run** `make wire` to regenerate `wire_gen.go`.

**Verify:**
- `make build` succeeds
- `make wire` succeeds without errors
- `go vet ./...` clean

---

### Task 5.10: Validation

**Run full validation:**

```bash
make build && make test && make lint
```

**Manual smoke test checklist (for the user, not automated):**

- Manual movie search → returns results, scored and filtered correctly
- Manual episode search → returns results with correct S/E filtering
- Manual season search → returns season packs, falls back to individual episodes
- RSS sync trigger → matches releases against wanted items, grabs best
- Scheduled autosearch → collects items from all enabled modules, searches sequentially
- Indexer with only movie categories → does NOT appear in TV searches
- Indexer with both movie and TV categories → appears in both

**Specific regression checks:**
- `buildSearchCriteria` for movies still sets `criteria.Type = "movie"`, `criteria.ImdbID`, `criteria.TmdbID`, `criteria.Year`, and uses ALL movie categories (not just defaults)
- `buildSearchCriteria` for episodes still sets `criteria.Type = "tvsearch"`, `criteria.TvdbID`, `criteria.Season`, `criteria.Episode`, and uses ALL TV categories
- `buildSearchCriteria` for season packs still omits Season and Categories (for boxset discovery)
- `FilterByCriteria` for movies still rejects TV content, checks title match, allows ±1 year tolerance
- `FilterByCriteria` for TV still rejects non-TV content, checks season/episode match, handles season packs and multi-episode ranges
- `SelectBestRelease` for season packs still rejects non-season-pack releases, accepts complete series boxsets (NOT refactored in this phase — retains hard-coded logic)
- RSS sync season pack matching still checks eligibility (via module `IsGroupSearchEligible` when registry available, `decisioning.IsSeasonPackEligible` as fallback) and creates synthetic season items
- RSS sync suppresses individual episodes covered by a grabbed season pack
- E01 season pack fallback in `SearchEpisode` still works for missing episodes only (not upgradable)
- Scheduled autosearch correctly applies backoff, release-date sorting, and season pack grouping when collecting via module WantedCollectors
- Dev mode toggle → search services still work after DB switch

**Spec deviations documented in this phase:**
1. `SearchStrategy.IsGroupSearchEligible` takes `childIdentifier int` parameter (not in spec §5.5) to pass season number for TV eligibility checks
2. `IsGroupSearchEligible` and `IsGroupUpgradeEligible` (spec §5.5) are merged into a single method with a `forUpgrade bool` parameter
3. `SuppressChildSearches` takes `seasonNumber int` instead of `grabbedRelease Release` (spec §5.5) — the suppression logic only needs the season number. This is a new formalization of behavior that currently happens implicitly via dedup keys
4. `decisioning.SearchableItem` struct is NOT removed in this phase — it coexists with `module.SearchableItem` interface. Full migration deferred to Phase 10
5. The autosearch `ScheduledSearcher` converts `module.SearchableItem` to `decisioning.SearchableItem` via a bridge function. This is temporary — Phase 10 migrates the autosearch pipeline to use `module.SearchableItem` directly
6. Scoring (`scoring.Scorer`) remains unchanged and framework-owned — it is not module-specific per this phase. Future modules may require scoring extensions
7. `FilterRelease` uses `ReleaseForFilter` (a value struct in the `module` package) instead of `indexer/types.TorrentInfo` to avoid coupling the module package to indexer types. The framework converts before calling the module
8. `SelectBestRelease` is NOT refactored to use `SearchStrategy` in this phase (deferred to Phase 10). It retains its hard-coded TV-specific logic in `shouldSkipTVRelease`
9. RSS sync matcher does NOT use `SearchStrategy.FilterRelease` or `TitlesMatch` for candidate matching — it retains its own index-lookup + candidate-check logic. Only `IsGroupSearchEligible` is dispatched through the module for season pack eligibility
10. `WantedCollector` implementations must populate `SearchParams.Extra["releaseDate"]` as `time.Time` for the autosearch scheduled task's release-date priority sorting

---

### Phase 5 File Inventory

### New files created in Phase 5
```
internal/module/titleutil/titleutil.go (shared title matching functions)
internal/modules/movie/search.go (Movie SearchStrategy implementation)
internal/modules/tv/search.go (TV SearchStrategy implementation)
internal/autosearch/convert.go (module.SearchableItem → decisioning.SearchableItem bridge)
```

### Existing files modified in Phase 5

**Backend — module framework types:**
```
internal/module/interfaces.go (finalize SearchStrategy interface — merged group search methods, added childIdentifier)
internal/module/search_types.go (add ReleaseForFilter, SearchCriteria, SearchParams types)
```

**Backend — module stubs:**
```
internal/modules/movie/module.go (add compile-time assertion: var _ module.SearchStrategy = (*Module)(nil))
internal/modules/tv/module.go (add compile-time assertion: var _ module.SearchStrategy = (*Module)(nil))
```

**Backend — title matching:**
```
internal/indexer/search/title_match.go (delegate to titleutil, keep as thin wrappers)
```

**Backend — indexer types:**
```
internal/indexer/types/types.go (add ModuleType field to SearchCriteria, add SupportsModule method to IndexerDefinition)
```

**Backend — search service:**
```
internal/indexer/search/service.go (add registry field, SetRegistry, module-aware getIndexersForSearch)
```

**Backend — RSS sync:**
```
internal/rsssync/service.go (add registry field, SetRegistry, module-aware collectWanted)
internal/rsssync/matcher.go (add registry to Matcher, module-aware type plumbing)
```

**Backend — autosearch:**
```
internal/autosearch/scheduled.go (add registry field, SetRegistry, module-aware collectSearchableItems)
```

**Backend — Wire DI:**
```
internal/api/setters.go (add SetRegistry calls for search, RSS sync, scheduled searcher)
internal/api/wire_gen.go (regenerated via make wire)
```

---

