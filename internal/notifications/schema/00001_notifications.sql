-- +goose Up

-- A notification connection's config. Only the API writes it.
CREATE TABLE notifications_connection (
    id BLOB PRIMARY KEY NOT NULL,
    name TEXT NOT NULL UNIQUE CHECK (length(name) BETWEEN 1 AND 100),
    provider TEXT NOT NULL,                       -- checked in Go against providers.Table
    is_on INTEGER NOT NULL CHECK (is_on IN (0, 1)),
    on_at INTEGER NOT NULL,                       -- when it was last turned on, or created on
    events TEXT NOT NULL CHECK (json_valid(events) AND json_type(events) = 'array'),
    warnings INTEGER NOT NULL CHECK (warnings IN (0, 1)),
    posters INTEGER NOT NULL CHECK (posters IN (0, 1)),
    node_id BLOB REFERENCES nodes_node (id),      -- Custom Script only
    settings TEXT NOT NULL CHECK (json_valid(settings) AND json_type(settings) = 'object'),
    secrets_sealed BLOB,
    revision INTEGER NOT NULL CHECK (revision >= 1),
    created_at INTEGER NOT NULL,
    CHECK ((provider = 'script') = (node_id IS NOT NULL))
) STRICT;

CREATE INDEX notifications_connection_node ON notifications_connection (node_id) WHERE node_id IS NOT NULL;

-- One connection's delivery state. Only the dispatcher writes it. No row:
-- ok, never sent, nothing given up.
CREATE TABLE notifications_line (
    connection_id BLOB PRIMARY KEY NOT NULL REFERENCES notifications_connection (id) ON DELETE CASCADE,
    state TEXT NOT NULL CHECK (state IN ('ok', 'failing', 'waiting', 'paused')),
    state_at INTEGER NOT NULL,
    -- Why it isn't ok, as facts (providers.Reason); worded when read.
    reason TEXT CHECK (reason IS NULL OR (json_valid(reason) AND json_type(reason) = 'object')),
    attempts INTEGER NOT NULL CHECK (attempts >= 0),
    next_at INTEGER NOT NULL,
    revision INTEGER NOT NULL CHECK (revision >= 1),
    last_sent_at INTEGER,
    -- Deliveries given up since the revision last moved.
    given_up INTEGER NOT NULL CHECK (given_up >= 0),
    CHECK ((state = 'ok') = (reason IS NULL)),
    CHECK (state = 'failing' OR attempts = 0)
) STRICT, WITHOUT ROWID;

-- What happened, recorded in the Write that made it true. Unreleased until
-- the dispatcher fans it out at due_at.
CREATE TABLE notifications_event (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL CHECK (kind <> ''),
    at INTEGER NOT NULL,
    due_at INTEGER NOT NULL,
    released_at INTEGER,
    subject TEXT NOT NULL CHECK (json_valid(subject)),
    CHECK (due_at >= at),
    CHECK (released_at IS NULL OR released_at >= at)
) STRICT;

CREATE INDEX notifications_event_at ON notifications_event (at);
CREATE INDEX notifications_event_unreleased ON notifications_event (due_at) WHERE released_at IS NULL;

-- The Health items of a New Health item or Health item cleared event.
CREATE TABLE notifications_item (
    event_id INTEGER NOT NULL REFERENCES notifications_event (id) ON DELETE CASCADE,
    key TEXT NOT NULL,                            -- "<category>.<key>"
    level TEXT NOT NULL CHECK (level IN ('warn', 'fail')),
    subject TEXT NOT NULL CHECK (subject <> ''),
    text TEXT NOT NULL,
    since_at INTEGER NOT NULL,
    cleared_at INTEGER,                           -- set on a cleared event's items
    PRIMARY KEY (event_id, key)
) STRICT, WITHOUT ROWID;

CREATE INDEX notifications_item_key ON notifications_item (key, event_id);

