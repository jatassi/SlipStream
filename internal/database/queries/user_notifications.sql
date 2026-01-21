-- name: GetUserNotification :one
SELECT * FROM user_notifications WHERE id = ? LIMIT 1;

-- name: ListUserNotifications :many
SELECT * FROM user_notifications
WHERE user_id = ?
ORDER BY name;

-- name: ListEnabledUserNotifications :many
SELECT * FROM user_notifications
WHERE user_id = ? AND enabled = 1
ORDER BY name;

-- name: ListUserNotificationsForAvailable :many
SELECT * FROM user_notifications
WHERE user_id = ? AND enabled = 1 AND on_available = 1
ORDER BY name;

-- name: ListUserNotificationsForApproved :many
SELECT * FROM user_notifications
WHERE user_id = ? AND enabled = 1 AND on_approved = 1
ORDER BY name;

-- name: ListUserNotificationsForDenied :many
SELECT * FROM user_notifications
WHERE user_id = ? AND enabled = 1 AND on_denied = 1
ORDER BY name;

-- name: CreateUserNotification :one
INSERT INTO user_notifications (
    user_id, type, name, settings, on_available, on_approved, on_denied, enabled
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateUserNotification :one
UPDATE user_notifications SET
    name = ?,
    type = ?,
    settings = ?,
    on_available = ?,
    on_approved = ?,
    on_denied = ?,
    enabled = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteUserNotification :exec
DELETE FROM user_notifications WHERE id = ?;

-- name: DeleteUserNotificationsByUser :exec
DELETE FROM user_notifications WHERE user_id = ?;

-- name: CountUserNotifications :one
SELECT COUNT(*) FROM user_notifications WHERE user_id = ?;
