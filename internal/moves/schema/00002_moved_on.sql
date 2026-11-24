-- +goose Up
-- A torrent unit past its switch, by the Torrent it moved, so a unit's
-- name follows its Grabs through the Moves that took the torrent later.
CREATE INDEX moves_unit_switched_torrent ON moves_unit (torrent_id)
    WHERE torrent_id IS NOT NULL AND phase IN ('cleanup', 'done');
