# RSS Sync Implementation Plan

## Context

Auto-search currently makes one indexer API call per wanted item per indexer per cycle. For a library with 50 missing items across 5 indexers, that's ~250 API calls. RSS Sync inverts this: fetch each indexer's RSS feed once (5 calls), then match releases client-side against all wanted items. This reduces API usage by 95%+ and catches new releases within minutes.

**Scope**: Full implementation including backend, Generic RSS Feed indexer, frontend UI, and tests.

## Phase 1: Shared Decisioning Package (`internal/decisioning/`) ✅ DONE

Extract shared logic from autosearch so both autosearch and rsssync can use it.

**New files:**
- `internal/decisioning/types.go` — `SearchableItem`, `MediaType` constants
- `internal/decisioning/selection.go` — `SelectBestRelease()` (exported, standalone func)
- `internal/decisioning/collection.go` — `CollectWantedItems()`, season pack eligibility checks, item conversion helpers
- `internal/decisioning/grablock.go` — Per-media-item grab lock

**What moves:**

| From | To |
|---|---|
| `autosearch/types.go` → `SearchableItem`, `MediaType` | `decisioning/types.go` |
| `autosearch/service.go:674` → `selectBestRelease()` | `decisioning/selection.go` → `SelectBestRelease()` |
| `autosearch/service.go:1064` → `isSeasonPackEligible()` | `decisioning/collection.go` → `IsSeasonPackEligible()` |
| `autosearch/service.go:1137` → `isSeasonPackUpgradeEligible()` | `decisioning/collection.go` → `IsSeasonPackUpgradeEligible()` |
| `autosearch/scheduled.go:377` → `collectSearchableItems()` + sub-collectors | `decisioning/collection.go` → `CollectWantedItems()` |
| `autosearch/service.go:796` → `movieToSearchableItem()` and friends | `decisioning/collection.go` |

**What stays in autosearch:**
- `searchAndGrab()`, `buildSearchCriteria()`, backoff tracking, scheduler, handlers, settings, events
- `SearchRequest`, `SearchResult`, `BatchSearchResult`, `SearchSource` types

**Approach:**
1. Create `decisioning` package with extracted types/functions
2. `SelectBestRelease` takes `logger zerolog.Logger` as param instead of `*Service` receiver
3. `CollectWantedItems` takes a `Collector` struct with `Queries *sqlc.Queries`, `Logger`, `BackoffChecker` interface
4. Autosearch uses type aliases (`type SearchableItem = decisioning.SearchableItem`) for zero-breakage
5. Autosearch calls `decisioning.SelectBestRelease()` and `decisioning.CollectWantedItems()`

**GrabLock** (in `decisioning/grablock.go`):
```go
type GrabLock struct {
    mu    sync.Mutex
    locks map[string]struct{} // "movie:123" or "episode:456"
}
func (g *GrabLock) TryAcquire(key string) bool
func (g *GrabLock) Release(key string)
```

Both auto-search and RSS sync must use this shared lock. Keyed by `mediaType + ":" + mediaID`.

**Files to modify:**
- `internal/autosearch/types.go`
- `internal/autosearch/service.go`
- `internal/autosearch/scheduled.go`

## Phase 2: DB Migration + Settings + Per-Indexer RSS Toggle ✅ DONE

**New migration** `internal/database/migrations/059_rss_sync.sql`:
```sql
ALTER TABLE indexers ADD COLUMN rss_enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE prowlarr_indexer_settings ADD COLUMN rss_enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE indexer_status ADD COLUMN last_rss_release_url TEXT;
ALTER TABLE indexer_status ADD COLUMN last_rss_release_date DATETIME;
```

**New SQL queries** in `queries/indexers.sql`:
- `ListRssEnabledIndexers`, `ListRssEnabledMovieIndexers`, `ListRssEnabledTVIndexers`
- `UpdateIndexerRssEnabled`
- `UpdateIndexerRssCache` — update cache boundary
- `GetIndexerRssCache` — get cache boundary

**New SQL query** in `queries/prowlarr_indexer_settings.sql`:
- `UpdateProwlarrIndexerRssEnabled`

