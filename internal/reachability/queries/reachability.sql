-- name: GetSettings :one
SELECT * FROM reachability_settings WHERE id = 1;

-- name: SetWarnAfter :exec
UPDATE reachability_settings SET warn_after_seconds = sqlc.narg(seconds) WHERE id = 1;

-- name: SetWatchEvery :exec
UPDATE reachability_settings SET watch_every_seconds = sqlc.narg(seconds) WHERE id = 1;

-- name: SetChangeWait :exec
UPDATE reachability_settings SET change_wait_seconds = sqlc.narg(seconds) WHERE id = 1;

-- name: SetSTUNServers :exec
UPDATE reachability_settings SET stun_servers = sqlc.narg(servers) WHERE id = 1;

-- name: InsertExposure :exec
INSERT INTO reachability_exposure (id, kind, server_id, public, wanted_at)
VALUES (sqlc.arg(id), sqlc.arg(kind), sqlc.narg(server_id), sqlc.narg(public), sqlc.arg(wanted_at));

-- name: GetExposure :one
SELECT * FROM reachability_exposure WHERE id = sqlc.arg(id);

-- name: ListExposures :many
SELECT * FROM reachability_exposure ORDER BY id;

-- name: SetPublic :execrows
UPDATE reachability_exposure SET public = sqlc.arg(public) WHERE id = sqlc.arg(id) AND kind = 'media-server';

-- name: DeleteExposure :execrows
DELETE FROM reachability_exposure WHERE id = sqlc.arg(id);

-- name: SetServerFacts :exec
UPDATE reachability_exposure SET server_facts = sqlc.arg(facts), server_observed_at = sqlc.arg(observed_at) WHERE id = sqlc.arg(id);

-- name: SetServerDigest :exec
UPDATE reachability_exposure SET server_digest = sqlc.arg(digest) WHERE id = sqlc.arg(id);

-- name: DismissUI :exec
INSERT INTO reachability_dismissed (kind, dismissed_at) VALUES ('ui-portal', sqlc.arg(at))
ON CONFLICT (kind) WHERE kind = 'ui-portal' DO NOTHING;

-- name: DismissServer :exec
INSERT INTO reachability_dismissed (kind, server_id, dismissed_at) VALUES ('media-server', sqlc.arg(server_id), sqlc.arg(at))
ON CONFLICT (server_id) WHERE server_id IS NOT NULL DO NOTHING;

-- name: DeleteDismissalUI :exec
DELETE FROM reachability_dismissed WHERE kind = 'ui-portal';

-- name: DeleteDismissalServer :exec
DELETE FROM reachability_dismissed WHERE server_id = sqlc.arg(server_id);

-- name: ListDismissals :many
SELECT * FROM reachability_dismissed ORDER BY kind, server_id;

-- name: GetResult :one
SELECT * FROM reachability_result WHERE exposure_id = sqlc.arg(exposure_id);

-- name: ListResults :many
SELECT * FROM reachability_result ORDER BY exposure_id;

-- name: UpsertResult :exec
INSERT INTO reachability_result (exposure_id, judged, verdict, tested, judged_at, failing_at, origin_node_id, tested_by_node_id)
VALUES (sqlc.arg(exposure_id), sqlc.arg(judged), sqlc.arg(verdict), sqlc.arg(tested), sqlc.arg(judged_at), sqlc.narg(failing_at),
    sqlc.narg(origin_node_id), sqlc.narg(tested_by_node_id))
ON CONFLICT (exposure_id) DO UPDATE SET judged = excluded.judged, verdict = excluded.verdict, tested = excluded.tested,
    judged_at = excluded.judged_at, failing_at = excluded.failing_at, origin_node_id = excluded.origin_node_id,
    tested_by_node_id = excluded.tested_by_node_id;

-- name: GetNode :one
SELECT * FROM reachability_node WHERE node_id = sqlc.arg(node_id);

-- name: ListNodes :many
SELECT * FROM reachability_node ORDER BY node_id;

-- name: UpsertNode :exec
-- The newest pass wins whatever order two units commit in.
INSERT INTO reachability_node (node_id, facts, observed_at, public4, prefixes6, previous4, previous6)
VALUES (sqlc.arg(node_id), sqlc.arg(facts), sqlc.arg(observed_at), sqlc.narg(public4), sqlc.arg(prefixes6), sqlc.narg(previous4), sqlc.arg(previous6))
ON CONFLICT (node_id) DO UPDATE SET facts = excluded.facts, observed_at = excluded.observed_at, public4 = excluded.public4,
    prefixes6 = excluded.prefixes6, previous4 = excluded.previous4, previous6 = excluded.previous6
WHERE reachability_node.observed_at < excluded.observed_at;

-- name: DeleteNodes :exec
DELETE FROM reachability_node WHERE node_id IN (sqlc.slice(ids));

-- name: ListNodeSettings :many
SELECT * FROM reachability_node_setting ORDER BY node_id;

-- name: GetNodeSetting :one
SELECT * FROM reachability_node_setting WHERE node_id = sqlc.arg(node_id);

-- name: SetMapPorts :exec
INSERT INTO reachability_node_setting (node_id, map_ports) VALUES (sqlc.arg(node_id), sqlc.narg(map_ports))
ON CONFLICT (node_id) DO UPDATE SET map_ports = excluded.map_ports;

-- name: SetStableIPv6 :exec
INSERT INTO reachability_node_setting (node_id, stable_ipv6) VALUES (sqlc.arg(node_id), sqlc.narg(stable_ipv6))
ON CONFLICT (node_id) DO UPDATE SET stable_ipv6 = excluded.stable_ipv6;

-- name: DeleteNodeSettings :exec
DELETE FROM reachability_node_setting WHERE node_id IN (sqlc.slice(ids));
