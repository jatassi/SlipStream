-- name: ListRules :many
SELECT * FROM trackers_rule ORDER BY tracker_id, source;

-- name: ListRulesOf :many
SELECT * FROM trackers_rule WHERE tracker_id = sqlc.arg(tracker_id) ORDER BY source;

-- name: SetAdminRule :exec
INSERT INTO trackers_rule (tracker_id, source, clears, ratio_centi, seed_seconds, updated_at)
VALUES (sqlc.arg(tracker_id), 'admin', sqlc.arg(clears), sqlc.narg(ratio_centi), sqlc.narg(seed_seconds), sqlc.arg(updated_at))
ON CONFLICT (tracker_id, source) DO UPDATE SET
    clears = excluded.clears, ratio_centi = excluded.ratio_centi, seed_seconds = excluded.seed_seconds, updated_at = excluded.updated_at;

-- name: ClearAdminRule :exec
DELETE FROM trackers_rule WHERE tracker_id = sqlc.arg(tracker_id) AND source = 'admin';

-- name: SetStatedRule :execrows
INSERT INTO trackers_rule (tracker_id, source, clears, ratio_centi, seed_seconds, stated_at, updated_at)
VALUES (sqlc.arg(tracker_id), 'releases', sqlc.arg(clears), sqlc.narg(ratio_centi), sqlc.narg(seed_seconds), sqlc.arg(stated_at), sqlc.arg(updated_at))
ON CONFLICT (tracker_id, source) DO UPDATE SET
    clears = excluded.clears, ratio_centi = excluded.ratio_centi, seed_seconds = excluded.seed_seconds,
    stated_at = excluded.stated_at, updated_at = excluded.updated_at
WHERE excluded.stated_at > trackers_rule.stated_at;

-- name: ListGoals :many
SELECT * FROM trackers_goal ORDER BY tracker_id;

-- name: GetGoal :one
SELECT * FROM trackers_goal WHERE tracker_id = sqlc.arg(tracker_id);

-- name: SetGoal :exec
INSERT INTO trackers_goal (tracker_id, combine, ratio_centi, seed_seconds, action)
VALUES (sqlc.arg(tracker_id), sqlc.arg(combine), sqlc.narg(ratio_centi), sqlc.narg(seed_seconds), sqlc.narg(action))
ON CONFLICT (tracker_id) DO UPDATE SET
    combine = excluded.combine, ratio_centi = excluded.ratio_centi, seed_seconds = excluded.seed_seconds, action = excluded.action;

-- name: ClearGoal :exec
DELETE FROM trackers_goal WHERE tracker_id = sqlc.arg(tracker_id);

-- name: GetSettings :one
SELECT * FROM trackers_settings;

-- name: SetMargin :exec
UPDATE trackers_settings SET margin_seconds = sqlc.narg(margin_seconds);

-- name: SetWarnAfter :exec
UPDATE trackers_settings SET warn_after_seconds = sqlc.narg(warn_after_seconds);

-- name: SetDefaultGoal :exec
UPDATE trackers_settings SET goal_combine = sqlc.narg(combine), goal_ratio_centi = sqlc.narg(ratio_centi),
    goal_seed_seconds = sqlc.narg(seed_seconds), goal_action = sqlc.narg(action);

-- name: RepointRules :exec
UPDATE OR IGNORE trackers_rule SET tracker_id = sqlc.arg(into_id) WHERE tracker_id = sqlc.arg(from_id);

-- name: RepointGoal :exec
UPDATE OR IGNORE trackers_goal SET tracker_id = sqlc.arg(into_id) WHERE tracker_id = sqlc.arg(from_id);

-- name: RepointWindows :exec
UPDATE trackers_window SET tracker_id = sqlc.arg(into_id) WHERE tracker_id = sqlc.arg(from_id);

-- name: DeleteGoalRuns :exec
DELETE FROM trackers_goal_run WHERE torrent_id IN (sqlc.slice(torrent_ids));
