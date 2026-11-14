-- +goose Up
-- S35 (Media Servers). A Media Server names one reaching Node, which calls
-- address, and the Controller calls fallback_address while that Node is
-- away. Each column group has one writer: the API writes the
-- configuration, the Role's apply writes contact, the sender writes sends
-- and unhealthy, the renewal task writes the token. File ids carry no
-- foreign key, because files deletes a file's row in the Write that dooms
-- it, before its bytes go.
CREATE TABLE mediaservers_server (
    id BLOB PRIMARY KEY NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('plex', 'jellyfin', 'emby')),
    name TEXT NOT NULL UNIQUE CHECK (name <> ''),
    address TEXT NOT NULL CHECK (address <> ''),
    fallback_address TEXT CHECK (fallback_address <> ''),
    browser_address TEXT CHECK (browser_address <> ''),
    -- Plex's machineIdentifier, which is also its plex.tv resource's
    -- clientIdentifier, or the Jellyfin or Emby server Id.
    identity TEXT NOT NULL CHECK (identity <> ''),
    version TEXT NOT NULL,
    monitor_delay_ms INTEGER CHECK (monitor_delay_ms >= 0),
    reaching_node_id BLOB NOT NULL REFERENCES nodes_node (id),
    runs_on_node_id BLOB REFERENCES nodes_node (id),
    -- One stable DeviceId per Jellyfin or Emby server: Jellyfin revokes a
    -- token when the same DeviceId signs in again. Plex sends the account's
    -- client_identifier instead.
    device_id TEXT CHECK (device_id <> ''),
    secrets_sealed BLOB NOT NULL,
    settings_version INTEGER NOT NULL DEFAULT 1 CHECK (settings_version >= 1),
    fallback TEXT NOT NULL DEFAULT 'untested' CHECK (fallback IN ('available', 'none', 'untested')),
    checked_at INTEGER,
    contact TEXT NOT NULL DEFAULT 'unknown'
        CHECK (contact IN ('unknown', 'responding', 'not_responding', 'sign_in_refused', 'other_server')),
    contact_since INTEGER,
    contact_found TEXT,
    last_contact_at INTEGER,
    sends TEXT NOT NULL DEFAULT 'ok' CHECK (sends IN ('ok', 'failing')),
    sends_since INTEGER,
    last_send_at INTEGER,
    unhealthy_since INTEGER,
    unhealthy_reason TEXT CHECK (unhealthy_reason IN ('not_responding', 'sends_failing', 'sign_in_refused', 'other_server')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK ((unhealthy_since IS NULL) = (unhealthy_reason IS NULL)),
    CHECK ((contact = 'unknown') = (contact_since IS NULL)),
    CHECK ((contact = 'other_server') = (contact_found IS NOT NULL)),
    CHECK ((sends = 'failing') = (sends_since IS NOT NULL)),
    CHECK ((fallback = 'untested') = (checked_at IS NULL)),
    CHECK ((kind = 'plex') = (device_id IS NULL)),
    CHECK ((kind = 'plex') = (monitor_delay_ms IS NULL)),
    CHECK (kind <> 'plex' OR browser_address IS NULL)
) STRICT;

CREATE INDEX mediaservers_server_reaching ON mediaservers_server (reaching_node_id);

-- Every Node that fetched a server's credential, with the settings version
-- it got. The Controller holds every credential and has no row.
CREATE TABLE mediaservers_credential_fetch (
    server_id BLOB NOT NULL REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    node_id BLOB NOT NULL REFERENCES nodes_node (id),
    settings_version INTEGER NOT NULL CHECK (settings_version >= 1),
    first_fetched_at INTEGER NOT NULL,
    fetched_at INTEGER NOT NULL,
    PRIMARY KEY (server_id, node_id)
) STRICT;

