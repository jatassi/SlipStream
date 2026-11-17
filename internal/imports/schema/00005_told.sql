-- +goose Up
-- When a placed file's placement went into an Imported or Upgraded
-- notification, for S37 (Notifications). NULL until then; root Imports
-- leave it NULL.
ALTER TABLE imports_file ADD COLUMN told_at INTEGER;
