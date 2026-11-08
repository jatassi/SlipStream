-- +goose Up
-- The Download Client a Tracker's Grabs go to when an Indexer on
-- home_node_id found them, while the Admin picked one. It names either an
-- external client or a Node's built-in client.
CREATE TABLE trackers_route (
    tracker_id BLOB NOT NULL REFERENCES indexers_tracker (id) ON DELETE CASCADE,
    home_node_id BLOB NOT NULL REFERENCES nodes_node (id) ON DELETE CASCADE,
    built_in_on BLOB REFERENCES nodes_node (id) ON DELETE CASCADE,
    client_id BLOB REFERENCES downloadclients_client (id) ON DELETE CASCADE,
    PRIMARY KEY (tracker_id, home_node_id),
    CHECK ((built_in_on IS NULL) <> (client_id IS NULL))
) STRICT, WITHOUT ROWID;

CREATE INDEX trackers_route_client ON trackers_route (client_id) WHERE client_id IS NOT NULL;