-- One library root as one server sees it. A root with no row is unmapped.
-- confirmed_place is the root's place when the Admin confirmed: a re-home
-- bumps the root's, which unconfirms the mapping. The fresh folder rule
-- reads scanned_at, and writes scan_requested_at before it asks for a full
-- scan and scan_started_at once the server accepted it, so a retry never
-- starts a second scan.
CREATE TABLE mediaservers_mapping (
    server_id BLOB NOT NULL REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    root_id BLOB NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE,
    shown INTEGER NOT NULL CHECK (shown IN (0, 1)),
    server_folder TEXT CHECK (server_folder <> ''),
    library_key TEXT CHECK (library_key <> ''),
    library_name TEXT,
    library_folder TEXT CHECK (library_folder <> ''),
    guess INTEGER NOT NULL DEFAULT 0 CHECK (guess IN (0, 1)),
    confirmed_at INTEGER,
    confirmed_place INTEGER CHECK (confirmed_place > 0),
    scanned_at INTEGER,
    scan_requested_at INTEGER,
    scan_started_at INTEGER,
    PRIMARY KEY (server_id, root_id),
    CHECK (shown = 0 OR (server_folder IS NOT NULL AND library_key IS NOT NULL
        AND library_name IS NOT NULL AND library_folder IS NOT NULL)),
    CHECK ((confirmed_at IS NULL) = (confirmed_place IS NULL)),
    CHECK (scan_started_at IS NULL OR scan_requested_at IS NOT NULL)
) STRICT;

CREATE INDEX mediaservers_mapping_root ON mediaservers_mapping (root_id);

-- The one Plex account. Only the Controller reads it: the device key and
-- the account JWT never leave it.
CREATE TABLE mediaservers_plex_account (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    client_identifier TEXT NOT NULL CHECK (client_identifier <> ''),
    key_sealed BLOB NOT NULL,
    jwt_sealed BLOB,
    jwt_expires_at INTEGER,
    account_name TEXT,
    plex_pass INTEGER CHECK (plex_pass IN (0, 1)),
    signed_in_at INTEGER,
    renewed_at INTEGER,
    -- plex.tv refused a renewal. The next sign-in or renewal clears it.
    refused_at INTEGER,
    CHECK ((jwt_sealed IS NULL) = (jwt_expires_at IS NULL)),
    CHECK ((jwt_sealed IS NULL) = (signed_in_at IS NULL)),
    CHECK ((jwt_sealed IS NULL) = (account_name IS NULL)),
    CHECK ((jwt_sealed IS NULL) = (plex_pass IS NULL))
) STRICT;

-- A Media Server change: one row per server and path, folded on write by
-- queue.Fold. A row with from_path is a carry (a moved change, or a
-- replaced one under a new name) and keeps its step, so a retry or a
-- fallback resumes where it stopped and never snapshots twice. prepared
-- holds a snapshot taken before a rename's or an upgrade's bytes moved.
-- step has no CHECK: each kind owns its steps, and link.ParseStep reads
-- them.
CREATE TABLE mediaservers_change (
    id BLOB PRIMARY KEY NOT NULL,
    server_id BLOB NOT NULL REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('added', 'replaced', 'removed', 'moved',
        'sidecar_added', 'sidecar_replaced', 'sidecar_removed', 'library')),
    root_id BLOB NOT NULL REFERENCES roots_root (id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    from_path TEXT CHECK (from_path <> ''),
    file_id BLOB,
    -- A removed whose file an upgrade replaced: the replacer's id.
    replaced_by BLOB,
    -- An added whose file an upgrade's removed will follow.
    upgrade INTEGER NOT NULL DEFAULT 0 CHECK (upgrade IN (0, 1)),
    state TEXT NOT NULL CHECK (state IN ('prepared', 'due', 'sending', 'sent')),
    step TEXT NOT NULL DEFAULT 'none' CHECK (step <> ''),
    step_at INTEGER,
    snapshot TEXT CHECK (snapshot IS NULL OR json_valid(snapshot)),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    sent_at INTEGER,
    result TEXT CHECK (result IN ('accepted', 'covered', 'converted')),
    lost TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(lost) AND json_type(lost) = 'array'),
    -- The last failure the server itself responded with, as link.Problem.
    problem TEXT CHECK (problem IS NULL OR json_valid(problem)),
    CHECK ((kind = 'library') = (path = '')),
    CHECK ((kind = 'library') = (file_id IS NULL)),
    CHECK (from_path IS NULL OR kind IN ('moved', 'replaced')),
    CHECK (kind <> 'moved' OR from_path IS NOT NULL),
    CHECK (from_path IS NULL OR from_path <> path),
    CHECK (state <> 'prepared' OR from_path IS NOT NULL),
    CHECK (replaced_by IS NULL OR kind = 'removed' OR (state = 'prepared' AND kind = 'replaced')),
    CHECK (upgrade = 0 OR kind = 'added'),
    CHECK ((state = 'sent') = (sent_at IS NOT NULL)),
    CHECK ((state = 'sent') = (result IS NOT NULL)),
    CHECK ((step = 'none') = (step_at IS NULL)),
    CHECK (snapshot IS NULL OR step <> 'none')
) STRICT;

