-- name: ListLines :many
SELECT * FROM notifications_line ORDER BY connection_id;

-- name: UpsertLine :exec
INSERT INTO notifications_line (connection_id, state, state_at, reason, attempts, next_at, revision, last_sent_at, given_up)
VALUES (sqlc.arg(connection_id), sqlc.arg(state), sqlc.arg(state_at), sqlc.narg(reason), sqlc.arg(attempts), sqlc.arg(next_at), sqlc.arg(revision), sqlc.arg(last_sent_at), sqlc.arg(given_up))
ON CONFLICT (connection_id) DO UPDATE SET
    state = excluded.state, state_at = excluded.state_at, reason = excluded.reason, attempts = excluded.attempts,
    next_at = excluded.next_at, revision = excluded.revision, last_sent_at = excluded.last_sent_at, given_up = excluded.given_up;

-- name: AddGivenUp :exec
UPDATE notifications_line SET given_up = given_up + sqlc.arg(n) WHERE connection_id = sqlc.arg(connection_id);
