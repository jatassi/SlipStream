# Release Importing Implementation Plan

This document outlines the implementation plan for the Release Importing feature as specified in [release-importing-spec.md](./release-importing-spec.md).

## Architecture Overview

The import system is built on existing SlipStream infrastructure:

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                           IMPORT PIPELINE                                     │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐   │
│  │  Triggers   │───▶│ Validation  │───▶│   Rename    │───▶│   Import    │   │
│  │             │    │             │    │             │    │             │   │
│  │ • Watcher   │    │ • Size      │    │ • Tokens    │    │ • Hardlink  │   │
│  │ • Scheduler │    │ • Extension │    │ • MediaInfo │    │ • Copy      │   │
│  │ • Startup   │    │ • Probe     │    │ • Character │    │ • Cleanup   │   │
│  │ • Manual    │    │ • Sample    │    │   Replace   │    │ • History   │   │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘   │
│         │                  │                  │                  │          │
│         └──────────────────┴──────────────────┴──────────────────┘          │
│                                      │                                       │
│                             ┌────────▼────────┐                             │
│                             │  Health System  │                             │
│                             │  + WebSocket    │                             │
│                             │  + History      │                             │
│                             └─────────────────┘                             │
└──────────────────────────────────────────────────────────────────────────────┘
```

## Code Reuse Strategy

### Existing Components to Extend

| Component | Location | Extension Needed |
|-----------|----------|------------------|
| `download_mappings` table | `migrations/012_*` | Add `file_status` per-file tracking |
| `Organizer` service | `internal/library/organizer/` | Add TV episode renaming, token expansion |
| `ParseFilename()` | `internal/library/scanner/parser.go` | Already complete, use as-is |
| `Watcher` service | `internal/watcher/` | Add conditional activation, import triggering |
| `Health` service | `internal/health/` | Add "import" category |
| `History` service | `internal/history/` | Add import event types |
| `Queue` service | `internal/downloader/queue.go` | Add completion detection |

### New Components Required

| Component | Location | Purpose |
|-----------|----------|---------|
| `Import` service | `internal/import/service.go` | Orchestrate import pipeline |
| `MediaInfo` service | `internal/mediainfo/service.go` | Extract video metadata |
| `Renamer` service | `internal/import/renamer/` | Token-based filename generation |
| Import settings | `internal/config/import.go` | Import configuration |
| Import API handlers | `internal/api/handlers/import.go` | REST endpoints |

---

## Phase 1: Database Schema

### 1.1 Queue-Media Junction Table

Extends existing `download_mappings` to support per-file status tracking for season packs.

**File**: `internal/database/migrations/XXX_add_queue_media.sql`

```sql
-- Junction table for queue items to individual episodes/movies
CREATE TABLE IF NOT EXISTS queue_media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    download_mapping_id INTEGER NOT NULL REFERENCES download_mappings(id) ON DELETE CASCADE,
    episode_id INTEGER REFERENCES episodes(id) ON DELETE CASCADE,
    movie_id INTEGER REFERENCES movies(id) ON DELETE CASCADE,
    file_path TEXT,
    file_status TEXT NOT NULL DEFAULT 'pending'
        CHECK(file_status IN ('pending', 'downloading', 'ready', 'importing', 'imported', 'failed')),
    error_message TEXT,
    import_attempts INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(download_mapping_id, episode_id),
    UNIQUE(download_mapping_id, movie_id)
);

CREATE INDEX idx_queue_media_status ON queue_media(file_status);
CREATE INDEX idx_queue_media_episode ON queue_media(episode_id);
CREATE INDEX idx_queue_media_movie ON queue_media(movie_id);
```

#### Tasks
- [x] Create migration file for queue_media table
- [x] Add sqlc queries for queue_media CRUD operations
- [x] Generate sqlc Go code
- [ ] Add helper methods in downloader service
- [ ] Implement queue record creation flow: Send to download client → Create queue record from response
- [ ] Add import_attempts tracking for retry logic

### 1.2 Import Settings Table

**File**: `internal/database/migrations/XXX_add_import_settings.sql`

```sql
CREATE TABLE IF NOT EXISTS import_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1), -- Singleton row

    -- Validation settings
    validation_level TEXT NOT NULL DEFAULT 'standard'
        CHECK(validation_level IN ('basic', 'standard', 'full')),
    minimum_file_size_mb INTEGER NOT NULL DEFAULT 100,
    video_extensions TEXT NOT NULL DEFAULT '.mkv,.mp4,.avi,.m4v,.mov,.wmv,.ts,.m2ts,.webm',

    -- Matching settings
    match_conflict_behavior TEXT NOT NULL DEFAULT 'trust_queue'
        CHECK(match_conflict_behavior IN ('trust_queue', 'trust_parse', 'fail')),
    unknown_media_behavior TEXT NOT NULL DEFAULT 'ignore'
        CHECK(unknown_media_behavior IN ('ignore', 'auto_add')),

    -- Renaming settings (TV)
    rename_episodes BOOLEAN NOT NULL DEFAULT true,
    replace_illegal_characters BOOLEAN NOT NULL DEFAULT true,
    colon_replacement TEXT NOT NULL DEFAULT 'smart'
        CHECK(colon_replacement IN ('delete', 'dash', 'space_dash', 'space_dash_space', 'smart', 'custom')),
    custom_colon_replacement TEXT DEFAULT NULL,

    -- Episode naming patterns
    standard_episode_format TEXT NOT NULL DEFAULT '{Series Title} - S{season:00}E{episode:00} - {Quality Title}',
    daily_episode_format TEXT NOT NULL DEFAULT '{Series Title} - {Air-Date} - {Episode Title} {Quality Full}',
    anime_episode_format TEXT NOT NULL DEFAULT '{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}',

    -- Folder patterns
    series_folder_format TEXT NOT NULL DEFAULT '{Series Title}',
    season_folder_format TEXT NOT NULL DEFAULT 'Season {season}',
    specials_folder_format TEXT NOT NULL DEFAULT 'Specials',

    -- Multi-episode style
    multi_episode_style TEXT NOT NULL DEFAULT 'extend'
        CHECK(multi_episode_style IN ('extend', 'duplicate', 'repeat', 'scene', 'range', 'prefixed_range')),

    -- Movie naming patterns
    rename_movies BOOLEAN NOT NULL DEFAULT true,
    movie_folder_format TEXT NOT NULL DEFAULT '{Movie Title} ({Year})',
    movie_file_format TEXT NOT NULL DEFAULT '{Movie Title} ({Year}) - {Quality Title}',

    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Ensure singleton row exists
