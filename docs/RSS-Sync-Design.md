# RSS Sync Design Document

## Overview

RSS Sync is a background feature that periodically fetches recent releases from indexer RSS feeds and matches them against all missing and upgradable media (movies and episodes) in the library. This dramatically reduces indexer API hits compared to the current auto-search approach, which makes one search per wanted item per indexer.

**Key principle**: RSS Sync operates independently of indexer mode. It works with native (Cardigann) indexers, Prowlarr-managed indexers, and a new "Generic RSS Feed" indexer type.

## Motivation

With auto-search, a library of 50 missing items across 5 indexers requires ~250 API calls per search cycle. With RSS sync, the same 5 indexers require only 5-10 API calls every 15 minutes — a 95%+ reduction. RSS also provides near-real-time detection of new releases (within minutes of appearing on indexers) versus the hours-long auto-search interval.

## Architecture

### How It Fits Into SlipStream

```
┌─────────────────────────────────────────────────────┐
│                    Scheduler                         │
│  ┌─────────────────┐   ┌─────────────────────────┐  │
│  │  Auto Search     │   │  RSS Sync               │  │
│  │  (every N hours) │   │  (every N minutes)      │  │
│  └────────┬─────────┘   └────────┬────────────────┘  │
│           │                      │                    │
│           ▼                      ▼                    │
│  Per-item targeted        Fetch all recent from      │
│  searches                 each RSS-enabled indexer    │
│           │                      │                    │
│           ▼                      ▼                    │
│  ┌──────────────────────────────────────────────┐    │
│  │       Shared Decisioning Pipeline             │    │
│  │  Match to wanted item → Score quality →       │    │
│  │  Check upgrade → History check → Grab         │    │
│  └──────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
```

RSS Sync is a **parallel path** to Auto Search, not a replacement. Both use the same shared decisioning pipeline (scoring, quality profile, upgrade checks, and grab infrastructure). RSS catches releases as they appear; auto-search handles backfill and items RSS misses.

### Component Overview

```
internal/decisioning/         — NEW: Shared decisioning logic (extracted from autosearch)
  types.go                    — SearchableItem, MediaType, WantedIndex
  selection.go                — SelectBestRelease (release filtering + quality decisioning)
  collection.go               — CollectWantedItems (missing + upgradable items)

internal/rsssync/
  service.go                  — Core RSS sync orchestration
  matcher.go                  — Matching RSS releases to wanted items via WantedIndex
  feedfetcher.go              — Fetching feeds from different indexer types
  cache.go                    — Per-indexer last-seen release tracking
  settings.go                 — RSS sync settings (interval, enable/disable)
  handlers.go                 — API endpoints
  events.go                   — WebSocket event definitions
```

### Shared Decisioning Package (`internal/decisioning`)

Currently, release selection and quality decisioning logic lives inside `autosearch/service.go` (specifically `selectBestRelease()` and related helpers). This logic is equally needed by RSS sync. We extract it into a shared package that both consumers import.

#### What Gets Extracted

| Current Location | Extracted To | Notes |
|---|---|---|
| `autosearch/types.go` → `SearchableItem` | `decisioning/types.go` | Universal wanted-item representation |
| `autosearch/types.go` → `MediaType` | `decisioning/types.go` | Movie, episode, season, series |
| `autosearch/service.go` → `selectBestRelease()` | `decisioning/selection.go` → `SelectBestRelease()` | Core filtering + quality logic |
| `autosearch/scheduled.go` → `collectSearchableItems()` and sub-collectors | `decisioning/collection.go` → `CollectWantedItems()` | Gathers missing + upgradable items |

#### Key Interfaces

```go
package decisioning

// SearchableItem represents a wanted media item for release decisioning.
type SearchableItem struct {
    MediaType        MediaType
    MediaID          int64
    Title            string
    Year             int

    ImdbID string
    TmdbID int
    TvdbID int

    SeriesID      int64
    SeasonNumber  int
    EpisodeNumber int

    QualityProfileID int64

    // CRITICAL for upgrades: must be populated when item has existing files.
    // CurrentQualityID must be the HIGHEST quality across all file records.
    HasFile          bool
    CurrentQualityID int

    TargetSlotID *int64
}

// SelectBestRelease picks the best acceptable release from a scored, sorted list.
// Releases MUST be pre-sorted by score (highest first).
// Returns nil if no acceptable release is found.
func SelectBestRelease(
    releases []types.TorrentInfo,
    profile *quality.Profile,
    item SearchableItem,
) *types.TorrentInfo
```

#### What Stays in Autosearch

- `searchAndGrab()` orchestration (coordinates search → select → grab)
- `buildSearchCriteria()` (auto-search specific: converts SearchableItem to indexer query)
- Backoff/failure tracking (`shouldSkipItem`, `autosearch_status` table)
- Scheduler integration

#### Refactoring Approach

1. Move `SearchableItem` and `MediaType` to `decisioning/types.go`
2. Move `selectBestRelease()` to `decisioning/selection.go` as exported `SelectBestRelease()`
3. Move `collectSearchableItems()` and sub-collectors to `decisioning/collection.go` as `CollectWantedItems()`
4. Update `autosearch` to import from `decisioning`
5. RSS sync imports from `decisioning` directly

## Detailed Design

### Phase 1: Feed Fetching

