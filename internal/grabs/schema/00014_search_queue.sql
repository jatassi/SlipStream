-- +goose Up
-- S34 (Automatic search, Backlog search and RSS sync). A search run becomes
-- a group run, every Grab a run took or joined is a join row, lines keep a
-- durable deadline, and the queue, failures, waits and settings get their
-- tables. The migration connection runs with foreign keys off, so the
-- pruning at the end deletes children before their runs.

-- grabs_search: a group run. Three kinds, told apart by manual_id and
-- trigger: a Manual run, an RSS run (decided in the Write that read its
-- feed) and a queue run. lane is a rank, presses first. grabs counts the
-- Grabs and joins it took when it decided.
CREATE TABLE grabs_search_next (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    group_key TEXT CHECK (group_key IS NULL OR length(group_key) BETWEEN 3 AND 200),
    lane INTEGER CHECK (lane IN (10, 30, 40)),
    trigger TEXT CHECK (trigger IN ('search', 'search_all', 'add', 'monitor', 'profile', 'editions', 'release_date', 'refresh',
        'backlog', 'failed_grab', 'rss')),
    include_gone INTEGER NOT NULL DEFAULT 0 CHECK (include_gone IN (0, 1)),
    phase INTEGER NOT NULL DEFAULT 1 CHECK (phase IN (1, 2)),
    manual_id BLOB REFERENCES grabs_manual (id) ON DELETE CASCADE,
    started_at INTEGER NOT NULL,
    finished_at INTEGER,
    outcome TEXT CHECK (outcome IN ('grabbed', 'waiting', 'nothing_found', 'nothing_acceptable', 'no_indexer', 'no_root',
        'listed')),
    candidates INTEGER NOT NULL DEFAULT 0 CHECK (candidates >= 0),
    grabs INTEGER NOT NULL DEFAULT 0 CHECK (grabs >= 0),
    notes TEXT CHECK (notes IS NULL OR json_valid(notes)),
    CHECK ((finished_at IS NULL) = (outcome IS NULL)),
    CHECK (outcome IS NULL OR (outcome = 'grabbed') = (grabs > 0)),
    CHECK (CASE
        WHEN manual_id IS NOT NULL THEN group_key IS NULL AND lane IS NULL AND trigger IS NULL AND include_gone = 0 AND phase = 1
            AND (outcome IS NULL OR outcome = 'listed')
        WHEN trigger = 'rss' THEN group_key IS NULL AND lane IS NULL AND include_gone = 0 AND phase = 1 AND outcome IS NOT NULL
            AND outcome NOT IN ('listed', 'no_indexer', 'no_root')
        ELSE group_key IS NOT NULL AND lane IS NOT NULL AND trigger IS NOT NULL AND (outcome IS NULL OR outcome <> 'listed')
    END)
) STRICT;

-- An automatic run was one Item's press: it becomes a press of its Item's
-- level, or of the Item when it has none (indexers.LevelGroup and
-- ItemGroup), with gone cells. A nothing_acceptable run with no candidates
-- is nothing_found, the meaning this split off.
INSERT INTO grabs_search_next (id, work_id, group_key, lane, trigger, include_gone, phase, manual_id, started_at, finished_at,
    outcome, candidates, grabs, notes)
SELECT s.id, s.work_id,
    CASE WHEN s.manual_id IS NULL THEN
        CASE WHEN i.level_id IS NULL THEN 'i:' || lower(hex(s.item_id)) ELSE 'l:' || lower(hex(i.level_id)) END
    END,
    CASE WHEN s.manual_id IS NULL THEN 10 END,
    CASE WHEN s.manual_id IS NULL THEN 'search' END,
    CASE WHEN s.manual_id IS NULL THEN 1 ELSE 0 END,
    1, s.manual_id, s.started_at, s.finished_at,
    CASE WHEN s.outcome = 'nothing_acceptable' AND s.candidates = 0 THEN 'nothing_found' ELSE s.outcome END,
    s.candidates,
    CASE WHEN s.grab_id IS NOT NULL THEN 1 ELSE 0 END,
    s.notes
FROM grabs_search s JOIN works_item i ON i.id = s.item_id;

-- The Items a run searched, in place of item_id.
CREATE TABLE grabs_search_item (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    PRIMARY KEY (search_id, item_id)
) STRICT, WITHOUT ROWID;

