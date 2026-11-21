-- name: GetState :one
SELECT * FROM reconciling_state WHERE id = 1;

-- name: InsertState :exec
INSERT INTO reconciling_state (id, restore_id, started_at, cause, backup_name, backup_taken_at, new_machine, rehome_saved_at)
VALUES (1, sqlc.arg(restore_id), sqlc.arg(started_at), sqlc.arg(cause), sqlc.arg(backup_name), sqlc.arg(backup_taken_at),
    sqlc.arg(new_machine), sqlc.narg(rehome_saved_at));

-- name: DeleteState :exec
DELETE FROM reconciling_state;

-- name: DeleteNodes :exec
DELETE FROM reconciling_node;

-- name: DeleteRoots :exec
DELETE FROM reconciling_root;

-- name: InsertNode :exec
-- A Node already keeping keeps its round.
INSERT INTO reconciling_node (node_id, cause, since, round)
VALUES (sqlc.arg(node_id), sqlc.arg(cause), sqlc.arg(since), 1)
ON CONFLICT (node_id) DO NOTHING;

-- name: GetNode :one
SELECT * FROM reconciling_node WHERE node_id = sqlc.arg(node_id);

-- name: ListPending :many
SELECT node_id FROM reconciling_node WHERE adopted_at IS NULL ORDER BY node_id;

-- name: ListNodes :many
SELECT * FROM reconciling_node ORDER BY node_id;

-- name: ListReady :many
-- The Nodes whose round is in and taken: those not yet adopted, and those
-- with a torrent whose move waits on another Node.
SELECT n.node_id, n.adopted_at IS NOT NULL AS adopted FROM reconciling_node n
WHERE n.reported_at IS NOT NULL AND n.refused_at IS NULL
  AND (n.adopted_at IS NULL OR EXISTS (SELECT 1 FROM reconciling_held h WHERE h.node_id = n.node_id AND h.settled_at IS NULL))
ORDER BY n.node_id;

-- name: CountFinished :one
-- How many Nodes Reconciling still waits on.
SELECT count(*) FROM reconciling_node WHERE adopted_at IS NULL AND continued_without_at IS NULL;

-- name: SeePage :execrows
INSERT INTO reconciling_page (node_id, round, page) VALUES (sqlc.arg(node_id), sqlc.arg(round), sqlc.arg(page))
ON CONFLICT DO NOTHING;

-- name: SetRound :exec
UPDATE reconciling_node SET pages = sqlc.arg(pages), pages_seen = sqlc.arg(pages_seen), held = sqlc.arg(held),
    reported_at = sqlc.narg(reported_at)
WHERE node_id = sqlc.arg(node_id);

-- name: Refuse :exec
UPDATE reconciling_node SET refused_at = sqlc.arg(at), refused = sqlc.arg(refused)
WHERE node_id = sqlc.arg(node_id) AND refused_at IS NULL;

-- name: SetAdopted :exec
UPDATE reconciling_node SET adopted_at = sqlc.arg(at), granted_at = sqlc.narg(granted_at)
WHERE node_id = sqlc.arg(node_id) AND adopted_at IS NULL;

-- name: ContinueWithout :exec
UPDATE reconciling_node SET continued_without_at = sqlc.arg(at)
WHERE node_id = sqlc.arg(node_id) AND adopted_at IS NULL AND continued_without_at IS NULL;

-- name: CountHeldOf :one
SELECT count(*) FROM reconciling_held WHERE node_id = sqlc.arg(node_id) AND info_hash IN (sqlc.slice(hashes));

-- name: PutHeld :exec
-- A page sent again refreshes a torrent's facts and keeps its decision.
INSERT INTO reconciling_held (node_id, info_hash, client_id, name, size, done_ppm, private, seed_seconds, hosts, save_folder)
VALUES (sqlc.arg(node_id), sqlc.arg(info_hash), sqlc.narg(client_id), sqlc.narg(name), sqlc.narg(size), sqlc.arg(done_ppm),
    sqlc.arg(private), sqlc.arg(seed_seconds), sqlc.arg(hosts), sqlc.narg(save_folder))
ON CONFLICT (node_id, info_hash) DO UPDATE SET
    client_id = excluded.client_id, name = excluded.name, size = excluded.size, done_ppm = excluded.done_ppm,
    private = excluded.private, seed_seconds = excluded.seed_seconds, hosts = excluded.hosts, save_folder = excluded.save_folder;

-- name: ListOpenHeld :many
SELECT * FROM reconciling_held
WHERE node_id = sqlc.arg(node_id) AND settled_at IS NULL AND info_hash > sqlc.arg(after)
ORDER BY info_hash
LIMIT sqlc.arg(page);

-- name: CountUndecided :one
-- How many of a Node's torrents still wait for a decision other than a
-- wait.
SELECT count(*) FROM reconciling_held
WHERE node_id = sqlc.arg(node_id) AND settled_at IS NULL AND (decision IS NULL OR decision <> 'wait');

-- name: Decide :exec
UPDATE reconciling_held SET decision = sqlc.arg(decision), settled_at = sqlc.narg(settled_at)
WHERE node_id = sqlc.arg(node_id) AND info_hash = sqlc.arg(info_hash) AND settled_at IS NULL;

-- name: ListKeepHashes :many
SELECT info_hash FROM reconciling_held
WHERE node_id = sqlc.arg(node_id) AND decision IN ('wait', 'conflict', 'stranded')
ORDER BY info_hash;

-- name: Holds :one
SELECT count(*) > 0 FROM reconciling_held WHERE node_id = sqlc.arg(node_id) AND info_hash = sqlc.arg(info_hash);

-- name: ListWaiting :many
-- The Nodes whose held torrent waits for the Node that records it.
SELECT node_id, client_id FROM reconciling_held
WHERE info_hash = sqlc.arg(info_hash) AND decision = 'wait' AND settled_at IS NULL
ORDER BY node_id;

-- name: InsertRoot :exec
INSERT INTO reconciling_root (root_id, saved_at) VALUES (sqlc.arg(root_id), sqlc.narg(saved_at));

-- name: ListRoots :many
SELECT * FROM reconciling_root ORDER BY root_id;

-- name: CountUnscanned :one
SELECT count(*) FROM reconciling_root WHERE scanned_at IS NULL AND continued_without_at IS NULL;

-- name: PutReview :exec
INSERT INTO reconciling_torrent (torrent_id, node_id, reason, since)
VALUES (sqlc.arg(torrent_id), sqlc.arg(node_id), sqlc.arg(reason), sqlc.arg(since))
ON CONFLICT (torrent_id) DO NOTHING;

-- name: ListReview :many
SELECT * FROM reconciling_torrent ORDER BY reason, files_found, since, torrent_id;

-- name: DeleteReview :exec
DELETE FROM reconciling_torrent WHERE torrent_id = sqlc.arg(torrent_id);
