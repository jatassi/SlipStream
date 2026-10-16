-- +goose Up
CREATE TABLE mail_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    host TEXT NOT NULL,
    port INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
    security TEXT NOT NULL CHECK (security IN ('none', 'starttls', 'tls')),
    username TEXT NOT NULL,
    password_sealed BLOB,
    from_address TEXT NOT NULL,
    from_name TEXT NOT NULL
) STRICT;

INSERT INTO mail_settings VALUES (1, '', 587, 'starttls', '', NULL, '', 'SlipStream');
