-- name: GetWork :one
SELECT * FROM library_work WHERE id = sqlc.arg(id);

-- name: ListWorks :many
SELECT * FROM library_work WHERE root_id = sqlc.arg(root_id);

-- name: ListWorksByTitle :many
SELECT * FROM library_work
WHERE root_id = sqlc.arg(root_id)
ORDER BY title, id;

-- name: NewestWork :one
SELECT * FROM library_work ORDER BY added_at DESC LIMIT 1;

-- name: AnyWork :one
SELECT * FROM library_work LIMIT 1;

-- name: ListRecentWorks :many
SELECT * FROM (
    SELECT * FROM library_work ORDER BY added_at DESC LIMIT 10
) recent;

-- name: ListRankedWorks :many
SELECT id, row_number() OVER (ORDER BY added_at) AS rank
FROM library_work
WHERE title <> 'ORDER BY title'
-- ORDER BY id
/* ORDER BY id */;

-- name: ListWorksInRoots :many
WITH roots AS (SELECT id FROM library_root ORDER BY id)
SELECT library_work.* FROM library_work JOIN roots ON roots.id = library_work.root_id
ORDER BY library_work.id;

-- name: CountWorks :one
SELECT count(*) FROM library_work WHERE id IN (SELECT work_id FROM library_file LIMIT 5);

-- name: ListQuotedOrder :many
SELECT "order", [limit] FROM library_work WHERE "order" > 0
ORDER BY "order";

-- name: ListFirstWorks :many
SELECT * FROM library_work LIMIT sqlc.arg(count);
