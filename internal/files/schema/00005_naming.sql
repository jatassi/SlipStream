-- +goose Up
-- The downloaded file's name, with its extension, before SlipStream
-- renamed it, or a found file's own name. It never changes.
ALTER TABLE files_file ADD COLUMN source_name TEXT CHECK (source_name <> '');
UPDATE files_file SET source_name = release_name WHERE grab_id IS NULL;

-- origin is where the bytes sat when the move was queued: where a move
-- whose target is taken at disk time returns them, and what a finished
-- rename records. ring is shared by the files of one cycle of renames,
-- which pass through their holding paths; NULL moves straight. Moves
-- queued before this migration keep their two hops.
ALTER TABLE files_move ADD COLUMN origin TEXT NOT NULL DEFAULT '';
UPDATE files_move SET origin = sits;
ALTER TABLE files_move ADD COLUMN ring BLOB;
UPDATE files_move SET ring = file_id;

-- Every finished rename, for S35 (Media Servers).
CREATE TABLE files_renamed (
    id INTEGER PRIMARY KEY,
    file_id BLOB NOT NULL,
    root_id BLOB NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE,
    from_path TEXT NOT NULL CHECK (from_path <> ''),
    to_path TEXT NOT NULL CHECK (to_path <> ''),
    renamed_at INTEGER NOT NULL
) STRICT;

-- A folder a finished move left, which the disk queue removes with its
-- empty parents unless a queued move still needs it.
CREATE TABLE files_prune (
    root_id BLOB NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE,
    folder TEXT NOT NULL CHECK (folder <> ''),
    PRIMARY KEY (root_id, folder)
) STRICT;
