-- +goose Up
-- A file holds one or more Items, in tree order: position 0's level folder
-- is the file's folder. Every file before this migration holds the one
-- Item its row named.
CREATE TABLE files_file_item (
    file_id BLOB NOT NULL REFERENCES files_file (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (file_id, item_id),
    UNIQUE (file_id, position)
) STRICT, WITHOUT ROWID;

INSERT INTO files_file_item (file_id, item_id, position) SELECT id, item_id, 0 FROM files_file;

CREATE INDEX files_file_item_item ON files_file_item (item_id);

CREATE TABLE files_file_next (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    cut TEXT NOT NULL,
    root_id BLOB NOT NULL REFERENCES roots_root (id),
    path TEXT NOT NULL CHECK (path <> ''),
    size INTEGER NOT NULL CHECK (size >= 0),
    probe TEXT NOT NULL,
    grab_id BLOB REFERENCES grabs_grab (id),
    quality TEXT NOT NULL DEFAULT 'sdtv' CHECK (quality IN (
        'sdtv', 'dvd', 'webrip-480p', 'hdtv-720p', 'webrip-720p', 'webdl-720p', 'bluray-720p',
        'hdtv-1080p', 'webrip-1080p', 'webdl-1080p', 'bluray-1080p', 'remux-1080p',
        'hdtv-2160p', 'webrip-2160p', 'webdl-2160p', 'bluray-2160p', 'remux-2160p', 'cam')),
    label TEXT NOT NULL DEFAULT '',
    extra INTEGER NOT NULL DEFAULT 0 CHECK (extra >= 0),
    release_name TEXT,
    title_read TEXT CHECK (json_valid(title_read)),
    modified_at INTEGER,
    node_file_id TEXT CHECK (node_file_id <> ''),
    node_volume_id TEXT,
    gone_at INTEGER,
    probe_version INTEGER NOT NULL GENERATED ALWAYS AS (coalesce(json_extract(probe, '$.version'), 0)) VIRTUAL,
    probed_at INTEGER,
    probe_error TEXT CHECK (probe_error <> ''),
    source_name TEXT CHECK (source_name <> ''),
    UNIQUE (root_id, path)
) STRICT;

INSERT INTO files_file_next (id, work_id, cut, root_id, path, size, probe, grab_id, quality, label, extra, release_name,
    title_read, modified_at, node_file_id, node_volume_id, gone_at, probed_at, probe_error, source_name)
SELECT id, work_id, cut, root_id, path, size, probe, grab_id, quality, label, extra, release_name,
    title_read, modified_at, node_file_id, node_volume_id, gone_at, probed_at, probe_error, source_name
FROM files_file;

DROP TABLE files_file;
ALTER TABLE files_file_next RENAME TO files_file;

CREATE INDEX files_file_work_id ON files_file (work_id);
CREATE INDEX files_file_cut ON files_file (cut);
CREATE INDEX files_file_gone ON files_file (root_id) WHERE gone_at IS NOT NULL;

-- forced_at is when an Admin's Force gave the fill its file: the fill
-- yields only to a better file arriving from another Import.
ALTER TABLE files_fill ADD COLUMN forced_at INTEGER;

-- Every landing of a file, with what it filled and a snapshot of each file
-- it replaced, for S35 (Media Servers), S37 (Notifications) and S38
-- (Calendar and History). import_id names an imports_import row but
-- holds no foreign key: that table comes in a later migration of the
-- imports set, and SQLite refuses any write to a table whose foreign key
-- names a table that isn't there.
CREATE TABLE files_imported (
    id BLOB PRIMARY KEY NOT NULL,
    file_id BLOB REFERENCES files_file (id) ON DELETE SET NULL,
    import_id BLOB,
    grab_id BLOB REFERENCES grabs_grab (id) ON DELETE SET NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    placed_by TEXT NOT NULL CHECK (placed_by IN ('link', 'clone', 'copy', 'transfer', 'move', 'in_place')),
    items TEXT NOT NULL CHECK (json_valid(items)),
    cells TEXT NOT NULL CHECK (json_valid(cells)),
    replaced TEXT NOT NULL CHECK (json_valid(replaced)),
    at INTEGER NOT NULL
) STRICT;

CREATE INDEX files_imported_file ON files_imported (file_id, at);
CREATE INDEX files_imported_work ON files_imported (work_id, at);
