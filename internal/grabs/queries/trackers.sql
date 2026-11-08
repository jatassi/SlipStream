-- name: ListStatedFloors :many
SELECT g.tracker_id, g.min_ratio, g.min_seed_seconds, g.created_at FROM grabs_grab g
WHERE g.tracker_id IS NOT NULL AND (g.min_ratio IS NOT NULL OR g.min_seed_seconds IS NOT NULL)
    AND NOT EXISTS (
        SELECT 1 FROM grabs_grab n
        WHERE n.tracker_id = g.tracker_id AND (n.min_ratio IS NOT NULL OR n.min_seed_seconds IS NOT NULL)
            AND (n.created_at > g.created_at OR (n.created_at = g.created_at AND n.id > g.id))
    )
ORDER BY g.tracker_id;
