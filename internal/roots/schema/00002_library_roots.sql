-- +goose Up
-- Several roots per Content Type, one default each. The rebuild keeps every
-- id, so files_file, files_doomed and grabs_grab keep pointing at their rows,
-- and every existing root becomes its type's default. place counts the
-- root's homes: a re-home bumps it, so the Controller never compares paths.
CREATE TABLE roots_root_next (
    id BLOB PRIMARY KEY NOT NULL,
    content_type TEXT NOT NULL,
    node_id BLOB NOT NULL,
    path TEXT NOT NULL CHECK (path <> ''),
    is_default INTEGER NOT NULL CHECK (is_default IN (0, 1)),
    place INTEGER NOT NULL DEFAULT 1 CHECK (place > 0)
) STRICT;
INSERT INTO roots_root_next (id, content_type, node_id, path, is_default)
SELECT id, content_type, node_id, path, 1 FROM roots_root;
DROP TABLE roots_root;
ALTER TABLE roots_root_next RENAME TO roots_root;
CREATE UNIQUE INDEX roots_root_default ON roots_root (content_type) WHERE is_default = 1;
CREATE INDEX roots_root_node_id ON roots_root (node_id);

-- The root each Work lives in. No row is a rootless Work, legal while its
-- type has no root.
CREATE TABLE roots_work (
    work_id BLOB PRIMARY KEY NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    root_id BLOB NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE
) STRICT;
CREATE INDEX roots_work_root_id ON roots_work (root_id);
INSERT INTO roots_work (work_id, root_id)
SELECT w.id, r.id FROM works_work w JOIN roots_root r ON r.content_type = w.content_type;

CREATE TABLE roots_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    -- Seconds; NULL is the default.
    sample_every INTEGER CHECK (sample_every > 0)
) STRICT;
INSERT INTO roots_settings (id, sample_every) VALUES (1, NULL);

-- The place of a root at which its Node last found its marker, and the
-- volume it sat on. A row whose place differs from roots_root's is stale:
-- the root moved and its new place is unconfirmed.
CREATE TABLE roots_seen (
    root_id BLOB PRIMARY KEY NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE,
    place INTEGER NOT NULL CHECK (place > 0),
    volume_id TEXT NOT NULL
) STRICT;

-- Each Storage Node's volumes as it last reported them, kept while the Node
-- is offline and across a Controller restart.
CREATE TABLE roots_volume (
    node_id BLOB NOT NULL,
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    mount TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('local', 'removable', 'network')),
    total INTEGER NOT NULL CHECK (total >= 0),
    free INTEGER NOT NULL CHECK (free >= 0),
    sampled_at INTEGER NOT NULL,
    PRIMARY KEY (node_id, id)
) STRICT;
