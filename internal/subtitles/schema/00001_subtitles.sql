-- +goose Up
-- A NULL setting is its default.
CREATE TABLE subtitles_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    search_on INTEGER CHECK (search_on IN (0, 1)),
    search_upgrades INTEGER CHECK (search_upgrades IN (0, 1)),
    longest_gap_seconds INTEGER CHECK (longest_gap_seconds > 0),
    full_pace_seconds INTEGER CHECK (full_pace_seconds >= 0),
    report_gap_seconds INTEGER CHECK (report_gap_seconds > 0),
    tries_per_search INTEGER CHECK (tries_per_search BETWEEN 1 AND 10),
    sync_match INTEGER CHECK (sync_match BETWEEN 10 AND 95),
    sync_max_offset_seconds INTEGER CHECK (sync_max_offset_seconds > 0),
    sync_framerates TEXT CHECK (sync_framerates IN ('none', 'standard', 'range')),
    provider_warn_after_seconds INTEGER CHECK (provider_warn_after_seconds > 0)
) STRICT;
INSERT INTO subtitles_settings (id) VALUES (1);

-- One row per Storage Node with a map_slots override or a fault. A build
-- that succeeds on the Node clears the fault, and the Health item with it.
CREATE TABLE subtitles_node (
    node_id BLOB PRIMARY KEY NOT NULL REFERENCES nodes_node (id) ON DELETE CASCADE,
    map_slots INTEGER CHECK (map_slots BETWEEN 1 AND 8),
    fault_at INTEGER,
    fault_words TEXT,                      -- ffmpeg's first line, under the item's Details
    CHECK ((fault_at IS NULL) = (fault_words IS NULL))
) STRICT;

-- One row per Spec that subtitleProviders() lists; Open inserts missing
-- ones at the end of the order.
CREATE TABLE subtitles_provider (
    id TEXT PRIMARY KEY NOT NULL CHECK (id <> ''),
    position INTEGER NOT NULL UNIQUE CHECK (position >= 1),
    is_on INTEGER NOT NULL CHECK (is_on IN (0, 1)),
    api_key_sealed BLOB,
    account TEXT,
    plan TEXT,
    allowance TEXT CHECK (allowance IS NULL OR json_valid(allowance)), -- calls.Allowance, the last OK answer's
    allowance_at INTEGER,
    paused_until INTEGER,
    pause_reason TEXT CHECK (pause_reason IN ('quota', 'rate', 'key', 'down')),
    pause_words TEXT,
    failing_since INTEGER,
    last_error TEXT,
    key_checked_at INTEGER,
    -- When the provider last gained the power to search (turned on, key
    -- accepted). A missing want last searched before a covering provider's
    -- gained_at is stamped due by the next bulk pass; no bulk update runs.
    gained_at INTEGER,
    CHECK ((paused_until IS NULL) = (pause_reason IS NULL))
) STRICT;

-- One row per live library file. The bulk pass and the Changed hook insert
-- it; files' cascade deletes it. The fact columns are copied from files by
-- records.putFacts, their one writer, from the same pure function wantsOf
-- uses.
CREATE TABLE subtitles_file (
    file_id BLOB PRIMARY KEY NOT NULL REFERENCES files_file (id) ON DELETE CASCADE,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    root_id BLOB NOT NULL REFERENCES roots_root (id),
    kind TEXT CHECK (kind IN ('feature', 'episode')),   -- the facet's; NULL: the type has no subtitles
    wantless INTEGER NOT NULL DEFAULT 0 CHECK (wantless IN (0, 1)), -- Kept or gone
    audio TEXT NOT NULL DEFAULT ',' CHECK (audio LIKE ',%,' OR audio = ','),       -- ",en,fr,": base codes of tagged audio tracks
    embedded TEXT NOT NULL DEFAULT ',' CHECK (embedded LIKE ',%,' OR embedded = ','), -- ",en,en-forced,": want keys embedded text tracks meet, ignores applied
    map_state TEXT NOT NULL DEFAULT 'idle' CHECK (map_state IN ('idle', 'due', 'building', 'built', 'failed')),
    map_fault TEXT CHECK (map_fault IN ('no_audio', 'decode_failed', 'node_fault')),
    map_words TEXT,
    map_node_id BLOB REFERENCES nodes_node (id) ON DELETE SET NULL,
    map_stream INTEGER,
    map_identity BLOB,
    map_version INTEGER,
    map_frames INTEGER,
    map_density REAL,
    map_tries INTEGER NOT NULL DEFAULT 0,
    map_next_at INTEGER,
    map_built_at INTEGER,
    map_rank INTEGER NOT NULL DEFAULT 3 CHECK (map_rank BETWEEN 0 AND 3), -- 0 report, 1 admin, 2 fresh import, 3 other
    map_since INTEGER NOT NULL,
    fresh INTEGER NOT NULL DEFAULT 0 CHECK (fresh IN (0, 1)),
    survey_due INTEGER NOT NULL DEFAULT 1 CHECK (survey_due IN (0, 1)),
    survey_from TEXT,
    surveyed_at INTEGER,
    companions TEXT,                       -- JSON []Extra: the last survey's companions that meet no want
    CHECK ((map_state = 'failed') = (map_fault IS NOT NULL)),
    CHECK ((map_state = 'built') = (map_built_at IS NOT NULL))
) STRICT;
CREATE INDEX subtitles_file_map_due ON subtitles_file (root_id, map_rank, map_since DESC) WHERE map_state = 'due';
CREATE INDEX subtitles_file_map_retry ON subtitles_file (map_next_at) WHERE map_state = 'failed' AND map_next_at IS NOT NULL;
CREATE INDEX subtitles_file_survey_due ON subtitles_file (file_id) WHERE survey_due = 1;
CREATE INDEX subtitles_file_work ON subtitles_file (work_id);
CREATE INDEX subtitles_file_node ON subtitles_file (map_node_id, file_id) WHERE map_state = 'built';

