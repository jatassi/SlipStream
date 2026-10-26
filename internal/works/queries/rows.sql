-- name: ListWorksByID :many
SELECT * FROM works_work WHERE id IN (sqlc.slice(ids)) ORDER BY sort_label, id;

-- name: ListLevelsOfWorks :many
SELECT * FROM works_level WHERE work_id IN (sqlc.slice(work_ids));

-- name: ListEveryLevel :many
SELECT * FROM works_level;

-- name: ListEveryItem :many
SELECT * FROM works_item;

-- name: ListWorkArtOfWorks :many
SELECT work_id, source, path FROM works_artwork
WHERE kind = sqlc.arg(kind) AND ifnull(item_id, x'') = x'' AND work_id IN (sqlc.slice(work_ids));

-- name: ListEveryWorkArt :many
SELECT work_id, source, path FROM works_artwork WHERE kind = sqlc.arg(kind) AND ifnull(item_id, x'') = x'';

-- name: ListWorkRatingsOfWorks :many
SELECT work_id, source, value, votes, fetched_at FROM works_rating
WHERE ifnull(item_id, x'') = x'' AND work_id IN (sqlc.slice(work_ids));

-- name: ListEveryWorkRating :many
SELECT work_id, source, value, votes, fetched_at FROM works_rating WHERE ifnull(item_id, x'') = x'';
