-- +goose Up
-- S51 (Reachability diagnostic). The router mappings this Node asked for,
-- in its own local.db, so it can delete them after a restart. The mapper is
-- the only writer. Each gateway call is bracketed by its row: the row is
-- written asking (granted_at NULL, with PCP's nonce) before the call and
-- set granted or refused after it, and a mapping's row is deleted only
-- after the router deleted it or said it holds none. A crash anywhere
-- leaves a row the next pass, or the release at removal, finishes.
CREATE TABLE netfacts_mapping (
    port INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
    protocol TEXT NOT NULL CHECK (protocol = 'tcp'),
    method TEXT NOT NULL CHECK (method IN ('pcp', 'nat-pmp', 'upnp')),
    gateway TEXT NOT NULL,
    external_port INTEGER NOT NULL CHECK (external_port BETWEEN 1 AND 65535),
    nonce BLOB CHECK (nonce IS NULL OR length(nonce) = 12),
    asked_at INTEGER NOT NULL,
    granted_at INTEGER,
    -- NULL with granted_at set: a permanent lease (UPnP 725).
    expires_at INTEGER,
    refused TEXT CHECK (refused IS NULL OR refused IN ('other-port', 'not-authorized', 'refused')),
    CHECK (granted_at IS NOT NULL OR expires_at IS NULL),
    CHECK (granted_at IS NULL OR refused IS NULL),
    CHECK (method <> 'pcp' OR nonce IS NOT NULL),
    PRIMARY KEY (port, protocol)
) STRICT;