INSERT OR IGNORE INTO import_settings (id) VALUES (1);
```

#### Tasks
- [x] Create migration file for import_settings table
- [x] Add sqlc queries for settings read/update
- [x] Generate sqlc Go code
- [ ] Create ImportSettings Go struct with field mappings

### 1.3 Import History Extension

Extend existing history table with import-specific event types.

**File**: `internal/database/migrations/XXX_extend_history_types.sql`

```sql
-- No migration needed, just add new event types to history service
-- Event types: import_started, import_completed, import_failed, import_upgrade
```

#### Tasks
- [ ] Add import event type constants to history service
- [ ] Add history queries for import events
- [ ] Add CreateImportEvent helper method

### 1.4 Media Files Extensions

Add fields to track original release info.

**File**: `internal/database/migrations/XXX_extend_media_files.sql`

```sql
ALTER TABLE movie_files ADD COLUMN original_path TEXT;
ALTER TABLE movie_files ADD COLUMN original_filename TEXT;
ALTER TABLE movie_files ADD COLUMN imported_at DATETIME;

ALTER TABLE episode_files ADD COLUMN original_path TEXT;
ALTER TABLE episode_files ADD COLUMN original_filename TEXT;
ALTER TABLE episode_files ADD COLUMN imported_at DATETIME;
```

#### Tasks
- [x] Create migration file for media files extensions
- [x] Update sqlc queries to include new fields
- [x] Generate sqlc Go code

### 1.5 Download Client Import Settings

Add import-related fields to download_clients table.

**File**: `internal/database/migrations/XXX_extend_download_clients.sql`

```sql
ALTER TABLE download_clients ADD COLUMN import_delay_seconds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE download_clients ADD COLUMN cleanup_mode TEXT NOT NULL DEFAULT 'leave'
    CHECK(cleanup_mode IN ('leave', 'delete_after_import', 'delete_after_seed_ratio'));
ALTER TABLE download_clients ADD COLUMN seed_ratio_target REAL DEFAULT NULL;
```

#### Tasks
- [x] Create migration file for download_clients import fields
- [x] Update sqlc queries and Go types
- [ ] Add UI for configuring per-client import delay and cleanup mode

### 1.6 Series Format Type Override

Add field to series table for manual format type override.

**File**: `internal/database/migrations/XXX_add_series_format_type.sql`

```sql
ALTER TABLE series ADD COLUMN format_type TEXT DEFAULT NULL
    CHECK(format_type IS NULL OR format_type IN ('standard', 'daily', 'anime'));
-- NULL means auto-detect from metadata
```

#### Tasks
- [x] Create migration file for series format_type
- [x] Update sqlc queries
- [ ] Add UI for per-series format type override

---

## Phase 2: Core Import Service

### 2.1 Import Service Structure

**File**: `internal/import/service.go`

```go
type Service struct {
    db           *sql.DB
    queries      *sqlc.Queries
    downloader   *downloader.Service
    movies       *movies.Service
    tv           *tv.Service
    organizer    *organizer.Service
    renamer      *renamer.Service
    mediainfo    *mediainfo.Service
    health       *health.Service
    history      *history.Service
    hub          *websocket.Hub
    logger       zerolog.Logger

    importQueue  chan ImportJob
    mu           sync.Mutex
    processing   map[string]bool  // Track in-progress imports by path
}

type ImportJob struct {
    SourcePath       string
    DownloadMapping  *DownloadMapping  // nil for manual imports
    QueueMedia       *QueueMedia       // nil for single-file downloads
    Manual           bool
    ConfirmedMatch   *ConfirmedMatch   // For manual imports
}
```

#### Tasks
- [x] Create import service package structure
- [x] Implement service constructor with dependency injection
- [x] Add service to API server initialization
- [x] Implement sequential job queue processing

### 2.2 Import Pipeline Methods

**File**: `internal/import/pipeline.go`

```go
// Main entry points
func (s *Service) ProcessCompletedDownload(ctx context.Context, mapping *DownloadMapping) error
func (s *Service) ProcessManualImport(ctx context.Context, job ManualImportJob) error
func (s *Service) ScanForPendingImports(ctx context.Context) error

