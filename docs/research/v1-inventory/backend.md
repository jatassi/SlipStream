# SlipStream v1 admin backend feature inventory

Source tree: `~/Git/SlipStream-v1` (all paths below are relative to it). Read-only survey for v2 parity planning.
Scope: admin-facing backend capabilities. Portal (requests) end-user surface is summarized briefly in its own section because the admin manages it.

Legend:
- **[HALF]** looks half-built, stubbed, or contradicted by code.
- **[TODO-DOC]** listed as remaining/unfinished in a doc.
- **Ext:** external service the feature talks to.

All HTTP endpoints are under `/api/v1` unless noted; admin routes are gated by `adminAuthMiddleware` (`internal/api/routes.go`).

## 0. Doc status snapshot

`docs/Remaining-features.md` (top-level roadmap):
- Done before "Ready for Dogfooding": automatic search, system health, release importing (naming), multiple quality versions, upgrades, notifications, developer mode, Prowlarr, search results page, external requests portal, passkeys, web app settings (port/auth/https), settings reorg, update mechanism.
- Done before "Ready for Beta": history page, activity UI, list pages, detail pages, missing page, more download clients and indexers, library in portal, deps update, refactor/security audit.
- Marked `[*]` (in progress): logo/color palette; **onboarding wizard (Sonarr/Radarr migration)**.
- Not started `[ ]`: **config backup & restore**, **update dashboard**, **transcode during import (FFmpeg, integrated with slots)**, **TRaSH Guides import**.

`docs/bugs.md`: all items fixed except **admin login redirect issue** (open).

`docs/Production-Readiness-Checklist.md` open items: static binary; entire CI/CD release workflow and artifacts (NSIS/Inno, portable zip, DMG, deb, rpm, AppImage); Windows tray/startup/installer items (unchecked in doc but see section 20 for code reality); **auto-install toggle UI**; **credential encryption** (says `internal/crypto/secrets.go` exists, but there is no `internal/crypto` dir in the tree, so it is absent); macOS/Linux/Docker packaging; code signing; "Sonarr/Radarr import of library, quality profiles, download clients" (unchecked, but arrimport exists, see section 12). Pre-update DB backup is a placeholder that only logs.

## 1. Library management

Paths: `internal/library/{movies,tv,librarymanager,rootfolder,scanner,organizer,status}`, `internal/availability`, `internal/defaults`, `internal/preferences`, `internal/filesystem`, `internal/mediainfo`. Wiring in `internal/api/routes.go` (`setupLibraryRoutes`).

### 1.1 Root folders
- Create, list (filter by `mediaType` movie|tv), get, delete root folders; path is resolved to absolute, validated, de-duplicated, and free space recorded. `internal/library/rootfolder/service.go`, `/rootfolders`.
- Creating a root folder auto-triggers a scan of it. `internal/api/routes.go` (`SetOnFolderCreated`).
- Mark a root folder as the default per media type (also exposed via generic defaults API). `rootfolder/service.go` (`SetDefault`/`ClearDefault`), `internal/defaults`.
- Free space per folder is tracked and refreshed; each root folder is registered with the health service. `rootfolder/service.go` (`RefreshAllFreeSpace`, `RegisterExistingRootFolders`).
- Scan a root folder on demand, get scan status, cancel a scan, list all scan statuses, scan all folders. `internal/library/librarymanager/scanning.go`, `/rootfolders/:id/scan`, `/scans`.
- Scan walks files, parses filenames, matches to TMDB/TVDB metadata, auto-creates movies/series in the library, links files, downloads artwork, reports progress as an activity. `librarymanager/scanning.go`, `librarymanager/matching.go`, `librarymanager/creation.go`, `internal/library/scanner/{scanner,parser,extensions}.go`. Ext: TMDB, TVDB.
- Scan verifies tracked files still exist; vanished files flip the item to `missing` and raise a health alert. `librarymanager/scanning.go` (`VerifyFileExistence`, `recompute*Status`).

### 1.2 Movies
- List movies with search text, monitored filter, root-folder filter. `internal/library/movies/handlers.go`, `GET /movies`.
- Get, update (title, year, ids, path, root folder, quality profile, monitored, studio, content rating, three release dates), delete movie (optional `deleteFiles=true` removes files from disk). `movies/service.go`, `/movies/:id`.
- Bulk monitor/unmonitor many movies. `PUT /movies/monitor`.
- List, add, remove movie files (file record carries quality, video/audio codec, channels, dynamic range, resolution, slot id). `/movies/:id/files`.
- Add movie from metadata search ("add flow"): auto-fills release dates (digital, physical, theatrical), content rating, logo and studio logo, downloads poster/backdrop/logo, optional search-on-add. `internal/library/librarymanager/add.go`, `POST /library/movies`. Ext: TMDB.
- Refresh metadata for one movie or all movies (re-matches, enriches, re-downloads artwork, then rescans the movie folder). `librarymanager/metadata_refresh.go`, `POST /movies/:id/refresh`, `POST /movies/refresh`.
- Movie status model: `unreleased`, `missing`, `downloading`, `failed`, `upgradable`, `available`, plus `statusMessage` and `activeDownloadId`. `internal/library/status/status.go`; transitions in `movies/service.go`, `internal/availability/service.go`.
- Tracks `addedBy` (portal user who requested it) with username. `movies/movie.go`.

### 1.3 TV (series, seasons, episodes)
- List series with search, monitored, root-folder filters; series carry per-status episode counts, network + network logo, production status, format type, first/last/next airing, season folder flag. `internal/library/tv/{series,service,handlers}.go`, `GET /series`.
- Get, update, delete series (optional `deleteFiles`). `/series/:id`.
- Bulk monitor/unmonitor many series. `PUT /series/monitor`.
- Apply a monitoring preset to a series: `all`, `none`, `future`, `first_season`, `latest_season`, `existing`, with an include-specials toggle. `PUT /series/:id/monitor`, `tv/series.go` (`MonitorType`).
- Get monitoring stats (seasons/episodes monitored vs total). `GET /series/:id/monitor/stats`.
- List seasons, update a season (monitored). `/series/:id/seasons`, `tv/seasons.go`.
- List episodes (optionally by season), get/update an episode, bulk monitor episodes. `/series/:id/episodes`, `tv/episodes.go`.
- Add/remove an episode file record. `/series/:id/episodes/:episodeId/files`.
- Add series from metadata ("add flow") with `monitorOnAdd` (none, first_season, latest_season, future, all), `searchOnAdd` (no, first_episode, first_season, latest_season, all), include specials, seasons/episodes payload, artwork and network logo download. `librarymanager/add.go`, `POST /library/series`. Ext: TMDB, TVDB.
- Refresh metadata for one series or all series, including season/episode list sync. `librarymanager/metadata_refresh.go`, `POST /series/:id/refresh`, `POST /series/refresh`.
- Episode status uses the same six-state model as movies; season and series roll up status counts. `tv/series.go` (`StatusCounts`).

### 1.4 Library-wide helpers
- Default entity per media type for root folder, quality profile, download client, indexer (get all, get by entity type, set, clear). `internal/defaults/{service,handlers}.go`, `/defaults/:entityType/:mediaType`.
- Add-flow preferences: movie search-on-add, series search-on-add, series monitor-on-add, include specials. `internal/preferences`, `GET/PUT /preferences/addflow`.
- Missing view: missing movies, missing series grouped by season, counts; upgradable movies/series and counts. `internal/missing/{service,handlers}.go`, `/missing/*`.
- Unreleased to missing transitions run hourly from release/air dates. `internal/availability/service.go`, task `status-refresh`.
- Filesystem browser for picking folders (drives/volumes on Windows/macOS/Linux, hides system dirs), import-oriented browse that lists video files and skips samples, storage info per volume containing root folders, and "scan path for media" that parses each file. `internal/filesystem/{service,storage_service,drives_unix,drives_windows,handlers}.go`, `/filesystem/browse`, `/filesystem/browse/import`, `/filesystem/storage`, `/filesystem/scan`.
- MediaInfo probing of files (codec, HDR, channels) when the `mediainfo` CLI is available; `/status` reports `mediainfoAvailable`. `internal/mediainfo/{cli,hdr,service}.go`. Ext: local `mediainfo` binary.
- File organizer (move/copy/hardlink helpers, folder creation) used by import and rename. `internal/library/organizer/{organizer,fileops}.go`.

