-- +goose Up
-- S36 (Playback report): a play keeps every item that holds a version its
-- player could choose, since no one item names them on Jellyfin or Emby.
ALTER TABLE mediaservers_play ADD COLUMN item_keys TEXT NOT NULL DEFAULT '[]'
    CHECK (json_valid(item_keys) AND json_type(item_keys) = 'array');
UPDATE mediaservers_play SET item_keys = json_array(item_key) WHERE item_key <> '';
ALTER TABLE mediaservers_play DROP COLUMN item_key;
