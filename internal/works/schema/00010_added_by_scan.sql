-- +goose Up
-- The Works a Library scan adopted, by when, for S37 (Notifications)'s
-- "added by a Library scan" count since a root's previous comparison.
CREATE INDEX works_work_added_by_scan ON works_work (added_at) WHERE added_by = 'scan';