-- One event for one connection while it is due, or after it was given up.
-- A sent or dropped delivery is deleted in the Write that decides it. Only
-- the dispatcher writes it.
CREATE TABLE notifications_delivery (
    event_id INTEGER NOT NULL REFERENCES notifications_event (id) ON DELETE CASCADE,
    connection_id BLOB NOT NULL REFERENCES notifications_connection (id) ON DELETE CASCADE,
    state TEXT NOT NULL CHECK (state IN ('due', 'given_up')),
    attempts INTEGER NOT NULL CHECK (attempts >= 0),
    -- The Health keys it carries, fixed at release and narrowed at claim for
    -- a clear; NULL for every other kind.
    items TEXT CHECK (items IS NULL OR (json_valid(items) AND json_type(items) = 'array')),
    -- Email addresses that already accepted it.
    accepted TEXT NOT NULL CHECK (json_valid(accepted) AND json_type(accepted) = 'array'),
    ended_at INTEGER,
    -- Why it was given up, as facts (providers.Reason).
    reason TEXT CHECK (reason IS NULL OR (json_valid(reason) AND json_type(reason) = 'object')),
    -- The first KiB of the refusing response, secret values masked.
    response TEXT,
    PRIMARY KEY (event_id, connection_id),
    CHECK ((state = 'due') = (ended_at IS NULL)),
    CHECK ((state = 'given_up') = (reason IS NOT NULL)),
    CHECK (state = 'given_up' OR response IS NULL)
) STRICT, WITHOUT ROWID;

CREATE INDEX notifications_delivery_line ON notifications_delivery (connection_id, state, event_id);
CREATE INDEX notifications_delivery_due ON notifications_delivery (event_id) WHERE state = 'due';

-- The bell's own events and warnings. choices holds the Admin's choices
-- over each kind's BellByDefault, so a later kind needs no migration.
CREATE TABLE notifications_bell (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    choices TEXT NOT NULL CHECK (json_valid(choices) AND json_type(choices) = 'object'),
    warnings INTEGER NOT NULL CHECK (warnings IN (0, 1))
) STRICT;

INSERT INTO notifications_bell (id, choices, warnings) VALUES (1, '{}', 1);

-- The bell's list. seq is taken when a row is shown, never reused. items
-- is the Health keys the row showed, NULL for other kinds.
CREATE TABLE notifications_shown (
    seq INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL UNIQUE REFERENCES notifications_event (id) ON DELETE CASCADE,
    shown_at INTEGER NOT NULL,
    items TEXT CHECK (items IS NULL OR (json_valid(items) AND json_type(items) = 'array'))
) STRICT;

-- An Admin with no notifications_seen row starts at the newest row shown
-- at or before their Account was created.
CREATE INDEX notifications_shown_at ON notifications_shown (shown_at);

-- Each Admin's read cursor. No row: the newest seq shown at or before the
-- Account was created.
CREATE TABLE notifications_seen (
    account_id BLOB PRIMARY KEY NOT NULL REFERENCES identity_account (id) ON DELETE CASCADE,
    seq INTEGER NOT NULL CHECK (seq >= 0)
) STRICT, WITHOUT ROWID;

-- Health pairing: a target ('bell' or a connection id) was sent an item.
-- Written when a delivery carrying it is sent, or a bell row showing it is
-- shown; deleted when its clear is sent or shown. Outlives the event's
-- prune, so it keeps the words a clear needs.
CREATE TABLE notifications_told (
    target TEXT NOT NULL,
    key TEXT NOT NULL,
    subject TEXT NOT NULL,
    since_at INTEGER NOT NULL,
    told_at INTEGER NOT NULL,
    PRIMARY KEY (target, key)
) STRICT, WITHOUT ROWID;

CREATE INDEX notifications_told_key ON notifications_told (key);

CREATE TABLE notifications_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    health_wait_seconds INTEGER CHECK (health_wait_seconds >= 0),
    give_up_seconds INTEGER CHECK (give_up_seconds > 0),
    warn_seconds INTEGER CHECK (warn_seconds > 0),
    keep_seconds INTEGER CHECK (keep_seconds > 0)
) STRICT;

INSERT INTO notifications_settings (id) VALUES (1);
