-- +goose Up

-- What happened to the library, recorded in the Write that made it true.
-- at is when it was captured. A kind's group is its catalog row's, so no
-- column holds it.
CREATE TABLE history_event (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL CHECK (kind <> ''),
    at INTEGER NOT NULL,
    content_type TEXT CHECK (content_type IS NULL OR content_type <> ''),   -- NULL for Health
    work_id BLOB,                                 -- the one Work it is about; NULL for none or many
    grab_id BLOB,                                 -- no FK: the API checks it still exists
    import_id BLOB,                               -- no FK, as grab_id
    subject TEXT NOT NULL CHECK (json_valid(subject))
) STRICT;

CREATE INDEX history_event_at ON history_event (at);
CREATE INDEX history_event_kind ON history_event (kind, id);
CREATE INDEX history_event_type ON history_event (content_type, kind, id) WHERE content_type IS NOT NULL;

-- Every Work an entry is about. No FK to works_work, so a deleted Work
-- keeps its entries.
CREATE TABLE history_work (
    event_id INTEGER NOT NULL REFERENCES history_event (id) ON DELETE CASCADE,
    work_id BLOB NOT NULL,
    PRIMARY KEY (event_id, work_id)
) STRICT, WITHOUT ROWID;

CREATE INDEX history_work_work ON history_work (work_id, event_id);

-- The Health items of a New Health item or Health item cleared entry.
-- since_at and cleared_at are the transition's own times, which its words show.
CREATE TABLE history_item (
    event_id INTEGER NOT NULL REFERENCES history_event (id) ON DELETE CASCADE,
    key TEXT NOT NULL,                            -- "<category>.<key>"
    level TEXT NOT NULL CHECK (level IN ('warn', 'fail')),
    subject TEXT NOT NULL CHECK (subject <> ''),
    text TEXT NOT NULL,
    since_at INTEGER NOT NULL,
    cleared_at INTEGER,                           -- set on a cleared entry's items
    PRIMARY KEY (event_id, key)
) STRICT, WITHOUT ROWID;

CREATE INDEX history_item_key ON history_item (key, event_id);

CREATE TABLE history_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    -- "Keep history for"; NULL is the default. 30 to 1,095 days, checked in Go.
    keep_seconds INTEGER CHECK (keep_seconds > 0)
) STRICT;

INSERT INTO history_settings (id) VALUES (1);

-- The last Clear: every entry at or below through is cleared. History reads
-- above it, and sweeps delete at or below it.
CREATE TABLE history_cleared (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    through INTEGER NOT NULL CHECK (through >= 0),
    at INTEGER
) STRICT;

INSERT INTO history_cleared (id, through) VALUES (1, 0);
