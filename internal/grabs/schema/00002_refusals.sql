-- +goose Up
-- Why a search took no release: one row per refused candidate, in the
-- order the Indexer listed them, with its reasons as JSON
-- [{code, targetId, attribute, value}].
CREATE TABLE grabs_refusal (
    search_id BLOB NOT NULL REFERENCES grabs_search (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    title TEXT NOT NULL,
    reasons TEXT NOT NULL CHECK (json_valid(reasons)),
    PRIMARY KEY (search_id, position)
) STRICT;

-- The notes on the release a search grabbed, such as size_not_checked,
-- JSON as a refusal's reasons.
ALTER TABLE grabs_search ADD COLUMN notes TEXT CHECK (notes IS NULL OR json_valid(notes));

-- A Target no Profile holds can be deleted, and the Targets its Grabs were
-- for go with it.
CREATE TABLE grabs_target_next (
    grab_id BLOB NOT NULL REFERENCES grabs_grab (id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    target_id BLOB NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    PRIMARY KEY (grab_id, position)
) STRICT;

INSERT INTO grabs_target_next (grab_id, position, target_id) SELECT grab_id, position, target_id FROM grabs_target;
DROP TABLE grabs_target;
ALTER TABLE grabs_target_next RENAME TO grabs_target;

CREATE TABLE grabs_fill_next (
    grab_id BLOB NOT NULL REFERENCES grabs_grab (id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    target_id BLOB NOT NULL REFERENCES targets_target (id) ON DELETE CASCADE,
    PRIMARY KEY (grab_id, position)
) STRICT;

INSERT INTO grabs_fill_next (grab_id, position, target_id) SELECT grab_id, position, target_id FROM grabs_fill;
DROP TABLE grabs_fill;
ALTER TABLE grabs_fill_next RENAME TO grabs_fill;

CREATE INDEX grabs_target_target_id ON grabs_target (target_id);
CREATE INDEX grabs_fill_target_id ON grabs_fill (target_id);

-- What a Grab's probe or import does with its file if it fills nothing,
-- as an Admin chose on saving a change that took its cells away: delete
-- it, or keep it as a Kept file. NULL when no save did.
ALTER TABLE grabs_grab ADD COLUMN unwanted TEXT CHECK (unwanted IN ('delete', 'keep'));
