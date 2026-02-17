# Library Import Spec

Mappings for importing Radarr (movies) and Sonarr (TV series) libraries into SlipStream.

> **Scope:** Only fields that have a corresponding SlipStream data element are mapped. Sonarr/Radarr fields with no SlipStream equivalent (e.g., ratings, genres, tags, custom formats, collection info) are omitted.

---

## 1. Data Access

**Primary mechanism: Direct SQLite reads.** Both Sonarr (`sonarr.db`) and Radarr (`radarr.db`) use SQLite by default, same as SlipStream. The user browses to the database file via the folder browser — no API keys needed.

- Default locations: `~/.config/Sonarr/sonarr.db`, `~/.config/Radarr/radarr.db`
- Database is opened **read-only** to avoid any risk to the source application
- Works even if Sonarr/Radarr is not running (common during migration)

**Fallback: API-based reads.** For users running Sonarr/Radarr with PostgreSQL or on a remote machine/container, provide an API-based import path using the v3 REST API with an `X-Api-Key` header.

| | SQLite (primary) | API (fallback) |
|---|---|---|
| **User provides** | Path to `.db` file | URL + API key |
| **Radarr caveat** | Must join `Movies` + `MovieMetadata` tables (split schema) | API joins automatically |
| **Works when app is stopped** | Yes | No |
| **Works for PostgreSQL users** | No | Yes |
| **Works across machines/containers** | No | Yes |

The field mappings in §2–§3 are described in terms of the API field names for clarity. The same data exists in the SQLite tables — column names are snake_case equivalents. Implementation should define a common reader interface with SQLite and API backends.

---

## 2. Movies (Radarr → SlipStream)

### 2.1 Movie Entity

| # | SlipStream Field | Type | Radarr Source Field | Type | Transformation |
|---|---|---|---|---|---|
| 1 | `title` | TEXT NOT NULL | `title` | string | Direct |
| 2 | `sort_title` | TEXT NOT NULL | `sortTitle` | string | Direct |
| 3 | `year` | INTEGER | `year` | int | Direct |
| 4 | `tmdb_id` | INTEGER UNIQUE | `tmdbId` | int | Direct |
| 5 | `imdb_id` | TEXT | `imdbId` | string | Direct |
| 6 | `overview` | TEXT | `overview` | string | Direct |
| 7 | `runtime` | INTEGER | `runtime` | int | Direct (both in minutes) |
| 8 | `path` | TEXT | `path` | string | Direct (absolute path) |
| 9 | `root_folder_id` | INTEGER FK | `rootFolderPath` | string | **Lookup:** Match `rootFolderPath` to an existing SlipStream root folder by path, or prompt user to map/create |
| 10 | `quality_profile_id` | INTEGER FK | `qualityProfileId` + quality profile name | int | **Lookup:** User maps source profile to SlipStream profile (see §4) |
| 11 | `monitored` | INTEGER (0/1) | `monitored` | bool | `true` → `1`, `false` → `0` |
| 12 | `status` | TEXT | — | — | **Derived by SlipStream:** See §2.3 |
| 13 | `release_date` | DATE | `digitalRelease` | DateTime? | Direct (format to DATE) |
| 14 | `physical_release_date` | DATE | `physicalRelease` | DateTime? | Direct (format to DATE) |
| 15 | `theatrical_release_date` | DATETIME | `inCinemas` | DateTime? | Direct |
| 16 | `studio` | TEXT | `studio` | string | Direct |
| 17 | `tvdb_id` | INTEGER | — | — | Not available in Radarr |
| 18 | `content_rating` | TEXT | `certification` | string | Direct |
| 19 | `added_at` | DATETIME | `added` | DateTime | Direct |
| 20 | `updated_at` | DATETIME | — | — | Set to import timestamp |

### 2.2 Movie File Entity

