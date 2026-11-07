-- name: GetSolver :one
SELECT * FROM indexers_node WHERE node_id = sqlc.arg(node_id);

-- name: PutSolver :exec
INSERT INTO indexers_node (node_id, solver_address, solver_timeout_seconds)
VALUES (sqlc.arg(node_id), sqlc.arg(solver_address), sqlc.narg(solver_timeout_seconds))
ON CONFLICT (node_id) DO UPDATE SET solver_address = excluded.solver_address, solver_timeout_seconds = excluded.solver_timeout_seconds;

-- name: DeleteSolver :exec
DELETE FROM indexers_node WHERE node_id = sqlc.arg(node_id);
