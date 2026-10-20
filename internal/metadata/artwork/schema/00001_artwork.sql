-- +goose Up
-- One file under the Controller's artwork folder per distinct image, named
-- by the SHA-256 of its bytes.
CREATE TABLE artwork_blob (
    hash TEXT PRIMARY KEY NOT NULL CHECK (length(hash) = 64),
    bytes INTEGER NOT NULL CHECK (bytes > 0),
    type TEXT NOT NULL CHECK (type IN ('image/png', 'image/jpeg', 'image/webp', 'image/svg+xml'))
) STRICT;

-- One image at one size: held as a blob, or missing until retry_at after a
-- fetch failed.
CREATE TABLE artwork_entry (
    source TEXT NOT NULL,
    shape TEXT NOT NULL,
    path TEXT NOT NULL,
    size TEXT NOT NULL CHECK (size IN ('small', 'medium', 'large', 'original')),
    hash TEXT REFERENCES artwork_blob (hash),
    retry_at INTEGER,
    served_at INTEGER NOT NULL,
    CHECK ((hash IS NULL) = (retry_at IS NOT NULL)),
    PRIMARY KEY (source, shape, path, size)
) STRICT, WITHOUT ROWID;

CREATE INDEX artwork_entry_served ON artwork_entry (served_at) WHERE hash IS NOT NULL;
CREATE INDEX artwork_entry_hash ON artwork_entry (hash);
