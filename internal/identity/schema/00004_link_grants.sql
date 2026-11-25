-- +goose Up
-- S43 (Member access to Media Servers): what a Signup link offers besides
-- the Account, as identity.Grants JSON.
ALTER TABLE identity_link ADD COLUMN grants TEXT
    CHECK (grants IS NULL OR (kind = 'signup' AND json_valid(grants) AND json_type(grants) = 'object'));
