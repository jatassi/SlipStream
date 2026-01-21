-- +goose Up
ALTER TABLE requests ADD COLUMN requested_seasons TEXT;

-- +goose Down
ALTER TABLE requests DROP COLUMN requested_seasons;
