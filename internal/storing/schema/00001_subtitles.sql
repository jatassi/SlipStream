-- +goose Up
-- One Speech Map this Node keeps under its data folder, as
-- speechmaps/<file id>.ssmp, and what it was built from: the file's
-- identity and size, the format version and the audio stream.
CREATE TABLE subtitles_speechmap (
    file_id BLOB PRIMARY KEY NOT NULL,
    root_id BLOB NOT NULL,
    identity BLOB NOT NULL,
    version INTEGER NOT NULL,
    stream INTEGER NOT NULL,
    frames INTEGER NOT NULL CHECK (frames >= 0),
    density REAL NOT NULL CHECK (density BETWEEN 0 AND 1),
    size INTEGER NOT NULL,
    built_at INTEGER NOT NULL
) STRICT;

-- Every hash SlipStream wrote as a Sidecar under a root (D27). path is the
-- last name written, a hint only: carry renames companions without
-- telling the ledger. Append-only: RemoveSidecar keeps the row, because two
-- files of one root can hold byte-identical Sidecars (a sibling copied at
-- offset zero) and removing one must not make the other read as found
-- after a restore.
CREATE TABLE subtitles_ledger (
    root_id BLOB NOT NULL,
    sha256 BLOB NOT NULL CHECK (length(sha256) = 32),
    path TEXT NOT NULL,
    written_at INTEGER NOT NULL,
    PRIMARY KEY (root_id, sha256)
) STRICT;
