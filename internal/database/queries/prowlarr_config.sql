-- name: GetProwlarrConfig :one
SELECT * FROM prowlarr_config WHERE id = 1 LIMIT 1;

-- name: UpdateProwlarrConfig :one
UPDATE prowlarr_config SET
    enabled = ?,
    url = ?,
    api_key = ?,
    movie_categories = ?,
    tv_categories = ?,
    timeout = ?,
    skip_ssl_verify = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
RETURNING *;

-- name: UpdateProwlarrConfigCapabilities :exec
UPDATE prowlarr_config SET
    capabilities = ?,
    capabilities_updated_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;

-- name: ClearProwlarrConfigCapabilities :exec
UPDATE prowlarr_config SET
    capabilities = NULL,
    capabilities_updated_at = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;

-- name: IsProwlarrEnabled :one
SELECT enabled FROM prowlarr_config WHERE id = 1 LIMIT 1;

-- name: SetProwlarrEnabled :exec
UPDATE prowlarr_config SET
    enabled = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;
