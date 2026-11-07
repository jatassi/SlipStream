-- +goose Up
CREATE TABLE series_series (
    work_id BLOB PRIMARY KEY NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    tvdb_id INTEGER NOT NULL UNIQUE CHECK (tvdb_id > 0),
    tmdb_id INTEGER UNIQUE CHECK (tmdb_id > 0),
    imdb_id TEXT CHECK (imdb_id <> ''),
    english_title TEXT NOT NULL CHECK (english_title <> ''),
    original_title TEXT,
    original_language TEXT,
    country TEXT,
    network TEXT,
    creators BLOB,
    credits BLOB,
    status TEXT NOT NULL CHECK (status IN ('continuing', 'ended', 'upcoming')),
    first_aired TEXT,
    last_aired TEXT,
    next_aired TEXT,
    average_runtime_seconds INTEGER CHECK (average_runtime_seconds > 0),
    certification TEXT,
    certification_country TEXT,
    genres BLOB,
    aliases BLOB,
    overview TEXT,
    episode_order TEXT NOT NULL CHECK (episode_order IN ('default', 'dvd', 'absolute')),
    orders BLOB NOT NULL,
    trailer_site TEXT,
    trailer_key TEXT,
    awards TEXT
) STRICT;

CREATE TABLE series_season (
    level_id BLOB PRIMARY KEY NOT NULL REFERENCES works_level (id) ON DELETE CASCADE,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    tvdb_season_id INTEGER NOT NULL CHECK (tvdb_season_id > 0),
    number INTEGER NOT NULL CHECK (number >= 0),
    episode_order TEXT NOT NULL CHECK (episode_order IN ('default', 'dvd', 'absolute')),
    name TEXT,
    overview TEXT
) STRICT;
CREATE INDEX series_season_work ON series_season (work_id);

CREATE TABLE series_episode (
    item_id BLOB PRIMARY KEY NOT NULL REFERENCES works_item (id) ON DELETE CASCADE,
    work_id BLOB NOT NULL REFERENCES works_work (id) ON DELETE CASCADE,
    tvdb_episode_id INTEGER NOT NULL CHECK (tvdb_episode_id > 0),
    season_number INTEGER NOT NULL CHECK (season_number >= 0),
    episode_number INTEGER NOT NULL CHECK (episode_number >= 0),
    absolute_number INTEGER CHECK (absolute_number > 0),
    aired_season INTEGER CHECK (aired_season >= 0),
    aired_number INTEGER CHECK (aired_number >= 0),
    air_date TEXT,
    runtime_seconds INTEGER CHECK (runtime_seconds > 0),
    overview TEXT,
    imdb_id TEXT
) STRICT;
CREATE INDEX series_episode_work ON series_episode (work_id);

CREATE TABLE series_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    -- NULL is the default.
    air_delay_days INTEGER CHECK (air_delay_days BETWEEN 0 AND 30)
) STRICT;
INSERT INTO series_settings (id) VALUES (1);
