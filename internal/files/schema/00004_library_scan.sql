-- +goose Up
-- What a library scan or an import read of a file besides its size: its
-- modification time and its Node's file and volume ids, as storing.FileID
-- spells them. NULL until one read them, and the ids NULL where the volume
-- keeps none.
ALTER TABLE files_file ADD COLUMN modified_at INTEGER;
ALTER TABLE files_file ADD COLUMN node_file_id TEXT CHECK (node_file_id <> '');
ALTER TABLE files_file ADD COLUMN node_volume_id TEXT;

-- gone_at is when a scan found the file's bytes absent from its Ready
-- root. A gone file keeps its row and holds no fill.
ALTER TABLE files_file ADD COLUMN gone_at INTEGER;

-- probe_version is the media record's own version, 0 for a record from
-- before it had one, so a scan finds the records older than its parser.
ALTER TABLE files_file ADD COLUMN probe_version INTEGER NOT NULL
    GENERATED ALWAYS AS (coalesce(json_extract(probe, '$.version'), 0)) VIRTUAL;
ALTER TABLE files_file ADD COLUMN probed_at INTEGER;
-- probe_error is ffprobe's first error line when the last probe failed;
-- the record before it stays.
ALTER TABLE files_file ADD COLUMN probe_error TEXT CHECK (probe_error <> '');

CREATE INDEX files_file_gone ON files_file (root_id) WHERE gone_at IS NOT NULL;
