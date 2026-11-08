-- +goose Up
ALTER TABLE works_work ADD COLUMN monitor_new INTEGER NOT NULL DEFAULT 0 CHECK (monitor_new IN (0, 1));
ALTER TABLE works_work ADD COLUMN monitor_asides INTEGER NOT NULL DEFAULT 0 CHECK (monitor_asides IN (0, 1));
ALTER TABLE works_work ADD COLUMN level_folders INTEGER NOT NULL DEFAULT 1 CHECK (level_folders IN (0, 1));

-- monitor is a monitoring preset's id, which Go parses; NULL is the default.
CREATE TABLE works_add_default_next (
    content_type TEXT PRIMARY KEY NOT NULL,
    monitor TEXT,
    asides INTEGER CHECK (asides IN (0, 1)),
    level_folders INTEGER CHECK (level_folders IN (0, 1))
) STRICT;
INSERT INTO works_add_default_next (content_type, monitor)
SELECT content_type, CASE monitor WHEN 1 THEN 'all' ELSE 'none' END FROM works_add_default;
DROP TABLE works_add_default;
ALTER TABLE works_add_default_next RENAME TO works_add_default;