// Pipeline stages
func (s *Service) validateFile(ctx context.Context, path string, settings *ImportSettings) error
func (s *Service) matchToLibrary(ctx context.Context, path string, mapping *DownloadMapping) (*LibraryMatch, error)
func (s *Service) computeDestination(ctx context.Context, match *LibraryMatch, mediaInfo *MediaInfo) (string, error)
func (s *Service) executeImport(ctx context.Context, source, dest string, useHardlink bool) error
func (s *Service) updateLibrary(ctx context.Context, match *LibraryMatch, destPath string) error
func (s *Service) handleCleanup(ctx context.Context, sourcePath string, clientConfig *ClientConfig) error
```

#### Tasks
- [x] Implement ProcessCompletedDownload orchestrator
- [x] Implement ProcessManualImport for user-initiated imports
- [x] Implement ScanForPendingImports for startup and periodic checks
- [x] Implement validateFile with configurable strictness levels
- [x] Implement matchToLibrary with queue-based and parse-based matching
- [x] Implement computeDestination using renamer service
- [x] Implement executeImport with hardlink/copy logic (hardlink to library, original stays for seeding)
- [x] Implement updateLibrary for movie/episode file records
- [x] Implement handleCleanup per download client config (leave/delete/seed-ratio modes)
- [x] Implement stalled/paused download detection and skip logic
- [x] Implement season pack splitting: detect multi-episode downloads, create individual queue_media entries
- [x] Implement individual episode import from season pack (don't wait for full pack)
- [ ] Implement unknown media auto-add mode with metadata refresh trigger
- [x] Preserve file extension exactly as-is during rename

### 2.3 File Validation

**File**: `internal/import/validation.go`

```go
type ValidationLevel string
const (
    ValidationBasic    ValidationLevel = "basic"
    ValidationStandard ValidationLevel = "standard"
    ValidationFull     ValidationLevel = "full"
)

func (s *Service) validateFile(ctx context.Context, path string, settings *ImportSettings) error {
    // 1. Check file exists
    // 2. Check size > 0 (basic)
    // 3. Check size > minimum (standard+)
    // 4. Check valid extension (standard+)
    // 5. Check not sample file (standard+)
    // 6. MediaInfo probe (full only)
}

func (s *Service) isSampleFile(path string) bool
func (s *Service) isValidExtension(path string, allowedExts []string) bool
func (s *Service) checkFileSize(path string, minSizeMB int64) error
```

#### Tasks
- [x] Implement file existence and basic size check
- [x] Implement minimum file size validation
- [x] Implement video extension whitelist check
- [x] Implement sample file detection (folder + filename)
- [x] Implement MediaInfo probe validation (full level)
- [x] Add configurable validation level support

### 2.4 File Completion Detection

**File**: `internal/import/completion.go`

```go
type CompletionChecker struct {
    downloader *downloader.Service
    logger     zerolog.Logger
}

func (c *CompletionChecker) IsFileComplete(ctx context.Context, path string, clientID int64) (bool, error) {
    // Priority order:
    // 1. Query download client API for file status
    // 2. Check file size stability (15-30 seconds)
    // 3. Check file lock status
}

func (c *CompletionChecker) checkSizeStability(path string, duration time.Duration) (bool, error)
func (c *CompletionChecker) isFileLocked(path string) bool
func (c *CompletionChecker) hasArchiveFiles(dirPath string) bool
func (c *CompletionChecker) waitForExtraction(ctx context.Context, dirPath string) error
```

#### Tasks
- [x] Implement download client API status check (priority 1)
- [x] Implement file size stability detection (15-30 sec wait) (priority 2)
- [x] Implement file lock detection (priority 3)
- [x] Implement archive file detection (.rar, .zip, .7z)
- [x] Implement extraction wait logic - wait for extracted files before importing
- [ ] Monitor for file size changes during watcher events

### 2.5 Library Matching

**File**: `internal/import/matching.go`

```go
type LibraryMatch struct {
    MediaType   string  // "movie" or "episode"
    MovieID     *int64
    SeriesID    *int64
    SeasonNum   *int
    EpisodeID   *int64
    Confidence  float64 // 0.0 - 1.0
    Source      string  // "queue", "parse", "manual"
}

func (s *Service) matchToLibrary(ctx context.Context, path string, mapping *DownloadMapping) (*LibraryMatch, error) {
    // 1. If mapping exists, use queue-based match
    // 2. Parse filename for backup match
    // 3. Compare and resolve conflicts per config
}

func (s *Service) queueBasedMatch(ctx context.Context, mapping *DownloadMapping) (*LibraryMatch, error)
func (s *Service) parseBasedMatch(ctx context.Context, path string) (*LibraryMatch, error)
func (s *Service) resolveMatchConflict(queue, parsed *LibraryMatch, behavior string) (*LibraryMatch, error)
```

#### Tasks
- [x] Implement queue-based matching from download_mappings
- [x] Implement parse-based matching using existing ParseFilename
- [x] Implement conflict resolution (trust_queue/trust_parse/fail)
- [x] Add confidence scoring for parse matches
- [ ] Implement episode matching for season packs

### 2.6 Error Handling & Retry

**File**: `internal/import/retry.go`

```go
const MaxImportAttempts = 2

func (s *Service) processWithRetry(ctx context.Context, job ImportJob) error {
    var lastErr error
    for attempt := 1; attempt <= MaxImportAttempts; attempt++ {
        if err := s.processImport(ctx, job); err != nil {
            lastErr = err
            s.logger.Warn().Err(err).Int("attempt", attempt).Msg("Import attempt failed")
            if attempt < MaxImportAttempts {
                continue // Immediate retry
            }
        } else {
            return nil
        }
    }
    return s.handleImportFailure(ctx, job, lastErr)
}

func (s *Service) handleImportFailure(ctx context.Context, job ImportJob, err error) error {
    // Update queue_media status to 'failed'
    // Register health warning
    // Log to history
}
```

#### Tasks
- [x] Implement retry logic (immediate retry once per file)
- [x] Implement failure handling with health system integration
- [x] Implement partial failure handling for multi-file downloads (continue with other files)
- [x] After all files in a batch attempted, retry failed files once each
- [x] Add import attempt tracking in queue_media
- [x] Total: 2 attempts per file maximum (implemented as 3 attempts with exponential backoff)

---

## Phase 3: Renaming System

### 3.1 Token Engine

**File**: `internal/import/renamer/tokens.go`

```go
type TokenContext struct {
    // Series info
    SeriesTitle    string
    SeriesYear     int
    SeriesType     string  // "standard", "daily", "anime"

    // Episode info
    SeasonNumber   int
    EpisodeNumber  int
    EpisodeNumbers []int   // For multi-episode
    AbsoluteNumber int     // For anime
    EpisodeTitle   string
    AirDate        time.Time

    // Quality info
    Quality        string  // "1080p", "720p"
    Source         string  // "WEBDL", "BluRay"
    Codec          string  // "x264", "x265"
    Revision       string  // "Proper", "Repack"

    // MediaInfo
    VideoCodec     string
    VideoBitDepth  int
    DynamicRange   string  // "HDR10", "DV"
    AudioCodec     string
    AudioChannels  string
    AudioLangs     []string
    SubtitleLangs  []string

    // Other
    ReleaseGroup   string
    OriginalTitle  string
    OriginalFile   string
    CustomFormats  []string
}

