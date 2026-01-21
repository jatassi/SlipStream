-- name: GetRequest :one
SELECT * FROM requests WHERE id = ? LIMIT 1;

-- name: GetRequestByTmdbID :one
SELECT * FROM requests
WHERE tmdb_id = ? AND media_type = ? AND status NOT IN ('denied', 'available')
LIMIT 1;

-- name: GetRequestByTvdbID :one
SELECT * FROM requests
WHERE tvdb_id = ? AND media_type = ? AND status NOT IN ('denied', 'available')
LIMIT 1;

-- name: GetRequestByTvdbIDAndSeason :one
SELECT * FROM requests
WHERE tvdb_id = ? AND media_type = 'season' AND season_number = ? AND status NOT IN ('denied', 'available')
LIMIT 1;

-- name: GetRequestByTvdbIDAndEpisode :one
SELECT * FROM requests
WHERE tvdb_id = ? AND media_type = 'episode' AND season_number = ? AND episode_number = ? AND status NOT IN ('denied', 'available')
LIMIT 1;

-- name: ListRequests :many
SELECT * FROM requests ORDER BY created_at DESC;

-- name: ListRequestsByStatus :many
SELECT * FROM requests
WHERE status = ?
ORDER BY created_at DESC;

-- name: ListRequestsByUser :many
SELECT * FROM requests
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: ListRequestsByUserAndStatus :many
SELECT * FROM requests
WHERE user_id = ? AND status = ?
ORDER BY created_at DESC;

-- name: ListRequestsByMediaType :many
SELECT * FROM requests
WHERE media_type = ?
ORDER BY created_at DESC;

-- name: ListPendingRequests :many
SELECT * FROM requests
WHERE status = 'pending'
ORDER BY created_at ASC;

-- name: CreateRequest :one
INSERT INTO requests (
    user_id, media_type, tmdb_id, tvdb_id, title, year,
    season_number, episode_number, status, monitor_type, target_slot_id, poster_url, requested_seasons
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateRequestStatus :one
UPDATE requests SET
    status = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: ApproveRequest :one
UPDATE requests SET
    status = 'approved',
    approved_at = CURRENT_TIMESTAMP,
    approved_by = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DenyRequest :one
UPDATE requests SET
    status = 'denied',
    denied_reason = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: LinkRequestToMedia :one
UPDATE requests SET
    media_id = ?,
    status = 'available',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteRequest :exec
DELETE FROM requests WHERE id = ?;

-- name: CountRequests :one
SELECT COUNT(*) FROM requests;

-- name: CountRequestsByStatus :one
SELECT COUNT(*) FROM requests WHERE status = ?;

-- name: CountRequestsByUser :one
SELECT COUNT(*) FROM requests WHERE user_id = ?;

-- name: CountUserAutoApprovedInPeriod :one
SELECT COUNT(*) FROM requests
WHERE user_id = ?
  AND media_type = ?
  AND approved_at >= ?
  AND approved_by IS NULL;

-- name: ListRequestsByMediaID :many
SELECT * FROM requests
WHERE media_id = ? AND media_type = ?
ORDER BY created_at DESC;

-- name: ListRequestsByMediaIDAndSeason :many
SELECT * FROM requests
WHERE media_id = ? AND media_type = ? AND season_number = ?
ORDER BY created_at DESC;

-- name: GetActiveRequestByTmdbID :one
-- Used for availability display - includes 'available' status to show completed requests
SELECT * FROM requests
WHERE tmdb_id = ? AND media_type = ? AND status NOT IN ('denied', 'cancelled')
LIMIT 1;

-- name: GetActiveRequestByTvdbID :one
-- Used for availability display - includes 'available' status to show completed requests
SELECT * FROM requests
WHERE tvdb_id = ? AND media_type = ? AND status NOT IN ('denied', 'cancelled')
LIMIT 1;

-- name: GetActiveRequestByTvdbIDAndSeason :one
-- Used for availability display - includes 'available' status to show completed requests
SELECT * FROM requests
WHERE tvdb_id = ? AND media_type = 'season' AND season_number = ? AND status NOT IN ('denied', 'cancelled')
LIMIT 1;

-- name: GetActiveRequestByTvdbIDAndEpisode :one
-- Used for availability display - includes 'available' status to show completed requests
SELECT * FROM requests
WHERE tvdb_id = ? AND media_type = 'episode' AND season_number = ? AND episode_number = ? AND status NOT IN ('denied', 'cancelled')
LIMIT 1;

-- name: ListActiveSeriesRequestsByTvdbID :many
-- Used for status tracking - find active series requests by TVDB ID
SELECT * FROM requests
WHERE tvdb_id = ? AND media_type = 'series' AND status IN ('downloading', 'approved')
ORDER BY created_at DESC;
