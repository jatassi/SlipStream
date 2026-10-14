-- +goose Up
ALTER TABLE works_item ADD COLUMN monitored INTEGER NOT NULL DEFAULT 0 CHECK (monitored IN (0, 1));
