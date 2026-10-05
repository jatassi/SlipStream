# SlipStream v2 parity matrix

This matrix lists every user-observable SlipStream v1 feature and the v2 call on each. It is built from the v1 backend and UI inventories and the charting answers in [Chart SlipStream v2](https://github.com/jatassi/SlipStream/issues/2). The parity rule counts every feature a v1 user can see or use, working or half-built, plus every item v1's docs list as not started. A row leaves parity only as an exception, and an exception needs the chef's explicit yes recorded on its row. A spec PR sets the Covered by cell of each row it owns to that spec, and sets the Status cell when the feature ships.

## Legend

**ID.** A stable `P-###` number, sequential across the file and never reused.

**v1 path.** The route, file or package the inventory cites. A `-` means v1 has no code for it.

**v1 state.**

- `working` means the feature works in v1.
- `half-built` means v1 has part of it, stubs it, or stores a setting nothing reads.
- `not started` means v1's docs list it as not started, or v1 lacks it and a charting answer adds it.

**v2 disposition.**

- `kept` means v2 ships the behavior. The look may change under the UI redesign.
- `superseded by <v2 feature>` means a charting answer already names the v2 feature that replaces it.
- `open` means a map ticket still decides it. The Covered by cell names that ticket.
- `out of scope` means the charting rules already exclude it.

An exception is written on its row only with the chef's explicit yes. No row carries one yet.

**Covered by.** The map ticket that settles or owns the row, or later the spec that builds it. A `-` means the row waits for the spec list.

**Status.**

- `unplanned` means no spec covers the row yet.
- `shipped` means the feature is live in v2.

## App shell, dashboard and global search

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-001 | Wide shell with sidebar and header, phone shell with bottom tabs and push navigation | `web/src/components/layout/shell-layout.tsx`, `phone-shell.tsx` | working | kept | [UI redesign direction](https://github.com/jatassi/SlipStream/issues/25) | unplanned |
| P-002 | Sidebar nav built from enabled Content Types plus Discover, Settings and System groups | `web/src/components/layout/sidebar-nav-config.ts` | working | kept | [UI redesign direction](https://github.com/jatassi/SlipStream/issues/25) | unplanned |
| P-003 | Activity nav link shows a live download count and progress | `web/src/components/layout/downloads-nav-link.tsx` | working | kept | - | unplanned |
| P-004 | Phone tab bar with Dashboard, Library, Activity badge, Search and More | `web/src/components/layout/tab-bar.tsx` | working | kept | [UI redesign direction](https://github.com/jatassi/SlipStream/issues/25) | unplanned |
| P-005 | Phone More page with Discover links, missing counts, settings, system and version footer | `web/src/routes/more/index.tsx` | working | kept | [UI redesign direction](https://github.com/jatassi/SlipStream/issues/25) | unplanned |
| P-006 | Header search box opens global search results | `web/src/components/search/search-bar.tsx` | working | kept | - | unplanned |
| P-007 | Header running-tasks pill lists running scheduled tasks | `web/src/components/layout/header-running-tasks.tsx` | working | kept | - | unplanned |
| P-008 | Header activity indicator shows live progress of long jobs, each dismissible | `web/src/components/layout/header-activity-indicator.tsx`, `internal/progress/progress.go` | working | kept | - | unplanned |
| P-009 | Header bell for in-app admin notifications, held client side with no history | `web/src/components/layout/header-notifications.tsx` | half-built | kept | - | unplanned |
| P-010 | Restart action with confirm, server re-exec and countdown reload | `web/src/components/layout/use-session-actions.ts`, `POST /system/restart` | working | kept | - | unplanned |
| P-011 | Log out action with confirm | `web/src/components/layout/use-session-actions.ts` | working | kept | - | unplanned |
| P-012 | Light, dark and system theme with no picker in the UI | `web/src/stores/ui.ts` | half-built | kept | [UI redesign direction](https://github.com/jatassi/SlipStream/issues/25) | unplanned |
| P-013 | Toasts placed per shell | `web/src/components/layout/shell-layout.tsx` | working | kept | - | unplanned |
| P-014 | Document title tracks the route | `web/src/hooks/use-document-title.ts` | working | kept | - | unplanned |
| P-015 | Error boundary per route | `web/src/components/error-boundary.tsx` | working | kept | - | unplanned |
| P-016 | Phone affordances such as safe areas, collapsing large titles, 44 px targets and action sheets | `docs/admin-native-overhaul-spec.md`, `web/src/components/screen/screen.tsx` | working | kept | [UI redesign direction](https://github.com/jatassi/SlipStream/issues/25) | unplanned |
| P-017 | Logo and color palette, in progress on the roadmap | `docs/Remaining-features.md` | half-built | kept | [UI redesign direction](https://github.com/jatassi/SlipStream/issues/25) | unplanned |
| P-018 | Dashboard with Downloading, Health, Recent and Storage groups | `web/src/routes/index.tsx`, `web/src/components/dashboard/` | working | kept | - | unplanned |
| P-019 | Global search matches library movies and series instantly by title | `web/src/routes/search/use-search-page.ts` | working | kept | - | unplanned |
| P-020 | Global search Add New section searches TMDB and links to the add flow or the library item | `web/src/routes/search/search-groups.tsx` | working | kept | - | unplanned |

## Library

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-021 | Root folders per Content Type with path validation, free space and delete | `internal/library/rootfolder/service.go`, `web/src/routes/settings/media/root-folders.tsx` | working | kept | - | unplanned |
| P-022 | Default root folder per Content Type | `rootfolder/service.go (SetDefault)` | working | kept | - | unplanned |
| P-023 | Adding a root folder triggers a scan of it | `internal/api/routes.go (SetOnFolderCreated)` | working | kept | - | unplanned |
| P-024 | Library scan parses files, matches metadata and creates movies and series | `internal/library/librarymanager/scanning.go`, `librarymanager/matching.go` | working | kept | - | unplanned |
| P-025 | Scan one folder or all folders on demand with status and cancel, API only | `/rootfolders/:id/scan`, `/scans` | working | kept | - | unplanned |
| P-026 | Vanished files flip the item to missing and raise a health alert | `librarymanager/scanning.go (VerifyFileExistence)` | working | kept | - | unplanned |
| P-027 | Server filesystem browser for picking folders across drives and volumes | `internal/filesystem/service.go`, `/filesystem/browse` | working | kept | - | unplanned |
| P-028 | Storage usage per volume holding root folders | `internal/filesystem/storage_service.go`, `/filesystem/storage` | working | kept | - | unplanned |
| P-029 | MediaInfo probe fills codec, HDR and channels when the mediainfo binary exists, with status shown | `internal/mediainfo/`, `media-info-status.tsx` | working | kept | - | unplanned |
| P-030 | Six-state item status (unreleased, missing, downloading, failed, upgradable, available) with message | `internal/library/status/status.go` | working | kept | - | unplanned |
| P-031 | Unreleased items turn missing hourly from release and air dates | `internal/availability/service.go` | working | kept | - | unplanned |
| P-032 | Added By shows the portal user who requested an item | `internal/library/movies/movie.go` | working | kept | - | unplanned |
| P-033 | Library list grid or table view with poster sizes and a column chooser | `web/src/components/media/library-options.tsx` | working | kept | [UI redesign direction](https://github.com/jatassi/SlipStream/issues/25) | unplanned |
| P-034 | Library sort options with list sections grouped by the sort field | `web/src/routes/movies/use-movie-list.ts`, `web/src/lib/grouping.ts` | working | kept | - | unplanned |
| P-035 | Library status filters, plus Continuing and Ended for series | `web/src/routes/movies/movie-list-utils.ts`, `series-list-utils.ts` | working | kept | - | unplanned |
| P-036 | Content Type switch at the top of the library when two or more are enabled | `web/src/components/media/library-screen.tsx` | working | kept | - | unplanned |
| P-037 | Per-row Search and Delete in table view | `web/src/lib/movie-columns.tsx`, `series-columns.tsx` | working | kept | - | unplanned |
| P-038 | Refresh metadata for one item or the whole Content Type, including season and episode sync | `librarymanager/metadata_refresh.go`, `POST /movies/refresh`, `POST /series/refresh` | working | kept | - | unplanned |
| P-039 | Bulk edit mode to monitor, unmonitor, set quality profile or delete with files | `web/src/components/media/library-edit-bar.tsx`, `PUT /movies/monitor` | working | kept | - | unplanned |
| P-040 | Poster status pill and dot | `web/src/components/media/status-pill.tsx` | working | kept | - | unplanned |
| P-041 | Add movie with root folder, profile, monitored and search on add, filling dates, rating and artwork | `internal/library/librarymanager/add.go`, `web/src/routes/movies/add.tsx` | working | kept | - | unplanned |
| P-042 | Add series with monitor preset, search on add, season folder and include specials | `librarymanager/add.go`, `web/src/routes/series/add.tsx` | working | kept | - | unplanned |
| P-043 | Add flow remembers last choices server side | `internal/preferences/`, `/preferences/addflow` | working | kept | - | unplanned |
| P-044 | Default quality profile, download client and indexer per Content Type | `internal/defaults/`, `/defaults/:entityType/:mediaType` | working | kept | - | unplanned |
| P-045 | Movie detail hero with backdrop, logo, status, year, runtime, genres and IMDb rating | `web/src/routes/movies/$id.tsx`, `media-detail-hero.tsx` | working | kept | - | unplanned |
| P-046 | Detail action row with monitor toggle, inline auto search states and manual search | `web/src/components/search/media-search-monitor-controls.tsx` | working | kept | - | unplanned |
| P-047 | Detail menu to edit, refresh metadata or delete with optional file deletion | `web/src/components/media/media-detail-menu.tsx`, `/movies/:id` | working | kept | - | unplanned |
| P-048 | Movie detail file list with media info, details group and cast strip | `web/src/routes/movies/movie-detail-groups.tsx`, `movie-detail-credits.tsx` | working | kept | - | unplanned |
| P-049 | Series detail with episode status counts, production status, network and creator | `web/src/routes/series/series-detail-groups.tsx` | working | kept | - | unplanned |
| P-050 | Season list with per-season and per-episode monitor and search | `web/src/components/series/season-list.tsx` | working | kept | - | unplanned |
| P-051 | Series monitoring presets (all, none, future, first, latest, existing, specials) with stats | `PUT /series/:id/monitor`, `internal/library/tv/series.go` | working | kept | - | unplanned |
| P-052 | File records carry quality, codecs, channels, dynamic range and resolution | `/movies/:id/files`, `/series/:id/episodes/:episodeId/files` | working | kept | - | unplanned |
| P-053 | Missing and Upgradable page per Content Type with inline search controls | `internal/missing/`, `web/src/routes/missing/index.tsx` | working | kept | - | unplanned |

## Metadata

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-054 | TMDB provider for movie and series search, details, credits, ratings, logos and artwork | `internal/metadata/tmdb` | working | kept | - | unplanned |
| P-055 | TVDB provider for series lookup and search fallback | `internal/metadata/tvdb` | working | kept | - | unplanned |
| P-056 | OMDb provider for IMDb, Rotten Tomatoes and Metacritic ratings and awards | `internal/metadata/omdb` | working | kept | - | unplanned |
| P-057 | Extended detail with cast, crew, trailer, external ratings and per-episode IMDb ratings | `/metadata/movie/:id/extended`, `/metadata/series/:id/extended` | working | kept | - | unplanned |
| P-058 | Digital, physical and theatrical release dates, content rating, studio and network logos | `internal/metadata/service.go`, `network_logo_store.go` | working | kept | - | unplanned |
| P-059 | Artwork download and serving for poster, backdrop, logo and studio logo | `internal/metadata/artwork.go`, `/metadata/artwork/:type/:id/:artworkType` | working | kept | - | unplanned |
| P-060 | Metadata response cache with clear on demand | `internal/metadata/cache.go`, `DELETE /metadata/cache` | working | kept | - | unplanned |
| P-061 | Metadata provider status and connection tests | `GET /metadata/status` | working | kept | - | unplanned |
| P-062 | TMDB search ordering toggle, API only | `POST /metadata/tmdb/search-ordering` | working | kept | - | unplanned |
| P-063 | Metadata API keys bundled at build with config override | `internal/config`, `Makefile` | working | kept | - | unplanned |

## Quality

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-064 | Quality profiles per Content Type with create, edit and delete | `internal/library/quality/service.go`, `/qualityprofiles`, `web/src/routes/settings/media/quality-profiles.tsx` | working | kept | - | unplanned |
| P-065 | Eighteen predefined qualities with cinema source opt-in | `internal/library/quality/profile.go` | working | kept | - | unplanned |
| P-066 | Cutoff, upgrades toggle, upgrade strategy and cutoff overrides strategy | `internal/library/quality/profile.go` | working | kept | - | unplanned |
| P-067 | Live upgrade strategy preview of allowed and blocked upgrades | `web/src/components/qualityprofiles/upgrade-strategy-preview.tsx` | working | kept | - | unplanned |
| P-068 | Allow Auto-Approve flag for portal requests on a profile | `internal/library/quality/profile.go` | working | kept | - | unplanned |
| P-069 | Attribute rules for HDR, video codec, audio codec and channels as required, preferred, acceptable or not allowed | `internal/library/quality/attributes.go` | working | kept | - | unplanned |
| P-070 | Default Any profile created per Content Type | `internal/library/quality/service.go (EnsureDefaults)` | working | kept | - | unplanned |
| P-071 | Release scoring with a breakdown and fixed weights | `internal/indexer/scoring/scorer.go` | working | kept | [Custom formats engine and TRaSH Guides import](https://github.com/jatassi/SlipStream/issues/7) | unplanned |
| P-072 | Best release selection against profile, current file and upgrade rules | `internal/decisioning/selection.go` | working | kept | - | unplanned |
| P-073 | Custom Formats naming token and notification fields, always empty | `internal/import/renamer/tokens.go`, `internal/notification/types/types.go` | half-built | kept | [Custom formats engine and TRaSH Guides import](https://github.com/jatassi/SlipStream/issues/7) | unplanned |

## Import and renaming

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-074 | Automatic import of completed downloads mapped back to the grabbed item | `internal/import/pipeline.go (ProcessCompletedDownload)` | working | kept | - | unplanned |
| P-075 | Periodic import scan of download folders that skips already imported files | `internal/scheduler/tasks/importscan.go`, `import/pipeline.go` | working | kept | - | unplanned |
| P-076 | Completion checks wait for stable size and archive extraction and skip stalled downloads | `internal/import/completion.go` | working | kept | - | unplanned |
| P-077 | Hardlink placement with symlink and copy fallback, source kept for seeding | `internal/library/organizer/fileops.go (LinkOrCopy)` | working | kept | - | unplanned |
| P-078 | Upgrade on import replaces and deletes the older file | `internal/import/service.go (evaluateUpgradeEligibility)` | working | kept | - | unplanned |
| P-079 | Season pack import with per-file status, imported progressively | `internal/import/seasonpack.go` | working | kept | - | unplanned |
| P-080 | Multi-episode files map to several episodes | `internal/import/renamer/multiepisode.go` | working | kept | - | unplanned |
| P-081 | Import retry with backoff, failed status and health warning | `internal/import/retry.go` | working | kept | - | unplanned |
| P-082 | Import success writes history, notifies, marks requests available and pushes a live event | `internal/import/service.go (handleSuccessfulImport)` | working | kept | - | unplanned |
| P-083 | Pending imports list with per-file error and tap to retry | `GET /import/pending`, `POST /import/:id/retry`, `web/src/routes/import/pending-imports-group.tsx` | working | kept | - | unplanned |
| P-084 | Manual import page browses server folders and parses and matches each video file | `web/src/routes/import/index.tsx`, `POST /import/scan` | working | kept | - | unplanned |
| P-085 | One-tap import and Import All for matched files | `web/src/routes/import/use-import-actions.ts`, `POST /import/manual` | working | kept | - | unplanned |
| P-086 | Edit match dialog with parsed info and a movie or episode picker | `web/src/routes/import/edit-match-dialog.tsx`, `POST /import/manual/preview` | working | kept | - | unplanned |
| P-087 | Import settings for minimum file size and allowed video extensions | `internal/import/settings.go`, `extension-manager.tsx` | working | kept | - | unplanned |
| P-088 | Validation level setting that the pipeline never reads | `internal/import/settings.go`, `naming-validation-tab.tsx` | half-built | kept | - | unplanned |
| P-089 | Match conflict behavior setting that the pipeline never reads | `internal/import/settings.go`, `naming-matching-tab.tsx` | half-built | kept | - | unplanned |
| P-090 | Unknown media Auto Add setting that the pipeline never reads | `internal/import/settings.go`, `naming-matching-tab.tsx` | half-built | kept | - | unplanned |
| P-091 | Rename toggle per Content Type | `internal/import/module_naming.go`, `/settings/:moduleId/naming` | working | kept | - | unplanned |
| P-092 | Movie folder and file name templates | `internal/modules/movie/path_naming.go`, `naming-movie-tab.tsx` | working | kept | - | unplanned |
| P-093 | Series, season and specials folder templates and standard, daily and anime episode templates | `internal/modules/tv/path_naming.go`, `naming-tv-tab.tsx` | working | kept | - | unplanned |
| P-094 | Colon replacement modes including smart and custom | `internal/import/module_naming.go`, `naming-tv-tab.tsx` | working | kept | - | unplanned |
| P-095 | Multi-episode styles (extend, duplicate, repeat, scene, range, prefixed range) | `internal/modules/tv/path_naming.go` | working | kept | - | unplanned |
| P-096 | Naming token catalog with padding, truncation, case and language modifiers | `internal/import/renamer/tokens.go`, `modifiers.go` | working | kept | - | unplanned |
| P-097 | Token reference tab and click-to-insert token builder | `naming-token-reference-tab.tsx`, `token-builder-dialog.tsx` | working | kept | - | unplanned |
| P-098 | Live naming preview, pattern validation and filename parse tester | `POST /settings/import/naming/preview`, `filename-tester.tsx` | working | kept | - | unplanned |
| P-099 | Mass rename of library files with a preview, API only | `internal/import/rename.go`, `/import/rename/preview`, `/import/rename/execute` | working | kept | - | unplanned |

## Arr migration

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-100 | Auto-detect local Radarr and Sonarr databases | `internal/arrimport/detect.go`, `GET /arrimport/detect-db` | working | kept | - | unplanned |
| P-101 | Connect to Radarr or Sonarr by database path or URL and API key | `internal/arrimport/reader_sqlite.go`, `reader_api.go`, `POST /arrimport/connect` | working | kept | - | unplanned |
| P-102 | Map source root folders and quality profiles to SlipStream ones | `GET /arrimport/source/rootfolders`, `GET /arrimport/source/qualityprofiles` | working | kept | - | unplanned |
| P-103 | Preview and select items to migrate with Ready, Already in Library and Unknown filters | `POST /arrimport/preview`, `web/src/components/arr-import/` | working | kept | - | unplanned |
| P-104 | Execute library migration linking existing files, with live progress and counts | `internal/arrimport/executor.go`, `POST /arrimport/execute` | working | kept | - | unplanned |
| P-105 | Config import of clients, indexers, notifications, profiles and naming, API only | `GET /arrimport/config/preview`, `POST /arrimport/config/import` | half-built | kept | - | unplanned |
| P-106 | Onboarding wizard built around arr migration, in progress | `docs/Remaining-features.md`, `web/src/components/arr-import/` | half-built | kept | - | unplanned |

## Indexers

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-107 | Native indexers with priority, enable, auto search and RSS toggles | `internal/indexer/service.go`, `web/src/routes/settings/download-pipeline/indexers.tsx` | working | kept | - | unplanned |
| P-108 | Test a saved or unsaved indexer | `POST /indexers/:id/test`, `POST /indexers/test` | working | kept | - | unplanned |
| P-109 | Cardigann definition catalog with protocol and privacy filters and a dynamic settings form | `internal/indexer/cardigann/manager.go`, `definition-search-table.tsx`, `dynamic-settings-form.tsx` | working | kept | - | unplanned |
| P-110 | Update definitions from the remote definitions repo plus a custom definitions folder | `internal/indexer/cardigann/repository.go`, `/indexers/definitions` | working | kept | - | unplanned |
| P-111 | Tracker login flows with a persisted cookie store | `internal/indexer/cardigann/login.go`, `internal/indexer/cookie_store.go` | working | kept | - | unplanned |
| P-112 | Generic RSS indexer for RSS-only feeds | `internal/indexer/genericrss/` | working | kept | - | unplanned |
| P-113 | Content Type applicability derived from Newznab and Torznab categories | `internal/indexer/categories.go` | working | kept | - | unplanned |
| P-114 | Per-indexer status with failure backoff and health classes | `internal/indexer/status/service.go`, `GET /indexers/status` | working | kept | - | unplanned |
| P-115 | Per-indexer query and grab rate limits, config file only | `internal/indexer/ratelimit/limiter.go` | working | kept | - | unplanned |
| P-116 | Scheduled indexer health check | `internal/scheduler/tasks/indexerhealth.go` | working | kept | - | unplanned |
| P-117 | Indexer grab history, API only | `GET /history/indexer`, `GET /search/grab/history` | working | kept | - | unplanned |
| P-118 | Indexer mode switch between SlipStream indexers and Prowlarr | `internal/prowlarr/mode.go`, `web/src/components/indexers/indexer-mode-toggle.tsx` | working | superseded by the built-in indexer catalog | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-119 | Prowlarr connection settings with test, save and refresh | `internal/prowlarr/service.go`, `prowlarr-config-form.tsx` | working | superseded by the built-in indexer catalog | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-120 | Prowlarr indexer list with status, capabilities and last failure | `/indexers/prowlarr/indexers`, `prowlarr-indexer-list.tsx` | working | superseded by the built-in indexer catalog | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-121 | Per Prowlarr indexer priority, content type and stats reset | `/indexers/prowlarr/indexers/:id/settings`, `indexer-settings-dialog.tsx` | working | superseded by the built-in indexer catalog | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-122 | Scheduled Prowlarr health check with data refresh | `internal/scheduler/tasks/prowlarrhealth.go` | working | superseded by the built-in indexer catalog | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |

## Search and grab

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-123 | Manual release search sheet scoped to a movie, series, season or episode | `web/src/components/search/search-modal.tsx`, `GET /search/movie`, `GET /search/tv` | working | kept | - | unplanned |
| P-124 | Editable query bar with re-search | `web/src/components/search/search-input-bar.tsx` | working | kept | - | unplanned |
| P-125 | Sortable results with score, quality, indexer, size, age, peers and a freeleech badge | `web/src/components/search/search-results-table.tsx` | working | kept | - | unplanned |
| P-126 | Results footer with totals and per-indexer errors | `web/src/components/search/search-footer.tsx` | working | kept | - | unplanned |
| P-127 | Grab a release with an optional client choice, plus bulk grab | `internal/indexer/grab/service.go`, `POST /search/grab`, `POST /search/grab/bulk` | working | kept | - | unplanned |
| P-128 | Results deduplicated, filtered, quality enriched and scored | `internal/indexer/search/aggregator.go`, `router.go` | working | kept | - | unplanned |
| P-129 | Title matching rejects wrong-title releases | `internal/indexer/search/title_match.go` | working | kept | - | unplanned |
| P-130 | Release parser for resolution, source, codecs, HDR, audio, group, revision, edition, language and packs | `internal/library/scanner/parser.go`, `internal/module/parseutil/` | working | kept | - | unplanned |
| P-131 | Torrents fetched through the indexer for private trackers with URL fallback | `internal/indexer/grab/service.go (sendViaIndexer)` | working | kept | - | unplanned |

## Auto search and RSS

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-132 | Search now for one movie, episode, season or series | `POST /autosearch/movie/:id`, `/autosearch/season/:seriesId/:seasonNumber`, `/autosearch/series/:id` | working | kept | - | unplanned |
| P-133 | Auto search status per item | `GET /autosearch/status/:mediaType/:id` | working | kept | - | unplanned |
| P-134 | Retry a failed movie or episode | `POST /autosearch/retry/movie/:id` | working | kept | - | unplanned |
| P-135 | Search All missing or upgradable with result toasts | `POST /autosearch/missing/*`, `web/src/routes/missing/use-missing-page.ts` | working | kept | - | unplanned |
| P-136 | Scheduled automatic search with interval and backoff threshold settings | `internal/autosearch/settings.go`, `auto-search-section.tsx` | working | kept | - | unplanned |
| P-137 | Scheduled run refreshes metadata, orders by release date and groups season packs | `internal/autosearch/scheduled.go`, `internal/decisioning/collection.go` | working | kept | - | unplanned |
| P-138 | Search on add from the add flow | `internal/library/librarymanager/add.go` | working | kept | - | unplanned |
| P-139 | RSS sync with enable, interval, Run Now and last run status | `internal/rsssync/service.go`, `rss-sync-section.tsx` | working | kept | - | unplanned |
| P-140 | RSS sync matches new releases to wanted items and grabs the best | `internal/rsssync/matcher.go`, `cache.go` | working | kept | - | unplanned |
| P-141 | Per-feed RSS health warnings | `internal/rsssync/service.go (reportFeedHealth)` | working | kept | - | unplanned |

## Download clients and queue

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-142 | Download clients with host, port, SSL, URL base, credentials, category, priority and enable | `internal/downloader/service.go`, `web/src/routes/settings/download-pipeline/clients.tsx` | working | kept | - | unplanned |
| P-143 | Client credentials redacted in API responses | `internal/api/handlers_downloads.go` | working | kept | - | unplanned |
| P-144 | Test a saved or unsaved client | `POST /downloadclients/:id/test`, `POST /downloadclients/test` | working | kept | - | unplanned |
| P-145 | Scheduled download client health check | `internal/scheduler/tasks/downloadclienthealth.go` | working | kept | - | unplanned |
| P-146 | Downloads land in a SlipStream subfolder per Content Type | `internal/downloader/service.go (downloadSubDir)` | working | kept | - | unplanned |
| P-147 | Import delay per client | `internal/import/pipeline.go` | working | kept | - | unplanned |
| P-148 | Cleanup mode and seed ratio target stored but never acted on or shown | `internal/downloader/service.go` | half-built | kept | - | unplanned |
| P-149 | qBittorrent client | `internal/downloader/qbittorrent` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-150 | Transmission client | `internal/downloader/transmission` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-151 | Deluge client | `internal/downloader/deluge` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-152 | rTorrent client | `internal/downloader/rtorrent` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-153 | Vuze client | `internal/downloader/vuze` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-154 | Flood client | `internal/downloader/flood` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-155 | Aria2 client | `internal/downloader/aria2` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-156 | uTorrent client | `internal/downloader/utorrent` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-157 | Hadouken client | `internal/downloader/hadouken` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-158 | Synology Download Station client | `internal/downloader/downloadstation` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-159 | Freebox Download client | `internal/downloader/freeboxdownload` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-160 | rqbit client | `internal/downloader/rqbit` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-161 | Tribler client | `internal/downloader/tribler` | working | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-162 | SABnzbd client, not creatable and with no usenet grab path | `internal/downloader/sabnzbd`, `internal/indexer/grab/service.go (sendToClient)` | half-built | kept | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-163 | NZBGet client, type constant only | `internal/downloader/factory.go` | half-built | kept | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-164 | Unified queue across clients with progress, speed, ETA, parsed quality and target | `internal/downloader/queue.go`, `web/src/routes/downloads/index.tsx` | working | kept | - | unplanned |
| P-165 | Queue filter by Content Type | `web/src/routes/downloads/use-activity-page.ts` | working | kept | - | unplanned |
| P-166 | Pause, resume, remove and remove with files | `POST /queue/:id/pause`, `DELETE /queue/:id`, `web/src/routes/downloads/use-queue-row.ts` | working | kept | - | unplanned |
| P-167 | Live queue push with adaptive polling | `internal/downloader/broadcaster.go` | working | kept | - | unplanned |
| P-168 | Unreachable client banner showing last known data | `web/src/routes/downloads/index.tsx` | working | kept | - | unplanned |
| P-169 | Completion detection with import attempt backoff | `internal/downloader/completion.go` | working | kept | - | unplanned |
| P-170 | Downloads removed from the client flip the item to failed | `internal/downloader/completion.go (CheckForDisappearedDownloads)` | working | kept | - | unplanned |

## Notifications

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-171 | Notification connections with per-event toggles, enable and test | `internal/notification/service.go`, `web/src/routes/settings/general/notifications.tsx` | working | kept | - | unplanned |
| P-172 | Schema-driven provider settings form with advanced fields | `internal/notification/schema.go`, `GET /notifications/schema` | working | kept | - | unplanned |
| P-173 | Event catalog built from framework and Content Type events | `GET /notifications/events`, `internal/module/framework_events.go` | working | kept | - | unplanned |
| P-174 | Failing connections back off and recover on success | `internal/notification/service.go (recordFailure)` | working | kept | - | unplanned |
| P-175 | On Grab event | `internal/api/notification_bridge.go` | working | kept | - | unplanned |
| P-176 | On Import event | `internal/import/service.go` | working | kept | - | unplanned |
| P-177 | On Upgrade event | `internal/import/service.go` | working | kept | - | unplanned |
| P-178 | Movie added and movie deleted events | `internal/library/movies/service.go` | working | kept | - | unplanned |
| P-179 | Series added and series deleted events | `internal/library/tv/service.go` | working | kept | - | unplanned |
| P-180 | Health Issue and Health Restored events with opt-in for warnings | `internal/health/service.go` | working | kept | - | unplanned |
| P-181 | New portal request announced to admin connections | `internal/portal/notifications/service.go (DispatchGenericMessage)` | working | kept | - | unplanned |
| P-182 | On App Update event declared but never dispatched | `internal/notification/types/types.go`, `internal/update` | half-built | kept | - | unplanned |
| P-183 | Discord provider with rich embeds and posters | `internal/notification/discord` | working | kept | - | unplanned |
| P-184 | Telegram provider with topic, silent and links | `internal/notification/telegram` | working | kept | - | unplanned |
| P-185 | Webhook provider with method, basic auth and custom headers | `internal/notification/webhook` | working | kept | - | unplanned |
| P-186 | Email provider over SMTP | `internal/notification/email` | working | kept | - | unplanned |
| P-187 | Slack provider | `internal/notification/slack` | working | kept | - | unplanned |
| P-188 | Pushover provider with devices, priority and sound | `internal/notification/pushover` | working | kept | - | unplanned |
| P-189 | Plex library refresh on import with OAuth sign-in, server and section pick, path mappings and partial refresh | `internal/notification/plex/`, `internal/scheduler/tasks/plexrefresh.go` | working | superseded by the Media Server contract | [Media Server contract](https://github.com/jatassi/SlipStream/issues/22) | unplanned |
| P-190 | Gotify provider, selectable but unsupported | `internal/notification/schema.go`, `internal/notification/factory.go` | half-built | kept | - | unplanned |
| P-191 | ntfy provider, selectable but unsupported | `internal/notification/schema.go`, `internal/notification/factory.go` | half-built | kept | - | unplanned |
| P-192 | Pushbullet provider, selectable but unsupported | `internal/notification/schema.go`, `internal/notification/factory.go` | half-built | kept | - | unplanned |
| P-193 | Join provider, selectable but unsupported | `internal/notification/schema.go`, `internal/notification/factory.go` | half-built | kept | - | unplanned |
| P-194 | Apprise provider, selectable but unsupported | `internal/notification/schema.go`, `internal/notification/factory.go` | half-built | kept | - | unplanned |
| P-195 | Custom Script provider, selectable but unsupported | `internal/notification/schema.go`, `internal/notification/factory.go` | half-built | kept | - | unplanned |
| P-196 | Prowl provider, type constant only | `internal/notification/types/types.go` | half-built | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-197 | Simplepush provider, type constant only | `internal/notification/types/types.go` | half-built | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |
| P-198 | Signal provider, type constant only | `internal/notification/types/types.go` | half-built | open | [Which v1 integrations stay beside the built-ins](https://github.com/jatassi/SlipStream/issues/4) | unplanned |

## Multiple versions

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-199 | Multi-version master toggle gated behind a dry run | `/slots/settings`, `web/src/components/settings/sections/master-toggle-card.tsx` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-200 | Up to three slots with name, enable, quality profile, order and root folder override | `internal/library/slots/service.go`, `/slots`, `web/src/routes/settings/media/version-slots.tsx` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-201 | Slot profiles must be mutually exclusive, with a resolve conflicts modal | `internal/library/quality/exclusivity.go`, `POST /slots/validate`, `resolve-config-modal.tsx` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-202 | Naming templates must differ per slot, with a resolve naming modal | `internal/library/slots/naming.go`, `POST /slots/validate-naming`, `resolve-naming-modal.tsx` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-203 | Evaluate which slot a release lands in, shown as a slot column in search results | `POST /slots/evaluate`, `release-slot-cell.tsx` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-204 | Disable a slot with a delete, keep or cancel action after an impact check | `GET /slots/:id/disable-check`, `POST /slots/:id/disable-with-action` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-205 | Change a slot profile with a keep, reevaluate or cancel action after a check | `GET /slots/:id/profile-change-check`, `POST /slots/:id/profile-with-action` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-206 | Migration dry run preview of existing files per slot, then execute | `internal/library/slots/migration.go`, `web/src/components/slots/DryRunModal/` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-207 | Review queue for files that could not be auto assigned | `/slots/review-queue` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-208 | Per-movie and per-episode slot rows with status, monitor, search and file assignment | `/slots/movies/:movieId`, `/slots/episodes/:episodeId`, `web/src/components/series/episode-slot-row.tsx` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-209 | Per-slot status rolls up to the item status | `internal/library/slots/status.go` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-210 | Per-slot search now and multi-slot automatic search grabbing several releases | `/autosearch/movie/:id/slot/:slotId`, `internal/autosearch/slots.go` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-211 | Grab and manual import into a chosen target slot, shown in the queue | `POST /search/grab`, `POST /import/manual`, `internal/downloader/queue.go` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-212 | Slot assigned, reassigned and unassigned history events | `internal/history/types.go` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |

## Health

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-213 | Health page grouped by category with status, message and since time | `internal/health/service.go`, `web/src/routes/system/health.tsx` | working | kept | - | unplanned |
| P-214 | Test one health item or a whole category | `POST /system/health/:category/test`, `POST /system/health/:category/:id/test` | working | kept | - | unplanned |
| P-215 | Scheduled storage check with free space warning and error thresholds | `internal/scheduler/tasks/storagehealth.go` | working | kept | - | unplanned |
| P-216 | Health check intervals and thresholds, config file only | `internal/config/config.go` | working | kept | - | unplanned |
| P-217 | Health warnings on settings hub rows and the dashboard | `web/src/routes/settings/use-health-warnings.ts`, `web/src/components/dashboard/health-group.tsx` | working | kept | - | unplanned |
| P-218 | Port and firewall status check across Windows, macOS and Linux firewalls | `internal/firewall/`, `GET /system/firewall`, `firewall-status-panel.tsx` | working | kept | [Relay Role and reachability diagnostic](https://github.com/jatassi/SlipStream/issues/16) | unplanned |

## Calendar, history and activity

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-219 | Calendar month view with Today and per-day events | `web/src/components/calendar/calendar-month-view.tsx`, `GET /calendar` | working | kept | - | unplanned |
| P-220 | Calendar list view of the next 30 days | `web/src/components/calendar/calendar-list-view.tsx` | working | kept | - | unplanned |
| P-221 | Calendar events for digital and physical releases and air dates, collapsed per season | `internal/calendar/service.go`, `event-presentation.ts` | working | kept | - | unplanned |
| P-222 | Theatrical release events emitted but not shown | `internal/calendar/service.go`, `web/src/types/calendar.ts` | half-built | kept | - | unplanned |
| P-223 | History list grouped by day with load more | `internal/history/service.go`, `web/src/routes/history/history.tsx` | working | kept | - | unplanned |
| P-224 | History filters by Content Type, event type and date preset | `web/src/routes/history/history-filters.tsx`, `GET /history` | working | kept | - | unplanned |
| P-225 | Grabbed, deleted and failed history events not offered as filters | `web/src/types/history.ts` | half-built | kept | - | unplanned |
| P-226 | Clear all history | `DELETE /history` | working | kept | - | unplanned |
| P-227 | History retention setting with nightly cleanup | `internal/history/settings.go`, `use-history-retention.ts` | working | kept | - | unplanned |

## Scheduled tasks

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-228 | Scheduled tasks page with last run, next run, running state and Run Now | `internal/scheduler/scheduler.go`, `web/src/routes/system/tasks.tsx` | working | kept | - | unplanned |
| P-229 | Task schedules fixed apart from auto search and RSS intervals | `internal/scheduler/tasks/` | working | kept | - | unplanned |
| P-230 | Daily library scan of all root folders | `internal/scheduler/tasks/ (library-scan)` | working | kept | - | unplanned |
| P-231 | Daily refresh of unreleased metadata | `internal/scheduler/tasks/ (unreleased-refresh)` | working | kept | - | unplanned |

## Auth and API

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-232 | Single fixed Administrator account created at first run with a 4-digit PIN from localhost | `POST /auth/setup`, `web/src/routes/auth/setup.tsx` | working | superseded by one identity with an admin permission | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-233 | Shared login page for admin and portal users with remembered username | `web/src/routes/requests/auth/login.tsx`, `internal/auth/handlers.go` | working | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-234 | Sessions as JWTs, 24 hours for admins and 30 days for portal users | `internal/auth/service.go` | working | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-235 | Change PIN and PIN re-confirmation for sensitive actions | `/requests/auth/profile`, `/requests/auth/verify-pin`, `change-pin-dialog.tsx` | working | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-236 | Admin passkeys with register, sign in, rename and delete | `internal/auth/passkey.go`, `web/src/components/portal/passkey-manager.tsx` | working | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-237 | Passkey relying party settings for RP ID, allowed origins and display name | `internal/api/webauthn_settings.go`, `webauthn-rp-config.tsx` | working | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-238 | Auth rate limit with progressive lockout | `internal/api/ratelimit/auth_limiter.go` | working | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-239 | Security headers, body size limit and same-origin CORS | `internal/api/middleware/security.go` | working | kept | - | unplanned |
| P-240 | API key generated and regenerable but accepted nowhere | `POST /settings/apikey`, `internal/api/handlers_system.go` | half-built | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |

## Live updates

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-241 | Admin WebSocket with reconnect that drives queue, progress, tasks, health and library refresh | `internal/websocket/hub.go`, `web/src/stores/websocket.ts` | working | kept | - | unplanned |
| P-242 | Library entity added, updated and deleted events | `internal/websocket/hub.go` | working | kept | - | unplanned |
| P-243 | Live events for search, grab, auto search, RSS, import, rename, history, health, indexers, artwork, tasks, updates, logs and requests | `internal/websocket/hub.go`, `web/src/stores/ws-message-handlers.ts` | working | kept | - | unplanned |

## System and logs

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-244 | Server port setting with restart notice | `internal/api/handlers_system.go`, `server-section.tsx` | working | kept | - | unplanned |
| P-245 | External access toggle between localhost and LAN with a warning | `server-section.tsx`, `GET/PUT /settings` | working | kept | [Relay Role and reachability diagnostic](https://github.com/jatassi/SlipStream/issues/16) | unplanned |
| P-246 | Port fallback across 10 ports with the actual port shown | `internal/config (FindAvailablePort)`, `server-section.tsx` | working | kept | - | unplanned |
| P-247 | Log level setting and copy log path | `internal/logger/`, `server-section.tsx` | working | kept | - | unplanned |
| P-248 | Log rotation settings for size, backups, age and compression | `internal/logger/`, `server-section.tsx` | working | kept | - | unplanned |
| P-249 | Live log viewer with level filter, search, pause and clear | `internal/api/logs.go`, `web/src/routes/system/logs.tsx` | working | kept | - | unplanned |
| P-250 | Download the current log file | `GET /system/logs/download` | working | kept | - | unplanned |
| P-251 | Status endpoint with version, counts and feature flags | `GET /status` | working | kept | - | unplanned |
| P-252 | Enable or disable a Content Type, API only | `GET/PUT /settings/modules` | working | kept | [The Content Type contract](https://github.com/jatassi/SlipStream/issues/5) | unplanned |
| P-253 | Network services retry with backoff when offline at boot | `internal/startup/retry.go` | working | kept | - | unplanned |
| P-254 | Config from environment, .env and config.yaml with defaults | `internal/config/config.go`, `configs/config.example.yaml` | working | kept | - | unplanned |

## Developer mode

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-255 | Runtime developer mode toggle on dev builds that swaps to a separate dev database | `internal/api/devmode.go`, `header-dev-mode.tsx` | working | open | [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26) | unplanned |
| P-256 | Mock metadata, indexer, download client and notifier in developer mode | `internal/metadata/mock`, `internal/indexer/mock`, `internal/downloader/mock`, `internal/notification/mock` | working | open | [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26) | unplanned |
| P-257 | Fast forward a mock download and add a mock download for testing | `POST /queue/:id/fastforward`, `download-client-dialog.tsx` | working | open | [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26) | unplanned |
| P-258 | Virtual filesystem with sample multi-version and partial TV content | `internal/filesystem/mock/virtualfs.go` | working | open | [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26) | unplanned |
| P-259 | Sample library, test slots and profiles seeded in developer mode | `internal/api/devmode.go (populateMockMedia, setupDevModeSlots)` | working | open | [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26) | unplanned |
| P-260 | Prowlarr mode overridden in developer mode | `internal/prowlarr/mode.go` | working | open | [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26) | unplanned |
| P-261 | Force Loading switch that shows every loading skeleton | `web/src/components/layout/dev-mode-controls.tsx` | working | open | [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26) | unplanned |
| P-262 | Release parse and profile match debug tools | `internal/library/slots/debug.go`, `slot-debug-panel.tsx` | working | open | [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26) | unplanned |
| P-263 | Update state debug switcher | `web/src/routes/system/use-update-page.ts (DEBUG_STATE_CONFIG)` | working | open | [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26) | unplanned |
| P-264 | Controls showcase page | `web/src/routes/dev/` | working | open | [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26) | unplanned |
| P-265 | Mock fixture tooling that pulls real TMDB, TVDB and indexer data | `cmd/fetchmockdata`, `cmd/convertmockdata` | working | open | [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26) | unplanned |

## Updates

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-266 | Update page with check, install, cancel, progress, release notes and failure retry | `internal/update/service.go`, `web/src/routes/system/update.tsx` | working | kept | - | unplanned |
| P-267 | Update check daily and on start against GitHub Releases with per-platform assets | `internal/scheduler/tasks/update.go`, `internal/update/service.go` | working | kept | - | unplanned |
| P-268 | Self-replace install that relaunches the app | `internal/update/`, `cmd/slipstream/main.go (--complete-update)` | working | kept | - | unplanned |
| P-269 | Auto-install setting with no UI | `GET/PUT /update/settings` | half-built | kept | - | unplanned |
| P-270 | Pre-update database backup that only logs | `internal/update/service.go (backupDatabase)` | half-built | kept | - | unplanned |
| P-271 | Linux stub launcher so packaged installs update without sudo | `cmd/stub/main.go` | working | kept | - | unplanned |

## Packaging

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-272 | Windows tray icon with Open and Quit | `internal/platform/app_windows.go` | working | kept | - | unplanned |
| P-273 | Windows MSI installer with Start Menu, Desktop and Startup shortcuts | `scripts/windows/installer.wxs` | working | kept | - | unplanned |
| P-274 | Windows portable ZIP | `.github/workflows/release.yml` | working | kept | - | unplanned |
| P-275 | macOS menu bar item with start at login | `internal/platform/app_darwin.go`, `scripts/macos/com.slipstream.slipstream.plist` | working | kept | - | unplanned |
| P-276 | macOS DMG for arm64 and amd64 | `scripts/macos/create-dmg.sh` | working | kept | - | unplanned |
| P-277 | Linux deb and rpm packages | `scripts/linux/postinstall.sh`, `.github/workflows/release.yml` | working | kept | - | unplanned |
| P-278 | Linux AppImage | `scripts/linux/create-appimage.sh` | working | kept | - | unplanned |
| P-279 | Linux portable tarballs for amd64 and arm64 | `.github/workflows/release.yml` | working | kept | - | unplanned |
| P-280 | Linux systemd user unit shipped as a doc, startup helpers never called | `scripts/linux/slipstream.service`, `internal/platform/app_linux.go` | half-built | kept | - | unplanned |
| P-281 | Console mode flag on any OS | `cmd/slipstream/main.go (--no-tray)` | working | kept | - | unplanned |
| P-282 | First run opens the browser | `internal/platform/util.go`, `cmd/slipstream/main.go` | working | kept | - | unplanned |
| P-283 | OS-standard data and log paths | `internal/config/config.go` | working | kept | - | unplanned |
| P-284 | Release CI uploads every package to GitHub Releases | `.github/workflows/release.yml` | working | kept | - | unplanned |

## Module system

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-285 | Media kinds as self-contained modules behind one registry | `internal/module/interfaces.go`, `internal/module/registry` | half-built | superseded by the Content Type contract | [The Content Type contract](https://github.com/jatassi/SlipStream/issues/5) | unplanned |
| P-286 | Movies as a Content Type | `internal/modules/movie` | working | kept | [The Content Type contract](https://github.com/jatassi/SlipStream/issues/5) | unplanned |
| P-287 | TV as a Content Type with series, seasons and episodes | `internal/modules/tv` | working | kept | [The Content Type contract](https://github.com/jatassi/SlipStream/issues/5) | unplanned |
| P-288 | Content Type declared events, naming contexts, calendar items, presets and portal provisioning | `internal/modules/movie/module.go`, `internal/modules/tv/module.go` | working | superseded by the Content Type contract | [The Content Type contract](https://github.com/jatassi/SlipStream/issues/5) | unplanned |
| P-289 | New module scaffolder | `scripts/new-module`, `make new-module` | working | superseded by the Content Type contract | [The Content Type contract](https://github.com/jatassi/SlipStream/issues/5) | unplanned |
| P-290 | Module validation command | `slipstream validate-module` | working | superseded by the Content Type contract | [The Content Type contract](https://github.com/jatassi/SlipStream/issues/5) | unplanned |
| P-291 | Music Content Type promised by the module docs | `internal/module/interfaces.go`, `docs/module-system-spec.md` | not started | out of scope | - | unplanned |

## Requests portal admin

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-292 | Enable or disable the requests portal, with a disabled banner on the queue | `internal/portal/admin/settings_handlers.go`, `web/src/routes/requests-admin/settings.tsx` | working | kept | - | unplanned |
| P-293 | Default weekly request quotas per Content Type | `internal/portal/quota/service.go`, `web/src/routes/requests-admin/settings.tsx` | working | kept | - | unplanned |
| P-294 | Default root folder for requested content | `web/src/routes/requests-admin/settings.tsx` | working | kept | - | unplanned |
| P-295 | Notify on new requests with an admin channel list | `internal/portal/notifications/service.go`, `web/src/routes/requests-admin/settings.tsx` | working | kept | - | unplanned |
| P-296 | Portal search rate limit per user | `internal/portal/ratelimit/search_limiter.go` | working | kept | - | unplanned |
| P-297 | Request queue with Pending, Approved, Downloading, Available and Denied segments | `web/src/routes/requests-admin/index.tsx` | working | kept | - | unplanned |
| P-298 | Request row with poster, type, seasons, requester and age | `web/src/routes/requests-admin/request-row.tsx` | working | kept | - | unplanned |
| P-299 | Approve, deny with a reason shown to the requester, or delete a request | `internal/portal/admin/requests_handlers.go`, `deny-sheet.tsx` | working | kept | - | unplanned |
| P-300 | Approve and Manual Search stepping through each requested season | `web/src/routes/requests-admin/request-search-modal.tsx` | working | kept | - | unplanned |
| P-301 | Approve and Auto Search with result toasts | `web/src/routes/requests-admin/use-request-approve.ts` | working | kept | - | unplanned |
| P-302 | Batch approve, deny and delete, API only | `POST /admin/requests/batch/approve` | half-built | kept | - | unplanned |
| P-303 | Portal users list with quota usage, enable, disable, edit and delete | `internal/portal/admin/users_handlers.go`, `web/src/routes/requests-admin/users.tsx` | working | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-304 | Edit a portal user's quality profile per Content Type and auto-approve | `web/src/routes/requests-admin/user-edit-sheet.tsx` | working | kept | - | unplanned |
| P-305 | Per-user quota override, API only | `/admin/requests/users/:id/quota` | half-built | kept | - | unplanned |
| P-306 | Invite a user with profile and auto-approve, creating a link | `internal/portal/invitations/service.go`, `web/src/routes/requests-admin/invite-sheet.tsx` | working | kept | [Portal onboarding flow and emails](https://github.com/jatassi/SlipStream/issues/24) | unplanned |
| P-307 | Invitations list with expiry, copy link, resend and delete | `internal/portal/admin/invitations_handlers.go`, `web/src/routes/requests-admin/users.tsx` | working | kept | [Portal onboarding flow and emails](https://github.com/jatassi/SlipStream/issues/24) | unplanned |
| P-308 | Auto-approve when the user flag or the profile allows it | `internal/portal/autoapprove/service.go` | working | kept | - | unplanned |
| P-309 | Request status tracked from library events, including per-season completion | `internal/portal/requests/status_tracker.go` | working | kept | - | unplanned |

## Requests portal user side

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-310 | Invite signup page that validates the token and sets a PIN | `web/src/routes/requests/auth/signup.tsx`, `internal/auth/handlers.go` | working | kept | [Portal onboarding flow and emails](https://github.com/jatassi/SlipStream/issues/24) | unplanned |
| P-311 | Self-service invitation resend endpoint with no UI | `POST /requests/auth/resend` | half-built | kept | [Portal onboarding flow and emails](https://github.com/jatassi/SlipStream/issues/24) | unplanned |
| P-312 | Portal disabled screen on login, signup and inside the app | `web/src/components/portal/portal-auth-guard.tsx` | working | kept | - | unplanned |
| P-313 | Sign in with passkey with PIN fallback | `web/src/components/portal/`, `internal/auth/passkey_handlers.go` | working | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-314 | Prompt to set up a passkey after login with don't show again | `web/src/components/portal/passkey-prompt-modal.tsx` | working | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-315 | Deep link that opens passkey registration | `web/src/components/portal/passkey-deep-link.ts` | working | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-316 | Portal settings with Security tab, Notifications tab and logout | `web/src/routes/requests/settings.tsx` | working | kept | - | unplanned |
| P-317 | Admin sessions can use the portal pages | `internal/portal/requests/handlers.go` | working | kept | [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23) | unplanned |
| P-318 | Portal header with search, library link, bell and settings | `web/src/components/portal/portal-header.tsx` | working | kept | [UI redesign direction](https://github.com/jatassi/SlipStream/issues/25) | unplanned |
| P-319 | In-app inbox with unread badge and mark read | `internal/portal/notifications/inbox_handlers.go`, `web/src/components/portal/notification-bell.tsx` | working | kept | - | unplanned |
| P-320 | Active downloads strip for the user's requests, matched client side by title | `web/src/components/portal/portal-downloads.tsx` | half-built | kept | - | unplanned |
| P-321 | Portal live updates by polling | `web/src/hooks/portal/use-requests.ts` | working | kept | - | unplanned |
| P-322 | My requests list with Mine and All tabs | `web/src/routes/requests/index.tsx` | working | kept | - | unplanned |
| P-323 | Request card with status badge and inline download progress | `web/src/lib/request-status-config.tsx`, `request-download-progress.tsx` | working | kept | - | unplanned |
| P-324 | Request list status tabs and sorting from the spec, not shipped | `docs/external-requests-spec.md`, `web/src/routes/requests/index.tsx` | half-built | kept | - | unplanned |
| P-325 | TMDB search split into In Library and Request sections | `internal/portal/search/handlers.go`, `web/src/routes/requests/search.tsx` | working | kept | - | unplanned |
| P-326 | Search card states for requested, watching, available, partial seasons and progress | `web/src/components/search/external-media-card.tsx`, `season-availability-badge.tsx` | working | kept | - | unplanned |
| P-327 | Movie in the library without files stays requestable | `internal/portal/requests/library_check.go` | working | kept | - | unplanned |
| P-328 | Request allowed when the user's profile targets an empty slot | `internal/portal/requests/library_check.go` | working | superseded by Renditions | [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6) | unplanned |
| P-329 | More info modal with credits, ratings, awards, cast and seasons | `web/src/components/search/media-info-modal.tsx` | working | kept | - | unplanned |
| P-330 | One-tap movie request, with Watch offered for a duplicate | `web/src/routes/requests/use-request-search.ts` | working | kept | - | unplanned |
| P-331 | Series request dialog with per-season picks, specials and monitor future episodes | `web/src/routes/requests/series-request-dialog.tsx` | working | kept | - | unplanned |
| P-332 | Episode requests in the data model with no UI | `web/src/types/portal.ts` | half-built | kept | - | unplanned |
| P-333 | Requests use the user's assigned profile and the admin default root folder | `internal/portal/requests/service.go` | working | kept | - | unplanned |
| P-334 | Auto-approved requests consume quota, add to the library and search, or stay pending over quota | `internal/portal/autoapprove/service.go`, `internal/portal/requests/searcher.go` | working | kept | - | unplanned |
| P-335 | Weekly quotas per Content Type with lazy reset | `internal/portal/quota/service.go` | working | kept | - | unplanned |
| P-336 | Request detail with dates, requester, denial reason, status and progress | `web/src/routes/requests/$id.tsx` | working | kept | - | unplanned |
| P-337 | Watch or unwatch a request, and cancel your own pending request | `internal/portal/requests/watchers.go`, `request-detail-header.tsx` | working | kept | - | unplanned |
| P-338 | Portal library of available movies and series with requests for missing seasons | `internal/portal/library/handlers.go`, `web/src/routes/requests/library.tsx` | working | kept | - | unplanned |
| P-339 | User-owned notification channels with approved, denied and available toggles | `internal/portal/notifications/service.go`, `notification-channels-card.tsx` | working | kept | - | unplanned |
| P-340 | Requester and watchers notified through channels and the inbox | `internal/portal/notifications/service.go` | working | kept | - | unplanned |
| P-341 | iOS portal companion app | `SlipStream-iOS/Packages` | half-built | out of scope | - | unplanned |

## API surface

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-342 | REST API under /api/v1 with bearer JWT | `internal/api/routes.go` | working | kept | - | unplanned |
| P-343 | Public liveness and status endpoints | `GET /health`, `GET /api/v1/status` | working | kept | - | unplanned |
| P-344 | Portal REST contract for auth, requests, search, library, notifications and inbox | `internal/portal/requests/handlers.go`, `internal/auth/handlers.go` | working | kept | - | unplanned |
| P-345 | Admin request management REST for requests, users, invitations and settings | `internal/portal/admin/` | working | kept | - | unplanned |
| P-346 | Admin WebSocket at /ws with the token in the subprotocol | `internal/websocket/hub.go` | working | kept | - | unplanned |

## Not-started items

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-347 | Config backup and restore | `docs/Remaining-features.md` | not started | kept | - | unplanned |
| P-348 | Dashboard rework | `docs/Remaining-features.md` | not started | superseded by the UI redesign | [UI redesign direction](https://github.com/jatassi/SlipStream/issues/25) | unplanned |
| P-349 | Transcode during import tied to slots | `docs/Remaining-features.md` | not started | superseded by the A/V pipeline | [A/V pipeline design](https://github.com/jatassi/SlipStream/issues/18) | unplanned |
| P-350 | TRaSH Guides import | `docs/Remaining-features.md` | not started | kept | [Custom formats engine and TRaSH Guides import](https://github.com/jatassi/SlipStream/issues/7) | unplanned |
| P-351 | Credential encryption at rest | `docs/Production-Readiness-Checklist.md` | not started | kept | - | unplanned |
| P-352 | HTTPS setting for the web app | `docs/Remaining-features.md`, `server-section.tsx` | not started | superseded by the Relay Role TLS forwarding | [Relay Role and reachability diagnostic](https://github.com/jatassi/SlipStream/issues/16) | unplanned |
| P-353 | Static binary | `docs/Production-Readiness-Checklist.md` | not started | open | [Choose the stack](https://github.com/jatassi/SlipStream/issues/11) | unplanned |
| P-354 | Code signing | `docs/Production-Readiness-Checklist.md` | not started | kept | - | unplanned |
| P-355 | Torznab and Newznab endpoint served by SlipStream | - | not started | out of scope | - | unplanned |
| P-356 | Prowlarr Applications sync into SlipStream | `docs/Prowlarr-other-arr-integration.md` | not started | out of scope | - | unplanned |

## Added in v2

| ID | Feature | v1 path | v1 state | v2 disposition | Covered by | Status |
|---|---|---|---|---|---|---|
| P-357 | Release blocklist with remove and blocklist from the queue | - | not started | kept | - | unplanned |
| P-358 | Proper and repack handling that triggers upgrades | - | not started | kept | - | unplanned |
| P-359 | Custom formats engine | - | not started | kept | [Custom formats engine and TRaSH Guides import](https://github.com/jatassi/SlipStream/issues/7) | unplanned |
| P-360 | Docker images | - | not started | kept | - | unplanned |
