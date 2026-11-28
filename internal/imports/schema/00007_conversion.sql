-- +goose Up
-- S44 (Conversion): an Import of the file a Conversion made, into the
-- root the Conversion placed it for. It starts once per Conversion. Its
-- file carries the release read and quality the Conversion predicted, and
-- is placed 'converted'. conversion_id and work_id SET NULL when their
-- rows go, as a Grab's do, so only the root is required of the source.
CREATE TABLE imports_import_next (
    id BLOB PRIMARY KEY NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('grab', 'root', 'folder', 'conversion')),
    grab_id BLOB REFERENCES grabs_grab (id) ON DELETE SET NULL,
    node_id BLOB,
    info_hash BLOB CHECK (info_hash IS NULL OR length(info_hash) = 20),
    root_id BLOB REFERENCES roots_root (id) ON DELETE CASCADE,
    folder TEXT CHECK (folder <> ''),
    content_type TEXT NOT NULL,
    hint TEXT NOT NULL,
    suggestion TEXT,
    work_id BLOB REFERENCES works_work (id) ON DELETE SET NULL,
    edition TEXT,
    move_files INTEGER NOT NULL DEFAULT 0 CHECK (move_files IN (0, 1)),
    failure_code TEXT CHECK (failure_code IN ('no_video_file', 'unreadable')),
    ended TEXT,
    created_at INTEGER NOT NULL,
    ignored_at INTEGER,
    finished_at INTEGER,
    conversion_id BLOB REFERENCES conversions_conversion (id) ON DELETE SET NULL,
    CHECK (source <> 'grab' OR (node_id IS NOT NULL AND info_hash IS NOT NULL AND folder IS NULL)),
    CHECK (source <> 'root' OR (root_id IS NOT NULL AND folder IS NOT NULL AND node_id IS NULL)),
    CHECK (source <> 'folder' OR (node_id IS NOT NULL AND folder IS NOT NULL AND root_id IS NULL)),
    CHECK (source <> 'conversion' OR root_id IS NOT NULL),
    CHECK (conversion_id IS NULL OR source = 'conversion'),
    CHECK (move_files = 0 OR source = 'folder')
) STRICT;

INSERT INTO imports_import_next (id, source, grab_id, node_id, info_hash, root_id, folder, content_type, hint, suggestion,
    work_id, edition, move_files, failure_code, ended, created_at, ignored_at, finished_at)
SELECT id, source, grab_id, node_id, info_hash, root_id, folder, content_type, hint, suggestion,
    work_id, edition, move_files, failure_code, ended, created_at, ignored_at, finished_at
FROM imports_import;

DROP TABLE imports_import;
ALTER TABLE imports_import_next RENAME TO imports_import;

CREATE UNIQUE INDEX imports_import_folder ON imports_import (node_id, folder, coalesce(work_id, x''), coalesce(edition, ''))
    WHERE finished_at IS NULL AND source = 'folder';
CREATE UNIQUE INDEX imports_import_grab ON imports_import (grab_id) WHERE finished_at IS NULL AND grab_id IS NOT NULL;
CREATE INDEX imports_import_grab_any ON imports_import (grab_id);
CREATE INDEX imports_import_prunable ON imports_import (finished_at, id) WHERE finished_at IS NOT NULL AND grab_id IS NULL;
CREATE UNIQUE INDEX imports_import_root ON imports_import (root_id, folder) WHERE finished_at IS NULL AND source = 'root';
CREATE INDEX imports_import_work ON imports_import (work_id) WHERE finished_at IS NULL;
CREATE UNIQUE INDEX imports_import_conversion ON imports_import (conversion_id) WHERE conversion_id IS NOT NULL;

CREATE TABLE imports_file_next (
    import_id BLOB NOT NULL REFERENCES imports_import (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    path TEXT NOT NULL CHECK (path <> ''),
    source_index INTEGER CHECK (source_index >= 0),
    size INTEGER NOT NULL CHECK (size >= 0),
    modified_at INTEGER,
    node_file_id TEXT CHECK (node_file_id <> ''),
    node_volume_id TEXT,
    probe TEXT,
    quality TEXT,
    cut TEXT NOT NULL,
    release_name TEXT NOT NULL,
    source_name TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('pending', 'left_out', 'not_better', 'waiting', 'sending', 'placing', 'placed', 'declined', 'failed')),
    reason TEXT,
    hold TEXT,
    confirmed INTEGER NOT NULL DEFAULT 0 CHECK (confirmed IN (0, 1)),
    root_id BLOB REFERENCES roots_root (id) ON DELETE SET NULL,
    dest TEXT CHECK (dest <> ''),
    sent_transfer BLOB,
    sent_order INTEGER,
    placed_by TEXT CHECK (placed_by IN ('link', 'clone', 'copy', 'transfer', 'move', 'in_place', 'converted')),
    file_id BLOB REFERENCES files_file (id) ON DELETE SET NULL,
    source_removed_at INTEGER,
    seeded INTEGER NOT NULL DEFAULT 0 CHECK (seeded IN (0, 1)),
    told_at INTEGER,
    -- The release.Read a Conversion's file carries, as JSON.
    title_read TEXT CHECK (json_valid(title_read)),
    CHECK (outcome NOT IN ('left_out', 'not_better', 'waiting', 'failed', 'declined') OR reason IS NOT NULL),
    CHECK (outcome NOT IN ('sending', 'placing', 'placed') OR reason IS NULL),
    CHECK (outcome <> 'placed' OR placed_by IS NOT NULL),
    CHECK (placed_by IS NULL OR outcome IN ('placing', 'placed')),
    CHECK ((root_id IS NULL) = (dest IS NULL)),
    CHECK ((sent_transfer IS NULL) = (sent_order IS NULL)),
    PRIMARY KEY (import_id, position)
) STRICT, WITHOUT ROWID;

INSERT INTO imports_file_next (import_id, position, path, source_index, size, modified_at, node_file_id, node_volume_id, probe,
    quality, cut, release_name, source_name, outcome, reason, hold, confirmed, root_id, dest, sent_transfer, sent_order,
    placed_by, file_id, source_removed_at, seeded, told_at)
SELECT import_id, position, path, source_index, size, modified_at, node_file_id, node_volume_id, probe,
    quality, cut, release_name, source_name, outcome, reason, hold, confirmed, root_id, dest, sent_transfer, sent_order,
    placed_by, file_id, source_removed_at, seeded, told_at
FROM imports_file;

DROP TABLE imports_file;
ALTER TABLE imports_file_next RENAME TO imports_file;

CREATE INDEX imports_file_outcome ON imports_file (outcome) WHERE outcome IN ('pending', 'waiting', 'sending', 'placing');
