## Phase 6: File Pipeline (Parsing, Paths, Naming, Import)

**Goal:** Extract the full file management pipeline — parsing, path generation, naming, and import — into module-provided implementations backed by a shared utility library. After this phase, modules own their filename parsing, path template declarations, naming configuration, and import matching/record creation. The framework's import pipeline dispatches through module interfaces instead of hard-coding movie/TV branches.

**Spec sections covered:** §8.1 (Download Categories), §8.2 (Download Mappings — wired here, schema done in Phase 1), §8.3 (ImportHandler), §8.4 (FileParser + shared utility library + orphan scan TryMatch), §8.5 (PathGenerator + conditional segments + special node paths), §8.6 (NamingProvider + token contexts + format options + naming settings storage), §18.1 (Multi-Entity Files), §18.2 (Special Nodes — path handling), §18.3 (Format Sub-Variants), §18.4 (Conditional Path Segments)

**Depends on:** Phase 5 complete (SearchStrategy implementations exist, module registry is fully wired into the startup path, all earlier interfaces are implemented on the movie/TV Module structs)

### Phase 6 Agent Context Management

This is the **largest phase** in the plan. It touches the import pipeline (`internal/import/`), the scanner/parser (`internal/library/scanner/`), the renamer (`internal/import/renamer/`), the organizer (`internal/library/organizer/`), the downloader (`internal/downloader/`), and the module packages (`internal/modules/movie/`, `internal/modules/tv/`).

**Execution strategy:**

1. **Task 6.1** (types) has no within-phase deps — start here.
2. After 6.1: **Tasks 6.2, 6.5, 6.6, 6.8, 6.9** can all run in **parallel** (they depend on 6.1 but not each other).
3. **Tasks 6.3 and 6.4** depend on 6.2 and can run in **parallel** with each other.
4. **Task 6.7** depends on 6.5 + 6.6 (needs NamingProvider implementations to know setting keys).
5. **Task 6.10** is the big integration task — depends on 6.3, 6.4, 6.7, 6.8, 6.9.
6. **Task 6.11** depends on 6.3 + 6.4 (FileParser TryMatch implementations).
7. Tasks 6.10 and 6.11 can run in **parallel** once their deps are met.
8. **Task 6.12** (Wire DI) depends on 6.10 + 6.11.
9. **Task 6.13** (validation) runs last.

**Context budget:** The import pipeline (`pipeline.go` ~1600 lines), import matching (`matching.go` ~850 lines), and scanner parser (`parser.go` ~620 lines) are large files. Do NOT read these into main context. Delegate all code changes to Sonnet subagents. Each subagent gets the specific task description, relevant file paths, and exact function signatures to produce. Main context only reads small verification files and runs compile checks.

**Critical safety rule for all subagents:** Include in every subagent prompt: "The import pipeline is bug-sensitive — erroneous changes can cause repeated bad downloads or lost files. Read `internal/CLAUDE.md` before making changes. Preserve all existing behavior when the module registry is nil (dual-path dispatch). Do NOT use git stash or any commands that affect the entire worktree."

