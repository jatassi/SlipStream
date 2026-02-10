# RSS Sync Feature Review

## Summary

The implementation covers all 11 phases of the implementation plan and is marked "DONE" across the board. The architecture is sound — the `decisioning` package extraction is the right call, and the RSS sync pipeline follows a clean fetch → match → group → score → select → grab flow. All findings from the initial review have been addressed.

---

## Critical Issues

### 1. GrabLock is NOT used by Auto Search (Spec Violation) — RESOLVED

**Fix:** Added `grabLock` field to `autosearch.Service`, with `SetGrabLock` method. `TryAcquire`/`Release` calls added in `searchAndGrab()` before the grab call. Wired in `server.go` via `s.autosearchService.SetGrabLock(s.grabLock)`.

### 2. Prowlarr Mode: Cache Boundary is Broken (IndexerID: 0) — RESOLVED

**Fix:** Rewrote `cache.go` to route Prowlarr cache boundary to the settings KV table (key: `prowlarr_rss_cache_boundary`) as JSON, avoiding the FK constraint on `indexer_status`. Sentinel constant `ProwlarrAggregatedID = 0` routes Get/Update calls through the settings path.

### 3. `highestQualityForSeason` is O(n) scan of entire index — RESOLVED

**Fix:** `highestQualityForSeason` now takes a `tvdbID` parameter and looks up `m.index.byTvdbID[tvdbID]` directly instead of scanning all maps. Falls back to title-based lookup only when tvdbID is 0.

---

## Moderate Issues

### 4. `extractExternalIDs()` is Defined But Never Called — RESOLVED

**Fix:** Added `extractExternalIDs(&release)` call at the start of `findCandidates()` in `matcher.go`, before ID-based lookups.

### 5. Season-Level Items Duplicated Between `CollectWantedItems` and `Matcher` — RESOLVED

**Fix:** Updated `matchSeasonPack` to check for existing `MediaTypeSeason` items in candidates first (from `CollectWantedItems`). Falls back to eligibility re-derivation from episode candidates only as a safety net.

### 6. No Year Validation in Movie Matching — RESOLVED

**Fix:** `matchMovie` now takes parsed media info and validates year for title-based matches. When matched by title (no external IDs), parsed year must match library year. ID-based matches skip year validation.

### 7. `HasRecentGrab` Query Doesn't Account for Season Packs — RESOLVED

**Fix:** Added `HasRecentSeasonGrab` SQL query that joins history with episodes table. In `processGroup`, season pack grabs now also check for recent individual episode grabs in the same season, and individual episode grabs check for recent season-level grabs.

### 8. Prowlarr RSS Fetch Doesn't Filter by RSS-Enabled Indexers — RESOLVED

**Fix:** Removed the per-indexer `rss_enabled` field from Prowlarr indexer settings entirely (migration `060_drop_prowlarr_rss_enabled.sql`). In Prowlarr mode, all indexers participate in RSS sync via the aggregated feed — per-indexer filtering is not possible through the Prowlarr API. Removed the unused rssSettings filtering code, RSS toggle from the Prowlarr settings dialog, and "No RSS" badge. Native indexer RSS toggle is unaffected.

### 9. Config IntervalMin Range Mismatch — NO ACTION

Current implementation (10-120 min range, 15 min default) is correct and consistent.

---

## Minor Issues

### 10. Duplicate Logger Key in `scoreAndGrab` — RESOLVED

**Fix:** Changed `Str("episode", g.item.Title)` to `Str("title", g.item.Title)` and `Int("episode", g.item.EpisodeNumber)` to `Int("episodeNumber", g.item.EpisodeNumber)`.

### 11. `SyncStatus.Error` Not Exposed in Status Endpoint — RESOLVED

**Fix:** Fixed frontend `RssSyncStatus` type to match backend (added `running`, `error` fields, renamed `lastSync` to `lastRun`). Added error display in the RSS Sync settings page with a styled alert. Additionally integrated RSS sync with the health system — feed fetch failures are reported as warnings under the indexers health category.

### 12. No `SetDB` for `SettingsHandler` — RESOLVED

**Fix:** Added `SetDB` method to `SettingsHandler`. Made `rssSyncSettingsHandler` a server field and wired it into `switchDB`.

### 13. Autosearch `scheduled.go` Not Updated to Use Shared `CollectWantedItems` — VERIFIED ACCEPTABLE

**Verification:** Autosearch intentionally uses its own collection pipeline because it requires per-item backoff tracking, failed status filtering, priority sorting by release date, and separate movie-only/series-only run modes. The core shared logic (season pack eligibility checks via `decisioning.IsSeasonPackEligible` and `decisioning.IsSeasonPackUpgradeEligible`) IS used. RSS sync uses `CollectWantedItems()` directly since it needs a simple flat list.

### 14. Generic RSS Indexer Has No Size Limit on Response Body — RESOLVED

**Fix:** Added `io.LimitReader` with 10MB limit for feed fetch and 50MB limit for download in `genericrss/indexer.go`.

### 15. No Monitoring/Alert for Repeated RSS Sync Failures — RESOLVED

**Fix:** Added in-memory backoff tracking to `FeedFetcher` with consecutive failure counts per indexer. After 3 consecutive failures, the indexer is skipped with a warning. Counts reset on success or server restart. Health system integration reports feed fetch failures as warnings under the indexers category.

---

## Design Spec Compliance Checklist

| Spec Requirement | Status | Notes |
|---|---|---|
| Shared decisioning package | Implemented | Clean extraction |
| Per-media grab lock (both systems) | **Implemented** | Both RSS sync and autosearch use shared GrabLock |
| RSS settings (enabled, interval) | Implemented | |
| Per-indexer RSS toggle (native) | Implemented | `rss_enabled` column |
| Per-indexer RSS toggle (Prowlarr) | **Removed** | Not possible through Prowlarr API; column dropped |
| Cache boundary tracking | **Implemented** | Prowlarr uses settings KV table; native uses indexer_status |
| Feed fetching (native) | Implemented | With backoff tracking |
| Feed fetching (Prowlarr) | Implemented | Aggregated dual search |
| Generic RSS Feed indexer | Implemented | Parser, settings, UI, size limits |
| WantedIndex + matching | Implemented | ID + title fallback with extractExternalIDs |
| Season pack eligibility | Implemented | Shared with CollectWantedItems |
| Season pack suppression | Implemented | |
| History-based re-grab prevention | **Implemented** | Cross-checks season/episode grabs |
| Scheduler integration | Implemented | Dynamic interval updates |
| WebSocket events | Implemented | 4 event types |
| Frontend settings UI | Implemented | With error display |
| Health integration | **Implemented** | Feed failures reported as indexer health warnings |
| Indexer backoff | **Implemented** | In-memory consecutive failure tracking |
| Test scenarios 1-15 | **Not verified** | Test files exist but not run |
