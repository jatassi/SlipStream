-- +goose Up
-- The client an Admin sent a Grab to before its fetch, which its fetch
-- takes over its Tracker's route: an external client, or a Node's
-- built-in client.
ALTER TABLE grabs_grab ADD COLUMN sent_built_in_on BLOB REFERENCES nodes_node (id);
ALTER TABLE grabs_grab ADD COLUMN sent_client_id BLOB REFERENCES downloadclients_client (id)
    CHECK (sent_client_id IS NULL OR sent_built_in_on IS NULL);
