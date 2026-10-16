-- +goose Up
ALTER TABLE vault_login ADD COLUMN tracker_cookie TEXT;
