# Unified Media Status Consolidation

## Overview

This spec replaces the current fragmented status tracking (separate `status`, `released`, `availabilityStatus`, and `hasFile` fields with inconsistent semantics across media types) with a single unified `status` field. Status is tracked at the **leaf level**: on movies and episodes when multi-version is disabled, or on individual **slot assignments** when multi-version is enabled (with media-level status computed from slots). Series and season status is computed from their child episodes and returned as structured counts.

## Current State (Problems)

| Entity | Field | Values | Tracks |
|--------|-------|--------|--------|
| Movie | `status` | missing, downloading, available | File presence |
| Movie | `released` | bool | Release date passed |
| Movie | `availabilityStatus` | "Available", "Unreleased" | Display badge (redundant with `released`) |
| Series | `status` | continuing, ended, upcoming | **Production** status (different semantics!) |
| Series | `released` | bool | All seasons released |
| Series | `availabilityStatus` | "Seasons 1-3 Available" etc | File-based completeness (misleading name) |
| Season | `released` | bool | All episodes aired |
| Episode | `released` | bool | Air date passed |
| Episode | (none) | derived from episode_files join | File presence (`hasFile`) |

**Key issues:**
- Movie `status` and series `status` use the same field name for completely different concepts
- `released` + `availabilityStatus` are partially redundant
- Series `availabilityStatus` is based on file presence, not release status
- Episodes have no explicit status column
- No way to represent "downloading" or "failed" for episodes
- Unreleased content must be excluded from search, but this requires cross-referencing `released` with `status`

## Unified Status Model

### Status Values

A single enum applies to both movies and episodes:

| Status | Meaning | Has File | Search Eligible |
|--------|---------|----------|-----------------|
| `unreleased` | No release date, or release/air datetime is in the future | No | No |
| `missing` | Released but no file on disk | No | Yes (if monitored) |
| `downloading` | Active download in progress | Maybe* | No |
| `failed` | Download or import failed | No | No (manual retry required) |
| `upgradable` | File exists but below quality profile cutoff | Yes | Yes (if monitored, upgrades enabled) |
| `available` | File exists at or above quality cutoff | Yes | No |

\* During an upgrade download, the previous file still exists on disk. The queue item context indicates whether it's an initial download or upgrade.

**Slot-aware status:** When multi-version slots are enabled, each slot assignment is a leaf node with its own independent status. The media item's `status` field is a cached aggregate computed from its slot statuses. See [Multi-Version Slot Integration](#multi-version-slot-integration) for rules.

### Status Transitions

```
                    ┌──────────────────────────────────────┐
                    │          date pushed forward          │
                    ▼                                       │
              ┌────────────┐   date passes   ┌─────────┐   │
              │ unreleased ├────────────────►│ missing  ├───┘
              └────────────┘                 └────┬─────┘
                                                  │ download starts
                                                  ▼
                                          ┌──────────────┐
                              ┌───────────┤ downloading  ├──────────┐
                              │           └──────┬───────┘          │
                              │                  │           import fail
                              │ import           │ import           │
                              │ (≥ cutoff)       │ (< cutoff)      │
                              ▼                  ▼                  ▼
     ┌────────────┐     ┌───────────┐                        ┌──────────┐
     │ upgradable │◄───►│ available │                        │  failed  │
     └─────┬──────┘     └─────┬─────┘                        └────┬─────┘
           │                  │                                    │
           │ upgrade starts   │ file removed                      │ manual retry
           │                  ▼                                    │
           │            ┌─────────┐                                │
           └───────────►│ missing │◄───────────────────────────────┘
                        └─────────┘
```

### Transition Details

These transitions apply at the **leaf level**: directly on the movie/episode when multi-version is disabled, or on the **slot assignment** when multi-version is enabled. When operating on a slot, the media-level cached status is recomputed after each transition.

| From | To | Trigger | Notes |
|------|----|---------|-------|
| `unreleased` | `missing` | Release date/air datetime passes | Hourly scheduler + metadata refresh. Applied to all slots for the item. |
| `missing` | `unreleased` | Air date pushed to future by metadata refresh | Auto-recalculate on metadata update. Applied to all slots for the item. |
| `missing` | `downloading` | Download mapping created (release grabbed) | Persists `active_download_id`. Triggered when SlipStream sends the release to the download client, not when the client begins transferring. When multi-version enabled, set on the specific slot identified by `download_mappings.target_slot_id`. |
| `downloading` | `available` | File imported at or above quality cutoff | Clears `active_download_id`. Quality must be evaluated during import. |
| `downloading` | `upgradable` | File imported below quality cutoff | Clears `active_download_id`. Only when upgrades are enabled on the profile. |
| `downloading` | `failed` | Import fails or download errors | Stores reason in `status_message` |
| `available` | `upgradable` | Quality profile cutoff changes | Immediate recalculation. When multi-version enabled, evaluated per-slot against that slot's quality profile. |
| `upgradable` | `available` | Quality profile cutoff lowered, or upgrade imported | Immediate recalculation |
| `upgradable` | `downloading` | Upgrade download starts | Existing file remains on disk. If upgrade fails, transitions to `failed`. |
| `available` | `missing` | File removed manually via UI | **Sets `monitored = false`** on the specific slot (multi-version) or the item (single-version). |
| `available` | `missing` | File disappears from disk (scan) | Keeps `monitored`, triggers health alert |
| `upgradable` | `missing` | File removed (manual or disk) | Same rules as `available` → `missing` |
| `failed` | `missing` or `upgradable` | User triggers manual retry | Check file presence at retry time: if a file exists → `upgradable`, if not → `missing`. Resets `failure_count` in autosearch_status. |
| `downloading` | `failed` | Download disappears from client without completing | Via download sync polling |

