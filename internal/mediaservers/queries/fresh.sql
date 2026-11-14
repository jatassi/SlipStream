-- The fresh folder rule's columns on one mapping, as its Job answered.

-- name: SetMappingScan :exec
UPDATE mediaservers_mapping
SET scan_requested_at = sqlc.arg(requested_at), scan_started_at = sqlc.arg(started_at), scanned_at = sqlc.arg(scanned_at)
WHERE server_id = sqlc.arg(server_id) AND root_id = sqlc.arg(root_id);
