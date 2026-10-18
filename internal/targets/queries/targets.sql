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

-- name: InsertProfile :exec
INSERT INTO targets_profile (id, name, content_type, is_default, editions)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(content_type), sqlc.arg(is_default), sqlc.arg(editions));

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

-- name: ListWantsOf :many
SELECT * FROM targets_wants WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: ListWorksOfProfile :many
SELECT work_id FROM targets_wants WHERE profile_id = sqlc.arg(profile_id) ORDER BY work_id;

-- name: TargetNameTakenByOther :one
SELECT EXISTS (SELECT 1 FROM targets_target WHERE name = sqlc.arg(name) AND id <> sqlc.arg(id));

-- name: UpdateTarget :exec
UPDATE targets_target SET name = sqlc.arg(name), cutoff_tier = sqlc.arg(cutoff_tier) WHERE id = sqlc.arg(id);

-- name: DeleteQualities :exec
DELETE FROM targets_quality WHERE target_id = sqlc.arg(target_id);

-- name: DeleteFilters :exec
DELETE FROM targets_filter WHERE target_id = sqlc.arg(target_id);

-- name: ListWorksOfTarget :many
SELECT DISTINCT w.work_id FROM targets_wants w
JOIN targets_profile_target pt ON pt.profile_id = w.profile_id
WHERE pt.target_id = sqlc.arg(target_id)
ORDER BY w.work_id;

-- name: ProfileNameTakenByOther :one
SELECT EXISTS (SELECT 1 FROM targets_profile WHERE name = sqlc.arg(name) AND id <> sqlc.arg(id));

-- name: UpdateProfile :exec
UPDATE targets_profile SET name = sqlc.arg(name), editions = sqlc.arg(editions) WHERE id = sqlc.arg(id);

-- name: DeleteProfileTargets :exec
DELETE FROM targets_profile_target WHERE profile_id = sqlc.arg(profile_id);

-- name: DeleteProfile :exec
DELETE FROM targets_profile WHERE id = sqlc.arg(id);

-- name: CountOwnLists :one
SELECT count(*) FROM targets_wants WHERE profile_id = sqlc.arg(profile_id) AND editions IS NOT NULL;

-- name: MoveWants :exec
UPDATE targets_wants SET profile_id = sqlc.arg(to_profile) WHERE profile_id = sqlc.arg(from_profile);

-- name: SetWants :exec
UPDATE targets_wants SET profile_id = sqlc.arg(profile_id), editions = sqlc.narg(editions) WHERE work_id = sqlc.arg(work_id);

-- name: InsertProfileCut :exec
INSERT INTO targets_profile_cut (profile_id, position, cut) VALUES (sqlc.arg(profile_id), sqlc.arg(position), sqlc.arg(cut));

-- name: DeleteProfileCuts :exec
DELETE FROM targets_profile_cut WHERE profile_id = sqlc.arg(profile_id);

-- name: ListProfileCuts :many
SELECT * FROM targets_profile_cut ORDER BY profile_id, position;

-- name: InsertWantsCut :exec
INSERT INTO targets_wants_cut (work_id, position, cut) VALUES (sqlc.arg(work_id), sqlc.arg(position), sqlc.arg(cut));

-- name: DeleteWantsCuts :exec
DELETE FROM targets_wants_cut WHERE work_id = sqlc.arg(work_id);

-- name: ListWantsCutsOf :many
SELECT * FROM targets_wants_cut WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id, position;

-- name: ListProfilesNamingCut :many
SELECT p.name FROM targets_profile_cut c
JOIN targets_profile p ON p.id = c.profile_id
WHERE c.cut = sqlc.arg(cut)
ORDER BY p.name;

-- name: CountOwnListsNamingCut :one
SELECT count(*) FROM targets_wants_cut WHERE cut = sqlc.arg(cut);

-- name: InsertCut :exec
INSERT INTO targets_cut (key, name, created_at) VALUES (sqlc.arg(key), sqlc.arg(name), sqlc.arg(created_at));

-- name: RenameCut :exec
UPDATE targets_cut SET name = sqlc.arg(name) WHERE key = sqlc.arg(key);

-- name: DeleteCut :exec
DELETE FROM targets_cut WHERE key = sqlc.arg(key);

-- name: ListCuts :many
SELECT * FROM targets_cut ORDER BY name, key;

-- name: InsertCutAlias :exec
INSERT INTO targets_cut_alias (cut, position, alias) VALUES (sqlc.arg(cut), sqlc.arg(position), sqlc.arg(alias));

-- name: DeleteCutAliases :exec
DELETE FROM targets_cut_alias WHERE cut = sqlc.arg(cut);

-- name: ListCutAliases :many
SELECT * FROM targets_cut_alias ORDER BY cut, position;

-- name: InsertSighting :exec
INSERT INTO targets_sighting (item_id, cut, release, seen_at)
VALUES (sqlc.arg(item_id), sqlc.arg(cut), sqlc.arg(release), sqlc.arg(seen_at))
ON CONFLICT (item_id, cut, release) DO NOTHING;

-- name: ListSightingsOf :many
SELECT DISTINCT item_id, cut FROM targets_sighting WHERE item_id IN (sqlc.slice(item_ids)) ORDER BY item_id, cut;

-- name: DeleteSightingsOfCut :exec
DELETE FROM targets_sighting WHERE cut = sqlc.arg(cut);

-- name: ListCutsOfProfile :many
SELECT cut FROM targets_profile_cut WHERE profile_id = sqlc.arg(profile_id) ORDER BY position;