**Concurrent downloads:** Each slot has its own `active_download_id`, so concurrent downloads targeting different slots are tracked independently. For the single-version case (or two grabs targeting the same slot), `active_download_id` stores only the most recent download mapping. The old download continues in the client but is no longer tracked—it will be cleaned up by `DeleteOldDownloadMappings()` after 7 days.

### Separate Fields (Not Part of Status)

| Field | Scope | Type | Purpose |
|-------|-------|------|---------|
| `monitored` | Movie, Episode, Season, Series | boolean | Whether auto-search should include this item |
| `production_status` | Series only | enum: continuing, ended, upcoming | Renamed from current `status`. Show lifecycle on network. |

## Schema Changes

### Movies Table

**Add columns:**
```sql
active_download_id TEXT,       -- Download client item ID when status='downloading'
status_message TEXT            -- Failure reason when status='failed'
```

**Modify columns:**
- `status` CHECK constraint: `CHECK(status IN ('unreleased', 'missing', 'downloading', 'failed', 'upgradable', 'available'))`

**Remove columns:**
- `released` (boolean) — derivable from `status != 'unreleased'`
- `availability_status` (text) — replaced by unified status
- `digital_release_date` — already deprecated

### Episodes Table

**Add columns:**
```sql
status TEXT NOT NULL DEFAULT 'unreleased'
    CHECK(status IN ('unreleased', 'missing', 'downloading', 'failed', 'upgradable', 'available')),
active_download_id TEXT,
status_message TEXT
```

**Modify columns:**
- `air_date` type: `DATE` → `DATETIME` (to support air time, not just date)

**Remove columns:**
- `released` (boolean) — derivable from `status != 'unreleased'`

### Series Table

**Rename columns:**
- `status` → `production_status` (same CHECK constraint: continuing, ended, upcoming)

**Remove columns:**
- `released` (boolean) — computed from episodes
- `availability_status` (text) — replaced by computed statusCounts

**Preserve columns (unchanged):**
- `format_type` (text, nullable) — series format override (standard/daily/anime), must be carried through table recreation

### Seasons Table

**Remove columns:**
- `released` (boolean) — computed from episodes

### Slot Assignment Tables (Multi-Version)

When multi-version is enabled, slot assignments are the leaf nodes for status tracking. Add status fields to both `movie_slot_assignments` and `episode_slot_assignments`:

**Add columns:**
```sql
status TEXT NOT NULL DEFAULT 'missing'
    CHECK(status IN ('unreleased', 'missing', 'downloading', 'failed', 'upgradable', 'available')),
active_download_id TEXT,       -- Download client item ID when slot status='downloading'
status_message TEXT            -- Failure reason when slot status='failed'
```

The existing `file_id`, `monitored`, `movie_id`/`episode_id`, and `slot_id` columns are unchanged.

**Behavior:** When multi-version is enabled, status transitions operate on the slot assignment row. After each slot status change, the media item's cached `status` column is recomputed from all its monitored slots. When multi-version is disabled, slot assignment rows may not exist—the media item's `status` column is the direct source of truth.

### Requests Table (Portal)

**Modify columns:**
- `status` CHECK constraint: add `'failed'` → `CHECK(status IN ('pending', 'approved', 'denied', 'downloading', 'failed', 'available'))`

## Data Migration

Single migration that transforms all existing data:

### Movies
```sql
-- Transform status values using released flag
UPDATE movies SET status = CASE
    WHEN status = 'available' THEN 'available'
    WHEN status = 'downloading' THEN 'downloading'
    WHEN status = 'missing' AND released = 1 THEN 'missing'
    WHEN status = 'missing' AND released = 0 THEN 'unreleased'
    ELSE 'missing'
END;
```

### Episodes
```sql
-- Compute initial status from released flag + file existence
UPDATE episodes SET status = CASE
    WHEN released = 0 THEN 'unreleased'
    WHEN released = 1 AND id IN (SELECT episode_id FROM episode_files) THEN 'available'
    WHEN released = 1 THEN 'missing'
    ELSE 'unreleased'
END;
```

### Series
```sql
-- Rename status to production_status (requires table recreation in SQLite)
ALTER TABLE series RENAME COLUMN status TO production_status;
```

### Slot Assignments
```sql
-- Initialize slot assignment status from file presence + media release state
-- Only for existing slot assignment rows (lazy creation means most items won't have any)
UPDATE movie_slot_assignments SET status = CASE
    WHEN (SELECT m.status FROM movies m WHERE m.id = movie_slot_assignments.movie_id) = 'unreleased'
        THEN 'unreleased'
    WHEN file_id IS NOT NULL THEN 'available'
    ELSE 'missing'
END;

UPDATE episode_slot_assignments SET status = CASE
    WHEN (SELECT e.status FROM episodes e WHERE e.id = episode_slot_assignments.episode_id) = 'unreleased'
        THEN 'unreleased'
    WHEN file_id IS NOT NULL THEN 'available'
    ELSE 'missing'
END;
```

