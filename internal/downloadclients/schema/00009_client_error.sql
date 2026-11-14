-- +goose Up
-- S33 (Import pipeline) counts how long an external client has reported an
-- error on a Torrent. client_error_ms is the time counted, and
-- client_error_at the Node's clock at the last report with the error, NULL
-- while the last report had none. The Grab fails once the count passes
-- client_error_fail_seconds.
ALTER TABLE downloadclients_torrent ADD COLUMN client_error_ms INTEGER NOT NULL DEFAULT 0 CHECK (client_error_ms >= 0);
ALTER TABLE downloadclients_torrent ADD COLUMN client_error_at INTEGER;
ALTER TABLE downloadclients_settings ADD COLUMN client_error_fail_seconds INTEGER CHECK (client_error_fail_seconds > 0);
