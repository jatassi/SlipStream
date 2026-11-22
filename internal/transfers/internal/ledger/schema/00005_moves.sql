-- +goose Up
-- S48 (Moves) copies a library file from a root its source Node holds,
-- and a torrent's payload into a download folder a Download Client keeps
-- on its destination Node. A library file stores its Rel in path and its
-- root in source_root. A download folder stores its Node, the client
-- (NULL for the built-in one), the Content Type ('' for an adopted
-- torrent) and the info hash, and no folder, since its Node resolves the
-- folder. Two CHECKs of 00004 would refuse the new kinds, so both tables
-- are rebuilt with them replaced, keeping their ids and every column.
CREATE TABLE transfers_record_next (
    id BLOB PRIMARY KEY NOT NULL,
    purpose TEXT NOT NULL CHECK (purpose <> ''),
    job TEXT NOT NULL,
    source_node BLOB NOT NULL REFERENCES nodes_node (id),
    destination_kind TEXT NOT NULL CHECK (destination_kind IN ('root', 'backup', 'backups_folder', 'download_folder')),
    destination_root BLOB REFERENCES roots_root (id),
    -- The Backup destination's id, in backups.json.
    destination_backup BLOB CHECK (length(destination_backup) = 16),
    destination_node BLOB REFERENCES nodes_node (id),
    -- As the Admin typed it, with either separator.
    destination_folder TEXT CHECK (destination_folder <> ''),
    -- An external Download Client's id, NULL for the built-in client.
    destination_client BLOB CHECK (length(destination_client) = 16),
    destination_content_type TEXT,
    destination_info_hash BLOB CHECK (length(destination_info_hash) = 20),
    priority TEXT NOT NULL DEFAULT 'normal' CHECK (priority IN ('normal', 'low')),
    -- First come, first served: (created_at, id).
    created_at INTEGER NOT NULL,
    failure_code TEXT CHECK (failure_code IN ('source_gone', 'source_failed', 'damaged', 'source_written_off', 'rerouted', 'folder_failed')),
    -- The file the failure names, NULL when it names none.
    failure_position INTEGER CHECK (failure_position >= 0),
    -- The engine's text, or 'read_back' for damage found only on reading
    -- the staged file back.
    failure_detail TEXT NOT NULL DEFAULT '',
    UNIQUE (purpose, job),
    CHECK ((destination_kind = 'root') = (destination_root IS NOT NULL)),
    CHECK ((destination_kind = 'root') = (destination_node IS NULL)),
    CHECK ((destination_kind = 'backup' OR destination_kind = 'backups_folder') = (destination_folder IS NOT NULL)),
    CHECK ((destination_kind = 'backup') = (destination_backup IS NOT NULL)),
    CHECK ((destination_kind = 'download_folder') = (destination_info_hash IS NOT NULL)),
    CHECK ((destination_kind = 'download_folder') = (destination_content_type IS NOT NULL)),
    CHECK (destination_kind = 'download_folder' OR destination_client IS NULL),
    CHECK (failure_code IS NOT NULL OR (failure_position IS NULL AND failure_detail = ''))
) STRICT;

INSERT INTO transfers_record_next (id, purpose, job, source_node, destination_kind, destination_root, destination_backup, destination_node, destination_folder, priority, created_at, failure_code, failure_position, failure_detail)
SELECT id, purpose, job, source_node, destination_kind, destination_root, destination_backup, destination_node, destination_folder, priority, created_at, failure_code, failure_position, failure_detail FROM transfers_record;

CREATE TABLE transfers_file_next (
    transfer_id BLOB NOT NULL REFERENCES transfers_record (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    -- The key the purpose groups files by, '' for none.
    group_key TEXT NOT NULL,
    -- A Backup's file is its Backup name.
    name TEXT NOT NULL CHECK (name <> ''),
    source_kind TEXT NOT NULL CHECK (source_kind IN ('torrent', 'path', 'backup', 'library')),
    info_hash BLOB CHECK (length(info_hash) = 20),
    file_index INTEGER CHECK (file_index >= 0),
    -- Absolute on the source Node, or a library file's Rel in its root.
    path TEXT CHECK (path <> ''),
    -- The folder on the source Node that holds the Backup named name.
    source_folder TEXT CHECK (source_folder <> ''),
    -- The root a library file is in.
    source_root BLOB CHECK (length(source_root) = 16),
    size INTEGER NOT NULL CHECK (size >= 0),
    -- The destination's place the file was verified at, NULL until it was.
    -- The file counts verified only while this equals the destination's
    -- place, so a re-home unverifies every unplaced file. A Backup or
    -- download folder destination is always at place 1.
    verified_place INTEGER CHECK (verified_place > 0),
    placed INTEGER NOT NULL DEFAULT 0 CHECK (placed IN (0, 1)),
    dropped INTEGER NOT NULL DEFAULT 0 CHECK (dropped IN (0, 1)),
    PRIMARY KEY (transfer_id, position),
    CHECK ((source_kind = 'torrent') = (info_hash IS NOT NULL AND file_index IS NOT NULL)),
    CHECK ((source_kind = 'path' OR source_kind = 'library') = (path IS NOT NULL)),
    CHECK ((source_kind = 'library') = (source_root IS NOT NULL)),
    CHECK ((source_kind = 'backup') = (source_folder IS NOT NULL)),
    CHECK (NOT (placed = 1 AND dropped = 1))
) STRICT;

INSERT INTO transfers_file_next (transfer_id, position, group_key, name, source_kind, info_hash, file_index, path, source_folder, size, verified_place, placed, dropped)
SELECT transfer_id, position, group_key, name, source_kind, info_hash, file_index, path, source_folder, size, verified_place, placed, dropped FROM transfers_file;

DROP TABLE transfers_file;
DROP INDEX transfers_record_line;
DROP TABLE transfers_record;
ALTER TABLE transfers_record_next RENAME TO transfers_record;
ALTER TABLE transfers_file_next RENAME TO transfers_file;
CREATE INDEX transfers_record_line ON transfers_record (created_at, id);
