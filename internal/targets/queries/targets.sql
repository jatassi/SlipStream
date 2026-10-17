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
INSERT INTO targets_profile_target (profile_id, position, target_id, sourcing)
VALUES (sqlc.arg(profile_id), sqlc.arg(position), sqlc.arg(target_id), sqlc.arg(sourcing));

-- name: ListProfiles :many
SELECT * FROM targets_profile ORDER BY content_type, name, id;

-- name: ListProfileTargets :many
SELECT * FROM targets_profile_target ORDER BY profile_id, position;

-- name: GetProfile :one
SELECT * FROM targets_profile WHERE id = sqlc.arg(id);

-- name: DefaultProfile :one
SELECT * FROM targets_profile WHERE content_type = sqlc.arg(content_type) AND is_default = 1;

-- name: ClearDefault :exec
UPDATE targets_profile SET is_default = 0 WHERE content_type = sqlc.arg(content_type) AND is_default = 1;

-- name: SetDefault :exec
UPDATE targets_profile SET is_default = 1 WHERE id = sqlc.arg(id);

-- name: CountWants :many
SELECT profile_id, count(*) AS works FROM targets_wants GROUP BY profile_id ORDER BY profile_id;

-- name: GetStarter :one
SELECT * FROM targets_starter WHERE id = 1;

-- name: InsertStarter :exec
INSERT INTO targets_starter (id, target_id) VALUES (1, sqlc.narg(target_id));

-- name: InsertWants :exec
INSERT INTO targets_wants (work_id, profile_id) VALUES (sqlc.arg(work_id), sqlc.arg(profile_id));

-- name: GetWants :one
SELECT * FROM targets_wants WHERE work_id = sqlc.arg(work_id);

-- name: ListWants :many
SELECT * FROM targets_wants ORDER BY work_id;

-- name: ListWorksOfProfile :many
SELECT work_id FROM targets_wants WHERE profile_id = sqlc.arg(profile_id) ORDER BY work_id;
