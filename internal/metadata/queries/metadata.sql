-- name: ListKeys :many
SELECT * FROM metadata_key ORDER BY provider;

-- name: PutKey :exec
INSERT INTO metadata_key (provider, api_key_sealed, set_at)
VALUES (sqlc.arg(provider), sqlc.arg(api_key_sealed), sqlc.arg(set_at))
ON CONFLICT (provider) DO UPDATE SET api_key_sealed = excluded.api_key_sealed, set_at = excluded.set_at;

-- name: DeleteKey :execrows
DELETE FROM metadata_key WHERE provider = sqlc.arg(provider);

-- name: GetSettings :one
SELECT * FROM metadata_settings;

-- name: SetSettings :exec
UPDATE metadata_settings SET
    language = sqlc.narg(language),
    region = sqlc.narg(region),
    unreleased_every_seconds = sqlc.narg(unreleased_every_seconds),
    growing_every_seconds = sqlc.narg(growing_every_seconds),
    others_every_seconds = sqlc.narg(others_every_seconds),
    follow_changes = sqlc.narg(follow_changes),
    ratings_every_seconds = sqlc.narg(ratings_every_seconds),
    cache_for_seconds = sqlc.narg(cache_for_seconds),
    artwork_store_mb = sqlc.narg(artwork_store_mb);

-- name: RefreshAll :exec
INSERT INTO metadata_type (content_type, refresh_all_at)
VALUES (sqlc.arg(content_type), sqlc.arg(refresh_all_at))
ON CONFLICT (content_type) DO UPDATE SET refresh_all_at = max(coalesce(refresh_all_at, 0), excluded.refresh_all_at);
