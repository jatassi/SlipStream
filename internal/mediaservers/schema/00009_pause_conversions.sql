-- +goose Up
-- S44 (Conversion): a server that Runs on a Node pauses that Node's
-- Conversions while it transcodes video, unless the Admin turns it off.
ALTER TABLE mediaservers_server ADD COLUMN pauses_conversions INTEGER NOT NULL DEFAULT 1 CHECK (pauses_conversions IN (0, 1));
