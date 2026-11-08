-- +goose Up
-- What this Node knows of each torrent it holds in an external client, by
-- client and info hash. It can be lost: the Controller's copy of the
-- origin comes back with the torrent's fetch, and the rest starts again.
CREATE TABLE downloading_external (
    client_id BLOB NOT NULL CHECK (length(client_id) = 16),
    info_hash BLOB NOT NULL CHECK (length(info_hash) = 20),
    -- origin is NULL until the Node adds the torrent or sees it already in
    -- the client.
    origin TEXT CHECK (origin IN ('added', 'attached')),
    -- sent_at is the last add, until the client lists the hash.
    sent_at INTEGER,
    resends INTEGER NOT NULL DEFAULT 0 CHECK (resends >= 0),
    seen INTEGER NOT NULL DEFAULT 0 CHECK (seen IN (0, 1)),
    -- applied is the last wanted state the Node applied or saw in the
    -- client: the intent is compared with it, never with the last intent.
    applied TEXT CHECK (applied IN ('started', 'paused')),
    -- finished_at is the Node's first sight of the torrent finished, the
    -- Import delay's start when the client gives no completion time.
    finished_at INTEGER,
    completed INTEGER NOT NULL DEFAULT 0 CHECK (completed IN (0, 1)),
    PRIMARY KEY (client_id, info_hash)
) STRICT;
