-- +goose Up
ALTER TABLE movies ADD COLUMN theatrical_release_date DATETIME;
CREATE INDEX idx_movies_theatrical_release_date ON movies(theatrical_release_date);

-- +goose Down
DROP INDEX IF EXISTS idx_movies_theatrical_release_date;
ALTER TABLE movies DROP COLUMN theatrical_release_date;
