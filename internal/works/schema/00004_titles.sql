-- +goose Up
CREATE TABLE works_title (
    row INTEGER PRIMARY KEY,
    work_id BLOB NOT NULL UNIQUE REFERENCES works_work (id) ON DELETE CASCADE,
    labels TEXT NOT NULL,
    titles TEXT NOT NULL
) STRICT;

CREATE VIRTUAL TABLE works_title_index USING fts5 (
    labels, titles, content = 'works_title', content_rowid = 'row', tokenize = 'unicode61 remove_diacritics 2'
);

-- +goose StatementBegin
CREATE TRIGGER works_title_insert AFTER INSERT ON works_title BEGIN
    INSERT INTO works_title_index (rowid, labels, titles) VALUES (new.row, new.labels, new.titles);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER works_title_delete AFTER DELETE ON works_title BEGIN
    INSERT INTO works_title_index (works_title_index, rowid, labels, titles) VALUES ('delete', old.row, old.labels, old.titles);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER works_title_update AFTER UPDATE ON works_title BEGIN
    INSERT INTO works_title_index (works_title_index, rowid, labels, titles) VALUES ('delete', old.row, old.labels, old.titles);
    INSERT INTO works_title_index (rowid, labels, titles) VALUES (new.row, new.labels, new.titles);
END;
-- +goose StatementEnd
