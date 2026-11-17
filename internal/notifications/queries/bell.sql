-- name: GetBell :one
SELECT * FROM notifications_bell;

-- name: SetBell :exec
UPDATE notifications_bell SET choices = sqlc.arg(choices), warnings = sqlc.arg(warnings);

-- name: InsertShown :one
INSERT INTO notifications_shown (event_id, kind, at, shown_at, items)
VALUES (sqlc.arg(event_id), sqlc.arg(kind), sqlc.arg(at), sqlc.arg(shown_at), sqlc.narg(items))
RETURNING seq;

-- name: ListFeed :many
SELECT seq, event_id, kind, at, shown_at, items FROM notifications_shown
WHERE seq < sqlc.arg(before)
ORDER BY seq DESC
LIMIT sqlc.arg(limit);

-- name: CountAbove :one
SELECT COUNT(*) FROM (
    SELECT seq FROM notifications_shown WHERE seq > sqlc.arg(seq) ORDER BY seq LIMIT sqlc.arg(cap)
);

-- name: NewestSeq :one
SELECT seq FROM notifications_shown ORDER BY seq DESC LIMIT 1;

-- name: NewestSeqAt :one
SELECT seq FROM notifications_shown WHERE shown_at <= sqlc.arg(at) ORDER BY shown_at DESC, seq DESC LIMIT 1;

-- name: GetSeen :one
SELECT seq FROM notifications_seen WHERE account_id = sqlc.arg(account_id);

-- name: UpsertSeen :exec
INSERT INTO notifications_seen (account_id, seq) VALUES (sqlc.arg(account_id), sqlc.arg(seq))
ON CONFLICT (account_id) DO UPDATE SET seq = MAX(seq, excluded.seq);