-- A searchable want's search state: a want with no Sidecar (unmet) or met
-- by one. Wants met by an embedded track or Not needed have no row.
CREATE TABLE subtitles_want (
    file_id BLOB NOT NULL REFERENCES subtitles_file (file_id) ON DELETE CASCADE,
    language TEXT NOT NULL CHECK (language <> ''),
    forced INTEGER NOT NULL CHECK (forced IN (0, 1)),
    work_id BLOB NOT NULL,                 -- subtitles_file's, for the Library grouping
    kind TEXT NOT NULL CHECK (kind IN ('feature', 'episode')), -- subtitles_file's, for the queue's ready-kind selection
    unmet INTEGER NOT NULL CHECK (unmet IN (0, 1)), -- no subtitles_sidecar row of this key; records.go is its one writer
    due_at INTEGER,                        -- NULL once the Sidecar is final
    gap_seconds INTEGER NOT NULL CHECK (gap_seconds >= 0),
    priority TEXT CHECK (priority IN ('admin', 'report')),
    queued_at INTEGER,
    queue_rank INTEGER CHECK (queue_rank BETWEEN 0 AND 4), -- 0 report, 1 admin, 2 fresh missing, 3 older missing, 4 upgrade
    queue_order INTEGER,
    attempt_id BLOB,                       -- the lease; no expiry ("Leases")
    last_searched_at INTEGER,
    last_found INTEGER,
    best_match INTEGER CHECK (best_match BETWEEN 0 AND 100),
    last_result TEXT CHECK (last_result IN ('passed', 'nothing_found', 'none_passed', 'no_provider', 'paused', 'unreachable')),
    PRIMARY KEY (file_id, language, forced),
    CHECK ((queued_at IS NULL) = (queue_rank IS NULL))
) STRICT;
CREATE INDEX subtitles_want_queue ON subtitles_want (kind, queue_rank, queue_order, file_id) WHERE queued_at IS NOT NULL AND attempt_id IS NULL;
CREATE INDEX subtitles_want_due ON subtitles_want (due_at) WHERE queued_at IS NULL AND due_at IS NOT NULL;
CREATE INDEX subtitles_want_unmet ON subtitles_want (work_id, last_result) WHERE unmet = 1;
CREATE INDEX subtitles_want_language ON subtitles_want (language, forced, work_id);
CREATE INDEX subtitles_want_work ON subtitles_want (work_id, file_id);
CREATE INDEX subtitles_want_leased ON subtitles_want (attempt_id) WHERE attempt_id IS NOT NULL;

