-- name: SetProfileOfWorks :exec
UPDATE targets_wants SET profile_id = sqlc.arg(profile_id) WHERE work_id IN (sqlc.slice(work_ids));

-- name: CountOwnListsOf :one
SELECT count(*) FROM targets_wants WHERE editions IS NOT NULL AND work_id IN (sqlc.slice(work_ids));

-- name: CountWantsOfOtherType :one
SELECT count(*) FROM targets_wants w JOIN targets_profile p ON p.id = w.profile_id
WHERE p.content_type <> sqlc.arg(content_type) AND w.work_id IN (sqlc.slice(work_ids));
