-- +goose Up
CREATE TABLE files_file (
    id BLOB PRIMARY KEY NOT NULL,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    edition TEXT NOT NULL,
    root_id BLOB NOT NULL REFERENCES roots_root (id),
    path TEXT NOT NULL CHECK (path <> ''),
    size INTEGER NOT NULL CHECK (size >= 0),
    probe TEXT NOT NULL,
    grab_id BLOB REFERENCES grabs_grab (id),
    UNIQUE (root_id, path)
) STRICT;

CREATE INDEX files_file_work_id ON files_file (work_id);
CREATE INDEX files_file_item_id ON files_file (item_id);

CREATE TABLE files_fill (
    file_id BLOB NOT NULL REFERENCES files_file (id) ON DELETE CASCADE,
    target_id BLOB NOT NULL REFERENCES targets_target (id),
    item_id BLOB NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    edition TEXT NOT NULL,
    UNIQUE (item_id, edition, target_id)
) STRICT;

CREATE INDEX files_fill_file_id ON files_fill (file_id);
CREATE INDEX files_fill_target_id ON files_fill (target_id);
