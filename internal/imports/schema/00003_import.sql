-- +goose Up
-- S33 (Import pipeline): an Import is a set of video files from one
-- source (a finished download, a folder a library scan found in a root, a
-- folder the Admin picks) into one Work. S22 (Library scan and file
-- probe)'s pending entries become root Imports. hint, suggestion, ended,
-- reason and hold are JSON.
CREATE TABLE imports_import (
    id BLOB PRIMARY KEY NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('grab', 'root', 'folder')),
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
    CHECK (source <> 'grab' OR (node_id IS NOT NULL AND info_hash IS NOT NULL AND folder IS NULL)),
    CHECK (source <> 'root' OR (root_id IS NOT NULL AND folder IS NOT NULL AND node_id IS NULL)),
    CHECK (source <> 'folder' OR (node_id IS NOT NULL AND folder IS NOT NULL AND root_id IS NULL)),
    CHECK (move_files = 0 OR source = 'folder')
) STRICT;
CREATE UNIQUE INDEX imports_import_grab ON imports_import (grab_id) WHERE finished_at IS NULL AND grab_id IS NOT NULL;
CREATE UNIQUE INDEX imports_import_root ON imports_import (root_id, folder) WHERE finished_at IS NULL AND source = 'root';
CREATE UNIQUE INDEX imports_import_folder ON imports_import (node_id, folder, coalesce(work_id, x''), coalesce(edition, ''))
    WHERE finished_at IS NULL AND source = 'folder';
CREATE INDEX imports_import_work ON imports_import (work_id) WHERE finished_at IS NULL;

-- An Import's files by position. path is torrent-relative for a grab,
-- root-relative for a root and absolute on the Node for a folder.
-- source_index is the torrent's file index. dest is the holding path in
-- root_id the bytes go to, or for a root Import the path they sit at.
CREATE TABLE imports_file (
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
    placed_by TEXT CHECK (placed_by IN ('link', 'clone', 'copy', 'transfer', 'move', 'in_place')),
    file_id BLOB REFERENCES files_file (id) ON DELETE SET NULL,
    source_removed_at INTEGER,
    CHECK (outcome NOT IN ('left_out', 'not_better', 'waiting', 'failed', 'declined') OR reason IS NOT NULL),
    CHECK (outcome NOT IN ('sending', 'placing', 'placed') OR reason IS NULL),
    CHECK (outcome <> 'placed' OR placed_by IS NOT NULL),
    CHECK (placed_by IS NULL OR outcome IN ('placing', 'placed')),
    CHECK ((root_id IS NULL) = (dest IS NULL)),
    CHECK ((sent_transfer IS NULL) = (sent_order IS NULL)),
    PRIMARY KEY (import_id, position)
) STRICT, WITHOUT ROWID;
CREATE INDEX imports_file_outcome ON imports_file (outcome) WHERE outcome IN ('pending', 'waiting', 'sending', 'placing');

-- A file's Items in tree order.
CREATE TABLE imports_file_item (
    import_id BLOB NOT NULL,
    position INTEGER NOT NULL,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    "order" INTEGER NOT NULL CHECK ("order" >= 0),
    FOREIGN KEY (import_id, position) REFERENCES imports_file (import_id, position) ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY (import_id, position, item_id)
) STRICT, WITHOUT ROWID;
CREATE INDEX imports_file_item_item ON imports_file_item (item_id);

-- The cells a file fills: decided, then as landed.
CREATE TABLE imports_file_fill (
    import_id BLOB NOT NULL,
    position INTEGER NOT NULL,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    edition TEXT NOT NULL,
    target_id BLOB NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    forced INTEGER NOT NULL DEFAULT 0 CHECK (forced IN (0, 1)),
    FOREIGN KEY (import_id, position) REFERENCES imports_file (import_id, position) ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY (import_id, position, item_id, edition, target_id)
) STRICT, WITHOUT ROWID;

INSERT INTO imports_import (id, source, root_id, folder, content_type, hint, suggestion, created_at, ignored_at)
SELECT id, 'root', root_id, folder, content_type, hint, suggestion, found_at, ignored_at FROM imports_pending;

-- A file that waited several_items goes, so the next scan finds it as new
-- and reads its Items again.
DELETE FROM imports_pending_file WHERE json_extract(reason, '$.code') = 'several_items';

-- A root file's reason is S22's scanning reason, kept whole under scan.
-- Its release and source names are its base name.
INSERT INTO imports_file (import_id, position, path, size, modified_at, node_file_id, node_volume_id, probe, quality, cut,
                          release_name, source_name, outcome, reason)
SELECT pending_id, ROW_NUMBER() OVER (PARTITION BY pending_id ORDER BY path) - 1, path, size, modified_at, node_file_id,
       node_volume_id, probe, quality, cut,
       replace(path, rtrim(path, replace(path, '/', '')), ''), replace(path, rtrim(path, replace(path, '/', '')), ''),
       'waiting', json_object('code', json_extract(reason, '$.code'), 'scan', json(reason))
FROM imports_pending_file;

-- S22 stored Item ids as dashed UUID text, and storage.UUID is 16 bytes.
INSERT INTO imports_file_item (import_id, position, item_id, "order")
SELECT f.import_id, f.position, unhex(replace(j.value, '-', '')), j.key
FROM imports_file f JOIN imports_pending_file p ON p.pending_id = f.import_id AND p.path = f.path, json_each(p.items) j;

DELETE FROM imports_import WHERE NOT EXISTS (SELECT 1 FROM imports_file WHERE import_id = imports_import.id);

DROP TABLE imports_pending_file;
DROP TABLE imports_pending;

-- The Importing settings, each NULL for its default.
ALTER TABLE imports_settings ADD COLUMN video_extensions TEXT;
ALTER TABLE imports_settings ADD COLUMN smallest_video INTEGER;
ALTER TABLE imports_settings ADD COLUMN sample_shorter INTEGER;
