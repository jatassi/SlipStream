-- name: GetConversion :one
SELECT * FROM conversions_conversion WHERE id = ?;
