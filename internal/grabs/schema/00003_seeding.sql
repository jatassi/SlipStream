-- +goose Up
-- seed_dropped_at is when a Grab's torrent left its Node's Downloading
-- intent: the Admin wrote off what the Node held. The torrent seeds no
-- more, and its files stay.
ALTER TABLE grabs_grab ADD COLUMN seed_dropped_at INTEGER;
