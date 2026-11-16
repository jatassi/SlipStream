-- name: GetBell :one
SELECT * FROM notifications_bell;

-- name: SetBell :exec
UPDATE notifications_bell SET choices = sqlc.arg(choices), warnings = sqlc.arg(warnings);

-- name: InsertShown :one
INSERT INTO notifications_shown (event_id, shown_at, items)
VALUES (sqlc.arg(event_id), sqlc.arg(shown_at), sqlc.narg(items))
RETURNING seq;

-- name: ListFeed :many
SELECT s.seq, s.shown_at, s.items, e.id, e.kind, e.at, e.subject FROM notifications_shown s
JOIN notifications_event e ON e.id = s.event_id
WHERE s.seq < sqlc.arg(before)
ORDER BY s.seq DESC
LIMIT sqlc.arg(limit);

-- name: CountAbove :one
SELECT COUNT(*) FROM (
    SELECT seq FROM notifications_shown WHERE seq > sqlc.arg(seq) ORDER BY seq LIMIT sqlc.arg(cap)
);

-- name: NewestSeq :one
SELECT COALESCE(MAX(seq), 0) FROM notifications_shown;

-- name: NewestSeqAt :one
SELECT seq FROM notifications_shown WHERE shown_at <= sqlc.arg(at) ORDER BY shown_at DESC, seq DESC LIMIT 1;

-- name: GetSeen :one
SELECT seq FROM notifications_seen WHERE account_id = sqlc.arg(account_id);

-- name: UpsertSeen :exec
INSERT INTO notifications_seen (account_id, seq) VALUES (sqlc.arg(account_id), sqlc.arg(seq))
ON CONFLICT (account_id) DO UPDATE SET seq = MAX(seq, excluded.seq);
