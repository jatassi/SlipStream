-- name: GetSlotRootFolder :one
SELECT * FROM slot_root_folders
WHERE slot_id = ? AND module_type = ? LIMIT 1;

-- name: ListSlotRootFolders :many
SELECT * FROM slot_root_folders
WHERE slot_id = ?
ORDER BY module_type;

-- name: UpsertSlotRootFolder :one
INSERT INTO slot_root_folders (slot_id, module_type, root_folder_id)
VALUES (?, ?, ?)
ON CONFLICT(slot_id, module_type) DO UPDATE SET
    root_folder_id = excluded.root_folder_id
RETURNING *;

-- name: DeleteSlotRootFolder :exec
DELETE FROM slot_root_folders WHERE slot_id = ? AND module_type = ?;

-- name: DeleteSlotRootFolders :exec
DELETE FROM slot_root_folders WHERE slot_id = ?;
