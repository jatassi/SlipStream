-- +goose Up
-- Pending imports: per root, each Work folder, or file loose at the root's
-- top, whose found files a library scan couldn't place with confidence, and
-- each folder a Leave files delete left on disk. hint and suggestion are
-- JSON, suggestion NULL for none.
CREATE TABLE imports_pending (
    id BLOB PRIMARY KEY NOT NULL,
    root_id BLOB NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE,
    folder TEXT NOT NULL CHECK (folder <> ''),
    content_type TEXT NOT NULL,
    hint TEXT NOT NULL,
    suggestion TEXT,
    found_at INTEGER NOT NULL,
    ignored_at INTEGER,
    UNIQUE (root_id, folder)
) STRICT;

-- An entry's files, root-relative. probe and quality are NULL for a file
-- ffprobe can't read. items is the JSON list of Item ids its path names,
-- reason the JSON scanning.Reason it waits for.
CREATE TABLE imports_pending_file (
    pending_id BLOB NOT NULL REFERENCES imports_pending (id) ON DELETE CASCADE,
    path TEXT NOT NULL CHECK (path <> ''),
    size INTEGER NOT NULL CHECK (size >= 0),
    modified_at INTEGER,
    node_file_id TEXT CHECK (node_file_id <> ''),
    node_volume_id TEXT,
    probe TEXT,
    quality TEXT,
    cut TEXT NOT NULL,
    items TEXT NOT NULL,
    reason TEXT NOT NULL,
    PRIMARY KEY (pending_id, path)
) STRICT, WITHOUT ROWID;

-- Each root's last comparison, at the place it compared: what its scan line
-- and the Library Health category read. compared_at is NULL until one
-- finished there. stopped_gone and stopped_of are set while the guard
-- stopped it. confirmed is the stopped count Mark them gone confirmed,
-- until the next comparison consumes it.
CREATE TABLE imports_root (
    root_id BLOB PRIMARY KEY NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE,
    place INTEGER NOT NULL CHECK (place > 0),
    compared_at INTEGER,
    found INTEGER NOT NULL DEFAULT 0 CHECK (found >= 0),
    changed INTEGER NOT NULL DEFAULT 0 CHECK (changed >= 0),
    gone INTEGER NOT NULL DEFAULT 0 CHECK (gone >= 0),
    returned INTEGER NOT NULL DEFAULT 0 CHECK (returned >= 0),
    added INTEGER NOT NULL DEFAULT 0 CHECK (added >= 0),
    stopped_gone INTEGER,
    stopped_of INTEGER,
    confirmed INTEGER,
    CHECK ((stopped_gone IS NULL) = (stopped_of IS NULL))
) STRICT;

CREATE TABLE imports_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    stop_share INTEGER,
    probes INTEGER
) STRICT;

INSERT INTO imports_settings (id) VALUES (1);

-- Roots already seen ready at their place wait for the nightly scan
-- instead of all scanning at the first start after the upgrade.
INSERT INTO imports_root (root_id, place) SELECT root_id, place FROM roots_seen;
