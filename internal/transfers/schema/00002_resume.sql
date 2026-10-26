-- +goose Up
-- transfers_staging is the folder this Node stages each Transfer it
-- receives in, so it can delete the folder once the Transfer leaves its
-- intent's keep list.
CREATE TABLE transfers_staging (
    transfer_id BLOB PRIMARY KEY NOT NULL,
    folder TEXT NOT NULL CHECK (folder <> '')
) STRICT;

-- transfers_resume is a receiver's verified chunks per staged file, kept
-- against the source identity it saw and the destination generation it
-- staged at: a changed identity or place restarts the file.
CREATE TABLE transfers_resume (
    transfer_id BLOB NOT NULL REFERENCES transfers_staging (transfer_id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    place INTEGER NOT NULL,
    size INTEGER NOT NULL CHECK (size >= 0),
    mtime_ns INTEGER NOT NULL,
    file_id BLOB,
    -- verified holds one bit per chunk, chunk i at bit i % 8 of byte i / 8.
    verified BLOB NOT NULL,
    -- done: synced, closed and FileVerified queued in the same write.
    done INTEGER NOT NULL DEFAULT 0 CHECK (done IN (0, 1)),
    PRIMARY KEY (transfer_id, position, place)
) STRICT;