INSERT INTO grabs_search_item (search_id, item_id) SELECT id, item_id FROM grabs_search;

-- Every Grab a run took (made) or joined. A Grab has at most one making
-- run. The earliest run naming a Grab made it, since a joining run named
-- it too.
CREATE TABLE grabs_search_grab (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    grab_id BLOB NOT NULL REFERENCES grabs_grab (id) ON DELETE CASCADE,
    made INTEGER NOT NULL CHECK (made IN (0, 1)),
    PRIMARY KEY (search_id, grab_id)
) STRICT, WITHOUT ROWID;

INSERT INTO grabs_search_grab (search_id, grab_id, made)
SELECT s.id, s.grab_id,
    CASE WHEN s.id = (SELECT t.id FROM grabs_search t WHERE t.grab_id = s.grab_id ORDER BY t.started_at, t.id LIMIT 1) THEN 1 ELSE 0 END
FROM grabs_search s WHERE s.grab_id IS NOT NULL;

-- grabs_grab: the source rss; release_guid, for refusals that outlive the
-- run's Releases; chosen_fills, the Items it was chosen to fill when the
-- fewer-Releases step decided.
CREATE TABLE grabs_grab_next (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    edition TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'search' CHECK (source IN ('search', 'manual', 'rss')),
    manual_id BLOB REFERENCES grabs_manual (id) ON DELETE SET NULL,
    grabbed_by BLOB REFERENCES identity_account (id) ON DELETE SET NULL,
    grabbed_by_name TEXT,
    overrides TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(overrides)),
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    tracker_id BLOB REFERENCES indexers_tracker (id),
    node_id BLOB NOT NULL REFERENCES nodes_node (id),
    release_title TEXT NOT NULL,
    release_size INTEGER NOT NULL,
    release_link_sealed BLOB,
    release_magnet TEXT,
    release_info_hash BLOB,
    release_guid TEXT CHECK (release_guid IS NULL OR length(CAST(release_guid AS BLOB)) BETWEEN 1 AND 512),
    chosen_fills INTEGER CHECK (chosen_fills IS NULL OR chosen_fills >= 2),
    fetch_query_id BLOB,
    min_ratio REAL,
    min_seed_seconds INTEGER,
    download_factor REAL,
    upload_factor REAL,
    stage TEXT NOT NULL CHECK (stage IN ('fetching', 'downloading', 'importing', 'imported', 'failed')),
    torrent_id BLOB REFERENCES downloadclients_torrent (id),
    probe TEXT,
    failed_stage TEXT,
    failure_code TEXT,
    failure_message TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    fetch_node_id BLOB REFERENCES nodes_node (id),
    metadata_started_at INTEGER,
    sent_built_in_on BLOB REFERENCES nodes_node (id),
    sent_client_id BLOB REFERENCES downloadclients_client (id),
    CHECK (stage <> 'fetching' OR release_link_sealed IS NOT NULL OR release_magnet IS NOT NULL),
    CHECK (stage IN ('fetching', 'failed') OR torrent_id IS NOT NULL),
    CHECK ((stage = 'failed') = (failure_code IS NOT NULL)),
    CHECK (sent_client_id IS NULL OR sent_built_in_on IS NULL),
    CHECK (source = 'manual' OR (manual_id IS NULL AND grabbed_by IS NULL AND grabbed_by_name IS NULL AND overrides = '{}'))
) STRICT;

INSERT INTO grabs_grab_next (
    id, work_id, item_id, edition, source, manual_id, grabbed_by, grabbed_by_name, overrides, indexer_id, tracker_id, node_id,
    release_title, release_size, release_link_sealed, release_magnet, release_info_hash, fetch_query_id, min_ratio,
    min_seed_seconds, download_factor, upload_factor, stage, torrent_id, probe, failed_stage, failure_code, failure_message,
    created_at, updated_at, fetch_node_id, metadata_started_at, sent_built_in_on, sent_client_id)
SELECT
    id, work_id, item_id, edition, source, manual_id, grabbed_by, grabbed_by_name, overrides, indexer_id, tracker_id, node_id,
    release_title, release_size, release_link_sealed, release_magnet, release_info_hash, fetch_query_id, min_ratio,
    min_seed_seconds, download_factor, upload_factor, stage, torrent_id, probe, failed_stage, failure_code, failure_message,
    created_at, updated_at, fetch_node_id, metadata_started_at, sent_built_in_on, sent_client_id
