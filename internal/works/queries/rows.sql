-- name: ListWorksByID :many
SELECT * FROM works_work WHERE id IN (sqlc.slice(ids)) ORDER BY sort_label, id;

-- name: ListLevelsOfWorks :many
SELECT * FROM works_level WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id, depth, withdrawn_at IS NOT NULL, position, id;

-- name: ListWorkArtOfWorks :many
SELECT work_id, source, path FROM works_artwork
WHERE kind = sqlc.arg(kind) AND ifnull(item_id, x'') = x'' AND work_id IN (sqlc.slice(work_ids))
ORDER BY work_id;

-- name: ListWorkRatingsOfWorks :many
SELECT work_id, source, value, votes, fetched_at FROM works_rating
WHERE ifnull(item_id, x'') = x'' AND work_id IN (sqlc.slice(work_ids))
ORDER BY work_id, source;