type Token struct {
    Name      string
    Separator string  // "" (space), ".", "-", "_"
    Modifier  string  // padding, truncation, filter
}

func ParseToken(raw string) (*Token, error)
func (t *Token) Resolve(ctx *TokenContext) string
```

#### Tasks
- [x] Implement Token struct and parsing
- [x] Implement all series tokens ({Series Title}, {Series TitleYear}, {Series CleanTitle}, {Series CleanTitleYear})
- [x] Implement season tokens ({season:0}, {season:00})
- [x] Implement episode tokens ({episode:0}, {episode:00})
- [x] Implement air date tokens ({Air-Date}, {Air Date})
- [x] Implement episode title tokens ({Episode Title}, {Episode CleanTitle})
- [x] Implement quality tokens ({Quality Full}, {Quality Title})
- [x] Implement all MediaInfo tokens (Simple, Full, VideoCodec, VideoBitDepth, VideoDynamicRange, VideoDynamicRangeType, AudioCodec, AudioChannels, AudioLanguages, SubtitleLanguages)
- [x] Implement other tokens ({Release Group}, {Custom Formats}, {Custom Format:Name}, {Original Title}, {Original Filename})
- [x] Implement anime-specific tokens ({absolute:0}, {absolute:00}, {absolute:000}, {version})
- [x] Implement {Revision} token (separate from quality)
- [x] Implement multi-episode title concatenation with separator
- [x] Implement daily show title fallback (use formatted air date when no title)

### 3.2 Token Modifiers

**File**: `internal/import/renamer/modifiers.go`

```go
// Separator modifiers
func applySeparator(value, separator string) string  // Space, ., -, _

// Case modifiers
func applyCase(value, caseType string) string  // upper, lower, title

// Truncation modifiers
func applyTruncation(value string, limit int, fromStart bool) string

// Language filtering
func filterLanguages(langs []string, filter string) []string
```

#### Tasks
- [x] Implement separator modifier (space, period, dash, underscore)
- [x] Implement case transformation (upper, lower, title)
- [x] Implement truncation with ellipsis
- [x] Implement language filtering for MediaInfo tokens
- [x] Implement multi-track language formatting (configurable separator/brackets)

### 3.3 Character Replacement

**File**: `internal/import/renamer/characters.go`

```go
var IllegalCharacters = []rune{'\\', '/', ':', '*', '?', '"', '<', '>', '|'}

type ColonReplacement string
const (
    ColonDelete         ColonReplacement = "delete"
    ColonDash           ColonReplacement = "dash"
    ColonSpaceDash      ColonReplacement = "space_dash"
    ColonSpaceDashSpace ColonReplacement = "space_dash_space"
    ColonSmart          ColonReplacement = "smart"
    ColonCustom         ColonReplacement = "custom"
)

func ReplaceIllegalCharacters(s string, replace bool, colonMode ColonReplacement, customColon string) string
func SmartColonReplace(s string) string  // Contextual replacement
func CleanTitle(s string) string  // Remove only filesystem-illegal chars
```

#### Tasks
- [x] Implement illegal character detection
- [x] Implement character replacement (replace vs remove)
- [x] Implement all colon replacement modes
- [x] Implement smart colon replacement (contextual)
- [x] Implement CleanTitle variant (preserve Unicode)

### 3.4 Multi-Episode Formatting

**File**: `internal/import/renamer/multiepisode.go`

```go
type MultiEpisodeStyle string
const (
    StyleExtend        MultiEpisodeStyle = "extend"         // S01E01-02-03
    StyleDuplicate     MultiEpisodeStyle = "duplicate"      // S01E01.S01E02
    StyleRepeat        MultiEpisodeStyle = "repeat"         // S01E01E02E03
    StyleScene         MultiEpisodeStyle = "scene"          // S01E01-E02-E03
    StyleRange         MultiEpisodeStyle = "range"          // S01E01-03
    StylePrefixedRange MultiEpisodeStyle = "prefixed_range" // S01E01-E03
)

func FormatMultiEpisode(season int, episodes []int, style MultiEpisodeStyle, padding int) string
```

#### Tasks
- [x] Implement Extend style (S01E01-02-03)
- [x] Implement Duplicate style (S01E01.S01E02)
- [x] Implement Repeat style (S01E01E02E03)
- [x] Implement Scene style (S01E01-E02-E03)
- [x] Implement Range style (S01E01-03)
- [x] Implement Prefixed Range style (S01E01-E03)

### 3.5 Pattern Resolver

**File**: `internal/import/renamer/resolver.go`

```go
type Resolver struct {
    settings *ImportSettings
}

func (r *Resolver) ResolveEpisodeFilename(ctx *TokenContext) (string, error)
func (r *Resolver) ResolveMovieFilename(ctx *TokenContext) (string, error)
func (r *Resolver) ResolveSeriesFolderName(ctx *TokenContext) string
func (r *Resolver) ResolveSeasonFolderName(seasonNum int) string
func (r *Resolver) ResolveMovieFolderName(ctx *TokenContext) string