-- Compacted per path: one foldable row per server, root and path, and one
-- foldable carry per server and file. Their predicate is queue.Foldable
-- (TestFoldablePredicate pins the two together).
CREATE UNIQUE INDEX mediaservers_change_open_path ON mediaservers_change (server_id, root_id, path)
    WHERE from_path IS NULL AND state IN ('prepared', 'due') AND step IN ('none', 'snapshotted');
CREATE UNIQUE INDEX mediaservers_change_open_carry ON mediaservers_change (server_id, file_id)
    WHERE from_path IS NOT NULL AND state IN ('prepared', 'due') AND step IN ('none', 'snapshotted');
CREATE INDEX mediaservers_change_due ON mediaservers_change (server_id, created_at)
    WHERE state IN ('due', 'sending');
CREATE INDEX mediaservers_change_file ON mediaservers_change (server_id, file_id)
    WHERE file_id IS NOT NULL AND state IN ('prepared', 'due');
CREATE INDEX mediaservers_change_partner ON mediaservers_change (server_id, replaced_by)
    WHERE replaced_by IS NOT NULL AND state IN ('prepared', 'due');
CREATE INDEX mediaservers_change_server ON mediaservers_change (server_id, id);
CREATE INDEX mediaservers_change_sent ON mediaservers_change (sent_at) WHERE state = 'sent';

-- The path index and the Open in cache: the item at each server path a
-- listing, a lookup or a step saw. path_key is link.ServerPath.Key, case
-- folded on a Windows server, so a lookup by a joined path finds the
-- server's own spelling.
CREATE TABLE mediaservers_item (
    server_id BLOB NOT NULL REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    path_key TEXT NOT NULL CHECK (path_key <> ''),
    server_path TEXT NOT NULL CHECK (server_path <> ''),
    library_key TEXT NOT NULL,
    file_id BLOB,
    item_key TEXT NOT NULL CHECK (item_key <> ''),
    -- The show item when the item has one, else item_key.
    top_key TEXT NOT NULL CHECK (top_key <> ''),
    seen_at INTEGER NOT NULL,
    -- A push said the item changed: check it before use.
    stale INTEGER NOT NULL DEFAULT 0 CHECK (stale IN (0, 1)),
    PRIMARY KEY (server_id, path_key)
) STRICT;

CREATE INDEX mediaservers_item_file ON mediaservers_item (file_id) WHERE file_id IS NOT NULL;
CREATE INDEX mediaservers_item_key ON mediaservers_item (server_id, item_key);

-- Play history. The reaching Node mints a play's id when it first sees the
-- session, so its start, changes and end name one row, and a late report
-- never opens a play again. The key includes the server, so a Node can
-- only ever name its own servers' plays.
CREATE TABLE mediaservers_play (
    server_id BLOB NOT NULL REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    id BLOB NOT NULL,
    session_key TEXT NOT NULL,
    file_id BLOB,
    server_path TEXT NOT NULL,
    user_name TEXT NOT NULL,
    player TEXT NOT NULL,
    product TEXT NOT NULL,
    decision TEXT NOT NULL CHECK (decision IN ('direct_play', 'copy', 'transcode')),
    hardware INTEGER NOT NULL DEFAULT 0 CHECK (hardware IN (0, 1)),
    reasons TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(reasons) AND json_type(reasons) = 'array'),
    started_at INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    ended_at INTEGER,
    PRIMARY KEY (server_id, id),
    CHECK (last_seen_at >= started_at),
    CHECK (ended_at IS NULL OR ended_at >= started_at)
) STRICT;

CREATE INDEX mediaservers_play_open ON mediaservers_play (server_id) WHERE ended_at IS NULL;
CREATE INDEX mediaservers_play_started ON mediaservers_play (started_at);
CREATE INDEX mediaservers_play_file ON mediaservers_play (file_id) WHERE file_id IS NOT NULL;
