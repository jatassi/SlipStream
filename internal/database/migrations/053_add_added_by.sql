-- +goose Up
ALTER TABLE movies ADD COLUMN added_by INTEGER REFERENCES portal_users(id) ON DELETE SET NULL;
ALTER TABLE series ADD COLUMN added_by INTEGER REFERENCES portal_users(id) ON DELETE SET NULL;

-- Backfill from requests table (requester = user who submitted the request)
UPDATE movies SET added_by = (
  SELECT r.user_id FROM requests r
  WHERE r.media_id = movies.id AND r.media_type = 'movie'
  ORDER BY r.created_at ASC LIMIT 1
) WHERE added_by IS NULL;

UPDATE series SET added_by = (
  SELECT r.user_id FROM requests r
  WHERE r.media_id = series.id AND r.media_type IN ('series', 'season', 'episode')
  ORDER BY r.created_at ASC LIMIT 1
) WHERE added_by IS NULL;

-- Default remaining (non-request) items to admin user
UPDATE movies SET added_by = (
  SELECT id FROM portal_users WHERE is_admin = 1 LIMIT 1
) WHERE added_by IS NULL;

UPDATE series SET added_by = (
  SELECT id FROM portal_users WHERE is_admin = 1 LIMIT 1
) WHERE added_by IS NULL;

-- +goose Down
