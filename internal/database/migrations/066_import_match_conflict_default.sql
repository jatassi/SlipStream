-- +goose Up
-- Change default match conflict behavior from 'fail' to 'trust_queue'.
-- The queue mapping is authoritative (created by the grab system for a specific media item),
-- so it should be trusted over best-effort filename parsing when they disagree.
UPDATE import_settings SET match_conflict_behavior = 'trust_queue' WHERE match_conflict_behavior = 'fail';

-- +goose Down
UPDATE import_settings SET match_conflict_behavior = 'fail' WHERE match_conflict_behavior = 'trust_queue';
