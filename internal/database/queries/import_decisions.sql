-- name: UpsertImportDecision :one
INSERT INTO import_decisions (
    source_path, decision, module_type, entity_type, entity_id, slot_id,
    candidate_quality_id, existing_quality_id, existing_file_id,
    quality_profile_id, reason, evaluated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT (source_path) DO UPDATE SET
    decision = excluded.decision,
    module_type = excluded.module_type,
    entity_type = excluded.entity_type,
    entity_id = excluded.entity_id,
    slot_id = excluded.slot_id,
    candidate_quality_id = excluded.candidate_quality_id,
    existing_quality_id = excluded.existing_quality_id,
    existing_file_id = excluded.existing_file_id,
    quality_profile_id = excluded.quality_profile_id,
    reason = excluded.reason,
    evaluated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetImportDecision :one
SELECT * FROM import_decisions WHERE source_path = ? LIMIT 1;

-- name: DeleteImportDecision :exec
DELETE FROM import_decisions WHERE source_path = ?;

-- name: DeleteImportDecisionsByMediaItem :exec
DELETE FROM import_decisions WHERE module_type = ? AND entity_type = ? AND entity_id = ?;

-- name: DeleteImportDecisionsByProfile :exec
DELETE FROM import_decisions WHERE quality_profile_id = ?;

-- name: DeleteImportDecisionsByExistingFile :exec
DELETE FROM import_decisions WHERE existing_file_id = ?;

-- name: DeleteImportDecisionsByPathPrefix :exec
DELETE FROM import_decisions WHERE source_path LIKE ? || '%';

-- name: CleanupAllImportDecisions :exec
DELETE FROM import_decisions;
