-- name: UpsertNetworkLogo :exec
INSERT INTO network_logos (name, logo_url, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(name) DO UPDATE SET
    logo_url = excluded.logo_url,
    updated_at = CURRENT_TIMESTAMP;

-- name: GetNetworkLogo :one
SELECT * FROM network_logos WHERE name = ? LIMIT 1;

-- name: GetAllNetworkLogos :many
SELECT * FROM network_logos ORDER BY name;
