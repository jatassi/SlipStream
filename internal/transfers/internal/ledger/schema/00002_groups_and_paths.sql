-- +goose Up
-- A file's group is any key its purpose groups by, a source may be a path
-- on the source Node, and a purpose that stops wanting an unplaced file
-- drops it: its receiver deletes the staged bytes, and the file counts as
-- done for its group and for the record.
CREATE TABLE transfers_file_next (
    transfer_id BLOB NOT NULL REFERENCES transfers_record (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    -- The key the purpose groups files by, '' for none.
    group_key TEXT NOT NULL,
    name TEXT NOT NULL CHECK (name <> ''),
    source_kind TEXT NOT NULL CHECK (source_kind IN ('torrent', 'path')),
    info_hash BLOB CHECK (length(info_hash) = 20),
    file_index INTEGER CHECK (file_index >= 0),
    -- Absolute on the source Node.
    path TEXT CHECK (path <> ''),
    size INTEGER NOT NULL CHECK (size >= 0),
    -- The destination's place the file was verified at, NULL until it was.
    -- The file counts verified only while this equals the destination's
    -- place, so a re-home unverifies every unplaced file.
    verified_place INTEGER CHECK (verified_place > 0),
    placed INTEGER NOT NULL DEFAULT 0 CHECK (placed IN (0, 1)),
    dropped INTEGER NOT NULL DEFAULT 0 CHECK (dropped IN (0, 1)),
    PRIMARY KEY (transfer_id, position),
    CHECK ((source_kind = 'torrent') = (info_hash IS NOT NULL AND file_index IS NOT NULL)),
    CHECK ((source_kind = 'path') = (path IS NOT NULL)),
    CHECK (NOT (placed = 1 AND dropped = 1))
) STRICT;

INSERT INTO transfers_file_next (transfer_id, position, group_key, name, source_kind, info_hash, file_index, size, verified_place, placed)
SELECT transfer_id, position, item, name, source_kind, info_hash, file_index, size, verified_place, placed FROM transfers_file;
DROP TABLE transfers_file;
ALTER TABLE transfers_file_next RENAME TO transfers_file;
