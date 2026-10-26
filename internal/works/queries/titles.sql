-- name: PutTitles :exec
INSERT INTO works_title (work_id, labels, titles) VALUES (sqlc.arg(work_id), sqlc.arg(labels), sqlc.arg(titles))
ON CONFLICT (work_id) DO UPDATE SET labels = excluded.labels, titles = excluded.titles
WHERE labels <> excluded.labels OR titles <> excluded.titles;

-- name: ListUntitled :many
SELECT * FROM works_work WHERE id NOT IN (SELECT work_id FROM works_title) ORDER BY id;

-- name: MatchTitles :many
SELECT t.work_id FROM works_title_index(CAST(sqlc.arg(query) AS TEXT)) i
JOIN works_title t ON t.row = i.rowid
JOIN works_work w ON w.id = t.work_id
WHERE CAST(sqlc.narg(content_type) AS TEXT) IS NULL OR w.content_type = sqlc.narg(content_type)
ORDER BY bm25(works_title_index, 4.0, 1.0), w.sort_label, w.id;
