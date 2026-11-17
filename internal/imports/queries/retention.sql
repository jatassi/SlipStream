-- name: DeletePrunable :execrows
DELETE FROM imports_import WHERE id IN (
    SELECT i.id FROM imports_import AS i INDEXED BY imports_import_prunable
    WHERE i.finished_at IS NOT NULL AND i.grab_id IS NULL AND i.finished_at < sqlc.arg(before)
    ORDER BY i.finished_at, i.id LIMIT sqlc.arg(limit));
