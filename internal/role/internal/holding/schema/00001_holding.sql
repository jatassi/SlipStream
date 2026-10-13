-- +goose Up
CREATE TABLE role_epoch (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    epoch BLOB NOT NULL,
    last_seq INTEGER NOT NULL
) STRICT;

CREATE TABLE role_intent (
    scope TEXT PRIMARY KEY,
    version BLOB NOT NULL,
    payload BLOB NOT NULL
) STRICT;

CREATE TABLE role_outbox (
    seq INTEGER PRIMARY KEY,
    scope TEXT NOT NULL,
    payload BLOB NOT NULL
) STRICT;
