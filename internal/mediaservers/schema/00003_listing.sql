-- +goose Up
-- One library's paged listing for the path index. started_at and cursor
-- are set while a listing runs, so one resumes at its page after a stop.
-- listed_at is the last listing's end: the next starts Timings.IndexAge
-- later.
CREATE TABLE mediaservers_listing (
    server_id BLOB NOT NULL REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    library_key TEXT NOT NULL CHECK (library_key <> ''),
    started_at INTEGER,
    cursor TEXT NOT NULL DEFAULT '',
    listed_at INTEGER,
    PRIMARY KEY (server_id, library_key),
    CHECK (started_at IS NOT NULL OR cursor = '')
) STRICT;