FROM grabs_grab;

DROP TABLE grabs_search;
ALTER TABLE grabs_search_next RENAME TO grabs_search;
DROP TABLE grabs_grab;
ALTER TABLE grabs_grab_next RENAME TO grabs_grab;

CREATE INDEX grabs_search_work_id ON grabs_search (work_id, started_at);
CREATE INDEX grabs_search_open ON grabs_search (finished_at, started_at);
CREATE INDEX grabs_search_group ON grabs_search (work_id, group_key, finished_at) WHERE manual_id IS NULL;
CREATE INDEX grabs_search_manual_id ON grabs_search (manual_id);
CREATE INDEX grabs_search_item_item_id ON grabs_search_item (item_id, search_id);
CREATE INDEX grabs_search_grab_grab_id ON grabs_search_grab (grab_id, made);
CREATE UNIQUE INDEX grabs_search_grab_made ON grabs_search_grab (grab_id) WHERE made = 1;

CREATE INDEX grabs_grab_stage ON grabs_grab (stage, created_at);
CREATE INDEX grabs_grab_work_id ON grabs_grab (work_id);
CREATE INDEX grabs_grab_node_id ON grabs_grab (node_id);
CREATE INDEX grabs_grab_torrent_id ON grabs_grab (torrent_id);
CREATE INDEX grabs_grab_indexer_id ON grabs_grab (indexer_id);
CREATE INDEX grabs_grab_tracker_id ON grabs_grab (tracker_id);
CREATE INDEX grabs_grab_manual_id ON grabs_grab (manual_id);
CREATE INDEX grabs_grab_grabbed_by ON grabs_grab (grabbed_by);

-- grabs_search_indexer: a line. item_id is a single-Item query, NULL for
-- the group's. Its state is its columns: planned (query_id and outcome
-- NULL), in flight (query_id set, outcome NULL), done (outcome set).
-- dispatched_at and deadline_at keep the deadline across a restart.
CREATE TABLE grabs_search_indexer_next (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    indexer_name TEXT NOT NULL,
    item_id BLOB REFERENCES works_item (id) ON DELETE CASCADE,
    query_id BLOB,
    dispatched_at INTEGER,
    deadline_at INTEGER,
    outcome TEXT CHECK (outcome IN ('found', 'no_releases', 'limited', 'failing', 'stopped', 'waiting', 'late',
        'limit_reached', 'no_categories', 'removed', 'rss_only')),
    releases INTEGER CHECK (releases >= 0),
    until_at INTEGER,
    since_at INTEGER,
    node_id BLOB,
    lapsed INTEGER CHECK (lapsed IN (0, 1)),
    cause TEXT,
    text TEXT,
    PRIMARY KEY (search_id, position),
    CHECK ((dispatched_at IS NULL) = (deadline_at IS NULL)),
    CHECK (dispatched_at IS NULL OR query_id IS NOT NULL),
    CHECK (outcome IS NOT NULL OR query_id IS NULL OR dispatched_at IS NOT NULL)
) STRICT;

-- A line with a query had it reserved when its run started, so it was
-- dispatched then, with the default wait of two minutes as its deadline.
INSERT INTO grabs_search_indexer_next (search_id, position, indexer_id, indexer_name, query_id, dispatched_at, deadline_at, outcome,
    releases, until_at, since_at, node_id, lapsed, cause, text)
SELECT l.search_id, l.position, l.indexer_id, l.indexer_name, l.query_id,
    CASE WHEN l.query_id IS NOT NULL THEN s.started_at END,
    CASE WHEN l.query_id IS NOT NULL THEN s.started_at + 120000000 END,
    l.outcome, l.releases, l.until_at, l.since_at, l.node_id, l.lapsed, l.cause, l.text
FROM grabs_search_indexer l JOIN grabs_search s ON s.id = l.search_id;

DROP TABLE grabs_search_indexer;
ALTER TABLE grabs_search_indexer_next RENAME TO grabs_search_indexer;

CREATE INDEX grabs_search_indexer_indexer_id ON grabs_search_indexer (indexer_id);
CREATE INDEX grabs_search_indexer_item_id ON grabs_search_indexer (item_id) WHERE item_id IS NOT NULL;
CREATE INDEX grabs_search_indexer_open ON grabs_search_indexer (indexer_id, dispatched_at) WHERE outcome IS NULL;

