-- +goose Up
CREATE TABLE identity_settings_new (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    pin_wait_after INTEGER,
    pin_wait_seconds INTEGER,
    pin_off_after INTEGER,
    address_failures INTEGER,
    address_window_seconds INTEGER,
    reconfirm_window_seconds INTEGER,
    session_lifetime_seconds INTEGER,
    signup_link_lifetime_seconds INTEGER,
    reset_link_lifetime_seconds INTEGER,
    email_link_lifetime_seconds INTEGER,
    address TEXT NOT NULL
) STRICT;

INSERT INTO identity_settings_new
SELECT
    id,
    NULLIF(pin_wait_after, 5),
    NULLIF(pin_wait_seconds, 900),
    NULLIF(pin_off_after, 10),
    NULLIF(address_failures, 30),
    NULLIF(address_window_seconds, 900),
    NULLIF(reconfirm_window_seconds, 900),
    NULLIF(session_lifetime_seconds, 2592000),
    NULLIF(signup_link_lifetime_seconds, 604800),
    NULLIF(reset_link_lifetime_seconds, 86400),
    NULLIF(email_link_lifetime_seconds, 86400),
    address
FROM identity_settings;

DROP TABLE identity_settings;

ALTER TABLE identity_settings_new RENAME TO identity_settings;
