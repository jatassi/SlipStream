-- +goose Up
ALTER TABLE identity_account ADD COLUMN disabled_at INTEGER;
ALTER TABLE identity_account ADD COLUMN pin_failures INTEGER NOT NULL DEFAULT 0 CHECK (pin_failures >= 0);
ALTER TABLE identity_account ADD COLUMN pin_wait_until INTEGER;
ALTER TABLE identity_account ADD COLUMN pin_disabled_at INTEGER;
ALTER TABLE identity_account ADD COLUMN key_prompt TEXT NOT NULL DEFAULT 'ask' CHECK (key_prompt IN ('ask', 'never'));
ALTER TABLE identity_account ADD COLUMN last_sign_in_at INTEGER;

ALTER TABLE identity_session ADD COLUMN confirmed_at INTEGER NOT NULL DEFAULT 0;
UPDATE identity_session SET confirmed_at = created_at;
ALTER TABLE identity_session ADD COLUMN device TEXT NOT NULL DEFAULT '';
ALTER TABLE identity_session ADD COLUMN pin_required INTEGER NOT NULL DEFAULT 0 CHECK (pin_required IN (0, 1));
ALTER TABLE identity_session ADD COLUMN prompted INTEGER NOT NULL DEFAULT 0 CHECK (prompted IN (0, 1));

CREATE TABLE identity_passkey (
    id BLOB PRIMARY KEY NOT NULL,
    account_id BLOB NOT NULL REFERENCES identity_account (id) ON DELETE CASCADE,
    relying_party TEXT NOT NULL,
    credential_id BLOB NOT NULL UNIQUE,
    public_key BLOB NOT NULL,
    sign_count INTEGER NOT NULL,
    aaguid BLOB NOT NULL,
    transports TEXT NOT NULL,
    backup_eligible INTEGER NOT NULL CHECK (backup_eligible IN (0, 1)),
    backup_state INTEGER NOT NULL CHECK (backup_state IN (0, 1)),
    name TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    last_used_at INTEGER
) STRICT;

CREATE INDEX identity_passkey_account_id ON identity_passkey (account_id, relying_party);

CREATE TABLE identity_link (
    id BLOB PRIMARY KEY NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('signup', 'reset', 'email')),
    token_hash BLOB NOT NULL UNIQUE,
    token_sealed BLOB,
    email TEXT NOT NULL,
    account_id BLOB REFERENCES identity_account (id) ON DELETE CASCADE,
    remove_passkeys INTEGER NOT NULL DEFAULT 0 CHECK (remove_passkeys IN (0, 1)),
    issued_by BLOB REFERENCES identity_account (id) ON DELETE SET NULL,
    issued_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    ended_at INTEGER,
    end_reason TEXT CHECK (end_reason IN ('used', 'withdrawn')),
    CHECK ((ended_at IS NULL) = (end_reason IS NULL)),
    CHECK (token_sealed IS NULL OR (kind = 'signup' AND ended_at IS NULL)),
    CHECK (kind = 'signup' OR account_id IS NOT NULL)
) STRICT;

CREATE INDEX identity_link_live ON identity_link (kind, email) WHERE ended_at IS NULL;

CREATE INDEX identity_link_account_id ON identity_link (account_id);

CREATE INDEX identity_link_issued_by ON identity_link (issued_by);

DROP TABLE identity_settings;

CREATE TABLE identity_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    pin_wait_after INTEGER NOT NULL CHECK (pin_wait_after BETWEEN 3 AND 10),
    pin_wait_seconds INTEGER NOT NULL CHECK (pin_wait_seconds BETWEEN 60 AND 86400),
    pin_off_after INTEGER NOT NULL CHECK (pin_off_after BETWEEN 4 AND 20),
    address_failures INTEGER NOT NULL CHECK (address_failures BETWEEN 10 AND 1000),
    address_window_seconds INTEGER NOT NULL CHECK (address_window_seconds BETWEEN 60 AND 86400),
    reconfirm_window_seconds INTEGER NOT NULL CHECK (reconfirm_window_seconds BETWEEN 60 AND 86400),
    session_lifetime_seconds INTEGER NOT NULL CHECK (session_lifetime_seconds BETWEEN 3600 AND 31536000),
    signup_link_lifetime_seconds INTEGER NOT NULL CHECK (signup_link_lifetime_seconds BETWEEN 60 AND 2592000),
    reset_link_lifetime_seconds INTEGER NOT NULL CHECK (reset_link_lifetime_seconds BETWEEN 60 AND 604800),
    email_link_lifetime_seconds INTEGER NOT NULL CHECK (email_link_lifetime_seconds BETWEEN 60 AND 604800),
    address TEXT NOT NULL,
    CHECK (pin_off_after > pin_wait_after)
) STRICT;

INSERT INTO identity_settings VALUES (1, 5, 900, 10, 30, 900, 900, 2592000, 604800, 86400, 86400, '');
