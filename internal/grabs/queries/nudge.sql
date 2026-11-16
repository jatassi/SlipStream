-- name: InsertNudge :exec
INSERT INTO grabs_nudge (work_id, trigger, lane, items, created_at)
VALUES (sqlc.arg(work_id), sqlc.arg(trigger), sqlc.narg(lane), sqlc.narg(items), sqlc.arg(created_at));

-- name: ListNudgedWorks :many
-- The Works nudged first, up to n.
SELECT work_id FROM grabs_nudge GROUP BY work_id ORDER BY min(id) LIMIT sqlc.arg(n);

-- name: ListNudgesOf :many
SELECT * FROM grabs_nudge WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY id;

-- name: DeleteNudgesOf :exec
DELETE FROM grabs_nudge WHERE work_id IN (sqlc.slice(work_ids));

-- name: ListNudges :many
-- Every nudge, oldest first: grabstest's snapshot of the queue.
SELECT * FROM grabs_nudge ORDER BY id;
