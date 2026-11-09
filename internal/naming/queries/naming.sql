-- name: GetType :one
SELECT * FROM naming_type WHERE content_type = sqlc.arg(content_type);

-- name: ListTemplates :many
SELECT * FROM naming_template WHERE content_type = sqlc.arg(content_type) ORDER BY field;

-- name: PutType :exec
INSERT INTO naming_type (content_type, rename_files, colon, colon_custom, multi_episode)
VALUES (sqlc.arg(content_type), sqlc.narg(rename_files), sqlc.narg(colon), sqlc.narg(colon_custom), sqlc.narg(multi_episode))
ON CONFLICT (content_type) DO UPDATE SET rename_files = excluded.rename_files, colon = excluded.colon,
    colon_custom = excluded.colon_custom, multi_episode = excluded.multi_episode;

-- name: ClearTemplates :exec
DELETE FROM naming_template WHERE content_type = sqlc.arg(content_type);

-- name: PutTemplate :exec
INSERT INTO naming_template (content_type, field, template) VALUES (sqlc.arg(content_type), sqlc.arg(field), sqlc.arg(template));
