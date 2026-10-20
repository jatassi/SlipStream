-- name: ListKeys :many
SELECT * FROM metadata_key ORDER BY provider;

-- name: PutKey :exec
INSERT INTO metadata_key (provider, api_key_sealed, set_at)
VALUES (sqlc.arg(provider), sqlc.arg(api_key_sealed), sqlc.arg(set_at))
ON CONFLICT (provider) DO UPDATE SET api_key_sealed = excluded.api_key_sealed, set_at = excluded.set_at;

-- name: DeleteKey :execrows
DELETE FROM metadata_key WHERE provider = sqlc.arg(provider);
