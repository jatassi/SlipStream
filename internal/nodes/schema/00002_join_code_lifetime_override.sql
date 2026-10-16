-- +goose Up
CREATE TABLE nodes_settings_new (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    join_code_lifetime_seconds INTEGER
) STRICT;

INSERT INTO nodes_settings_new (id, join_code_lifetime_seconds)
SELECT id, NULLIF(join_code_lifetime_seconds, 3600) FROM nodes_settings;

DROP TABLE nodes_settings;

ALTER TABLE nodes_settings_new RENAME TO nodes_settings;
