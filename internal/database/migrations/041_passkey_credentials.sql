-- +goose Up
CREATE TABLE passkey_credentials (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    credential_id BLOB NOT NULL UNIQUE,
    public_key BLOB NOT NULL,
    attestation_type TEXT NOT NULL,
    transport TEXT,  -- JSON array of transports: ["internal", "usb", "ble", "nfc"]
    flags_user_present BOOLEAN NOT NULL DEFAULT FALSE,
    flags_user_verified BOOLEAN NOT NULL DEFAULT FALSE,
    flags_backup_eligible BOOLEAN NOT NULL DEFAULT FALSE,
    flags_backup_state BOOLEAN NOT NULL DEFAULT FALSE,
    sign_count INTEGER NOT NULL DEFAULT 0,
    name TEXT NOT NULL,  -- User-friendly label like "MacBook Pro Touch ID"
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES portal_users(id) ON DELETE CASCADE
);

CREATE INDEX idx_passkey_credentials_user_id ON passkey_credentials(user_id);
CREATE INDEX idx_passkey_credentials_credential_id ON passkey_credentials(credential_id);

-- +goose Down
DROP INDEX IF EXISTS idx_passkey_credentials_credential_id;
DROP INDEX IF EXISTS idx_passkey_credentials_user_id;
DROP TABLE IF EXISTS passkey_credentials;