**New SQL query** for re-grab prevention in `queries/history.sql`:
```sql
-- name: HasRecentGrab :one
SELECT COUNT(*) > 0 FROM history
WHERE media_type = ? AND media_id = ?
AND event_type IN ('grabbed', 'autosearch_download')
AND created_at > datetime('now', '-12 hours');
```

Run `sqlc generate` after adding queries.

**Config** — add to `internal/config/config.go`:
```go
type RssSyncConfig struct {
    Enabled     bool `mapstructure:"enabled"`
    IntervalMin int  `mapstructure:"interval_min"`
}
```

## Phase 3: RSS Sync Core (`internal/rsssync/`) ✅ DONE

**New files:**
- `service.go` — Core orchestration
- `matcher.go` — WantedIndex + release matching
- `feedfetcher.go` — Feed fetching from indexers
- `cache.go` — Per-indexer last-seen tracking
- `settings.go` — Settings handler (GET/PUT `/api/v1/settings/rsssync`)
- `handlers.go` — API endpoints (trigger, status)
- `events.go` — WebSocket event definitions

### 3A: Feed Fetcher

Fetches recent releases from all RSS-enabled indexers. Hard limit of **1000 results per indexer**.

**Native (Cardigann) indexers:**
- `indexer.Search(ctx, SearchCriteria{})` with empty/minimal query → returns recent items (equivalent to Torznab `?t=search`)
- Support paginated feeds: follow `IndexerPageableRequestChain` pattern until reaching the last-seen release or 1000-item hard limit
- For indexers supporting only one media type, note this and only match against the relevant type

**Prowlarr indexers:**
- Two calls per indexer: `Search(ctx, {Type: "movie"})` and `Search(ctx, {Type: "tvsearch"})` to ensure broad coverage
- **Must bypass Prowlarr's 5-minute search result cache** to avoid stale results when auto-search ran recently with similar parameters
- Existing Torznab XML parsing (`TorznabItem.ToTorrentInfo()`) handles result conversion

Returns `[]types.TorrentInfo` per indexer.

### 3B: Wanted Items Index + Matcher

```go
type WantedIndex struct {
    byTitle  map[string][]decisioning.SearchableItem
    byImdbID map[string][]decisioning.SearchableItem
    byTmdbID map[int][]decisioning.SearchableItem
    byTvdbID map[int][]decisioning.SearchableItem
}
```

**Match algorithm per release:**
1. Parse with `scanner.ParseFilename()` to extract: title, year, season, episode, quality, source
2. ID-based match first — if Torznab item includes `imdbid`, `tmdbid`, or `tvdbid`, look up directly in index
3. Title fallback — `search.NormalizeTitle()` + `search.TitlesMatch()` (strict equality after normalization)
4. TV verification — confirm season/episode matches a wanted episode
5. Season pack handling — when `parsed.IsSeasonPack`, run full eligibility check (see below)

**Unreleased episode filtering:** `CollectWantedItems()` only returns episodes with actionable statuses (missing, upgradable). Unreleased episodes are never in the WantedIndex. No special filtering needed in the matcher — exclusion happens at index construction.

**Season pack eligibility** (requires DB query for full season, not just WantedIndex):
1. Query ALL episodes for the series + season from DB (all statuses)
2. Any episode has **unreleased** status → **reject** season pack
3. **Missing scenario**: ALL episodes must be missing → eligible. If some available/upgradable and others missing → **reject** (partial overlap)
4. **Upgrade scenario**: ALL episodes must have files (upgradable or available). Match only if pack upgrades the highest `CurrentQualityID` across all episode files. If some at/above cutoff and others below → **reject** (mixed upgrade state)
5. If eligible, create synthetic `SearchableItem{MediaType: "season"}` for scoring

**Season pack vs individual episode conflict:** When an individual episode matches, skip it if a season pack for the same series+season was already matched and scored higher in this sync cycle.

### 3C: Service Run() flow

