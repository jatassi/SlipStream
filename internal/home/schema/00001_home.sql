-- +goose Up
-- S39 (Home): Home's one setting. NULL is the default.
CREATE TABLE home_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    -- "Rows on Home": 3 to 50, checked in Go. Not "rows", which SQLite
    -- reserves for window frames.
    rows_shown INTEGER CHECK (rows_shown > 0)
) STRICT;

INSERT INTO home_settings (id) VALUES (1);
