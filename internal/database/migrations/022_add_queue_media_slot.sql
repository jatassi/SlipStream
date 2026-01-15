-- +goose Up
-- Req 16.2.1, 16.2.3: Per-episode slot tracking for season packs
-- Each episode in a season pack can have its own target slot assignment
ALTER TABLE queue_media ADD COLUMN target_slot_id INTEGER REFERENCES version_slots(id);

-- +goose Down
ALTER TABLE queue_media DROP COLUMN target_slot_id;