// Validation
func (r *Resolver) ValidatePattern(pattern string) error
func (r *Resolver) PreviewPattern(pattern string, sampleData *TokenContext) (string, error)
```

#### Tasks
- [x] Implement pattern tokenization and parsing
- [x] Implement token resolution with context
- [x] Implement empty token cleanup (remove token AND surrounding separators, prevent double-spaces)
- [x] Implement path length validation (Windows MAX_PATH = 260 chars)
- [x] Fail import with error if path exceeds limit (no auto-truncation)
- [x] Implement strict pattern validation for settings UI (reject invalid tokens immediately)
- [x] Implement preview generation with sample data
- [x] Note: No per-token length limits - only total path length is enforced
- [x] Note: No conditional tokens - keep patterns simple, empty token cleanup handles most cases

### 3.6 Extend Existing Organizer

**File**: `internal/library/organizer/organizer.go` (extend)

The existing organizer has basic file operations. Extend with:

```go
// Add to existing organizer
func (o *Organizer) CreateHardlink(source, dest string) error
func (o *Organizer) DeleteFileIfUpgrade(oldPath string, newImport bool) error
func (o *Organizer) HandleExtraFiles(sourceDir, destDir string, mainFile string) error
```

#### Tasks
- [x] Implement hardlink creation (cross-platform)
- [x] Implement symlink fallback when hardlinks not supported (cross-filesystem)
- [x] Note: Hardlink approach avoids file lock issues - original stays for seeding, library hardlink is renamed
- [x] Implement upgrade file deletion (delete old file immediately upon successful import)
- [x] Implement extra files handling (subtitles, NFO - preserve original names, only rename main video)
- [x] Implement folder permission inheritance (inherit from parent folder)
- [x] Implement file collision handling: always overwrite existing file
- [x] Implement folder rename on series title/pattern change (move entire series structure)

---

## Phase 4: MediaInfo Integration

### 4.1 MediaInfo Service

**File**: `internal/mediainfo/service.go`

```go
type Service struct {
    cache  map[string]*MediaInfo
    mutex  sync.RWMutex
    logger zerolog.Logger
}

type MediaInfo struct {
    VideoCodec      string
    VideoBitDepth   int
    VideoResolution string  // "1920x1080"
    DynamicRange    string  // "HDR10", "DV HDR10", "SDR"
    AudioCodec      string  // "DTS-HD MA", "TrueHD Atmos"
    AudioChannels   string  // "7.1", "5.1"
    AudioLanguages  []string
    SubtitleLanguages []string
    Duration        time.Duration
    ContainerFormat string
}

func (s *Service) Probe(path string) (*MediaInfo, error)
func (s *Service) ProbeWithFallback(path string, parsed *ParsedMedia) *MediaInfo
```

#### Tasks
- [x] Research Go MediaInfo library options (or shell out to mediainfo CLI)
- [x] Implement Probe method for video file analysis
- [x] Implement caching to avoid re-probing
- [x] Implement fallback to parsed release info on probe failure
- [x] Map codec names to standard format (x265 → HEVC, etc.)
- [x] Detect HDR types (DV, HDR10+, HDR10, HLG)
- [x] Extract audio track languages
- [x] Extract subtitle track languages

### 4.2 MediaInfo CLI Wrapper (if using CLI)

**File**: `internal/mediainfo/cli.go`

```go
func (s *Service) probeViaCLI(path string) (*MediaInfo, error) {
    // Execute: mediainfo --Output=JSON <path>
    // Parse JSON output
    // Map to MediaInfo struct
}
```

#### Tasks
- [x] Implement CLI execution with timeout
- [x] Parse JSON output from mediainfo
- [x] Handle CLI not found gracefully
- [x] Add configuration for mediainfo binary path

---

## Phase 5: Import Triggers

### 5.1 Extend Watcher Service

**File**: `internal/watcher/service.go` (extend)

```go
// Add conditional activation
func (s *Service) SetQueueHasItems(hasItems bool)
func (s *Service) shouldWatch() bool

// Add import trigger
func (s *Service) handleImportableFile(ctx context.Context, path string) error
```

#### Tasks
- [x] Add queue item count tracking
- [x] Implement conditional watcher activation (only when queue has items)
- [x] Add download folder monitoring (monitor for new files, size changes, file completion)
- [ ] Integrate with import service for file detection
- [x] Add debounce for repeated filesystem events (debounce by file path within short window)

### 5.2 Scheduler Integration

**File**: `internal/scheduler/import_tasks.go`

```go
var ImportScanTask = TaskConfig{
    ID:          "import-scan",
    Name:        "Import Scan",
    Description: "Scan download folders for completed files",
    Cron:        "*/5 * * * *",  // Every 5 minutes
    Func:        scanForImports,
    RunOnStart:  true,
}

func scanForImports(ctx context.Context) error {
    // Call import.Service.ScanForPendingImports()
}
```

#### Tasks
- [x] Create import scan scheduled task
- [x] Implement startup scan on application launch
- [x] Register task with scheduler service

### 5.3 Download Client Polling Enhancement

**File**: `internal/downloader/service.go` (extend)

```go
func (s *Service) CheckForCompletedDownloads(ctx context.Context) ([]CompletedDownload, error) {
    // Get all queue items
    // Filter for status == "completed"
    // Return items ready for import
}

func (s *Service) GetDownloadPath(ctx context.Context, clientID int64, downloadID string) (string, error) {
    // Get the actual file path from download client
}
```

#### Tasks
- [x] Add completion status detection to queue polling
- [x] Implement file path retrieval from download clients
- [x] Add import delay handling per client configuration
- [x] Emit WebSocket events for completed downloads

---

## Phase 6: Health & History Integration

### 6.1 Health System Extension

**File**: `internal/health/service.go` (extend)

```go
const CategoryImport = "import"

