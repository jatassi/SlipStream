-- +goose Up
-- S34 (Automatic search, Backlog search and RSS sync). What an add
-- searches, per Content Type. NULL is the type's default.
ALTER TABLE works_add_default ADD COLUMN search TEXT
    CHECK (search IS NULL OR search IN ('monitored', 'first_level', 'latest_level', 'first_item', 'none'));
