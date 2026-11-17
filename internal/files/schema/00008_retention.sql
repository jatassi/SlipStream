-- +goose Up
-- S38 (Calendar and History): a Grab can be pruned behind a live file, and
-- the facts that outlive an import record move onto the file and the fill.
-- grab_id SET NULLs on a Grab's delete. placed_by is how the file's import
-- placed it, NULL for a file a scan found, backfilled from its newest
-- import record.
CREATE TABLE files_file_next (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    cut TEXT NOT NULL,
    root_id BLOB NOT NULL REFERENCES roots_root (id),
    path TEXT NOT NULL CHECK (path <> ''),
    size INTEGER NOT NULL CHECK (size >= 0),
    probe TEXT NOT NULL,
    grab_id BLOB REFERENCES grabs_grab (id) ON DELETE SET NULL,
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
    placed_by TEXT CHECK (placed_by IN ('link', 'clone', 'copy', 'transfer', 'move', 'in_place')),
    UNIQUE (root_id, path)
) STRICT;

INSERT INTO files_file_next (id, work_id, cut, root_id, path, size, probe, grab_id, quality, label, extra, release_name,
    title_read, modified_at, node_file_id, node_volume_id, gone_at, probed_at, probe_error, source_name, placed_by)
SELECT f.id, f.work_id, f.cut, f.root_id, f.path, f.size, f.probe, f.grab_id, f.quality, f.label, f.extra, f.release_name,
    f.title_read, f.modified_at, f.node_file_id, f.node_volume_id, f.gone_at, f.probed_at, f.probe_error, f.source_name,
    (SELECT i.placed_by FROM files_imported i WHERE i.file_id = f.id ORDER BY i.at DESC, i.rowid DESC LIMIT 1)
FROM files_file f;

DROP TABLE files_file;
ALTER TABLE files_file_next RENAME TO files_file;

CREATE INDEX files_file_work_id ON files_file (work_id);
CREATE INDEX files_file_cut ON files_file (cut);
CREATE INDEX files_file_gone ON files_file (root_id) WHERE gone_at IS NOT NULL;
CREATE INDEX files_file_grab ON files_file (grab_id);

-- forced_import_id is the Import whose Force gave the fill its file. It
-- holds no foreign key for the reason files_imported.import_id holds none.
ALTER TABLE files_fill ADD COLUMN forced_import_id BLOB;
UPDATE files_fill SET forced_import_id = (
    SELECT i.import_id FROM files_imported i
    WHERE i.file_id = files_fill.file_id AND i.import_id IS NOT NULL
    ORDER BY i.at DESC, i.rowid DESC LIMIT 1)
WHERE forced_at IS NOT NULL;

-- Import records by age, for the retention walk, and by Grab, so deleting
-- a Grab SET NULLs its records without a scan.
CREATE INDEX files_imported_at ON files_imported (at);
CREATE INDEX files_imported_grab ON files_imported (grab_id);

-- Nothing reads finished renames.
DROP TABLE files_renamed;
