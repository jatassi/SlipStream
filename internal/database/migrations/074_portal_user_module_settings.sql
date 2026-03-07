-- +goose Up
-- +goose StatementBegin

PRAGMA foreign_keys = OFF;

-- 1. Create portal_user_module_settings
CREATE TABLE portal_user_module_settings (
    user_id INTEGER NOT NULL REFERENCES portal_users(id) ON DELETE CASCADE,
    module_type TEXT NOT NULL,
    quota_limit INTEGER,          -- NULL = use global default, 0 = unlimited
    quota_used INTEGER NOT NULL DEFAULT 0,
    quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL,
    period_start DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, module_type)
);

-- 2. Migrate quality profiles from portal_users + quotas from user_quotas
-- Movie rows: quality profile from portal_users, quota from user_quotas
INSERT INTO portal_user_module_settings (user_id, module_type, quota_limit, quota_used, quality_profile_id, period_start)
SELECT
    pu.id,
    'movie',
    uq.movies_limit,
    COALESCE(uq.movies_used, 0),
    pu.movie_quality_profile_id,
    COALESCE(uq.period_start, CURRENT_TIMESTAMP)
FROM portal_users pu
LEFT JOIN user_quotas uq ON pu.id = uq.user_id
WHERE pu.is_admin = 0;

-- TV rows: quality profile from portal_users, quota = min of seasons/episodes limits
INSERT INTO portal_user_module_settings (user_id, module_type, quota_limit, quota_used, quality_profile_id, period_start)
SELECT
    pu.id,
    'tv',
    CASE
        WHEN uq.seasons_limit IS NULL AND uq.episodes_limit IS NULL THEN NULL
        WHEN uq.seasons_limit IS NULL THEN uq.episodes_limit
        WHEN uq.episodes_limit IS NULL THEN uq.seasons_limit
        ELSE MIN(uq.seasons_limit, uq.episodes_limit)
    END,
    CASE
        WHEN uq.seasons_used IS NULL AND uq.episodes_used IS NULL THEN 0
        ELSE MAX(COALESCE(uq.seasons_used, 0), COALESCE(uq.episodes_used, 0))
    END,
    pu.tv_quality_profile_id,
    COALESCE(uq.period_start, CURRENT_TIMESTAMP)
FROM portal_users pu
LEFT JOIN user_quotas uq ON pu.id = uq.user_id
WHERE pu.is_admin = 0;

-- 3. Create portal_invitation_module_settings
CREATE TABLE portal_invitation_module_settings (
    invitation_id INTEGER NOT NULL REFERENCES portal_invitations(id) ON DELETE CASCADE,
    module_type TEXT NOT NULL,
    quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL,
    PRIMARY KEY (invitation_id, module_type)
);

-- 4. Migrate invitation quality profiles
INSERT INTO portal_invitation_module_settings (invitation_id, module_type, quality_profile_id)
SELECT id, 'movie', movie_quality_profile_id
FROM portal_invitations
WHERE movie_quality_profile_id IS NOT NULL;

INSERT INTO portal_invitation_module_settings (invitation_id, module_type, quality_profile_id)
SELECT id, 'tv', tv_quality_profile_id
FROM portal_invitations
WHERE tv_quality_profile_id IS NOT NULL;

-- 5. Drop user_quotas table
DROP TABLE IF EXISTS user_quotas;

-- 6. Recreate portal_users without per-module quality profile columns
CREATE TABLE portal_users_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    auto_approve INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1,
    is_admin INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO portal_users_new (id, username, password_hash, auto_approve, enabled, is_admin, created_at, updated_at)
SELECT id, username, password_hash, auto_approve, enabled, is_admin, created_at, updated_at
FROM portal_users;

DROP TABLE portal_users;
ALTER TABLE portal_users_new RENAME TO portal_users;

CREATE INDEX idx_portal_users_username ON portal_users(username);
CREATE INDEX idx_portal_users_enabled ON portal_users(enabled);

-- 7. Recreate portal_invitations without per-module quality profile columns
CREATE TABLE portal_invitations_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    auto_approve INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO portal_invitations_new (id, username, token, expires_at, used_at, auto_approve, created_at)
SELECT id, username, token, expires_at, used_at, auto_approve, created_at
FROM portal_invitations;

