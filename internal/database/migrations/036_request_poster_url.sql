-- +goose Up
ALTER TABLE requests ADD COLUMN poster_url TEXT;

-- +goose Down
ALTER TABLE requests DROP COLUMN poster_url;
