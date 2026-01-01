-- +goose Up
-- +goose StatementBegin

-- Add season metadata columns
ALTER TABLE seasons ADD COLUMN overview TEXT;
ALTER TABLE seasons ADD COLUMN poster_url TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite doesn't support DROP COLUMN directly, so we recreate the table
CREATE TABLE seasons_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    monitored INTEGER NOT NULL DEFAULT 1,
    UNIQUE(series_id, season_number)
);

INSERT INTO seasons_new (id, series_id, season_number, monitored)
SELECT id, series_id, season_number, monitored FROM seasons;

DROP TABLE seasons;
ALTER TABLE seasons_new RENAME TO seasons;

CREATE INDEX IF NOT EXISTS idx_seasons_series_id ON seasons(series_id);

-- +goose StatementEnd