| # | SlipStream Field | Type | Radarr Source Field | Type | Transformation |
|---|---|---|---|---|---|
| 1 | `movie_id` | INTEGER FK | — | — | Set to the newly-created SlipStream movie ID |
| 2 | `path` | TEXT NOT NULL | `path` | string | Direct (absolute path) |
| 3 | `size` | INTEGER | `size` | long | Direct (both in bytes) |
| 4 | `quality_id` | INTEGER | `quality.quality.id` + `quality.quality.name` | int, string | **Map:** See §2.4 Quality ID Mapping |
| 5 | `video_codec` | TEXT | `mediaInfo.videoCodec` | string | Direct |
| 6 | `audio_codec` | TEXT | `mediaInfo.audioCodec` | string | Direct |
| 7 | `resolution` | TEXT | `mediaInfo.resolution` | string | Direct (Radarr format: `"1920x1080"`) |
| 8 | `audio_channels` | TEXT | `mediaInfo.audioChannels` | decimal | Convert decimal to string (e.g., `5.1` → `"5.1"`) |
| 9 | `dynamic_range` | TEXT | `mediaInfo.videoDynamicRange` | string | Direct (e.g., `"HDR"`, `"HDR10"`) |
| 10 | `original_path` | TEXT | `originalFilePath` | string | Direct |
| 11 | `original_filename` | TEXT | `originalFilePath` | string | **Derive:** Extract filename from `originalFilePath` using path.Base() |
| 12 | `imported_at` | DATETIME | `dateAdded` | DateTime | Direct |
| 13 | `created_at` | DATETIME | `dateAdded` | DateTime | Direct |
| 14 | `slot_id` | INTEGER FK | — | — | Set by slots service if multi-version enabled (see §8) |

### 2.3 Movie Status Derivation

Status is **not** mapped from Radarr. SlipStream's own logic handles it entirely:

1. **On movie creation** (`movies.Service.Create`): `isMovieReleased()` checks if any release date (digital, physical, or theatrical+90 days) is in the past.
   - Released → `"missing"`
   - Not released → `"unreleased"`
2. **On file attachment** (`movies.Service.AddFile`): `profile.StatusForQuality(qualityID)` checks the file's quality against the assigned quality profile's cutoff.
   - At or above cutoff → `"available"`
   - Below cutoff with upgrades enabled → `"upgradable"`
   - Below cutoff with upgrades disabled → `"available"`

**Import approach:** Create the movie with its release dates (mapped in rows 13-15), then call `AddFile()` for each movie that has a file in Radarr (`hasFile == true`). SlipStream will derive the correct status at each step. Skip movies with Radarr status `deleted` (-1).

### 2.4 Quality ID Mapping (Radarr → SlipStream)

Quality IDs share the same naming convention but use different numeric IDs. Map by **name**:

| Radarr Quality (ID → Name) | SlipStream Quality (ID → Name) |
|---|---|
| 1 → SDTV | 1 → SDTV |
| 2 → DVD | 2 → DVD |
| 8 → WEBDL-480p | — (no 480p WEBDL; nearest: 3 → WEBRip-480p) |
| 12 → WEBRip-480p | 3 → WEBRip-480p |
| 4 → HDTV-720p | 4 → HDTV-720p |
| 5 → WEBDL-720p | 6 → WEBDL-720p |
| 14 → WEBRip-720p | 5 → WEBRip-720p |
| 6 → Bluray-720p | 7 → Bluray-720p |
| 9 → HDTV-1080p | 8 → HDTV-1080p |
| 3 → WEBDL-1080p | 10 → WEBDL-1080p |
| 15 → WEBRip-1080p | 9 → WEBRip-1080p |
| 7 → Bluray-1080p | 11 → Bluray-1080p |
| 30 → Remux-1080p | 12 → Remux-1080p |
| 16 → HDTV-2160p | 13 → HDTV-2160p |
| 18 → WEBDL-2160p | 15 → WEBDL-2160p |
| 17 → WEBRip-2160p | 14 → WEBRip-2160p |
| 19 → Bluray-2160p | 16 → Bluray-2160p |
| 31 → Remux-2160p | 17 → Remux-2160p |
| 0 → Unknown | — (no match; omit `quality_id`) |
| 20 → Bluray-480p | — (no match; nearest: 2 → DVD) |
| 21 → Bluray-576p | — (no match; nearest: 2 → DVD) |
| 10 → Raw-HD | — (no match; omit) |
| 22 → BR-DISK | — (no match; omit) |
| 23 → DVD-R | 2 → DVD |
| 24 → WORKPRINT | — (no match; omit) |
| 25 → CAM | — (no match; omit) |
| 26 → TELESYNC | — (no match; omit) |
| 27 → TELECINE | — (no match; omit) |
| 28 → DVDSCR | — (no match; omit) |
| 29 → REGIONAL | — (no match; omit) |

