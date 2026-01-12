-- name: GetDownloadClient :one
SELECT * FROM download_clients WHERE id = ? LIMIT 1;

-- name: ListDownloadClients :many
SELECT * FROM download_clients ORDER BY priority, name;

-- name: ListEnabledDownloadClients :many
SELECT * FROM download_clients WHERE enabled = 1 ORDER BY priority, name;

-- name: CreateDownloadClient :one
INSERT INTO download_clients (
    name, type, host, port, username, password, use_ssl, category, priority, enabled,
    import_delay_seconds, cleanup_mode, seed_ratio_target
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateDownloadClient :one
UPDATE download_clients SET
    name = ?,
    type = ?,
    host = ?,
    port = ?,
    username = ?,
    password = ?,
    use_ssl = ?,
    category = ?,
    priority = ?,
    enabled = ?,
    import_delay_seconds = ?,
    cleanup_mode = ?,
    seed_ratio_target = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteDownloadClient :exec
DELETE FROM download_clients WHERE id = ?;

-- name: CountDownloadClients :one
SELECT COUNT(*) FROM download_clients;