1. Broadcast `rss-sync:started`
2. Collect wanted items via `decisioning.CollectWantedItems()` (no backoff — RSS doesn't use backoff)
3. Build `WantedIndex`
4. For each RSS-enabled indexer:
   a. Fetch feed (paginate if needed, up to 1000 items)
   b. Process releases newest to oldest
   c. Stop at cache boundary (matching `last_rss_release_url` AND `last_rss_release_date`)
   d. If cache boundary never reached (first sync or gap), process all items and log gap warning
   e. Match each release against WantedIndex
5. Group matched releases by target wanted item
6. For each group:
   a. Score with `scoring.NewDefaultScorer()` using quality profile from wanted item
   b. Sort by score (highest first)
   c. `decisioning.SelectBestRelease()` — applies `IsAcceptable()`, `IsUpgrade()` (when `HasFile`)
   d. History check — reject if media item grabbed within last 12 hours
   e. Acquire `GrabLock` (keyed by `mediaType:mediaID`) — skip if already locked
   f. Grab via `grab.Service.Grab()` with `GrabRequest.Source = "rss-sync"`
   g. Release `GrabLock`
7. Season pack suppression: suppress individual episode grabs covered by higher-scoring season pack
8. Update per-indexer cache boundary only after successful sync completion
9. Broadcast `rss-sync:completed`

**Edge cases handled in Run():**
- **First sync**: No cache exists → process all items, establish cache from newest release
- **Indexer temporarily down**: Skip it, retry next cycle. Don't invalidate cache. Auto-search serves as backfill
- **High-volume indexer**: If cache boundary not reached within 1000 items, log a gap warning suggesting higher sync frequency
- **Mixed content feeds**: Only match against media types in SlipStream; ignore unrecognized items silently
- **Grab failure**: Failed grabs do not advance cache boundary. Release re-evaluated next cycle. History only written on successful grabs

### 3D: Settings Handler

Following `autosearch/settings.go` pattern:
- Settings key: `"rsssync_settings"` in `settings` KV table
- `LoadSettingsIntoConfig()` at startup
- Dynamic scheduler update on settings change
- Minimum interval: 10 minutes, default: 15 minutes

### 3E: API Endpoints

```
GET    /api/v1/settings/rsssync    — Get settings
PUT    /api/v1/settings/rsssync    — Update settings
POST   /api/v1/rsssync/trigger     — Manual trigger
GET    /api/v1/rsssync/status      — Last sync stats
```

Generic RSS Feed indexer CRUD uses existing `/api/v1/indexers` endpoints (with `definitionId: "generic-rss"`).

### 3F: WebSocket Events

```
rss-sync:started    { indexerCount }
rss-sync:progress   { indexer, releasesFound, matched }
rss-sync:completed  { totalReleases, matched, grabbed, elapsed }
rss-sync:failed     { error }
```

## Phase 4: Scheduler Task ✅ DONE

**New file** `internal/scheduler/tasks/rsssync.go`:
- `RegisterRssSyncTask()`, `UpdateRssSyncTask()`
- Default cron: `"*/15 * * * *"` (every 15 min, configurable)
- `RunOnStart: true`

## Phase 5: Wire in Server ✅ DONE

In `internal/api/server.go`:
1. Create `decisioning.GrabLock` shared instance
2. Create `rsssync.Service` with all dependencies
3. Pass GrabLock to autosearch service and rsssync service
4. Load RSS sync settings into config at startup
5. Register scheduler task
6. Register API routes under `/api/v1/settings/rsssync` and `/api/v1/rsssync`
7. Handle `SetDB()` for dev mode switching

## Phase 6: Generic RSS Feed Indexer Type ✅ DONE

A new indexer type for plain RSS feeds with no Torznab/Newznab API.

### 6A: Backend

**Definition**: Uses existing `indexers` table with `definition_id = 'generic-rss'`.

**Settings JSON:**
```json
{
    "url": "https://example.com/rss",
    "cookie": "",
    "contentType": "both"  // "movies", "tv", or "both"
}
```

**Capabilities:**
- `SupportsRSS`: true
- `SupportsSearch`: false
- No search methods — only participates in RSS sync
- Not used by auto-search or interactive search

**Feed parsing** — auto-detect format (mirroring Sonarr's `TorrentRssParserFactory`):
- **Standard RSS** — `<item>` with `<title>`, `<link>`/`<enclosure>`, `<pubDate>`, `<guid>`
- **TorrentPotato** — JSON format used by some private trackers
- **EzRSS** — namespace-extended RSS with torrent-specific elements
- **Custom** — fallback heuristic parsing based on feed structure

**Extract from each item:**
- Title — release name (parsed for quality/season/episode info)
- Download URL — torrent file or magnet link
- Publish date — for cache ordering
- GUID / download URL — for deduplication
- Size — file size (optional; some feeds omit this)

**New files:**
- `internal/indexer/genericrss/indexer.go` — Implements indexer interface (RSS-only, no search)
- `internal/indexer/genericrss/parser.go` — Auto-detect parser for Standard RSS, TorrentPotato, EzRSS, custom
- `internal/indexer/genericrss/parser_test.go` — Tests for each feed format

**Files to modify:**
- `internal/indexer/service.go` — Register `generic-rss` definition
- `internal/rsssync/feedfetcher.go` — Add fetching path for generic-rss indexers
- API handlers for indexer CRUD already handle generic definitions; verify `generic-rss` works

### 6B: Frontend — Generic RSS Feed Indexer Form

The Generic RSS Feed indexer integrates with the existing `IndexerDialog` two-step wizard. The backend registers `generic-rss` as a definition returned by `GET /api/v1/indexers/definitions`, so it appears in `DefinitionSearchTable` alongside Cardigann definitions. The definition schema endpoint (`GET /api/v1/indexers/definitions/generic-rss/schema`) returns the settings fields so `DynamicSettingsForm` renders the form automatically.

**Backend definition registration** (in `internal/indexer/service.go`):
- Register `generic-rss` in the definitions list with `protocol: "torrent"`, `privacy: "private"`
- Schema returns 3 fields: `url` (text, required), `cookie` (password, optional), `contentType` (select: movies/tv/both)

**Frontend behavior** — no custom form component needed:
- User clicks "Add Indexer" → sees `generic-rss` ("Generic RSS Feed") in the definition table
- Selects it → `DynamicSettingsForm` renders URL, Cookie, Content Type fields from schema
- Movies/TV toggles, Priority, Enabled, Auto Search, RSS Sync toggles all work as normal
- Auto Search toggle greyed out with helper text since generic-rss doesn't support search
- Test button validates feed URL and returns sample parsed item count

## Phase 7: Frontend UI ✅ DONE

### 7A: Types

**New file** `web/src/types/rsssync.ts`:
```typescript
export interface RssSyncSettings {
  enabled: boolean
  intervalMin: number
}

export interface RssSyncStatus {
  lastSync?: string
  totalReleases: number
  matched: number
  grabbed: number
  elapsed: number
}
```

**Modify** `web/src/types/indexer.ts`:
- `Indexer`: add `rssEnabled: boolean`
- `CreateIndexerInput`: add `rssEnabled?: boolean`
- `UpdateIndexerInput`: add `rssEnabled?: boolean`

**Modify** `web/src/types/prowlarr.ts`:
- `ProwlarrIndexerSettings`: add `rssEnabled: boolean`
- `ProwlarrIndexerSettingsInput`: add `rssEnabled: boolean`

**Modify** `web/src/types/index.ts`: export new RSS sync types.

### 7B: API Module + Hooks

**New file** `web/src/api/rsssync.ts`:
```typescript
rssSyncApi = {
  getSettings()     → GET /api/v1/settings/rsssync
  updateSettings()  → PUT /api/v1/settings/rsssync
  getStatus()       → GET /api/v1/rsssync/status
  trigger()         → POST /api/v1/rsssync/trigger
}
```

**New file** `web/src/hooks/useRssSync.ts` (follows `useAutosearch.ts` pattern):
- `useRssSyncSettings()` — query for settings
- `useUpdateRssSyncSettings()` — mutation for settings
- `useRssSyncStatus()` — query for last sync status (5s staleTime)
- `useTriggerRssSync()` — mutation for manual trigger

**Modify** `web/src/hooks/index.ts`: export new hooks.

### 7C: RSS Sync Settings Page

**New route file** `web/src/routes/settings/downloads/rss-sync.tsx`:
- Follows `auto-search.tsx` pattern: `PageHeader` + `DownloadsNav` + `RssSyncSection`
- Register route in `web/src/router.tsx` under `/settings/downloads/rss-sync`

**Modify** `web/src/routes/settings/downloads/DownloadsNav.tsx`:
- Add fourth tab: `{ title: 'RSS Sync', href: '/settings/downloads/rss-sync', icon: Rss }`

**New file** `web/src/components/settings/sections/RssSyncSection.tsx`:

Follows `AutoSearchSection.tsx` pattern (render-time state sync, `hasChanges` tracking, save button):

**Card 1 — "RSS Sync":**
- Enable/disable switch with label "Enable RSS Sync" and description "Periodically fetch RSS feeds from indexers and grab matching releases"
- Interval slider: min=10, max=60, step=5, default=15 minutes. Disabled when RSS sync is off
- Description: "How often to check RSS feeds for new releases (10-60 minutes)"

**Card 2 — "Last Sync"** (read-only, from `useRssSyncStatus()`):
- Last sync time (relative, e.g., "3 minutes ago")
- Stats row: releases found / matched / grabbed / elapsed time
- Manual trigger button (`useTriggerRssSync()`) with loading spinner

Save button at bottom (only for Card 1 settings changes).

**Modify** `web/src/components/settings/index.ts`: export `RssSyncSection`.

### 7D: Per-Indexer RSS Toggle — SlipStream Mode

**Modify** `web/src/components/indexers/IndexerDialog.tsx`:

Add `rssEnabled` to `formData` state (default: `true`). Add to edit-mode hydration and create/update payload.

Add toggle after the existing "Enable for Automatic Search" toggle (line ~348):
```tsx
<div className="flex items-center justify-between">
  <div className="space-y-0.5">
    <Label htmlFor="rssEnabled">Enable for RSS Sync</Label>
    <p className="text-xs text-muted-foreground">
      Include this indexer when fetching RSS feeds for new releases
    </p>
  </div>
  <Switch
    id="rssEnabled"
    checked={formData.rssEnabled}
    onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, rssEnabled: checked }))}
  />
</div>
```

For `generic-rss` definitions: RSS toggle is always on and disabled (greyed out). Auto Search toggle is always off and disabled. Detect via `selectedDefinition?.id === 'generic-rss'`.

**Modify** `web/src/components/settings/sections/IndexersSection.tsx`:

Add "No RSS" indicator on indexer cards, alongside existing "Manual search only" indicator (line ~217):
```tsx
{!indexer.rssEnabled && (
  <>
    <span className="text-muted-foreground/50">|</span>
    <span className="text-yellow-500">No RSS</span>
  </>
)}
```

### 7E: Per-Indexer RSS Toggle — Prowlarr Mode

**Modify** `web/src/components/indexers/ProwlarrIndexerList.tsx`:

In `IndexerSettingsDialog` (line ~256), add `rssEnabled` state and toggle alongside existing Priority and Content Type fields:
```tsx
<div className="flex items-center justify-between">
  <div className="space-y-0.5">
    <Label>Enable for RSS Sync</Label>
    <p className="text-xs text-muted-foreground">
      Include this indexer when fetching RSS feeds
    </p>
  </div>
  <Switch checked={rssEnabled} onCheckedChange={setRssEnabled} />
</div>
```

Include `rssEnabled` in `handleSave` payload. Initialize from `indexer.settings?.rssEnabled ?? true` in `handleOpenChange`.

In `IndexerRow`, add "No RSS" indicator in the description metadata row when `settings?.rssEnabled === false`.

### 7F: Sync Status Display

- WebSocket events (`rss-sync:started`, `rss-sync:progress`, `rss-sync:completed`, `rss-sync:failed`) display in the existing activity/toast system
- History entries from RSS sync grabs are distinguishable via `source: "rss-sync"` (already handled by existing history UI which shows the source field)

## Phase 8: Tests ✅ DONE

Implement test scenarios 1-15 from the design document. Each scenario tests both the auto-search path and the RSS sync path using the same library state and quality profiles.

### Shared Test Fixtures

**Profile "HD-1080p"** (ID: 5):
- Allowed: HDTV-720p (4), WEBRip-720p (5), WEBDL-720p (6), Bluray-720p (7), HDTV-1080p (8), WEBRip-1080p (9), WEBDL-1080p (10), Bluray-1080p (11)
- Cutoff: Bluray-1080p (11), upgrades enabled, balanced strategy

**Standard noise releases** (N1-N7): included in every RSS sync test feed, all must produce no match.

### Test Scenarios

| # | Scenario | Key Assertion |
|---|---|---|
| 1 | Missing movie | Bluray-1080p selected; 2160p rejected as unacceptable |
| 2 | Upgradable movie | Bluray-1080p upgrade from WEBDL-720p; same-quality rejected |
| 3 | Upgradable movie, no upgrade available | Returns nil; both same-or-lower quality fail IsUpgrade |
| 4 | Missing episode | Exact episode match; season pack + wrong episode rejected |
| 5 | Upgradable episode — disc source upgrade | Bluray-1080p upgrades HDTV-1080p via disc source; WEBDL-1080p rejected |
| 6 | Upgradable episode — non-disc upgrade rejected | Returns nil; WEBDL and HDTV fail balanced strategy upgrade |
| 7 | All episodes missing — season pack eligible (ended series) | Season pack selected; individual grabs suppressed |
| 8 | All episodes upgradable — season pack upgrade (ended series) | Season pack upgrade selected; same-quality pack rejected |
| 9 | Season upgrade rejected — not all upgradable | Season pack ineligible; individual episode upgrade selected |
| 10 | Continuing season — missing + unreleased | Season pack rejected (unreleased present); individual missing eps matched |
| 11 | Season premiere — 1 aired + rest unreleased | Only E01 matched; season pack and unreleased eps rejected |
| 12 | Upgradable season + unreleased episodes | Season pack rejected (unreleased); individual upgrade selected |
| 13 | Season transitions — unreleased → missing (two-phase) | Phase A: pack rejected. Phase B: all missing → pack eligible |
| 14 | Season pack blocked after partial grabs | Pack rejected (E01 available); individual missing eps matched |
| 15 | Title-based matching (no external IDs) | Strict title normalization match; partial/fuzzy titles rejected |

Each scenario includes both A (auto-search path) and B (RSS sync path) sub-tests. RSS sync tests additionally verify all 7 noise releases produce no match.

**Test files:**
- `internal/decisioning/selection_test.go` — SelectBestRelease scenarios (1A-15A)
- `internal/rsssync/matcher_test.go` — WantedIndex matching + grouping (1B-15B)
- `internal/rsssync/service_test.go` — End-to-end sync orchestration
- `internal/rsssync/cache_test.go` — Cache boundary tracking
- `internal/indexer/genericrss/parser_test.go` — Feed format detection and parsing

## Key Reuse Points

| Reused | Location | Purpose |
|---|---|---|
| `scanner.ParseFilename()` | `library/scanner/parser.go` | Parse release titles |
| `search.TitlesMatch()` / `NormalizeTitle()` | `indexer/search/title_match.go` | Title matching fallback |
| `scoring.NewDefaultScorer()` | `indexer/scoring/scorer.go` | Score releases |
| `quality.Profile.IsAcceptable()` / `IsUpgrade()` | `library/quality/profile.go` | Quality checks |
| `grab.Service.Grab()` | `indexer/grab/service.go` | Download client dispatch |
| `history.Service.LogAutoSearchDownload()` | `history/service.go` | Log grabs (source="rss-sync") |
| Settings KV pattern | `autosearch/settings.go` | Settings storage |
| Scheduler task pattern | `scheduler/tasks/autosearch.go` | Task registration |

## Out of Scope (V1)

Per design doc, explicitly deferred:
- Proper/repack handling
- Multi-episode releases (e.g., "S01E05E06")
- Daily shows (date-based episode matching)
- Anime absolute numbering
- Release size validation
- Indexer tag filtering

## Verification

After each phase: `make test` to ensure no regressions. After Phase 5: restart backend, verify settings endpoints respond, manual trigger works, scheduler task registered. After Phase 7: `cd web && bun run build` succeeds with no type errors; verify RSS Sync tab appears in Download Pipeline nav; settings page loads/saves; per-indexer RSS toggle appears in both SlipStream and Prowlarr mode indexer dialogs; generic-rss definition appears in indexer add flow. After Phase 8: all 15 test scenarios pass for both auto-search and RSS sync paths.
