-- +goose Up
-- S24 (Naming and renaming): a Target name prints in file names through
-- {Target}, and NTFS and APFS compare file names without case, so Target
-- names are unique case-insensitively as Custom Format names are.
-- SQLite's lower() folds only ASCII, so Go fills name_folded with
-- cases.Fold(). It is NULL only until the catch-up at start fills it, and
-- a unique index already lets NULLs repeat.
ALTER TABLE targets_target ADD COLUMN name_folded TEXT;

CREATE UNIQUE INDEX targets_target_name_folded ON targets_target (name_folded);