The RSS sync task fetches recent releases from all RSS-enabled indexers. Each indexer is fetched independently, with pagination support and a hard limit of **1000 results per indexer** (matching Sonarr's `MaxNumResultsPerQuery`). The fetch strategy depends on the indexer type:

#### Native (Cardigann) Indexers
- The existing `Indexer.Search(ctx, SearchCriteria{})` with an empty/minimal query returns recent items (equivalent to Torznab `?t=search`)
- For indexers that support both movies and TV, a single fetch returns mixed results
- For indexers that only support one type, the fetcher notes this and only matches against the relevant media type
- Support paginated feeds: follow `IndexerPageableRequestChain` pattern until reaching the last-seen release or the 1000-item hard limit

#### Prowlarr-Managed Indexers
- Make two calls: `prowlarr.Service.Search(ctx, SearchCriteria{Type: "movie"})` and `prowlarr.Service.Search(ctx, SearchCriteria{Type: "tvsearch"})` to ensure broad coverage
- **Important**: RSS sync must bypass Prowlarr's 5-minute search result cache to avoid stale results when auto-search ran recently with similar parameters
- The existing Torznab XML parsing (`TorznabItem.ToTorrentInfo()`) already handles result conversion

#### Generic RSS Feed (New Indexer Type)
- A new indexer type for plain RSS feeds with no Torznab/Newznab API
- Settings: `URL`, `Cookie` (optional), `ContentType` (movies/tv/both)
- No search capability — RSS-only
- **Auto-detect feed format** (mirroring Sonarr's `TorrentRssParserFactory`): standard RSS, TorrentPotato, EzRSS, and custom parsing based on feed structure
- This is the SlipStream equivalent of Sonarr's "Torrent RSS Feed" indexer type

### Phase 2: Matching Releases to Wanted Items

After fetching, the sync service builds a **wanted items index** and matches each RSS release against it. This is the inverse of auto-search: instead of "for each wanted item, search indexers", it's "for each RSS release, find matching wanted items."

The decisioning logic applied to matched releases is identical to auto-search. The only difference is that RSS sync performs client-side matching (release → item) rather than relying on indexers to filter via search terms. Once a release is matched to a wanted item, the same quality checks, upgrade logic, scoring, and selection apply.

#### Building the Wanted Items Index

Use the shared `decisioning.CollectWantedItems()` to gather all missing and upgradable movies and episodes. This already filters to only monitored items with non-unreleased status. Build lookup structures for fast matching:

```go
type WantedIndex struct {
    // Title-based lookup (normalized title → items)
    byTitle map[string][]decisioning.SearchableItem

    // External ID lookup
    byImdbID map[string][]decisioning.SearchableItem
    byTmdbID map[int][]decisioning.SearchableItem
    byTvdbID map[int][]decisioning.SearchableItem
}
```

#### Matching Algorithm

For each RSS release:
1. **Parse the release title** using `scanner.ParseFilename()` to extract: title, year, season, episode, quality, source
2. **Attempt ID-based match first** — if the Torznab item includes `imdbid`, `tmdbid`, or `tvdbid` attributes, look up directly in the index
3. **Fall back to title matching** — normalize the parsed title and search the title index using the existing `search.TitlesMatch()` logic
4. **Verify specifics** — for TV, confirm the season/episode numbers match a wanted episode or season pack (see Season Pack Handling below)

#### Unreleased Episode Filtering

`CollectWantedItems()` only returns episodes with actionable statuses (missing, upgradable). Episodes with unreleased status are **never** included in the WantedIndex. This means:

- An RSS release for an unreleased episode (e.g., a leak) will not match any wanted item and is silently ignored
- Auto-search will never search for unreleased episodes
- No special filtering is needed in the matching algorithm itself — the exclusion happens at index construction time

However, season pack eligibility requires knowledge of unreleased episodes even though they're not in the index (see below).

#### Season Pack Handling

When a release is identified as a season pack (via `scanner.ParseFilename().IsSeasonPack`), determining eligibility requires querying the **full season** from the database — not just the WantedIndex — because unreleased episodes are excluded from the index but must block season pack grabs.

**Eligibility check:**

1. Query ALL episodes for the series + season from the database (all statuses)
2. If any episode has **unreleased** status → **reject season pack** (would include content not yet available)
3. Determine if this is a missing or upgrade scenario:
   - **Missing**: ALL episodes in the season must have status = missing. If some are available/upgradable and others missing → **reject** (partial overlap; individual episode search handles this better)
   - **Upgrade**: ALL episodes must have files (upgradable or available status). Match only if the pack upgrades the highest `CurrentQualityID` across all episode files in the season. If some episodes are at/above cutoff and others below → **reject** (mixed upgrade state)
4. If eligible, create a synthetic `SearchableItem{MediaType: "season"}` for scoring

**Eligibility matrix** (for a 10-episode season):

| Episode Statuses | Season Pack? | Individual Eps? |
|---|---|---|
| 10 missing | Yes (missing) | Yes |
| 7 missing + 3 unreleased | **No** (unreleased present) | Yes (7 missing only) |
| 5 missing + 5 available | **No** (partial) | Yes (5 missing only) |
| 3 missing + 4 available + 3 unreleased | **No** (partial + unreleased) | Yes (3 missing only) |
| 10 upgradable | Yes (upgrade) | Yes |
| 7 upgradable + 3 unreleased | **No** (unreleased present) | Yes (7 upgradable only) |
| 7 upgradable + 3 at cutoff | **No** (not all upgradable) | Yes (7 upgradable only) |
| 5 missing + 5 upgradable | **No** (mixed missing/upgrade) | Yes (all 10) |
| 10 unreleased | **No** | **No** (none in index) |
| 1 missing + 9 unreleased | **No** (unreleased present) | Yes (1 missing only) |

**Season pack vs individual episode conflict:**

When an individual episode release matches, skip it if a season pack for the same series+season was already matched and scored higher in this sync cycle. This prevents grabbing both a season pack and individual episodes from the same feed.

#### Group by Wanted Item, Then Score and Select

After matching, group all matched releases by their target wanted item. For each group:

1. **Score each release** using `scoring.NewDefaultScorer()` with a `ScoringContext` built from the wanted item's quality profile
2. **Sort by score** (highest first)
3. **Select best** using `decisioning.SelectBestRelease()` — applies `profile.IsAcceptable()`, `profile.IsUpgrade()` (when `HasFile` is true), and returns the first passing release
4. **History check** — before grabbing, verify the item was not recently grabbed (within the last 12 hours) by checking the history table. This prevents duplicate grabs from concurrent RSS sync and auto-search runs
5. If a release passes all checks, queue it for grabbing

### Phase 3: Grab Pipeline

#### Concurrent Grab Protection

Before grabbing, acquire a per-media-item grab lock (keyed by `mediaType + mediaID`). This prevents race conditions where RSS sync and auto-search both attempt to grab a release for the same item simultaneously. The lock is held for the duration of the grab attempt and released afterward.

```go
type GrabLock struct {
    mu    sync.Mutex
    locks map[string]struct{} // key: "movie:123" or "episode:456"
}
```

Both auto-search and RSS sync must use this shared lock.

#### Grab Execution

Matched and accepted releases are sent through the existing `grab.Service.Grab()` pipeline — identical to how auto-search grabs work. History logging, WebSocket notifications, download client integration, and portal request status tracking are all reused.

The `GrabRequest.Source` field is set to `"rss-sync"` to distinguish from `"auto-search"` and `"manual-search"` in history and logs.

### Phase 4: Cache and Deduplication

#### Per-Indexer Last-Seen Cache

Each indexer tracks the most recent release from its last RSS sync. The `indexer_status` table already has a `last_rss_sync` datetime column (migration 003). Add two more columns for cache boundary tracking:

```sql
ALTER TABLE indexer_status ADD COLUMN last_rss_release_url TEXT;
ALTER TABLE indexer_status ADD COLUMN last_rss_release_date DATETIME;
```

**Deduplication strategy** (following Sonarr's approach): use **publish date** as the primary boundary and **download URL** as the secondary identifier. GUID is not reliable — some feeds have unstable GUIDs or omit them entirely.

On each sync:
1. Fetch the feed (paginating if necessary)
2. Process releases from newest to oldest
3. Stop when reaching a release whose download URL matches `last_rss_release_url` AND whose publish date is ≤ `last_rss_release_date`
4. If the cache boundary is never reached (all 1000 items are newer, or first sync), process all items and log a gap warning
5. Update the cache with the newest release's URL and date from this sync

#### Re-Grab Prevention

Do not maintain an in-memory dedup cache. Instead, rely on the **history table** to prevent re-grabbing:
- Before grabbing any matched release, check if the target media item was grabbed within the last 12 hours
- This survives service restarts and provides a single source of truth shared with auto-search

#### Dedup Cache Timing

Only update the per-indexer last-seen cache (`last_rss_release_url` / `last_rss_release_date`) after the sync cycle completes successfully. If a release is eligible for grab but the grab fails, the release will be re-evaluated on the next sync (since the cache boundary hasn't advanced past it).

### Phase 5: RSS Sync Settings

#### Global Settings

Stored in the `settings` KV table (same pattern as `autosearch_settings`):

```go
type RssSyncSettings struct {
    Enabled       bool `json:"enabled"`        // Master enable/disable
    IntervalMin   int  `json:"intervalMin"`     // Sync interval in minutes (min: 10, default: 15)
}
```

#### Per-Indexer RSS Toggle

**Native indexers**: Add `rss_enabled` column to the `indexers` table (default: `true`). This is independent of the existing `enabled` and `auto_search_enabled` flags.

**Prowlarr indexers**: Add `rss_enabled` column to `prowlarr_indexer_settings` table (default: `true`).

**Generic RSS Feed indexers**: Always RSS-enabled (that's their only purpose).

### Phase 6: Generic RSS Feed Indexer Type

A new indexer type that can be added alongside existing Cardigann/Prowlarr indexers.

#### Database Schema

Uses the existing `indexers` table with `definition_id = 'generic-rss'`:

```json
// Settings JSON for generic-rss indexers
{
    "url": "https://example.com/rss",
    "cookie": "",
    "contentType": "both"  // "movies", "tv", or "both"
}
```

#### Capabilities

- `SupportsRSS`: true
- `SupportsSearch`: false
- No search methods — only participates in RSS sync
- Not used by auto-search or interactive search

#### Feed Parsing

Auto-detect feed format (mirroring Sonarr's `TorrentRssParserFactory`). Supported formats:
- **Standard RSS** — `<item>` with `<title>`, `<link>`/`<enclosure>`, `<pubDate>`, `<guid>`
- **TorrentPotato** — JSON format used by some private trackers
- **EzRSS** — namespace-extended RSS with torrent-specific elements
- **Custom** — fallback heuristic parsing based on feed structure

Extract from each item:
- Title — release name (parsed for quality/season/episode info)
- Download URL — torrent file or magnet link
- Publish date — for cache ordering
- GUID / download URL — for deduplication
- Size — file size (optional; some feeds omit this)

### Phase 7: Scheduled Task Integration

Register with the existing `scheduler.Scheduler`:

```go
scheduler.RegisterTask(scheduler.TaskConfig{
    ID:          "rss-sync",
    Name:        "RSS Sync",
    Description: "Fetch recent releases from RSS feeds and grab matching items",
    Cron:        "*/15 * * * *",  // Every 15 minutes (configurable)
    Func:        rssSyncService.Run,
    RunOnStart:  true,
})
```

### Phase 8: API Endpoints

```
GET    /api/v1/settings/rsssync          — Get RSS sync settings
PUT    /api/v1/settings/rsssync          — Update RSS sync settings
POST   /api/v1/rsssync/trigger           — Manually trigger RSS sync
GET    /api/v1/rsssync/status            — Get last sync status/stats
```

Generic RSS Feed indexer CRUD uses the existing `/api/v1/indexers` endpoints (with `definitionId: "generic-rss"`).

### Phase 9: WebSocket Events

```
rss-sync:started    — { indexerCount: int }
rss-sync:progress   — { indexer: string, releasesFound: int, matched: int }
rss-sync:completed  — { totalReleases: int, matched: int, grabbed: int, elapsed: int }
rss-sync:failed     — { error: string }
```

## Implementation Order

1. **Shared decisioning package** — extract `SearchableItem`, `SelectBestRelease()`, `CollectWantedItems()` from autosearch; update autosearch to import from shared package
2. **Grab lock** — implement per-media-item `GrabLock`, integrate into both auto-search and `grab.Service`
3. **RSS Sync Settings** — settings KV entry, API endpoints, scheduler task registration
4. **Per-indexer RSS toggle** — DB migration, add `rss_enabled` to native and Prowlarr indexer settings; add `last_rss_release_url` and `last_rss_release_date` to `indexer_status`
5. **Feed Fetcher** — abstract fetching from native, Prowlarr, and generic-rss indexers; pagination support; Prowlarr cache bypass
6. **Wanted Items Index + Matcher** — build `WantedIndex` from `CollectWantedItems()`, match RSS releases via ID and title lookup
7. **Cache** — per-indexer last-seen tracking with publish date + download URL boundary
8. **Generic RSS Feed indexer type** — new definition, auto-detect parser, UI for adding/editing
9. **RSS Sync orchestration** — wire it all together: fetch → match → group by item → score → select → history check → grab
10. **WebSocket events + UI** — sync status display, settings UI
11. **Frontend** — settings page, sync status/history, generic RSS feed indexer form

## Interaction with Auto Search

RSS Sync and Auto Search are complementary. They share the same decisioning pipeline (`decisioning.SelectBestRelease()`), scoring engine, quality profiles, and grab infrastructure. The only difference is the source of releases: RSS sync fetches recent releases from feeds and matches client-side, while auto-search makes targeted per-item queries to indexers.

| Aspect | RSS Sync | Auto Search |
|--------|----------|-------------|
| Direction | Release → wanted item | Wanted item → search |
| API cost | 1 call per indexer per cycle | 1 call per item per indexer |
| Frequency | Every 15 min | Every N hours |
| Coverage | Only recent releases | Full catalog (backfill) |
| Best for | New releases, popular content | Old/rare content, backfill |
| Decisioning | Shared pipeline | Shared pipeline |

Both can run simultaneously. The shared grab lock and history-based dedup prevent double-grabbing. Auto-search's existing history check naturally skips items that RSS sync recently grabbed.

## Edge Cases

- **First sync**: No cache exists. Process all items from the feed and grab normally (same decisioning logic as subsequent syncs). The cache boundary is established from the newest release in this first fetch.
- **Indexer temporarily down**: Skip it, retry next cycle. Don't invalidate the cache. If the indexer is down for multiple cycles, releases may scroll off the feed — auto-search serves as the backfill mechanism.
- **High-volume indexer**: If releases scroll off the feed faster than the sync interval (cache boundary not reached within 1000 items), log a gap warning. User should increase sync frequency or the indexer isn't suitable for RSS.
- **Mixed content feeds**: A single RSS feed may contain movies, TV, music, etc. Only match against the media types tracked in SlipStream; ignore unrecognized items silently.
- **Duplicate grabs**: Before grabbing, check the history table for recent grabs (12h window) for the same media item. The per-media-item grab lock prevents concurrent grabs from RSS sync and auto-search.
- **Season pack vs episode conflict**: When both a season pack and individual episodes from the same season appear in a feed, prefer the season pack if all episodes are missing. Skip individual episode grabs for episodes covered by a higher-scoring season pack match.
- **Grab failure**: Failed grabs do not advance the per-indexer cache boundary. The release will be re-evaluated on the next sync cycle. History is only written on successful grabs.

## Out of Scope (V1)

The following are explicitly deferred to future iterations:

- **Proper/repack handling** — requires quality profile infrastructure changes
- **Multi-episode releases** — e.g., "S01E05E06" matching multiple episodes
- **Daily shows** — date-based episode matching (e.g., "The Daily Show 2024.02.09")
- **Anime absolute numbering** — e.g., "One Piece 1085" without season/episode
- **Release size validation** — rejecting undersized/oversized releases
- **Indexer tag filtering** — filtering indexers by series/movie tags

## Testing

Tests validate the shared decisioning pipeline from both entry points — auto-search and RSS sync — using concrete scenarios. Each scenario is tested twice: once through the auto-search path (targeted indexer query → SelectBestRelease) and once through the RSS sync path (broad feed → WantedIndex matching → grouping → SelectBestRelease). Use `testutil.NewTestDB(t)` for isolated database state.

### Shared Test Fixtures

**Profile "HD-1080p"** (ID: 5):
- Allowed: HDTV-720p (4), WEBRip-720p (5), WEBDL-720p (6), Bluray-720p (7), HDTV-1080p (8), WEBRip-1080p (9), WEBDL-1080p (10), Bluray-1080p (11)
- Not allowed: all 480p, all 2160p, all Remux
- Cutoff: Bluray-1080p (11)
- Upgrades enabled, strategy: balanced

**Library state** (all items use profile HD-1080p):

| Media | Status | File Quality | TMDB/TVDB |
|---|---|---|---|
| Dune: Part Two (2024) | missing | — | tmdb:693134 |
| Inception (2010) | upgradable | WEBDL-720p (6) | tmdb:27205 |
| Breaking Bad (ended, tvdb:81189) S3 | *varies per scenario* | *varies* | tvdb:81189 |
| Stranger Things (continuing, tvdb:305288) S4 | *varies per scenario* | *varies* | tvdb:305288 |
| The Mandalorian (continuing, tvdb:361753) S3 | *varies per scenario* | *varies* | tvdb:361753 |
| House of the Dragon (continuing, tvdb:371572) S3 | *varies per scenario* | *varies* | tvdb:371572 |
| The Last of Us (tvdb:100088) S2 | *varies per scenario* | *varies* | tvdb:100088 |

### Standard RSS Feed Noise

Every RSS sync test includes these noise releases in the feed alongside scenario-specific releases. All should result in **no match** against the WantedIndex:

| # | Title | External ID | Why No Match |
|---|---|---|---|
| N1 | `Ubuntu.24.04.LTS.Desktop.AMD64` | — | Not parseable as media |
| N2 | `The.Holdovers.2023.1080p.WEB-DL` | tmdb:840430 | Movie not in library |
| N3 | `Taylor.Swift.Eras.Tour.2024.1080p` | — | Not in library, no ID |
| N4 | `Shogun.S01E05.1080p.WEB-DL` | tvdb:392369 | TV show not in library |
| N5 | `Game.of.Thrones.S08E06.2160p.UHD.BluRay` | tvdb:121361 | TV show not in library |
| N6 | `Oppenheimer.2023.HDCAM` | tmdb:872585 | Movie not in library |
| N7 | `How.to.Train.Your.Dragon.2025.1080p.WEB-DL` | tmdb:11999 | Movie not in library |

Tests must assert that all 7 noise releases produce no match.

---

### Scenario 1: Missing Movie

**Setup**: Dune: Part Two (2024), status=missing, tmdb=693134.

#### 1A: Auto Search

**SearchableItem**: `{MediaType: "movie", Title: "Dune: Part Two", Year: 2024, TmdbID: 693134, HasFile: false}`

**Indexer returns** (sorted by score):

| Title | Quality ID | Score |
|---|---|---|
| `Dune.Part.Two.2024.1080p.BluRay` | 11 (Bluray-1080p) | 91 |
| `Dune.Part.Two.2024.1080p.WEB-DL` | 10 (WEBDL-1080p) | 85 |
| `Dune.Part.Two.2024.2160p.UHD.BluRay` | 16 (Bluray-2160p) | 72 |
| `Dune.Part.Two.2024.720p.BluRay` | 7 (Bluray-720p) | 68 |
| `Dune.2021.1080p.BluRay` | 11 (Bluray-1080p) | 40 |
| `Dune.Part.Two.2024.HDCAM` | 0 (unknown) | 12 |

**Assert**: `SelectBestRelease()` returns `Dune.Part.Two.2024.1080p.BluRay` (score 91). Bluray-1080p acceptable, no file so no upgrade check. The 2160p release would fail `IsAcceptable(16)`.

#### 1B: RSS Sync

**WantedIndex**: `byTmdbID[693134]` → [Dune: Part Two item]

**RSS feed** = standard noise + scenario releases:

| Title | tmdbid | Parsed |
|---|---|---|
| `Dune.Part.Two.2024.1080p.BluRay` | 693134 | "Dune Part Two", Bluray-1080p |
| `Dune.Part.Two.2024.2160p.UHD.BluRay` | 693134 | "Dune Part Two", Bluray-2160p |
| `Dune.Part.Two.2024.1080p.WEB-DL` | 693134 | "Dune Part Two", WEBDL-1080p |
| `Dune.2021.1080p.BluRay` | tmdb:438631 | "Dune", Bluray-1080p |
| `Dune.Part.Two.2024.HDCAM` | 693134 | "Dune Part Two", unknown |

**Assert matching**:
- Noise N1-N7 → no match
- `Dune.Part.Two.2024.1080p.BluRay` → byTmdbID[693134] → **match Dune: Part Two**
- `Dune.Part.Two.2024.2160p.UHD.BluRay` → byTmdbID[693134] → **match Dune: Part Two**
- `Dune.Part.Two.2024.1080p.WEB-DL` → byTmdbID[693134] → **match Dune: Part Two**
- `Dune.2021.1080p.BluRay` → byTmdbID[438631] → **no match** (wrong movie, different TMDB ID)
- `Dune.Part.Two.2024.HDCAM` → byTmdbID[693134] → **match Dune: Part Two**

**Assert grouping**: 1 group — Dune: Part Two with 4 releases.

**Assert selection**: Score all 4 against HD-1080p profile, sort by score. `Dune.Part.Two.2024.1080p.BluRay` selected (highest-scored acceptable release). 2160p fails `IsAcceptable(16)`. HDCAM (quality ID 0) passes quality check vacuously but scores lowest.

---

### Scenario 2: Upgradable Movie

**Setup**: Inception (2010), status=upgradable, tmdb=27205. File: WEBDL-720p (quality ID 6).

#### 2A: Auto Search

**SearchableItem**: `{MediaType: "movie", Title: "Inception", Year: 2010, TmdbID: 27205, HasFile: true, CurrentQualityID: 6}`

**Indexer returns**:

| Title | Quality ID | Score |
|---|---|---|
| `Inception.2010.1080p.BluRay` | 11 (Bluray-1080p) | 90 |
| `Inception.2010.1080p.WEB-DL` | 10 (WEBDL-1080p) | 83 |
| `Inception.2010.720p.WEB-DL` | 6 (WEBDL-720p) | 65 |
| `Inception.2010.720p.HDTV` | 4 (HDTV-720p) | 55 |

**Assert**: `SelectBestRelease()` returns `Inception.2010.1080p.BluRay`. `IsUpgrade(6, 11)` = true (720→1080). WEBDL-720p (ID 6) fails `IsUpgrade(6, 6)`. HDTV-720p (ID 4) fails `IsUpgrade(6, 4)`.

#### 2B: RSS Sync

**WantedIndex**: `byTmdbID[27205]` → [Inception item (HasFile: true, CurrentQualityID: 6)]

**RSS feed** = standard noise + scenario releases:

| Title | tmdbid | Parsed |
|---|---|---|
| `Inception.2010.1080p.BluRay` | 27205 | "Inception", Bluray-1080p |
| `Inception.2010.720p.WEB-DL` | 27205 | "Inception", WEBDL-720p |
| `Inception.2010.720p.HDTV` | 27205 | "Inception", HDTV-720p |
| `Interception.2024.1080p.WEB-DL` | tmdb:999888 | "Interception" |

**Assert matching**:
- Noise N1-N7 → no match
- `Inception.2010.1080p.BluRay` → byTmdbID[27205] → **match**
- `Inception.2010.720p.WEB-DL` → byTmdbID[27205] → **match**
- `Inception.2010.720p.HDTV` → byTmdbID[27205] → **match**
- `Interception.2024.1080p.WEB-DL` → byTmdbID[999888] → **no match** (different movie)

**Assert grouping**: 1 group — Inception with 3 releases.

**Assert selection**: `Inception.2010.1080p.BluRay` selected. `IsUpgrade(6, 11)` = true. The two 720p releases fail `IsUpgrade()` (same or lower quality).

---

### Scenario 3: Upgradable Movie, No Upgrade Available

**Setup**: Same as Scenario 2 (Inception, file=WEBDL-720p ID 6).

#### 3A: Auto Search

**Indexer returns** (only same-or-lower quality):

| Title | Quality ID | Score |
|---|---|---|
| `Inception.2010.720p.WEB-DL` | 6 (WEBDL-720p) | 65 |
| `Inception.2010.720p.HDTV` | 4 (HDTV-720p) | 55 |

**Assert**: `SelectBestRelease()` returns `nil`. Both fail `IsUpgrade()`.

#### 3B: RSS Sync

**RSS feed** = standard noise + scenario releases:

| Title | tmdbid | Parsed |
|---|---|---|
| `Inception.2010.720p.WEB-DL` | 27205 | "Inception", WEBDL-720p |
| `Inception.2010.720p.HDTV` | 27205 | "Inception", HDTV-720p |

**Assert matching**: Both match Inception via tmdbid. Noise → no match.

**Assert grouping**: 1 group — Inception with 2 releases.

**Assert selection**: `nil`. Both fail `IsUpgrade(6, 6)` and `IsUpgrade(6, 4)`. No grab queued.

---

### Scenario 4: Missing Episode

**Setup**: Breaking Bad S3E07 status=missing. The rest of S3 varies — for this scenario, assume a mix (some available, some missing) so no season pack is eligible.

#### 4A: Auto Search

**SearchableItem**: `{MediaType: "episode", Title: "Breaking Bad", SeriesID: 42, SeasonNumber: 3, EpisodeNumber: 7, TvdbID: 81189, HasFile: false}`

**Indexer returns**:

| Title | Quality ID | Score |
|---|---|---|
| `Breaking.Bad.S03.1080p.BluRay` | 11 (Bluray-1080p) | 88 |
| `Breaking.Bad.S03E07.1080p.WEB-DL` | 10 (WEBDL-1080p) | 85 |
| `Breaking.Bad.S03E08.1080p.WEB-DL` | 10 (WEBDL-1080p) | 84 |
| `Breaking.Bad.S03E07.720p.BluRay` | 7 (Bluray-720p) | 70 |

**Assert**: `SelectBestRelease()` returns `Breaking.Bad.S03E07.1080p.WEB-DL` (score 85).

**Why**: Season pack (score 88) → `parsed.IsSeasonPack=true`, `parsed.Episode=0 != 7` → **rejected** (episode search requires exact match). S03E08 → `parsed.Episode=8 != 7` → **rejected**. S03E07 1080p WEB-DL → `parsed.Episode=7 == 7`, `IsAcceptable(10)` → yes. Selected.

#### 4B: RSS Sync

**WantedIndex**: `byTvdbID[81189]` → [Breaking Bad S3E07 item]

**RSS feed** = standard noise + scenario releases:

| Title | tvdbid | Parsed |
|---|---|---|
| `Breaking.Bad.S03.1080p.BluRay` | 81189 | Season pack S3 |
| `Breaking.Bad.S03E07.1080p.WEB-DL` | 81189 | S3E7, WEBDL-1080p |
| `Breaking.Bad.S03E08.1080p.WEB-DL` | 81189 | S3E8, WEBDL-1080p |
| `Breaking.Bad.S02E10.1080p.BluRay` | 81189 | S2E10, Bluray-1080p |
| `Breaking.Bad.S03E07.720p.BluRay` | 81189 | S3E7, Bluray-720p |

**Assert matching**:
- Noise N1-N7 → no match
- Season pack S03 → tvdb match. Season pack eligibility: not all S3 episodes missing → **rejected**
- S03E07 1080p → tvdb match, episode 7 matches S3E07 → **match**
- S03E08 → tvdb match, episode 8 not in index (only E07 is wanted) → **no match**
- S02E10 → tvdb match, season 2 doesn't match any S3 items → **no match**
- S03E07 720p → tvdb match, episode 7 matches S3E07 → **match**

**Assert grouping**: 1 group — S3E07 with 2 releases.

**Assert selection**: Score both against HD-1080p profile. `Breaking.Bad.S03E07.1080p.WEB-DL` selected (higher score). Both acceptable, no file so no upgrade check.

---

### Scenario 5: Upgradable Episode — Disc Source Upgrade

**Setup**: Stranger Things S4E05, status=upgradable, file=HDTV-1080p (ID 8). For this scenario, S4 has other episodes in various states that prevent season pack eligibility (e.g., E08-E09 unreleased).

#### 5A: Auto Search

**SearchableItem**: `{MediaType: "episode", SeasonNumber: 4, EpisodeNumber: 5, TvdbID: 305288, HasFile: true, CurrentQualityID: 8}`

**Indexer returns**:

| Title | Quality ID | Score |
|---|---|---|
| `Stranger.Things.S04E05.1080p.BluRay` | 11 (Bluray-1080p) | 89 |
| `Stranger.Things.S04E05.1080p.WEB-DL` | 10 (WEBDL-1080p) | 82 |
| `Stranger.Things.S04E05.1080p.HDTV` | 8 (HDTV-1080p) | 70 |

**Assert**: `SelectBestRelease()` returns `Stranger.Things.S04E05.1080p.BluRay`.

**Why**: `IsUpgrade(8, 11)`: HDTV-1080p vs Bluray-1080p. Same resolution (1080). Balanced strategy: `IsDiscSource("bluray") && !IsDiscSource("tv")` → true → **upgrade**. WEBDL-1080p (ID 10): `IsUpgrade(8, 10)`: `IsDiscSource("webdl")` → false → **not upgrade**. HDTV-1080p (ID 8): `IsUpgrade(8, 8)` → same quality → **not upgrade**.

#### 5B: RSS Sync

**WantedIndex**: `byTvdbID[305288]` → [S4E05 item (HasFile: true, CurrentQualityID: 8)]

**RSS feed** = standard noise + scenario releases:

| Title | tvdbid | Parsed |
|---|---|---|
| `Stranger.Things.S04.1080p.BluRay` | 305288 | Season pack S4 |
| `Stranger.Things.S04E05.1080p.BluRay` | 305288 | S4E5, Bluray-1080p |
| `Stranger.Things.S04E05.1080p.WEB-DL` | 305288 | S4E5, WEBDL-1080p |
| `Stranger.Things.S04E09.1080p.WEB-DL` | 305288 | S4E9 (unreleased) |
| `Stranger.Things.S03E05.1080p.BluRay` | 305288 | S3E5 (wrong season) |

**Assert matching**:
- Noise N1-N7 → no match
- Season pack S04 → tvdb match. Eligibility: E08-E09 unreleased → **rejected**
- S04E05 BluRay → tvdb match, episode 5 in index → **match**
- S04E05 WEB-DL → tvdb match, episode 5 in index → **match**
- S04E09 → tvdb match, episode 9 not in index (unreleased) → **no match**
- S03E05 → tvdb match, season 3 doesn't match S4 items → **no match**

**Assert grouping**: 1 group — S4E05 with 2 releases.

**Assert selection**: `Stranger.Things.S04E05.1080p.BluRay` selected. `IsUpgrade(8, 11)` → disc source upgrade. The WEB-DL would fail `IsUpgrade(8, 10)` → non-disc, same resolution.

---

### Scenario 6: Upgradable Episode — Non-Disc Upgrade Rejected

**Setup**: Same as Scenario 5 (S4E05, file=HDTV-1080p ID 8), but only non-disc releases available.

#### 6A: Auto Search

**Indexer returns**:

| Title | Quality ID | Score |
|---|---|---|
| `Stranger.Things.S04E05.1080p.WEB-DL` | 10 (WEBDL-1080p) | 82 |
| `Stranger.Things.S04E05.1080p.HDTV` | 8 (HDTV-1080p) | 70 |

**Assert**: `SelectBestRelease()` returns `nil`.

**Why**: WEBDL-1080p: `IsUpgrade(8, 10)` — same resolution, `IsDiscSource("webdl")` → false → not upgrade. HDTV-1080p: `IsUpgrade(8, 8)` → same quality → not upgrade.

#### 6B: RSS Sync

**RSS feed** = standard noise + only non-disc S4E05 releases (same as 6A titles, with tvdb:305288).

**Assert matching**: Both S4E05 releases match via tvdb + episode number.

**Assert grouping**: 1 group — S4E05 with 2 releases.

**Assert selection**: `nil`. Neither release is an upgrade under balanced strategy. No grab queued.

---

### Scenario 7: All Episodes Missing — Season Pack Eligible (Ended Series)

**Setup**: Breaking Bad S3, 13 episodes, show ended. ALL 13 episodes status=missing. None unreleased, none available.

#### 7A: Auto Search

**Assert `CollectWantedItems()`**: All 13 missing, none unreleased → season pack eligible. Creates `SearchableItem{MediaType: "season", SeasonNumber: 3, HasFile: false}` AND 13 individual episode items. Season-level item searched first.

**Indexer returns** (for season search "Breaking Bad S03"):

| Title | Quality ID | Score |
|---|---|---|
| `Breaking.Bad.S03.1080p.BluRay` | 11 (Bluray-1080p) | 90 |
| `Breaking.Bad.S03.720p.WEB-DL` | 6 (WEBDL-720p) | 65 |
| `Breaking.Bad.S03E01.1080p.WEB-DL` | 10 (WEBDL-1080p) | 84 |

**Assert**: `SelectBestRelease()` with season item returns `Breaking.Bad.S03.1080p.BluRay`.

**Why**: `MediaType == "season"` → requires `parsed.IsSeasonPack == true`. #1 passes. `IsAcceptable(11)` → yes. `HasFile == false` → no upgrade check. Selected. #3 (individual episode) would be rejected: `parsed.IsSeasonPack == false`.

#### 7B: RSS Sync

**WantedIndex**: `byTvdbID[81189]` → [S3E01, S3E02, ..., S3E13 items — all 13]

**RSS feed** = standard noise + scenario releases:

| Title | tvdbid | Parsed |
|---|---|---|
| `Breaking.Bad.S03.1080p.BluRay` | 81189 | Season pack S3, Bluray-1080p |
| `Breaking.Bad.S03E01.1080p.WEB-DL` | 81189 | S3E1, WEBDL-1080p |
| `Breaking.Bad.S03E02.1080p.WEB-DL` | 81189 | S3E2, WEBDL-1080p |
| `Breaking.Bad.S02.1080p.BluRay` | 81189 | Season pack S2 |
| `Breaking.Bad.S04E01.1080p.WEB-DL` | 81189 | S4E1 |

**Assert matching**:
- Noise N1-N7 → no match
- Season pack S03 → tvdb match. Eligibility: all 13 missing, none unreleased → **eligible, match as season**
- S03E01 → tvdb match, episode 1 in index → **match**
- S03E02 → tvdb match, episode 2 in index → **match**
- Season pack S02 → tvdb match. Eligibility: no S2 episodes in index (all S2 available) → **no match**
- S04E01 → tvdb match, season 4 doesn't match S3 items → **no match**

**Assert grouping**: 3 groups — S3 season pack (1 release), S3E01 (1 release), S3E02 (1 release).

**Assert selection**: Season pack selected (Bluray-1080p, acceptable, no file). **Season pack wins → suppress S3E01 and S3E02 individual grabs** (covered by pack).

---

### Scenario 8: All Episodes Upgradable — Season Pack Upgrade (Ended Series)

**Setup**: Breaking Bad S3, 13 episodes, show ended. ALL 13 have WEBDL-720p (ID 6) files, all status=upgradable. None unreleased.

#### 8A: Auto Search

**Assert `CollectWantedItems()`**: All 13 upgradable, none unreleased → season pack eligible. Creates `SearchableItem{MediaType: "season", SeasonNumber: 3, HasFile: true, CurrentQualityID: 6}`.

**Indexer returns**:

| Title | Quality ID | Score |
|---|---|---|
| `Breaking.Bad.S03.1080p.BluRay` | 11 (Bluray-1080p) | 90 |
| `Breaking.Bad.S03E01.1080p.WEB-DL` | 10 (WEBDL-1080p) | 84 |

**Assert**: `SelectBestRelease()` returns `Breaking.Bad.S03.1080p.BluRay`. `IsUpgrade(6, 11)` → 720→1080, upgrade. Individual episode rejected (`IsSeasonPack == false`).

#### 8B: RSS Sync

**WantedIndex**: `byTvdbID[81189]` → [S3E01-E13 items, all HasFile: true, CurrentQualityID: 6]

**RSS feed** = standard noise + scenario releases:

| Title | tvdbid | Parsed |
|---|---|---|
| `Breaking.Bad.S03.1080p.BluRay` | 81189 | Season pack S3, Bluray-1080p |
| `Breaking.Bad.S03.720p.WEB-DL` | 81189 | Season pack S3, WEBDL-720p |
| `Breaking.Bad.S03E05.1080p.WEB-DL` | 81189 | S3E5, WEBDL-1080p |

**Assert matching**:
- Season pack S03 Bluray → eligible (all 13 upgradable, none unreleased) → **match as season**
- Season pack S03 WEBDL-720p → eligible → **match as season**
- S03E05 → tvdb match, episode 5 in index → **match**

**Assert grouping**: 2 groups — S3 season (2 releases), S3E05 (1 release).

**Assert selection**: Season group: `Breaking.Bad.S03.1080p.BluRay` selected. `IsUpgrade(6, 11)` → upgrade. The 720p WEBDL pack would fail `IsUpgrade(6, 6)` → same quality. **Season pack wins → suppress S3E05 individual grab.**

---

### Scenario 9: Season Upgrade Rejected — Not All Upgradable

**Setup**: Breaking Bad S3, 13 episodes, show ended. E01-E10 have WEBDL-720p (ID 6) files, status=upgradable. E11-E13 have Bluray-1080p (ID 11) files, status=available (at cutoff).

#### 9A: Auto Search

**Assert `CollectWantedItems()`**: Not all episodes upgradable (E11-E13 at cutoff) → season pack NOT eligible. Returns 10 individual items (E01-E10).

**SearchableItem** (for E01): `{MediaType: "episode", EpisodeNumber: 1, HasFile: true, CurrentQualityID: 6}`

**Indexer returns** (for E01 search):

| Title | Quality ID | Score |
|---|---|---|
| `Breaking.Bad.S03.1080p.BluRay` | 11 (Bluray-1080p) | 90 |
| `Breaking.Bad.S03E01.1080p.BluRay` | 11 (Bluray-1080p) | 88 |

**Assert**: `SelectBestRelease()` returns `Breaking.Bad.S03E01.1080p.BluRay` (score 88).

**Why**: Season pack (score 90) → `parsed.IsSeasonPack == true`, `parsed.Episode == 0 != 1` → **rejected** (episode search requires exact match). Individual S03E01 → `IsUpgrade(6, 11)` → upgrade. Selected.

#### 9B: RSS Sync

**WantedIndex**: `byTvdbID[81189]` → [S3E01-E10 items, all HasFile: true, CurrentQualityID: 6]. E11-E13 NOT in index (at cutoff, not upgradable).

**RSS feed** = standard noise + scenario releases:

| Title | tvdbid | Parsed |
|---|---|---|
| `Breaking.Bad.S03.1080p.BluRay` | 81189 | Season pack S3 |
| `Breaking.Bad.S03E01.1080p.BluRay` | 81189 | S3E1, Bluray-1080p |
| `Breaking.Bad.S03E11.1080p.BluRay` | 81189 | S3E11, Bluray-1080p |

**Assert matching**:
- Season pack S03 → eligibility check: E11-E13 are available (at cutoff), not all upgradable → **rejected**
- S03E01 → tvdb match, episode 1 in index → **match**
- S03E11 → tvdb match, episode 11 NOT in index (at cutoff) → **no match**

**Assert grouping**: 1 group — S3E01 (1 release).

**Assert selection**: `Breaking.Bad.S03E01.1080p.BluRay` selected. `IsUpgrade(6, 11)` → upgrade.

---

### Scenario 10: Continuing Season — Missing + Unreleased

**Setup**: The Mandalorian S3, 8 episodes. E01-E05 status=missing. E06-E08 status=unreleased.

#### 10A: Auto Search

**Assert `CollectWantedItems()`**: Unreleased episodes present → no season-level item. Returns 5 individual items (E01-E05).

**SearchableItem** (for E03): `{MediaType: "episode", SeasonNumber: 3, EpisodeNumber: 3, TvdbID: 361753, HasFile: false}`

**Indexer returns**:

| Title | Quality ID | Score |
|---|---|---|
| `The.Mandalorian.S03.1080p.WEB-DL` | 10 (WEBDL-1080p) | 88 |
| `The.Mandalorian.S03E03.1080p.WEB-DL` | 10 (WEBDL-1080p) | 86 |
| `The.Mandalorian.S03E06.1080p.WEB-DL` | 10 (WEBDL-1080p) | 85 |

**Assert**: `SelectBestRelease()` returns `The.Mandalorian.S03E03.1080p.WEB-DL` (score 86).

**Why**: Season pack (score 88) → `parsed.Episode == 0 != 3` → rejected (episode search). S03E06 → `parsed.Episode == 6 != 3` → rejected. S03E03 → exact match. Selected.

#### 10B: RSS Sync

**WantedIndex**: `byTvdbID[361753]` → [E01, E02, E03, E04, E05 items]. E06-E08 excluded (unreleased).

**RSS feed** = standard noise + scenario releases:

| Title | tvdbid | Parsed |
|---|---|---|
| `The.Mandalorian.S03.1080p.WEB-DL` | 361753 | Season pack S3 |
| `The.Mandalorian.S03E03.1080p.WEB-DL` | 361753 | S3E3, WEBDL-1080p |
| `The.Mandalorian.S03E06.1080p.WEB-DL` | 361753 | S3E6 (leaked) |
| `The.Mandalorian.S03E04.720p.HDTV` | 361753 | S3E4, HDTV-720p |
| `The.Mandalorian.S02E08.1080p.BluRay` | 361753 | S2E8 (wrong season) |

**Assert matching**:
- Noise N1-N7 → no match
- Season pack S03 → eligibility: E06-E08 unreleased → **rejected**
- S03E03 → episode 3 in index → **match**
- S03E06 → episode 6 NOT in index (unreleased) → **no match** (leak ignored)
- S03E04 → episode 4 in index → **match**
- S02E08 → season 2 doesn't match S3 items → **no match**

**Assert grouping**: 2 groups — S3E03 (1 release), S3E04 (1 release).

**Assert selection**: S3E03 → `The.Mandalorian.S03E03.1080p.WEB-DL` selected. S3E04 → `The.Mandalorian.S03E04.720p.HDTV` selected. Both acceptable, no files.

---

### Scenario 11: Season Premiere — 1 Aired + Rest Unreleased

**Setup**: House of the Dragon S3, 10 episodes. S3E01 status=missing. S3E02-E10 status=unreleased.

#### 11A: Auto Search

**Assert `CollectWantedItems()`**: Returns 1 item (S3E01). No season-level item (unreleased present).

**SearchableItem**: `{MediaType: "episode", SeasonNumber: 3, EpisodeNumber: 1, HasFile: false}`

**Indexer returns**:

| Title | Quality ID | Score |
|---|---|---|
| `House.of.the.Dragon.S03.1080p.WEB-DL` | 10 (WEBDL-1080p) | 88 |
| `House.of.the.Dragon.S03E01.1080p.WEB-DL` | 10 (WEBDL-1080p) | 86 |
| `House.of.the.Dragon.S03E01.720p.HDTV` | 4 (HDTV-720p) | 62 |

**Assert**: `SelectBestRelease()` returns `House.of.the.Dragon.S03E01.1080p.WEB-DL` (score 86). Season pack rejected (episode 0 != 1).

#### 11B: RSS Sync

**WantedIndex**: `byTvdbID[371572]` → [S3E01 item only]

**RSS feed** = standard noise + scenario releases:

| Title | tvdbid | Parsed |
|---|---|---|
| `House.of.the.Dragon.S03.1080p.WEB-DL` | 371572 | Season pack S3 |
| `House.of.the.Dragon.S03E01.1080p.WEB-DL` | 371572 | S3E1, WEBDL-1080p |
| `House.of.the.Dragon.S03E01.720p.HDTV` | 371572 | S3E1, HDTV-720p |
| `House.of.the.Dragon.S03E02.1080p.WEB-DL` | 371572 | S3E2 (unreleased) |
| `House.of.the.Dragon.S02E10.1080p.BluRay` | 371572 | S2E10 (wrong season) |

**Assert matching**:
- Noise N1-N7 → no match
- Season pack S03 → eligibility: E02-E10 unreleased → **rejected**
- S03E01 WEB-DL → episode 1 in index → **match**
- S03E01 HDTV → episode 1 in index → **match**
- S03E02 → episode 2 NOT in index (unreleased) → **no match**
- S02E10 → season 2 doesn't match S3 items → **no match**

**Assert grouping**: 1 group — S3E01 with 2 releases.

**Assert selection**: `House.of.the.Dragon.S03E01.1080p.WEB-DL` selected (higher score). Both acceptable, no file.

---

### Scenario 12: Upgradable Season + Unreleased Episodes

**Setup**: Stranger Things S4, 9 episodes. E01-E07 status=upgradable, file=HDTV-1080p (ID 8). E08-E09 status=unreleased.

#### 12A: Auto Search

**Assert `CollectWantedItems()`**: E08-E09 unreleased → no season-level item. Returns 7 individual items (E01-E07, all HasFile: true, CurrentQualityID: 8).

**SearchableItem** (for E05): `{MediaType: "episode", SeasonNumber: 4, EpisodeNumber: 5, HasFile: true, CurrentQualityID: 8}`

**Indexer returns**:

| Title | Quality ID | Score |
|---|---|---|
| `Stranger.Things.S04E05.1080p.BluRay` | 11 (Bluray-1080p) | 89 |
| `Stranger.Things.S04E05.1080p.WEB-DL` | 10 (WEBDL-1080p) | 82 |

**Assert**: `SelectBestRelease()` returns `Stranger.Things.S04E05.1080p.BluRay`. `IsUpgrade(8, 11)` → disc source upgrade.

#### 12B: RSS Sync

**WantedIndex**: `byTvdbID[305288]` → [S4E01-E07 items, all HasFile: true, CurrentQualityID: 8]

**RSS feed** = standard noise + scenario releases:

| Title | tvdbid | Parsed |
|---|---|---|
| `Stranger.Things.S04.1080p.BluRay` | 305288 | Season pack S4 |
| `Stranger.Things.S04E05.1080p.BluRay` | 305288 | S4E5, Bluray-1080p |
| `Stranger.Things.S04E05.1080p.WEB-DL` | 305288 | S4E5, WEBDL-1080p |
| `Stranger.Things.S04E09.1080p.WEB-DL` | 305288 | S4E9 (unreleased) |
| `Stranger.Things.S03E05.1080p.BluRay` | 305288 | S3E5 (wrong season) |

**Assert matching**:
- Noise N1-N7 → no match
- Season pack S04 → eligibility: E08-E09 unreleased → **rejected**
- S04E05 BluRay → episode 5 in index → **match**
- S04E05 WEB-DL → episode 5 in index → **match**
- S04E09 → episode 9 NOT in index (unreleased) → **no match**
- S03E05 → season 3 doesn't match S4 items → **no match**

**Assert grouping**: 1 group — S4E05 with 2 releases.

**Assert selection**: `Stranger.Things.S04E05.1080p.BluRay` selected. `IsUpgrade(8, 11)` → disc source upgrade. WEB-DL fails `IsUpgrade(8, 10)`.

---

### Scenario 13: Season Transitions — Unreleased → Missing

Tests that season pack eligibility changes as episodes air. Two-phase test.

**Setup (Phase A)**: The Last of Us S2, 7 episodes. E01-E06 status=missing. E07 status=unreleased.

#### 13A: RSS Sync — Phase A (Unreleased Present)

**WantedIndex**: `byTvdbID[100088]` → [E01-E06 items]

**RSS feed** = standard noise + `The.Last.of.Us.S02.1080p.WEB-DL` (tvdb:100088, season pack).

**Assert**: Season pack eligibility → E07 unreleased → **rejected**. No season-level match. Individual episodes E01-E06 matchable if present in feed.

#### 13B: Auto Search — Phase A (Unreleased Present)

**Assert `CollectWantedItems()`**: Returns 6 individual items (E01-E06). No season-level item.

**SearchableItem** (for E01): `{MediaType: "episode", EpisodeNumber: 1, HasFile: false}`

**Indexer returns**: `The.Last.of.Us.S02.1080p.WEB-DL` (season pack) + `The.Last.of.Us.S02E01.1080p.WEB-DL`.

**Assert**: Season pack rejected (`parsed.Episode == 0 != 1`). S02E01 selected.

**Setup (Phase B)**: E07 metadata refresh → status changes from unreleased to missing. All 7 now missing.

#### 13C: RSS Sync — Phase B (All Missing)

**WantedIndex**: `byTvdbID[100088]` → [E01-E07 items — all 7]

**RSS feed** = standard noise + `The.Last.of.Us.S02.1080p.WEB-DL` (tvdb:100088, season pack).

**Assert**: Season pack eligibility → all 7 missing, none unreleased → **eligible**. Season pack matched and selected. Individual episode grabs suppressed.

#### 13D: Auto Search — Phase B (All Missing)

**Assert `CollectWantedItems()`**: All 7 missing, none unreleased → season pack eligible. Creates `SearchableItem{MediaType: "season", SeasonNumber: 2, HasFile: false}`.

**Indexer returns**: `The.Last.of.Us.S02.1080p.WEB-DL` (season pack).

**Assert**: `SelectBestRelease()` → `parsed.IsSeasonPack == true`, `IsAcceptable(10)` → yes. Selected.

---

### Scenario 14: Season Pack Blocked After Partial Grabs

**Setup**: The Last of Us S2, 7 episodes. E01 status=available (grabbed last week). E02-E07 status=missing. None unreleased.

#### 14A: Auto Search

**Assert `CollectWantedItems()`**: Not all episodes missing (E01 is available) → season pack NOT eligible. Returns 6 individual items (E02-E07).

**SearchableItem** (for E02): `{MediaType: "episode", EpisodeNumber: 2, HasFile: false}`

**Indexer returns**:

| Title | Quality ID | Score |
|---|---|---|
| `The.Last.of.Us.S02.1080p.WEB-DL` | 10 (WEBDL-1080p) | 88 |
| `The.Last.of.Us.S02E02.1080p.WEB-DL` | 10 (WEBDL-1080p) | 86 |

**Assert**: Season pack rejected (`parsed.Episode == 0 != 2`). `The.Last.of.Us.S02E02.1080p.WEB-DL` selected.

#### 14B: RSS Sync

**WantedIndex**: `byTvdbID[100088]` → [E02-E07 items]. E01 NOT in index (available).

**RSS feed** = standard noise + scenario releases:

| Title | tvdbid | Parsed |
|---|---|---|
| `The.Last.of.Us.S02.1080p.WEB-DL` | 100088 | Season pack S2 |
| `The.Last.of.Us.S02E02.1080p.WEB-DL` | 100088 | S2E2, WEBDL-1080p |
| `The.Last.of.Us.S02E01.1080p.BluRay` | 100088 | S2E1 (already available) |

**Assert matching**:
- Noise N1-N7 → no match
- Season pack S02 → eligibility: E01 is available, not all missing → **rejected**
- S02E02 → episode 2 in index → **match**
- S02E01 → episode 1 NOT in index (available, not wanted) → **no match**

**Assert grouping**: 1 group — S2E02 (1 release).

**Assert selection**: `The.Last.of.Us.S02E02.1080p.WEB-DL` selected. Acceptable, no file.

---

### Scenario 15: RSS Sync — Title-Based Matching (No External IDs)

Tests the title-matching fallback when Torznab items lack external IDs (e.g., generic RSS feeds).

**Setup**: Library contains Dune: Part Two (missing, tmdb=693134) and Breaking Bad S3E07 (missing, tvdb=81189).

**WantedIndex**:
- `byTmdbID[693134]` → [Dune: Part Two]
- `byTvdbID[81189]` → [S3E07]
- `byTitle["dune part two"]` → [Dune: Part Two]
- `byTitle["breaking bad"]` → [S3E07]

#### 15A: RSS Sync — Title Match Success

**RSS feed** = standard noise + releases **without external IDs** (generic RSS feed):

| Title | External IDs | Parsed Title |
|---|---|---|
| `Dune.Part.Two.2024.1080p.BluRay` | none | "Dune Part Two" |
| `Dune.1984.720p.BluRay` | none | "Dune" |
| `Breaking.Bad.S03E07.1080p.WEB-DL` | none | "Breaking Bad" S3E7 |

**Assert matching**:
- `Dune.Part.Two.2024.1080p.BluRay` → no IDs → title fallback: `TitlesMatch("dune part two", "dune part two")` → **match**
- `Dune.1984.720p.BluRay` → no IDs → title fallback: `TitlesMatch("dune", "dune part two")` → **no match** (strict equality after normalization)
- `Breaking.Bad.S03E07` → no IDs → title fallback: `TitlesMatch("breaking bad", "breaking bad")` → match candidates. Verify: S3E7 matches S3E07 → **match**

#### 15B: RSS Sync — Title Match with Ambiguous Names

**RSS feed** = standard noise + releases without external IDs:

| Title | External IDs | Parsed Title |
|---|---|---|
| `Breaking.Bad.S03E08.1080p.WEB-DL` | none | "Breaking Bad" S3E8 |
| `Breaking.Bad.S04E07.1080p.WEB-DL` | none | "Breaking Bad" S4E7 |
| `Breaking.Bad.Movie.2024.1080p.WEB-DL` | none | "Breaking Bad Movie" |

**Assert matching**:
- `S03E08` → title "breaking bad" matches. Verify episode: S3E8 not in index (only S3E07 wanted) → **no match**
- `S04E07` → title "breaking bad" matches. Verify season: S4 doesn't match S3 items → **no match**
- `Breaking.Bad.Movie` → title "breaking bad movie" → `TitlesMatch("breaking bad movie", "breaking bad")` → **no match** (strict equality fails)