**Code block accuracy note:** Inline code blocks in this phase are **conceptual/pseudocode**. When they conflict with Implementation Notes or Architecture Decisions, the notes/decisions take precedence. Key known simplifications in code blocks: `rootFolderSvc.GetPath(ctx, id)` is shorthand for `rootFolderSvc.Get(ctx, id) → rf.Path` (see AD #11); the import package is `importer` (not `import`); `resolvePattern` takes 2 args `(pattern, ctx)` not 3.

---

### Phase 6 Architecture Decisions

1. **Dual-path dispatch (consistent with Phases 3–5).** The import service gains a `registry` field. When set, the pipeline dispatches to module `ImportHandler`, `FileParser`, and uses per-module renamer instances. When nil, all paths fall back to the existing hard-coded behavior. This means the system works identically with or without the module framework wired in.

2. **Shared parsing utilities extracted to `internal/module/parseutil/`.** The current `scanner/parser.go` contains both shared utilities (quality parsing, title cleaning, year extraction, codec detection, HDR detection, audio parsing, release group extraction) and media-type-specific pattern matching (S01E02 for TV, Title (Year) for movies). Phase 6 extracts the shared utilities to a new `parseutil` package. Module `FileParser` implementations compose these utilities with their own pattern matching. The scanner package is NOT removed — it continues to work for library scanning. FileParser wraps and delegates to it for now; full scanner refactoring is deferred to Phase 10.

3. **`ParseResult` uses common fields + `Extra any` for module-specific data.** The framework-level `module.ParseResult` struct has common fields (title, year, quality, source, codec, HDR, audio, release group, revision, edition, languages). Module-specific parsed data (TV's season/episode numbers, season pack flags) goes in the `Extra any` field. The framework only inspects common fields (for quality matching). Module code type-asserts Extra when needed (e.g., TV's `MatchToEntity` reads season/episode from Extra). **Spec deviation (§18.1):** The spec says `FileParser.ParseFilename()` returns "a list of matched entity identifiers." This plan returns a single `ParseResult` instead — multi-entity info is carried via `TVParseExtra.EndEpisode`, and `MatchedEntity.EntityIDs` is populated later during `MatchToEntity`/`MatchIndividualFile`. This is more practical because parsing a filename doesn't know entity IDs (those require DB lookup).

4. **`MatchedEntity.TokenData` carries template variable values.** When `ImportHandler.MatchDownload` returns matched entities, each entity's `TokenData map[string]any` field is pre-populated with the module's template variable values (e.g., movie: `MovieTitle`, `MovieYear`; TV: `SeriesTitle`, `SeasonNumber`, `EpisodeNumber`, `EpisodeTitle`, etc.). The framework converts TokenData to a `renamer.TokenContext` via a centralized `buildTokenContextFromData()` function that maps well-known key names to struct fields — no type switch on module type needed. The framework then adds quality/media-info fields and passes the TokenContext to the renamer.

5. **Per-module renamer instances.** The import service maintains a `map[module.Type]*renamer.Resolver` loaded from `module_naming_settings`. Each module's naming patterns, colon replacement, and format options are stored per-module. The renamer gains a `ResolveContext(contextName string, ctx *TokenContext, ext string) (string, error)` method that looks up patterns from a `Patterns map[string]string` field on Settings. Existing type-specific methods (`ResolveMovieFolderName`, etc.) remain as convenience wrappers for backward compatibility. Context names follow the convention `<level>-folder` / `<level>-file` with optional `.variant` suffix for format sub-variants (e.g., `episode-file.daily`).

6. **Generic path computation via node schema walk.** The import pipeline's `computeDestination` gains a module-aware path that walks the module's `NodeSchema` levels from root to leaf: root level → root-folder template, intermediate levels → level-folder template (with conditional segment and special node checks), leaf level → level-file template. For flat modules (movie: 1 level, both root+leaf), the path is `rootFolder/folder/filename`. For hierarchical modules (TV: 3 levels), the path is `rootFolder/seriesFolder/seasonFolder/filename`. This is fully data-driven from the schema — no switch on module type.

7. **Conditional path segments are per-entity, not per-module.** The spec §18.4 says conditions are "per-module settings." However, the current `series.season_folder` column is per-entity (each series independently chooses season folders). Phase 6 preserves per-entity behavior: `PathGenerator.ConditionalSegments()` declares what conditions exist (for the settings UI), but the actual value comes from `TokenData` (e.g., `TokenData["SeasonFolder"] = true/false`). The path builder reads from TokenData. This is noted as a pragmatic deviation from the spec — global per-module toggles would break existing behavior.

8. **Format sub-variants use context name suffixes.** TV's episode level has `FormatVariants: ["standard", "daily", "anime"]`. The naming settings store variant-specific patterns under keys like `episode-file.standard`, `episode-file.daily`, `episode-file.anime`. The path builder determines the variant from `TokenData["SeriesType"]` and resolves the variant-specific context name (falling back to the base `episode-file` if no variant pattern exists).

9. **`module_naming_settings` table replaces naming columns in `import_settings`.** A new `module_naming_settings` table with `(module_type, setting_key, setting_value)` rows stores all per-module naming configuration. The migration copies existing values from `import_settings` naming columns into the new table (one set of rows for movie, one for TV). The naming columns on `import_settings` are NOT dropped (Phase 10 cleanup) — they become unused when the registry is present. Framework-level settings (`validation_level`, `minimum_file_size_mb`, `video_extensions`, `match_conflict_behavior`, `unknown_media_behavior`, `replace_illegal_characters`) stay on `import_settings`.

10. **Download categories auto-derived from module PluralName.** The hard-coded `SlipStreamSubdirs` and `detectMediaType` function are replaced with module-aware versions. Each module's download subdirectory is `SlipStream/{PluralName}` (e.g., `SlipStream/Movies`, `SlipStream/Series`). The base `SlipStream/` directory remains as a fallback. `detectMediaType` becomes `detectModuleType` and checks the download path against each registered module's subdirectory.

11. **Root folder path lookup uses `rootfolder.Service.Get(ctx, id)`.** Code blocks in this phase show a conceptual `rootFolderSvc.GetPath(ctx, id)` for brevity. The actual API is `rootfolder.Service.Get(ctx, id) (*RootFolder, error)` returning a struct with a `.Path` string field. All `fileParser`, `importHandler`, and `pathGenerator` structs that need root folder paths should hold a `*rootfolder.Service` reference and call `rf, err := s.rootFolderSvc.Get(ctx, entity.RootFolderID); rootFolderPath := rf.Path`.

12. **ImportHandler.MatchDownload replaces the import service's matching logic.** The current `matchToLibrary` function in `import/matching.go` has interleaved queue-match + parse-match + conflict resolution logic with hard-coded movie/TV branches. In the module path, the module's `ImportHandler.MatchDownload` encapsulates all of this: it receives the `CompletedDownload` (which carries module_type, entity_type, entity_id derived from the download mapping — see `CompletedDownload` comments in Task 6.1) and returns `[]MatchedEntity`. For queue-mapped downloads, the module uses the entity IDs directly. For unmapped downloads (orphan scan), the framework calls `FileParser.TryMatch` to determine the module, then `FileParser.MatchToEntity` to find the library entity. The existing `matching.go` code is NOT removed — it's wrapped by the module implementations.

13. **Multi-entity file support (§18.1) is handled by TV ImportHandler.** The TV module's `ImportHandler.MatchIndividualFile` parses S01E01E02 patterns and returns a MatchedEntity with `EntityIDs` populated for all matched episodes. The framework's file record creation calls `ImportFile` once for multi-entity files — the TV module creates file records for all matched episodes linked to the same file path. The `NamingProvider` provides the `MultiEntityStyle` format option (extend, duplicate, range, etc.) which controls how the filename is rendered.

---

### Phase 6 Dependency Graph

```
Task 6.1 (types)
 ├──→ Task 6.2 (parseutil) ──→ Task 6.3 (Movie FileParser) ──┐
 │                         └──→ Task 6.4 (TV FileParser) ─────┤
 ├──→ Task 6.5 (Movie Path+Naming) ──→ Task 6.7 (migration)  │
 ├──→ Task 6.6 (TV Path+Naming) ─────→ Task 6.7              │
 ├──→ Task 6.8 (Movie ImportHandler) ─────────────────────────┤
 └──→ Task 6.9 (TV ImportHandler) ────────────────────────────┤
                                                               │
 Task 6.3 + 6.4 ──→ Task 6.11 (orphan scan)                  │
 Task 6.3 + 6.4 + 6.7 + 6.8 + 6.9 ──→ Task 6.10 (pipeline)  │
 Task 6.10 + 6.11 ──→ Task 6.12 (Wire DI)                    │
 Task 6.12 ──→ Task 6.13 (validation)
```

**Parallel groups:**
- After 6.1: {6.2, 6.5, 6.6, 6.8, 6.9} — five tasks in parallel
- After 6.2: {6.3, 6.4} — two tasks in parallel
- After 6.3+6.4 and 6.7+6.8+6.9: {6.10, 6.11} — two tasks in parallel

---

### Task 6.1: Flesh Out File Pipeline Interface Types

**Depends on:** Phase 5 complete

**Modify** `internal/module/parse_types.go` — flesh out the `ParseResult` skeleton from Phase 0:

```go
package module

// ParseResult is the result of parsing a filename.
// Common fields are populated by all modules. Module-specific data goes in Extra.
type ParseResult struct {
    Title     string   // Cleaned media title
    Year      int      // Release year (0 if not detected)
    Quality   string   // "2160p", "1080p", "720p", "480p", ""
    Source    string   // "BluRay", "WEBDL", "WEBRip", "HDTV", "Remux", etc.
    Codec     string   // "x265", "x264", "AV1", etc.
    HDRFormats    []string // "DV", "HDR10+", "HDR10", "HDR", "HLG", "SDR"
    AudioCodecs   []string // Normalized: "TrueHD", "DTS-HD MA", "DDP", "DD", "AAC", etc.
    AudioChannels []string // "7.1", "5.1", "2.0"
    ReleaseGroup  string   // e.g., "SPARKS", "NTb"
    Revision      string   // "Proper", "REPACK", "REAL", "RERIP"
    Edition       string   // "Director's Cut", "Extended", etc.
    Languages     []string // Non-English languages detected

    // Extra carries module-specific parsed data.
    // Movie module: nil (all data in common fields).
    // TV module: *TVParseExtra (defined in internal/modules/tv/).
    Extra any
}

```

> **Note:** `MonitoringPreset` was already fleshed out in Phase 4 (Task 4.1) within `internal/module/parse_types.go`. It is NOT re-added or modified in Phase 6 — the earlier `MonitoringPreset` code block that was here has been removed to avoid duplication. The struct already has: `ID`, `Label`, `Description`, `HasOptions`.

**Modify** `internal/module/import_types.go` — flesh out import types:

```go
package module

// CompletedDownload represents a download that has finished and is ready for import.
// Populated by the framework from the download mapping and download client data.
type CompletedDownload struct {
    DownloadID   string
    ClientID     int64
    ClientName   string
    DownloadPath string // Full path to the download directory or file
    Size         int64

    // ModuleType and EntityType are DERIVED by the framework from the legacy
    // download_mappings columns (movie_id, series_id, episode_id, is_season_pack).
    // The download_mappings table does NOT have module_type/entity_type columns.
    // Derivation logic:
    //   movie_id != NULL           → ModuleType="movie", EntityType="movie", EntityID=movie_id
    //   episode_id != NULL         → ModuleType="tv", EntityType="episode", EntityID=episode_id
    //   series_id != NULL + is_season_pack → ModuleType="tv", EntityType="series", EntityID=series_id, IsGroupDownload=true
    ModuleType Type
    EntityType EntityType
    EntityID   int64

    // Group download info (season packs, full albums, etc.)
    IsGroupDownload  bool
    GroupEntityType  EntityType // Parent entity type (e.g., "season", "series")
    GroupEntityID    int64      // Parent entity ID
    SeasonNumber     *int       // TV convenience field (from download_mappings.season_number)

    TargetSlotID *int64
    Source       string // "auto-search", "manual-search", "portal-request"

    MappingID      int64
    ImportAttempts int64
}

// MatchedEntity represents a library entity matched to a download or file.
type MatchedEntity struct {
    ModuleType Type
    EntityType EntityType
    EntityID   int64       // Primary entity
    EntityIDs  []int64     // All matched entities (populated for multi-entity files, e.g., S01E01E02)
    Title      string      // Display title

    RootFolder  string  // Destination root folder path
    Confidence  float64 // 0.0–1.0
    Source      string  // "queue", "parse", "manual"

    IsUpgrade        bool
    ExistingFileID   *int64
    ExistingFilePath string
    CandidateQualityID int
    ExistingQualityID  int
    QualityProfileID   int64

    // TokenData carries template variable values for path/naming resolution.
    // Populated by the module's ImportHandler. Keys are well-known names that map
    // to renamer.TokenContext fields (e.g., "MovieTitle", "SeriesTitle", "SeasonNumber").
    // The framework adds quality/media-info keys during import.
    TokenData map[string]any

    // GroupInfo for multi-file downloads (populated for season packs, albums, etc.)
    GroupInfo *GroupMatchInfo
}

// GroupMatchInfo carries parent entity info for group downloads.
type GroupMatchInfo struct {
    ParentEntityType EntityType
    ParentEntityID   int64
    IsSeasonPack     bool
    IsCompleteSeries bool
}

// QualityInfo describes the quality of a file being imported.
type QualityInfo struct {
    QualityID  int
    Quality    string // "1080p", "720p", etc.
    Source     string // "WEBDL", "BluRay", etc.
    Codec      string
    HDRFormats []string
}

// ImportResult is the result of a module importing a file (creating the file record).
type ImportResult struct {
    FileID          int64  // Created file record ID
    DestinationPath string // Final path in library
    QualityID       int    // Quality ID assigned to the file record
}

// MediaInfoFieldDecl declares a media info field relevant to the module.
type MediaInfoFieldDecl struct {
    Name     string // "video_codec", "audio_codec", "resolution", "audio_channels", "dynamic_range"
    Required bool   // If true, probe failure prevents import completion
}
```

**Modify** `internal/module/path_types.go` — flesh out path/naming types:

```go
package module

// TemplateVariable describes a variable available for path/naming templates.
type TemplateVariable struct {
    Name        string // e.g., "Movie Title", "Series Title", "season"
    Description string // Human-readable description
    Example     string // Example output value
    DataKey     string // Key in TokenData/PathData (e.g., "MovieTitle", "SeasonNumber")
}

// ConditionalSegment declares an optional path segment with a toggle.
type ConditionalSegment struct {
    Name         string // Condition name (e.g., "SeasonFolder")
    Label        string // UI label (e.g., "Use Season Folders")
    Description  string
    DefaultValue bool   // Default for new entities
    DataKey      string // Key in TokenData that carries the per-entity value
}

// TokenContext describes a named scope of template variables for the settings UI.
// Each context corresponds to a naming template (e.g., "movie-file", "episode-file").
type TokenContext struct {
    Name        string             // Context name (e.g., "movie-folder", "episode-file")
    Label       string             // UI label (e.g., "Movie Folder Format", "Episode File Format")
    Variables   []TemplateVariable // Variables available in this context
    Variants    []string           // Format sub-variants (e.g., ["standard", "daily", "anime"])
    IsFolder    bool               // true for folder templates, false for file templates
}

// FormatOption declares a module-specific naming option.
type FormatOption struct {
    Key          string   // Setting key (e.g., "multi_episode_style", "colon_replacement")
    Label        string   // UI label
    Description  string
    Type         string   // "enum" or "bool"
    EnumValues   []string // For enum type: allowed values
    DefaultValue string   // Default value as string
}
```

**Verify:** `go build ./internal/module/...` compiles.

---

### Task 6.2: Extract Shared Parsing Utility Library

**Depends on:** Task 6.1

**Create** `internal/module/parseutil/` package

Extract shared parsing utilities from `internal/library/scanner/parser.go` and `internal/library/scanner/extensions.go` into a reusable package that module FileParser implementations compose.

**Create** `internal/module/parseutil/quality.go`:

Extract from `scanner/parser.go` functions `parseVideoQuality`, `parseHDRAndAttributes`, `parseAudioInfo` into public functions:

```go
package parseutil

// ParseVideoQuality extracts resolution and source from a filename.
func ParseVideoQuality(filename string) (quality, source, codec string)

// ParseHDRFormats extracts HDR format info from a filename.
func ParseHDRFormats(filename string) []string

// ParseAudioInfo extracts audio codecs, channels, and enhancements from a filename.
func ParseAudioInfo(filename string) (codecs, channels, enhancements []string)

// DetectQualityAttributes is a convenience that calls all quality parsers and returns a consolidated result.
func DetectQualityAttributes(filename string) QualityAttributes

// QualityAttributes holds all quality-related parsed info.
type QualityAttributes struct {
    Quality           string
    Source            string
    Codec             string
    HDRFormats        []string
    AudioCodecs       []string
    AudioChannels     []string
    AudioEnhancements []string
}
```

The implementations are direct extractions of the existing regex-based logic from `parser.go` lines 352–476. The regexes and priority ordering must be preserved exactly.

**Create** `internal/module/parseutil/title.go`:

Extract from `parser.go` the `cleanTitle` function and year extraction:

```go
package parseutil

// CleanTitle replaces separator characters (dots, underscores, hyphens) with spaces and trims.
func CleanTitle(title string) string

// ExtractYear attempts to extract a 4-digit year (1900–2100) from a string.
// Returns the year and the remaining string with the year removed.
func ExtractYear(s string) (year int, remainder string, found bool)

// NormalizeTitle lowercases, removes articles, and normalizes whitespace for comparison.
func NormalizeTitle(title string) string
```

`NormalizeTitle` should combine the existing normalization logic from `scanner/parser.go`'s `cleanTitle` and the similarity comparison prep in `import/matching.go`'s `calculateTitleSimilarity` (lowercasing, removing "the"/"a"/"an" prefixes, cleaning punctuation). Having one canonical normalize function prevents divergence.

**Create** `internal/module/parseutil/metadata.go`:

Extract release group, revision, edition, and language parsing:

```go
package parseutil

// ParseReleaseGroup extracts the release group from a filename (e.g., "-SPARKS" → "SPARKS").
func ParseReleaseGroup(filename string) string

// ParseRevision detects PROPER, REPACK, REAL, RERIP in a filename.
func ParseRevision(filename string) string

// ParseEdition detects edition tags (Director's Cut, Extended, etc.) in a filename.
func ParseEdition(filename string) string

// ParseLanguages detects non-English language tags in a filename.
func ParseLanguages(filename string) []string
```

**Create** `internal/module/parseutil/extensions.go`:

Extract from `scanner/extensions.go`:

```go
package parseutil

// IsVideoFile checks whether a filename has a recognized video extension.
func IsVideoFile(filename string) bool

// IsSampleFile checks whether a filename appears to be a sample, trailer, or proof file.
func IsSampleFile(filename string) bool

// VideoExtensions returns the list of recognized video file extensions.
func VideoExtensions() []string
```

**Implementation note for the subagent:** The existing functions in `scanner/parser.go` and `scanner/extensions.go` should NOT be removed. Instead:
1. Extract the core logic into `parseutil` as new public functions.
2. Update the scanner functions to call through to `parseutil` (e.g., `scanner.cleanTitle` becomes a one-line wrapper around `parseutil.CleanTitle`). This avoids breaking library scanning.
3. Update `import/matching.go`'s title normalization to use `parseutil.NormalizeTitle` instead of its local implementation.

**Create** `internal/module/parseutil/parseutil_test.go` — test the extracted functions with cases from the existing `parser_test.go` and `extensions_test.go`.

**Verify:**
- `go build ./internal/module/parseutil/...` compiles
- `go test ./internal/module/parseutil/...` passes
- `go test ./internal/library/scanner/...` still passes (scanner wraps parseutil)
- `go test ./internal/import/...` still passes

---

### Task 6.3: Implement Movie FileParser

**Depends on:** Task 6.2

**Create** `internal/modules/movie/fileparser.go`

Implements `module.FileParser` for the Movie module. Wraps the existing scanner for movie-specific parsing.

```go
package movie

import (
    "context"
    "strings"

    "github.com/rs/zerolog"

    "github.com/slipstream/slipstream/internal/library/movies"
    "github.com/slipstream/slipstream/internal/library/scanner"
    "github.com/slipstream/slipstream/internal/module"
    "github.com/slipstream/slipstream/internal/module/parseutil"
)

type fileParser struct {
    movieSvc *movies.Service
    logger   zerolog.Logger
}

func newFileParser(movieSvc *movies.Service, logger *zerolog.Logger) *fileParser {
    return &fileParser{
        movieSvc: movieSvc,
        logger:   logger.With().Str("component", "movie-fileparser").Logger(),
    }
}
```

**`ParseFilename` — extract movie info from a filename:**

```go
func (p *fileParser) ParseFilename(filename string) (*module.ParseResult, error) {
    parsed := scanner.ParseFilename(filename)

    // Only claim this as a movie parse if NOT detected as TV
    if parsed.IsTV {
        return nil, fmt.Errorf("filename %q detected as TV, not movie", filename)
    }

    return &module.ParseResult{
        Title:         parsed.Title,
        Year:          parsed.Year,
        Quality:       parsed.Quality,
        Source:        parsed.Source,
        Codec:         parsed.Codec,
        HDRFormats:    parsed.HDRFormats,
        AudioCodecs:   parsed.AudioCodecs,
        AudioChannels: parsed.AudioChannels,
        ReleaseGroup:  parsed.ReleaseGroup,
        Revision:      parsed.Revision,
        Edition:       parsed.Edition,
        Languages:     parsed.Languages,
        Extra:         nil, // No module-specific data for movies
    }, nil
}
```

**`TryMatch` — confidence-based orphan detection:**

```go
func (p *fileParser) TryMatch(filename string) (confidence float64, match *module.ParseResult) {
    if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
        return 0, nil
    }

    parsed := scanner.ParseFilename(filename)

    // TV patterns get 0 confidence from the movie parser
    if parsed.IsTV {
        return 0, nil
    }

    // Movie with year: high confidence
    if parsed.Title != "" && parsed.Year > 0 {
        result := parsedMediaToResult(parsed)
        return 0.8, result
    }

    // Movie without year: low confidence (could be anything)
    if parsed.Title != "" {
        result := parsedMediaToResult(parsed)
        return 0.3, result
    }

    return 0, nil
}
```

**`MatchToEntity` — find the library movie for a parse result:**

```go
func (p *fileParser) MatchToEntity(ctx context.Context, parseResult *module.ParseResult) (*module.MatchedEntity, error) {
    // Search library by title + year (see implementation note — method needs to be added)
    movie, err := p.movieSvc.FindByTitleAndYear(ctx, parseResult.Title, parseResult.Year)
    if err != nil {
        return nil, err
    }
    if movie == nil {
        return nil, nil // No match found
    }

    // RootFolder path must be derived from movie.RootFolderID via root folder service
    rootFolderPath := p.rootFolderSvc.GetPath(ctx, movie.RootFolderID)

    return &module.MatchedEntity{
        ModuleType:       module.TypeMovie,
        EntityType:       module.EntityMovie,
        EntityID:         movie.ID,
        Title:            movie.Title,
        RootFolder:       rootFolderPath,
        Confidence:       0.8,
        Source:           "parse",
        QualityProfileID: movie.QualityProfileID,
        TokenData: map[string]any{
            "MovieTitle": movie.Title,
            "MovieYear":  movie.Year,
        },
    }, nil
}
```

**Helper:**

```go
func parsedMediaToResult(parsed *scanner.ParsedMedia) *module.ParseResult {
    return &module.ParseResult{
        Title: parsed.Title, Year: parsed.Year,
        Quality: parsed.Quality, Source: parsed.Source, Codec: parsed.Codec,
        HDRFormats: parsed.HDRFormats, AudioCodecs: parsed.AudioCodecs,
        AudioChannels: parsed.AudioChannels, ReleaseGroup: parsed.ReleaseGroup,
        Revision: parsed.Revision, Edition: parsed.Edition, Languages: parsed.Languages,
    }
}
```

**Implementation note for the subagent:**
- `movies.Service` does NOT have a `FindByTitleAndYear` method. The existing lookup methods are `Get(ctx, id)` (by ID), `GetByTmdbID(ctx, tmdbID)`, and `List(ctx, opts)`. Add a new `FindByTitleAndYear` method that uses `List` with title filter + `parseutil.NormalizeTitle` for comparison, or add a query `FindMovieByTitleYear` to `internal/database/queries/movies.sql` that does a case-insensitive title match with optional year filter, then run `sqlc generate`.
- The `Movie` struct does NOT have `RootFolderPath`. It has `RootFolderID int64` and `Path string` (the movie's own directory path, e.g., `/movies/Inception (2010)/`). The fileParser struct should hold a `*rootfolder.Service` field. Call `rf, _ := s.rootFolderSvc.Get(ctx, movie.RootFolderID)` then use `rf.Path` (see Architecture Decision #11).

**Verify:**
- `go build ./internal/modules/movie/...` compiles
- Compile-time assertion: `var _ module.FileParser = (*fileParser)(nil)` in the file

---

### Task 6.4: Implement TV FileParser

**Depends on:** Task 6.2

**Create** `internal/modules/tv/fileparser.go`

Implements `module.FileParser` for the TV module. Handles S01E02 patterns, season packs, multi-episode detection, daily/anime variants.

```go
package tv

import (
    "context"
    "fmt"

    "github.com/rs/zerolog"

    "github.com/slipstream/slipstream/internal/library/scanner"
    tvlib "github.com/slipstream/slipstream/internal/library/tv"
    "github.com/slipstream/slipstream/internal/module"
    "github.com/slipstream/slipstream/internal/module/parseutil"
)

// TVParseExtra carries TV-specific parsed data in ParseResult.Extra.
type TVParseExtra struct {
    Season           int
    EndSeason        int
    Episode          int
    EndEpisode       int
    IsSeasonPack     bool
    IsCompleteSeries bool
}

type fileParser struct {
    tvSvc  *tvlib.Service
    logger zerolog.Logger
}
```

**`ParseFilename`:**

```go
func (p *fileParser) ParseFilename(filename string) (*module.ParseResult, error) {
    parsed := scanner.ParseFilename(filename)

    if !parsed.IsTV {
        return nil, fmt.Errorf("filename %q not detected as TV", filename)
    }

    return &module.ParseResult{
        Title: parsed.Title, Year: parsed.Year,
        Quality: parsed.Quality, Source: parsed.Source, Codec: parsed.Codec,
        HDRFormats: parsed.HDRFormats, AudioCodecs: parsed.AudioCodecs,
        AudioChannels: parsed.AudioChannels, ReleaseGroup: parsed.ReleaseGroup,
        Revision: parsed.Revision, Edition: parsed.Edition, Languages: parsed.Languages,
        Extra: &TVParseExtra{
            Season:           parsed.Season,
            EndSeason:        parsed.EndSeason,
            Episode:          parsed.Episode,
            EndEpisode:       parsed.EndEpisode,
            IsSeasonPack:     parsed.IsSeasonPack,
            IsCompleteSeries: parsed.IsCompleteSeries,
        },
    }, nil
}
```

**`TryMatch` — confidence-based with TV pattern scoring:**

```go
func (p *fileParser) TryMatch(filename string) (confidence float64, match *module.ParseResult) {
    if !parseutil.IsVideoFile(filename) || parseutil.IsSampleFile(filename) {
        return 0, nil
    }

    parsed := scanner.ParseFilename(filename)

    if !parsed.IsTV {
        return 0, nil
    }

    result := tvParsedMediaToResult(parsed)

    // S##E## pattern: high confidence
    if parsed.Season > 0 && parsed.Episode > 0 {
        return 0.9, result
    }

    // Season pack (S## without episode): good confidence
    if parsed.IsSeasonPack {
        return 0.85, result
    }

    // Title-only TV detection (less certain): low confidence
    return 0.4, result
}
```

**`MatchToEntity` — find series + episode:**

```go
func (p *fileParser) MatchToEntity(ctx context.Context, parseResult *module.ParseResult) (*module.MatchedEntity, error) {
    tvExtra, ok := parseResult.Extra.(*TVParseExtra)
    if !ok || tvExtra == nil {
        return nil, fmt.Errorf("parse result missing TV extra data")
    }

    // Find series by title (+ optional year) — see implementation note, method needs to be added
    series, err := p.tvSvc.FindByTitle(ctx, parseResult.Title, parseResult.Year)
    if err != nil {
        return nil, err
    }
    if series == nil {
        return nil, nil
    }

    // RootFolder path must be derived from series.RootFolderID via root folder service
    rootFolderPath := p.rootFolderSvc.GetPath(ctx, series.RootFolderID)

    // For season packs, return a season-level match
    if tvExtra.IsSeasonPack {
        return &module.MatchedEntity{
            ModuleType:       module.TypeTV,
            EntityType:       module.EntitySeason,
            EntityID:         0, // Season entity ID looked up by ImportHandler
            Title:            fmt.Sprintf("%s - Season %d", series.Title, tvExtra.Season),
            RootFolder:       rootFolderPath,
            Confidence:       0.8,
            Source:           "parse",
            QualityProfileID: series.QualityProfileID,
            TokenData: map[string]any{
                "SeriesTitle":  series.Title,
                "SeriesYear":   series.Year,
                "SeasonNumber": tvExtra.Season,
            },
            GroupInfo: &module.GroupMatchInfo{
                ParentEntityType: module.EntitySeason,
                IsSeasonPack:     true,
                IsCompleteSeries: tvExtra.IsCompleteSeries,
            },
        }, nil
    }

    // Find specific episode
    episode, err := p.tvSvc.GetEpisodeByNumber(ctx, series.ID, tvExtra.Season, tvExtra.Episode)
    if err != nil {
        return nil, err
    }
    if episode == nil {
        return nil, nil
    }

    entity := &module.MatchedEntity{
        ModuleType:       module.TypeTV,
        EntityType:       module.EntityEpisode,
        EntityID:         episode.ID,
        Title:            fmt.Sprintf("%s - S%02dE%02d - %s", series.Title, tvExtra.Season, tvExtra.Episode, episode.Title),
        RootFolder:       rootFolderPath,
        Confidence:       0.8,
        Source:           "parse",
        QualityProfileID: series.QualityProfileID,
        TokenData:        buildTVTokenData(series, episode),
    }

    // Multi-episode: populate EntityIDs for all episodes (e.g., E01E02)
    if tvExtra.EndEpisode > tvExtra.Episode {
        var ids []int64
        for epNum := tvExtra.Episode; epNum <= tvExtra.EndEpisode; epNum++ {
            ep, err := p.tvSvc.GetEpisodeByNumber(ctx, series.ID, tvExtra.Season, epNum)
            if err == nil && ep != nil {
                ids = append(ids, ep.ID)
            }
        }
        entity.EntityIDs = ids
    }

    return entity, nil
}
```

**Implementation note for the subagent:**
- `tv.Service` does NOT have `FindByTitle`. The existing lookup methods are `GetSeries(ctx, id)` (by ID), `GetSeriesByTvdbID(ctx, tvdbID)`, `GetSeriesByPath(ctx, path)`, and `ListSeries(ctx, opts)`. Add a new `FindByTitle` method that uses `ListSeries` with title filter + `parseutil.NormalizeTitle` for comparison, or add a query `FindSeriesByTitle` to `internal/database/queries/tv.sql`.
- `tv.Service` has `GetEpisodeByNumber(ctx, seriesID, seasonNumber, episodeNumber)` — use this for episode lookup (the code above already uses it).
- The `Series` struct does NOT have `RootFolderPath`. It has `RootFolderID int64` and `Path string` (the series directory). The fileParser struct should hold a `*rootfolder.Service` field and call `rf, _ := s.rootFolderSvc.Get(ctx, series.RootFolderID)` then use `rf.Path` (see Architecture Decision #11).
- The `Episode` struct does NOT have `AbsoluteNumber`. It has: `ID`, `SeriesID`, `SeasonNumber`, `EpisodeNumber`, `Title`, `AirDate (*time.Time)`, `Overview`, `Monitored`, `Status`, `StatusMessage`, `ActiveDownloadID`, `EpisodeFile`. AbsoluteNumber is available in TVDB metadata (`metadata/tvdb/models.go`) but is not persisted to the Episode domain struct. Either add it to the Episode struct + migration, or omit from TokenData until anime support is added.
- `episode.AirDate` is `*time.Time` (nullable). Handle nil when populating TokenData.
- The `buildTVTokenData` helper should populate: `SeriesTitle`, `SeriesYear`, `SeriesType` (from `series.FormatType`), `SeasonNumber`, `EpisodeNumber`, `EpisodeTitle`, `AirDate` (dereference `*time.Time`, use zero value if nil), `SeasonFolder` (from `series.SeasonFolder`), `IsSpecial` (seasonNumber == 0). Omit `AbsoluteNumber` unless the Episode struct is extended.

**Verify:**
- `go build ./internal/modules/tv/...` compiles
- Compile-time assertion: `var _ module.FileParser = (*fileParser)(nil)`

---

### Task 6.5: Implement Movie PathGenerator and NamingProvider

**Depends on:** Task 6.1

**Create** `internal/modules/movie/path_naming.go`

Implements both `module.PathGenerator` and `module.NamingProvider` for the Movie module. Movie has the simplest path structure (flat: folder + file).

```go
package movie

import (
    "context"

    "github.com/slipstream/slipstream/internal/module"
)

// --- PathGenerator ---

type pathGenerator struct{}

func (p *pathGenerator) DefaultTemplates() map[string]string {
    return map[string]string{
        "movie-folder": "{Movie Title} ({Year})",
        "movie-file":   "{Movie Title} ({Year}) - {Quality Title}",
    }
}

func (p *pathGenerator) AvailableVariables(contextName string) []module.TemplateVariable {
    common := []module.TemplateVariable{
        {Name: "Movie Title", Description: "Movie title", Example: "Inception", DataKey: "MovieTitle"},
        {Name: "Movie TitleYear", Description: "Title with year", Example: "Inception (2010)", DataKey: "MovieTitleYear"},
        {Name: "Movie CleanTitle", Description: "Title without special characters", Example: "Inception", DataKey: "MovieCleanTitle"},
        {Name: "Year", Description: "Release year", Example: "2010", DataKey: "MovieYear"},
        {Name: "IMDb", Description: "IMDb ID", Example: "tt1375666", DataKey: "ImdbID"},
        {Name: "TMDB", Description: "TMDB ID", Example: "27205", DataKey: "TmdbID"},
    }

    if contextName == "movie-file" {
        // File context adds quality/media info variables
        common = append(common, qualityVariables()...)
        common = append(common, mediaInfoVariables()...)
        common = append(common, metadataVariables()...)
    }

    return common
}

func (p *pathGenerator) ConditionalSegments() []module.ConditionalSegment {
    return nil // No conditional segments for movies
}

func (p *pathGenerator) IsSpecialNode(_ context.Context, _ module.EntityType, _ int64) (bool, error) {
    return false, nil // Movies have no special nodes
}

func (p *pathGenerator) ResolveTemplate(template string, data map[string]any) (string, error) {
    // Delegate to the shared template resolution engine (renamer).
    // This method is primarily for validation/preview, not runtime import.
    // During import, the framework uses per-module renamer instances directly.
    return resolveTemplateGeneric(template, data)
}
```

```go
// --- NamingProvider ---

type namingProvider struct{}

func (n *namingProvider) TokenContexts() []module.TokenContext {
    return []module.TokenContext{
        {
            Name:      "movie-folder",
            Label:     "Movie Folder Format",
            Variables: (&pathGenerator{}).AvailableVariables("movie-folder"),
            IsFolder:  true,
        },
        {
            Name:      "movie-file",
            Label:     "Movie File Format",
            Variables: (&pathGenerator{}).AvailableVariables("movie-file"),
            IsFolder:  false,
        },
    }
}

func (n *namingProvider) DefaultFileTemplates() map[string]string {
    return map[string]string{
        "movie-folder": "{Movie Title} ({Year})",
        "movie-file":   "{Movie Title} ({Year}) - {Quality Title}",
    }
}

func (n *namingProvider) FormatOptions() []module.FormatOption {
    // Spec deviation: §8.6 says "Movie Module: none currently" for format options.
    // However, colon_replacement is a shared renaming concern (the existing import_settings
    // table has a single colon_replacement column used for BOTH movie and TV naming). Storing
    // it per-module in module_naming_settings is more flexible and preserves existing behavior
    // when migrating from import_settings.
    return []module.FormatOption{
        {
            Key:          "colon_replacement",
            Label:        "Colon Replacement",
            Description:  "How to replace colons in filenames",
            Type:         "enum",
            EnumValues:   []string{"delete", "dash", "space_dash", "space_dash_space", "smart", "custom"},
            DefaultValue: "smart",
        },
    }
}
```

**Implementation note for the subagent:**
- Create shared helper functions `qualityVariables()`, `mediaInfoVariables()`, `metadataVariables()` that return the quality/mediainfo/metadata template variables common to all video modules. These can live in a shared file (`internal/modules/shared_variables.go` or within the movie package for now — they'll be extracted in Phase 10). Alternatively, define them inline.
- The `resolveTemplateGeneric` function is a lightweight template resolver for validation/preview. It can call `renamer.ParseTokens` and resolve against a data map. Keep it simple — full path resolution during import uses the renamer directly.
- Compile-time assertions: `var _ module.PathGenerator = (*pathGenerator)(nil)` and `var _ module.NamingProvider = (*namingProvider)(nil)`.

**Verify:** `go build ./internal/modules/movie/...` compiles.

---

### Task 6.6: Implement TV PathGenerator and NamingProvider

**Depends on:** Task 6.1

**Create** `internal/modules/tv/path_naming.go`

Implements `module.PathGenerator` and `module.NamingProvider` for the TV module. TV has the most complex path structure (3 levels, conditional season folders, specials, format sub-variants, multi-episode styles).

```go
package tv

import (
    "context"

    tvlib "github.com/slipstream/slipstream/internal/library/tv"
    "github.com/slipstream/slipstream/internal/module"
)

// --- PathGenerator ---

type pathGenerator struct {
    tvSvc *tvlib.Service
}

func (p *pathGenerator) DefaultTemplates() map[string]string {
    // These defaults must match the import_settings DB defaults (migration 014).
    // The migration seeds: season_folder_format = "Season {season}" (no zero-padding),
    // standard_episode_format = "...{Quality Title}", daily/anime = "...{Quality Full}".
    return map[string]string{
        "series-folder":   "{Series Title}",
        "season-folder":   "Season {season}",
        "specials-folder": "Specials",
        "episode-file.standard": "{Series Title} - S{season:00}E{episode:00} - {Quality Title}",
        "episode-file.daily":    "{Series Title} - {Air-Date} - {Episode Title} {Quality Full}",
        "episode-file.anime":    "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}",
    }
}

func (p *pathGenerator) AvailableVariables(contextName string) []module.TemplateVariable {
    switch contextName {
    case "series-folder":
        return []module.TemplateVariable{
            {Name: "Series Title", Description: "Series title", Example: "Breaking Bad", DataKey: "SeriesTitle"},
            {Name: "Series TitleYear", Description: "Title with year", Example: "Breaking Bad (2008)", DataKey: "SeriesTitleYear"},
            {Name: "Series CleanTitle", Description: "Title without special characters", Example: "Breaking Bad", DataKey: "SeriesCleanTitle"},
        }
    case "season-folder", "specials-folder":
        return []module.TemplateVariable{
            {Name: "season", Description: "Season number", Example: "01", DataKey: "SeasonNumber"},
        }
    default: // episode-file and variants
        vars := []module.TemplateVariable{
            {Name: "Series Title", Description: "Series title", Example: "Breaking Bad", DataKey: "SeriesTitle"},
            {Name: "season", Description: "Season number", Example: "01", DataKey: "SeasonNumber"},
            {Name: "episode", Description: "Episode number", Example: "01", DataKey: "EpisodeNumber"},
            {Name: "Episode Title", Description: "Episode title", Example: "Pilot", DataKey: "EpisodeTitle"},
            {Name: "Air-Date", Description: "Air date (YYYY-MM-DD)", Example: "2008-01-20", DataKey: "AirDate"},
            {Name: "absolute", Description: "Absolute episode number (anime)", Example: "001", DataKey: "AbsoluteNumber"},
        }
        vars = append(vars, qualityVariables()...)
        vars = append(vars, mediaInfoVariables()...)
        vars = append(vars, metadataVariables()...)
        return vars
    }
}

func (p *pathGenerator) ConditionalSegments() []module.ConditionalSegment {
    return []module.ConditionalSegment{
        {
            Name:         "SeasonFolder",
            Label:        "Use Season Folders",
            Description:  "Organize episodes into season subfolders",
            DefaultValue: true,
            DataKey:      "SeasonFolder",
        },
    }
}

func (p *pathGenerator) IsSpecialNode(ctx context.Context, entityType module.EntityType, entityID int64) (bool, error) {
    if entityType != module.EntitySeason {
        return false, nil
    }
    // Season 0 = Specials — see implementation note, GetSeasonByID needs to be added
    season, err := p.tvSvc.GetSeasonByID(ctx, entityID)
    if err != nil {
        return false, err
    }
    return season.SeasonNumber == 0, nil
}
```

```go
// --- NamingProvider ---

type namingProvider struct{}

func (n *namingProvider) TokenContexts() []module.TokenContext {
    pg := &pathGenerator{}
    return []module.TokenContext{
        {
            Name:      "series-folder",
            Label:     "Series Folder Format",
            Variables: pg.AvailableVariables("series-folder"),
            IsFolder:  true,
        },
        {
            Name:      "season-folder",
            Label:     "Season Folder Format",
            Variables: pg.AvailableVariables("season-folder"),
            IsFolder:  true,
        },
        {
            Name:      "specials-folder",
            Label:     "Specials Folder Format",
            Variables: pg.AvailableVariables("specials-folder"),
            IsFolder:  true,
        },
        {
            Name:      "episode-file",
            Label:     "Episode File Format",
            Variables: pg.AvailableVariables("episode-file"),
            Variants:  []string{"standard", "daily", "anime"},
            IsFolder:  false,
        },
    }
}

func (n *namingProvider) DefaultFileTemplates() map[string]string {
    return (&pathGenerator{}).DefaultTemplates()
}

func (n *namingProvider) FormatOptions() []module.FormatOption {
    return []module.FormatOption{
        {
            Key:          "colon_replacement",
            Label:        "Colon Replacement",
            Description:  "How to replace colons in filenames",
            Type:         "enum",
            EnumValues:   []string{"delete", "dash", "space_dash", "space_dash_space", "smart", "custom"},
            DefaultValue: "smart",
        },
        {
            Key:          "multi_episode_style",
            Label:        "Multi-Episode Style",
            Description:  "How multi-episode filenames are formatted",
            Type:         "enum",
            EnumValues:   []string{"extend", "duplicate", "repeat", "scene", "range", "prefixed_range"},
            DefaultValue: "extend",
        },
    }
}
```

**Implementation note for the subagent:**
- `tv.Service` does NOT have a `GetSeasonByID` method. It has `GetSeasonByNumber(ctx, seriesID, seasonNumber)` via sqlc query (returns `*Season`), and `ListSeasons(ctx, seriesID)`. To implement `IsSpecialNode(ctx, EntitySeason, seasonID)`, either: (a) add a `GetSeasonByID(ctx, seasonID)` method with a new sqlc query `SELECT * FROM seasons WHERE id = ?` (preferred), or (b) refactor `IsSpecialNode` to accept `(ctx, entityType, entityID, parentID)` so it can use `GetSeasonByNumber`. The Season struct has `ID`, `SeriesID`, `SeasonNumber`, `Monitored`, `Overview`, `PosterURL`, `SizeOnDisk`, `StatusCounts`.
- Reuse the `qualityVariables()`, `mediaInfoVariables()`, `metadataVariables()` helpers from Task 6.5 (either import from a shared location or duplicate — Phase 10 deduplicates).
- Compile-time assertions for both interfaces.

**Verify:** `go build ./internal/modules/tv/...` compiles.

---

### Task 6.7: Naming Settings Migration and Settings Loading Refactor

**Depends on:** Tasks 6.5 + 6.6 (need to know per-module setting keys)

#### 6.7a: Create Migration

**Create** `internal/database/migrations/0XX_module_naming_settings.sql` (use next available migration number)

```sql
-- +goose Up
-- +goose StatementBegin

CREATE TABLE module_naming_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    module_type TEXT NOT NULL,
    setting_key TEXT NOT NULL,
    setting_value TEXT NOT NULL DEFAULT '',
    UNIQUE(module_type, setting_key)
);

-- Migrate movie naming settings from import_settings
INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'movie', 'rename_enabled', CASE WHEN rename_movies = 1 THEN 'true' ELSE 'false' END
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'movie', 'colon_replacement', colon_replacement
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'movie', 'custom_colon_replacement', COALESCE(custom_colon_replacement, '')
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'movie', 'movie-folder', movie_folder_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'movie', 'movie-file', movie_file_format
FROM import_settings WHERE id = 1;

-- Migrate TV naming settings from import_settings
INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'rename_enabled', CASE WHEN rename_episodes = 1 THEN 'true' ELSE 'false' END
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'colon_replacement', colon_replacement
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'custom_colon_replacement', COALESCE(custom_colon_replacement, '')
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'multi_episode_style', multi_episode_style
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'series-folder', series_folder_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'season-folder', season_folder_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'specials-folder', specials_folder_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'episode-file.standard', standard_episode_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'episode-file.daily', daily_episode_format
FROM import_settings WHERE id = 1;

INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
SELECT 'tv', 'episode-file.anime', anime_episode_format
FROM import_settings WHERE id = 1;

-- NOTE: import_settings naming columns are NOT dropped. They become unused when
-- the module registry is present. Phase 10 cleanup removes them.

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS module_naming_settings;
```

#### 6.7b: Add sqlc Queries

**Create** `internal/database/queries/module_naming_settings.sql`:

```sql
-- name: ListModuleNamingSettings :many
SELECT setting_key, setting_value
FROM module_naming_settings
WHERE module_type = ?;

-- name: GetModuleNamingSetting :one
SELECT setting_value
FROM module_naming_settings
WHERE module_type = ? AND setting_key = ?;

-- name: UpsertModuleNamingSetting :exec
INSERT INTO module_naming_settings (module_type, setting_key, setting_value)
VALUES (?, ?, ?)
ON CONFLICT(module_type, setting_key) DO UPDATE SET setting_value = excluded.setting_value;

-- name: DeleteModuleNamingSettings :exec
DELETE FROM module_naming_settings WHERE module_type = ?;
```

Run: `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate`

#### 6.7c: Add Renamer Generic Context Resolution

**Modify** `internal/import/renamer/resolver.go`:

Add `Patterns` field to `Settings` and `ResolveContext` method to `Resolver`:

```go
// Add to Settings struct:
type Settings struct {
    // ... existing fields unchanged ...

    // Patterns maps context names to format patterns for generic module resolution.
    // Keys follow "<level>-folder" / "<level>-file" convention with optional ".variant" suffix.
    // Example: {"movie-folder": "{Movie Title} ({Year})", "episode-file.daily": "{Series Title} - {Air-Date}"}
    Patterns map[string]string
}
```

```go
// ResolveContext resolves a named template context using Patterns map.
// Used by the module-aware import pipeline. Falls back to an error if the
// context name is not found in Patterns.
func (r *Resolver) ResolveContext(contextName string, ctx *TokenContext, ext string) (string, error) {
    pattern, ok := r.settings.Patterns[contextName]
    if !ok {
        return "", fmt.Errorf("no pattern for context %q", contextName)
    }

    // Handle multi-episode if applicable
    if len(ctx.EpisodeNumbers) > 1 && r.settings.MultiEpisodeStyle != "" {
        return r.resolveMultiEpisodeContext(pattern, ctx, ext)
    }

    // Use the existing resolvePattern (tokens → resolve → cleanup → trim)
    resolved, err := r.resolvePattern(pattern, ctx)
    if err != nil {
        return "", err
    }
    // Sanitize (colon replacement + illegal chars) then apply case transformation
    // — consistent with ResolveMovieFilename/ResolveEpisodeFilename pipeline
    resolved = SanitizeFilename(
        resolved,
        r.settings.ReplaceIllegalCharacters,
        r.settings.ColonReplacement,
        r.settings.CustomColonReplacement,
    )
    resolved = ApplyCase(resolved, r.settings.CaseMode)
    if ext != "" {
        if !strings.HasPrefix(ext, ".") {
            ext = "." + ext
        }
        resolved += ext
    }
    return resolved, nil
}
```

**Implementation note for the subagent:**
- The existing `resolvePattern(pattern string, ctx *TokenContext) (string, error)` method handles the core pipeline: `ParseTokens` → `token.Resolve` → `replaceTokenWithCleanup` → `cleanupOrphanedSeparators` → `TrimSpace`. Do NOT add new parameters to it — `ResolveContext` calls it and appends the extension separately.
- `resolveMultiEpisodeContext` follows the same logic as the existing `applyMultiEpisodeFormat` but uses the pattern from the Patterns map instead of a typed Settings field.
- Ensure existing methods (`ResolveMovieFilename`, `ResolveEpisodeFilename`, etc.) continue to work unchanged — they still read from their typed Settings fields.

#### 6.7d: Add Module Naming Settings Loader

**Create** `internal/import/module_naming.go`:

```go
package importer

import (
    "context"

    "github.com/slipstream/slipstream/internal/database/sqlc"
    "github.com/slipstream/slipstream/internal/import/renamer"
    "github.com/slipstream/slipstream/internal/module"
)

// loadModuleRenamer creates a renamer.Resolver configured with the given module's naming settings.
func loadModuleRenamer(ctx context.Context, queries *sqlc.Queries, mod module.Module) (*renamer.Resolver, error) {
    rows, err := queries.ListModuleNamingSettings(ctx, string(mod.ID()))
    if err != nil {
        return nil, err
    }

    // Build settings map from DB rows
    dbSettings := make(map[string]string)
    for _, row := range rows {
        dbSettings[row.SettingKey] = row.SettingValue
    }

    // Get module defaults for template patterns
    defaults := mod.(module.NamingProvider).DefaultFileTemplates()

    // Build renamer Settings
    // NOTE: ReplaceIllegalCharacters and CaseMode are framework-level settings from
    // import_settings (not per-module). The caller should read these from import_settings
    // and set them after this function returns, or pass them in as a parameter.
    settings := &renamer.Settings{
        RenameEpisodes: true, // Overridden below
        RenameMovies:   true, // Overridden below
        Patterns:       make(map[string]string),
    }

    // Populate Patterns: DB overrides, falling back to module defaults
    for key, defaultVal := range defaults {
        if dbVal, ok := dbSettings[key]; ok {
            settings.Patterns[key] = dbVal
        } else {
            settings.Patterns[key] = defaultVal
        }
    }

    // Populate format options
    if v, ok := dbSettings["rename_enabled"]; ok {
        enabled := v == "true"
        settings.RenameEpisodes = enabled
        settings.RenameMovies = enabled
    }
    if v, ok := dbSettings["colon_replacement"]; ok {
        settings.ColonReplacement = renamer.ColonReplacement(v)
    }
    if v, ok := dbSettings["custom_colon_replacement"]; ok {
        settings.CustomColonReplacement = v
    }
    if v, ok := dbSettings["multi_episode_style"]; ok {
        settings.MultiEpisodeStyle = renamer.MultiEpisodeStyle(v)
    }

    // Also populate legacy typed fields from Patterns for backward compat
    populateLegacyFields(settings, mod.ID())

    return renamer.NewResolver(settings), nil
}

// populateLegacyFields copies Patterns values to the typed Settings fields
// so that existing Resolve* methods (called by legacy code paths) still work.
func populateLegacyFields(s *renamer.Settings, moduleType module.Type) {
    switch moduleType {
    case module.TypeMovie:
        s.MovieFolderFormat = s.Patterns["movie-folder"]
        s.MovieFileFormat = s.Patterns["movie-file"]
    case module.TypeTV:
        s.SeriesFolderFormat = s.Patterns["series-folder"]
        s.SeasonFolderFormat = s.Patterns["season-folder"]
        s.SpecialsFolderFormat = s.Patterns["specials-folder"]
        s.StandardEpisodeFormat = s.Patterns["episode-file.standard"]
        s.DailyEpisodeFormat = s.Patterns["episode-file.daily"]
        s.AnimeEpisodeFormat = s.Patterns["episode-file.anime"]
    }
}
```

**Implementation note for the subagent:**
- The `populateLegacyFields` switch on module type is acceptable here — it's a bridge between the new per-module settings and the existing renamer's typed fields. It will be removed in Phase 10 when the renamer is fully generic.
- Add a `ModuleResolvers() map[module.Type]*renamer.Resolver` cache to the import service (populated lazily or at startup). The `getModuleResolver(moduleType)` method returns the cached resolver.

**Verify:**
- `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate` succeeds
- `go build ./internal/import/...` compiles
- `go build ./internal/import/renamer/...` compiles
- `make test` — existing tests pass (new migration runs, old naming behavior unchanged)

---

### Task 6.8: Implement Movie ImportHandler

**Depends on:** Task 6.1

**Create** `internal/modules/movie/importhandler.go`

Implements `module.ImportHandler` for the Movie module. Wraps existing movie import logic.

```go
package movie

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/rs/zerolog"

    "github.com/slipstream/slipstream/internal/database/sqlc"
    "github.com/slipstream/slipstream/internal/library/movies"
    "github.com/slipstream/slipstream/internal/module"
)

type importHandler struct {
    movieSvc *movies.Service
    queries  *sqlc.Queries
    logger   zerolog.Logger
}

func newImportHandler(movieSvc *movies.Service, queries *sqlc.Queries, logger *zerolog.Logger) *importHandler {
    return &importHandler{
        movieSvc: movieSvc,
        queries:  queries,
        logger:   logger.With().Str("component", "movie-import").Logger(),
    }
}
```

**`MatchDownload` — match a completed download to a movie:**

```go
func (h *importHandler) MatchDownload(ctx context.Context, download module.CompletedDownload) ([]module.MatchedEntity, error) {
    // The download mapping already tells us which movie
    movie, err := h.movieSvc.Get(ctx, download.EntityID)
    if err != nil {
        return nil, fmt.Errorf("movie %d not found: %w", download.EntityID, err)
    }

    // RootFolder path derived from movie.RootFolderID via root folder service
    rootFolderPath := h.rootFolderSvc.GetPath(ctx, movie.RootFolderID)

    entity := module.MatchedEntity{
        ModuleType:       module.TypeMovie,
        EntityType:       module.EntityMovie,
        EntityID:         movie.ID,
        Title:            movie.Title,
        RootFolder:       rootFolderPath,
        Confidence:       1.0,
        Source:           "queue",
        QualityProfileID: movie.QualityProfileID,
        TokenData: map[string]any{
            "MovieTitle": movie.Title,
            "MovieYear":  movie.Year,
            "ImdbID":     movie.ImdbID,
            "TmdbID":     movie.TmdbID,
        },
    }

    if download.TargetSlotID != nil {
        // Carry slot target through
        entity.TokenData["TargetSlotID"] = *download.TargetSlotID
    }

    return []module.MatchedEntity{entity}, nil
}
```

**`ImportFile` — create the movie file record:**

```go
func (h *importHandler) ImportFile(ctx context.Context, filePath string, entity module.MatchedEntity, qi module.QualityInfo) (*module.ImportResult, error) {
    // Delegate to existing movie service's file creation
    // AddFile signature: AddFile(ctx, movieID int64, input *CreateMovieFileInput) (*MovieFile, error)
    qid := int64(qi.QualityID)
    originalPath, _ := entity.TokenData["OriginalPath"].(string)
    movieFile, err := h.movieSvc.AddFile(ctx, entity.EntityID, &movies.CreateMovieFileInput{
        Path:         filePath,
        QualityID:    &qid,
        Quality:      qi.Quality,
        OriginalPath: originalPath,
    })
    if err != nil {
        return nil, fmt.Errorf("create movie file record: %w", err)
    }

    return &module.ImportResult{
        FileID:          movieFile.ID,
        DestinationPath: filePath,
        QualityID:       qi.QualityID,
    }, nil
}
```

**Simple methods:**

```go
func (h *importHandler) SupportsMultiFileDownload() bool { return false }

func (h *importHandler) MatchIndividualFile(_ context.Context, _ string, _ module.MatchedEntity) (*module.MatchedEntity, error) {
    return nil, fmt.Errorf("movie module does not support multi-file downloads")
}

func (h *importHandler) IsGroupImportReady(_ context.Context, _ module.MatchedEntity, _ []module.MatchedEntity) bool {
    return false
}

func (h *importHandler) MediaInfoFields() []module.MediaInfoFieldDecl {
    return []module.MediaInfoFieldDecl{
        {Name: "video_codec", Required: false},
        {Name: "audio_codec", Required: false},
        {Name: "audio_channels", Required: false},
        {Name: "resolution", Required: false},
        {Name: "dynamic_range", Required: false},
    }
}
```

**Implementation note for the subagent:**
- `movies.Service` has `Get(ctx, id)` (not `GetByID`), and `AddFile(ctx, movieID int64, input *CreateMovieFileInput) (*MovieFile, error)` (not `CreateFileRecord`). The `CreateMovieFileInput` struct has: `Path`, `Size`, `Quality`, `QualityID (*int64)`, `VideoCodec`, `AudioCodec`, `AudioChannels`, `DynamicRange`, `Resolution`, `OriginalPath`, `OriginalFilename`. Note `QualityID` is `*int64`, not `int`.
- The `Movie` struct does NOT have `RootFolderPath`. It has `RootFolderID int64` and `Path string`. The importHandler struct should hold a `*rootfolder.Service` field and call `rf, _ := h.rootFolderSvc.Get(ctx, movie.RootFolderID)` then use `rf.Path` (see Architecture Decision #11).
- `entity.TokenData["OriginalPath"]` is set by the framework during import (Task 6.10 sets it).
- Compile-time assertion: `var _ module.ImportHandler = (*importHandler)(nil)`

**Verify:** `go build ./internal/modules/movie/...` compiles.

---

### Task 6.9: Implement TV ImportHandler (Multi-File Support)

**Depends on:** Task 6.1

**Create** `internal/modules/tv/importhandler.go`

Implements `module.ImportHandler` for the TV module with full multi-file download support (season packs, multi-episode files).

```go
package tv

import (
    "context"
    "fmt"
    "path/filepath"

    "github.com/rs/zerolog"

    "github.com/slipstream/slipstream/internal/database/sqlc"
    "github.com/slipstream/slipstream/internal/library/scanner"
    tvlib "github.com/slipstream/slipstream/internal/library/tv"
    "github.com/slipstream/slipstream/internal/module"
)

type importHandler struct {
    tvSvc   *tvlib.Service
    queries *sqlc.Queries
    logger  zerolog.Logger
}
```

**`MatchDownload` — handles single episodes AND season packs:**

```go
func (h *importHandler) MatchDownload(ctx context.Context, download module.CompletedDownload) ([]module.MatchedEntity, error) {
    if download.IsGroupDownload {
        // Season pack: return a parent-level match. Individual files matched via MatchIndividualFile.
        series, err := h.tvSvc.GetSeries(ctx, download.EntityID)
        if err != nil {
            return nil, fmt.Errorf("series %d not found: %w", download.EntityID, err)
        }

        // RootFolder path derived from series.RootFolderID via root folder service
        rootFolderPath := h.rootFolderSvc.GetPath(ctx, series.RootFolderID)

        return []module.MatchedEntity{{
            ModuleType:       module.TypeTV,
            EntityType:       module.EntitySeries,
            EntityID:         series.ID,
            Title:            series.Title,
            RootFolder:       rootFolderPath,
            Confidence:       1.0,
            Source:           "queue",
            QualityProfileID: series.QualityProfileID,
            TokenData:        buildSeriesTokenData(series),
            GroupInfo: &module.GroupMatchInfo{
                ParentEntityType: download.GroupEntityType,
                ParentEntityID:   download.GroupEntityID,
                IsSeasonPack:     true,
            },
        }}, nil
    }

    // Single episode match
    episode, err := h.tvSvc.GetEpisode(ctx, download.EntityID)
    if err != nil {
        return nil, fmt.Errorf("episode %d not found: %w", download.EntityID, err)
    }

    series, err := h.tvSvc.GetSeries(ctx, episode.SeriesID)
    if err != nil {
        return nil, err
    }

    // RootFolder path derived from series.RootFolderID via root folder service
    rootFolderPath := h.rootFolderSvc.GetPath(ctx, series.RootFolderID)

    return []module.MatchedEntity{{
        ModuleType:       module.TypeTV,
        EntityType:       module.EntityEpisode,
        EntityID:         episode.ID,
        Title:            fmt.Sprintf("%s - S%02dE%02d", series.Title, episode.SeasonNumber, episode.EpisodeNumber),
        RootFolder:       rootFolderPath,
        Confidence:       1.0,
        Source:           "queue",
        QualityProfileID: series.QualityProfileID,
        TokenData:        buildTVTokenData(series, episode),
    }}, nil
}
```

**`MatchIndividualFile` — match a single file within a season pack:**

```go
func (h *importHandler) MatchIndividualFile(ctx context.Context, filePath string, parentEntity module.MatchedEntity) (*module.MatchedEntity, error) {
    filename := filepath.Base(filePath)
    parsed := scanner.ParseFilename(filename)

    if !parsed.IsTV || parsed.Episode == 0 {
        return nil, fmt.Errorf("cannot parse TV episode from %q", filename)
    }

    seasonNum := parsed.Season
    if seasonNum == 0 && parentEntity.GroupInfo != nil {
        // Inherit season from parent match if not in filename
        if sn, ok := parentEntity.TokenData["SeasonNumber"].(int); ok {
            seasonNum = sn
        }
    }

    series, err := h.tvSvc.GetSeries(ctx, parentEntity.EntityID)
    if err != nil {
        return nil, err
    }

    episode, err := h.tvSvc.GetEpisodeByNumber(ctx, series.ID, seasonNum, parsed.Episode)
    if err != nil || episode == nil {
        return nil, fmt.Errorf("episode S%02dE%02d not found for series %d", seasonNum, parsed.Episode, series.ID)
    }

    entity := &module.MatchedEntity{
        ModuleType:       module.TypeTV,
        EntityType:       module.EntityEpisode,
        EntityID:         episode.ID,
        Title:            fmt.Sprintf("%s - S%02dE%02d", series.Title, seasonNum, parsed.Episode),
        RootFolder:       h.rootFolderSvc.GetPath(ctx, series.RootFolderID),
        Confidence:       0.9,
        Source:           "parse",
        QualityProfileID: series.QualityProfileID,
        TokenData:        buildTVTokenData(series, episode),
    }

    // Multi-episode detection (S01E01E02)
    if parsed.EndEpisode > parsed.Episode {
        var ids []int64
        for epNum := parsed.Episode; epNum <= parsed.EndEpisode; epNum++ {
            ep, err := h.tvSvc.GetEpisodeByNumber(ctx, series.ID, seasonNum, epNum)
            if err == nil && ep != nil {
                ids = append(ids, ep.ID)
            }
        }
        entity.EntityIDs = ids
    }

    return entity, nil
}
```

**`IsGroupImportReady` — check if all season pack files are matched:**

```go
func (h *importHandler) IsGroupImportReady(ctx context.Context, parentEntity module.MatchedEntity, matchedFiles []module.MatchedEntity) bool {
    // For season packs: ready when at least one file is matched.
    // Individual file imports proceed independently.
    return len(matchedFiles) > 0
}
```

**`ImportFile` — create episode file record(s):**

```go
func (h *importHandler) ImportFile(ctx context.Context, filePath string, entity module.MatchedEntity, qi module.QualityInfo) (*module.ImportResult, error) {
    // Create file record for primary episode
    // AddEpisodeFile signature: AddEpisodeFile(ctx, episodeID int64, input *CreateEpisodeFileInput) (*EpisodeFile, error)
    qid := int64(qi.QualityID)
    originalPath, _ := entity.TokenData["OriginalPath"].(string)
    epFile, err := h.tvSvc.AddEpisodeFile(ctx, entity.EntityID, &tvlib.CreateEpisodeFileInput{
        Path:         filePath,
        QualityID:    &qid,
        Quality:      qi.Quality,
        OriginalPath: originalPath,
    })
    if err != nil {
        return nil, fmt.Errorf("create episode file record: %w", err)
    }

    // For multi-entity files (S01E01E02): create file records for additional episodes
    if len(entity.EntityIDs) > 1 {
        for _, epID := range entity.EntityIDs[1:] { // Skip first (already created above)
            _, err := h.tvSvc.AddEpisodeFile(ctx, epID, &tvlib.CreateEpisodeFileInput{
                Path:      filePath, // Same file path for all episodes
                QualityID: &qid,
                Quality:   qi.Quality,
            })
            if err != nil {
                h.logger.Warn().Err(err).Int64("episodeID", epID).Msg("Failed to create multi-episode file record")
            }
        }
    }

    return &module.ImportResult{
        FileID:          epFile.ID,
        DestinationPath: filePath,
        QualityID:       qi.QualityID,
    }, nil
}
```

**Other methods:**

```go
func (h *importHandler) SupportsMultiFileDownload() bool { return true }

func (h *importHandler) MediaInfoFields() []module.MediaInfoFieldDecl {
    return []module.MediaInfoFieldDecl{
        {Name: "video_codec", Required: false},
        {Name: "audio_codec", Required: false},
        {Name: "audio_channels", Required: false},
        {Name: "resolution", Required: false},
        {Name: "dynamic_range", Required: false},
    }
}
```

**Helper:**

```go
func buildSeriesTokenData(series *tvlib.Series) map[string]any {
    return map[string]any{
        "SeriesTitle": series.Title,
        "SeriesYear":  series.Year,
        "SeriesType":  series.FormatType, // "standard", "daily", "anime"
    }
}

func buildTVTokenData(series *tvlib.Series, episode *tvlib.Episode) map[string]any {
    data := map[string]any{
        "SeriesTitle":   series.Title,
        "SeriesYear":    series.Year,
        "SeriesType":    series.FormatType,
        "SeasonNumber":  episode.SeasonNumber,
        "EpisodeNumber": episode.EpisodeNumber,
        "EpisodeTitle":  episode.Title,
        "SeasonFolder":  series.SeasonFolder, // bool — conditional path segment
        "IsSpecial":     episode.SeasonNumber == 0,
    }
    // AirDate is *time.Time (nullable)
    if episode.AirDate != nil {
        data["AirDate"] = *episode.AirDate
    }
    // AbsoluteNumber omitted — not on Episode struct. Add if Episode is extended.
    return data
}
```

**Implementation note for the subagent:**
- `tv.Service` method names: `GetSeries(ctx, id)` (not `GetByID`), `GetEpisode(ctx, id)` (not `GetEpisodeByID`), `GetEpisodeByNumber(ctx, seriesID, seasonNum, epNum)` (not `FindEpisode`), `AddEpisodeFile(ctx, episodeID, *CreateEpisodeFileInput) (*EpisodeFile, error)` (not `CreateEpisodeFileRecord`). The code above uses the correct names.
- `series.FormatType` (string) stores "standard"/"daily"/"anime" ✓
- `series.SeasonFolder` (bool) controls per-series season folder organization ✓
- The `Series` struct does NOT have `RootFolderPath`. It has `RootFolderID int64` and `Path string`. The importHandler struct should hold a `*rootfolder.Service` field and call `rf, _ := h.rootFolderSvc.Get(ctx, series.RootFolderID)` then use `rf.Path` (see Architecture Decision #11).
- `CreateEpisodeFileInput` struct fields: `Path`, `Size`, `Quality`, `QualityID (*int64)`, `VideoCodec`, `AudioCodec`, `AudioChannels`, `DynamicRange`, `Resolution`, `OriginalPath`, `OriginalFilename`.
- `episode.AirDate` is `*time.Time` (nullable). Handle nil when populating TokenData.
- Compile-time assertion: `var _ module.ImportHandler = (*importHandler)(nil)`

**Verify:** `go build ./internal/modules/tv/...` compiles.

---

### Task 6.10: Refactor Download Categories and Import Pipeline for Module Dispatch

**Depends on:** Tasks 6.3, 6.4, 6.7, 6.8, 6.9

This is the **largest and most critical task** in Phase 6. It modifies the import pipeline to dispatch through module interfaces when the registry is present.

#### 6.10a: Module-Aware Download Categories

**Modify** `internal/import/pipeline.go` — replace hard-coded `SlipStreamSubdirs` (a package-level `var` at line ~105, used by `scanClientDownloads`):

```go
// GetSlipStreamSubdirs returns download subdirectories for all registered modules.
// Falls back to hard-coded values when registry is nil.
func (s *Service) GetSlipStreamSubdirs() []string {
    if s.registry == nil {
        return []string{"SlipStream/Movies", "SlipStream/Series", "SlipStream"}
    }

    var dirs []string
    for _, mod := range s.registry.All() {
        dirs = append(dirs, "SlipStream/"+mod.PluralName())
    }
    dirs = append(dirs, "SlipStream") // Base fallback
    return dirs
}
```

**Modify** `internal/downloader/queue.go` — replace `detectMediaType`:

Add a `registry` field to the download queue service (or accept it as a parameter). When set:

```go
func detectModuleType(path string, registry *module.Registry) string {
    if registry == nil {
        return detectMediaTypeLegacy(path)
    }

    pathLower := strings.ToLower(path)
    for _, mod := range registry.All() {
        subdir := strings.ToLower("slipstream/" + mod.PluralName())
        if strings.Contains(pathLower, subdir) || strings.Contains(pathLower, strings.ReplaceAll(subdir, "/", "\\")) {
            return string(mod.ID())
        }
    }
    return "unknown"
}

// detectMediaTypeLegacy is the original hard-coded detection (renamed from detectMediaType).
// NOTE: returns mediaTypeSeries ("series"), not "tv", for backward compatibility.
// The existing constants are: mediaTypeMovie = "movie", mediaTypeSeries = "series" (in queue.go).
func detectMediaTypeLegacy(path string) string {
    pathLower := strings.ToLower(path)
    if strings.Contains(pathLower, "slipstream/movies") || strings.Contains(pathLower, "slipstream\\movies") {
        return mediaTypeMovie
    }
    if strings.Contains(pathLower, "slipstream/series") || strings.Contains(pathLower, "slipstream\\series") {
        return mediaTypeSeries
    }
    return "unknown"
}
```

**Important:** The module-aware path returns `string(mod.ID())` = `"tv"`, while the legacy path returns `"series"`. Callers that switch on the returned value (e.g., in `QueueItem.MediaType` comparisons) must handle both. When the registry is present, use `"tv"` consistently; when nil, the existing `"series"` convention is preserved. The download_mappings table's `module_type` column uses `"tv"` (module ID), so downstream mapping code should already expect it.
```

#### 6.10b: Import Service Registry Integration

**Modify** `internal/import/service.go`:

Add registry field and module resolver cache:

```go
type Service struct {
    // ... existing fields ...
    registry        *module.Registry
    moduleResolvers map[module.Type]*renamer.Resolver
}

// SetRegistry is called by the DI setter to inject the module registry.
func (s *Service) SetRegistry(r *module.Registry) {
    s.registry = r
    // Pre-load per-module renamers
    if r != nil {
        s.moduleResolvers = make(map[module.Type]*renamer.Resolver)
        for _, mod := range r.All() {
            resolver, err := loadModuleRenamer(context.Background(), s.queries, mod)
            if err != nil {
                s.logger.Error().Err(err).Str("module", string(mod.ID())).Msg("Failed to load module renamer")
                continue
            }
            s.moduleResolvers[mod.ID()] = resolver
        }
    }
}
```

#### 6.10c: Module-Aware Token Context Builder

**Create** `internal/import/token_context.go`:

```go
package importer

import (
    "time"

    "github.com/slipstream/slipstream/internal/import/renamer"
    "github.com/slipstream/slipstream/internal/mediainfo"
    "github.com/slipstream/slipstream/internal/module"
)

// buildTokenContextFromData converts a MatchedEntity's TokenData + parsed quality + media info
// into a renamer.TokenContext. This is the centralized bridge between module-provided data
// and the framework's renamer. No type switch on module type needed — it maps well-known keys.
//
// Note: uses *module.ParseResult (from FileParser) rather than *scanner.ParsedMedia since this
// is the module-aware code path. ParseResult has the same quality/metadata fields.
func buildTokenContextFromData(entity *module.MatchedEntity, parsed *module.ParseResult, mi *mediainfo.MediaInfo) *renamer.TokenContext {
    d := entity.TokenData
    ctx := &renamer.TokenContext{}

    // Module-provided fields (from TokenData)
    if v, ok := d["MovieTitle"].(string); ok { ctx.MovieTitle = v }
    if v, ok := d["MovieYear"].(int); ok { ctx.MovieYear = v }
    if v, ok := d["SeriesTitle"].(string); ok { ctx.SeriesTitle = v }
    if v, ok := d["SeriesYear"].(int); ok { ctx.SeriesYear = v }
    if v, ok := d["SeriesType"].(string); ok { ctx.SeriesType = v }
    if v, ok := d["SeasonNumber"].(int); ok { ctx.SeasonNumber = v }
    if v, ok := d["EpisodeNumber"].(int); ok { ctx.EpisodeNumber = v }
    if v, ok := d["EpisodeTitle"].(string); ok { ctx.EpisodeTitle = v }
    if v, ok := d["AirDate"].(time.Time); ok { ctx.AirDate = v } // Value is time.Time after dereferencing *time.Time in buildTVTokenData
    if v, ok := d["AbsoluteNumber"].(int); ok { ctx.AbsoluteNumber = v }
    if v, ok := d["ReleaseVersion"].(int); ok { ctx.ReleaseVersion = v }
    if v, ok := d["ImdbID"].(string); ok { /* available for template use */ }

    // Multi-episode numbers
    if len(entity.EntityIDs) > 1 {
        // Build episode number list from entity IDs (or from TokenData if pre-populated)
        if v, ok := d["EpisodeNumbers"].([]int); ok {
            ctx.EpisodeNumbers = v
        }
    }

    // Quality fields from parsed filename (module.ParseResult)
    if parsed != nil {
        ctx.Quality = parsed.Quality
        ctx.Source = parsed.Source
        ctx.Codec = parsed.Codec
        ctx.Revision = parsed.Revision
        ctx.ReleaseGroup = parsed.ReleaseGroup
        ctx.EditionTags = parsed.Edition
        // OriginalFile set separately from TokenData or framework context, not from ParseResult
    }

    // Media info fields (from probe)
    if mi != nil {
        ctx.VideoCodec = mi.VideoCodec
        ctx.VideoBitDepth = mi.VideoBitDepth
        ctx.VideoDynamicRange = mi.DynamicRange
        ctx.AudioCodec = mi.AudioCodec
        ctx.AudioChannels = mi.AudioChannels
        if len(mi.AudioLanguages) > 0 {
            ctx.AudioLanguages = mi.AudioLanguages
        }
        if len(mi.SubtitleLanguages) > 0 {
            ctx.SubtitleLanguages = mi.SubtitleLanguages
        }
    }

    return ctx
}
```

#### 6.10d: Module-Aware Destination Path Computation

**Modify** `internal/import/pipeline.go`:

Add a `computeDestinationViaModule` method that builds paths by walking the node schema:

```go
// computeDestinationViaModule computes the destination path for a file using the module's
// node schema, naming settings, and template resolution. This replaces the hard-coded
// movie/TV branches in computeDestination when the registry is present.
func (s *Service) computeDestinationViaModule(
    ctx context.Context,
    entity *module.MatchedEntity,
    parsed *module.ParseResult,
    mi *mediainfo.MediaInfo,
    ext string,
) (string, error) {
    mod := s.registry.Get(entity.ModuleType)
    if mod == nil {
        return "", fmt.Errorf("module %s not found in registry", entity.ModuleType)
    }

    resolver, ok := s.moduleResolvers[entity.ModuleType]
    if !ok {
        return "", fmt.Errorf("no renamer configured for module %s", entity.ModuleType)
    }

    tokenCtx := buildTokenContextFromData(entity, parsed, mi)
    schema := mod.NodeSchema()
    segments := []string{entity.RootFolder}

    for _, level := range schema.Levels {
        if level.IsRoot && level.IsLeaf {
            // Flat module (e.g., movie): folder + filename
            folderName, err := resolver.ResolveContext(level.Name+"-folder", tokenCtx, "")
            if err != nil {
                return "", fmt.Errorf("resolve %s-folder: %w", level.Name, err)
            }
            segments = append(segments, folderName)

            fileName, err := resolver.ResolveContext(level.Name+"-file", tokenCtx, ext)
            if err != nil {
                return "", fmt.Errorf("resolve %s-file: %w", level.Name, err)
            }
            segments = append(segments, fileName)
            break
        }

        if level.IsRoot {
            // Root folder (e.g., series folder)
            folderName, err := resolver.ResolveContext(level.Name+"-folder", tokenCtx, "")
            if err != nil {
                return "", fmt.Errorf("resolve %s-folder: %w", level.Name, err)
            }
            segments = append(segments, folderName)
            continue
        }

        if level.IsLeaf {
            // Leaf filename — check for format variants
            contextName := level.Name + "-file"
            if seriesType, ok := entity.TokenData["SeriesType"].(string); ok && seriesType != "" {
                variantKey := contextName + "." + seriesType
                if resolver.HasPattern(variantKey) {
                    contextName = variantKey
                }
            }

            fileName, err := resolver.ResolveContext(contextName, tokenCtx, ext)
            if err != nil {
                return "", fmt.Errorf("resolve %s: %w", contextName, err)
            }
            segments = append(segments, fileName)
            continue
        }

        // Intermediate level (e.g., season folder)
        // Check for conditional segment
        isConditionalEnabled := true
        for _, cond := range mod.(module.PathGenerator).ConditionalSegments() {
            if cond.DataKey != "" {
                if val, ok := entity.TokenData[cond.DataKey].(bool); ok {
                    isConditionalEnabled = val
                }
            }
        }
        if !isConditionalEnabled {
            continue // Skip this level
        }

        // Check for special node
        contextName := level.Name + "-folder"
        if level.IsSpecial {
            if isSpecial, ok := entity.TokenData["IsSpecial"].(bool); ok && isSpecial {
                contextName = "specials-folder"
            }
        }

        folderName, err := resolver.ResolveContext(contextName, tokenCtx, "")
        if err != nil {
            return "", fmt.Errorf("resolve %s: %w", contextName, err)
        }
        segments = append(segments, folderName)
    }

    return filepath.Join(segments...), nil
}
```

#### 6.10e: Module-Aware Import Dispatch

**Modify** `internal/import/pipeline.go` and `internal/import/matching.go`:

Update the `processImport` pipeline (in `pipeline.go`) and `matchToLibrary` (in `matching.go`) to use module dispatch when registry is present. The key integration points:

1. **matchToLibrary** (in `matching.go`) — when registry is set and download mapping has a module_type, dispatch to `mod.ImportHandler.MatchDownload`:

```go
func (s *Service) matchToLibrary(ctx context.Context, path string, mapping *DownloadMapping) (*LibraryMatch, error) {
    if s.registry != nil && mapping != nil && mapping.ModuleType != "" {
        return s.matchToLibraryViaModule(ctx, path, mapping)
    }
    return s.matchToLibraryLegacy(ctx, path, mapping)
}
```

2. **computeDestination** (in `pipeline.go`) — when registry is set, use `computeDestinationViaModule`:

```go
func (s *Service) computeDestination(ctx context.Context, match *LibraryMatch, ...) (string, error) {
    if s.registry != nil && match.ModuleEntity != nil {
        return s.computeDestinationViaModule(ctx, match.ModuleEntity, parsed, mi, ext)
    }
    return s.computeDestinationLegacy(ctx, match, ...)
}
```

3. **finalizeImport** — when registry is set, dispatch to `mod.ImportHandler.ImportFile`:

```go
func (s *Service) finalizeImport(ctx context.Context, match *LibraryMatch, destPath string, qi QualityInfo) error {
    if s.registry != nil && match.ModuleEntity != nil {
        return s.finalizeImportViaModule(ctx, match.ModuleEntity, destPath, qi)
    }
    return s.finalizeImportLegacy(ctx, match, destPath, qi)
}
```

**Implementation note for the subagent:**
- **This is the most complex task.** Read `internal/CLAUDE.md` before making changes.
- The `LibraryMatch` struct gains an optional `ModuleEntity *module.MatchedEntity` field. When populated (module path), the pipeline uses it. When nil (legacy path), existing code runs.
- `matchToLibraryViaModule` converts the `DownloadMapping` to a `module.CompletedDownload`, calls `mod.(module.ImportHandler).MatchDownload()`, and converts the result back to `LibraryMatch` (populating both legacy fields AND `ModuleEntity`). This ensures downstream code (status updates, WebSocket broadcasts, history logging) that reads legacy fields still works.
- **DownloadMapping → CompletedDownload derivation:** The actual `DownloadMapping` struct has `MovieID *int64`, `SeriesID *int64`, `EpisodeID *int64`, `IsSeasonPack bool`, `SeasonNumber *int`, `MediaType string` (derived). It does NOT have `module_type`/`entity_type` columns. The conversion must derive these:
  - `MovieID != nil` → `ModuleType=TypeMovie, EntityType=EntityMovie, EntityID=*MovieID`
  - `EpisodeID != nil` → `ModuleType=TypeTV, EntityType=EntityEpisode, EntityID=*EpisodeID`
  - `SeriesID != nil && IsSeasonPack` → `ModuleType=TypeTV, EntityType=EntitySeries, EntityID=*SeriesID, IsGroupDownload=true, GroupEntityType=EntitySeason`
  - Map `DownloadClientID` → `ClientID`. `ClientName` is NOT on `DownloadMapping` — populate from the download client service if needed, or leave empty (it's informational).
  - Copy `TargetSlotID`, `Source`, `SeasonNumber`, `ID` (→ MappingID), `ImportAttempts` directly.
  - `DownloadPath` comes from the download client's completion data (passed to the import pipeline), not from `DownloadMapping`.
- `finalizeImportViaModule` calls `mod.(module.ImportHandler).ImportFile()` and then performs the same post-import work as the legacy path (status update, history, notifications, slot assignment).
- `renamer.Resolver` currently has no public accessor for its `settings` field. Add a `HasPattern(contextName string) bool` method (preferred over exposing the full Settings struct) that checks `r.settings.Patterns[contextName]`. Task 6.10d uses this for variant lookup: `if resolver.HasPattern(variantKey) { contextName = variantKey }`. Alternatively, add `Settings() *Settings` but `HasPattern` is more encapsulated.
- Rename existing methods to `*Legacy` suffix (e.g., `matchToLibraryLegacy`, `computeDestinationLegacy`, `finalizeImportLegacy`). The original function names become dispatchers.
- Multi-file downloads (season packs): when the module's `SupportsMultiFileDownload()` returns true and the download is a group download, the pipeline calls `MatchIndividualFile` for each file in the download directory. Track per-file status via the existing `queue_media` table. Call `IsGroupImportReady` before proceeding with batch import.

**Verify:**
- `go build ./internal/import/...` compiles
- `go build ./internal/downloader/...` compiles
- `make test` — all existing tests pass
- Manual verification: with registry=nil, all import behavior is identical (dual-path dispatch)

---

### Task 6.11: Refactor Orphan Scan with TryMatch

**Depends on:** Tasks 6.3 + 6.4

**Modify** `internal/import/pipeline.go` — update orphan file handling in the scan pipeline.

The actual scan flow is: `ScanForPendingImports` → `scanClientDownloads` → `scanSubdirectory` (iterates `SlipStreamSubdirs`). The module-aware orphan logic should be injected at the point where individual video files are processed within `scanSubdirectory`. The pseudocode below is simplified — adapt to the actual scan method structure:

```go
func (s *Service) ScanForPendingImports(ctx context.Context) error {
    // ... existing completion/client checks ...

    subdirs := s.GetSlipStreamSubdirs()

    for _, subDir := range subdirs {
        // ... existing directory walk logic ...

        for _, file := range videoFiles {
            if s.isFileAlreadyImported(ctx, file) {
                continue
            }

            if s.registry != nil {
                s.matchOrphanViaModules(ctx, file)
            } else {
                s.matchOrphanLegacy(ctx, file)
            }
        }
    }
    return nil
}

// matchOrphanViaModules calls TryMatch on all enabled modules and dispatches to the best match.
func (s *Service) matchOrphanViaModules(ctx context.Context, filePath string) {
    filename := filepath.Base(filePath)

    var bestConfidence float64
    var bestModule module.Module
    var bestParse *module.ParseResult

    for _, mod := range s.registry.All() {
        // Check if module is enabled via settings DB (module.<id>.enabled key).
        // No IsModuleEnabledCached function exists yet — use module.IsModuleEnabled(ctx, db, mod.ID())
        // or add a simple in-memory cache to avoid per-file DB queries during scan.
        if !module.IsModuleEnabled(ctx, s.db, mod.ID()) {
            continue
        }

        confidence, parseResult := mod.(module.FileParser).TryMatch(filename)
        if confidence > bestConfidence {
            bestConfidence = confidence
            bestModule = mod
            bestParse = parseResult
        }
    }

    if bestModule == nil || bestConfidence == 0 {
        s.logger.Debug().Str("file", filename).Msg("No module claimed orphan file")
        return
    }

    s.logger.Info().
        Str("file", filename).
        Str("module", string(bestModule.ID())).
        Float64("confidence", bestConfidence).
        Msg("Orphan file claimed by module")

    // Match to library entity
    entity, err := bestModule.(module.FileParser).MatchToEntity(ctx, bestParse)
    if err != nil || entity == nil {
        s.logger.Debug().Str("file", filename).Msg("No library match for orphan file")
        return
    }

    // Queue for import
    s.queueImport(ImportJob{
        SourcePath:     filePath,
        ConfirmedMatch: entityToLibraryMatch(entity),
    })
}
```

**Implementation note for the subagent:**
- `matchOrphanLegacy` is the existing orphan scan code path. The actual scan processes files in `scanSubdirectory` → `processOrphanFile` (or similar inline logic). Extract the existing handling into `matchOrphanLegacy` for the dual-path pattern.
- `entityToLibraryMatch` converts a `module.MatchedEntity` to a `LibraryMatch` for the existing import pipeline.
- `IsModuleEnabled` — no `internal/module/settings.go` exists yet; it is created during Phase 0. The function checks the `settings` table for key `module.<id>.enabled`. For the orphan scan, add a simple in-memory cache (populated once at the start of `ScanForPendingImports`) to avoid per-file DB queries.
- The orphan scan should handle the case where multiple modules claim a file with similar confidence — take the highest, with ties broken by registration order.

**Verify:**
- `go build ./internal/import/...` compiles
- Orphan scan still works in legacy mode (registry=nil)

---

### Task 6.12: Wire DI Integration

**Depends on:** Tasks 6.10 + 6.11

Wire the registry into the import service and downloader.

**Modify** `internal/api/setters.go`:

Add `SetRegistry` calls for the import service and download queue service:

```go
// In the setters initialization function:
importService.SetRegistry(registry)
// If downloader needs registry for detectModuleType:
downloadQueue.SetRegistry(registry)
```

**Modify** `internal/api/wire.go` — if the import service or downloader constructors need updated, add the registry as a dependency. More likely, the registry is injected via `setters.go` (late-binding pattern consistent with Phases 3–5).

**Modify** module `Wire()` methods if needed — the movie and TV modules now have more interface implementations. Their `Wire()` provider sets should return constructors that provide the full `Module` struct with all Phase 6 implementations wired:

```go
// internal/modules/movie/module.go — update Module struct
type Module struct {
    *Descriptor
    // Phase 3:
    *metadataProvider
    // Phase 4:
    *monitoringPresets
    *releaseDateResolver
    *wantedCollector
    // Phase 5:
    *searchStrategy
    // Phase 6:
    *fileParser
    *pathGenerator
    *namingProvider
    *importHandler
    // ... future phases add more ...
}
```

Run: `make wire` to regenerate `wire_gen.go`.

**Verify:**
- `make wire` succeeds
- `make build` succeeds
- `make test` passes
- `make lint` passes

---

### Task 6.13: Phase 6 Validation

**Run all of these after all Phase 6 tasks complete:**

1. `go build ./internal/module/parseutil/...` — parseutil compiles
2. `go build ./internal/modules/movie/...` — movie module compiles with all Phase 6 interfaces
3. `go build ./internal/modules/tv/...` — TV module compiles with all Phase 6 interfaces
4. `go build ./internal/import/...` — import pipeline compiles
5. `go build ./internal/downloader/...` — downloader compiles
6. `go test ./internal/module/parseutil/...` — parseutil tests pass
7. `go test ./internal/library/scanner/...` — scanner tests still pass (wraps parseutil)
8. `go test ./internal/import/...` — import tests still pass
9. `make build` — full project builds
10. `make test` — all existing tests pass
11. `make lint` — no new lint issues

**Compile-time interface assertions that must exist:**

```
internal/modules/movie/module.go:  var _ module.FileParser = (*fileParser)(nil)
internal/modules/movie/module.go:  var _ module.PathGenerator = (*pathGenerator)(nil)
internal/modules/movie/module.go:  var _ module.NamingProvider = (*namingProvider)(nil)
internal/modules/movie/module.go:  var _ module.ImportHandler = (*importHandler)(nil)
internal/modules/tv/module.go:    var _ module.FileParser = (*fileParser)(nil)
internal/modules/tv/module.go:    var _ module.PathGenerator = (*pathGenerator)(nil)
internal/modules/tv/module.go:    var _ module.NamingProvider = (*namingProvider)(nil)
internal/modules/tv/module.go:    var _ module.ImportHandler = (*importHandler)(nil)
```

**What exists after Phase 6:**
- `internal/module/parseutil/` — shared parsing utility library (quality, title, metadata, extensions)
- `internal/modules/movie/` — Movie module implements FileParser, PathGenerator, NamingProvider, ImportHandler (plus all earlier interfaces)
- `internal/modules/tv/` — TV module implements FileParser, PathGenerator, NamingProvider, ImportHandler (plus all earlier interfaces)
- `internal/database/migrations/0XX_module_naming_settings.sql` — per-module naming settings table
- `internal/database/queries/module_naming_settings.sql` — sqlc queries
- `internal/import/module_naming.go` — per-module renamer loader
- `internal/import/token_context.go` — centralized TokenData → TokenContext converter
- `internal/import/renamer/resolver.go` — gains `Patterns` map and `ResolveContext` method
- `internal/import/pipeline.go` — dual-path dispatch for destination computation, finalization, and orphan scan
- `internal/import/matching.go` — dual-path dispatch for matchToLibrary
- `internal/import/service.go` — registry field, module resolver cache, `SetRegistry` method
- `internal/downloader/queue.go` — module-aware `detectModuleType`
- Zero changes to frontend (all API shapes preserved)

**Files touched summary:**

**New files:**
```
internal/module/parseutil/quality.go
internal/module/parseutil/title.go
internal/module/parseutil/metadata.go
internal/module/parseutil/extensions.go
internal/module/parseutil/parseutil_test.go
internal/modules/movie/fileparser.go
internal/modules/movie/path_naming.go
internal/modules/movie/importhandler.go
internal/modules/tv/fileparser.go
internal/modules/tv/path_naming.go
internal/modules/tv/importhandler.go
internal/database/migrations/0XX_module_naming_settings.sql
internal/database/queries/module_naming_settings.sql
internal/import/module_naming.go
internal/import/token_context.go
```

**Modified files:**
```
internal/module/parse_types.go (flesh out ParseResult — MonitoringPreset already done in Phase 4)
internal/module/import_types.go (flesh out CompletedDownload, MatchedEntity, QualityInfo, ImportResult, MediaInfoFieldDecl)
internal/module/path_types.go (flesh out TemplateVariable, ConditionalSegment, TokenContext, FormatOption)
internal/library/scanner/parser.go (delegate to parseutil — wrap, don't remove)
internal/library/scanner/extensions.go (delegate to parseutil)
internal/import/matching.go (use parseutil.NormalizeTitle, dual-path matchToLibrary dispatch)
internal/import/renamer/resolver.go (add Patterns map on Settings, add ResolveContext + HasPattern methods)
internal/import/pipeline.go (dual-path dispatch for computeDestination/finalizeImport, orphan scan refactor)
internal/import/service.go (registry field, moduleResolvers cache, SetRegistry, LibraryMatch gains ModuleEntity)
internal/downloader/queue.go (detectModuleType with registry, legacy fallback; also referenced in completion.go)
internal/modules/movie/module.go (Module struct gains Phase 6 fields, compile-time assertions)
internal/modules/tv/module.go (Module struct gains Phase 6 fields, compile-time assertions)
internal/api/setters.go (SetRegistry for import service and downloader)
internal/api/wire_gen.go (regenerated)
internal/database/sqlc/* (regenerated from sqlc generate)
```

---

