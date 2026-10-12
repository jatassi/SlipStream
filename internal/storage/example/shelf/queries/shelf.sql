-- name: InsertShelf :one
INSERT INTO shelf (name) VALUES (sqlc.arg(name)) RETURNING id;

-- name: CountShelves :one
SELECT count(*) FROM shelf;

-- name: ListShelfNames :many
SELECT name FROM shelf ORDER BY name;
