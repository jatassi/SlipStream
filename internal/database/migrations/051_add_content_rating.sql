-- +goose Up
ALTER TABLE movies ADD COLUMN content_rating TEXT;

-- +goose Down
