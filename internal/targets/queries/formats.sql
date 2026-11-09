-- name: FormatNameTaken :one
SELECT EXISTS (SELECT 1 FROM targets_format WHERE name_folded = sqlc.arg(name_folded) AND id <> sqlc.arg(id));

-- name: InsertFormat :exec
INSERT INTO targets_format (id, name, name_folded, include_when_renaming, dialect, conditions, source, revision, created_at)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(name_folded), sqlc.arg(include_when_renaming), sqlc.arg(dialect),
    sqlc.arg(conditions), sqlc.arg(source), 1, sqlc.arg(created_at));

-- name: ReplaceFormat :execrows
UPDATE targets_format SET name = sqlc.arg(name), name_folded = sqlc.arg(name_folded),
    include_when_renaming = sqlc.arg(include_when_renaming), dialect = sqlc.arg(dialect), conditions = sqlc.arg(conditions),
    source = sqlc.arg(source), revision = revision + 1, timeouts = 0
WHERE id = sqlc.arg(id);

-- name: DeleteFormat :execrows
DELETE FROM targets_format WHERE id = sqlc.arg(id);

-- name: GetFormat :one
SELECT * FROM targets_format WHERE id = sqlc.arg(id);

-- name: ListFormats :many
SELECT * FROM targets_format ORDER BY name_folded, id;

-- name: ListFormatRevisions :many
SELECT id, revision FROM targets_format ORDER BY id;

-- name: AddFormatTimeouts :exec
UPDATE targets_format SET timeouts = timeouts + sqlc.arg(releases) WHERE id = sqlc.arg(id);

-- name: CountFormatsOf :one
SELECT count(*) FROM targets_format WHERE id IN (sqlc.slice(ids));

-- name: InsertFormatScore :exec
INSERT INTO targets_format_score (target_id, format_id, score) VALUES (sqlc.arg(target_id), sqlc.arg(format_id), sqlc.arg(score));

-- name: DeleteFormatScores :exec
DELETE FROM targets_format_score WHERE target_id = sqlc.arg(target_id);

-- name: ListFormatScores :many
SELECT * FROM targets_format_score ORDER BY target_id, format_id;

-- name: ListScoringTargets :many
SELECT s.format_id, t.id, t.name FROM targets_format_score s
JOIN targets_target t ON t.id = s.target_id
ORDER BY t.name, t.id;

-- name: ListWorksScoring :many
SELECT DISTINCT w.work_id FROM targets_wants w
JOIN targets_profile_target pt ON pt.profile_id = w.profile_id
JOIN targets_format_score s ON s.target_id = pt.target_id
WHERE s.format_id IN (sqlc.slice(format_ids))
ORDER BY w.work_id;
