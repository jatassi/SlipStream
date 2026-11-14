-- +goose Up
-- grabs_release: the ranking gains the fewer-Releases step after want, so a
-- Work fills with the fewest Releases.
CREATE TABLE grabs_release_next (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    title TEXT NOT NULL,
    indexer_id BLOB REFERENCES indexers_indexer (id) ON DELETE SET NULL,
    indexer_name TEXT NOT NULL,
    size INTEGER NOT NULL,
    seeders INTEGER NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('grabbed', 'not_taken', 'outranked', 'kept', 'refused', 'listed')),
    rank INTEGER CHECK (rank >= 1),
    reasons TEXT NOT NULL CHECK (json_valid(reasons)),
    lost_step TEXT CHECK (lost_step IN ('want', 'fewer_releases', 'tier', 'revision', 'score', 'chosen_by_another', 'fewer_unwanted', 'priority',
        'freeleech', 'seeders', 'size', 'title')),
    lost_target_id BLOB REFERENCES targets_target (id) ON DELETE SET NULL,
    lost_own TEXT,
    lost_ref TEXT,
    scores TEXT NOT NULL CHECK (json_valid(scores)),
    formats TEXT NOT NULL CHECK (json_valid(formats)),
    covers TEXT NOT NULL CHECK (json_valid(covers)),
    copy_of INTEGER CHECK (copy_of >= 0),
    release TEXT CHECK (release IS NULL OR json_valid(release)),
    link_sealed BLOB,
    read TEXT CHECK (read IS NULL OR json_valid(read)),
    PRIMARY KEY (search_id, position),
    CHECK (outcome IN ('refused', 'listed') OR rank IS NOT NULL),
    CHECK (lost_step IS NULL OR outcome IN ('outranked', 'kept')),
    CHECK ((lost_step IS NULL) = (lost_own IS NULL) AND (lost_step IS NULL) = (lost_ref IS NULL)),
    CHECK (link_sealed IS NULL OR release IS NOT NULL),
    CHECK ((outcome = 'listed') = (read IS NOT NULL)),
    CHECK (outcome <> 'listed' OR (rank IS NULL AND copy_of IS NULL AND release IS NOT NULL AND link_sealed IS NOT NULL))
) STRICT;

INSERT INTO grabs_release_next (search_id, position, title, indexer_id, indexer_name, size, seeders, outcome, rank, reasons,
    lost_step, lost_target_id, lost_own, lost_ref, scores, formats, covers, copy_of, release, link_sealed, read)
SELECT search_id, position, title, indexer_id, indexer_name, size, seeders, outcome, rank, reasons, lost_step, lost_target_id,
    lost_own, lost_ref, scores, formats, covers, copy_of, release, link_sealed, read
FROM grabs_release;

DROP TABLE grabs_release;
ALTER TABLE grabs_release_next RENAME TO grabs_release;

CREATE INDEX grabs_release_indexer_id ON grabs_release (indexer_id);
CREATE INDEX grabs_release_lost_target_id ON grabs_release (lost_target_id);
