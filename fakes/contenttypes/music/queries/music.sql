-- name: UpsertArtist :exec
INSERT INTO music_artist (work_id, country, disambiguation)
VALUES (sqlc.arg(work_id), sqlc.arg(country), sqlc.arg(disambiguation))
ON CONFLICT (work_id) DO UPDATE SET country = excluded.country, disambiguation = excluded.disambiguation;

-- name: UpsertAlbum :exec
INSERT INTO music_album (level_id, work_id, released, primary_type, release_id)
VALUES (sqlc.arg(level_id), sqlc.arg(work_id), sqlc.arg(released), sqlc.arg(primary_type), sqlc.arg(release_id))
ON CONFLICT (level_id) DO UPDATE SET released = excluded.released, primary_type = excluded.primary_type, release_id = excluded.release_id;

-- name: UpsertTrack :exec
INSERT INTO music_track (item_id, work_id, disc, number, length_ms)
VALUES (sqlc.arg(item_id), sqlc.arg(work_id), sqlc.arg(disc), sqlc.arg(number), sqlc.arg(length_ms))
ON CONFLICT (item_id) DO UPDATE SET disc = excluded.disc, number = excluded.number, length_ms = excluded.length_ms;

-- name: GetArtist :one
SELECT * FROM music_artist WHERE work_id = sqlc.arg(work_id);

-- name: ListAlbums :many
SELECT * FROM music_album WHERE work_id = sqlc.arg(work_id) ORDER BY level_id;

-- name: ListTracks :many
SELECT * FROM music_track WHERE work_id = sqlc.arg(work_id) ORDER BY item_id;

-- name: GetTrack :one
SELECT * FROM music_track WHERE item_id = sqlc.arg(item_id);