**Strategy:** Map by quality name first. For unmatched qualities, fall back to using `MatchQuality(source, resolution)` from the scoring package.

---

## 3. TV Series (Sonarr → SlipStream)

### 3.1 Series Entity

| # | SlipStream Field | Type | Sonarr Source Field | Type | Transformation |
|---|---|---|---|---|---|
| 1 | `title` | TEXT NOT NULL | `title` | string | Direct |
| 2 | `sort_title` | TEXT NOT NULL | `sortTitle` | string | Direct |
| 3 | `year` | INTEGER | `year` | int | Direct |
| 4 | `tvdb_id` | INTEGER UNIQUE | `tvdbId` | int | Direct |
| 5 | `tmdb_id` | INTEGER | `tmdbId` | int | Direct (added in Sonarr v4) |
| 6 | `imdb_id` | TEXT | `imdbId` | string | Direct |
| 7 | `overview` | TEXT | `overview` | string | Direct |
| 8 | `runtime` | INTEGER | `runtime` | int | Direct (both in minutes) |
| 9 | `path` | TEXT | `path` | string | Direct (absolute path) |
| 10 | `root_folder_id` | INTEGER FK | `rootFolderPath` | string | **Lookup:** Match `rootFolderPath` to an existing SlipStream root folder by path, or prompt user to map/create |
| 11 | `quality_profile_id` | INTEGER FK | `qualityProfileId` + profile name | int | **Lookup:** User maps source profile to SlipStream profile (see §4) |
| 12 | `monitored` | INTEGER (0/1) | `monitored` | bool | `true` → `1`, `false` → `0` |
| 13 | `season_folder` | INTEGER (0/1) | `seasonFolder` | bool | `true` → `1`, `false` → `0` |
| 14 | `production_status` | TEXT | `status` | string | Direct (Sonarr API serializes as `"continuing"`, `"ended"`, `"upcoming"` — same values). Skip `"deleted"` series. |
| 15 | `network` | TEXT | `network` | string | Direct |
| 16 | `network_logo_url` | TEXT | — | — | Skip — metadata refresh provides via `GetSeriesLogoURL` |
| 17 | `format_type` | TEXT | `seriesType` | string | Direct (Sonarr API serializes as `"standard"`, `"daily"`, `"anime"` — same values) |
| 18 | `added_at` | DATETIME | `added` | DateTime | Direct |
| 19 | `updated_at` | DATETIME | — | — | Set to import timestamp |

### 3.2 Season Entity

| # | SlipStream Field | Type | Sonarr Source Field (embedded in seasons) | Type | Transformation |
|---|---|---|---|---|---|
| 1 | `series_id` | INTEGER FK | — | — | Set to the newly-created SlipStream series ID |
| 2 | `season_number` | INTEGER | `seasonNumber` | int | Direct |
| 3 | `monitored` | INTEGER (0/1) | `monitored` | bool | `true` → `1`, `false` → `0` |
| 4 | `overview` | TEXT | — | — | Skip — metadata refresh provides via `GetSeriesSeasons` |
| 5 | `poster_url` | TEXT | — | — | Skip — metadata refresh provides via `GetSeriesSeasons` |

### 3.3 Episode Entity

| # | SlipStream Field | Type | Sonarr Source Field | Type | Transformation |
|---|---|---|---|---|---|
| 1 | `series_id` | INTEGER FK | `seriesId` | int | **Remap:** Map Sonarr series ID to newly-created SlipStream series ID |
| 2 | `season_number` | INTEGER | `seasonNumber` | int | Direct |
| 3 | `episode_number` | INTEGER | `episodeNumber` | int | Direct |
| 4 | `title` | TEXT (nullable) | `title` | string | Direct (wrap in `sql.NullString`) |
| 5 | `overview` | TEXT | `overview` | string | Direct |
| 6 | `air_date` | DATETIME | `airDateUtc` | DateTime? | Direct (prefer `airDateUtc` over `airDate` string) |
| 7 | `monitored` | INTEGER (0/1) | `monitored` | bool | `true` → `1`, `false` → `0` |
| 8 | `status` | TEXT | — | — | **Derived by SlipStream:** See §3.5 |

### 3.4 Episode File Entity

