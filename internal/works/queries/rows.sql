-- name: ListWorksByID :many
SELECT * FROM works_work WHERE id IN (sqlc.slice(ids)) ORDER BY sort_label, id;

-- name: ListLevelsOfWorks :many
SELECT * FROM works_level WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: ListEveryLevel :many
SELECT * FROM works_level ORDER BY rowid;

-- name: ListEveryItem :many
SELECT * FROM works_item ORDER BY rowid;

-- name: ListWorkArtOfWorks :many
SELECT work_id, source, path FROM works_artwork
WHERE kind = sqlc.arg(kind) AND ifnull(item_id, x'') = x'' AND work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: ListEveryWorkArt :many
SELECT work_id, source, path FROM works_artwork WHERE kind = sqlc.arg(kind) AND ifnull(item_id, x'') = x'' ORDER BY rowid;

-- name: ListWorkRatingsOfWorks :many
SELECT work_id, source, value, votes, fetched_at FROM works_rating
WHERE ifnull(item_id, x'') = x'' AND work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: ListEveryWorkRating :many
SELECT work_id, source, value, votes, fetched_at FROM works_rating WHERE ifnull(item_id, x'') = x'' ORDER BY rowid;
