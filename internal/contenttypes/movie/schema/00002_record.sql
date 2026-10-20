-- +goose Up
-- The whole record. released goes: availability is read from movie_release.
-- Existing rows keep their facts until a refresh reads the rest.
CREATE TABLE movie_movie_next (
    work_id BLOB PRIMARY KEY NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    tmdb_id INTEGER NOT NULL UNIQUE CHECK (tmdb_id >= 1),
    imdb_id TEXT CHECK (imdb_id <> ''),
    -- The Work's label, in the metadata language.
    title TEXT NOT NULL CHECK (title <> ''),
    -- Search terms and naming tokens read this one.
    title_english TEXT NOT NULL CHECK (title_english <> ''),
    original_title TEXT NOT NULL CHECK (original_title <> ''),
    original_language TEXT NOT NULL CHECK (original_language <> ''),
    year INTEGER,
    primary_release TEXT CHECK (primary_release GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    overview TEXT NOT NULL DEFAULT '',
    tagline TEXT NOT NULL DEFAULT '',
    runtime_minutes INTEGER CHECK (runtime_minutes >= 1),
    status TEXT NOT NULL CHECK (status IN ('rumored', 'planned', 'in-production', 'post-production', 'released', 'canceled')),
    certification TEXT CHECK (certification <> ''),
    certification_country TEXT CHECK (length(certification_country) = 2),
    studio_id INTEGER CHECK (studio_id >= 1),
    studio_name TEXT CHECK (studio_name <> ''),
    collection_id INTEGER CHECK (collection_id >= 1),
    collection_name TEXT CHECK (collection_name <> ''),
    trailer_site TEXT CHECK (trailer_site <> ''),
    trailer_key TEXT CHECK (trailer_key <> ''),
    -- OMDb's line; when OMDb last answered is works_work.ratings_read_at.
    awards TEXT CHECK (awards <> ''),
    CHECK ((certification IS NULL) = (certification_country IS NULL)),
    CHECK ((studio_id IS NULL) = (studio_name IS NULL)),
    CHECK ((collection_id IS NULL) = (collection_name IS NULL)),
    CHECK ((trailer_site IS NULL) = (trailer_key IS NULL))
) STRICT;
INSERT INTO movie_movie_next (work_id, tmdb_id, imdb_id, title, title_english, original_title, original_language, year, runtime_minutes, status)
SELECT work_id, tmdb_id, imdb_id, title, title, title, 'en', year, runtime_minutes, 'released' FROM movie_movie;
DROP TABLE movie_movie;
ALTER TABLE movie_movie_next RENAME TO movie_movie;

CREATE TABLE movie_genre (
    work_id BLOB NOT NULL REFERENCES movie_movie (work_id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 1),
    name TEXT NOT NULL CHECK (name <> ''),
    PRIMARY KEY (work_id, position)
) STRICT, WITHOUT ROWID;

-- Alternative titles.
CREATE TABLE movie_title (
    work_id BLOB NOT NULL REFERENCES movie_movie (work_id) ON DELETE CASCADE,
    country TEXT NOT NULL CHECK (length(country) = 2),
    title TEXT NOT NULL CHECK (title <> ''),
    kind TEXT NOT NULL DEFAULT ''
) STRICT;
CREATE INDEX movie_title_work ON movie_title (work_id, country);

CREATE TABLE movie_release (
    work_id BLOB NOT NULL REFERENCES movie_movie (work_id) ON DELETE CASCADE,
    country TEXT NOT NULL CHECK (length(country) = 2),
    -- TMDB's release type: premiere, limited, cinema, digital, physical, TV.
    kind INTEGER NOT NULL CHECK (kind BETWEEN 1 AND 6),
    date TEXT NOT NULL CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    certification TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT ''
) STRICT;
CREATE INDEX movie_release_work ON movie_release (work_id, kind, country, date);

-- Directors, then writers, then the cast by billing.
CREATE TABLE movie_credit (
    work_id BLOB NOT NULL REFERENCES movie_movie (work_id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 1),
    person_id INTEGER NOT NULL CHECK (person_id >= 1),
    name TEXT NOT NULL CHECK (name <> ''),
    department TEXT NOT NULL CHECK (department IN ('directing', 'writing', 'acting')),
    -- The job, or the character for the cast.
    role TEXT NOT NULL DEFAULT '',
    photo_path TEXT CHECK (photo_path <> ''),
    PRIMARY KEY (work_id, position)
) STRICT, WITHOUT ROWID;

CREATE TABLE movie_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    -- NULL is the default for each.
    counts_as TEXT CHECK (counts_as IN ('home', 'cinema')),
    cinema_delay_seconds INTEGER CHECK (cinema_delay_seconds >= 0)
) STRICT;
INSERT INTO movie_settings (id) VALUES (1);
