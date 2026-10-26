-- +goose Up
CREATE TABLE works_add_default (
    content_type TEXT PRIMARY KEY NOT NULL,
    monitor INTEGER NOT NULL CHECK (monitor IN (0, 1))
) STRICT;
