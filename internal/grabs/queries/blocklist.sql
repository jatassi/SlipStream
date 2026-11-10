-- name: IsBlocked :one
SELECT EXISTS (SELECT 1 FROM grabs_blocklist WHERE info_hash = sqlc.arg(info_hash));