-- grabs_release: heard, a line's answer kept whole until its run decides,
-- and waiting, a candidate a wait holds. heard_at is when the line
-- answered.
CREATE TABLE grabs_release_next (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    title TEXT NOT NULL,
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    indexer_name TEXT NOT NULL,
    size INTEGER NOT NULL,
    seeders INTEGER NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('grabbed', 'not_taken', 'outranked', 'kept', 'refused', 'listed', 'heard',
        'waiting')),
    rank INTEGER CHECK (rank >= 1),
    reasons TEXT NOT NULL CHECK (json_valid(reasons)),
    lost_step TEXT CHECK (lost_step IN ('want', 'fewer_releases', 'tier', 'revision', 'score', 'chosen_by_another', 'fewer_unwanted', 'priority',
        'freeleech', 'seeders', 'size', 'title')),
    lost_target_id BLOB REFERENCES targets_target (id) ON DELETE SET NULL,
    lost_own TEXT,
    lost_ref TEXT,
    scores TEXT NOT NULL CHECK (json_valid(scores)),
    formats TEXT NOT NULL CHECK (json_valid(formats)),
    covers TEXT NOT NULL CHECK (json_valid(covers)),
    copy_of INTEGER CHECK (copy_of >= 0),
    release TEXT CHECK (release IS NULL OR json_valid(release)),
    link_sealed BLOB,
    read TEXT CHECK (read IS NULL OR json_valid(read)),
    heard_at INTEGER,
    PRIMARY KEY (search_id, position),
    CHECK (outcome IN ('refused', 'listed', 'heard') OR rank IS NOT NULL),
    CHECK (lost_step IS NULL OR outcome IN ('outranked', 'kept')),
    CHECK ((lost_step IS NULL) = (lost_own IS NULL) AND (lost_step IS NULL) = (lost_ref IS NULL)),
    CHECK (link_sealed IS NULL OR release IS NOT NULL),
    CHECK ((outcome = 'listed') = (read IS NOT NULL)),
    CHECK (outcome <> 'listed' OR (rank IS NULL AND copy_of IS NULL AND release IS NOT NULL AND link_sealed IS NOT NULL)),
    CHECK ((outcome = 'heard') = (heard_at IS NOT NULL)),
    CHECK (outcome <> 'heard' OR rank IS NULL),
    CHECK (outcome NOT IN ('heard', 'waiting') OR (release IS NOT NULL AND copy_of IS NULL AND link_sealed IS NOT NULL))
) STRICT;

INSERT INTO grabs_release_next (search_id, position, title, indexer_id, indexer_name, size, seeders, outcome, rank, reasons,
    lost_step, lost_target_id, lost_own, lost_ref, scores, formats, covers, copy_of, release, link_sealed, read)
SELECT search_id, position, title, indexer_id, indexer_name, size, seeders, outcome, rank, reasons, lost_step, lost_target_id,
    lost_own, lost_ref, scores, formats, covers, copy_of, release, link_sealed, read
FROM grabs_release;

DROP TABLE grabs_release;
ALTER TABLE grabs_release_next RENAME TO grabs_release;

CREATE INDEX grabs_release_indexer_id ON grabs_release (indexer_id);
CREATE INDEX grabs_release_lost_target_id ON grabs_release (lost_target_id);

-- When any automatic run first searched an Item, read before the pruning
-- below drops older runs. Deleted when the Item's last Want closes.
CREATE TABLE grabs_item_search (
    item_id BLOB PRIMARY KEY NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    first_searched_at INTEGER NOT NULL
) STRICT;

INSERT INTO grabs_item_search (item_id, first_searched_at)
SELECT si.item_id, min(s.started_at) FROM grabs_search_item si JOIN grabs_search s ON s.id = si.search_id
WHERE s.manual_id IS NULL GROUP BY si.item_id;

-- One row per trigger, written in the Write that caused it, and drained
-- oldest first into grabs_due. lane NULL only resets the gap. items NULL
-- is the whole Work.
CREATE TABLE grabs_nudge (
    id INTEGER PRIMARY KEY,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    trigger TEXT NOT NULL CHECK (trigger IN ('search', 'search_all', 'add', 'monitor', 'profile', 'editions', 'retarget',
        'release_date', 'refresh', 'backlog', 'failed_grab')),
    lane INTEGER CHECK (lane IN (10, 30, 40)),
    items TEXT CHECK (items IS NULL OR (json_valid(items) AND json_type(items) = 'array')),
    created_at INTEGER NOT NULL
) STRICT;