CREATE TABLE subtitles_sidecar (
    id BLOB PRIMARY KEY NOT NULL,
    file_id BLOB NOT NULL REFERENCES subtitles_file (file_id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    forced INTEGER NOT NULL CHECK (forced IN (0, 1)),
    hi INTEGER NOT NULL CHECK (hi IN (0, 1)),
    tail TEXT NOT NULL CHECK (tail <> '' AND instr(tail, '/') = 0 AND instr(tail, '\') = 0),
    source TEXT NOT NULL CHECK (source IN ('provider', 'upload', 'found', 'sibling')),
    provider TEXT,
    candidate_key TEXT,
    release TEXT,
    ai_translated INTEGER NOT NULL DEFAULT 0 CHECK (ai_translated IN (0, 1)),
    human_made INTEGER NOT NULL DEFAULT 0 CHECK (human_made IN (0, 1)),
    sibling_of BLOB,
    final INTEGER NOT NULL CHECK (final IN (0, 1)),
    timed INTEGER NOT NULL CHECK (timed IN (0, 1)),
    offset_ms INTEGER,
    ratio REAL,
    segments INTEGER,
    match INTEGER CHECK (match BETWEEN 0 AND 100),
    sha256 BLOB NOT NULL CHECK (length(sha256) = 32),
    size INTEGER NOT NULL CHECK (size > 0),
    cues INTEGER NOT NULL CHECK (cues >= 0),
    timing_due INTEGER NOT NULL DEFAULT 0 CHECK (timing_due IN (0, 1)),
    stray TEXT,                            -- root-relative path of the bytes while away from <stem>.<tail>
    added_at INTEGER NOT NULL,
    UNIQUE (file_id, language, forced),
    -- A ledger-known companion a survey adopts after a restore is the
    -- provider's, which it no longer names.
    CHECK (provider IS NULL OR source = 'provider'),
    CHECK ((timed = 1) = (match IS NOT NULL))
) STRICT;
CREATE INDEX subtitles_sidecar_pending ON subtitles_sidecar (file_id) WHERE timing_due = 1 OR stray IS NOT NULL;
CREATE INDEX subtitles_sidecar_language ON subtitles_sidecar (language, forced, source);

-- The intent: an attempt's decision to write bytes, before WriteSidecar.
-- At most one per want, which is also the want's byte lease.
CREATE TABLE subtitles_write (
    file_id BLOB NOT NULL REFERENCES subtitles_file (file_id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    forced INTEGER NOT NULL CHECK (forced IN (0, 1)),
    attempt_id BLOB NOT NULL,
    tail TEXT NOT NULL,
    sha256 BLOB NOT NULL CHECK (length(sha256) = 32),
    sidecar TEXT NOT NULL CHECK (json_valid(sidecar)), -- every subtitles_sidecar field but id and added_at
    replaces BLOB CHECK (replaces IS NULL OR length(replaces) = 32),
    started_at INTEGER NOT NULL,
    PRIMARY KEY (file_id, language, forced)
) STRICT;

CREATE TABLE subtitles_tried (
    file_id BLOB NOT NULL REFERENCES subtitles_file (file_id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    forced INTEGER NOT NULL CHECK (forced IN (0, 1)),
    candidate_key TEXT NOT NULL,
    provider TEXT,
    release TEXT,
    match INTEGER CHECK (match BETWEEN 0 AND 100),
    outcome TEXT NOT NULL CHECK (outcome IN ('passed', 'failed_timing', 'marked_bad', 'unreadable', 'image', 'wrong_episode', 'too_large')),
    tried_at INTEGER NOT NULL,
    PRIMARY KEY (file_id, language, forced, candidate_key)
) STRICT;
CREATE INDEX subtitles_tried_aged ON subtitles_tried (tried_at) WHERE outcome <> 'marked_bad';

-- A doomed file's Sidecar or in-flight intent on its way to the file
-- replacing it (D24). No FKs: from_file is gone in the doom Write and
-- to_file's bytes may not sit at their name yet. Settled under from_file's
-- key, so it waits behind any write in flight for the doomed file.
CREATE TABLE subtitles_parked (
    id BLOB PRIMARY KEY NOT NULL,
    from_file BLOB NOT NULL,
    to_file BLOB NOT NULL,
    what TEXT NOT NULL CHECK (what IN ('sidecar', 'intent')),
    sidecar TEXT NOT NULL CHECK (json_valid(sidecar)),
    root_id BLOB NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE,
    path TEXT NOT NULL CHECK (path <> ''),  -- where the bytes sit, or will sit if the intent's write lands
    sha256 BLOB NOT NULL CHECK (length(sha256) = 32),
    parked_at INTEGER NOT NULL
) STRICT;
CREATE INDEX subtitles_parked_from ON subtitles_parked (from_file);
CREATE INDEX subtitles_parked_to ON subtitles_parked (to_file);

-- Bytes to remove from a Storage Node, only if they still hash to sha256.
-- file_id is the file the bytes belonged to, live or doomed (no FK); the
-- Keeper settles the row under that file's key. Every insert is
-- ON CONFLICT (root_id, path, sha256) DO NOTHING.
CREATE TABLE subtitles_removal (
    id BLOB PRIMARY KEY NOT NULL,
    file_id BLOB NOT NULL,
    root_id BLOB NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE,
    path TEXT NOT NULL CHECK (path <> ''),
    sha256 BLOB NOT NULL CHECK (length(sha256) = 32),
    why TEXT NOT NULL CHECK (why IN ('deleted', 'dropped', 'marked_bad', 'superseded', 'retime_failed')),
    tries INTEGER NOT NULL DEFAULT 0,
    next_at INTEGER NOT NULL,
    UNIQUE (root_id, path, sha256)
) STRICT;
CREATE INDEX subtitles_removal_file ON subtitles_removal (file_id);
CREATE INDEX subtitles_removal_next ON subtitles_removal (next_at);

-- "Don't count this track" (D19): the track's ordinal in
-- media.Record.Subtitles and its fingerprint. The ignore counts while the
-- track at that ordinal has the same codec, language and forced flag; the
-- bulk pass deletes inert ones.
CREATE TABLE subtitles_track_ignored (
    file_id BLOB NOT NULL REFERENCES files_file (id) ON DELETE CASCADE,
    stream_index INTEGER NOT NULL CHECK (stream_index >= 0),
    codec TEXT NOT NULL,
    language TEXT NOT NULL,
    forced INTEGER NOT NULL CHECK (forced IN (0, 1)),
    PRIMARY KEY (file_id, stream_index)
) STRICT;