## 2. Metadata providers

Paths: `internal/metadata/{service,provider,cache,artwork,network_logo_store,handlers}.go`, clients in `internal/metadata/{tmdb,tvdb,omdb}`, mocks in `internal/metadata/mock`.
- Search movies (query + year) and series. `GET /metadata/movie/search`, `GET /metadata/series/search`. Ext: TMDB (movies, series), TVDB (series search fallback/merge).
- Get movie by TMDB id; get series by TMDB id or TVDB id; generic series get. `/metadata/movie/:id`, `/metadata/series/tmdb/:id`, `/metadata/series/tvdb/:id`, `/metadata/series/:id`.
- Extended detail ("more info"): cast/crew (directors, writers, creators, cast with photos), content rating, studio, trailer URL, external ratings (IMDb rating/votes, Rotten Tomatoes critic + audience, Metacritic, awards), seasons, per-episode IMDb ratings. `/metadata/movie/:id/extended`, `/metadata/series/:id/extended`. Ext: TMDB, OMDb.
- Movie release dates split into digital, physical, theatrical; movie content rating; movie logo and studio logo; series logo; network logos cached in DB. `metadata/service.go`, `metadata/network_logo_store.go`.
- Download artwork (poster, backdrop, logo, studio logo) to `data/artwork`, serve it back, broadcast `artwork:ready`. `metadata/artwork.go`, `POST /metadata/{movie,series}/:id/artwork`, `GET /metadata/artwork/:type/:id/:artworkType` (unauthenticated artwork route).
- In-memory TTL cache (15 min) of metadata responses; clear on demand. `metadata/cache.go`, `DELETE /metadata/cache`.
- Provider status and connection tests (TMDB, TVDB, OMDb). `GET /metadata/status`, health tests in `internal/health`.
- TMDB "search ordering" toggle (disable TMDB's popularity ordering). `POST /metadata/tmdb/search-ordering`, `internal/metadata/tmdb/client.go`, `internal/api/handlers_system.go`.
- API keys from config/env or baked in at build via ldflags. `internal/config`, `Makefile`.
- Metadata routes are shared with portal users (AnyAuth). `internal/api/routes.go`.

## 3. Quality profiles, cutoffs, attributes, scoring

Paths: `internal/library/quality/{profile,attributes,exclusivity,matcher,service,handlers}.go`, `internal/indexer/scoring/{scorer,quality_matcher,types}.go`, `internal/decisioning/selection.go`.
- CRUD quality profiles per module type (movie, tv). `/qualityprofiles`.
- Fixed list of 18 predefined qualities (SDTV, DVD, WEBRip/HDTV/WEBDL/Bluray/Remux at 480/720/1080/2160, CAM). `GET /qualityprofiles/qualities`, `quality/profile.go`.
- Per profile: ordered allowed qualities, cutoff quality, upgrades on/off, upgrade strategy (`aggressive`, `balanced`, `resolution_only`), "cutoff overrides strategy", allow-auto-approve flag (for portal requests). `quality/profile.go`.
- Per profile attribute rules for HDR format, video codec, audio codec, audio channels, each value set to `acceptable`, `preferred`, `required`, or `notAllowed`. `quality/attributes.go`, `GET /qualityprofiles/attributes`.
- Default "Any" profile is created per module if none exist (CAM disallowed). `quality/service.go` (`EnsureDefaults`), `internal/api/server.go`.
- Mutual-exclusivity check between two profiles (used for slots). `POST /qualityprofiles/check-exclusivity`, `quality/exclusivity.go`.
- Release scoring (not user-tunable): quality points vs profile, torrent health (seeders, ratio, freeleech), indexer priority, year/episode/season-pack match, age penalty, language mismatch penalty (preferred language is never set by any caller, so it is always English), disallowed-quality penalty; results sorted and carry a score breakdown. Weights are hard-coded defaults, not user-editable. `internal/indexer/scoring/scorer.go`, `types.go`.
- Best-release selection applies profile, current file quality, upgrade rules, and module-specific filters. `internal/decisioning/selection.go`.
- **Custom formats do not exist.** Only placeholders: notification payload fields `customFormats`/`customFormatScore` and a renamer token are always empty. `internal/notification/types/types.go`, `internal/notification/discord/notifier.go`, `internal/import/renamer/tokens.go`. **[HALF]**
- TRaSH Guides import is not started. **[TODO-DOC]** `docs/Remaining-features.md`.

## 4. Import pipeline (post-download), renaming, naming tokens

Paths: `internal/import/{service,pipeline,completion,seasonpack,retry,rename,validation,matching,module_matching,module_naming,naming_handlers,handlers,settings,token_context}.go`, `internal/import/renamer/*`, `internal/library/organizer`, `internal/downloader/completion.go`.

### 4.1 Automatic import
- Completed downloads are detected (download client status + path), mapped back to the library item via the download mapping created at grab time, validated, renamed, and placed in the library. `import/pipeline.go` (`ProcessCompletedDownload`), `import/service.go` (`CheckAndProcessCompletedDownloads`).
- Periodic import scan (every 5 min, also on start) of download client folders for SlipStream-category subdirs; skips files already imported, including hardlinks to library files (inode compare). `import/pipeline.go` (`ScanForPendingImports`, `isHardlinkToLibraryFile`), task `import-scan` in `internal/scheduler/tasks/importscan.go`.
- File completion checks: wait for file size to stabilize, detect archives (rar/zip etc.) and wait for extraction, skip stalled downloads. `import/completion.go`.
- Matching: queue mapping first, filename parse as fallback (the conflict-behavior setting is not wired, see 4.3); module-aware matching through the registry; match confidence. `import/matching.go`, `import/module_matching.go`.
- Placement: hardlink, fall back to symlink on cross-device, then copy; source kept for seeding. Not user-configurable (no move/copy choice). `internal/library/organizer/fileops.go` (`LinkOrCopy`).
- Upgrade handling: compare candidate vs existing quality against the profile; replace and delete the old file when it is an upgrade; reject otherwise. Import decisions recorded per profile and cleared when the profile changes. `import/service.go` (`evaluateUpgradeEligibility`, `recordImportDecision`, `ClearDecisionsForProfile`).
- Season packs: analyze pack, per-file status rows (`pending`, `downloading`, `ready`, `importing`, `imported`, `failed`), create missing episode records, import ready files progressively, assign each episode to a slot individually. `import/seasonpack.go`.
- Multi-episode files map to several episodes. `import/renamer/multiepisode.go`, `LibraryMatch.EpisodeIDs` in `import/service.go`.
- Retry with error classification and backoff; failed imports set item status to `failed`, raise a health warning, broadcast `import:failed`. `import/retry.go`, `import/service.go`.
- Success side effects: history entry, Import or Upgrade notification, portal request status update to available, WebSocket `import:completed` and entity update. `import/service.go` (`handleSuccessfulImport`, `dispatchImportNotification`, `updatePortalRequestStatus`).
- MediaInfo probe after import to fill codec/HDR/channels on the file record. `internal/mediainfo`, `tv/episodes.go` (`UpdateEpisodeFileMediaInfo`).

### 4.2 Manual import and import queue UI
- Get import queue status (queue length, processing count) and pending imports list with per-file status, error, attempts. `GET /import/status`, `GET /import/pending`.
- Scan a directory and get each file's parse result, validation result, and suggested library match. `POST /import/scan`.
- Preview manual import (shows slot assignments, recommended slot, upgrade vs new fill) and execute manual import with optional target slot. `POST /import/manual/preview`, `POST /import/manual`, `import/handlers.go`.
- Retry a failed import by queue-media id. `POST /import/:id/retry`.

### 4.3 Import settings
- Validation level (`basic`, `standard`, `full` with MediaInfo), minimum file size MB (default 100), allowed video extensions list, match conflict behavior (`trust_queue`, `trust_parse`, `fail`), unknown media behavior (`ignore`, `auto_add`). `import/settings.go`, `GET/PUT /settings/import`.
- **[HALF]** Only the extension list and minimum size are read by the pipeline (`import/validation.go`, `import/pipeline.go`). Validation level, match conflict behavior, and unknown media behavior (`auto_add`) are stored and editable but never consulted anywhere in `internal/`. The UI exposes them (`web/src/components/settings/sections/naming-matching-tab.tsx`, `naming-validation-tab.tsx`), so users can change settings that do nothing.

### 4.4 Renaming and naming templates
- Per-module naming settings: rename on/off, colon replacement (`delete`, `dash`, `space_dash`, `space_dash_space`, `smart`, `custom` + custom string), templates per context. `import/naming_handlers.go`, `import/module_naming.go`, `GET/PUT /settings/:moduleId/naming`.
- Movie contexts: `movie-folder` default `{Movie Title} ({Year})`, `movie-file` default `{Movie Title} ({Year}) - {Quality Title}`. `internal/modules/movie/path_naming.go`.
- TV contexts: `series-folder`, `season-folder`, `specials-folder`, `episode-file` with variants `standard`, `daily`, `anime`; season folder toggle; multi-episode style (`extend`, `duplicate`, `repeat`, `scene`, `range`, `prefixed_range`). `internal/modules/tv/path_naming.go`.
- Token catalog: Series Title/TitleYear/CleanTitle/CleanTitleYear, season, episode, Air-Date, Air Date, Episode Title/CleanTitle, Quality Full/Title, MediaInfo Simple/Full/VideoCodec/VideoBitDepth/VideoDynamicRange/VideoDynamicRangeType/AudioCodec/AudioChannels/AudioLanguages/SubtitleLanguages, Release Group, Edition Tags, Custom Formats (always empty), Original Title, Original Filename, Revision, absolute, version, Movie Title/TitleYear/CleanTitle/CleanTitleYear, Year, IMDb, TMDB. Modifiers: zero padding (`{season:00}`), truncation, case modes, language include/exclude filters. `import/renamer/{tokens,modifiers,characters,resolver}.go`.
- Preview a naming pattern, validate a pattern, parse a filename (global and per-module). `POST /settings/import/naming/{preview,validate,parse}`, `POST /settings/:moduleId/naming/{preview,parse}`.
- Mass rename existing library files: preview the rename plan, execute it; writes `file_renamed` history and broadcasts `rename:completed`. `import/rename.go`, `GET /import/rename/preview`, `POST /import/rename/execute`.
- Transcode during import (FFmpeg, tied to slots) is not started. **[TODO-DOC]**

## 5. Radarr/Sonarr migration (library import and config import)

Paths: `internal/arrimport/{service,handlers,detect,reader,reader_sqlite,reader_api,executor,quality_map,config_map,config_types,types}.go`. Frontend wizard is the "onboarding wizard", marked in progress **[TODO-DOC]** in `docs/Remaining-features.md`. Specs: `docs/Library-Import-Spec.md`, `docs/Library-Import-Plan.md`, `docs/config-import-plan.md`.
- Auto-detect local Radarr/Sonarr SQLite DB candidates on disk. `arrimport/detect.go`, `GET /arrimport/detect-db`.
- Connect to a source either by SQLite DB path or by URL + API key (Radarr/Sonarr v3 API). `arrimport/reader_sqlite.go`, `arrimport/reader_api.go`, `POST /arrimport/connect`, `DELETE /arrimport/session`. Ext: Radarr API, Sonarr API (or their DB files).
- List the source's root folders and quality profiles (with in-use flag) for mapping. `GET /arrimport/source/rootfolders`, `GET /arrimport/source/qualityprofiles`.
- Preview library import (movies or series with files) under a root-folder and profile mapping, then execute it; existing files are linked, not moved. `POST /arrimport/preview`, `POST /arrimport/execute`, `arrimport/executor.go`, `arrimport/quality_map.go`.
- Config import preview and execute with per-item selection: download clients, indexers, notifications, quality profiles, naming config (with "same/different" diff). Report lists created/skipped counts, warnings, errors. `GET /arrimport/config/preview`, `POST /arrimport/config/import`, `arrimport/config_map.go`.
- Download client mapping covers Transmission, qBittorrent, Deluge, rTorrent, Vuze, Aria2, Flood, uTorrent, Hadouken, DownloadStation, FreeboxDownload. SABnzbd and NZBGet map to "" (skipped). Blackhole, NzbVortex, UsenetDownloadStation, Pneumatic skipped. `arrimport/config_map.go`.
- Notification mapping includes Gotify, Ntfy, Apprise, Pushbullet, Join, Prowl, Simplepush, Signal, CustomScript, which SlipStream cannot actually send (see section 9). **[HALF]**
- Bug history: duplicate series creation after Sonarr import (fixed). `docs/bugs.md`.

## 6. Indexers

Paths: `internal/indexer/{service,handlers,indexer,categories,cookie_store,events,types}.go`, `internal/indexer/cardigann/*`, `internal/indexer/genericrss/*`, `internal/indexer/status/*`, `internal/indexer/ratelimit/*`, `internal/prowlarr/*`. Two indexer modes: `slipstream` (native indexers) and `prowlarr` (all search goes through Prowlarr). `internal/prowlarr/mode.go`, `GET/PUT /indexers/mode`.

### 6.1 Native indexers (SlipStream mode)
- CRUD indexers: name, definition id, categories, protocol (torrent/usenet), privacy, supports movies/TV/search/RSS, priority, enabled, auto-search enabled, RSS enabled, definition-specific settings JSON (merged on update). `internal/indexer/service.go`, `/indexers`.
- Test a saved indexer or an unsaved config. `POST /indexers/:id/test`, `POST /indexers/test`.
- Cardigann (Prowlarr/Jackett YAML) definitions: list, search with filters, get one, get its settings schema, update definitions from the remote repo (zip package). Local definition cache on disk plus a `custom` definitions folder. `internal/indexer/cardigann/{manager,repository,cache,definition}.go`, `/indexers/definitions*`. Ext: `https://indexers.prowlarr.com` (definitions repo, version 11).
- Cardigann runtime: templated search URLs, selectors (CSS/HTML), JSONPath responses, filters, login flows (`form`/POST, `cookie`, none) with persisted cookie store. `cardigann/{search,selector,jsonpath,filters,template,login,client}.go`, `internal/indexer/cookie_store.go`. Ext: each tracker site.
- Generic RSS indexer: URL, optional cookie, content type (movies/tv/both); RSS-only, no search. `internal/indexer/genericrss/{indexer,parser}.go`.
- Newznab/Torznab categories with movie (2000 range) and TV (5000 range) subcategories; indexer applicability to a module is derived from category ranges. `internal/indexer/categories.go`, `types/types.go` (`SupportsModule`).
- Per-indexer status: failure escalation with `disabledTill` backoff, last RSS sync, last search, health classification (healthy, warning, disabled, unknown), summary counts. `internal/indexer/status/{models,service}.go`, `GET /indexers/status`, `GET /indexers/:id/status`, WebSocket `indexer:status`.
- Per-indexer rate limits (default 100 queries/hour, 25 grabs/hour), enforced on search and grab. Tunable only in the config file (`indexer.rate_limit.*`), not in the UI. Indexer failure backoff (initial 5 min, x2, max 3 h) is also config-file only (`indexer.status.*`). `internal/indexer/ratelimit/limiter.go`, used in `indexer/search/service.go` and `indexer/grab/service.go`.
- Each indexer is registered with health monitoring; a scheduled health check tests all enabled indexers. `internal/scheduler/tasks/indexerhealth.go`.
- Indexer history (grab history across indexers, paginated). `GET /history/indexer`, `GET /search/grab/history`, `internal/api/handlers_downloads.go`.
- Default indexer per media type via the defaults API (section 1.4).
- Mock indexer (dev mode). `internal/indexer/mock`.

### 6.2 Prowlarr integration
- Configure Prowlarr connection: enabled, URL, API key, movie categories, TV categories, timeout, skip SSL verify. `internal/prowlarr/{service,types,handlers}.go`, `GET/PUT /indexers/prowlarr`. Ext: Prowlarr API.
- Test connection, get Prowlarr status, refresh cached data, read capabilities (server, limits, search types, categories), list Prowlarr's indexers with status and caps. `/indexers/prowlarr/{test,status,refresh,capabilities,indexers}`.
- Per-Prowlarr-indexer overrides: priority, content type (movies/tv/both), movie/TV categories; success/failure counters with last failure reason; reset stats; delete overrides. `/indexers/prowlarr/indexers/:id/settings`, `/indexers/prowlarr/indexers/:id/reset-stats`, `prowlarr/types.go` (`IndexerSettings`).
- Caching: search results 5 min, capabilities 15 min, indexer list 15 min; capabilities persisted to DB. `prowlarr/service.go`.
- Adaptive rate limiter (exponential backoff to 30s, recovers after 5 successes). `prowlarr/ratelimit.go`.
- Grabs in Prowlarr mode download through Prowlarr's download URL. `prowlarr/grabprovider.go`.
- Scheduled Prowlarr health check that also refreshes indexer data. `internal/scheduler/tasks/prowlarrhealth.go`.
- Mode switch is overridden in dev mode (`devModeOverride`). `prowlarr/mode.go`.
- Doc gaps vs Radarr (per-indexer push sync, richer status) are listed in `docs/Prowlarr-integration-audit.md`; the implementation plan addressed part of them. Treat the audit's gap tables as candidates, not confirmed v1 features.

## 7. Search, grab, decisioning

Paths: `internal/indexer/search/{service,router,aggregator,title_match,handlers}.go`, `internal/indexer/grab/{service,handlers}.go`, `internal/decisioning/*`, `internal/indexer/scoring/*`. Specs: `docs/search-relevance-improvements-spec.md`, `docs/search-results-enhancements-spec.md`.
- Interactive (manual) search: general, movie, TV, and scored torrent search with query, type, categories, IMDb/TMDB/TVDB ids, season, episode, year, limit/offset, quality profile for scoring. `GET /search`, `/search/movie`, `/search/tv`, `/search/torrents`.
- Search router sends queries to native indexers in parallel or to Prowlarr depending on mode; results deduplicated (higher-priority indexer wins for releases, most seeders for torrents), filtered by criteria, quality-enriched, scored. `search/router.go`, `search/aggregator.go`.
- Title matching to reject wrong-title releases. `search/title_match.go`.
- Broadcasts `search:started` and `search:completed`. `search/service.go`.
- Grab a release (optionally choose download client, target media, season pack/complete series flags, target slot) and bulk grab. Picks a client supporting the protocol (torrent/usenet) with priority, sends the torrent/NZB or URL, records grab history, creates the download mapping, marks the item `downloading`, sends Grab notification, broadcasts `grab:started`/`grab:completed`. `grab/service.go`, `POST /search/grab`, `POST /search/grab/bulk`.
- Per-item grab lock prevents RSS sync and autosearch grabbing the same item at once. `decisioning/grablock.go`.
- Wanted-item collection (missing + upgrade candidates, season pack eligibility when all episodes in a season are released, monitored, missing or upgradable). `decisioning/collection.go`.
- Release/filename parser extracts title, year, season/end season (multi-season packs), episode/end episode (multi-episode), season pack, complete series, resolution, source, video codec, audio codecs, channels, audio enhancements (Atmos, DTS:X), HDR formats, REMUX, release group, revision (Proper/REPACK/REAL), edition, languages, TV vs movie. `internal/library/scanner/parser.go`, `internal/module/parseutil/*`, module `fileparser.go`. Debug parse endpoints in 4.4 and 11.
- Revision (Proper/Repack) is parsed and shown in naming tokens but no decision code uses it, so propers do not trigger upgrades. Observed by grep across `decisioning`, `scoring`, `quality`, `autosearch`, `rsssync`, `import`.
- No release blocklist or "mark failed and search again" blocklist exists (no blocklist table or API in `internal/`). Retry endpoints in 8.1 re-run search only.

## 8. Automatic search and RSS sync

### 8.1 Automatic search
Paths: `internal/autosearch/{service,scheduled,slots,handlers,settings,events,types,convert}.go`, `internal/scheduler/tasks/autosearch.go`. Spec: `docs/automatic-search-spec.md`.
- Search now for one movie, one episode, one season, or a whole series; per-slot variants for movie and episode. `POST /autosearch/{movie,episode}/:id`, `/autosearch/{movie,episode}/:id/slot/:slotId`, `/autosearch/season/:seriesId/:seasonNumber`, `/autosearch/series/:id`.
- Get autosearch status for an item. `GET /autosearch/status/:mediaType/:id`.
- Retry a failed movie or episode. `POST /autosearch/retry/{movie,episode}/:id`.
- Bulk: search all missing (all, movies, series) and all upgradable (all, movies, series). `POST /autosearch/missing/*`, `/autosearch/upgradable/*`.
- Scheduled automatic search: enabled flag, interval hours, backoff threshold (skip items after N consecutive failed searches; counter resets on success). `autosearch/settings.go`, `GET/PUT /settings/autosearch`, task `autosearch`.
- Scheduled run refreshes monitored series metadata first, collects wanted items via module collectors, prioritizes by release date, groups TV into season-pack searches when eligible with fallback to individual episodes. `autosearch/scheduled.go`.
- Multi-slot search: searches each monitored slot that needs a file in parallel and may grab several releases. `autosearch/slots.go`.
- Search-on-add hooks from the add flow. `librarymanager/add.go`.
- History entries `autosearch_download` / `autosearch_failed`, WebSocket `autosearch:started|completed|failed` and `autosearch:task:started|progress|completed`. `autosearch/events.go`.

### 8.2 RSS sync
Paths: `internal/rsssync/{service,feedfetcher,matcher,cache,settings,events,handlers}.go`, task `internal/scheduler/tasks/rsssync.go`. Docs: `docs/RSS-Sync-Design.md`, `docs/RSS-Sync-Implementation-Plan.md`, `docs/RSS-Sync-Review.md`.
- Enable/disable and set interval minutes. `GET/PUT /settings/rsssync`.
- Trigger a sync now; read last status (running, last run, total releases, matched, grabbed, elapsed, error). `POST /rsssync/trigger`, `GET /rsssync/status`.
- Fetches recent releases from every RSS-enabled indexer (or Prowlarr aggregated feed), stops at the per-indexer cache boundary (last seen release), matches against wanted items, scores, grabs best per item, skips items with a recent grab. `rsssync/service.go`, `rsssync/cache.go`, `rsssync/matcher.go`.
- Per-feed health warnings (`rss-<indexer>`). `rsssync/service.go` (`reportFeedHealth`).
- WebSocket `rss-sync:started|progress|completed|failed`; history entries with source `rss-sync`.

## 9. Download clients and queue

Paths: `internal/downloader/{service,factory,client,queue,queue_media,completion,broadcaster}.go`, per-client packages under `internal/downloader/*`, HTTP in `internal/api/handlers_downloads.go`. Docs: `docs/torrent-client-parity-plan.md`, `docs/torrent-client-agent-guide.md`.

### 9.1 Client configuration
- CRUD download clients: name, type, host, port, username, password, SSL, API key, category, URL base, priority (default 50), enabled, import delay seconds, cleanup mode (`leave`, `delete_after_import`, `delete_after_seed_ratio`), seed ratio target. `downloader/service.go`, `/downloadclients`.
- Credentials are redacted (`********`) in API responses and restored on update when the sentinel is sent back. `internal/api/handlers_downloads.go`.
- Test a saved client or an unsaved config. `POST /downloadclients/:id/test`, `POST /downloadclients/test`.
- Default download client per media type via the defaults API (section 1.4).
- Each client registered with health monitoring; scheduled connectivity check. `internal/scheduler/tasks/downloadclienthealth.go`.
- Downloads land in `<client default dir>/SlipStream/<Module plural name>` (e.g. `SlipStream/Movies`). `downloader/service.go` (`downloadSubDir`).
- Import delay seconds is honored before import. `internal/import/pipeline.go`.
- **[HALF]** Cleanup mode and seed ratio target are stored but nothing acts on them (no consumer outside CRUD and arr import mapping). The web UI does not expose them either.

### 9.2 Client types (factory in `internal/downloader/factory.go`, allow-list in `downloader/service.go` `validClientTypes`)
Each talks to the named external client's API.
- qBittorrent (`internal/downloader/qbittorrent`). Ext: qBittorrent Web API.
- Transmission (`internal/downloader/transmission`). Ext: Transmission RPC.
- Deluge (`internal/downloader/deluge`). Ext: Deluge JSON-RPC.
- rTorrent (`internal/downloader/rtorrent`). Ext: rTorrent XML-RPC.
- Vuze (`internal/downloader/vuze`, wraps Transmission RPC). Ext: Vuze.
- Flood (`internal/downloader/flood`). Ext: Flood REST.
- Aria2 (`internal/downloader/aria2`). Ext: Aria2 XML/JSON-RPC.
- uTorrent (`internal/downloader/utorrent`). Ext: uTorrent Web UI API.
- Hadouken (`internal/downloader/hadouken`). Ext: Hadouken JSON-RPC.
- Synology Download Station (`internal/downloader/downloadstation`). Ext: Synology API.
- Freebox Download (`internal/downloader/freeboxdownload`). Ext: Freebox OS API (HMAC challenge auth).
- RQBit (`internal/downloader/rqbit`). Ext: rqbit REST.
- Tribler (`internal/downloader/tribler`). Ext: Tribler REST.
- Mock (`internal/downloader/mock`), dev mode only, supports "fast forward".
- **[HALF]** SABnzbd: client package exists and the factory builds it, but `sabnzbd` is missing from `validClientTypes`, so it cannot be created through the API. The grab path also always calls `AddTorrent`, so usenet grabs are not wired end to end. README and parity plan both claim SABnzbd is supported; the web UI's client type list (`web/src/types/download-client.ts`) omits it too. `internal/downloader/sabnzbd`, `internal/indexer/grab/service.go` (`sendToClient`).
- **[HALF]** NZBGet: type constant only; factory returns "not yet implemented". `internal/downloader/factory.go`.

### 9.3 Queue
- Unified queue across all clients with title, release name, media type, status, progress, size, downloaded, speed, ETA, parsed quality/source/codec/attributes (HDR, Atmos, REMUX), season/episode, linked movie/series/episode, season pack flags, target slot name. `downloader/queue.go`, `GET /queue`.
- Pause, resume, remove (optionally delete files) a queue item; fast-forward a mock download. `POST /queue/:id/{pause,resume,fastforward}`, `DELETE /queue/:id`.
- Live queue push over WebSocket with adaptive polling (2s when active, 30s idle). `downloader/broadcaster.go`, event `queue:state`.
- Completion detection, per-download import attempt counting with backoff (1, 5, 15, 60 min; max 5 tries), failed download clears slot assignment and mapping. `downloader/completion.go`, `downloader/queue_media.go`.
- Disappeared downloads (removed from client while item was `downloading`) flip the item to failed. `downloader/completion.go` (`CheckForDisappearedDownloads`).
- Torrent added via indexer-fetched .torrent content first (for private trackers) with URL fallback. `internal/indexer/grab/service.go` (`sendViaIndexer`).

## 10. Notifications

Paths: `internal/notification/{service,factory,schema,handlers,events,types}.go`, providers under `internal/notification/*`, bridge in `internal/api/notification_bridge.go`.

### 10.1 Configuration
- CRUD notification connections: name, type, enabled, settings JSON, per-event toggles, include health warnings, tags (stored, unused). `notification/service.go`, `/notifications`.
- Test a saved or unsaved notification. `POST /notifications/:id/test`, `POST /notifications/test`.
- Schema endpoint drives the dynamic settings form (field types text, password, url, number, bool, select, action/oauth; advanced flag). `GET /notifications/schema`, `notification/schema.go`.
- Event catalog endpoint, built from framework events plus module-declared events. `GET /notifications/events`, `internal/module/framework_events.go`, `internal/modules/*/module.go` (`DeclareEvents`).
- Failure backoff: a failing notification is disabled with escalating `disabledTill` and cleared on success. `notification/service.go` (`recordFailure`, `isDisabled`).

### 10.2 Events
- `grab` (On Grab) dispatched from grabs. `internal/api/notification_bridge.go`.
- `import` (On Import) and `upgrade` (On Upgrade) from the import pipeline. `internal/import/service.go`.
- `movie:added`, `movie:deleted`, `tv:added`, `tv:deleted` from library services. `internal/library/{movies,tv}/service.go`.
- `health_issue`, `health_restored` from the health service. `internal/health/service.go`.
- Generic message: new portal requests are announced to all enabled admin notifications. `internal/portal/notifications/service.go` (`DispatchGenericMessage`).
- **[HALF]** `app_update` (On App Update) is declared in the catalog and every provider implements `OnApplicationUpdate`, but nothing dispatches it (no call site in `internal/update`).

### 10.3 Providers that send (factory cases in `notification/factory.go`)
- Discord: webhook URL, username, avatar, include posters; rich embeds with selectable fields. `internal/notification/discord`. Ext: Discord webhooks.
- Telegram: bot token, chat id, topic id, silent, include links. `internal/notification/telegram`. Ext: Telegram Bot API.
- Webhook: URL, method, basic auth, custom headers JSON; Sonarr/Radarr-style payload. `internal/notification/webhook`. Ext: any HTTP endpoint.
- Email: SMTP host, port, TLS, auth, from, to, cc, bcc. `internal/notification/email`. Ext: SMTP server.
- Slack: webhook URL, username, icon emoji, channel. `internal/notification/slack`. Ext: Slack incoming webhooks.
- Pushover: user key, API token, devices, priority, sound. `internal/notification/pushover`. Ext: Pushover API.
- Plex: "Connect to Plex" PIN OAuth, pick server, pick library sections, path mappings, update library on import/upgrade, partial (path-only) refresh. Refresh requests are queued and processed every 5 minutes. `internal/notification/plex/{oauth,client,notifier,service,handlers}.go`, `/notifications/plex/auth/start`, `/notifications/plex/auth/status/:pinId`, `/notifications/plex/servers`, `/notifications/plex/servers/:id/sections`, task `plex-refresh` in `internal/scheduler/tasks/plexrefresh.go`. Ext: plex.tv, app.plex.tv, Plex Media Server.
- Mock (dev mode): records sent notifications and broadcasts `notification:mock`. `internal/notification/mock`.

### 10.4 Declared but not implemented
- **[HALF]** Gotify, ntfy, Pushbullet, Join, Apprise, Custom Script have schemas in `notification/schema.go` (so they appear in the UI form list via `GET /notifications/schema`) but the factory returns "unsupported notifier type". Prowl, Simplepush, Signal exist only as type constants in `internal/notification/types/types.go`.

## 11. Multiple quality versions ("slots")

Paths: `internal/library/slots/{slot,service,handlers,assignment,status,naming,search,fileops,migration,debug}.go`, `internal/library/quality/exclusivity.go`, `internal/autosearch/slots.go`, `internal/import/seasonpack.go`, `internal/api/slot_bridge.go`. Spec: `docs/multiple-quality-versions-spec.md` (+ plan and test suite docs).
- Master toggle for multi-version mode, with dry-run-completed flag and last migration timestamp. `GET/PUT /slots/settings`.
- Up to 3 slots, each with custom name, enabled flag, assigned quality profile, display order, per-module root folder override. `/slots`, `/slots/:id`, `/slots/:id/enabled`, `/slots/:id/profile`.
- Configuration validation: enabled slots' profiles must be mutually exclusive (non-overlapping qualities or conflicting required attributes), with per-attribute issue messages. `POST /slots/validate`, `quality/exclusivity.go`.
- Naming validation: filename templates must contain differentiating tokens so slot files do not collide. `POST /slots/validate-naming`, `slots/naming.go`.
- Evaluate which slot a release would land in (match score, upgrade vs new fill). `POST /slots/evaluate`.
- Disable a slot with an action for its files (`delete`, `keep`, `cancel`) after a pre-check of impact. `GET /slots/:id/disable-check`, `POST /slots/:id/disable-with-action`.
- Change a slot's profile with an action (`keep`, `reevaluate`, `cancel`) after a pre-check. `GET /slots/:id/profile-change-check`, `POST /slots/:id/profile-with-action`.
- Migration to multi-version: dry-run preview of how existing files map to slots, execute migration. `POST /slots/migration/preview`, `POST /slots/migration/execute`, `slots/migration.go` (a `migration.go.backup` file sits next to it).
- Review queue for files that could not be auto-assigned: list, stats, item details, resolve (`assign`, `delete`, `skip`). `/slots/review-queue*`.
- Per-movie and per-episode slot views: assignments, status per slot, assign a file to a slot, unassign, set per-slot monitored. `/slots/movies/:movieId/*`, `/slots/episodes/:episodeId/*`.
- Per-slot missing/upgrade status rolls up to the item status. `slots/status.go`.
- Per-slot autosearch and grab with `targetSlotId`; queue shows target slot; failed download clears slot assignment. Sections 7, 8.1, 9.3.
- History events `slot_assigned`, `slot_reassigned`, `slot_unassigned`. `internal/history/types.go`.
- Dev-mode debug endpoints: parse release, profile match, simulate import, generate preview. `slots/debug.go`, `/slots/debug/*` (gated by dev mode).
- Dev mode auto-creates test slots and profiles. `internal/api/devmode.go` (`setupDevModeSlots`), `docs/configured-test-slots-and-profiles.md`.

## 12. Health monitoring

Paths: `internal/health/{service,handlers,types,storage,filesystem}.go`, tasks in `internal/scheduler/tasks/{downloadclienthealth,indexerhealth,prowlarrhealth,storagehealth}.go`. Specs: `docs/system-health-spec.md`, `docs/system-health-implementation-plan.md`.
- Health items grouped by category: download clients, indexers, Prowlarr, root folders, metadata, storage, import. Status ok/warning/error with message and timestamp. `health/types.go`.
- Get all health, summary per category (ok/warning/error counts, hasIssues), get one category. `GET /system/health`, `/system/health/summary`, `/system/health/:category`.
- Test a whole category or a single item (download client test, indexer test, root folder path check, TMDB/TVDB test). `POST /system/health/:category/test`, `POST /system/health/:category/:id/test`, wiring in `internal/api/routes.go`.
- Scheduled checks: download clients (default every 6h), indexers (6h), Prowlarr, storage (1h, warn under 20% free, error under 5%). Intervals and thresholds are config-file only (`health.*`). `internal/config/config.go`.
- Other health sources: import failures, RSS feed failures, vanished files after scan.
- Health changes broadcast `health:updated` and fire Health Issue / Health Restored notifications (warnings only if the connection opts in). `health/service.go`.
- Port and firewall check: is the server port listening and does the OS firewall allow it (Windows Firewall rules, macOS application firewall incl. stealth/block-all, Linux ufw/firewalld/iptables). `internal/firewall/*`, `GET /system/firewall`.

## 13. Calendar, history, activity

- Calendar events for a date range (`start`, `end`), aggregated from module calendar providers: movie theatrical/digital/physical release dates and episode air dates, with status and monitored flag. `internal/calendar/{service,handlers}.go`, `internal/modules/{movie,tv}/calendar.go`, `GET /calendar`. No iCal/ICS feed exists.
- History log of events: `grabbed`, `imported`, `deleted`, `failed`, `file_renamed`, `autosearch_download`, `autosearch_failed`, `import_failed`, `slot_assigned`, `slot_reassigned`, `slot_unassigned`, `status_changed`; each with media type/id, title, quality, source, and typed data (release, indexer, client, upgrade old/new quality, paths, error, slot). `internal/history/{types,service,handlers}.go`.
- List history with pagination and filters (event type, media type, before, after); clear all history. `GET /history`, `DELETE /history`.
- History retention settings (enabled, retention days) and a nightly cleanup task at 02:00. `history/settings.go`, `GET/PUT /history/settings`, task `history-cleanup`.
- Live `history:added` WebSocket event. `history/service.go`.
- Activity/progress tracker for long jobs (scan, download, import, metadata refresh, file operation) with pending/in-progress/completed/failed/cancelled states, broadcast as `progress:*`. `internal/progress/progress.go`.

## 14. Scheduled tasks

Paths: `internal/scheduler/scheduler.go` (gocron wrapper), `internal/scheduler/tasks/*`, `internal/api/handlers/scheduler.go`, registrations in `internal/api/setters.go` and `internal/api/server.go`.
- List tasks (id, name, description, cron, last run, next run, running), get one, run now. `GET /scheduler/tasks`, `GET /scheduler/tasks/:id`, `POST /scheduler/tasks/:id/run`. WebSocket `scheduler:task:started|completed`.
- Registered tasks:
  - `status-refresh` "Refresh Media Status", hourly, run on start.
  - `autosearch` "Automatic Search", every N hours from settings.
  - `rss-sync` "RSS Sync", every N minutes from settings, run on start.
  - `update-check` "Update Check", daily 00:00, run on start.
  - `plex-refresh` "Plex Library Refresh", every 5 min.
  - `library-scan` "Library Scan", daily 23:30.
  - `unreleased-refresh` "Refresh Unreleased Metadata", daily 04:00.
  - `download-client-health`, `indexer-health`, `prowlarr-health`, `storage-health`, all run on start with config intervals.
  - `import-scan` "Import Scan", every 5 min, run on start.
  - `history-cleanup` "History Cleanup", daily 02:00.
- Task schedules are not user-editable (only autosearch and RSS intervals via their settings). Module `ScheduledTasks()` hooks return nil for both modules.
- Bug history: next-run and last-run columns showed the same value (fixed via cron-based next run). `docs/bugs.md`, `scheduler.go` (`nextRunFromCron`).

## 15. Admin auth and API security

Paths: `internal/api/auth_handlers.go`, `internal/auth/{service,handlers,passkey,passkey_handlers}.go`, `internal/api/middleware/security.go`, `internal/api/ratelimit/auth_limiter.go`, `internal/portal/users`. Plan: `docs/passkey-implementation-plan.md`.
- First-run setup: create the single admin with a 4-digit PIN; allowed only from localhost. `POST /auth/setup`, `GET /auth/status` (requiresSetup).
- Admin login: username `Administrator` + PIN through the shared login endpoint; returns a 24h admin JWT (audience admin). All admin API calls use `Authorization: Bearer`. README's "session cookies for admins" is inaccurate. `internal/auth/handlers.go` (`handleAdminLogin`), `internal/auth/service.go`.
- **[HALF]** The admin login route lives under `/requests/auth/*`, which is wrapped by the portal-enabled middleware, so disabling the requests portal appears to block admin login. Open bug "admin login redirect issue" in `docs/bugs.md` may be related. `internal/api/routes.go` (`setupPortalRoutes`).
- Verify PIN for re-confirmation of sensitive actions; profile get/update (username, password). `POST /requests/auth/verify-pin`, `/requests/auth/profile`.
- Passkeys (WebAuthn): register begin/finish, login begin/finish, list/rename/delete credentials. RP id, origins, display name editable in settings. `internal/auth/passkey*.go`, `internal/api/webauthn_settings.go`. Passkey UI hidden when the context is not secure (`docs/bugs.md`).
- Rate limit on auth endpoints (10 req/min per IP) and progressive account lockout after 5 failures (15 min, 30 min, 1 h). `internal/api/ratelimit/auth_limiter.go`.
- Middleware: panic recovery, request id, security headers (nosniff, frame options, XSS, referrer policy, CSP frame-ancestors), 2 MB body limit, same-origin CORS, request logging, proxy-probe blocking, gzip. `internal/api/routes.go`, `internal/api/middleware/security.go`.
- SPA served only for known route prefixes; other paths 404. `cmd/slipstream/main.go` (`registerFrontendHandler`).
- **[HALF]** API key: generated and regenerable (`POST /settings/apikey`) and shown in settings, but no middleware accepts it, so it authenticates nothing. The web client has `regenerateApiKey` in `web/src/api/system.ts` but no component calls it. `internal/api/handlers_system.go`.
- **[TODO-DOC]** Credential encryption at rest: checklist says a crypto package exists, but there is no `internal/crypto` in the tree. Secrets are plaintext in SQLite; API responses redact download-client secrets only.
- JWT secret auto-generated if unset; copied into dev DB on dev-mode toggle. `internal/config`, `internal/api/devmode.go`.

## 16. WebSocket live updates

Paths: `internal/websocket/hub.go`, endpoint `GET /ws` registered in `cmd/slipstream/main.go`.
- Single hub; admin JWT passed in `Sec-WebSocket-Protocol`; same-hostname origin check; ping/pong keepalive.
- Message envelope: `type`, `payload`, `timestamp`, plus `module`, `entityType`, `entityId`, `action` for entity events.
- Events emitted: `movie:added|updated|deleted`, `series:added|updated|deleted` (module-tagged entity events), `library:updated`, `queue:state`, `progress:started|update|completed|error|cancelled`, `search:started|completed`, `grab:started|completed`, `autosearch:started|completed|failed`, `autosearch:task:started|progress|completed`, `rss-sync:started|progress|completed|failed`, `import:completed`, `import:failed`, `download:completed`, `rename:completed`, `history:added`, `health:updated`, `indexer:status`, `artwork:ready`, `scheduler:task:started|completed`, `update:status`, `logs:entry`, `request:created|updated|deleted`, `portal:inbox:created`, `notification:mock`, `devmode:changed|error`.
- Client-to-server message: `devmode:set {enabled}` toggles developer mode. `websocket/hub.go` (`handleIncoming`).

## 17. System settings, logs, server control

- General settings: server port, log level, log rotation (max size MB, backups, age days, compress), log path (read-only), external access toggle (default binds 127.0.0.1 port 8080; toggle binds 0.0.0.0; the checklist's "8888" is wrong), WebAuthn RP settings, API key. Persisted in DB settings table and applied at next start. `internal/api/handlers_system.go`, `GET/PUT /settings`.
- Status endpoint: version, start time, movie/series counts, developer mode, dev build, portal enabled, requires setup, actual port and configured port (when it fell back), mediainfo available, enabled modules, TMDB ordering. `GET /status`; unauthenticated liveness `GET /health`.
- Port conflict handling: try up to 10 ports from the configured one and report both. `internal/config` (`FindAvailablePort`), `cmd/slipstream/main.go`.
- Restart the server (re-exec self). `POST /system/restart`.
- Enable/disable modules (movie, tv). `GET/PUT /settings/modules`, `internal/api/handlers_system.go`.
- Logs: recent entries from an in-memory ring buffer (1000), live stream via `logs:entry`, download the current log file. File logs rotate via lumberjack. `internal/logger/*`, `internal/api/logs.go`, `GET /system/logs`, `GET /system/logs/download`.
- Bootstrap and updater diagnostic logs written before the main logger exists. `cmd/slipstream/main.go`.
- Network services init retries with exponential backoff when offline at boot. `internal/startup/retry.go`.
- Config sources: env (`SLIPSTREAM_*`), `.env`, `config.yaml` in platform paths, defaults; metadata keys and version can be baked in via ldflags. `internal/config/config.go`, `configs/config.example.yaml`, `Makefile`.
- Config backup and restore: not started. **[TODO-DOC]**

## 18. Developer mode, mocks, virtual filesystem

Paths: `internal/api/devmode.go`, `internal/database/manager.go`, `internal/api/switchable.go`, mocks in `internal/{metadata,indexer,downloader,notification,filesystem}/mock`, module mock factories `internal/modules/{movie,tv}/mock_factory.go`, tooling `cmd/fetchmockdata`, `cmd/convertmockdata`. Guide: `docs/developer-mode-testing.md`.
- Toggle at runtime (WebSocket `devmode:set`) or at startup (`--dev-mode` flag or `SLIPSTREAM_DEV_MODE=1`, `make dev-mode`). Status reported in `/status` (`developerMode`, `isDevBuild`). UI toggle is hidden outside dev builds (`docs/bugs.md`).
- Switches to a separate `*_dev.db` database, runs module migrations there, copies JWT secret, core settings (port, log level, API key), quality profiles, portal users with their module settings, and portal user notification channels from prod. `devmode.go`.
- Swaps in mock TMDB, TVDB, OMDb clients (fuzzy search fallback), creates a mock indexer (catalog + generated releases with varied quality attributes), a mock download client (simulated progress, pause/resume, fast-forward), and a mock notification (records sends, broadcasts `notification:mock`).
- Virtual filesystem for root folders and browse/import, pre-populated with multi-version and partial TV content. `internal/filesystem/mock/virtualfs.go`.
- Populates sample library media through each module's `MockFactory`, and creates test slots and profiles. `devmode.go` (`populateMockMedia`, `setupDevModeSlots`), `docs/configured-test-slots-and-profiles.md`.
- Prowlarr mode is overridden in dev mode. `internal/prowlarr/mode.go`.
- Dev-only slot debug endpoints (section 11).
- `fetchmockdata` pulls real TMDB/TVDB/indexer data to regenerate mock fixtures; `convertmockdata` converts indexer results to Go maps. Ext: TMDB, TVDB, indexers (dev tooling only).

## 19. Updates

Paths: `internal/update/{service,handlers,version,detach_*}.go`, `internal/scheduler/tasks/update.go`, updater entry in `cmd/slipstream/main.go` (`--complete-update`).
- Update status with states `idle`, `checking`, `up-to-date`, `update-available`, `error`, `downloading`, `installing`, `restarting`, `complete`, `failed`; latest release info (version, notes, asset, size, published), download progress MB, last checked. `GET /update`, WebSocket `update:status`.
- Check now, install, cancel. `POST /update/check`, `/update/install`, `/update/cancel`.
- Auto-install setting. `GET/PUT /update/settings`. **[TODO-DOC]** UI toggle missing per `docs/Production-Readiness-Checklist.md`.
- Daily check at midnight and on start; picks the platform asset (Windows `.zip`, macOS `.dmg`, Linux `.tar.gz` or `.AppImage` when running as AppImage). Ext: GitHub Releases API (`api.github.com/repos/jatassi/SlipStream-v1/releases/latest`).
- Self-replace flow: download, extract, spawn updater with `--complete-update <target> <port>`, wait for ports to free, copy binary or `.app` bundle, relaunch detached, clean temp dir. Windows handles locked exe by rename; warns when installed in Program Files by the old NSIS installer.
- **[HALF]** Pre-update database backup is a log line only (`update/service.go` `backupDatabase`).
- Linux deb/rpm use a stub launcher that bootstraps the real binary into `~/.local/share/slipstream/bin` so updates need no sudo. `cmd/stub/main.go`.

## 20. Platform packaging and desktop integration

Paths: `internal/platform/{app,app_windows,app_darwin,app_linux,util}.go`, `cmd/slipstream/{main,detach_windows,detach_other}.go`, `scripts/{windows,macos,linux}`, `.github/workflows/release.yml`. Plans: `docs/Windows-Application-Plan.md`, `docs/macOS-Application-Plan.md`, `docs/Linux-Application-Plan.md`.
- Windows: system tray icon (left click opens browser; menu Open / Quit), embedded icon and manifest, MSI installer (WiX v5) into `%LOCALAPPDATA%\SlipStream` with Start Menu, optional Desktop, and Startup-folder shortcuts, launch after install, data left on uninstall; portable ZIP. `scripts/windows/{installer.wxs,winres.json,slipstream.manifest}`.
- macOS: menu bar item (Open SlipStream, Enable/Disable Start at Login, Quit), LaunchAgent plist for login start, DMG per arch (arm64, amd64). `internal/platform/app_darwin.go`, `scripts/macos/{create-dmg.sh,com.slipstream.slipstream.plist}`.
- Linux: no tray (console mode), xdg-open for browser, deb/rpm via nfpm with stub launcher, AppImage, portable tarballs for amd64 and arm64, systemd user unit shipped as a doc with manual enable steps. `scripts/linux/{create-appimage.sh,postinstall.sh,slipstream.service,slipstream.desktop}`. **[HALF]** `EnableStartup`/`DisableStartup` for a systemd user service exist in `app_linux.go` but nothing calls them.
- `--no-tray` flag for console mode on any OS.
- First run opens the browser automatically. `platform.IsFirstRun`, `cmd/slipstream/main.go`.
- OS-standard data and log paths (Windows `%LOCALAPPDATA%\SlipStream`, macOS `~/Library/Application Support/SlipStream` and `~/Library/Logs/SlipStream`, Linux XDG `~/.config/slipstream` or `~/.local/share/slipstream` for packaged installs). `internal/config/config.go`.
- Release CI builds all of the above on tag/dispatch and uploads to GitHub Releases. `.github/workflows/release.yml`. Code signing and Docker image are absent (no Dockerfile). **[TODO-DOC]** `docs/Production-Readiness-Checklist.md` still lists these packaging items as unchecked, so the checklist is stale for everything except signing and Docker.
- Firewall: read-only check (section 12); no firewall rule creation.

## 21. Module system

Paths: `internal/module/*` (interfaces, registry, schema, migrate, cascade, status, delete, validate, parseutil, titleutil, moduletest), implementations `internal/modules/{movie,tv,shared}`, scaffolder `scripts/new-module`, CLI `slipstream validate-module <id>`. Docs: `docs/module-system-spec.md`, `docs/module-system-plan-phase0..11.md`, `docs/adding-a-module.md`.
- Promise: a media type is a self-contained Go package implementing 16 required interfaces (descriptor + node schema, metadata provider, search strategy, import handler, path generator, naming provider, calendar provider, quality definition, wanted collector, monitoring presets, file parser, mock factory, notification events, release-date resolver, route provider, task provider) plus optional portal provisioner, slot support, arr import adapter. New modules (music, books, comics, games) slot in without touching core. `internal/module/interfaces.go`, `optional_interfaces.go`.
- Delivered: movie (flat node schema) and tv (series, season, episode) modules registered in a central registry; per-module DB migrations with their own version tables; module-declared notification events, naming contexts, calendar items, monitoring presets, wanted collectors, mock data, arr import adapters, portal provisioners; module enable/disable toggle; generic entity WebSocket events; `make new-module MODULE_ID=x` scaffolding; `validate-module` static checks.
- **[HALF]** `RegisterRoutes` and `ScheduledTasks` are empty for both modules; all movie/TV HTTP routes and tasks still live in framework packages (`internal/library/{movies,tv}`, `internal/scheduler/tasks`).
- **[HALF]** Legacy non-module code paths remain beside module paths: `collectLegacy` in `autosearch/scheduled.go`, `collectWantedLegacy` in `rsssync/service.go`, `legacyRefresh` in `librarymanager/metadata_refresh.go`, a TODO to dispatch file lookup to modules in `import/service.go`, and `library/status/status.go` says its constants "will be replaced by module.Status*". Quality tiers are still the global video list, not module-declared (`internal/modules/shared/video_qualities.go`, `library/quality/profile.go`).
- No third module exists to prove the abstraction.
- `docs/fix-queue-import-module-bypass.md` reports that queue-triggered imports fell through to legacy matching and naming. The module path (`import/module_matching.go`, `computeDestinationViaModule`) is now reachable from `import/matching.go`, so it looks fixed, but the doc carries no resolved marker.

## 22. Requests portal (admin-managed side)

Paths: `internal/portal/{admin,users,invitations,quota,autoapprove,requests,notifications,library,search,middleware,ratelimit}`, `internal/api/portal_bridge.go`. Specs: `docs/external-requests-spec.md`, `docs/external-requests-implementation-plan.md`, `docs/portal-in-library-search-result-enhancement-spec.md`.
Admin capabilities (under `/admin/requests`):
- Portal settings: enabled, default per-module quotas, default root folder, notify admin on new requests, portal search rate limit. `portal/admin/settings_handlers.go`.
- Users: list, get, update (username, per-module quality profile, auto-approve), enable, disable, delete; per-user quota get/set/clear per module. `portal/admin/users_handlers.go`, `portal/quota`.
- Invitations: list, create (username, per-module profiles, auto-approve), get, delete, resend; invitation links with token and expiry (no email delivery). `portal/admin/invitations_handlers.go`, `portal/invitations`.
- Requests: list, get, approve (with `auto_search` or `manual_search` action and optional root folder), deny with reason, delete, batch approve/deny/delete; shows whether the item is already in library. `portal/admin/requests_handlers.go`.
- Auto-approve rules: user flag plus quality profile `allowAutoApprove`; approved requests are provisioned into the library via the module's portal provisioner and searched. `portal/autoapprove/service.go`.
- Request status lifecycle tracked from library events: `pending`, `approved`, `denied`, `searching`, `downloading`, `failed`, `available`. `portal/requests/{service,status_tracker}.go`.
- New-request alert to admin notification channels. `portal/notifications/service.go`.
End-user portal surface (for reference, likely its own inventory): invite signup, login, passkeys, profile, TMDB/TVDB search with in-library flags, library browse, create/cancel/watch requests, view request downloads, personal notification channels with test, in-app inbox with unread count. `/requests/*`.

## 23. Data, persistence, CLI

- Single SQLite DB (pure Go `modernc.org/sqlite`, WAL), 77 framework Goose migrations embedded plus per-module migration tracks; sqlc queries; separate dev DB. `internal/database/{manager,database}.go`, `internal/database/migrations`, `internal/modules/*/migrations`, `internal/module/migrate.go`.
- Services swap DB handle on dev-mode toggle (`SetDB` on every service). `internal/api/switchable.go`, `internal/api/setters.go`.
- CLI: `slipstream [--config path] [--no-tray] [--dev-mode]`, `slipstream validate-module <id>`, `slipstream --complete-update <target> <port>` (internal). `cmd/slipstream/main.go`.

## 24. Not started or absent (parity scoping aids)

- **[TODO-DOC]** Config backup and restore; update dashboard; transcode during import (FFmpeg + slots); TRaSH Guides import; onboarding wizard finishing touches (in progress). `docs/Remaining-features.md`.
- Absent in code, present in Sonarr/Radarr: custom formats engine, release blocklist, delay profiles, tags-based routing (tags field stored on notifications only), import lists, iCal feed, remote path mappings for download clients (only Plex has path mappings), editable scoring weights, recycle bin, NZBGet, working usenet grab path.

