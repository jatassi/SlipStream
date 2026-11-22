-- +goose Up
-- One row of nullable overrides. A NULL is its default, kept in Go. Every
-- other fact about Backups is in backups.json, outside library.db, so a
-- restore keeps every Backup taken since.
CREATE TABLE backups_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    keep_scheduled_days INTEGER CHECK (keep_scheduled_days BETWEEN 1 AND 3650),
    keep_each INTEGER CHECK (keep_each BETWEEN 1 AND 100),
    warn_after INTEGER CHECK (warn_after BETWEEN 1 AND 30),
    copy_wait_limit_hours INTEGER CHECK (copy_wait_limit_hours BETWEEN 1 AND 168),
    check_copies_days INTEGER CHECK (check_copies_days BETWEEN 1 AND 90)
) STRICT;
INSERT INTO backups_settings (id) VALUES (1);