// Add import-specific methods
func (s *Service) RegisterPendingImport(id, name string)
func (s *Service) SetImportError(id, message string)
func (s *Service) SetImportWarning(id, message string)
func (s *Service) ClearImportStatus(id string)
```

#### Tasks
- [x] Add "import" category to health service
- [x] Implement import error/warning registration
- [x] Add WebSocket broadcast for import health changes
- [x] Clear import health status on successful import

### 6.2 History Service Extension

**File**: `internal/history/service.go` (extend)

```go
const (
    EventTypeImportStarted   = "import_started"
    EventTypeImportCompleted = "import_completed"
    EventTypeImportFailed    = "import_failed"
    EventTypeImportUpgrade   = "import_upgrade"
)

type ImportEventData struct {
    SourcePath       string `json:"source_path"`
    DestinationPath  string `json:"destination_path"`
    OriginalFilename string `json:"original_filename"`
    FinalFilename    string `json:"final_filename"`
    Quality          string `json:"quality"`
    Source           string `json:"source"`
    Codec            string `json:"codec"`
    Size             int64  `json:"size"`
    Error            string `json:"error,omitempty"`
    IsUpgrade        bool   `json:"is_upgrade"`
    PreviousFile     string `json:"previous_file,omitempty"`
}

func (s *Service) LogImportEvent(ctx context.Context, eventType string, mediaType string, mediaID int64, data *ImportEventData) error
```

#### Tasks
- [x] Add import event type constants
- [x] Implement LogImportEvent helper
- [x] Store original/final filenames in history data
- [x] Track upgrade operations with previous file info

---

## Phase 7: API Endpoints

### 7.1 Import API Handlers

**File**: `internal/api/handlers/import.go`

```go
// GET /api/v1/import/pending
func (h *ImportHandler) GetPendingImports(c echo.Context) error

// POST /api/v1/import/manual
func (h *ImportHandler) ManualImport(c echo.Context) error

// POST /api/v1/import/manual/preview
func (h *ImportHandler) PreviewManualImport(c echo.Context) error

// GET /api/v1/import/history
func (h *ImportHandler) GetImportHistory(c echo.Context) error

// POST /api/v1/import/{id}/retry
func (h *ImportHandler) RetryImport(c echo.Context) error
```

#### Tasks
- [x] Create import handler struct
- [x] Implement GetPendingImports endpoint
- [x] Implement ManualImport endpoint
- [x] Implement PreviewManualImport (show suggested matches)
- [x] Implement GetImportHistory with filtering
- [x] Implement RetryImport for failed imports
- [x] Register routes in server.go

### 7.2 Import Settings API

**File**: `internal/api/handlers/settings.go` (extend)

```go
// GET /api/v1/settings/import
func (h *SettingsHandler) GetImportSettings(c echo.Context) error

// PUT /api/v1/settings/import
func (h *SettingsHandler) UpdateImportSettings(c echo.Context) error

// POST /api/v1/settings/import/naming/preview
func (h *SettingsHandler) PreviewNamingPattern(c echo.Context) error

// POST /api/v1/settings/import/naming/validate
func (h *SettingsHandler) ValidateNamingPattern(c echo.Context) error
```

#### Tasks
- [x] Implement GetImportSettings endpoint
- [x] Implement UpdateImportSettings endpoint
- [x] Implement PreviewNamingPattern for live preview
- [x] Implement ValidateNamingPattern for syntax checking
- [x] Add sample data for pattern preview

### 7.3 Manual Import Browse API

**File**: `internal/api/handlers/filesystem.go` (extend)

```go
// GET /api/v1/filesystem/browse/import
func (h *FilesystemHandler) BrowseForImport(c echo.Context) error

