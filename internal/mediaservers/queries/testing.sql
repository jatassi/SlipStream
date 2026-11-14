-- mstest sets up with these a state the Controller reaches through steps
-- a test doesn't drive.

-- name: MarkShownScanned :exec
UPDATE mediaservers_mapping SET scanned_at = sqlc.arg(at) WHERE shown = 1;
