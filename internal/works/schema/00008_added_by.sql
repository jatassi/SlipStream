-- +goose Up
ALTER TABLE works_work ADD COLUMN added_by TEXT NOT NULL DEFAULT 'admin' CHECK (added_by IN ('admin', 'scan'));
