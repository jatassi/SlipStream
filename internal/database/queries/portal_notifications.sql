-- name: CreatePortalNotification :one
INSERT INTO portal_notifications (
    user_id, request_id, type, title, message
) VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListPortalNotifications :many
SELECT * FROM portal_notifications
WHERE user_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListUnreadPortalNotifications :many
SELECT * FROM portal_notifications
WHERE user_id = ? AND read = 0
ORDER BY created_at DESC;

-- name: CountUnreadPortalNotifications :one
SELECT COUNT(*) FROM portal_notifications
WHERE user_id = ? AND read = 0;

-- name: MarkPortalNotificationRead :exec
UPDATE portal_notifications SET read = 1
WHERE id = ? AND user_id = ?;

-- name: MarkAllPortalNotificationsRead :exec
UPDATE portal_notifications SET read = 1
WHERE user_id = ? AND read = 0;

-- name: DeletePortalNotification :exec
DELETE FROM portal_notifications WHERE id = ? AND user_id = ?;

-- name: DeleteOldPortalNotifications :exec
DELETE FROM portal_notifications
WHERE user_id = ? AND created_at < ?;
