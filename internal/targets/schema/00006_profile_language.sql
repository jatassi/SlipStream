-- +goose Up
-- S46 (Subtitles). A Profile's Wanted Languages in the Admin's order: a
-- language, full or forced only, how hearing-impaired subtitles rank and
-- whether audio in the language meets it. added_at is set on insert and
-- kept on reorder: it is where a gained want's first due times spread from.
CREATE TABLE targets_profile_language (
    profile_id BLOB NOT NULL REFERENCES targets_profile (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 1),
    language TEXT NOT NULL CHECK (language <> ''),
    forced INTEGER NOT NULL CHECK (forced IN (0, 1)),
    hi TEXT NOT NULL CHECK (hi IN ('avoid', 'allow', 'prefer')),
    audio_condition TEXT NOT NULL CHECK (audio_condition IN ('always', 'no_audio')),
    added_at INTEGER NOT NULL,
    PRIMARY KEY (profile_id, position),
    UNIQUE (profile_id, language, forced)
) STRICT;
