-- name: GetAddDefault :one
SELECT * FROM works_add_default WHERE content_type = sqlc.arg(content_type);

-- name: PutAddDefault :exec
INSERT INTO works_add_default (content_type, monitor, asides, level_folders)
VALUES (sqlc.arg(content_type), sqlc.narg(monitor), sqlc.narg(asides), sqlc.narg(level_folders))
ON CONFLICT (content_type) DO UPDATE SET monitor = excluded.monitor, asides = excluded.asides,
    level_folders = excluded.level_folders;
