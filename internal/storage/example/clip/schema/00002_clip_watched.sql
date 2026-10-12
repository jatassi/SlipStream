-- +goose Up
ALTER TABLE clip ADD COLUMN watched_at INTEGER;
