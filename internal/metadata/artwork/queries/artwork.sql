-- name: GetEntry :one
SELECT e.hash, e.retry_at, b.type
FROM artwork_entry e LEFT JOIN artwork_blob b ON b.hash = e.hash
WHERE e.source = sqlc.arg(source) AND e.shape = sqlc.arg(shape) AND e.path = sqlc.arg(path) AND e.size = sqlc.arg(size);

-- name: MarkServed :exec
UPDATE artwork_entry SET served_at = sqlc.arg(served_at)
WHERE source = sqlc.arg(source) AND shape = sqlc.arg(shape) AND path = sqlc.arg(path) AND size = sqlc.arg(size);

-- name: PutBlob :exec
INSERT INTO artwork_blob (hash, bytes, type) VALUES (sqlc.arg(hash), sqlc.arg(bytes), sqlc.arg(type))
ON CONFLICT (hash) DO NOTHING;

-- name: PutEntry :exec
INSERT INTO artwork_entry (source, shape, path, size, hash, retry_at, served_at)
VALUES (sqlc.arg(source), sqlc.arg(shape), sqlc.arg(path), sqlc.arg(size), sqlc.arg(hash), NULL, sqlc.arg(served_at))
ON CONFLICT (source, shape, path, size) DO UPDATE SET hash = excluded.hash, retry_at = NULL, served_at = excluded.served_at;

-- name: MarkMissing :exec
INSERT INTO artwork_entry (source, shape, path, size, hash, retry_at, served_at)
VALUES (sqlc.arg(source), sqlc.arg(shape), sqlc.arg(path), sqlc.arg(size), NULL, sqlc.arg(retry_at), sqlc.arg(served_at))
ON CONFLICT (source, shape, path, size) DO UPDATE SET retry_at = excluded.retry_at
WHERE artwork_entry.hash IS NULL;

-- name: HeldBytes :one
SELECT CAST(coalesce(sum(bytes), 0) AS INTEGER) AS held FROM artwork_blob;

-- name: LeastRecentlyServed :one
SELECT e.source, e.shape, e.path, e.size, e.hash, b.bytes
FROM artwork_entry e JOIN artwork_blob b ON b.hash = e.hash
ORDER BY e.served_at, e.source, e.shape, e.path, e.size
LIMIT 1;

-- name: DeleteEntry :exec
DELETE FROM artwork_entry
WHERE source = sqlc.arg(source) AND shape = sqlc.arg(shape) AND path = sqlc.arg(path) AND size = sqlc.arg(size);

-- name: DeleteUnheldBlob :execrows
DELETE FROM artwork_blob
WHERE artwork_blob.hash = sqlc.arg(hash) AND NOT EXISTS (SELECT 1 FROM artwork_entry e WHERE e.hash = artwork_blob.hash);

-- name: DeleteSpentMisses :exec
DELETE FROM artwork_entry WHERE hash IS NULL AND retry_at <= sqlc.arg(now);

-- name: ListBlobs :many
SELECT hash FROM artwork_blob ORDER BY hash;
