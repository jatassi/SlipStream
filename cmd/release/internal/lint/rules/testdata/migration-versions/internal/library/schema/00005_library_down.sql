-- +goose Up
ALTER TABLE library ADD COLUMN note TEXT;

-- +goose Down
ALTER TABLE library DROP COLUMN note;