### Post-Migration Recalculation

After the migration runs, trigger a one-time recalculation to set `upgradable` status for items with files below their quality profile cutoff. This applies to both media-level status (single-version items) and slot-level status (multi-version items with assignments). This reuses the quality profile recalculation logic (see [Quality Profile Changes](#quality-profile-changes)).

## Scheduler Changes

### Availability Refresh (Renamed: Status Refresh)

**Current:** Runs daily at midnight (`0 0 * * *`)
**New:** Runs hourly (`0 * * * *`)

The task is renamed from "Refresh Media Availability" to "Refresh Media Status" and performs:

1. **Movies:** Update `status` from `unreleased` → `missing` where `release_date IS NOT NULL AND release_date <= date('now')` and `status = 'unreleased'`
2. **Episodes:** Update `status` from `unreleased` → `missing` where `air_date IS NOT NULL AND air_date <= datetime('now')` and `status = 'unreleased'`

Note: Episodes use datetime comparison (not just date) to respect air times.

**Air time fallback:** Most metadata providers (TVDB, TMDB) only supply air *dates*, not times. When `air_date` has no time component (stored as midnight UTC), the scheduler should treat date-only values the same as movies: `air_date <= date('now')`. Only apply the more precise datetime comparison when a non-midnight time is present. Without this fallback, episodes would show as `missing` at midnight UTC—potentially hours before they actually air in their local timezone.

The reverse transition (`missing` → `unreleased` when dates are pushed forward) is handled by the metadata refresh process, not the scheduler.

### Download Sync

The existing download completion checker (`CheckForCompletedDownloads`) is extended to handle status sync. All status writes target the **leaf node**: the slot assignment when multi-version is enabled (identified by `download_mappings.target_slot_id`), or the media item directly when multi-version is disabled. After any slot-level write, recompute the media item's cached status.

1. **Download starts:** When a download mapping is created (release grabbed from indexer), set leaf status to `downloading` and store `active_download_id`. This happens at grab time, not when the download client begins transferring.
2. **Download completes + imports:** Evaluate imported file quality against the applicable quality profile cutoff (slot's profile when multi-version, media's profile when single-version). Set leaf status to `available` if at or above cutoff, or `upgradable` if below cutoff and upgrades are enabled. Clear `active_download_id`. Currently `AddFile()` always sets status to `"available"`—this must be changed to perform quality evaluation.
3. **Download fails:** Set leaf status to `failed`, store reason in `status_message`, clear `active_download_id`
4. **Download disappears:** If a leaf has `status = 'downloading'` but no matching active download exists in any client, set status to `failed` with message "Download removed from client"

### Season Pack Downloads and `queue_media`

The existing `queue_media` table tracks per-file status within multi-file downloads (season packs). Each file has its own `file_status` (`pending|downloading|ready|importing|imported|failed`). When a season pack is grabbed:

1. A single `download_mapping` is created with `is_season_pack = 1`
2. `queue_media` entries are created for each episode file as they are identified
3. Individual episode status should transition to `downloading` when their `queue_media` entry is created
4. Each episode transitions independently to `available`/`upgradable`/`failed` as its file completes import
5. If any episode file fails, only that episode gets `failed` status—other episodes in the pack proceed normally

## Quality Profile Changes

When a quality profile's cutoff or `upgrades_enabled` flag changes, immediately recalculate status for all affected leaf nodes.

**Single-version items** (media-level status is the leaf):
```
For each movie/episode using the changed profile:
    if status IN ('available', 'upgradable'):
        if file_quality < new_cutoff AND upgrades_enabled:
            set status = 'upgradable'
        else:
            set status = 'available'
```

**Multi-version items** (slot assignments are the leaves):
```
For each slot assignment where the slot's quality_profile_id matches the changed profile:
    if slot.status IN ('available', 'upgradable'):
        if file_quality < new_cutoff AND upgrades_enabled:
            set slot.status = 'upgradable'
        else:
            set slot.status = 'available'
    Recompute media-level cached status from all monitored slots
```

Note: Each slot has its own `quality_profile_id` (inherited from the `version_slots` table), so a profile change only affects slots using that specific profile. Two slots on the same movie can use different profiles.

This recalculation is triggered synchronously when the quality profile is updated via the API.

## Multi-Version Slot Integration

SlipStream supports multi-version slots (Primary, Secondary, Tertiary), where each slot can hold a separate file with its own quality profile. When multi-version is enabled, **slot assignments are the leaf nodes** for status tracking. Each slot operates independently—its own status, its own download tracking, its own failure state. The media-level `status` is a cached aggregate recomputed from monitored slots.

### Dual-Path Status Model

| Multi-version state | Status leaf node | Media `status` column role |
|---|---|---|
| **Disabled** | Movie/episode directly | Source of truth (as specified in the rest of this doc) |
| **Enabled, no slot assignments exist yet** | Movie/episode directly | Source of truth until first slot interaction |
| **Enabled, slot assignments exist** | Slot assignments | Cached aggregate, recomputed on any slot status change |

Slot assignments are **lazy**—created only when a file is imported targeting a slot, or when a user toggles slot monitoring. Most items won't have assignment rows until explicitly interacted with.

### Per-Slot Status Fields

Each slot assignment row carries the full set of status fields (see [Schema Changes](#slot-assignment-tables-multi-version)):

| Field | Type | Purpose |
|-------|------|---------|
| `status` | enum | Same 6-value enum as media items |
| `active_download_id` | text | Download client item ID for this specific slot |
| `status_message` | text | Failure reason for this specific slot |

These are independent per slot. Slot 1 can be `available` while Slot 2 is `downloading`—each has its own `active_download_id`.

### Cached Aggregate Computation

After any slot-level status change, recompute the media item's cached `status` from all **monitored** slot assignments:

| Condition | Cached Status |
|-----------|---------------|
| Release date hasn't passed | `unreleased` (regardless of slots) |
| Any monitored slot has `status = 'downloading'` | `downloading` |
| Any monitored slot has `status = 'failed'` | `failed` |
| Any monitored slot is empty (`status = 'missing'`) | `missing` |
| All monitored slots have files, at least one below cutoff (`upgradable`) | `upgradable` |
| All monitored slots have files at or above cutoff (`available`) | `available` |

Priority order (highest wins): `unreleased` > `downloading` > `failed` > `missing` > `upgradable` > `available`.

If no monitored slot assignments exist (all disabled or none created), fall back to the media item's own status.

### Status Transitions on Slots

All transitions from the [Transition Details](#transition-details) table apply identically at the slot level. The `download_mappings.target_slot_id` identifies which slot assignment row to update when a download event occurs.

**`unreleased` ↔ `missing`:** These are global to the media item (release/air date is not per-slot). When the scheduler or metadata refresh changes release state, it updates **all** slot assignments for that item simultaneously, plus the media item's own status.

**All other transitions** (`downloading`, `failed`, `available`, `upgradable`) operate on the individual slot targeted by the action. After each, recompute the cached aggregate.

### Auto-Search with Slots

The existing slot-aware auto-search (`GetMovieSlotsNeedingSearch`, `GetEpisodeSlotsNeedingSearch`) already searches per-slot using each slot's quality profile. With per-slot status:

1. **Eligibility check:** Media-level cached `status` determines if the item is worth examining (quick filter: `status IN ('missing', 'upgradable')`)
2. **Slot-level search:** For each monitored slot with `status = 'missing'` or `status = 'upgradable'`, search independently using that slot's quality profile
3. **Independent outcomes:** Each slot's search result is grabbed independently, creating separate download mappings with the appropriate `target_slot_id`

Example: Movie with Slot 1 (`available` at 1080p) and Slot 2 (`missing`). Media cached status = `missing`. Auto-search examines the movie, sees Slot 2 needs a file, searches using Slot 2's quality profile. Slot 1 is untouched.

### Manual File Removal with Slots

When a file is removed from a specific slot via the UI:
1. Set that slot's `status = 'missing'` and `monitored = false`
2. Recompute the media item's cached status from remaining monitored slots
3. If no monitored slots remain with files, the media cached status becomes `missing`

### Manual Retry with Slots

When a user retries a `failed` slot:
1. Check `file_id` on the slot assignment:
   - `file_id` is set → set slot `status = 'upgradable'` (old file survived the failed upgrade)
   - `file_id` is NULL → set slot `status = 'missing'`
2. Clear `status_message` and `active_download_id`
3. Reset `autosearch_status.failure_count` for the item+slot combination
4. Recompute cached aggregate
5. Optionally trigger immediate search for that slot

## Computed Status for Series and Seasons

Series and seasons do **not** store status in the database. Instead, their status is computed from their child episodes and returned inline in API responses.

### Status Counts Structure

```json
{
  "statusCounts": {
    "unreleased": 2,
    "missing": 3,
    "downloading": 1,
    "failed": 0,
    "upgradable": 1,
    "available": 5,
    "total": 12
  }
}
```

### Computation

**Season statusCounts:** Aggregate `status` values of all episodes in the season.

**Series statusCounts:** Aggregate `status` values of all episodes across all seasons (excluding specials/season 0 unless monitored).

### API Response Shape

**Series object:**
```json
{
  "id": 1,
  "title": "Breaking Bad",
  "productionStatus": "ended",
  "monitored": true,
  "statusCounts": {
    "unreleased": 0,
    "missing": 5,
    "downloading": 0,
    "failed": 0,
    "upgradable": 3,
    "available": 54,
    "total": 62
  },
  "seasons": [
    {
      "seasonNumber": 1,
      "monitored": true,
      "statusCounts": {
        "unreleased": 0,
        "missing": 0,
        "downloading": 0,
        "failed": 0,
        "upgradable": 1,
        "available": 6,
        "total": 7
      }
    }
  ]
}
```

**Movie object:**
```json
{
  "id": 1,
  "title": "Inception",
  "status": "available",
  "monitored": true,
  "statusMessage": null,
  "activeDownloadId": null,
  "releaseDate": "2010-07-16",
  "physicalReleaseDate": "2010-12-07"
}
```

**Episode object:**
```json
{
  "id": 42,
  "title": "Ozymandias",
  "status": "available",
  "monitored": true,
  "statusMessage": null,
  "activeDownloadId": null,
  "airDate": "2013-09-15T21:00:00-04:00"
}
```

### Frontend Display String Computation

The frontend computes display strings from statusCounts. Examples:

| Counts | Display |
|--------|---------|
| All available | "Available" |
| All unreleased | "Unreleased" |
| Mix with some available | "Seasons 1-3 Available" (computed from per-season counts) |
| Some downloading | "3 Downloading" |
| Some missing + available | "5/12 Episodes" |

This logic lives in a frontend utility function, not the backend.

## Removed API Fields

The following fields are removed from API responses:

| Entity | Removed Field | Replacement | Notes |
|--------|---------------|-------------|-------|
| Movie | `released` | `status !== 'unreleased'` | DB column removed |
| Movie | `availabilityStatus` | `status` field directly | DB column removed |
| Movie | `hasFile` | `status in ['available', 'upgradable']` | Computed field (Go struct only, not a DB column) |
| Series | `status` (production) | `productionStatus` | DB column renamed |
| Series | `released` | Derive from `statusCounts.unreleased === 0` | DB column removed |
| Series | `availabilityStatus` | `statusCounts` object | DB column removed |
| Series | `episodeFileCount` | `statusCounts.available + statusCounts.upgradable` | Computed field |
| Series | `episodeCount` | `statusCounts.total` | Computed field |
| Season | `released` | Derive from `statusCounts.unreleased === 0` | DB column removed |
| Season | `episodeFileCount` | `statusCounts.available + statusCounts.upgradable` | Computed field |
| Season | `episodeCount` | `statusCounts.total` | Computed field |
| Episode | `released` | `status !== 'unreleased'` | DB column removed |
| Episode | `hasFile` | `status in ['available', 'upgradable']` | Computed field (Go struct only, not a DB column) |

**Preserved fields** (unchanged by this spec): `sizeOnDisk` on Movie, Series, and Season remains as a computed field. `formatType` on Series is unchanged.

## Auto-Search Integration

### Search Eligibility

The auto-search system uses the media-level cached status as a **fast filter** to identify items worth examining, then delegates to slot-level logic for the actual search decisions.

**Media-level eligibility queries** (used to build the candidate list):

```sql
-- Missing movies (replace ListMissingMovies)
SELECT m.* FROM movies m
WHERE m.status IN ('missing', 'failed')
  AND m.monitored = 1;
-- Note: 'failed' included so slot-level check can find slots that are
-- 'missing' even when another slot is 'failed' (making the aggregate 'failed')

-- Upgrade movies (replace ListMovieUpgradeCandidates)
SELECT m.* FROM movies m
WHERE m.status = 'upgradable'
  AND m.monitored = 1;

-- Missing episodes (replace ListMissingEpisodes)
SELECT e.* FROM episodes e
JOIN series s ON e.series_id = s.id
JOIN seasons sea ON e.series_id = sea.series_id AND e.season_number = sea.season_number
WHERE e.status IN ('missing', 'failed')
  AND s.monitored = 1
  AND sea.monitored = 1
  AND e.monitored = 1;

-- Upgrade episodes (replace ListEpisodeUpgradeCandidates)
SELECT e.* FROM episodes e
JOIN series s ON e.series_id = s.id
JOIN seasons sea ON e.series_id = sea.series_id AND e.season_number = sea.season_number
WHERE e.status = 'upgradable'
  AND s.monitored = 1
  AND sea.monitored = 1
  AND e.monitored = 1;
```

**Slot-level search** (when multi-version is enabled): After the media-level filter identifies candidates, the existing `GetMovieSlotsNeedingSearch` / `GetEpisodeSlotsNeedingSearch` functions examine each monitored slot assignment and only search slots where `slot.status = 'missing'` or `slot.status = 'upgradable'`. Slots with `downloading`, `failed`, or `available` status are skipped.

**Single-version search** (when multi-version is disabled): The media-level queries above are sufficient. No slot-level check needed.

### Backoff Integration

| Status | Backoff Behavior |
|--------|-----------------|
| `missing` | Normal backoff via `autosearch_status.failure_count`. Incremented on each failed search attempt. |
| `failed` | **Excluded from auto-search entirely.** User must manually retry, which resets status to `missing` and clears `failure_count`. |
| `downloading` | Excluded from auto-search (already being handled). |
| `upgradable` | Same backoff rules as `missing`. Uses separate failure tracking for upgrade searches (see below). |
| `unreleased` / `available` | Not eligible for search. |

### Upgrade Backoff Tracking

The current `autosearch_status` table uses `UNIQUE(item_type, item_id)` to track backoff per item. Upgrade searches need separate failure tracking so that a "missing" search hitting backoff doesn't prevent upgrade searches (and vice versa). Add a `search_type` column:

```sql
ALTER TABLE autosearch_status ADD COLUMN search_type TEXT NOT NULL DEFAULT 'missing'
    CHECK(search_type IN ('missing', 'upgrade'));
-- Update unique constraint: UNIQUE(item_type, item_id, search_type)
```

This allows independent backoff counters for missing vs upgrade searches on the same item.

### Manual Retry Flow

When a user triggers a manual retry on a `failed` item (or a specific `failed` slot when multi-version is enabled):
1. **Check file presence** on the leaf (slot assignment's `file_id`, or media item's file join):
   - File exists → set leaf status to `upgradable` (failed upgrade attempt, old file still on disk)
   - No file → set leaf status to `missing` (failed initial download)
2. Clear `status_message` and `active_download_id`
3. Reset `autosearch_status.failure_count = 0` for the item
4. Recompute media-level cached status (if slot-level retry)
5. Optionally trigger an immediate search (same as current "Search" button behavior)

## Portal Request Integration

### Auto-Mirror After Approval

When a portal request is approved and linked to a media item (`media_id` set), the request status automatically mirrors the media item's status transitions.

### Extended StatusTracker Events

Add new event handlers to the existing `StatusTracker`:

| Event | Trigger | Request Status Change |
|-------|---------|----------------------|
| `OnDownloadStarted(mediaType, mediaID)` | Media status → `downloading` | Request status → `downloading` |
| `OnDownloadFailed(mediaType, mediaID)` | Media status → `failed` | Request status → `failed` |
| `OnMovieAvailable(movieID)` | (existing) Media status → `available` | Request status → `available` |
| `OnEpisodeAvailable(episodeID)` | (existing) Media status → `available` | Request status → `available` (if all requested content is available) |

### Request Status Values (Updated)

| Status | Meaning |
|--------|---------|
| `pending` | Awaiting admin approval |
| `approved` | Approved, not yet downloading |
| `denied` | Rejected by admin |
| `downloading` | Linked media is downloading |
| `failed` | Linked media download/import failed |
| `available` | Linked media is available |

The `cancelled` status remains frontend-only (request is deleted from DB on cancellation).

### Season/Series Request Status Rules

For requests spanning multiple episodes (season or series requests):

| Transition | Trigger |
|-----------|---------|
| Request → `downloading` | **First** linked episode transitions to `downloading` |
| Request → `failed` | **All** linked episodes are `failed` (partial failure keeps request as `downloading`) |
| Request → `available` | **All** requested content is available (existing `OnEpisodeAvailable` + `AreSeasonsComplete` logic) |

This means a season request shows `downloading` as soon as any episode starts downloading, and stays in that state until everything completes or everything fails.

## Health Alerts for Disappeared Files

### Category

Use the existing `storage` health category (non-binary, supports OK/Warning/Error).

**Note:** The `rootFolders` category is binary (only OK or Error)—`SetWarning()` is a no-op for binary categories. The `storage` category supports warnings and is semantically appropriate for disappeared-file alerts.

### Detection

During periodic library scans (or file system watches), when a previously-existing file is no longer found on disk:

1. Set media item `status = 'missing'` (keep `monitored` unchanged)
2. Register a health alert:
   ```go
   healthService.SetWarning(
       health.CategoryStorage,
       fmt.Sprintf("root_%d", rootFolderID),
       fmt.Sprintf("Files missing from %s: %d movie(s), %d episode(s) no longer found on disk", path, movieCount, episodeCount),
   )
   ```
3. The alert clears automatically on the next scan if no files are missing

### Notification

Uses existing health notification infrastructure:
- `DispatchHealthIssue()` sends notification when files first go missing
- `DispatchHealthRestored()` sends notification when issue resolves

## History Integration

Status transitions should be recorded in the existing history system. Add a new event type:

| Event Type | Trigger | Data |
|-----------|---------|------|
| `status_changed` | Any media status transition | `{ "from": "missing", "to": "downloading", "reason": "Auto-search grab" }` |

Record entries for significant transitions only (not scheduler-driven `unreleased → missing` which would generate noise). Significant transitions:
- `missing → downloading` (grab)
- `downloading → available` / `downloading → upgradable` (import success)
- `downloading → failed` (import/download failure)
- `available → missing` / `upgradable → missing` (file disappeared or removed)
- `failed → missing` (manual retry)

The existing `grabbed`, `imported`, `import_failed` event types already cover some of these—avoid double-recording. The `status_changed` event is for transitions not covered by existing events (e.g., file disappearance, manual retry reset).

## WebSocket Events for Status Changes

Existing `movie:updated` and `series:updated` WebSocket events already fire when status changes. These carry the full updated entity including the new `status` field, which is sufficient for the frontend to reactively update badges and counts.

No new event types are needed, but ensure the following are broadcast:
- `movie:updated` — on any movie status transition
- `series:updated` — on any episode status transition (so series cards update statusCounts)
- `health:updated` — on disappeared file health alerts (already wired)

## Frontend Changes

### New Unified Badge Component

Replace both `StatusBadge` and `AvailabilityBadge` with a single `MediaStatusBadge` component.

**File:** `web/src/components/media/MediaStatusBadge.tsx`

```tsx
type MediaStatus = 'unreleased' | 'missing' | 'downloading' | 'failed' | 'upgradable' | 'available'

interface MediaStatusBadgeProps {
  status: MediaStatus
  className?: string
}
```

### Badge Color Mapping

| Status | Color | Variant | Icon |
|--------|-------|---------|------|
| `unreleased` | Amber | outline | Clock |
| `missing` | Red | destructive | AlertCircle |
| `downloading` | Blue | default | ArrowDownCircle |
| `failed` | Red/Dark | destructive + border | XCircle |
| `upgradable` | Yellow | outline | ArrowUpCircle |
| `available` | Green | default | CheckCircle |

### Production Status Badge (Series Only)

A separate `ProductionStatusBadge` component for series production status:

| Production Status | Color | Variant |
|-------------------|-------|---------|
| `continuing` | Green | default |
| `ended` | Gray | secondary |
| `upcoming` | Amber | outline |

### Series/Season Status Display

Replace the single `availabilityStatus` string badge with a richer display computed from `statusCounts`:

**Series card:** Show a compact summary (e.g., "8/10 episodes" or a mini progress bar).

**Series detail page:** Show per-season breakdown with statusCounts rendered as segmented bars or count badges.

**Season detail page:** Show per-episode status list with individual `MediaStatusBadge` per episode.

### Components to Remove

- `web/src/components/media/StatusBadge.tsx` → replaced by `MediaStatusBadge`
- `web/src/components/media/AvailabilityBadge.tsx` → replaced by `MediaStatusBadge` + statusCounts display

### Type Changes

**`web/src/types/movie.ts`:**
```typescript
interface Movie {
  status: 'unreleased' | 'missing' | 'downloading' | 'failed' | 'upgradable' | 'available'
  monitored: boolean
  statusMessage: string | null
  activeDownloadId: string | null
  releaseDate?: string
  physicalReleaseDate?: string
  // removed: released, availabilityStatus, hasFile
}
```

**`web/src/types/series.ts`:**
```typescript
interface StatusCounts {
  unreleased: number
  missing: number
  downloading: number
  failed: number
  upgradable: number
  available: number
  total: number
}

interface Series {
  productionStatus: 'continuing' | 'ended' | 'upcoming'
  monitored: boolean
  statusCounts: StatusCounts
  // removed: status (was production), released, availabilityStatus
}

interface Season {
  monitored: boolean
  statusCounts: StatusCounts
  // removed: released, episodeFileCount (derivable from statusCounts)
}

interface Episode {
  status: 'unreleased' | 'missing' | 'downloading' | 'failed' | 'upgradable' | 'available'
  monitored: boolean
  statusMessage: string | null
  activeDownloadId: string | null
  airDate?: string  // Now includes time: ISO 8601 datetime
  // removed: released, hasFile
}
```

**`web/src/types/portal.ts`:**
```typescript
export type RequestStatus = 'pending' | 'approved' | 'denied' | 'downloading' | 'failed' | 'available' | 'cancelled'
```

### Frontend Utility: Status Counts Display

New utility function for computing display strings from statusCounts:

**`web/src/lib/formatters.ts`** (or new file):

```typescript
function formatStatusSummary(counts: StatusCounts): string
function formatSeasonAvailability(seasons: Season[]): string
```

Logic:
- All `available` + `upgradable` → "Available"
- All `unreleased` → "Unreleased"
- Mixed → Per-season range formatting (e.g., "Seasons 1-3 Available") using per-season statusCounts
- Any `failed` → Include failure count (e.g., "2 Failed")

## Go Type Changes

### Movies

**`internal/library/movies/movie.go`:**
```go
type Movie struct {
    ID                  int64      `json:"id"`
    // ... metadata fields unchanged ...
    Status              string     `json:"status"`              // unified status (cached aggregate when multi-version)
    Monitored           bool       `json:"monitored"`
    StatusMessage       *string    `json:"statusMessage"`       // NEW: failure reason (single-version only)
    ActiveDownloadID    *string    `json:"activeDownloadId"`    // NEW: download client item ID (single-version only)
    ReleaseDate         *time.Time `json:"releaseDate,omitempty"`
    PhysicalReleaseDate *time.Time `json:"physicalReleaseDate,omitempty"`
    SizeOnDisk          int64      `json:"sizeOnDisk,omitempty"` // unchanged
    MovieFiles          []MovieFile `json:"movieFiles,omitempty"` // unchanged
    // Removed: Released, AvailabilityStatus, HasFile
}
```

### Series

**`internal/library/tv/series.go`:**
```go
type StatusCounts struct {
    Unreleased  int `json:"unreleased"`
    Missing     int `json:"missing"`
    Downloading int `json:"downloading"`
    Failed      int `json:"failed"`
    Upgradable  int `json:"upgradable"`
    Available   int `json:"available"`
    Total       int `json:"total"`
}

type Series struct {
    // ... metadata fields unchanged ...
    ProductionStatus string       `json:"productionStatus"` // renamed from Status
    Monitored        bool         `json:"monitored"`
    StatusCounts     StatusCounts `json:"statusCounts"`     // NEW: computed
    SizeOnDisk       int64        `json:"sizeOnDisk,omitempty"` // unchanged
    FormatType       string       `json:"formatType,omitempty"` // unchanged
    // Removed: Status (was production), Released, AvailabilityStatus, EpisodeCount, EpisodeFileCount
}

type Season struct {
    // ... metadata fields unchanged ...
    Monitored    bool         `json:"monitored"`
    StatusCounts StatusCounts `json:"statusCounts"` // NEW: computed
    SizeOnDisk   int64        `json:"sizeOnDisk,omitempty"` // unchanged
    // Removed: Released, EpisodeCount, EpisodeFileCount (derivable from statusCounts)
}

type Episode struct {
    // ... metadata fields unchanged ...
    Status           string     `json:"status"`              // NEW: unified status (cached aggregate when multi-version)
    Monitored        bool       `json:"monitored"`
    StatusMessage    *string    `json:"statusMessage"`       // NEW (single-version only)
    ActiveDownloadID *string    `json:"activeDownloadId"`    // NEW (single-version only)
    AirDate          *time.Time `json:"airDate,omitempty"`   // now datetime
    // Removed: Released, HasFile
}
```

### Slot Assignment Status

**`internal/library/slots/status.go`:**
```go
type SlotStatus struct {
    SlotID           int64   `json:"slotId"`
    SlotNumber       int     `json:"slotNumber"`
    SlotName         string  `json:"slotName"`
    Monitored        bool    `json:"monitored"`
    Status           string  `json:"status"`                  // NEW: unified status per slot
    StatusMessage    *string `json:"statusMessage,omitempty"`  // NEW: per-slot failure reason
    ActiveDownloadID *string `json:"activeDownloadId,omitempty"` // NEW: per-slot download tracking
    FileID           *int64  `json:"fileId,omitempty"`
    CurrentQuality   string  `json:"currentQuality,omitempty"`
    CurrentQualityID *int64  `json:"currentQualityId,omitempty"`
    ProfileCutoff    int     `json:"profileCutoff"`
    // Removed: HasFile, NeedsUpgrade, IsMissing (derivable from status)
}
```

`HasFile` is replaced by `status in ['available', 'upgradable']`. `NeedsUpgrade` is replaced by `status == 'upgradable'`. `IsMissing` is replaced by `status == 'missing'`.

## Implementation Order

### Phase 1: Database Migration
1. Write SQLite migration (table recreation for constraint changes, preserve `format_type`)
2. Add `status`, `active_download_id`, `status_message` to slot assignment tables
3. Add `search_type` column to `autosearch_status` table
4. Transform existing data per migration rules above (movies, episodes, slot assignments)
5. Run post-migration quality recalculation for `upgradable` status (both media-level and slot-level)

### Phase 2: Backend Core
6. Update Go types (Movie, Episode, Series, Season, StatusCounts, SlotStatus)
7. Update sqlc queries and regenerate
8. Implement dual-path status write logic: write to slot assignment when multi-version + assignment exists, else write to media item
9. Implement cached aggregate recomputation (slot statuses → media status)
10. Update movie service (status transitions, file add/remove with quality evaluation)
11. Update episode service (status transitions, file add/remove with quality evaluation)
12. Add StatusCounts computation queries and service methods
13. Rename series `status` → `productionStatus` in service layer

### Phase 3: Status Lifecycle
14. Update availability service → status refresh service (hourly, datetime support, air time fallback)
15. Update download completion checker (set downloading/failed/available/upgradable at leaf level)
16. Update import pipeline to evaluate quality on import (not always `available`), target correct slot
17. Add season pack support: per-episode status transitions via `queue_media`
18. Add quality profile change → status recalculation trigger (slot-aware: each slot uses its own profile)
19. Add manual file removal → unmonitor behavior (slot-level when multi-version)
20. Add disappeared file detection → health alert (use `storage` category), update slot status if applicable

### Phase 4: Auto-Search ✅
21. ✅ Update search eligibility queries (media-level cached status as fast filter)
22. ✅ Update slot-level search to use `slot.status` instead of computed `IsMissing`/`NeedsUpgrade`
23. ✅ Update backoff integration for `failed` status
24. ✅ Add separate upgrade backoff tracking (`search_type` column)
25. ✅ Add manual retry endpoint (failed → missing + reset, slot-aware)

### Phase 5: Portal Integration
26. ✅ Add `failed` to request status constraint
27. ✅ Extend StatusTracker with OnDownloadStarted, OnDownloadFailed
28. ✅ Wire media status changes to request status sync
29. ✅ Implement season/series request status rules (first downloading, all failed, all available)

### Phase 6: History & Events
30. ✅ Add `status_changed` history event type for transitions not covered by existing events
31. ✅ Ensure `movie:updated` / `series:updated` WebSocket events fire on all status transitions (including slot-triggered)

### Phase 7: Frontend
32. ✅ Create `MediaStatusBadge` component
33. ✅ Create `ProductionStatusBadge` component
34. ✅ Update TypeScript types (Movie, Episode, Series, Season, SlotStatus)
35. ✅ Update API client layer
36. ✅ Add statusCounts display utilities
37. ✅ Update MovieCard, SeriesCard, detail pages
38. ✅ Update slot status UI to show per-slot `MediaStatusBadge` with independent states
39. ✅ Remove old StatusBadge, AvailabilityBadge components
40. ✅ Update portal request status display (add `failed` state)

### Phase 8: Cleanup
41. ✅ Remove deprecated columns from queries/types
42. ✅ Remove old availability service code
43. ✅ Remove `HasFile`, `NeedsUpgrade`, `IsMissing` from SlotStatus (replaced by `status`)
44. ✅ Update CLAUDE.md / AGENTS.md documentation
