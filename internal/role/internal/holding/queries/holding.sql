-- name: GetEpoch :one
SELECT epoch FROM role_epoch WHERE id = 1;

-- name: InsertEpoch :exec
INSERT INTO role_epoch (id, epoch, last_seq) VALUES (1, sqlc.arg(epoch), 0);

-- name: NextSeq :one
UPDATE role_epoch SET last_seq = last_seq + 1 WHERE id = 1 RETURNING last_seq;

-- name: InsertOutbox :exec
INSERT INTO role_outbox (seq, scope, payload) VALUES (sqlc.arg(seq), sqlc.arg(scope), sqlc.arg(payload));

-- name: ListOutboxAfter :many
SELECT * FROM role_outbox WHERE seq > sqlc.arg(seq) ORDER BY seq;

-- name: DeleteOutboxThrough :exec
DELETE FROM role_outbox WHERE seq <= sqlc.arg(seq);

-- name: ListIntents :many
SELECT * FROM role_intent ORDER BY scope;

-- name: PutIntent :exec
INSERT INTO role_intent (scope, version, payload) VALUES (sqlc.arg(scope), sqlc.arg(version), sqlc.arg(payload))
ON CONFLICT (scope) DO UPDATE SET version = excluded.version, payload = excluded.payload;

-- name: DeleteIntents :exec
DELETE FROM role_intent;
