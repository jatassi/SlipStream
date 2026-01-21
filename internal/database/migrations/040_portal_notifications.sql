-- +goose Up
-- In-app notification messages for portal users
CREATE TABLE portal_notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE,
    request_id INTEGER REFERENCES requests(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('approved', 'denied', 'available')),
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    read INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_portal_notifications_user_id ON portal_notifications(user_id);
CREATE INDEX idx_portal_notifications_user_read ON portal_notifications(user_id, read);

-- +goose Down
DROP INDEX IF EXISTS idx_portal_notifications_user_read;
DROP INDEX IF EXISTS idx_portal_notifications_user_id;
DROP TABLE IF EXISTS portal_notifications;
