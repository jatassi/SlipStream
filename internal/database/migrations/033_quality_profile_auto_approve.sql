-- +goose Up
-- Add allow_auto_approve column to quality_profiles
ALTER TABLE quality_profiles ADD COLUMN allow_auto_approve INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE quality_profiles DROP COLUMN allow_auto_approve;