// POST /api/v1/filesystem/scan
func (h *FilesystemHandler) ScanForMedia(c echo.Context) error
```

#### Tasks
- [x] Extend browse endpoint to show files (not just directories)
- [x] Implement media file detection in browse
- [x] Implement ScanForMedia to find all importable files in path
- [x] Return parsed metadata for each found file

---

## Phase 8: Mass Rename Feature

### 8.1 Mass Rename Service

**File**: `internal/import/rename.go`

```go
func (s *Service) GetRenamePreview(ctx context.Context, mediaType string) ([]RenamePreview, error)
func (s *Service) ExecuteMassRename(ctx context.Context, mediaType string, items []int64) error
```

#### Tasks
- [x] Implement preview of files that would be renamed
- [x] Implement mass rename execution
- [x] Handle errors gracefully (continue on individual failures)
- [x] Update history with rename operations
- [ ] Prompt user with confirmation dialog when pattern settings change
- [x] Only affects library files, not download folder files

### 8.2 Specials Detection

**File**: `internal/import/specials.go`

```go
func (s *Service) IsSpecialEpisode(ctx context.Context, seriesID int64, filename string) (bool, error)
func (s *Service) DetectSpecialFromMetadata(episode *Episode) bool
func (s *Service) DetectSpecialFromFilename(filename string) bool
```

#### Tasks
- [x] Detect specials from TVDB/TMDB metadata (primary source, Season 0)
- [x] Detect specials from filename patterns: "Special", "SP01", "S00E01"
- [x] Metadata takes precedence when available

### 8.3 Mass Rename API

```go
// GET /api/v1/import/rename/preview?type=series
// POST /api/v1/import/rename/execute
```

#### Tasks
- [x] Implement rename preview endpoint
- [x] Implement rename execution endpoint
- [x] Add progress tracking via WebSocket

---

## Phase 9: Frontend (Overview)

### 9.1 Import Settings Page

**Location**: `web/src/routes/settings/import.tsx`

#### Tasks
- [x] Create import settings page layout
- [x] Implement validation level selector
- [x] Implement file extension management
- [x] Implement naming pattern editor with live preview (real-time preview as user types)
- [x] Show validation errors inline in pattern editor
- [x] Implement colon replacement selector
- [x] Implement multi-episode style selector
- [x] Add token reference documentation
- [x] Implement debug/test mode with detailed token breakdown
- [x] Show each token's resolved value step-by-step
- [x] Note: Renaming settings are global only - no per-series overrides for patterns

### 9.2 Manual Import UI

**Location**: `web/src/routes/import/index.tsx`

#### Tasks
- [x] Create file browser component
- [x] Create match suggestion display
- [x] Create match confirmation dialog
- [ ] Implement import progress tracking
- [ ] Add import history view

### 9.3 Queue Integration

**Location**: `web/src/pages/queue/`

#### Tasks
- [ ] Add import status to queue items
- [ ] Show file-level status for season packs
- [ ] Add retry button for failed imports
- [ ] Add manual import trigger from queue

---

## Phase 10: Testing

### 10.1 Unit Tests

#### Tasks
- [ ] Test token parsing and resolution
- [ ] Test character replacement logic
- [ ] Test multi-episode formatting
- [ ] Test filename parsing edge cases
- [ ] Test validation levels
- [ ] Test sample file detection
- [ ] Test path length validation
- [ ] Test empty token cleanup

### 10.2 Integration Tests

#### Tasks
- [ ] Test import pipeline end-to-end
- [ ] Test queue-based matching
- [ ] Test parse-based matching
- [ ] Test hardlink creation
- [ ] Test file move operations
- [ ] Test history logging
- [ ] Test health system integration

### 10.3 Manual Testing Scenarios

#### Tasks
- [ ] Test single movie import
- [ ] Test single episode import
- [ ] Test season pack import (split into episodes)
- [ ] Test upgrade scenario (replace existing file)
- [ ] Test failed import and retry
- [ ] Test manual import workflow
- [ ] Test mass rename feature
- [ ] Test naming pattern preview

---

## Implementation Order

### Recommended Sequence

1. **Phase 1** (Database) - Foundation for all features
2. **Phase 3.1-3.5** (Renaming Core) - Token engine and character handling
3. **Phase 4** (MediaInfo) - Required for token resolution
4. **Phase 2** (Import Service) - Core orchestration
5. **Phase 5** (Triggers) - Automation
6. **Phase 6** (Health/History) - Observability
7. **Phase 7** (API) - External interface
8. **Phase 8** (Mass Rename) - Post-import tooling
9. **Phase 9** (Frontend) - User interface
10. **Phase 10** (Testing) - Validation

### Dependencies

```
Phase 1 (DB)
    ↓
Phase 3.1-3.5 (Renaming) ←── Phase 4 (MediaInfo)
    ↓
Phase 2 (Import Service)
    ↓
Phase 5 (Triggers)
    ↓
Phase 6 (Health/History)
    ↓
Phase 7 (API)
    ↓
Phase 8 (Mass Rename)
    ↓
Phase 9 (Frontend)
    ↓
