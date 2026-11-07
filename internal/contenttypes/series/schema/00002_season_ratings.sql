-- +goose Up
CREATE TABLE series_season_rating (
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    aired_season INTEGER NOT NULL CHECK (aired_season > 0),
    read_at INTEGER NOT NULL,
    PRIMARY KEY (work_id, aired_season)
) STRICT;
