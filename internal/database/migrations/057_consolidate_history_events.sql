-- +goose Up
UPDATE history SET event_type = 'autosearch_download' WHERE event_type = 'autosearch_upgrade';
UPDATE history SET event_type = 'imported' WHERE event_type IN ('import_completed', 'import_upgrade', 'import_started');

-- +goose Down
-- Cannot reverse: original event types are not recoverable from data alone.
