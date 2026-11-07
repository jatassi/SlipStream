-- +goose Up
ALTER TABLE works_level ADD COLUMN aside INTEGER NOT NULL DEFAULT 0 CHECK (aside IN (0, 1));
ALTER TABLE works_item ADD COLUMN aside INTEGER NOT NULL DEFAULT 0 CHECK (aside IN (0, 1));

ALTER TABLE works_artwork ADD COLUMN level_id BLOB REFERENCES works_level (id) ON DELETE CASCADE
    CHECK (level_id IS NULL OR item_id IS NULL);
DROP INDEX works_artwork_one;
CREATE UNIQUE INDEX works_artwork_one ON works_artwork (work_id, ifnull(level_id, x''), ifnull(item_id, x''), kind);
CREATE INDEX works_artwork_level_id ON works_artwork (level_id);
