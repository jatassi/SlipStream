-- +goose Up
-- S51 (Reachability diagnostic). An Exposure is something the Admin wants
-- reachable from outside the home. Each has one latest result and no
-- history. reachability is the only writer of every table here. Node ids
-- reference nodes_node, whose rows are tombstoned and never deleted, so
-- reachability deletes a removed Node's rows itself when it sees the
-- removal (R17), and every reader filters removed Nodes. A Media Server's
-- Exposure, result and dismissal go with its mediaservers_server row by
-- cascade (R5), so this set is listed after mediaservers in LibrarySchemas.

CREATE TABLE reachability_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    -- Shape only: the setting declarations hold the bounds, so a bound can
    -- change without a migration.
    warn_after_seconds INTEGER CHECK (warn_after_seconds > 0),
    watch_every_seconds INTEGER CHECK (watch_every_seconds > 0),
    change_wait_seconds INTEGER CHECK (change_wait_seconds >= 0),
    -- NULL: each Node's own stun Endpoints entry. '[]': contact none.
    stun_servers TEXT CHECK (stun_servers IS NULL OR (json_valid(stun_servers) AND json_type(stun_servers) = 'array'))
) STRICT;
INSERT INTO reachability_settings (id) VALUES (1);

CREATE TABLE reachability_exposure (
    id BLOB PRIMARY KEY NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('ui-portal', 'media-server')),
    server_id BLOB REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    -- A Media Server's public address, prefilled and editable. The UI and
    -- portal's is read from Address on every run, never copied (spec).
    public TEXT,
    wanted_at INTEGER NOT NULL,
    -- Written by the run only: the last Media Server facts and when they
    -- were read, kept while the reaching Node is away.
    server_facts TEXT CHECK (server_facts IS NULL OR json_valid(server_facts)),
    server_observed_at INTEGER,
    -- A digest of the server record's address, reaching, runs-on, kind and
    -- identity; a media-servers.changed that leaves it alone reruns nothing.
    server_digest BLOB,
    CHECK ((kind = 'ui-portal' AND server_id IS NULL AND public IS NULL AND server_facts IS NULL AND server_digest IS NULL)
        OR (kind = 'media-server' AND server_id IS NOT NULL AND public IS NOT NULL)),
    CHECK ((server_facts IS NULL) = (server_observed_at IS NULL))
) STRICT;
CREATE UNIQUE INDEX reachability_exposure_server ON reachability_exposure (server_id) WHERE server_id IS NOT NULL;
CREATE UNIQUE INDEX reachability_exposure_ui ON reachability_exposure (kind) WHERE kind = 'ui-portal';

-- A suggestion the Admin dismissed. Wanting the subject deletes its row.
CREATE TABLE reachability_dismissed (
    kind TEXT NOT NULL CHECK (kind IN ('ui-portal', 'media-server')),
    server_id BLOB REFERENCES mediaservers_server (id) ON DELETE CASCADE,
    dismissed_at INTEGER NOT NULL,
    CHECK ((kind = 'ui-portal') = (server_id IS NULL))
) STRICT;
CREATE UNIQUE INDEX reachability_dismissed_server ON reachability_dismissed (server_id) WHERE server_id IS NOT NULL;
CREATE UNIQUE INDEX reachability_dismissed_ui ON reachability_dismissed (kind) WHERE kind = 'ui-portal';

-- The latest result per Exposure. judged is judge.Judged as versioned JSON,
-- parsed at the storage boundary; a row of another version reads as not
-- judged yet. verdict and failing_at are columns because Health
-- and the Nodes summary read them without parsing.
CREATE TABLE reachability_result (
    exposure_id BLOB PRIMARY KEY NOT NULL REFERENCES reachability_exposure (id) ON DELETE CASCADE,
    judged TEXT NOT NULL CHECK (json_valid(judged)),
    verdict TEXT NOT NULL CHECK (verdict IN ('reachable', 'fixable', 'needs-relay')),
    tested INTEGER NOT NULL CHECK (tested IN (0, 1)),
    judged_at INTEGER NOT NULL,
    -- When the verdict began resting on a failing check (R10); NULL while
    -- it rests on none. Not tested never sets it.
    failing_at INTEGER,
    -- The unit's Node: TiedTo for the next run. NULL is the Controller.
    origin_node_id BLOB REFERENCES nodes_node (id),
    -- The Node whose outside test counted, NULL for none or for Plex's own check.
    tested_by_node_id BLOB REFERENCES nodes_node (id),
    CHECK (verdict <> 'reachable' OR failing_at IS NULL),
    CHECK (failing_at IS NULL OR failing_at <= judged_at)
) STRICT;
CREATE INDEX reachability_result_failing ON reachability_result (failing_at) WHERE failing_at IS NOT NULL;

-- Each Node's latest full facts from Read, with when they were observed.
-- Upserted only WHERE observed_at < excluded.observed_at, so the newest pass
-- wins whatever order two units commit in. public4 and prefixes6 are the
-- facts' own, kept as columns for Carry; previous4 and previous6 are owned
-- here, never sent by a Node: Carry sets previous4 to the stored public4
-- when a new, known Public4 differs, and previous6 to the stored /64s the
-- new set lacks when the set changes.
CREATE TABLE reachability_node (
    node_id BLOB PRIMARY KEY NOT NULL REFERENCES nodes_node (id),
    facts TEXT NOT NULL CHECK (json_valid(facts)),
    observed_at INTEGER NOT NULL,
    public4 BLOB CHECK (public4 IS NULL OR length(public4) = 4),
    prefixes6 TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(prefixes6) AND json_type(prefixes6) = 'array'),
    previous4 BLOB CHECK (previous4 IS NULL OR length(previous4) = 4),
    previous6 TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(previous6) AND json_type(previous6) = 'array')
) STRICT;

-- The two per-Node settings, stored on the Controller because it derives
-- every mapping target (trust) and Workers keep no settings of their own.
CREATE TABLE reachability_node_setting (
    node_id BLOB PRIMARY KEY NOT NULL REFERENCES nodes_node (id),
    map_ports INTEGER CHECK (map_ports IN (0, 1)),
    stable_ipv6 TEXT
) STRICT;
