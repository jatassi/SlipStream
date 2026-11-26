-- +goose Up
-- S36 (Playback report). Play rows change shape and the old ones go:
-- nothing read their decision vocabulary, and a CHECK can't change in
-- place. A play records each stream's decision, and its decision is the
-- one those streams give.
DROP TABLE mediaservers_play;

CREATE TABLE mediaservers_play (
    server_id BLOB NOT NULL REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    id BLOB NOT NULL,
    session_key TEXT NOT NULL,
    server_path TEXT NOT NULL CHECK (server_path <> ''),
    path_key TEXT NOT NULL CHECK (path_key <> ''),
    item_key TEXT NOT NULL,
    player_key TEXT NOT NULL,
    player TEXT NOT NULL,
    platform TEXT NOT NULL,
    product TEXT NOT NULL,
    user_name TEXT NOT NULL,
    decision TEXT NOT NULL CHECK (decision IN ('direct_play', 'direct_stream', 'transcode')),
    video TEXT NOT NULL CHECK (video IN ('direct', 'transcode', 'none')),
    audio TEXT NOT NULL CHECK (audio IN ('direct', 'transcode', 'none')),
    subtitles TEXT NOT NULL CHECK (subtitles IN ('none', 'direct', 'burned')),
    hardware INTEGER NOT NULL DEFAULT 0 CHECK (hardware IN (0, 1)),
    reasons TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(reasons) AND json_type(reasons) = 'array'),
    -- {container, videoCodec, height, bitrate, audioCodec, channels}, each absent when unreported.
    received TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(received) AND json_type(received) = 'object'),
    -- {position} or {external: {codec, language}}, null when unreported.
    audio_track TEXT CHECK (audio_track IS NULL OR (json_valid(audio_track) AND json_type(audio_track) = 'object')),
    subtitle_track TEXT CHECK (subtitle_track IS NULL OR (json_valid(subtitle_track) AND json_type(subtitle_track) = 'object')),
    file_id BLOB,
    -- Null until file_id resolves, or while it can't be taken.
    snapshot TEXT CHECK (snapshot IS NULL OR (json_valid(snapshot) AND json_type(snapshot) = 'object')),
    started_at INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    ended_at INTEGER,
    PRIMARY KEY (server_id, id),
    CHECK ((decision = 'transcode') = (video = 'transcode' OR audio = 'transcode' OR subtitles = 'burned')),
    CHECK (decision <> 'direct_stream' OR (video <> 'transcode' AND audio <> 'transcode' AND subtitles <> 'burned')),
    CHECK (snapshot IS NULL OR file_id IS NOT NULL),
    CHECK (last_seen_at >= started_at),
    CHECK (ended_at IS NULL OR ended_at = last_seen_at)
) STRICT;

CREATE INDEX mediaservers_play_started ON mediaservers_play (server_id, started_at);
CREATE INDEX mediaservers_play_player ON mediaservers_play (server_id, player_key, started_at, id);
CREATE INDEX mediaservers_play_ended ON mediaservers_play (ended_at) WHERE ended_at IS NOT NULL;
CREATE INDEX mediaservers_play_open ON mediaservers_play (server_id) WHERE ended_at IS NULL;
CREATE INDEX mediaservers_play_file ON mediaservers_play (file_id) WHERE file_id IS NOT NULL;
CREATE INDEX mediaservers_play_unfiled ON mediaservers_play (server_id, path_key) WHERE file_id IS NULL AND ended_at IS NULL;

-- plays_since is when SlipStream started recording the server's plays:
-- now, for a server that exists when the old rows go.
ALTER TABLE mediaservers_server ADD COLUMN plays_since INTEGER NOT NULL DEFAULT 0;
UPDATE mediaservers_server SET plays_since = CAST(strftime('%s', 'now') AS INTEGER) * 1000000;
