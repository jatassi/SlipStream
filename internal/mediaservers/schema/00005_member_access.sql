-- +goose Up
-- S43 (Member access to Media Servers).
ALTER TABLE mediaservers_server ADD COLUMN members_see INTEGER NOT NULL DEFAULT 0 CHECK (members_see IN (0, 1));
-- Where a Member's app connects to a Jellyfin or Emby server. Plex apps
-- find the server through the Member's Plex account.
ALTER TABLE mediaservers_server ADD COLUMN member_address TEXT CHECK (member_address IS NULL OR (member_address <> '' AND kind <> 'plex'));

-- plex.tv's id of the Admin's Plex account, filled at sign-in and by the
-- next renewal. It stays outside the CHECKs that tie jwt_sealed's nullness.
ALTER TABLE mediaservers_plex_account ADD COLUMN plex_id INTEGER;

-- The Libraries for Members of a server, keyed as
-- mediaservers_mapping.library_key. name is the library's name when the
-- Admin saved it, for the by-hand copy and plex_libraries_short while the
-- server may be unreached.
CREATE TABLE mediaservers_member_library (
    server_id BLOB NOT NULL REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    library_key TEXT NOT NULL CHECK (library_key <> ''),
    name TEXT NOT NULL CHECK (name <> ''),
    PRIMARY KEY (server_id, library_key)
) STRICT, WITHOUT ROWID;

-- One row per Account that started a Plex sign-in, kept for the Account's
-- life.
CREATE TABLE mediaservers_member_plex (
    account_id BLOB PRIMARY KEY NOT NULL REFERENCES identity_account (id) ON DELETE CASCADE,
    -- Made at the Member's first PIN and reused for every later one.
    client_identifier TEXT NOT NULL CHECK (client_identifier <> ''),
    plex_id INTEGER UNIQUE,
    plex_uuid TEXT,
    username TEXT,
    email TEXT,
    signed_in_at INTEGER,
    -- Sealed per row. Cleared in the Write that leaves the Member no
    -- waiting or shared row.
    token_sealed BLOB,
    CHECK ((plex_id IS NULL) = (plex_uuid IS NULL)),
    CHECK ((plex_id IS NULL) = (username IS NULL)),
    CHECK ((plex_id IS NULL) = (email IS NULL)),
    CHECK ((plex_id IS NULL) = (signed_in_at IS NULL)),
    CHECK (token_sealed IS NULL OR plex_id IS NOT NULL)
) STRICT;

CREATE TABLE mediaservers_member_access (
    id BLOB PRIMARY KEY NOT NULL,
    server_id BLOB NOT NULL REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    -- How the server's kind gives access, written at the offer. A server's
    -- kind never changes, so the CHECKs below name the states and columns
    -- each way holds.
    way TEXT NOT NULL CHECK (way IN ('plex_tv', 'users')),
    -- CASCADE: Leaving sets it to NULL on each row it couldn't settle before
    -- identity deletes the Account, so only those rows outlive it.
    account_id BLOB REFERENCES identity_account (id) ON DELETE CASCADE,
    -- Kept so a removal that outlives the Account still names the person.
    member_name TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('offered', 'waiting', 'shared', 'making', 'active', 'removing', 'removed')),
    changed_at INTEGER NOT NULL,
    offered_by BLOB REFERENCES identity_account (id) ON DELETE SET NULL,
    offered_at INTEGER NOT NULL,
    -- Plex. plex_id and plex_uuid are the Member's Plex account, copied at
    -- sign-in so a removal works after the Account is gone.
    plex_id INTEGER,
    plex_uuid TEXT CHECK (plex_uuid <> ''),
    -- plex.tv's id for the share, which is also the pending invite's id the
    -- Member accepts. TEXT with no shape CHECK: SlipStream only copies it
    -- back to plex.tv.
    shared_server_id TEXT CHECK (shared_server_id <> ''),
    by_hand INTEGER NOT NULL DEFAULT 0 CHECK (by_hand IN (0, 1)),
    plex_home INTEGER NOT NULL DEFAULT 0 CHECK (plex_home IN (0, 1)),
    -- Jellyfin and Emby
    server_user_id TEXT CHECK (server_user_id <> ''),
    username TEXT CHECK (username <> ''),
    problem_code TEXT CHECK (problem_code IN ('plex_refused', 'plex_libraries_short', 'username_taken', 'server_refused', 'media_server_unreached', 'failed')),
    problem_detail TEXT,
    UNIQUE (server_id, account_id),
    CHECK ((problem_code IS NULL) = (problem_detail IS NULL)),
    -- Only a Leave that couldn't finish outlives its Account.
    CHECK (account_id IS NOT NULL OR state IN ('removing', 'making')),
    -- Each way's states.
    CHECK (way = 'plex_tv' OR state NOT IN ('waiting', 'shared')),
    CHECK (way = 'users' OR state NOT IN ('making', 'removed')),
    -- Each way's columns.
    CHECK (way = 'plex_tv' OR (plex_id IS NULL AND plex_uuid IS NULL AND shared_server_id IS NULL AND plex_home = 0 AND by_hand = 0)),
    CHECK (way = 'users' OR (server_user_id IS NULL AND username IS NULL)),
    CHECK ((plex_id IS NULL) = (plex_uuid IS NULL)),
    CHECK (shared_server_id IS NULL OR plex_id IS NOT NULL),
    -- A Plex row past offered names its Plex account. A users row past
    -- making names its user.
    CHECK (way = 'users' OR state = 'offered' OR plex_id IS NOT NULL),
    CHECK (way = 'plex_tv' OR state NOT IN ('active', 'removing', 'removed') OR server_user_id IS NOT NULL),
    CHECK (state <> 'making' OR username IS NOT NULL),
    CHECK (server_user_id IS NULL OR username IS NOT NULL)
) STRICT;

CREATE INDEX mediaservers_member_access_account ON mediaservers_member_access (account_id);
CREATE INDEX mediaservers_member_access_state ON mediaservers_member_access (state) WHERE state IN ('waiting', 'shared', 'making', 'removing');
CREATE INDEX mediaservers_member_access_offered_by ON mediaservers_member_access (offered_by);
