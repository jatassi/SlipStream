-- +goose Up
-- What this Node knows of each torrent it holds, by info hash. It can be
-- lost, and the Controller then hands the torrent back.
CREATE TABLE downloading_torrent (
    info_hash BLOB PRIMARY KEY NOT NULL CHECK (length(info_hash) = 20),
    save_folder TEXT NOT NULL CHECK (save_folder <> ''),
    private INTEGER NOT NULL CHECK (private IN (0, 1)),
    -- resume is the engine's resume data, metainfo included; NULL until
    -- its first save.
    resume BLOB
) STRICT;