CREATE INDEX grabs_nudge_work_id ON grabs_nudge (work_id);

-- One row per Work and search group Backlog search paces. group_key is the
-- Content Type's opaque key. The queued entry is lane, queued_at, trigger,
-- include_gone and the order keys; lane NULL is not queued. search_id is
-- the open run. changed_at is the last time any writer touched the row, so
-- a reconcile never deletes what a trigger queued after its snapshot. The
-- gap's bounds are the settings' ranges: the shortest Backlog interval and
-- the longest gap.
CREATE TABLE grabs_due (
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    group_key TEXT NOT NULL CHECK (length(group_key) BETWEEN 3 AND 200),
    lane INTEGER CHECK (lane IN (10, 30, 40)),
    queued_at INTEGER,
    trigger TEXT CHECK (trigger IN ('search', 'search_all', 'add', 'monitor', 'profile', 'editions', 'release_date', 'refresh',
        'backlog', 'failed_grab')),
    include_gone INTEGER NOT NULL DEFAULT 0 CHECK (include_gone IN (0, 1)),
    upgrade_only INTEGER NOT NULL DEFAULT 0 CHECK (upgrade_only IN (0, 1)),
    fresh INTEGER NOT NULL DEFAULT 0 CHECK (fresh IN (0, 1)),
    released_at INTEGER,
    due_at INTEGER NOT NULL,
    gap_seconds INTEGER NOT NULL CHECK (gap_seconds BETWEEN 3600 AND 7776000),
    search_id BLOB REFERENCES grabs_search (id) ON DELETE SET NULL,
    changed_at INTEGER NOT NULL,
    PRIMARY KEY (work_id, group_key),
    CHECK ((lane IS NULL) = (queued_at IS NULL) AND (lane IS NULL) = (trigger IS NULL)),
    CHECK (lane IS NOT NULL OR include_gone = 0)
) STRICT, WITHOUT ROWID;

CREATE INDEX grabs_due_queue ON grabs_due (lane, upgrade_only, fresh DESC, released_at DESC, queued_at)
    WHERE lane IS NOT NULL AND search_id IS NULL;
CREATE INDEX grabs_due_due_at ON grabs_due (due_at) WHERE lane IS NULL;
CREATE INDEX grabs_due_search_id ON grabs_due (search_id) WHERE search_id IS NOT NULL;

-- The queued scope. No rows: the whole group.
CREATE TABLE grabs_due_item (
    work_id BLOB NOT NULL,
    group_key TEXT NOT NULL,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    PRIMARY KEY (work_id, group_key, item_id),
    FOREIGN KEY (work_id, group_key) REFERENCES grabs_due (work_id, group_key) ON DELETE CASCADE
) STRICT, WITHOUT ROWID;

CREATE INDEX grabs_due_item_item_id ON grabs_due_item (item_id);

-- One row per failed Grab and cell it held unsettled. No FK to grabs_grab,
-- so the stop's count and the refusal outlive pruning. code has no CHECK,
-- so a new Code needs no rebuild. counted rows count toward the stop, and
-- stopped_at is set on the row whose insert reached it. refuses rows
-- refuse the Release for the Work, by info hash or by Indexer and GUID.
CREATE TABLE grabs_failure (
    grab_id BLOB NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    edition TEXT NOT NULL,
    target_id BLOB NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    info_hash BLOB CHECK (info_hash IS NULL OR length(info_hash) = 20),
    indexer_id BLOB,
    guid TEXT CHECK (guid IS NULL OR length(CAST(guid AS BLOB)) BETWEEN 1 AND 512),
    failed_at INTEGER NOT NULL,
    counted INTEGER NOT NULL CHECK (counted IN (0, 1)),
    refuses INTEGER NOT NULL CHECK (refuses IN (0, 1)),
    stopped_at INTEGER,
    PRIMARY KEY (grab_id, item_id, edition, target_id),
    CHECK (stopped_at IS NULL OR counted = 1)
) STRICT, WITHOUT ROWID;

CREATE INDEX grabs_failure_cell ON grabs_failure (item_id, edition, target_id, failed_at);
CREATE INDEX grabs_failure_work_id ON grabs_failure (work_id) WHERE refuses = 1;
CREATE INDEX grabs_failure_target_id ON grabs_failure (target_id);

