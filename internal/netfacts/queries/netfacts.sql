-- name: ListMappings :many
SELECT * FROM netfacts_mapping ORDER BY port, protocol;

-- name: AskMapping :exec
INSERT INTO netfacts_mapping (port, protocol, method, gateway, external_port, nonce, asked_at)
VALUES (?, 'tcp', ?, ?, ?, ?, ?)
ON CONFLICT (port, protocol) DO UPDATE SET
    method = excluded.method, gateway = excluded.gateway, external_port = excluded.external_port,
    nonce = excluded.nonce, asked_at = excluded.asked_at, granted_at = NULL, expires_at = NULL, refused = NULL;

-- name: GrantMapping :exec
UPDATE netfacts_mapping SET granted_at = ?, expires_at = ?, refused = NULL WHERE port = ? AND protocol = 'tcp';

-- name: RefuseMapping :exec
UPDATE netfacts_mapping SET refused = ?, granted_at = NULL, expires_at = NULL WHERE port = ? AND protocol = 'tcp';

-- name: DeleteMapping :exec
DELETE FROM netfacts_mapping WHERE port = ? AND protocol = 'tcp';
