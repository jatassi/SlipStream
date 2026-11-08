-- +goose Up
-- S23 (Release parsing and Custom Formats) keeps every Release a run
-- listed as one row with its outcome, so a run's refusals move into
-- grabs_release. v2 has no installs to keep: the refusals and the
-- Releases kept for a rerun are dropped.
DROP TABLE grabs_refusal;
DROP TABLE grabs_release;

-- A run's Releases as its decision saw them, kept for each Item's latest
-- finished run and for a run whose Grab still fetches. rank is the place
-- among the Releases grabbable at some cell, from 1. reasons is JSON as
-- grabs_search.notes, scores [{targetId, score}], formats and covers the
-- matched format and covered Item ids. copy_of is the position of the
-- Release this one is a copy of. release is the Release as JSON with its
-- link sealed apart, kept while a rerun may take it. A Release whose link
-- isn't kept can't be taken again.
CREATE TABLE grabs_release (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    title TEXT NOT NULL,
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    indexer_name TEXT NOT NULL,
    size INTEGER NOT NULL,
    seeders INTEGER NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('grabbed', 'not_taken', 'outranked', 'kept', 'refused')),
    rank INTEGER CHECK (rank >= 1),
    reasons TEXT NOT NULL CHECK (json_valid(reasons)),
    lost_step TEXT CHECK (lost_step IN ('want', 'tier', 'revision', 'score', 'priority', 'freeleech', 'seeders', 'size', 'title')),
    lost_target_id BLOB REFERENCES targets_target (id) ON DELETE SET NULL,
    lost_own TEXT,
    lost_ref TEXT,
    scores TEXT NOT NULL CHECK (json_valid(scores)),
    formats TEXT NOT NULL CHECK (json_valid(formats)),
    covers TEXT NOT NULL CHECK (json_valid(covers)),
    copy_of INTEGER CHECK (copy_of >= 0),
    release TEXT CHECK (release IS NULL OR json_valid(release)),
    link_sealed BLOB,
    PRIMARY KEY (search_id, position),
    CHECK (outcome = 'refused' OR rank IS NOT NULL),
    CHECK (lost_step IS NULL OR outcome IN ('outranked', 'kept')),
    CHECK ((lost_step IS NULL) = (lost_own IS NULL) AND (lost_step IS NULL) = (lost_ref IS NULL)),
    CHECK (link_sealed IS NULL OR release IS NOT NULL)
) STRICT;

CREATE INDEX grabs_release_indexer_id ON grabs_release (indexer_id);
CREATE INDEX grabs_release_lost_target_id ON grabs_release (lost_target_id);
CREATE INDEX grabs_search_item_id ON grabs_search (item_id, finished_at);
