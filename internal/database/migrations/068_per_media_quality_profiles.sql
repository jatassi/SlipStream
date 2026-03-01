-- +goose Up
-- Split quality_profile_id into per-media-type columns for portal_users
ALTER TABLE portal_users ADD COLUMN movie_quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL;
ALTER TABLE portal_users ADD COLUMN tv_quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL;
UPDATE portal_users SET movie_quality_profile_id = quality_profile_id, tv_quality_profile_id = quality_profile_id;
ALTER TABLE portal_users DROP COLUMN quality_profile_id;

-- Split quality_profile_id into per-media-type columns for portal_invitations
ALTER TABLE portal_invitations ADD COLUMN movie_quality_profile_id INTEGER;
ALTER TABLE portal_invitations ADD COLUMN tv_quality_profile_id INTEGER;
UPDATE portal_invitations SET movie_quality_profile_id = quality_profile_id, tv_quality_profile_id = quality_profile_id;
ALTER TABLE portal_invitations DROP COLUMN quality_profile_id;

-- +goose Down
ALTER TABLE portal_users ADD COLUMN quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL;
UPDATE portal_users SET quality_profile_id = movie_quality_profile_id;
ALTER TABLE portal_users DROP COLUMN movie_quality_profile_id;
ALTER TABLE portal_users DROP COLUMN tv_quality_profile_id;

ALTER TABLE portal_invitations ADD COLUMN quality_profile_id INTEGER;
UPDATE portal_invitations SET quality_profile_id = movie_quality_profile_id;
ALTER TABLE portal_invitations DROP COLUMN movie_quality_profile_id;
ALTER TABLE portal_invitations DROP COLUMN tv_quality_profile_id;
