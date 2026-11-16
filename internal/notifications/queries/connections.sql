-- name: ListConnections :many
SELECT * FROM notifications_connection ORDER BY name;

-- name: GetConnection :one
SELECT * FROM notifications_connection WHERE id = sqlc.arg(id);

-- name: InsertConnection :exec
INSERT INTO notifications_connection (id, name, provider, is_on, on_at, events, warnings, posters, node_id, settings, secrets_sealed, revision, created_at)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(provider), sqlc.arg(is_on), sqlc.arg(on_at), sqlc.arg(events), sqlc.arg(warnings), sqlc.arg(posters), sqlc.narg(node_id), sqlc.arg(settings), sqlc.narg(secrets_sealed), 1, sqlc.arg(created_at));

-- name: UpdateConnection :exec
UPDATE notifications_connection
SET name = sqlc.arg(name), is_on = sqlc.arg(is_on), on_at = sqlc.arg(on_at), events = sqlc.arg(events),
    warnings = sqlc.arg(warnings), posters = sqlc.arg(posters), node_id = sqlc.narg(node_id),
    settings = sqlc.arg(settings), secrets_sealed = sqlc.narg(secrets_sealed), revision = revision + 1
WHERE id = sqlc.arg(id);

-- name: BumpRevision :exec
UPDATE notifications_connection SET revision = revision + 1 WHERE id = sqlc.arg(id);

-- name: DeleteConnection :exec
DELETE FROM notifications_connection WHERE id = sqlc.arg(id);

-- name: ListScriptsOn :many
SELECT * FROM notifications_connection WHERE node_id = sqlc.arg(node_id) ORDER BY name;