Phase 10 (Testing)
```

---

## Effort Estimates (Lines of Code)

| Phase | Files | Est. Lines |
|-------|-------|------------|
| Phase 1: Database | 4 | ~300 |
| Phase 2: Import Service | 5 | ~800 |
| Phase 3: Renaming | 6 | ~1200 |
| Phase 4: MediaInfo | 2 | ~400 |
| Phase 5: Triggers | 3 | ~300 |
| Phase 6: Health/History | 2 | ~200 |
| Phase 7: API | 3 | ~500 |
| Phase 8: Mass Rename | 2 | ~300 |
| Phase 9: Frontend | 10+ | ~2000 |
| Phase 10: Testing | 5+ | ~1000 |
| **Total** | **~40** | **~7000** |

---

## Risk Areas

1. **MediaInfo Integration**: External dependency, may need fallback strategy
2. **Hardlink Compatibility**: Cross-filesystem and OS-specific behavior
3. **File Locking**: Detecting when torrent clients release files
4. **Path Length**: Windows MAX_PATH edge cases
5. **Season Pack Parsing**: Complex multi-episode detection

---

## Requirements Coverage Audit

This section verifies every requirement from the spec is covered in the plan.

### Import Entry Point Section

| Requirement | Spec Line | Plan Coverage |
|-------------|-----------|---------------|
| Separate mapping table links queue items to library | Line 14 | Phase 1.1 queue_media table |
| Queue creation flow: Send to client → Create record | Line 15 | Phase 1.1 task added |
| Junction table with file_status per episode/movie | Line 16 | Phase 1.1 queue_media |
| Individual episode status in season packs | Lines 18-21 | Phase 1.1 queue_media, Phase 2.2 task |
| Filesystem watcher as primary trigger | Lines 25-28 | Phase 5.1 |
| Watcher only active when queue has items | Line 27 | Phase 5.1 conditional activation |
| Full scan on startup | Lines 30-33 | Phase 5.2 startup scan |
| Download client API status check (priority 1) | Line 40 | Phase 2.4 |
| File size stability 15-30 seconds (priority 2) | Line 41 | Phase 2.4 |
| File lock detection (priority 3) | Line 42 | Phase 2.4 |
| Archive detection (.rar, .zip) | Lines 44-46 | Phase 2.4 |
| Validation levels: Basic, Standard, Full | Lines 51-55 | Phase 1.2 import_settings, Phase 2.3 |
| Minimum file size default 100 MB | Lines 57-59 | Phase 1.2 import_settings |
| Sample file detection (folder + filename) | Lines 62-66 | Phase 2.3 |
| Configurable video extensions whitelist | Lines 68-71 | Phase 1.2 import_settings |
| Queue-based matching | Lines 75-78 | Phase 2.5 |
| Match conflict handling (configurable) | Lines 80-83 | Phase 1.2, Phase 2.5 |
| Manual import with user confirmation | Lines 85-89 | Phase 7.1, Phase 9.2 |
| Unknown media toggle (ignore/auto-add) | Lines 91-96 | Phase 1.2, Phase 2.2 task |
| Sequential processing | Lines 100-103 | Phase 2.1 |
| Hardlinks for torrent seeding | Lines 105-109 | Phase 3.6 |
| Import delay per download client | Lines 111-114 | Phase 1.5 import_delay_seconds |
| Retry once, then fail (2 attempts total) | Lines 118-121 | Phase 2.6 |
| Partial failure handling | Lines 123-126 | Phase 2.6 |
| Health system warning for failures | Lines 128-131 | Phase 6.1 |
| Ignore stalled downloads | Lines 133-136 | Phase 2.2 task |
| Source cleanup configurable per client | Lines 138-143 | Phase 1.5 cleanup_mode |
| Debounce filesystem events | Line 147 | Phase 5.1 |
| Manual import from any path | Lines 154-162 | Phase 7.3, Phase 9.2 |

### TV Item Renaming Section

| Requirement | Spec Line | Plan Coverage |
|-------------|-----------|---------------|
| Rename Episodes toggle (default true) | Lines 172-175 | Phase 1.2 rename_episodes |
| Replace Illegal Characters toggle | Lines 179-183 | Phase 1.2, Phase 3.3 |
| Colon replacement modes (6 options) | Lines 185-192 | Phase 1.2, Phase 3.3 |
| Three episode format types | Lines 196-213 | Phase 1.2 *_episode_format fields |
| Folder format types | Lines 215-230 | Phase 1.2 folder format fields |
| Multi-episode styles (6 options) | Lines 232-242 | Phase 1.2, Phase 3.4 |
| All series tokens | Lines 248-255 | Phase 3.1 |
| Season tokens with padding | Lines 257-262 | Phase 3.1 |
| Episode tokens with padding | Lines 264-269 | Phase 3.1 |
| Air date tokens | Lines 271-276 | Phase 3.1 |
| Episode title tokens | Lines 278-283 | Phase 3.1 |
| Quality tokens | Lines 285-290 | Phase 3.1 |
| MediaInfo tokens (10 types) | Lines 292-305 | Phase 3.1, Phase 4 |
| Other tokens (Release Group, etc.) | Lines 307-315 | Phase 3.1 |
| Separator control modifiers | Lines 319-324 | Phase 3.2 |
| Case control modifiers | Lines 326-331 | Phase 3.2 |
| Truncation modifiers | Lines 333-337 | Phase 3.2 |
| Language filtering for MediaInfo | Lines 339-343 | Phase 3.2 |
| Multi-track language format config | Lines 345-349 | Phase 3.2 |
| Anime tokens (absolute, version) | Lines 351-360 | Phase 3.1 |
| Revision token | Lines 362-369 | Phase 3.1 |

### Behavioral Requirements Section

| Requirement | Spec Line | Plan Coverage |
|-------------|-----------|---------------|
| Settings scope global only | Lines 376-378 | Phase 9.1 note |
| Format type auto-detection + manual override | Lines 380-383 | Phase 1.6 series format_type |
| Empty token cleanup with separators | Lines 385-389 | Phase 3.5 |
| Multi-episode title concatenation | Lines 391-395 | Phase 3.1 task |
| Daily show title fallback to air date | Lines 397-400 | Phase 3.1 task |

### Import Pipeline Section

| Requirement | Spec Line | Plan Coverage |
|-------------|-----------|---------------|
| Rename timing: MediaInfo → Compute → Copy | Lines 406-410 | Phase 2.2, architecture |
| MediaInfo extraction failure fallback | Lines 412-416 | Phase 4.1 ProbeWithFallback |
| Hardlink behavior (original keeps name) | Lines 418-422 | Phase 3.6 note |
| File lock avoided by hardlink | Lines 424-427 | Phase 3.6 note |

### Validation and Errors Section

| Requirement | Spec Line | Plan Coverage |
|-------------|-----------|---------------|
| Strict pattern validation | Lines 433-437 | Phase 3.5 |
| Path length limits (MAX_PATH) | Lines 439-443 | Phase 3.5 |
| File collisions: always overwrite | Lines 445-449 | Phase 3.6 task |
| CleanTitle removes only illegal chars | Lines 451-454 | Phase 3.3 |

### File Operations Section

| Requirement | Spec Line | Plan Coverage |
|-------------|-----------|---------------|
| Preserve file extension exactly | Lines 460-462 | Phase 2.2 task |
| Extra files preserve original names | Lines 464-467 | Phase 3.6 task |
| Delete old file on upgrade | Lines 469-473 | Phase 3.6 task |
| Folder permissions inherit from parent | Lines 475-478 | Phase 3.6 task |
| Folder rename on pattern change | Lines 480-484 | Phase 3.6 task |

### Other Sections

| Requirement | Spec Line | Plan Coverage |
|-------------|-----------|---------------|
| Season pack split into episodes | Lines 489-493 | Phase 2.2 task |
| Specials detection (metadata + filename) | Lines 497-502 | Phase 8.2 |
| Mass rename with confirmation | Lines 506-512 | Phase 8.1 task |
| Live preview in UI | Lines 517-522 | Phase 9.1 |
| Debug/test mode with token breakdown | Lines 524-528 | Phase 9.1 tasks |
| Rename history as part of import history | Lines 533-536 | Phase 6.2 |
| Original name stored in database | Lines 538-542 | Phase 1.4 |
| Release group from filename only | Lines 546-550 | Phase 3.1 |
| Default patterns match Sonarr | Lines 554-561 | Phase 1.2 defaults |
| Video files only | Lines 567-569 | Implied by extension whitelist |
| No conditional tokens | Lines 571-574 | Phase 3.5 note |
| No per-token length limits | Lines 576-579 | Phase 3.5 note |

---

## Next Steps

1. Review and approve this plan
2. Begin Phase 1 (Database Schema)
3. Iterate through phases in recommended order
4. Regular check-ins to validate against spec
