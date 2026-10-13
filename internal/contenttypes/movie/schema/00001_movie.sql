-- +goose Up
CREATE TABLE movie_movie (
    work_id BLOB PRIMARY KEY NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    tmdb_id INTEGER NOT NULL UNIQUE CHECK (tmdb_id >= 1),
    imdb_id TEXT CHECK (imdb_id <> ''),
    title TEXT NOT NULL CHECK (title <> ''),
    year INTEGER,
    released TEXT CHECK (released <> ''),
    runtime_minutes INTEGER CHECK (runtime_minutes >= 1)
) STRICT;
