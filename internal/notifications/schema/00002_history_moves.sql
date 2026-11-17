-- +goose Up

-- 1. Every event keeps its id, so deliveries, shown rows and told pairing stay valid.
INSERT INTO history_event (id, kind, at, content_type, work_id, grab_id, import_id, subject)
SELECT id, kind, at,
       json_extract(subject, '$.work.type'),
       unhex(replace(json_extract(subject, '$.work.id'), '-', '')),
       NULL, NULL, subject
FROM notifications_event
ORDER BY id;

-- A single-Work delete holds its Work at $.types[0].work.
UPDATE history_event
SET content_type = json_extract(subject, '$.types[0].work.type'),
    work_id = unhex(replace(json_extract(subject, '$.types[0].work.id'), '-', ''))
WHERE kind = 'works_deleted' AND json_extract(subject, '$.types[0].work.id') IS NOT NULL;

INSERT INTO history_work (event_id, work_id)
SELECT id, work_id FROM history_event WHERE work_id IS NOT NULL;

-- Ids are never reused across the move. sqlite_sequence has no unique key,
-- so insert the row when it's missing, then raise it.
INSERT INTO sqlite_sequence (name, seq)
SELECT 'history_event', 0
WHERE NOT EXISTS (SELECT 1 FROM sqlite_sequence WHERE name = 'history_event');
UPDATE sqlite_sequence
SET seq = MAX(seq, COALESCE((SELECT seq FROM sqlite_sequence WHERE name = 'notifications_event'), 0))
WHERE name = 'history_event';

-- 2. Health items, as they are. A Worse inside the wait already replaced its item in place.
INSERT INTO history_item (event_id, key, level, subject, text, since_at, cleared_at)
SELECT event_id, key, level, subject, text, since_at, cleared_at FROM notifications_item;

-- 3. The queue: every unreleased event, with the Health keys it still carries.
CREATE TABLE notifications_pending (
    event_id INTEGER PRIMARY KEY REFERENCES history_event (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind <> ''),
    at INTEGER NOT NULL,
    due_at INTEGER NOT NULL,
    items TEXT CHECK (items IS NULL OR (json_valid(items) AND json_type(items) = 'array')),
    CHECK (due_at >= at)
) STRICT;

CREATE INDEX notifications_pending_due ON notifications_pending (due_at, event_id);

INSERT INTO notifications_pending (event_id, kind, at, due_at, items)
SELECT e.id, e.kind, e.at, e.due_at,
       NULLIF((SELECT json_group_array(i.key) FROM notifications_item i WHERE i.event_id = e.id), '[]')
FROM notifications_event e
WHERE e.released_at IS NULL;

-- 4. Which newer entry's item a pending or released Health key renders from:
--    a Worse inside the wait leaves the key on the earlier entry, keeping its due time.
CREATE TABLE notifications_worse (
    event_id INTEGER NOT NULL REFERENCES history_event (id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    worse_id INTEGER NOT NULL REFERENCES history_event (id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, key),
    CHECK (worse_id > event_id)
) STRICT, WITHOUT ROWID;

CREATE INDEX notifications_worse_by ON notifications_worse (worse_id);

-- 5. Deliveries against history_event, with their event's kind and time.
CREATE TABLE notifications_delivery_next (
    event_id INTEGER NOT NULL REFERENCES history_event (id) ON DELETE CASCADE,
    connection_id BLOB NOT NULL REFERENCES notifications_connection (id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    at INTEGER NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('due', 'given_up')),
    attempts INTEGER NOT NULL CHECK (attempts >= 0),
    items TEXT CHECK (items IS NULL OR (json_valid(items) AND json_type(items) = 'array')),
    accepted TEXT NOT NULL CHECK (json_valid(accepted) AND json_type(accepted) = 'array'),
    ended_at INTEGER,
    reason TEXT CHECK (reason IS NULL OR (json_valid(reason) AND json_type(reason) = 'object')),
    response TEXT,
    PRIMARY KEY (event_id, connection_id),
    CHECK ((state = 'due') = (ended_at IS NULL)),
    CHECK ((state = 'given_up') = (reason IS NOT NULL)),
    CHECK (state = 'given_up' OR response IS NULL)
) STRICT, WITHOUT ROWID;

INSERT INTO notifications_delivery_next
SELECT d.event_id, d.connection_id, e.kind, e.at, d.state, d.attempts, d.items, d.accepted, d.ended_at, d.reason, d.response
FROM notifications_delivery d JOIN notifications_event e ON e.id = d.event_id;

DROP TABLE notifications_delivery;
ALTER TABLE notifications_delivery_next RENAME TO notifications_delivery;
CREATE INDEX notifications_delivery_line ON notifications_delivery (connection_id, state, event_id);
CREATE INDEX notifications_delivery_due ON notifications_delivery (event_id) WHERE state = 'due';
CREATE INDEX notifications_delivery_due_at ON notifications_delivery (at) WHERE state = 'due';

-- 6. The bell's list against history_event. seq and its sequence are kept,
--    so read cursors stay right.
CREATE TABLE notifications_shown_next (
    seq INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL UNIQUE REFERENCES history_event (id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    at INTEGER NOT NULL,
    shown_at INTEGER NOT NULL,
    items TEXT CHECK (items IS NULL OR (json_valid(items) AND json_type(items) = 'array'))
) STRICT;

INSERT INTO notifications_shown_next (seq, event_id, kind, at, shown_at, items)
SELECT s.seq, s.event_id, e.kind, e.at, s.shown_at, s.items
FROM notifications_shown s JOIN notifications_event e ON e.id = s.event_id;

INSERT INTO sqlite_sequence (name, seq)
SELECT 'notifications_shown_next', 0
WHERE NOT EXISTS (SELECT 1 FROM sqlite_sequence WHERE name = 'notifications_shown_next');
UPDATE sqlite_sequence
SET seq = MAX(seq, COALESCE((SELECT seq FROM sqlite_sequence WHERE name = 'notifications_shown'), 0))
WHERE name = 'notifications_shown_next';

DROP TABLE notifications_shown;
ALTER TABLE notifications_shown_next RENAME TO notifications_shown;   -- renames its sqlite_sequence row
CREATE INDEX notifications_shown_at ON notifications_shown (shown_at);

-- 7. Settings without Keep notifications for. The value isn't carried: its range and default differ.
CREATE TABLE notifications_settings_next (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    health_wait_seconds INTEGER CHECK (health_wait_seconds >= 0),
    give_up_seconds INTEGER CHECK (give_up_seconds > 0),
    warn_seconds INTEGER CHECK (warn_seconds > 0)
) STRICT;
INSERT INTO notifications_settings_next (id, health_wait_seconds, give_up_seconds, warn_seconds)
SELECT id, health_wait_seconds, give_up_seconds, warn_seconds FROM notifications_settings;
DROP TABLE notifications_settings;
ALTER TABLE notifications_settings_next RENAME TO notifications_settings;

-- 8. The old log.
DROP TABLE notifications_item;
DROP TABLE notifications_event;
