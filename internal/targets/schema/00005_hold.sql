-- +goose Up
-- S34 (Automatic search, Backlog search and RSS sync). A Target's Hold: how
-- long an automatic grab waits after a Release's publish time for a better
-- one, in whole minutes up to 7 days. 0 grabs at once.
ALTER TABLE targets_target ADD COLUMN hold_seconds INTEGER NOT NULL DEFAULT 0
    CHECK (hold_seconds BETWEEN 0 AND 604800 AND hold_seconds % 60 = 0);
