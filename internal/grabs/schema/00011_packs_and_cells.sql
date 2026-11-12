-- +goose Up
-- The Items a Grab's Release covers in its Work, in tree order. Every Grab
-- before S33 (Import pipeline) covers its one Item.
CREATE TABLE grabs_grab_item (
    grab_id BLOB NOT NULL REFERENCES grabs_grab (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (grab_id, item_id),
    UNIQUE (grab_id, position)
) STRICT, WITHOUT ROWID;

INSERT INTO grabs_grab_item (grab_id, item_id, position) SELECT id, item_id, 0 FROM grabs_grab;

CREATE INDEX grabs_grab_item_item_id ON grabs_grab_item (item_id);

-- grabs_target: each cell a Grab holds, by its Item and Target, whether the
-- Admin forced it, and how it settled. An imported Grab placed every cell.
CREATE TABLE grabs_target_next (
    grab_id BLOB NOT NULL REFERENCES grabs_grab (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    target_id BLOB NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    forced INTEGER NOT NULL DEFAULT 0 CHECK (forced IN (0, 1)),
    settled TEXT CHECK (settled IN ('placed', 'none', 'declined')),
    settled_at INTEGER,
    CHECK ((settled IS NULL) = (settled_at IS NULL)),
    PRIMARY KEY (grab_id, position),
    UNIQUE (grab_id, item_id, target_id)
) STRICT, WITHOUT ROWID;

INSERT INTO grabs_target_next (grab_id, position, item_id, target_id, settled, settled_at)
SELECT t.grab_id, t.position, g.item_id, t.target_id,
    CASE g.stage WHEN 'imported' THEN 'placed' END, CASE g.stage WHEN 'imported' THEN g.updated_at END
FROM grabs_target t JOIN grabs_grab g ON g.id = t.grab_id;

DROP TABLE grabs_target;
ALTER TABLE grabs_target_next RENAME TO grabs_target;

CREATE INDEX grabs_target_target_id ON grabs_target (target_id);
CREATE INDEX grabs_target_item_id ON grabs_target (item_id);

-- The Blocklist gains the three causes an Import finds, none with an Admin.
CREATE TABLE grabs_blocklist_next (
    info_hash BLOB PRIMARY KEY NOT NULL CHECK (length(info_hash) = 20),
    release_title TEXT NOT NULL,
    release_size INTEGER NOT NULL CHECK (release_size >= 0),
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    indexer_name TEXT,
    tracker_id BLOB REFERENCES indexers_tracker (id) ON DELETE SET NULL,
    work_id BLOB REFERENCES works_work (id) ON DELETE SET NULL,
    grab_id BLOB REFERENCES grabs_grab (id) ON DELETE SET NULL,
    cause TEXT NOT NULL CHECK (cause IN ('unregistered', 'no_payload', 'engine_error', 'marked_failed', 'by_hand', 'no_video', 'unreadable', 'misdescribed')),
    client_name TEXT,
    added_by BLOB REFERENCES identity_account (id) ON DELETE SET NULL,
    added_by_name TEXT,
    added_at INTEGER NOT NULL,
    CHECK (cause NOT IN ('unregistered', 'no_payload', 'engine_error', 'no_video', 'unreadable', 'misdescribed') OR (added_by IS NULL AND added_by_name IS NULL)),
    CHECK (cause <> 'engine_error' OR client_name IS NOT NULL)
) STRICT;

INSERT INTO grabs_blocklist_next (info_hash, release_title, release_size, indexer_id, indexer_name, tracker_id, work_id, grab_id, cause,
    client_name, added_by, added_by_name, added_at)
SELECT info_hash, release_title, release_size, indexer_id, indexer_name, tracker_id, work_id, grab_id, cause,
    client_name, added_by, added_by_name, added_at
FROM grabs_blocklist;

DROP TABLE grabs_blocklist;
ALTER TABLE grabs_blocklist_next RENAME TO grabs_blocklist;

CREATE INDEX grabs_blocklist_added_at ON grabs_blocklist (added_at, info_hash);
CREATE INDEX grabs_blocklist_indexer_id ON grabs_blocklist (indexer_id);
CREATE INDEX grabs_blocklist_tracker_id ON grabs_blocklist (tracker_id);
CREATE INDEX grabs_blocklist_work_id ON grabs_blocklist (work_id);
CREATE INDEX grabs_blocklist_grab_id ON grabs_blocklist (grab_id);
CREATE INDEX grabs_blocklist_added_by ON grabs_blocklist (added_by);

-- grabs_release: the ranking gains the pack tie-breaks after score.
CREATE TABLE grabs_release_next (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    title TEXT NOT NULL,
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    indexer_name TEXT NOT NULL,
    size INTEGER NOT NULL,
    seeders INTEGER NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('grabbed', 'not_taken', 'outranked', 'kept', 'refused', 'listed')),
    rank INTEGER CHECK (rank >= 1),
    reasons TEXT NOT NULL CHECK (json_valid(reasons)),
    lost_step TEXT CHECK (lost_step IN ('want', 'tier', 'revision', 'score', 'chosen_by_another', 'fewer_unwanted', 'priority', 'freeleech',
        'seeders', 'size', 'title')),
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
    PRIMARY KEY (search_id, position),
    CHECK (outcome IN ('refused', 'listed') OR rank IS NOT NULL),
    CHECK (lost_step IS NULL OR outcome IN ('outranked', 'kept')),
    CHECK ((lost_step IS NULL) = (lost_own IS NULL) AND (lost_step IS NULL) = (lost_ref IS NULL)),
    CHECK (link_sealed IS NULL OR release IS NOT NULL),
    CHECK ((outcome = 'listed') = (read IS NOT NULL)),
    CHECK (outcome <> 'listed' OR (rank IS NULL AND copy_of IS NULL AND release IS NOT NULL AND link_sealed IS NOT NULL))
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
