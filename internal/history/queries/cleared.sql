-- name: GetCleared :one
SELECT through FROM history_cleared;

-- name: RaiseCleared :exec
UPDATE history_cleared SET through = MAX(through, sqlc.arg(through)), at = sqlc.arg(at);
