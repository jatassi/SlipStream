-- +goose Up
-- S36 (Playback report): a page of one player's plays seeks this index.
CREATE INDEX mediaservers_play_player ON mediaservers_play (server_id, player_key, started_at, id);