-- A cell waiting for a better Release until until_at. Holds are per
-- Target, so two Targets of a cell's Edition wait apart.
CREATE TABLE grabs_wait (
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    edition TEXT NOT NULL,
    target_id BLOB NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    until_at INTEGER NOT NULL,
    started_at INTEGER NOT NULL,
    PRIMARY KEY (item_id, edition, target_id),
    CHECK (until_at > started_at)
) STRICT, WITHOUT ROWID;

CREATE INDEX grabs_wait_until_at ON grabs_wait (until_at);
CREATE INDEX grabs_wait_work_id ON grabs_wait (work_id);
CREATE INDEX grabs_wait_target_id ON grabs_wait (target_id);

-- The candidates a wait holds: Release rows, kept whole, of the runs that
-- found them. A wait that ends deletes its rows, and the runs fall under
-- the bound.
CREATE TABLE grabs_wait_candidate (
    item_id BLOB NOT NULL,
    edition TEXT NOT NULL,
    target_id BLOB NOT NULL,
    search_id BLOB NOT NULL,
    position INTEGER NOT NULL,
    PRIMARY KEY (item_id, edition, target_id, search_id, position),
    FOREIGN KEY (item_id, edition, target_id) REFERENCES grabs_wait (item_id, edition, target_id) ON DELETE CASCADE,
    FOREIGN KEY (search_id, position) REFERENCES grabs_release (search_id, position) ON DELETE CASCADE
) STRICT, WITHOUT ROWID;

CREATE INDEX grabs_wait_candidate_release ON grabs_wait_candidate (search_id, position);

-- The Automatic search settings. NULL is the default, kept in Go.
CREATE TABLE grabs_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    backlog_on INTEGER CHECK (backlog_on IN (0, 1)),
    backlog_upgrades INTEGER CHECK (backlog_upgrades IN (0, 1)),
    rss_on INTEGER CHECK (rss_on IN (0, 1)),
    longest_gap_seconds INTEGER CHECK (longest_gap_seconds IS NULL OR longest_gap_seconds BETWEEN 86400 AND 7776000),
    full_pace_seconds INTEGER CHECK (full_pace_seconds IS NULL OR full_pace_seconds BETWEEN 0 AND 7776000),
    stop_count INTEGER CHECK (stop_count IS NULL OR stop_count BETWEEN 1 AND 20),
    stop_window_seconds INTEGER CHECK (stop_window_seconds IS NULL OR stop_window_seconds BETWEEN 3600 AND 2592000)
) STRICT;

INSERT INTO grabs_settings (id) VALUES (1);

-- The run bound. A finished queue run is pruned once every Item it
-- searched is in a later finished queue run of its group, no wait holds
-- one of its Releases and no Grab it made still fetches. So each group
-- keeps its latest run and each Item's last result.
CREATE TEMP TABLE pruned AS
SELECT s.id FROM grabs_search s
WHERE s.manual_id IS NULL AND s.group_key IS NOT NULL AND s.finished_at IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM grabs_search_item si WHERE si.search_id = s.id AND NOT EXISTS (
            SELECT 1 FROM grabs_search_item ti JOIN grabs_search t ON t.id = ti.search_id
            WHERE ti.item_id = si.item_id AND t.work_id = s.work_id AND t.group_key = s.group_key AND t.manual_id IS NULL
                AND t.finished_at IS NOT NULL AND (t.finished_at, t.id) > (s.finished_at, s.id)))
    AND NOT EXISTS (SELECT 1 FROM grabs_wait_candidate c WHERE c.search_id = s.id)
    AND NOT EXISTS (
        SELECT 1 FROM grabs_search_grab sg JOIN grabs_grab g ON g.id = sg.grab_id
        WHERE sg.search_id = s.id AND sg.made = 1 AND g.stage = 'fetching');

DELETE FROM grabs_search_grab WHERE search_id IN (SELECT id FROM pruned);
DELETE FROM grabs_release WHERE search_id IN (SELECT id FROM pruned);
DELETE FROM grabs_search_indexer WHERE search_id IN (SELECT id FROM pruned);
DELETE FROM grabs_search_item WHERE search_id IN (SELECT id FROM pruned);
DELETE FROM grabs_search WHERE id IN (SELECT id FROM pruned);
DROP TABLE pruned;
