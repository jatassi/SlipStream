-- name: ListServerUserIDs :many
SELECT server_user_id FROM mediaservers_member_access
WHERE server_id = sqlc.arg(server_id) AND server_user_id IS NOT NULL
ORDER BY server_user_id;