| # | SlipStream Field | Type | Sonarr Source Field | Type | Transformation |
|---|---|---|---|---|---|
| 1 | `episode_id` | INTEGER FK | — | — | Set to newly-created SlipStream episode ID |
| 2 | `path` | TEXT NOT NULL | `path` | string | Direct (absolute path) |
| 3 | `size` | INTEGER | `size` | long | Direct (both in bytes) |
| 4 | `quality_id` | INTEGER | `quality.quality.id` + `quality.quality.name` | int, string | **Map:** See §3.6 Quality ID Mapping |
| 5 | `video_codec` | TEXT | `mediaInfo.videoCodec` | string | Direct |
| 6 | `audio_codec` | TEXT | `mediaInfo.audioCodec` | string | Direct |
| 7 | `resolution` | TEXT | `mediaInfo.resolution` | string | Direct (Sonarr format: `"1920x1080"`) |
| 8 | `audio_channels` | TEXT | `mediaInfo.audioChannels` | decimal | Convert decimal to string (e.g., `5.1` → `"5.1"`) |
| 9 | `dynamic_range` | TEXT | `mediaInfo.videoDynamicRange` | string | Direct (e.g., `"HDR"`, `"HDR10"`) |
| 10 | `original_path` | TEXT | `originalFilePath` | string | Direct |
| 11 | `original_filename` | TEXT | `originalFilePath` | string | **Derive:** Extract filename from `originalFilePath` using path.Base() |
| 12 | `imported_at` | DATETIME | `dateAdded` | DateTime | Direct |
| 13 | `created_at` | DATETIME | `dateAdded` | DateTime | Direct |
| 14 | `slot_id` | INTEGER FK | — | — | Set by slots service if multi-version enabled (see §8) |

### 3.5 Episode Status Derivation

Status is **not** mapped from Sonarr. SlipStream's own logic handles it entirely:

1. **On episode creation** (`tv.Service.createSeasonsAndEpisodes` / `computeEpisodeStatus`): Checks the episode's `airDate`.
   - `airDate` is nil or in the future → `"unreleased"`
   - `airDate` is in the past → `"missing"`
2. **On file attachment** (`tv.Service.AddEpisodeFile`): `profile.StatusForQuality(qualityID)` checks the file's quality against the series' quality profile cutoff.
   - At or above cutoff → `"available"`
   - Below cutoff with upgrades enabled → `"upgradable"`
   - Below cutoff with upgrades disabled → `"available"`

**Import approach:** Create the episode with its `airDateUtc` (mapped in row 6), then call `AddEpisodeFile()` for each episode that has a file in Sonarr (`hasFile == true`). SlipStream will derive the correct status at each step. Skip series with Sonarr status `deleted` (-1).

### 3.6 Quality ID Mapping (Sonarr → SlipStream)

Sonarr uses the same quality naming system as Radarr. The mapping is identical to §2.4 with the addition of:

| Sonarr Quality (ID → Name) | SlipStream Quality (ID → Name) |
|---|---|
| 13 → Bluray-480p | — (no match; nearest: 2 → DVD) |
| 20 → Bluray-1080p Remux | 12 → Remux-1080p |
| 21 → Bluray-2160p Remux | 17 → Remux-2160p |
| 22 → Bluray-576p | — (no match; nearest: 2 → DVD) |

All other quality names share the same mapping as §2.4.

---

## 4. Quality Profile Mapping

Quality profiles are not imported. During the import flow, the user maps each Sonarr/Radarr profile to an existing SlipStream profile. The import UI presents a mapping step showing each source profile name alongside a dropdown of SlipStream profiles.

| Source Data | Usage |
|---|---|
| Source profile `name` | Display label so user can identify it |
| Source profile `id` | Key for looking up which media items use this profile |

Each imported movie/series gets its `quality_profile_id` set to the SlipStream profile the user selected for the corresponding source profile.

---

## 5. Root Folder Mapping

| # | SlipStream Field | Sonarr/Radarr Field | Transformation |
|---|---|---|---|
| 1 | `path` | `path` | Direct |
| 2 | `name` | `path` | **Derive:** Extract last directory component from path |
| 3 | `media_type` | — | `"movie"` for Radarr, `"tv"` for Sonarr |
| 4 | `free_space` | `freeSpace` | Direct (both in bytes) |

**Note:** Root folders must be created/matched before importing media, since movies and series reference them by FK.

---

## 6. Import Workflow Summary

