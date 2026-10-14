-- name: TargetNameTaken :one
SELECT EXISTS (SELECT 1 FROM targets_target WHERE name = sqlc.arg(name));

-- name: HasTarget :one
SELECT EXISTS (SELECT 1 FROM targets_target WHERE id = sqlc.arg(id));

-- name: InsertTarget :exec
INSERT INTO targets_target (id, name, created_at) VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(created_at));

-- name: InsertResolution :exec
INSERT INTO targets_resolution (target_id, position, resolution)
VALUES (sqlc.arg(target_id), sqlc.arg(position), sqlc.arg(resolution));

-- name: ListTargets :many
SELECT * FROM targets_target ORDER BY name, id;

-- name: ListResolutions :many
SELECT * FROM targets_resolution ORDER BY target_id, position;

-- name: ProfileNameTaken :one
SELECT EXISTS (SELECT 1 FROM targets_profile WHERE name = sqlc.arg(name));

-- name: HasDefaultProfile :one
SELECT EXISTS (SELECT 1 FROM targets_profile WHERE content_type = sqlc.arg(content_type) AND is_default = 1);

-- name: InsertProfile :exec
INSERT INTO targets_profile (id, name, content_type, is_default)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(content_type), sqlc.arg(is_default));

-- name: InsertProfileTarget :exec
INSERT INTO targets_profile_target (profile_id, position, target_id)
VALUES (sqlc.arg(profile_id), sqlc.arg(position), sqlc.arg(target_id));

-- name: ListProfiles :many
SELECT * FROM targets_profile ORDER BY content_type, name, id;

-- name: ListProfileTargets :many
SELECT * FROM targets_profile_target ORDER BY profile_id, position;

-- name: ListDefaultTargets :many
SELECT t.* FROM targets_profile_target pt
JOIN targets_profile p ON p.id = pt.profile_id
JOIN targets_target t ON t.id = pt.target_id
WHERE p.content_type = sqlc.arg(content_type) AND p.is_default = 1
ORDER BY pt.position;
