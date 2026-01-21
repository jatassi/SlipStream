-- +goose Up
-- Request watchers for notification subscriptions
CREATE TABLE request_watchers (
    request_id INTEGER NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (request_id, user_id)
);

CREATE INDEX idx_request_watchers_user_id ON request_watchers(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_request_watchers_user_id;
DROP TABLE IF EXISTS request_watchers;
