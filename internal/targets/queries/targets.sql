-- name: TargetNameTaken :one
SELECT EXISTS (SELECT 1 FROM targets_target WHERE name = sqlc.arg(name));

-- name: HasTarget :one
SELECT EXISTS (SELECT 1 FROM targets_target WHERE id = sqlc.arg(id));

-- name: InsertTarget :exec
INSERT INTO targets_target (id, name, created_at, cutoff_tier)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(created_at), sqlc.arg(cutoff_tier));

-- name: InsertQuality :exec
INSERT INTO targets_quality (target_id, quality, tier, position, min_mib_per_minute, max_mib_per_minute)
VALUES (sqlc.arg(target_id), sqlc.arg(quality), sqlc.arg(tier), sqlc.arg(position), sqlc.arg(min_mib_per_minute),
    sqlc.narg(max_mib_per_minute));

-- name: InsertFilter :exec
INSERT INTO targets_filter (target_id, attribute, mode) VALUES (sqlc.arg(target_id), sqlc.arg(attribute), sqlc.arg(mode));

-- name: InsertFilterValue :exec
INSERT INTO targets_filter_value (target_id, attribute, position, value)
VALUES (sqlc.arg(target_id), sqlc.arg(attribute), sqlc.arg(position), sqlc.arg(value));

-- name: DeleteTarget :execrows
DELETE FROM targets_target WHERE id = sqlc.arg(id);

-- name: ListTargets :many
SELECT * FROM targets_target ORDER BY name, id;

-- name: GetTarget :one
SELECT * FROM targets_target WHERE id = sqlc.arg(id);

-- name: ListQualities :many
SELECT * FROM targets_quality ORDER BY target_id, position;

-- name: ListFilters :many
SELECT * FROM targets_filter ORDER BY target_id, attribute;

-- name: ListFilterValues :many
SELECT * FROM targets_filter_value ORDER BY target_id, attribute, position;

-- name: ListHolders :many
SELECT pt.target_id, p.id, p.name FROM targets_profile_target pt
JOIN targets_profile p ON p.id = pt.profile_id
ORDER BY p.name, p.id;

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