DROP TABLE portal_invitations;
ALTER TABLE portal_invitations_new RENAME TO portal_invitations;

CREATE INDEX idx_portal_invitations_token ON portal_invitations(token);
CREATE INDEX idx_portal_invitations_username ON portal_invitations(username);

-- 8. Migrate global quota settings keys
INSERT OR REPLACE INTO settings (key, value)
SELECT
    'requests_default_tv_quota',
    CAST(MIN(
        CAST((SELECT value FROM settings WHERE key = 'requests_default_season_quota') AS INTEGER),
        CAST((SELECT value FROM settings WHERE key = 'requests_default_episode_quota') AS INTEGER)
    ) AS TEXT)
WHERE EXISTS (SELECT 1 FROM settings WHERE key = 'requests_default_season_quota')
   OR EXISTS (SELECT 1 FROM settings WHERE key = 'requests_default_episode_quota');

DELETE FROM settings WHERE key IN ('requests_default_season_quota', 'requests_default_episode_quota');

PRAGMA foreign_keys = ON;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Down migration is destructive — cannot fully reverse quota split

PRAGMA foreign_keys = OFF;

CREATE TABLE user_quotas (
    user_id INTEGER PRIMARY KEY REFERENCES portal_users(id) ON DELETE CASCADE,
    movies_limit INTEGER,
    seasons_limit INTEGER,
    episodes_limit INTEGER,
    movies_used INTEGER NOT NULL DEFAULT 0,
    seasons_used INTEGER NOT NULL DEFAULT 0,
    episodes_used INTEGER NOT NULL DEFAULT 0,
    period_start DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Restore user_quotas from portal_user_module_settings
INSERT INTO user_quotas (user_id, movies_limit, movies_used, seasons_limit, seasons_used, episodes_limit, episodes_used, period_start)
SELECT
    m.user_id,
    m.quota_limit,
    m.quota_used,
    t.quota_limit,
    t.quota_used,
    t.quota_limit,
    t.quota_used,
    m.period_start
FROM portal_user_module_settings m
LEFT JOIN portal_user_module_settings t ON m.user_id = t.user_id AND t.module_type = 'tv'
WHERE m.module_type = 'movie';

-- Recreate portal_users with quality profile columns
ALTER TABLE portal_users ADD COLUMN movie_quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL;
ALTER TABLE portal_users ADD COLUMN tv_quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL;

UPDATE portal_users SET
    movie_quality_profile_id = (SELECT quality_profile_id FROM portal_user_module_settings WHERE user_id = portal_users.id AND module_type = 'movie'),
    tv_quality_profile_id = (SELECT quality_profile_id FROM portal_user_module_settings WHERE user_id = portal_users.id AND module_type = 'tv');

-- Recreate portal_invitations with quality profile columns
ALTER TABLE portal_invitations ADD COLUMN movie_quality_profile_id INTEGER;
ALTER TABLE portal_invitations ADD COLUMN tv_quality_profile_id INTEGER;

UPDATE portal_invitations SET
    movie_quality_profile_id = (SELECT quality_profile_id FROM portal_invitation_module_settings WHERE invitation_id = portal_invitations.id AND module_type = 'movie'),
    tv_quality_profile_id = (SELECT quality_profile_id FROM portal_invitation_module_settings WHERE invitation_id = portal_invitations.id AND module_type = 'tv');

DROP TABLE IF EXISTS portal_user_module_settings;
DROP TABLE IF EXISTS portal_invitation_module_settings;

-- Restore old settings keys
INSERT OR REPLACE INTO settings (key, value)
SELECT 'requests_default_season_quota', value FROM settings WHERE key = 'requests_default_tv_quota';
INSERT OR REPLACE INTO settings (key, value)
SELECT 'requests_default_episode_quota', value FROM settings WHERE key = 'requests_default_tv_quota';
DELETE FROM settings WHERE key = 'requests_default_tv_quota';

PRAGMA foreign_keys = ON;

-- +goose StatementEnd
