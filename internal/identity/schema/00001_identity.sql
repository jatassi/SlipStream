-- +goose Up
CREATE TABLE identity_account (
    id BLOB PRIMARY KEY NOT NULL,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    admin INTEGER NOT NULL CHECK (admin IN (0, 1)),
    pin_hash TEXT NOT NULL,
    created_at INTEGER NOT NULL
) STRICT;

CREATE TABLE identity_session (
    id BLOB PRIMARY KEY NOT NULL,
    token_hash BLOB NOT NULL UNIQUE,
    account_id BLOB NOT NULL REFERENCES identity_account (id) ON DELETE CASCADE,
    address TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL
) STRICT;

CREATE INDEX identity_session_account_id ON identity_session (account_id);

CREATE TABLE identity_api_key (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    token_hash BLOB NOT NULL,
    created_at INTEGER NOT NULL
) STRICT;

CREATE TABLE identity_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    session_lifetime_days INTEGER NOT NULL CHECK (session_lifetime_days > 0)
) STRICT;

INSERT INTO identity_settings (id, session_lifetime_days) VALUES (1, 30);
