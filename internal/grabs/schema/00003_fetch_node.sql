-- +goose Up
-- fetch_node_id is the Searching Node that fetched a Grab's .torrent, and
-- node_id becomes the Downloading Node it handed the .torrent to. Until
-- now both were the Indexer's home Node.
ALTER TABLE grabs_grab ADD COLUMN fetch_node_id BLOB REFERENCES nodes_node (id);
UPDATE grabs_grab SET fetch_node_id = node_id WHERE info_hash IS NOT NULL;
