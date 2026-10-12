-- name: GetLibrary :one
SELECT * FROM library WHERE id = ?;
-- +goose Down
