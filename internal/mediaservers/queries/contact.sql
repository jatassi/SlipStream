-- name: GetContact :one
SELECT reaching_node_id, settings_version, contact, contact_since, last_contact_at
FROM mediaservers_server
WHERE id = sqlc.arg(id);

-- name: SetContact :exec
UPDATE mediaservers_server
SET contact = sqlc.arg(contact),
    contact_since = sqlc.arg(contact_since),
    contact_found = sqlc.narg(contact_found),
    last_contact_at = sqlc.narg(last_contact_at)
WHERE id = sqlc.arg(id);
