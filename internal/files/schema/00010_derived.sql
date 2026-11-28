-- +goose Up
-- S44 (Conversion): a file a Conversion made remembers the file it was
-- made from, the Target it was made for and that Target's settings
-- version. derived stays 1 once the source and the Target are gone, so a
-- made file stays made. A made file is placed 'converted'.
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
    placed_by TEXT CHECK (placed_by IN ('link', 'clone', 'copy', 'transfer', 'move', 'in_place', 'converted')),
    derived_from_file_id BLOB REFERENCES files_file (id) ON DELETE SET NULL,
    derived_target_id BLOB REFERENCES targets_target (id) ON DELETE SET NULL,
    conversion_version INTEGER CHECK (conversion_version >= 1),
    derived INTEGER NOT NULL DEFAULT 0 CHECK (derived IN (0, 1)),
    UNIQUE (root_id, path),
    CHECK (derived = 1 OR (derived_from_file_id IS NULL AND derived_target_id IS NULL AND conversion_version IS NULL))
) STRICT;

INSERT INTO files_file_next (id, work_id, cut, root_id, path, size, probe, grab_id, quality, label, extra, release_name,
    title_read, modified_at, node_file_id, node_volume_id, gone_at, probed_at, probe_error, source_name, placed_by)
SELECT id, work_id, cut, root_id, path, size, probe, grab_id, quality, label, extra, release_name,
    title_read, modified_at, node_file_id, node_volume_id, gone_at, probed_at, probe_error, source_name, placed_by
FROM files_file;

DROP TABLE files_file;
ALTER TABLE files_file_next RENAME TO files_file;

CREATE INDEX files_file_work_id ON files_file (work_id);
CREATE INDEX files_file_cut ON files_file (cut);
CREATE INDEX files_file_gone ON files_file (root_id) WHERE gone_at IS NOT NULL;
CREATE INDEX files_file_grab ON files_file (grab_id);
CREATE INDEX files_file_derived_from ON files_file (derived_from_file_id);
CREATE INDEX files_file_derived_target ON files_file (derived_target_id, conversion_version) WHERE derived = 1;

CREATE TABLE files_imported_next (
    id BLOB PRIMARY KEY NOT NULL,
    file_id BLOB REFERENCES files_file (id) ON DELETE SET NULL,
    import_id BLOB,
    grab_id BLOB REFERENCES grabs_grab (id) ON DELETE SET NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    placed_by TEXT NOT NULL CHECK (placed_by IN ('link', 'clone', 'copy', 'transfer', 'move', 'in_place', 'converted')),
    items TEXT NOT NULL CHECK (json_valid(items)),
    cells TEXT NOT NULL CHECK (json_valid(cells)),
    replaced TEXT NOT NULL CHECK (json_valid(replaced)),
    at INTEGER NOT NULL,
    quality TEXT CHECK (quality IN (
        'sdtv', 'dvd', 'webrip-480p', 'hdtv-720p', 'webrip-720p', 'webdl-720p', 'bluray-720p',
        'hdtv-1080p', 'webrip-1080p', 'webdl-1080p', 'bluray-1080p', 'remux-1080p',
        'hdtv-2160p', 'webrip-2160p', 'webdl-2160p', 'bluray-2160p', 'remux-2160p', 'cam'))
) STRICT;

INSERT INTO files_imported_next (id, file_id, import_id, grab_id, work_id, placed_by, items, cells, replaced, at, quality)
SELECT id, file_id, import_id, grab_id, work_id, placed_by, items, cells, replaced, at, quality FROM files_imported;

DROP TABLE files_imported;
ALTER TABLE files_imported_next RENAME TO files_imported;

CREATE INDEX files_imported_at ON files_imported (at);
CREATE INDEX files_imported_file ON files_imported (file_id, at);
CREATE INDEX files_imported_grab ON files_imported (grab_id);
CREATE INDEX files_imported_import ON files_imported (import_id);
CREATE INDEX files_imported_work ON files_imported (work_id, at);
