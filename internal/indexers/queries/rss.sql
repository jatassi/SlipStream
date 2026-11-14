-- name: GetRSS :one
SELECT * FROM indexers_rss WHERE indexer_id = sqlc.arg(indexer_id);

-- name: PutRSS :exec
INSERT INTO indexers_rss (indexer_id, last_guids, last_published_at, last_fetch_at, last_pages, gap_at, gap_kind)
VALUES (sqlc.arg(indexer_id), sqlc.arg(last_guids), sqlc.narg(last_published_at), sqlc.arg(last_fetch_at), sqlc.arg(last_pages),
    sqlc.narg(gap_at), sqlc.narg(gap_kind))
ON CONFLICT (indexer_id) DO UPDATE SET last_guids = excluded.last_guids, last_published_at = excluded.last_published_at,
    last_fetch_at = excluded.last_fetch_at, last_pages = excluded.last_pages, gap_at = excluded.gap_at, gap_kind = excluded.gap_kind;
