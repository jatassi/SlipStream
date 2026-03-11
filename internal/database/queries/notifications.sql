-- name: GetNotification :one
SELECT * FROM notifications WHERE id = ? LIMIT 1;

-- name: ListNotifications :many
SELECT * FROM notifications ORDER BY name;

-- name: ListEnabledNotifications :many
SELECT * FROM notifications WHERE enabled = 1 ORDER BY name;

-- name: CreateNotification :one
INSERT INTO notifications (
    name, type, enabled, settings,
    event_toggles, include_health_warnings, tags
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateNotification :one
UPDATE notifications SET
    name = ?,
    type = ?,
    enabled = ?,
    settings = ?,
    event_toggles = ?,
    include_health_warnings = ?,
    tags = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteNotification :exec
DELETE FROM notifications WHERE id = ?;

-- name: CountNotifications :one
SELECT COUNT(*) FROM notifications;

-- Notification status queries for failure tracking

-- name: GetNotificationStatus :one
SELECT * FROM notification_status WHERE notification_id = ? LIMIT 1;

-- name: UpsertNotificationStatus :exec
INSERT INTO notification_status (notification_id, initial_failure, most_recent_failure, escalation_level, disabled_till)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(notification_id) DO UPDATE SET
    initial_failure = COALESCE(excluded.initial_failure, notification_status.initial_failure),
    most_recent_failure = excluded.most_recent_failure,
    escalation_level = excluded.escalation_level,
    disabled_till = excluded.disabled_till;

-- name: ClearNotificationStatus :exec
DELETE FROM notification_status WHERE notification_id = ?;

-- name: ListDisabledNotifications :many
SELECT n.*, ns.disabled_till
FROM notifications n
JOIN notification_status ns ON n.id = ns.notification_id
WHERE ns.disabled_till IS NOT NULL AND ns.disabled_till > CURRENT_TIMESTAMP;