1. **Connect:** User browses to the Sonarr/Radarr `.db` file (or provides URL + API key for API fallback)
2. **Read:** Load root folders, quality profiles, and all media from source
3. **Map root folders:** Match or create SlipStream root folders
4. **Map quality profiles:** User maps each source profile to a SlipStream profile
5. **Preview:** Show library preview (re-use MigrationPreviewModal pattern) with counts, conflicts (duplicate TMDB/TVDB IDs), and path validation
6. **Import:** Create movies/series/seasons/episodes with mapped data, create file records for existing files, derive statuses
7. **Slot assignment:** If multi-version enabled, initialize slot assignments for all media and assign files to best-matching slots (see §8)
8. **Report:** Show import summary (created, skipped/duplicate, errors)

---

## 7. Multi-Version (Slots) Mode

Sonarr and Radarr are single-version systems — one file per movie/episode. SlipStream supports up to 3 concurrent quality versions via version slots. The import must handle both single-version and multi-version modes.

### 7.1 Detection

Check `slots.Service.IsMultiVersionEnabled()` at import time to determine which path to follow.

### 7.2 Single-Version Mode (slots disabled)

No slot-related work needed. Files are created with `slot_id = NULL`. The `quality_profile_id` on the movie/series entity is used directly for status derivation. This is the standard path described in §2–§3.

### 7.3 Multi-Version Mode (slots enabled)

When multi-version is enabled, each imported file must be assigned to the best-matching slot, and slot assignment records must be created for all enabled slots.

#### After creating each movie/series/episode:

1. **Initialize slot assignments** — Call `slots.Service.InitializeSlotAssignments(ctx, mediaType, mediaID)` to create `movie_slot_assignments` / `episode_slot_assignments` rows for every enabled slot. Each is created with `file_id = NULL`, `monitored = 1`, `status = "missing"`.

#### After creating each file (for media that has a file):

2. **Match file to best slot** — Use `slots.Service.matchFileToSlot()` to evaluate the file's quality against each enabled slot's quality profile. The slot whose profile best matches (highest score, file quality is acceptable) wins.
3. **Assign file to slot** — Call `slots.Service.AssignFileToSlot(ctx, mediaType, mediaID, slotID, fileID)` which:
   - Sets `movie_files.slot_id` / `episode_files.slot_id` on the file record
   - Upserts the slot assignment: sets `file_id` and computes status via the slot's quality profile

#### Slot assignment entity (created automatically, not mapped from source)

| Field | Value |
|---|---|
| `movie_id` / `episode_id` | Newly-created SlipStream media ID |
| `slot_id` | Each enabled slot gets a row |
| `file_id` | Set for the matched slot, `NULL` for others |
| `monitored` | `1` (default) |
| `status` | `"available"` or `"upgradable"` for the matched slot; `"missing"` or `"unreleased"` for empty slots |
| `active_download_id` | `NULL` |
| `status_message` | `NULL` |

#### Status implications in multi-version mode

The movie/episode aggregate status is the highest-priority status across all **monitored** slot assignments (priority: downloading > failed > missing > upgradable > available > unreleased). Since the imported file only fills one slot, other monitored slots will be `"missing"`, making the overall media status `"missing"` even though a file exists. This is correct — it reflects that the library still needs content for the other slots.

---

## 8. Fields Intentionally Not Mapped

These Sonarr/Radarr fields have no SlipStream equivalent and are excluded from import:

| Source Field | Reason |
|---|---|
| `genres`, `ratings`, `actors` | SlipStream does not store this metadata (fetched on-demand from TMDB/TVDB) |
| `tags` | SlipStream has no tagging system |
| `customFormats`, `customFormatScore` | SlipStream uses attribute-based quality settings instead |
| `minimumAvailability` | SlipStream derives availability from release dates |
| `collection` | SlipStream has no collection concept |
| `alternativeTitles` | SlipStream has no alternative titles table |
| `addOptions`, `monitorNewItems` | One-time import settings, not applicable |
| `titleSlug` | SlipStream uses numeric IDs for routing |
| `originalLanguage`, `languages` | SlipStream has no language tracking |
| `releaseGroup`, `sceneName`, `edition` | SlipStream does not store release metadata on file records |
| `indexerFlags` | Not applicable outside of *arr ecosystem |
| `sceneNumbering` (Sonarr) | SlipStream uses standard numbering only |
| `absoluteEpisodeNumber` (Sonarr) | SlipStream does not support absolute numbering |
