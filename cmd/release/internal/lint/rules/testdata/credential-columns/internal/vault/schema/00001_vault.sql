-- +goose Up
CREATE TABLE vault_login (
    id BLOB PRIMARY KEY NOT NULL,
    password TEXT NOT NULL,
    api_key_sealed BLOB,
    token_hash BLOB NOT NULL,
    tokens_used INTEGER NOT NULL
) STRICT;
